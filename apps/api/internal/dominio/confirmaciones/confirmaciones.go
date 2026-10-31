// Package confirmaciones: la confirmación y el recordatorio de los turnos por WhatsApp (Propuesta
// 02), con la WhatsApp Cloud API de Meta y el número de la plataforma.
//
// El día antes, a la hora que eligió el negocio, el worker manda la plantilla «Hola Sofía 👋 Te
// esperamos mañana… ¿Venís?» con dos botones. El webhook de Meta trae lo que tocó: «Confirmo»
// confirma el turno (en vivo, por el trigger de tiempo real de turnos, y con la campanita) y le
// responde con el link al turno; «Necesito cambiarlo» lo marca para que recepción llame. Si no
// contesta en las horas que eligió el negocio, también queda para llamar. El turno guarda de dónde
// vino la confirmación (whatsapp, web, panel) y cuándo, para el talón.
//
// Todo mensaje pasa por la cola whatsapp_mensajes (como los mails por avisos) y lo manda el worker.
package confirmaciones

import (
	"context"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/avisos"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/auditoria"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/db"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/problema"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/sesion"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/web"
)

// Tipos de la campanita que suma este paquete (migración 00014).
const (
	NotiConfirmacion = "confirmacion"
	NotiCambioPedido = "cambio_pedido"
	NotiSinRespuesta = "sin_respuesta"
)

// Servicio de las rutas (API). Ahora es el reloj (tests); nil: time.Now.
type Servicio struct {
	Pool   *db.Pool
	Sesion *sesion.Servicio
	Ahora  func() time.Time
	// Sitio público, para los links que van en las respuestas.
	Sitio string
	// Disponible: la plataforma tiene número de WhatsApp configurado (token e id del número).
	Disponible bool
	// TelefonoID: el id del número de la plataforma; un webhook de otro número se ignora.
	TelefonoID string
	// SecretoApp valida la firma del webhook; Verificacion es el token de la suscripción (GET).
	SecretoApp, Verificacion string
}

func (s *Servicio) ahora() time.Time {
	if s.Ahora != nil {
		return s.Ahora()
	}
	return time.Now()
}

// Rutas del dominio.
func (s *Servicio) Rutas(mux *http.ServeMux) {
	ses := s.Sesion
	mux.HandleFunc("GET /v1/panel/ajustes/whatsapp", ses.Panel(sesion.Agenda, web.Envolver(s.leerAjustes)))
	mux.HandleFunc("PUT /v1/panel/ajustes/whatsapp", ses.Panel(sesion.Gestion, web.Envolver(s.guardarAjustes)))
	mux.HandleFunc("GET /v1/panel/whatsapp/mensajes", ses.Panel(avisos.Mostrador, web.Envolver(s.mensajes)))
	mux.HandleFunc("GET /v1/panel/para-llamar", ses.Panel(sesion.Agenda, web.Envolver(s.paraLlamar)))
	mux.HandleFunc("POST /v1/panel/turnos/{id}/llamado", ses.Panel(sesion.Agenda, web.Envolver(s.llamado)))
	// El cliente confirma desde su link (/turno/{codigo}).
	mux.HandleFunc("POST /v1/t/{codigo}/confirmar", web.Envolver(s.confirmarWeb))
	// Webhook de Meta: GET para verificar la suscripción, POST firmado con los botones y estados.
	mux.HandleFunc("GET /v1/webhooks/whatsapp", s.verificarWebhook)
	mux.HandleFunc("POST /v1/webhooks/whatsapp", web.Envolver(s.webhook))
}

// Ajustes de WhatsApp del negocio (Configuración › Avisos).
type Ajustes struct {
	// Disponible: la plataforma tiene WhatsApp (sólo lectura). Sin eso no se puede prender.
	Disponible bool `json:"disponible"`
	// Confirmacion: mandar el día antes el mensaje con «Confirmo» / «Necesito cambiarlo».
	Confirmacion bool `json:"confirmacion"`
	// Hora (del negocio, «HH:MM», entre 07:00 y 20:00) del día anterior en que sale.
	Hora string `json:"hora"`
	// SinRespuestaHoras: si no contesta en tantas horas (1 a 12), se lo marca a recepción para llamar.
	SinRespuestaHoras int `json:"sinRespuestaHoras"`
	// Recordatorio: el de 24 horas antes también por WhatsApp. Con la confirmación prendida no sale:
	// el mensaje de confirmación ya es el recordatorio.
	Recordatorio bool `json:"recordatorio"`
}

