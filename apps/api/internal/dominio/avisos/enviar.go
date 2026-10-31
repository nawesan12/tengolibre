package avisos

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/correo"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/db"
)

// MaxIntentos: después de esto el aviso queda fallido (y la campanita lo cuenta).
const MaxIntentos = 6

// EsperaReintento: espera creciente entre intentos: 1, 2, 4, 8, 16 minutos… con techo de seis horas.
func EsperaReintento(intentos int) time.Duration {
	d := time.Minute << max(intentos-1, 0)
	return min(d, 6*time.Hour)
}

// Enviador consume la cola: arma cada aviso pendiente y lo manda. Ahora es el reloj (tests).
type Enviador struct {
	Pool   *db.Pool
	Correo correo.Correo
	// Sitio público, para los links de los mails.
	Sitio string
	Ahora func() time.Time
	Lote  int
	Log   *slog.Logger
}

// ResultadoEnvio de una pasada.
type ResultadoEnvio struct {
	Enviados    int `json:"enviados"`
	Reintentos  int `json:"reintentos"`
	Fallidos    int `json:"fallidos"`
	Descartados int `json:"descartados"`
	Errores     int `json:"errores"`
}

func (e *Enviador) log() *slog.Logger {
	if e.Log != nil {
		return e.Log
	}
	return slog.Default()
}

// Correr es la pasada con la forma de un trabajo (trabajos.Trabajo): manda los nuevos y los
// reintentos que ya cumplieron su espera.
func (e *Enviador) Correr(ctx context.Context, ahora time.Time) (any, error) {
	return e.Pasada(ctx, ahora)
}

// Pasada manda lo que está pendiente a esa hora.
func (e *Enviador) Pasada(ctx context.Context, ahora time.Time) (ResultadoEnvio, error) {
	var res ResultadoEnvio
	lote := e.Lote
	if lote == 0 {
		lote = 100
	}
	type pendiente struct{ ID, Negocio string }
	var ps []pendiente
	err := e.Pool.EnTx(ctx, func(tx pgx.Tx) error {
		filas, err := tx.Query(ctx, "SELECT aviso_id::text, negocio_id::text FROM app.avisos_pendientes($1, $2)", ahora, lote)
		if err != nil {
			return err
		}
		ps, err = pgx.CollectRows(filas, pgx.RowToStructByPos[pendiente])
		return err
	})
	if err != nil {
		return res, err
	}
	for _, p := range ps {
		err := e.Pool.EnNegocio(ctx, p.Negocio, func(tx pgx.Tx) error { return e.uno(ctx, tx, p.ID, ahora, &res) })
		if err != nil {
			res.Errores++
			e.log().Error("avisos: no se pudo procesar", "aviso", p.ID, "error", err)
		}
	}
	return res, nil
}

// fila de la cola.
type fila struct {
	ID, Tipo, Para string
	TurnoID        *string
	Datos          map[string]string
	Intentos       int
}

func (e *Enviador) uno(ctx context.Context, tx pgx.Tx, id string, ahora time.Time, res *ResultadoEnvio) error {
	var f fila
	var datos []byte
	err := tx.QueryRow(ctx, `SELECT id::text, tipo, para, turno_id::text, datos, intentos FROM avisos
		WHERE id = $1 AND estado = 'pendiente' AND proximo_intento <= $2 FOR UPDATE SKIP LOCKED`, id, ahora).
		Scan(&f.ID, &f.Tipo, &f.Para, &f.TurnoID, &datos, &f.Intentos)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil // lo tomó otro worker, o ya no está pendiente
	}
	if err != nil {
		return err
	}
	f.Datos = map[string]string{}
	var crudos map[string]any
	_ = json.Unmarshal(datos, &crudos)
	for k, v := range crudos {
		f.Datos[k] = fmt.Sprint(v)
	}
	// El armado va en un savepoint: si falla una consulta, la transacción sigue sirviendo para
	// anotar el error.
	var m correo.Mensaje
	var descarte string
	sp, err := tx.Begin(ctx)
	if err != nil {
		return err
	}
	m, descarte, errArmar := e.armar(ctx, sp, f, ahora)
	if errArmar != nil {
		_ = sp.Rollback(ctx)
	} else if err := sp.Commit(ctx); err != nil {
		return err
	}
	if errArmar == nil && descarte != "" {
		res.Descartados++
		_, err := tx.Exec(ctx, "UPDATE avisos SET estado = 'descartado', ultimo_error = $2, para = $3 WHERE id = $1", f.ID, descarte, m.Para)
		return err
	}
	errEnvio := errArmar
	if errEnvio == nil {
		errEnvio = e.Correo.Enviar(ctx, m)
	}
	if errEnvio == nil {
		res.Enviados++
		_, err := tx.Exec(ctx, `UPDATE avisos SET estado = 'enviado', enviado_en = $2, intentos = intentos + 1, ultimo_error = NULL,
			para = $3 WHERE id = $1`, f.ID, ahora, m.Para)
		return err
	}
	intentos := f.Intentos + 1
	msj := errEnvio.Error()
	if len(msj) > 1000 {
		msj = msj[:1000]
	}
	e.log().Warn("avisos: no salió", "aviso", f.ID, "tipo", f.Tipo, "intento", intentos, "error", msj)
	if intentos >= MaxIntentos {
		res.Fallidos++
		if _, err := tx.Exec(ctx, "UPDATE avisos SET estado = 'fallido', intentos = $2, ultimo_error = $3, para = $4 WHERE id = $1",
			f.ID, intentos, msj, m.Para); err != nil {
			return err
		}
		texto := "Probamos " + strconv.Itoa(intentos) + " veces y no salió. Revisá el email o reenviálo desde Avisos."
		para := m.Para
		if para == "" {
			para = f.Para
		}
		return Notificar(ctx, tx, Notificacion{Tipo: NotiAvisoFallido, Titulo: "No pudimos mandar un mail a " + para, Texto: texto,
			TurnoID: valor(f.TurnoID)})
	}
	res.Reintentos++
	_, err = tx.Exec(ctx, "UPDATE avisos SET intentos = $2, ultimo_error = $3, proximo_intento = $4 WHERE id = $1",
		f.ID, intentos, msj, ahora.Add(EsperaReintento(intentos)))
	return err
}

