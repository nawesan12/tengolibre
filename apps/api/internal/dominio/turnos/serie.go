package turnos

// Turno fijo (propuesta 03 del diseño v2): desde Nuevo turno del panel se elige cada cuánto se repite
// (cada 1 a 4 semanas) y hasta cuándo (una fecha o una cantidad). La vista previa dice, fecha por
// fecha, si está libre o choca, y para las que chocan propone el horario libre más cercano de ese día
// con el mismo profesional. La reserva de la serie es atómica: o entran todas las que el panel
// aceptó, o ninguna. Las reglas de disponibilidad son las de siempre (motor y elegir); lo único que
// cambia es la vista, que en el panel llega hasta la última fecha de la serie.

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"

	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/catalogo"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/disponibilidad"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/auditoria"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/problema"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/sesion"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/web"
)

// MaxSerie: cuántos turnos puede tener un turno fijo (un año, semana por medio, sobra).
const MaxSerie = 52

// MaxDiasSerie: hasta cuántos días después del primero puede ir la última fecha.
const MaxDiasSerie = 366

// Motivos por los que una fecha de la serie choca.
const (
	MotivoOcupado   = "ocupado"    // otro turno del profesional
	MotivoBloqueado = "bloqueado"  // bloqueo, vacaciones o un horario bloqueado
	MotivoNoAtiende = "no_atiende" // fuera del horario del profesional ese día
	MotivoPasado    = "pasado"     // ya pasó o no llega a la anticipación mínima
)

// PedidoSerie de la vista previa: el primer turno y cómo se repite. Va Hasta (AAAA-MM-DD, inclusive)
// o Cantidad (contando el primero), no los dos.
type PedidoSerie struct {
	ServicioID    string    `json:"servicioId"`
	ProfesionalID string    `json:"profesionalId"`
	Inicio        time.Time `json:"inicio"`
	CadaSemanas   int       `json:"cadaSemanas"`
	Hasta         string    `json:"hasta,omitempty"`
	Cantidad      int       `json:"cantidad,omitempty"`
}

// Alternativa para una fecha que choca: el horario libre más cercano ese día.
type Alternativa struct {
	Inicio time.Time `json:"inicio"`
	Hora   string    `json:"hora"`
}

// FechaSerie de la vista previa.
type FechaSerie struct {
	Inicio time.Time `json:"inicio"`
	Fecha  string    `json:"fecha"`
	Hora   string    `json:"hora"`
	// Estado: libre o choca.
	Estado string `json:"estado"`
	// Motivo (sólo si choca): ocupado, bloqueado, no_atiende o pasado.
	Motivo string `json:"motivo,omitempty"`
	// Alternativa (sólo si choca): null si ese día no queda nada libre con ese profesional.
	Alternativa *Alternativa `json:"alternativa"`
}

// Previa de un turno fijo.
type Previa struct {
	CadaSemanas   int          `json:"cadaSemanas"`
	ServicioID    string       `json:"servicioId"`
	Servicio      string       `json:"servicio"`
	ProfesionalID string       `json:"profesionalId"`
	Profesional   string       `json:"profesional"`
	Fechas        []FechaSerie `json:"fechas"`
	Libres        int          `json:"libres"`
	Chocan        int          `json:"chocan"`
}

// Serie como la ve el panel.
type Serie struct {
	ID          string    `json:"id"`
	CadaSemanas int       `json:"cadaSemanas"`
	ClienteID   string    `json:"clienteId"`
	CreadoEn    time.Time `json:"creadoEn"`
	Turnos      []Turno   `json:"turnos"`
}

