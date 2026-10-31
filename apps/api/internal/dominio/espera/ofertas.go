package espera

// Ofrecer el hueco solo (propuesta 01 del diseño v2). Un horario que se libera se le ofrece a UNA
// persona de la lista por vez, en el orden en que se anotaron, con un link propio y un tiempo para
// tomarlo (ajustes.espera_minutos). «Tomar el lugar» reserva el turno con sus datos, igual que la
// página pública (con seña si el negocio la cobra); «No me sirve» o el tiempo vencido lo pasan a la
// siguiente. La cola se arma una vez, al liberarse el horario (oferta_intentos), y la hace avanzar
// quien responde o, si nadie responde, el trabajo del worker. Si el horario se ocupa por otro lado,
// la oferta se cae (el trigger app.turno_ocupa, y Caer para los bloqueos).
//
// El orden de los bloqueos es siempre oferta → intento (el trigger también): así quien toma, quien
// pasa y el worker no se cruzan.

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/avisos"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/negocios"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/turnos"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/auditoria"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/problema"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/sesion"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/trabajos"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/web"
)

// Hasta cuántas personas entran en la cola de un hueco: más que eso, el horario ya pasó antes de
// llegar a la última.
const maxCola = 20

// Hora (del negocio) desde la que un horario es «tarde» para quien se anotó por la mañana o la tarde.
const horaTarde = 13

// Oferta: un horario liberado que se ofrece a la lista, como la ve el panel.
type Oferta struct {
	ID            string     `json:"id"`
	Estado        string     `json:"estado"`
	Origen        string     `json:"origen"`
	ServicioID    string     `json:"servicioId"`
	Servicio      string     `json:"servicio"`
	ProfesionalID string     `json:"profesionalId"`
	Profesional   string     `json:"profesional"`
	Inicio        time.Time  `json:"inicio"`
	Fin           time.Time  `json:"fin"`
	Precio        int64      `json:"precio"`
	TurnoID       *string    `json:"turnoId"`
	CreadoEn      time.Time  `json:"creadoEn"`
	CerradaEn     *time.Time `json:"cerradaEn"`
	// Actual: a quién se le está ofreciendo ahora (con su vencimiento); Sigue: quién viene después.
	Actual *Intento  `json:"actual"`
	Sigue  *Intento  `json:"sigue"`
	Cola   []Intento `json:"cola"`
}

// Intento: una persona de la cola de una oferta.
type Intento struct {
	ID           string     `json:"id"`
	EsperaID     *string    `json:"esperaId"`
	Nombre       string     `json:"nombre"`
	Orden        int        `json:"orden"`
	Estado       string     `json:"estado"`
	OfrecidaEn   *time.Time `json:"ofrecidaEn"`
	VenceEn      *time.Time `json:"venceEn"`
	RespondidaEn *time.Time `json:"respondidaEn"`
}

// AjustesEspera: cuánto tiene cada persona para tomar el lugar y si se ofrece sola.
type AjustesEspera struct {
	Minutos    int  `json:"minutos"`
	Automatica bool `json:"automatica"`
}

func leerAjustes(ctx context.Context, tx pgx.Tx) (AjustesEspera, error) {
	var a AjustesEspera
	err := tx.QueryRow(ctx, "SELECT espera_minutos, espera_automatica FROM ajustes WHERE negocio_id = app.negocio_actual()").
		Scan(&a.Minutos, &a.Automatica)
	return a, err
}

// momentoDe: «manana» o «tarde», según la hora del negocio.
func momentoDe(inicio time.Time, z *time.Location) string {
	if inicio.In(z).Hour() < horaTarde {
		return "manana"
	}
	return "tarde"
}