func valor(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

// armar construye el mail del aviso con los datos de ahora. descarte no vacío: no se manda, por ese
// motivo (queda en ultimo_error para el panel).
func (e *Enviador) armar(ctx context.Context, tx pgx.Tx, f fila, ahora time.Time) (correo.Mensaje, string, error) {
	n, err := leerNegocio(ctx, tx)
	if err != nil {
		return correo.Mensaje{}, "", err
	}
	switch f.Tipo {
	case Reservado, Recordatorio, Cancelado, Reprogramado, Resena:
		return e.armarTurno(ctx, tx, f, n, ahora)
	case Espera:
		return e.armarEspera(ctx, tx, f, n)
	case Invitacion:
		return e.armarInvitacion(ctx, tx, f, n, ahora)
	case Resumen:
		return e.armarResumen(ctx, tx, f, n)
	}
	return correo.Mensaje{}, "tipo desconocido: " + f.Tipo, nil
}

func (e *Enviador) armarTurno(ctx context.Context, tx pgx.Tx, f fila, n negocio, ahora time.Time) (correo.Mensaje, string, error) {
	if f.TurnoID == nil {
		return correo.Mensaje{}, "el turno ya no existe", nil
	}
	t, err := leerTurno(ctx, tx, *f.TurnoID)
	if errors.Is(err, pgx.ErrNoRows) {
		return correo.Mensaje{}, "el turno ya no existe", nil
	}
	if err != nil {
		return correo.Mensaje{}, "", err
	}
	sin := correo.Mensaje{Para: t.ClienteEmail}
	if t.ClienteEmail == "" {
		return sin, "el cliente no dejó email", nil
	}
	p, err := PreferenciasDe(ctx, tx, t.ClienteID)
	if err != nil {
		return sin, "", err
	}
	if !p.Email {
		return sin, "el cliente pidió no recibir mails", nil
	}
	activo := t.Estado == "pendiente" || t.Estado == "confirmado" || t.Estado == "pendiente_pago"
	inicio, fin := t.Inicio.In(n.zona), t.Fin.In(n.zona)
	c := Carta{Encabezado: n.Nombre, Saludo: "Hola " + primerNombre(t.ClienteNombre) + ",", Baja: LinkPreferencias(e.Sitio, p.Token)}
	filas := []Fila{{"Servicio", t.Servicio}, {"Profesional", t.Profesional}, {"Fecha", Fecha(inicio)}, {"Hora", Hora(inicio)}}
	if l := n.lugar(); l != "" {
		filas = append(filas, Fila{"Dónde", l})
	}
	conICS := false
	switch f.Tipo {
	case Reservado:
		if !activo {
			return sin, "el turno ya no está activo", nil
		}
		c.Asunto, c.Titulo = "Turno reservado — "+n.Nombre, "Listo, tu turno quedó reservado"
		if t.Estado == "pendiente_pago" {
			c.Titulo = "Tu turno te espera: falta el pago"
		}
		c.Filas, c.Boton, c.Link = filas, "Ver o cambiar mi turno", LinkTurno(e.Sitio, t.Codigo)
		c.Notas = []string{"¡Te esperamos!"}
		conICS = true
	case Recordatorio:
		if !activo || !t.Inicio.After(ahora) {
			return sin, "el turno ya no está activo", nil
		}
		if v := f.Datos["inicio"]; v != "" {
			if antes, err := time.Parse(time.RFC3339Nano, v); err == nil && !antes.Equal(t.Inicio) {
				return sin, "el turno cambió de horario", nil
			}
		}
		if !p.Recordatorios {
			return sin, "el cliente pidió no recibir recordatorios", nil
		}
		c.Asunto, c.Titulo = "Mañana tenés turno — "+n.Nombre, "Mañana tenés turno"
		c.Filas, c.Boton, c.Link = filas, "Ver o cambiar mi turno", LinkTurno(e.Sitio, t.Codigo)
		c.Notas = []string{"¡Te esperamos!"}
	case Cancelado:
		c.Asunto, c.Titulo = "Turno cancelado — "+n.Nombre, "Tu turno fue cancelado"
		c.Filas, c.Boton, c.Link = filas[:4], "Reservar otro turno", LinkReserva(e.Sitio, n.Slug)
	case Reprogramado:
		if !activo {
			return sin, "el turno ya no está activo", nil
		}
		c.Asunto, c.Titulo = "Cambiamos tu turno — "+n.Nombre, "Tu turno cambió de horario"
		if v := f.Datos["anterior"]; v != "" {
			if antes, err := time.Parse(time.RFC3339Nano, v); err == nil {
				a := antes.In(n.zona)
				c.Parrafos = []string{"Antes era el " + Fecha(a) + " a las " + Hora(a) + ". Ahora queda así:"}
			}
		}
		c.Filas, c.Boton, c.Link = filas, "Ver o cambiar mi turno", LinkTurno(e.Sitio, t.Codigo)
		conICS = true
	case Resena:
		token := f.Datos["token"]
		var vence time.Time
		var dejada bool
		err := tx.QueryRow(ctx, `SELECT vence_en, EXISTS (SELECT 1 FROM resenas WHERE turno_id = $2) FROM resena_tokens WHERE token = $1`,
			token, t.ID).Scan(&vence, &dejada)
		if errors.Is(err, pgx.ErrNoRows) {
			return sin, "el pedido de reseña ya no existe", nil
		}
		if err != nil {
			return sin, "", err
		}
		if dejada {
			return sin, "ya dejó su reseña", nil
		}
		if !vence.After(ahora) {
			return sin, "el pedido de reseña venció", nil
		}
		c.Asunto, c.Titulo = "¿Cómo fue tu visita? — "+n.Nombre, "¿Cómo te fue?"
		c.Parrafos = []string{"Pasaste por " + n.Nombre + " por un " + t.Servicio + ". Si tenés un minuto, contanos cómo estuvo: elegí las estrellas y listo."}
		c.Boton, c.Link = "Dejar mi reseña", LinkResena(e.Sitio, token)
		c.Notas = []string{"El enlace vence en 7 días."}
	}
	m := c.Mensaje(t.ClienteEmail)
	if conICS {
		desc := t.Servicio + " con " + t.Profesional + "\n" + LinkTurno(e.Sitio, t.Codigo)
		m.Adjuntos = []correo.Adjunto{{Nombre: "turno.ics", Tipo: "text/calendar; charset=utf-8; method=PUBLISH",
			Datos: ICS(t.ID, t.Servicio+" — "+n.Nombre, desc, n.lugar(), inicio, fin, ahora)}}
	}
	return m, "", nil
}

func (e *Enviador) armarEspera(ctx context.Context, tx pgx.Tx, f fila, n negocio) (correo.Mensaje, string, error) {
	var nombre, email, servicio, servicioID string
	err := tx.QueryRow(ctx, `SELECT e.nombre, e.email, s.nombre, s.id::text FROM espera e JOIN servicios s ON s.id = e.servicio_id
		WHERE e.id = $1`, f.Datos["espera"]).Scan(&nombre, &email, &servicio, &servicioID)
	if errors.Is(err, pgx.ErrNoRows) {
		return correo.Mensaje{Para: f.Para}, "ya no está en la lista de espera", nil
	}
	if err != nil {
		return correo.Mensaje{}, "", err
	}
	c := Carta{Encabezado: n.Nombre, Asunto: "Se liberó un turno — " + n.Nombre, Titulo: "Se liberó el horario que esperabas",
		Saludo: "Hola " + primerNombre(nombre) + ",", Parrafos: []string{"El primero que reserve se lo lleva, así que no lo dejes para después."},
		Boton: "Reservar este turno"}
	filas := []Fila{{"Servicio", servicio}}
	if p := f.Datos["profesional"]; p != "" {
		filas = append(filas, Fila{"Profesional", p})
	}
	link := LinkReserva(e.Sitio, n.Slug) + "?servicio=" + servicioID
	if v := f.Datos["inicio"]; v != "" {
		if t, err := time.Parse(time.RFC3339Nano, v); err == nil {
			t = t.In(n.zona)
			filas = append(filas, Fila{"Fecha", Fecha(t)}, Fila{"Hora", Hora(t)})
			link += "&fecha=" + t.Format(time.DateOnly)
		}
	} else {
		c.Titulo = "Hay lugar para tu turno"
	}
	c.Filas, c.Link = filas, link
	if nota := f.Datos["nota"]; nota != "" {
		c.Parrafos = append(c.Parrafos, nota)
	}
	return c.Mensaje(email), "", nil
}

var nombresRol = map[string]string{"encargado": "encargado/a", "recepcion": "recepción", "profesional": "profesional"}

func (e *Enviador) armarInvitacion(ctx context.Context, tx pgx.Tx, f fila, n negocio, ahora time.Time) (correo.Mensaje, string, error) {
	token := f.Datos["token"]
	var email, rol, quien string
	var vence time.Time
	var aceptada bool
	err := tx.QueryRow(ctx, `SELECT i.email, i.rol, i.vence_en, i.aceptada_en IS NOT NULL, coalesce(m.nombre, '')
		FROM invitaciones i LEFT JOIN miembros m ON m.clerk_id = i.invito AND i.invito <> '' WHERE i.token = $1`, token).
		Scan(&email, &rol, &vence, &aceptada, &quien)
	if errors.Is(err, pgx.ErrNoRows) {
		return correo.Mensaje{Para: f.Para}, "la invitación ya no existe", nil
	}
	if err != nil {
		return correo.Mensaje{}, "", err
	}
	if aceptada {
		return correo.Mensaje{Para: email}, "la invitación ya se aceptó", nil
	}
	if !vence.After(ahora) {
		return correo.Mensaje{Para: email}, "la invitación venció", nil
	}
	if quien == "" {
		quien = "Alguien de " + n.Nombre
	}
	link := LinkInvitacion(e.Sitio, token)
	c := Carta{Encabezado: "TengoLibre", Asunto: "Te invitaron a " + n.Nombre + " — TengoLibre",
		Titulo: "Te sumaron al equipo de " + n.Nombre,
		Parrafos: []string{quien + " te invitó a manejar la agenda con ellos, con el rol de " + nombresRol[rol] + ". " +
			"Aceptá la invitación: si todavía no tenés cuenta, la creás en el mismo paso."},
		Boton: "Aceptar la invitación", Link: link,
		Notas: []string{"El enlace vence en 7 días. Si no esperabas esta invitación, ignorá este mensaje.",
			"O pegá este enlace en el navegador: " + link}}
	return c.Mensaje(email), "", nil
}

func (e *Enviador) armarResumen(ctx context.Context, tx pgx.Tx, f fila, n negocio) (correo.Mensaje, string, error) {
	sin := correo.Mensaje{Para: f.Para}
	dia, err := time.ParseInLocation(time.DateOnly, f.Datos["fecha"], n.zona)
	if err != nil {
		return sin, "fecha inválida", nil
	}
	filas, err := tx.Query(ctx, `SELECT t.inicio, c.nombre, t.servicio, t.profesional FROM turnos t JOIN clientes c ON c.id = t.cliente_id
		WHERE t.inicio >= $1 AND t.inicio < $2 AND t.estado IN ('pendiente', 'confirmado') ORDER BY t.inicio`, dia, dia.AddDate(0, 0, 1))
	if err != nil {
		return sin, "", err
	}
	var lista []string
	for filas.Next() {
		var inicio time.Time
		var cliente, servicio, profesional string
		if err := filas.Scan(&inicio, &cliente, &servicio, &profesional); err != nil {
			filas.Close()
			return sin, "", err
		}
		lista = append(lista, Hora(inicio.In(n.zona))+": "+cliente+" · "+servicio+" con "+profesional)
	}
	filas.Close()
	if err := filas.Err(); err != nil {
		return sin, "", err
	}
	// Un día sin turnos no manda nada: "mañana no tenés turnos" todas las tardes enseña a ignorar los mails.
	if len(lista) == 0 {
		return sin, "no hay turnos ese día", nil
	}
	cuantos := strconv.Itoa(len(lista)) + " turnos"
	if len(lista) == 1 {
		cuantos = "1 turno"
	}
	c := Carta{Encabezado: n.Nombre, Asunto: "Mañana tenés " + cuantos + " — " + n.Nombre, Titulo: "Mañana tenés " + cuantos,
		Parrafos: []string{Fecha(dia)}, Lista: lista,
		Boton: "Ver mi agenda", Link: LinkAgenda(e.Sitio, f.Datos["fecha"]),
		Notas: []string{"Abrí el panel para ver con quién va cada uno, los teléfonos y las notas.",
			"Te llega una vez por día, a la tarde. Lo apagás en Ajustes."}}
	return c.Mensaje(f.Para), "", nil
}