// fechasDe arma las fechas de la serie, a la misma hora del reloj del negocio.
func fechasDe(in PedidoSerie, zona *time.Location) ([]time.Time, error) {
	if in.CadaSemanas < 1 || in.CadaSemanas > 4 {
		return nil, problema.Invalido("Se repite cada 1, 2, 3 o 4 semanas.", problema.Campo{Ruta: "cadaSemanas", Mensaje: "entre 1 y 4"})
	}
	if in.Inicio.IsZero() {
		return nil, problema.Invalido("Falta el primer turno.", problema.Campo{Ruta: "inicio", Mensaje: "requerido"})
	}
	if (in.Hasta == "") == (in.Cantidad == 0) {
		return nil, problema.Invalido("Decí hasta qué fecha o cuántos turnos.", problema.Campo{Ruta: "hasta", Mensaje: "hasta o cantidad"})
	}
	ini := in.Inicio.In(zona)
	tope := MaxSerie
	limite := ini.AddDate(0, 0, MaxDiasSerie)
	if in.Cantidad != 0 {
		if in.Cantidad < 2 || in.Cantidad > MaxSerie {
			return nil, problema.Invalido("Un turno fijo tiene de 2 a "+strconv.Itoa(MaxSerie)+" turnos.",
				problema.Campo{Ruta: "cantidad", Mensaje: "entre 2 y " + strconv.Itoa(MaxSerie)})
		}
		tope = in.Cantidad
	} else {
		h, err := time.ParseInLocation(time.DateOnly, in.Hasta, zona)
		if err != nil {
			return nil, problema.Invalido("hasta va como AAAA-MM-DD.", problema.Campo{Ruta: "hasta", Mensaje: "inválido"})
		}
		h = h.AddDate(0, 0, 1) // inclusive
		if !h.After(ini) || h.After(limite.AddDate(0, 0, 1)) {
			return nil, problema.Invalido("hasta va después del primer turno y dentro del año.", problema.Campo{Ruta: "hasta", Mensaje: "fuera de rango"})
		}
		limite = h
	}
	var out []time.Time
	for k := 0; len(out) < tope; k++ {
		f := ini.AddDate(0, 0, 7*in.CadaSemanas*k)
		if !f.Before(limite) {
			break
		}
		out = append(out, f)
	}
	if len(out) < 2 {
		return nil, problema.Invalido("Con esa fecha de fin queda un solo turno: no hace falta un turno fijo.",
			problema.Campo{Ruta: "hasta", Mensaje: "un solo turno"})
	}
	return out, nil
}

// reglasHasta: las del negocio, con la vista estirada hasta ultima (el panel reserva más allá de lo
// que ve la web; el resto de las reglas no cambia).
func reglasHasta(c contexto, ahora, ultima time.Time) disponibilidad.Reglas {
	r := c.reglas()
	a := ahora.In(c.zona)
	hoy := time.Date(a.Year(), a.Month(), a.Day(), 0, 0, 0, 0, c.zona)
	u := ultima.In(c.zona)
	dias := int(time.Date(u.Year(), u.Month(), u.Day(), 12, 0, 0, 0, c.zona).Sub(hoy).Hours()/24) + 1
	r.DiasVista = max(r.DiasVista, dias)
	return r
}

// servicioYProfesional del pedido, los dos activos y el profesional haciendo el servicio.
func servicioYProfesional(ctx context.Context, tx pgx.Tx, servicioID, profesionalID string) (catalogo.Prestacion, catalogo.Profesional, error) {
	srv, err := catalogo.LeerServicio(ctx, tx, servicioID)
	if err != nil || !srv.Activo {
		return srv, catalogo.Profesional{}, problema.NoEncontrado("Ese servicio no existe o no se puede reservar.")
	}
	if profesionalID == "" {
		return srv, catalogo.Profesional{}, problema.Invalido("Un turno fijo va con un profesional.",
			problema.Campo{Ruta: "profesionalId", Mensaje: "requerido"})
	}
	p, err := catalogo.LeerProfesional(ctx, tx, profesionalID, false)
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && (!p.Activo || !p.Hace(srv.ID))) {
		return srv, p, problema.NoEncontrado("Ese profesional no existe o no hace ese servicio.")
	}
	return srv, p, err
}