// candidatos: los de la lista a los que les sirve el horario (ese servicio, ese profesional o
// cualquiera, ese día, ese momento), que siguen esperando, en el orden en que se anotaron.
func candidatos(ctx context.Context, tx pgx.Tx, servicioID, profesionalID string, inicio time.Time, z *time.Location, limite int) ([]Anotado, error) {
	filas, err := tx.Query(ctx, "SELECT "+columnas+desde+` WHERE e.servicio_id::text = $1
		AND (e.profesional_id IS NULL OR e.profesional_id::text = $2) AND e.avisado_en IS NULL
		AND $3::date BETWEEN e.desde AND e.hasta AND e.momento IN ('cualquiera', $4)
		ORDER BY e.creado_en LIMIT $5`, servicioID, profesionalID, inicio.In(z).Format(time.DateOnly), momentoDe(inicio, z), limite)
	if err != nil {
		return nil, err
	}
	defer filas.Close()
	out := []Anotado{}
	for filas.Next() {
		var a Anotado
		if err := escanear(filas, &a); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, filas.Err()
}

// crearOferta arma la cola del horario y se lo ofrece a la primera. Los problemas (horario ocupado,
// nadie esperando, ya se ofrece) salen antes de escribir nada.
func crearOferta(ctx context.Context, tx pgx.Tx, servicioID, profesionalID string, inicio time.Time, origen string, ahora time.Time) (string, error) {
	if !inicio.After(ahora) {
		return "", problema.Invalido("Ese horario ya pasó.", problema.Campo{Ruta: "inicio", Mensaje: "pasado"})
	}
	var servicio, profesional string
	var duracion int
	var precio int64
	err := tx.QueryRow(ctx, `SELECT s.nombre, s.duracion, s.precio, p.nombre FROM servicios s, profesionales p
		WHERE s.id::text = $1 AND p.id::text = $2`, servicioID, profesionalID).Scan(&servicio, &duracion, &precio, &profesional)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", problema.NoEncontrado("No encontramos ese servicio o ese profesional.")
	}
	if err != nil {
		return "", err
	}
	libre, err := turnos.Disponible(ctx, tx, servicioID, profesionalID, inicio, ahora)
	if err != nil {
		return "", err
	}
	if !libre {
		return "", problema.Conflicto("horario_ocupado", "Ese horario ya no está libre.")
	}
	var ya bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM ofertas WHERE profesional_id::text = $1 AND inicio = $2 AND estado = 'ofreciendo')`,
		profesionalID, inicio).Scan(&ya); err != nil {
		return "", err
	}
	if ya {
		return "", problema.Conflicto("ya_ofreciendo", "Ese horario ya se está ofreciendo a la lista.")
	}
	cs, err := candidatos(ctx, tx, servicioID, profesionalID, inicio, zona(ctx, tx), maxCola)
	if err != nil {
		return "", err
	}
	if len(cs) == 0 {
		return "", problema.Conflicto("sin_espera", "Nadie de la lista de espera busca ese horario.")
	}
	var id string
	if err := tx.QueryRow(ctx, `INSERT INTO ofertas (negocio_id, servicio_id, servicio, profesional_id, profesional, inicio, fin, precio, origen)
		VALUES (app.negocio_actual(), $1, $2, $3, $4, $5, $6, $7, $8) RETURNING id::text`,
		servicioID, servicio, profesionalID, profesional, inicio, inicio.Add(time.Duration(duracion)*time.Minute), precio, origen).Scan(&id); err != nil {
		return "", err
	}
	for i, c := range cs {
		if _, err := tx.Exec(ctx, `INSERT INTO oferta_intentos (negocio_id, oferta_id, espera_id, nombre, orden, token)
			VALUES (app.negocio_actual(), $1, $2, $3, $4, $5)`, id, c.ID, c.Nombre, i+1, avisos.Token()); err != nil {
			return "", err
		}
	}
	return id, avanzar(ctx, tx, id, ahora)
}

// avanzar deja la oferta en su próximo paso: si alguien tiene el link vigente, nada; si no, vence al
// que no respondió y se la ofrece al siguiente que siga esperando (o la cierra: pasó la hora, se
// ocupó, no queda nadie).
func avanzar(ctx context.Context, tx pgx.Tx, id string, ahora time.Time) error {
	var estado, servicioID, profesionalID, servicio, profesional string
	var inicio time.Time
	err := tx.QueryRow(ctx, `SELECT estado, servicio_id::text, profesional_id::text, servicio, profesional, inicio FROM ofertas
		WHERE id::text = $1 FOR UPDATE`, id).Scan(&estado, &servicioID, &profesionalID, &servicio, &profesional, &inicio)
	if err != nil || estado != "ofreciendo" {
		return err
	}
	var vigente bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM oferta_intentos WHERE oferta_id::text = $1 AND estado = 'ofrecida' AND vence_en > $2)`,
		id, ahora).Scan(&vigente); err != nil || vigente {
		return err
	}
	// Quien no respondió a tiempo queda avisado (no vuelve a la cola de otros horarios: «no respondió»).
	if _, err := tx.Exec(ctx, `UPDATE oferta_intentos SET estado = 'vencida' WHERE oferta_id::text = $1 AND estado = 'ofrecida'`, id); err != nil {
		return err
	}
	if !inicio.After(ahora) {
		return cerrar(ctx, tx, id, "vencida", ahora)
	}
	libre, err := turnos.Disponible(ctx, tx, servicioID, profesionalID, inicio, ahora)
	if err != nil {
		return err
	}
	if !libre {
		return cerrar(ctx, tx, id, "caida", ahora)
	}
	a, err := leerAjustes(ctx, tx)
	if err != nil {
		return err
	}
	for {
		var intento string
		var esperaID, email *string
		var sigue bool
		err := tx.QueryRow(ctx, `SELECT i.id::text, i.espera_id::text, e.email, e.id IS NOT NULL AND e.avisado_en IS NULL
			FROM oferta_intentos i LEFT JOIN espera e ON e.id = i.espera_id
			WHERE i.oferta_id::text = $1 AND i.estado = 'en_cola' ORDER BY i.orden LIMIT 1 FOR UPDATE OF i`, id).
			Scan(&intento, &esperaID, &email, &sigue)
		if errors.Is(err, pgx.ErrNoRows) {
			if err := cerrar(ctx, tx, id, "agotada", ahora); err != nil {
				return err
			}
			z := zona(ctx, tx)
			return avisos.Notificar(ctx, tx, avisos.Notificacion{Tipo: avisos.NotiEspera,
				Titulo: "Nadie de la lista tomó el hueco de las " + avisos.Hora(inicio.In(z)),
				Texto:  servicio + " con " + profesional + ", el " + avisos.Fecha(inicio.In(z)) + ". Sigue libre en la agenda."})
		}
		if err != nil {
			return err
		}
		if !sigue {
			if _, err := tx.Exec(ctx, "UPDATE oferta_intentos SET estado = 'salteada' WHERE id::text = $1", intento); err != nil {
				return err
			}
			continue
		}
		vence := ahora.Add(time.Duration(a.Minutos) * time.Minute)
		if vence.After(inicio) {
			vence = inicio
		}
		if _, err := tx.Exec(ctx, `UPDATE oferta_intentos SET estado = 'ofrecida', ofrecida_en = $2, vence_en = $3 WHERE id::text = $1`,
			intento, ahora, vence); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, "UPDATE espera SET avisado_en = $2 WHERE id::text = $1", *esperaID, ahora); err != nil {
			return err
		}
		_, err = avisos.Encolar(ctx, tx, avisos.Aviso{Tipo: avisos.Espera, Para: *email, Clave: "oferta:" + intento,
			Datos: map[string]any{"intento": intento, "espera": *esperaID}}, ahora)
		return err
	}
}