func leerAjustes(ctx context.Context, tx pgx.Tx) (Ajustes, error) {
	var a Ajustes
	err := tx.QueryRow(ctx, `SELECT whatsapp_confirmacion, to_char(whatsapp_hora, 'HH24:MI'), whatsapp_sin_respuesta_horas,
		whatsapp_recordatorio FROM ajustes WHERE negocio_id = app.negocio_actual()`).
		Scan(&a.Confirmacion, &a.Hora, &a.SinRespuestaHoras, &a.Recordatorio)
	return a, err
}

func (s *Servicio) leerAjustes(w http.ResponseWriter, r *http.Request) error {
	var a Ajustes
	err := s.Pool.EnNegocio(r.Context(), sesion.De(r.Context()).NegocioID, func(tx pgx.Tx) (err error) {
		a, err = leerAjustes(r.Context(), tx)
		return err
	})
	if err != nil {
		return err
	}
	a.Disponible = s.Disponible
	web.JSON(w, http.StatusOK, a)
	return nil
}

var formatoHora = regexp.MustCompile(`^([01][0-9]|2[0-3]):[0-5][0-9]$`)

func (s *Servicio) guardarAjustes(w http.ResponseWriter, r *http.Request) error {
	var in struct {
		Confirmacion      bool   `json:"confirmacion"`
		Hora              string `json:"hora"`
		SinRespuestaHoras int    `json:"sinRespuestaHoras"`
		Recordatorio      bool   `json:"recordatorio"`
		// Disponible se ignora (viene si el front reenvía lo que leyó).
		Disponible *bool `json:"disponible"`
	}
	if err := web.Leer(w, r, &in); err != nil {
		return err
	}
	if in.Hora == "" {
		in.Hora = "10:00"
	}
	if !formatoHora.MatchString(in.Hora) || in.Hora < "07:00" || in.Hora > "20:00" {
		return problema.Invalido("La confirmación sale entre las 07:00 y las 20:00.", problema.Campo{Ruta: "hora", Mensaje: "HH:MM entre 07:00 y 20:00"})
	}
	if in.SinRespuestaHoras < 1 || in.SinRespuestaHoras > 12 {
		return problema.Invalido("Las horas para contestar van de 1 a 12.", problema.Campo{Ruta: "sinRespuestaHoras", Mensaje: "fuera de rango"})
	}
	if (in.Confirmacion || in.Recordatorio) && !s.Disponible {
		return problema.Conflicto("whatsapp_apagado", "Los mensajes por WhatsApp todavía no están disponibles.")
	}
	a := Ajustes{Disponible: s.Disponible, Confirmacion: in.Confirmacion, Hora: in.Hora, SinRespuestaHoras: in.SinRespuestaHoras,
		Recordatorio: in.Recordatorio}
	ctx := r.Context()
	err := s.Pool.EnNegocio(ctx, sesion.De(ctx).NegocioID, func(tx pgx.Tx) error {
		antes, err := leerAjustes(ctx, tx)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `UPDATE ajustes SET whatsapp_confirmacion = $1, whatsapp_hora = $2::time, whatsapp_sin_respuesta_horas = $3,
			whatsapp_recordatorio = $4, actualizado_en = now() WHERE negocio_id = app.negocio_actual()`,
			a.Confirmacion, a.Hora, a.SinRespuestaHoras, a.Recordatorio); err != nil {
			return err
		}
		antes.Disponible = s.Disponible
		return auditoria.Anotar(ctx, tx, auditoria.Registro{Accion: "ajustes.whatsapp", Entidad: "ajustes",
			Detalle: map[string]any{"antes": antes, "despues": a}})
	})
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusOK, a)
	return nil
}

// Mensaje de la cola como lo ve el panel (¿le llegó el WhatsApp?).
type Mensaje struct {
	ID             string     `json:"id"`
	Tipo           string     `json:"tipo"`
	Para           string     `json:"para"`
	TurnoID        *string    `json:"turnoId"`
	Cliente        *string    `json:"cliente"`
	Estado         string     `json:"estado"`
	Intentos       int        `json:"intentos"`
	UltimoError    *string    `json:"ultimoError"`
	EnviadoEn      *time.Time `json:"enviadoEn"`
	EntregadoEn    *time.Time `json:"entregadoEn"`
	LeidoEn        *time.Time `json:"leidoEn"`
	Respuesta      *string    `json:"respuesta"`
	RespondidoEn   *time.Time `json:"respondidoEn"`
	SinRespuestaEn *time.Time `json:"sinRespuestaEn"`
	CreadoEn       time.Time  `json:"creadoEn"`
}