// estadoDe una fecha: libre, o por qué choca y la alternativa. El motivo sale del mismo motor,
// sacando de a una las cosas que tapan el horario.
func estadoDe(q disponibilidad.Consulta, inicio, ahora time.Time) FechaSerie {
	f := FechaSerie{Inicio: inicio, Fecha: inicio.In(q.Zona).Format(time.DateOnly), Hora: inicio.In(q.Zona).Format("15:04"), Estado: "libre"}
	hs := disponibilidad.Dia(q)
	if disponibilidad.Libre(q, inicio) {
		return f
	}
	f.Estado = "choca"
	sinTurnos := q
	sinTurnos.Profesional.Turnos = nil
	sinNada := sinTurnos
	sinNada.Profesional.Bloqueos, sinNada.Profesional.Semanales, sinNada.Profesional.Excepciones = nil, nil, nil
	switch {
	case inicio.Before(ahora.Add(time.Duration(q.Reglas.AnticipacionMin) * time.Minute)):
		f.Motivo = MotivoPasado
	case disponibilidad.Libre(sinTurnos, inicio):
		f.Motivo = MotivoOcupado
	case disponibilidad.Libre(sinNada, inicio):
		f.Motivo = MotivoBloqueado
	default:
		f.Motivo = MotivoNoAtiende
	}
	f.Alternativa = masCercano(hs, inicio)
	return f
}

// masCercano: el horario libre de hs más cerca de inicio (si empatan, el más tarde).
func masCercano(hs []disponibilidad.Horario, inicio time.Time) *Alternativa {
	var mejor *disponibilidad.Horario
	var dist time.Duration
	for i := range hs {
		h := &hs[i]
		if !h.Libre {
			continue
		}
		d := h.Inicio.Sub(inicio)
		if d < 0 {
			d = -d
		}
		if mejor == nil || d < dist || (d == dist && h.Inicio.After(mejor.Inicio)) {
			mejor, dist = h, d
		}
	}
	if mejor == nil {
		return nil
	}
	return &Alternativa{Inicio: mejor.Inicio, Hora: mejor.Hora}
}

// previa de la serie en el negocio fijado.
func previa(ctx context.Context, tx pgx.Tx, in PedidoSerie, ahora time.Time) (Previa, error) {
	c, err := cargarContexto(ctx, tx)
	if err != nil {
		return Previa{}, err
	}
	fechas, err := fechasDe(in, c.zona)
	if err != nil {
		return Previa{}, err
	}
	srv, p, err := servicioYProfesional(ctx, tx, in.ServicioID, in.ProfesionalID)
	if err != nil {
		return Previa{}, err
	}
	reglas := reglasHasta(c, ahora, fechas[len(fechas)-1])
	out := Previa{CadaSemanas: in.CadaSemanas, ServicioID: srv.ID, Servicio: srv.Nombre, ProfesionalID: p.ID, Profesional: p.Nombre,
		Fechas: []FechaSerie{}}
	for _, f := range fechas {
		fecha := f.Format(time.DateOnly)
		d, err := datosDelDia(ctx, tx, p, fecha, c.zona, "")
		if err != nil {
			return out, err
		}
		x := estadoDe(disponibilidad.Consulta{Profesional: d, Fecha: fecha, Duracion: srv.Duracion, Reglas: reglas, Ahora: ahora, Zona: c.zona},
			f, ahora)
		if x.Estado == "libre" {
			out.Libres++
		} else {
			out.Chocan++
		}
		out.Fechas = append(out.Fechas, x)
	}
	return out, nil
}

func (s *Servicio) previaSerie(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	var in PedidoSerie
	if err := web.Leer(w, r, &in); err != nil {
		return err
	}
	var out Previa
	err := s.Pool.EnNegocio(ctx, sesion.De(ctx).NegocioID, func(tx pgx.Tx) (err error) {
		out, err = previa(ctx, tx, in, s.ahora())
		return err
	})
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusOK, out)
	return nil
}

// ReservaSerie: los turnos que el panel aceptó de la vista previa (con las alternativas ya elegidas).
type ReservaSerie struct {
	ServicioID    string  `json:"servicioId"`
	ProfesionalID string  `json:"profesionalId"`
	CadaSemanas   int     `json:"cadaSemanas"`
	Cliente       Cliente `json:"cliente"`
	// ClienteID: un cliente que ya está en el negocio (en lugar de Cliente).
	ClienteID string `json:"clienteId,omitempty"`
	Nota      string `json:"nota"`
	Turnos    []struct {
		Inicio time.Time `json:"inicio"`
	} `json:"turnos"`
}