// cerrar la oferta con ese estado. A quien tenía el link se le cancela y vuelve a esperar.
func cerrar(ctx context.Context, tx pgx.Tx, id, estado string, ahora time.Time) error {
	if _, err := tx.Exec(ctx, `UPDATE ofertas SET estado = $2, cerrada_en = $3 WHERE id::text = $1 AND estado = 'ofreciendo'`,
		id, estado, ahora); err != nil {
		return err
	}
	_, err := tx.Exec(ctx, `WITH i AS (UPDATE oferta_intentos SET estado = 'cancelada' WHERE oferta_id::text = $1 AND estado = 'ofrecida'
		RETURNING espera_id) UPDATE espera SET avisado_en = NULL WHERE id IN (SELECT espera_id FROM i)`, id)
	return err
}

// Caer cierra las ofertas de un profesional que caen entre desde y hasta: el horario se ocupó por
// otro lado (p. ej. un bloqueo por días libres). Con el negocio fijado.
func Caer(ctx context.Context, tx pgx.Tx, profesionalID string, desde, hasta, ahora time.Time) error {
	filas, err := tx.Query(ctx, `SELECT id::text FROM ofertas WHERE profesional_id::text = $1 AND estado = 'ofreciendo'
		AND inicio < $3 AND fin > $2 FOR UPDATE`, profesionalID, desde, hasta)
	if err != nil {
		return err
	}
	ids, err := pgx.CollectRows(filas, pgx.RowTo[string])
	if err != nil {
		return err
	}
	for _, id := range ids {
		if err := cerrar(ctx, tx, id, "caida", ahora); err != nil {
			return err
		}
	}
	return nil
}