// mensajes: los últimos 50 (filtros: turno, tipo, antes = el creadoEn del último de la página).
func (s *Servicio) mensajes(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	q := r.URL.Query()
	cond := []string{"true"}
	var args []any
	agregar := func(sql string, v any) {
		args = append(args, v)
		cond = append(cond, strings.ReplaceAll(sql, "?", "$"+strconv.Itoa(len(args))))
	}
	if v := q.Get("turno"); v != "" {
		agregar("m.turno_id::text = ?", v)
	}
	if v := q.Get("tipo"); v != "" {
		agregar("m.tipo = ANY(?)", strings.Split(v, ","))
	}
	if v := q.Get("antes"); v != "" {
		t, err := time.Parse(time.RFC3339Nano, v)
		if err != nil {
			return problema.Invalido("antes va como fecha RFC 3339.")
		}
		agregar("m.creado_en < ?", t)
	}
	out := []Mensaje{}
	err := s.Pool.EnNegocio(ctx, sesion.De(ctx).NegocioID, func(tx pgx.Tx) error {
		filas, err := tx.Query(ctx, `SELECT m.id::text, m.tipo, m.para, m.turno_id::text, c.nombre, m.estado, m.intentos, m.ultimo_error,
			m.enviado_en, m.entregado_en, m.leido_en, m.respuesta, m.respondido_en, m.sin_respuesta_en, m.creado_en
			FROM whatsapp_mensajes m LEFT JOIN turnos t ON t.id = m.turno_id LEFT JOIN clientes c ON c.id = t.cliente_id
			WHERE `+strings.Join(cond, " AND ")+` ORDER BY m.creado_en DESC, m.id LIMIT 50`, args...)
		if err != nil {
			return err
		}
		out, err = pgx.CollectRows(filas, pgx.RowToStructByPos[Mensaje])
		return err
	})
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusOK, map[string]any{"mensajes": out})
	return nil
}

// ParaLlamar: un turno que recepción tiene que llamar.
type ParaLlamar struct {
	TurnoID     string    `json:"turnoId"`
	Cliente     string    `json:"cliente"`
	Telefono    string    `json:"telefono"`
	Servicio    string    `json:"servicio"`
	Profesional string    `json:"profesional"`
	Inicio      time.Time `json:"inicio"`
	Estado      string    `json:"estado"`
	// Motivo: cambio (pidió cambiarlo), sin_respuesta (no contestó), no_llego (el WhatsApp no salió).
	Motivo string    `json:"motivo"`
	Desde  time.Time `json:"desde"`
}

func (s *Servicio) paraLlamar(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	out := []ParaLlamar{}
	err := s.Pool.EnNegocio(ctx, sesion.De(ctx).NegocioID, func(tx pgx.Tx) error {
		filas, err := tx.Query(ctx, `SELECT t.id::text, c.nombre, c.telefono, t.servicio, t.profesional, t.inicio, t.estado,
			t.llamar_motivo, t.llamar_desde FROM turnos t JOIN clientes c ON c.id = t.cliente_id
			WHERE t.llamar_desde IS NOT NULL ORDER BY t.inicio LIMIT 100`)
		if err != nil {
			return err
		}
		out, err = pgx.CollectRows(filas, pgx.RowToStructByPos[ParaLlamar])
		return err
	})
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusOK, map[string]any{"turnos": out})
	return nil
}

// llamado: recepción ya lo llamó (el turno sale de la lista).
func (s *Servicio) llamado(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	err := s.Pool.EnNegocio(ctx, sesion.De(ctx).NegocioID, func(tx pgx.Tx) error {
		var motivo *string
		if err := tx.QueryRow(ctx, "SELECT llamar_motivo FROM turnos WHERE id = $1 FOR UPDATE", r.PathValue("id")).Scan(&motivo); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `UPDATE turnos SET llamar_desde = NULL, llamar_motivo = NULL, actualizado_en = now()
			WHERE id = $1`, r.PathValue("id")); err != nil {
			return err
		}
		return auditoria.Anotar(ctx, tx, auditoria.Registro{Accion: "turno.llamado", Entidad: "turno", EntidadID: r.PathValue("id"),
			Detalle: map[string]any{"motivo": motivo}})
	})
	if err != nil {
		return err
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}

// Confirmacion: lo que responde POST /v1/t/{codigo}/confirmar.
type Confirmacion struct {
	Estado        string     `json:"estado"`
	ConfirmadoPor *string    `json:"confirmadoPor"`
	ConfirmadoEn  *time.Time `json:"confirmadoEn"`
}