func (s *Servicio) reservarSerie(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	var in ReservaSerie
	if err := web.Leer(w, r, &in); err != nil {
		return err
	}
	if in.CadaSemanas < 1 || in.CadaSemanas > 4 {
		return problema.Invalido("Se repite cada 1, 2, 3 o 4 semanas.", problema.Campo{Ruta: "cadaSemanas", Mensaje: "entre 1 y 4"})
	}
	if len(in.Turnos) < 1 || len(in.Turnos) > MaxSerie {
		return problema.Invalido("Elegí de 1 a "+strconv.Itoa(MaxSerie)+" turnos.", problema.Campo{Ruta: "turnos", Mensaje: "cantidad"})
	}
	for i, t := range in.Turnos {
		if t.Inicio.IsZero() || (i > 0 && !t.Inicio.After(in.Turnos[i-1].Inicio)) {
			return problema.Invalido("Los turnos van en orden y sin repetir.", problema.Campo{Ruta: "turnos[" + strconv.Itoa(i) + "].inicio", Mensaje: "orden"})
		}
	}
	if in.Turnos[len(in.Turnos)-1].Inicio.Sub(in.Turnos[0].Inicio) > MaxDiasSerie*24*time.Hour {
		return problema.Invalido("Un turno fijo llega hasta un año.", problema.Campo{Ruta: "turnos", Mensaje: "más de un año"})
	}
	if utf8.RuneCountInString(in.Nota) > 500 {
		return problema.Invalido("La nota es muy larga (hasta 500).")
	}
	negocio := sesion.De(ctx).NegocioID
	ahora := s.ahora()
	var serie Serie
	err := s.Pool.EnNegocio(ctx, negocio, func(tx pgx.Tx) error {
		c, err := cargarContexto(ctx, tx)
		if err != nil {
			return err
		}
		if _, _, err := servicioYProfesional(ctx, tx, in.ServicioID, in.ProfesionalID); err != nil {
			return err
		}
		clienteID := in.ClienteID
		if clienteID != "" {
			err := tx.QueryRow(ctx, "SELECT id::text FROM clientes WHERE id::text = $1 AND suprimido_en IS NULL", clienteID).Scan(&clienteID)
			if errors.Is(err, pgx.ErrNoRows) {
				return problema.NoEncontrado("No encontramos ese cliente.")
			}
			if err != nil {
				return err
			}
		} else {
			if err := validarCliente(&in.Cliente); err != nil {
				return err
			}
			if clienteID, err = clienteDe(ctx, tx, in.Cliente); err != nil {
				return err
			}
		}
		if err := tx.QueryRow(ctx, `INSERT INTO series (negocio_id, cliente_id, cada_semanas, clerk_id) VALUES (app.negocio_actual(), $1, $2, $3)
			RETURNING id::text, cada_semanas, cliente_id::text, creado_en`, clienteID, in.CadaSemanas, sesion.De(ctx).ClerkID).
			Scan(&serie.ID, &serie.CadaSemanas, &serie.ClienteID, &serie.CreadoEn); err != nil {
			return err
		}
		reglas := reglasHasta(c, ahora, in.Turnos[len(in.Turnos)-1].Inicio)
		for i, x := range in.Turnos {
			pedido := Pedido{ServicioID: in.ServicioID, ProfesionalID: in.ProfesionalID, Inicio: x.Inicio, Nota: in.Nota,
				ClienteID: clienteID, SerieID: serie.ID, reglas: &reglas}
			t, err := reservar(ctx, tx, pedido, "panel", ahora)
			var p *problema.Problema
			if errors.As(err, &p) && p.Codigo == "horario_ocupado" {
				l := x.Inicio.In(c.zona)
				p.Detalle = "El turno del " + l.Format("02/01") + " a las " + l.Format("15:04") + " ya no está libre. Revisá la vista previa."
				p.Campos = []problema.Campo{{Ruta: "turnos[" + strconv.Itoa(i) + "].inicio", Mensaje: "ocupado"}}
				return p
			}
			if err != nil {
				return err
			}
			if err := s.alReservar(ctx, tx, &t, pedido, "panel"); err != nil {
				return err
			}
			serie.Turnos = append(serie.Turnos, t)
		}
		return auditoria.Anotar(ctx, tx, auditoria.Registro{Accion: "serie.creada", Entidad: "serie", EntidadID: serie.ID,
			Detalle: map[string]any{"cadaSemanas": serie.CadaSemanas, "turnos": len(serie.Turnos)}})
	})
	if err != nil {
		return err
	}
	for i := range serie.Turnos {
		s.despuesDeReservar(ctx, negocio, &serie.Turnos[i])
	}
	// Un solo aviso (el del primer turno): doce mails iguales seguidos serían spam. Los recordatorios
	// de cada turno llegan igual, como los de cualquier turno.
	s.avisar(ctx, negocio, serie.Turnos[0].ID, "reservado")
	web.JSON(w, http.StatusCreated, serie)
	return nil
}