// Liberado es el gancho para avisos.Servicio.AlLiberarse: en la misma transacción que el aviso del
// turno, si el negocio ofrece solo, arma la cola del horario y se lo ofrece a la primera de la lista.
// Si lo dispara recepción, sólo suma a la campanita cuántos esperan ese horario.
func Liberado(ctx context.Context, tx pgx.Tx, l avisos.Liberado) error {
	if l.ProfesionalID == "" {
		return nil
	}
	a, err := leerAjustes(ctx, tx)
	if err != nil {
		return err
	}
	if !a.Automatica {
		cs, err := candidatos(ctx, tx, l.ServicioID, l.ProfesionalID, l.Inicio, l.Zona, maxCola)
		if err != nil || len(cs) == 0 {
			return err
		}
		n := "1 persona espera"
		if len(cs) > 1 {
			n = strconv.Itoa(len(cs)) + " personas esperan"
		}
		return avisos.Notificar(ctx, tx, avisos.Notificacion{Tipo: avisos.NotiEspera,
			Titulo: "Se liberó las " + avisos.Hora(l.Inicio.In(l.Zona)) + " con " + l.Profesional,
			Texto:  n + " ese horario en la lista. Ofrecéselo desde Turnos."})
	}
	_, err = crearOferta(ctx, tx, l.ServicioID, l.ProfesionalID, l.Inicio, "automatica", l.Ahora)
	// Que no haya nadie esperando, o que el horario ya no esté, no es un error del aviso.
	var p *problema.Problema
	if errors.As(err, &p) {
		return nil
	}
	return err
}

// ---- Panel

func leerOferta(ctx context.Context, tx pgx.Tx, id string) (Oferta, error) {
	var o Oferta
	err := tx.QueryRow(ctx, `SELECT id::text, estado, origen, servicio_id::text, servicio, profesional_id::text, profesional, inicio, fin,
		precio, turno_id::text, creado_en, cerrada_en FROM ofertas WHERE id::text = $1`, id).Scan(&o.ID, &o.Estado, &o.Origen,
		&o.ServicioID, &o.Servicio, &o.ProfesionalID, &o.Profesional, &o.Inicio, &o.Fin, &o.Precio, &o.TurnoID, &o.CreadoEn, &o.CerradaEn)
	if err != nil {
		return o, err
	}
	filas, err := tx.Query(ctx, `SELECT id::text, espera_id::text, nombre, orden, estado, ofrecida_en, vence_en, respondida_en
		FROM oferta_intentos WHERE oferta_id = $1 ORDER BY orden`, o.ID)
	if err != nil {
		return o, err
	}
	o.Cola, err = pgx.CollectRows(filas, pgx.RowToStructByPos[Intento])
	if err != nil {
		return o, err
	}
	for i := range o.Cola {
		c := &o.Cola[i]
		if c.Estado == "ofrecida" && o.Actual == nil {
			o.Actual = c
		}
		if c.Estado == "en_cola" && o.Sigue == nil && o.Estado == "ofreciendo" {
			o.Sigue = c
		}
	}
	return o, nil
}

// listarOfertas: las que se están ofreciendo; con ?fecha=AAAA-MM-DD, todas las de ese día.
func (s *Servicio) listarOfertas(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	out := []Oferta{}
	err := s.Pool.EnNegocio(ctx, sesion.De(ctx).NegocioID, func(tx pgx.Tx) error {
		q := "SELECT id::text FROM ofertas WHERE estado = 'ofreciendo' ORDER BY inicio LIMIT 100"
		var args []any
		if f := r.URL.Query().Get("fecha"); f != "" {
			dia, err := time.ParseInLocation(time.DateOnly, f, zona(ctx, tx))
			if err != nil {
				return problema.Invalido("La fecha va como AAAA-MM-DD.")
			}
			q = "SELECT id::text FROM ofertas WHERE inicio >= $1 AND inicio < $2 ORDER BY inicio, creado_en LIMIT 100"
			args = []any{dia, dia.AddDate(0, 0, 1)}
		}
		filas, err := tx.Query(ctx, q, args...)
		if err != nil {
			return err
		}
		ids, err := pgx.CollectRows(filas, pgx.RowTo[string])
		if err != nil {
			return err
		}
		for _, id := range ids {
			o, err := leerOferta(ctx, tx, id)
			if err != nil {
				return err
			}
			out = append(out, o)
		}
		return nil
	})
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusOK, map[string]any{"ofertas": out})
	return nil
}