// confirmarWeb: el cliente confirma desde su link. Si ya había confirmado él (por WhatsApp o acá), no
// cambia nada.
func (s *Servicio) confirmarWeb(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	var negocio *string
	err := s.Pool.EnTx(ctx, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, "SELECT app.negocio_del_turno($1)::text", r.PathValue("codigo")).Scan(&negocio)
	})
	if err != nil {
		return err
	}
	if negocio == nil {
		return problema.NoEncontrado("No encontramos ese turno.")
	}
	ahora := s.ahora()
	var out Confirmacion
	err = s.Pool.EnNegocio(ctx, *negocio, func(tx pgx.Tx) error {
		var id string
		d, err := leerTurno(ctx, tx, "t.codigo = $1", r.PathValue("codigo"), &id)
		if err != nil {
			return err
		}
		if d.ConfirmadoPor != nil && (*d.ConfirmadoPor == "web" || *d.ConfirmadoPor == "whatsapp") && d.Estado == "confirmado" {
			return tx.QueryRow(ctx, "SELECT estado, confirmado_por, confirmado_en FROM turnos WHERE id = $1", id).
				Scan(&out.Estado, &out.ConfirmadoPor, &out.ConfirmadoEn)
		}
		if (d.Estado != "pendiente" && d.Estado != "confirmado") || !d.Inicio.After(ahora) {
			return problema.Conflicto("estado", "Este turno ya no se puede confirmar.")
		}
		if err := tx.QueryRow(ctx, `UPDATE turnos SET estado = 'confirmado', confirmado_por = 'web', confirmado_en = $2,
			cambio_pedido_en = NULL, llamar_desde = NULL, llamar_motivo = NULL, actualizado_en = now() WHERE id = $1
			RETURNING estado, confirmado_por, confirmado_en`, id, ahora).Scan(&out.Estado, &out.ConfirmadoPor, &out.ConfirmadoEn); err != nil {
			return err
		}
		if err := avisos.Notificar(ctx, tx, avisos.Notificacion{Tipo: NotiConfirmacion, TurnoID: id,
			Titulo: d.Cliente + " confirmó su turno", Texto: conQuien(d) + ", el " + cuando(d) + ". Desde el link del turno."}); err != nil {
			return err
		}
		return auditoria.Anotar(ctx, tx, auditoria.Registro{Accion: "turno.confirmado_web", Entidad: "turno", EntidadID: id})
	})
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusOK, out)
	return nil
}

// leerTurno trae (y bloquea) el turno con lo que necesitan los mensajes, en el negocio fijado.
func leerTurno(ctx context.Context, tx pgx.Tx, donde string, arg any, id *string) (datosTurno, error) {
	var d datosTurno
	var zona string
	err := tx.QueryRow(ctx, `SELECT t.id::text, t.codigo, t.estado, c.nombre, c.telefono, t.servicio, t.profesional, t.inicio,
		t.confirmado_por, n.nombre, n.slug, n.zona_horaria, coalesce(p.recordatorios, true)
		FROM turnos t JOIN clientes c ON c.id = t.cliente_id JOIN negocios n ON n.id = t.negocio_id
		LEFT JOIN preferencias p ON p.cliente_id = c.id
		WHERE `+donde+` FOR UPDATE OF t`, arg).Scan(id, &d.Codigo, &d.Estado, &d.Cliente, &d.Telefono, &d.Servicio, &d.Profesional,
		&d.Inicio, &d.ConfirmadoPor, &d.Negocio, &d.Slug, &zona, &d.Recordatorios)
	if err != nil {
		return d, err
	}
	d.Zona = zonaDe(zona)
	return d, nil
}

func zonaDe(nombre string) *time.Location {
	z, err := time.LoadLocation(nombre)
	if err != nil {
		z, _ = time.LoadLocation("America/Argentina/Buenos_Aires")
	}
	return z
}

// encolarRespuesta deja un texto libre para el cliente en la cola (lo manda el worker).
func encolarRespuesta(ctx context.Context, tx pgx.Tx, turnoID, para, texto string, ahora time.Time) error {
	if para == "" {
		return nil
	}
	_, err := tx.Exec(ctx, `INSERT INTO whatsapp_mensajes (negocio_id, turno_id, tipo, para, token, datos, proximo_intento)
		VALUES (app.negocio_actual(), nullif($1, '')::uuid, 'respuesta', $2, $3, jsonb_build_object('texto', $4::text), $5)`,
		turnoID, para, avisos.Token(), texto, ahora)
	return err
}