func turnosDeSerie(ctx context.Context, tx pgx.Tx, serieID string) ([]Turno, error) {
	filas, err := tx.Query(ctx, "SELECT "+columnasTurno+desdeTurnos+" WHERE t.serie_id = $1 ORDER BY t.inicio", serieID)
	if err != nil {
		return nil, err
	}
	defer filas.Close()
	out := []Turno{}
	for filas.Next() {
		var t Turno
		if err := escanearTurno(filas, &t); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, filas.Err()
}

func (s *Servicio) verSerie(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	var serie Serie
	err := s.Pool.EnNegocio(ctx, sesion.De(ctx).NegocioID, func(tx pgx.Tx) error {
		err := tx.QueryRow(ctx, "SELECT id::text, cada_semanas, cliente_id::text, creado_en FROM series WHERE id::text = $1", r.PathValue("id")).
			Scan(&serie.ID, &serie.CadaSemanas, &serie.ClienteID, &serie.CreadoEn)
		if errors.Is(err, pgx.ErrNoRows) {
			return problema.NoEncontrado("No encontramos ese turno fijo.")
		}
		if err != nil {
			return err
		}
		serie.Turnos, err = turnosDeSerie(ctx, tx, serie.ID)
		return err
	})
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusOK, serie)
	return nil
}

// cancelarSiguientes: «este y los siguientes» de un turno fijo. Cancela el turno y los de su serie que
// vienen después y siguen activos; los atendidos, ausentes y ya cancelados quedan como están.
func (s *Servicio) cancelarSiguientes(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	negocio := sesion.De(ctx).NegocioID
	var cancelados []string
	var serie Serie
	err := s.Pool.EnNegocio(ctx, negocio, func(tx pgx.Tx) error {
		t, err := leerTurno(ctx, tx, "t.id::text = $1", r.PathValue("id"), true)
		if errors.Is(err, pgx.ErrNoRows) {
			return problema.NoEncontrado("No encontramos ese turno.")
		}
		if err != nil {
			return err
		}
		if t.SerieID == nil {
			return problema.Conflicto("sin_serie", "Este turno no es parte de un turno fijo.")
		}
		filas, err := tx.Query(ctx, `SELECT id::text FROM turnos WHERE serie_id = $1 AND inicio >= $2
			AND estado IN ('pendiente_pago', 'pendiente', 'confirmado') ORDER BY inicio FOR UPDATE`, *t.SerieID, t.Inicio)
		if err != nil {
			return err
		}
		ids, err := pgx.CollectRows(filas, pgx.RowTo[string])
		if err != nil {
			return err
		}
		for _, id := range ids {
			if _, err := tx.Exec(ctx, "UPDATE turnos SET estado = 'cancelado', vence_en = NULL, actualizado_en = now() WHERE id = $1", id); err != nil {
				return err
			}
			if err := s.alCancelar(ctx, tx, id); err != nil {
				return err
			}
		}
		cancelados = ids
		serie.ID = *t.SerieID
		if err := tx.QueryRow(ctx, "SELECT cada_semanas, cliente_id::text, creado_en FROM series WHERE id = $1", serie.ID).
			Scan(&serie.CadaSemanas, &serie.ClienteID, &serie.CreadoEn); err != nil {
			return err
		}
		serie.Turnos, err = turnosDeSerie(ctx, tx, serie.ID)
		return err
	})
	if err != nil {
		return err
	}
	// Cada cancelación libera un horario: avisa al cliente y a la lista de espera, como siempre.
	for _, id := range cancelados {
		s.avisar(ctx, negocio, id, "cancelado")
	}
	if cancelados == nil {
		cancelados = []string{}
	}
	web.JSON(w, http.StatusOK, map[string]any{"cancelados": cancelados, "serie": serie})
	return nil
}