func (s *Servicio) verOferta(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	var o Oferta
	err := s.Pool.EnNegocio(ctx, sesion.De(ctx).NegocioID, func(tx pgx.Tx) (err error) {
		o, err = leerOferta(ctx, tx, r.PathValue("id"))
		return err
	})
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusOK, o)
	return nil
}

type pedidoHueco struct {
	ServicioID    string    `json:"servicioId"`
	ProfesionalID string    `json:"profesionalId"`
	Inicio        time.Time `json:"inicio"`
}

func (p pedidoHueco) validar() error {
	if p.ServicioID == "" || p.ProfesionalID == "" || p.Inicio.IsZero() {
		return problema.Invalido("Faltan servicioId, profesionalId e inicio.")
	}
	return nil
}

// candidatosPanel: a quiénes se les ofrecería ese horario, en orden (el «Ofrecer a la lista · N»).
func (s *Servicio) candidatosPanel(w http.ResponseWriter, r *http.Request) error {
	q := r.URL.Query()
	inicio, err := time.Parse(time.RFC3339, q.Get("inicio"))
	if err != nil || q.Get("servicio") == "" || q.Get("profesional") == "" {
		return problema.Invalido("Faltan servicio, profesional e inicio (RFC 3339).")
	}
	ctx := r.Context()
	var out []Anotado
	err = s.Pool.EnNegocio(ctx, sesion.De(ctx).NegocioID, func(tx pgx.Tx) (err error) {
		out, err = candidatos(ctx, tx, q.Get("servicio"), q.Get("profesional"), inicio, zona(ctx, tx), maxCola)
		return err
	})
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusOK, map[string]any{"espera": out})
	return nil
}

// ofrecer: recepción lanza la oferta de un hueco («Ofrecer a la lista»).
func (s *Servicio) ofrecer(w http.ResponseWriter, r *http.Request) error {
	var in pedidoHueco
	if err := web.Leer(w, r, &in); err != nil {
		return err
	}
	if err := in.validar(); err != nil {
		return err
	}
	ctx := r.Context()
	var o Oferta
	err := s.Pool.EnNegocio(ctx, sesion.De(ctx).NegocioID, func(tx pgx.Tx) error {
		id, err := crearOferta(ctx, tx, in.ServicioID, in.ProfesionalID, in.Inicio, "panel", s.ahora())
		if err != nil {
			return err
		}
		if o, err = leerOferta(ctx, tx, id); err != nil {
			return err
		}
		return auditoria.Anotar(ctx, tx, auditoria.Registro{Accion: "oferta.creada", Entidad: "oferta", EntidadID: id,
			Detalle: map[string]any{"inicio": in.Inicio, "profesional": o.Profesional, "cola": len(o.Cola)}})
	})
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusCreated, o)
	return nil
}

// conOferta corre fn con la oferta (que se está ofreciendo) bloqueada.
func (s *Servicio) conOferta(r *http.Request, accion string, fn func(tx pgx.Tx, o *Oferta) error) (Oferta, error) {
	ctx := r.Context()
	var o Oferta
	err := s.Pool.EnNegocio(ctx, sesion.De(ctx).NegocioID, func(tx pgx.Tx) error {
		var estado string
		if err := tx.QueryRow(ctx, "SELECT id::text, estado FROM ofertas WHERE id::text = $1 FOR UPDATE", r.PathValue("id")).
			Scan(&o.ID, &estado); err != nil {
			return err
		}
		if estado != "ofreciendo" {
			return problema.Conflicto("oferta_cerrada", "Ese horario ya no se está ofreciendo.")
		}
		var err error
		if o, err = leerOferta(ctx, tx, o.ID); err != nil {
			return err
		}
		if err := fn(tx, &o); err != nil {
			return err
		}
		if o, err = leerOferta(ctx, tx, o.ID); err != nil {
			return err
		}
		return auditoria.Anotar(ctx, tx, auditoria.Registro{Accion: "oferta." + accion, Entidad: "oferta", EntidadID: o.ID,
			Detalle: map[string]any{"inicio": o.Inicio, "profesional": o.Profesional}})
	})
	return o, err
}

// frenar: el panel deja de ofrecer el horario.
func (s *Servicio) frenar(w http.ResponseWriter, r *http.Request) error {
	o, err := s.conOferta(r, "frenada", func(tx pgx.Tx, o *Oferta) error {
		return cerrar(r.Context(), tx, o.ID, "frenada", s.ahora())
	})
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusOK, o)
	return nil
}

// cargar («Cargarlo yo»): recepción resuelve el hueco. Sin esperaId deja de ofrecerlo y devuelve el
// hueco para cargar el turno a mano; con esperaId se lo reserva a esa persona de la lista (confirmado,
// como un turno del panel).
func (s *Servicio) cargar(w http.ResponseWriter, r *http.Request) error {
	var in struct {
		EsperaID string `json:"esperaId"`
	}
	if r.ContentLength != 0 {
		if err := web.Leer(w, r, &in); err != nil {
			return err
		}
	}
	ctx := r.Context()
	ahora := s.ahora()
	var t *turnos.Turno
	o, err := s.conOferta(r, "cargada", func(tx pgx.Tx, o *Oferta) error {
		if err := cerrar(ctx, tx, o.ID, "cargada", ahora); err != nil {
			return err
		}
		if in.EsperaID == "" {
			return nil
		}
		if s.Turnos == nil {
			return errors.New("espera: falta Servicio.Turnos")
		}
		var a Anotado
		if err := leer(ctx, tx, in.EsperaID, &a); err != nil {
			return err
		}
		nuevo, err := s.Turnos.ReservarEn(ctx, tx, turnos.Pedido{ServicioID: o.ServicioID, ProfesionalID: o.ProfesionalID, Inicio: o.Inicio,
			Cliente: turnos.Cliente{Nombre: a.Nombre, Telefono: a.Telefono, Email: a.Email}, Nota: "Le dimos el lugar desde la lista de espera.",
			SinAnticipacion: true}, "panel")
		if err != nil {
			return err
		}
		t = &nuevo
		if _, err := tx.Exec(ctx, "UPDATE ofertas SET turno_id = $2 WHERE id::text = $1", o.ID, t.ID); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `UPDATE oferta_intentos SET estado = 'tomada', respondida_en = $3
			WHERE oferta_id::text = $1 AND espera_id::text = $2`, o.ID, a.ID, ahora); err != nil {
			return err
		}
		_, err = tx.Exec(ctx, "UPDATE espera SET avisado_en = $2 WHERE id::text = $1", a.ID, ahora)
		return err
	})
	if err != nil {
		return err
	}
	if t != nil {
		s.Turnos.Confirmado(ctx, sesion.De(ctx).NegocioID, t, "reservado")
	}
	web.JSON(w, http.StatusOK, map[string]any{"oferta": o, "turno": t})
	return nil
}

func (s *Servicio) verAjustes(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	var a AjustesEspera
	err := s.Pool.EnNegocio(ctx, sesion.De(ctx).NegocioID, func(tx pgx.Tx) (err error) {
		a, err = leerAjustes(ctx, tx)
		return err
	})
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusOK, a)
	return nil
}

func (s *Servicio) guardarAjustes(w http.ResponseWriter, r *http.Request) error {
	var a AjustesEspera
	if err := web.Leer(w, r, &a); err != nil {
		return err
	}
	if a.Minutos < 5 || a.Minutos > 720 {
		return problema.Invalido("El tiempo para tomarlo va de 5 a 720 minutos.", problema.Campo{Ruta: "minutos", Mensaje: "fuera de rango"})
	}
	ctx := r.Context()
	err := s.Pool.EnNegocio(ctx, sesion.De(ctx).NegocioID, func(tx pgx.Tx) error {
		antes, err := leerAjustes(ctx, tx)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `UPDATE ajustes SET espera_minutos = $1, espera_automatica = $2, actualizado_en = now()
			WHERE negocio_id = app.negocio_actual()`, a.Minutos, a.Automatica); err != nil {
			return err
		}
		return auditoria.Anotar(ctx, tx, auditoria.Registro{Accion: "ajustes.espera", Entidad: "ajustes",
			Detalle: map[string]any{"antes": antes, "despues": a}})
	})
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusOK, a)
	return nil
}

// ---- El link de la persona («Se liberó lo que esperabas»)

// OfertaPublica: lo que ve quien recibió el link.
type OfertaPublica struct {
	// ofrecida (la puede tomar o pasar), tomada, paso, vencida o cerrada (se ocupó, la frenaron).
	Estado      string           `json:"estado"`
	Nombre      string           `json:"nombre"`
	Negocio     negocios.Negocio `json:"negocio"`
	Servicio    string           `json:"servicio"`
	Profesional string           `json:"profesional"`
	Inicio      time.Time        `json:"inicio"`
	Fin         time.Time        `json:"fin"`
	Precio      int64            `json:"precio"`
	VenceEn     *time.Time       `json:"venceEn"`
	// Turno: con estado tomada, el código del link del turno (/turno/{codigo}).
	Turno *string `json:"turno"`
}

func (s *Servicio) negocioDeOferta(ctx context.Context, token string) (string, error) {
	var id *string
	err := s.Pool.EnTx(ctx, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, "SELECT app.negocio_de_oferta($1)::text", token).Scan(&id)
	})
	if err != nil {
		return "", err
	}
	if id == nil {
		return "", problema.NoEncontrado("No encontramos ese lugar.")
	}
	return *id, nil
}

// delLink: datos del intento del token, con la oferta y el intento bloqueados (en ese orden).
type delLink struct {
	intento, oferta, estadoOferta string
	estado                        string
	esperaID                      *string
	vence                         *time.Time
	pub                           OfertaPublica
}

func (s *Servicio) conLink(ctx context.Context, token string, fn func(tx pgx.Tx, l *delLink) error) (OfertaPublica, string, error) {
	negocio, err := s.negocioDeOferta(ctx, token)
	if err != nil {
		return OfertaPublica{}, "", err
	}
	var l delLink
	err = s.Pool.EnNegocio(ctx, negocio, func(tx pgx.Tx) error {
		if err := tx.QueryRow(ctx, "SELECT oferta_id::text FROM oferta_intentos WHERE token = $1", token).Scan(&l.oferta); err != nil {
			return err
		}
		p := &l.pub
		if err := tx.QueryRow(ctx, `SELECT o.estado, o.servicio, o.profesional, o.inicio, o.fin, o.precio, t.codigo
			FROM ofertas o LEFT JOIN turnos t ON t.id = o.turno_id WHERE o.id::text = $1 FOR UPDATE OF o`, l.oferta).
			Scan(&l.estadoOferta, &p.Servicio, &p.Profesional, &p.Inicio, &p.Fin, &p.Precio, &p.Turno); err != nil {
			return err
		}
		if err := tx.QueryRow(ctx, `SELECT id::text, espera_id::text, nombre, estado, vence_en FROM oferta_intentos WHERE token = $1 FOR UPDATE`, token).
			Scan(&l.intento, &l.esperaID, &p.Nombre, &l.estado, &l.vence); err != nil {
			return err
		}
		var err error
		if p.Negocio, err = negocios.Leer(ctx, tx); err != nil {
			return err
		}
		if err := fn(tx, &l); err != nil {
			return err
		}
		p.Estado, p.VenceEn = estadoPublico(l.estado, l.vence, s.ahora()), l.vence
		if p.Estado != "tomada" {
			p.Turno = nil
		}
		return nil
	})
	l.pub.Nombre = primerNombre(l.pub.Nombre)
	return l.pub, negocio, err
}

func estadoPublico(estado string, vence *time.Time, ahora time.Time) string {
	switch estado {
	case "ofrecida":
		if vence != nil && vence.After(ahora) {
			return "ofrecida"
		}
		return "vencida"
	case "tomada", "paso", "vencida":
		return estado
	}
	return "cerrada"
}

func vigente(l *delLink, ahora time.Time) error {
	switch estadoPublico(l.estado, l.vence, ahora) {
	case "ofrecida":
		if l.estadoOferta == "ofreciendo" {
			return nil
		}
	case "vencida":
		return problema.Conflicto("oferta_vencida", "Pasó el tiempo para tomarlo y ya se lo ofrecimos a otra persona.")
	}
	return problema.Conflicto("oferta_cerrada", "Este lugar ya no está disponible.")
}

func (s *Servicio) verLink(w http.ResponseWriter, r *http.Request) error {
	p, _, err := s.conLink(r.Context(), r.PathValue("token"), func(pgx.Tx, *delLink) error { return nil })
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusOK, p)
	return nil
}

// tomar («Tomar el lugar»): reserva el turno con los datos con que se anotó, con las reglas de la
// página pública (si el negocio cobra seña, el turno queda esperando el pago y viene el link).
func (s *Servicio) tomar(w http.ResponseWriter, r *http.Request) error {
	if s.Turnos == nil {
		return errors.New("espera: falta Servicio.Turnos")
	}
	ctx := r.Context()
	ahora := s.ahora()
	var t turnos.Turno
	_, negocio, err := s.conLink(ctx, r.PathValue("token"), func(tx pgx.Tx, l *delLink) error {
		if err := vigente(l, ahora); err != nil {
			return err
		}
		if l.esperaID == nil {
			return problema.Conflicto("oferta_cerrada", "Este lugar ya no está disponible.")
		}
		var a Anotado
		if err := leer(ctx, tx, *l.esperaID, &a); err != nil {
			return err
		}
		var servicioID, profesionalID string
		if err := tx.QueryRow(ctx, "SELECT servicio_id::text, profesional_id::text FROM ofertas WHERE id::text = $1", l.oferta).
			Scan(&servicioID, &profesionalID); err != nil {
			return err
		}
		// Primero se cierra la oferta: así el turno nuevo no la «cae» (app.turno_ocupa).
		if _, err := tx.Exec(ctx, "UPDATE ofertas SET estado = 'tomada', cerrada_en = $2 WHERE id::text = $1", l.oferta, ahora); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, "UPDATE oferta_intentos SET estado = 'tomada', respondida_en = $2 WHERE id::text = $1", l.intento, ahora); err != nil {
			return err
		}
		var err error
		t, err = s.Turnos.ReservarEn(ctx, tx, turnos.Pedido{ServicioID: servicioID, ProfesionalID: profesionalID, Inicio: l.pub.Inicio,
			Cliente: turnos.Cliente{Nombre: a.Nombre, Telefono: a.Telefono, Email: a.Email}, Nota: "Tomó el lugar desde la lista de espera.",
			SinAnticipacion: true}, "web")
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, "UPDATE ofertas SET turno_id = $2 WHERE id::text = $1", l.oferta, t.ID); err != nil {
			return err
		}
		l.estado = "tomada"
		return nil
	})
	if err != nil {
		return err
	}
	s.Turnos.Confirmado(ctx, negocio, &t, "reservado_espera")
	t.Cliente.ID = ""
	web.JSON(w, http.StatusCreated, t)
	return nil
}

// pasar («No me sirve, que pase»): vuelve a esperar otros horarios y el lugar pasa a la siguiente.
func (s *Servicio) pasar(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	ahora := s.ahora()
	p, _, err := s.conLink(ctx, r.PathValue("token"), func(tx pgx.Tx, l *delLink) error {
		if err := vigente(l, ahora); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, "UPDATE oferta_intentos SET estado = 'paso', respondida_en = $2 WHERE id::text = $1", l.intento, ahora); err != nil {
			return err
		}
		if l.esperaID != nil {
			if _, err := tx.Exec(ctx, "UPDATE espera SET avisado_en = NULL WHERE id::text = $1", *l.esperaID); err != nil {
				return err
			}
		}
		l.estado = "paso"
		return avanzar(ctx, tx, l.oferta, ahora)
	})
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusOK, p)
	return nil
}

// ---- Worker

// Trabajo del worker: pasa a la siguiente las ofertas cuyo link venció sin respuesta (y cierra las
// que llegaron a su hora).
func (s *Servicio) Trabajo() trabajos.Trabajo {
	return trabajos.Trabajo{Nombre: "ofertas", Cada: 30 * time.Second, Correr: s.Avanzar}
}

// Avanzar hace una pasada sobre las ofertas de todos los negocios.
func (s *Servicio) Avanzar(ctx context.Context, ahora time.Time) (any, error) {
	avanzadas := 0
	err := trabajos.PorNegocio(ctx, s.Pool, func(_ string, tx pgx.Tx) error {
		filas, err := tx.Query(ctx, `SELECT o.id::text FROM ofertas o WHERE o.estado = 'ofreciendo' AND (o.inicio <= $1
			OR NOT EXISTS (SELECT 1 FROM oferta_intentos i WHERE i.oferta_id = o.id AND i.estado = 'ofrecida' AND i.vence_en > $1))
			ORDER BY o.inicio LIMIT 200`, ahora)
		if err != nil {
			return err
		}
		ids, err := pgx.CollectRows(filas, pgx.RowTo[string])
		if err != nil {
			return err
		}
		for _, id := range ids {
			if err := avanzar(ctx, tx, id, ahora); err != nil {
				return err
			}
			avanzadas++
		}
		return nil
	})
	return map[string]int{"avanzadas": avanzadas}, err
}
