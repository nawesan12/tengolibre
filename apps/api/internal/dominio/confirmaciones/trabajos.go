package confirmaciones

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/avisos"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/db"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/trabajos"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/whatsapp"
)

// Enviador es lo que hace falta de la Cloud API (*whatsapp.Cliente lo cumple; en los tests, el
// cliente apuntado a un servidor falso).
type Enviador interface {
	EnviarPlantilla(context.Context, whatsapp.Plantilla) (string, error)
	EnviarTexto(ctx context.Context, para, texto string) (string, error)
}

// Trabajos del worker: encolar confirmaciones y recordatorios, mandar la cola, marcar los que no
// contestaron y purgar. Sin WhatsApp (nil), no hay trabajos.
type Trabajos struct {
	Pool     *db.Pool
	WhatsApp Enviador
	// Sitio público (los links de las respuestas).
	Sitio string
	Log   *slog.Logger
}

// Nuevos arma los trabajos. Un cliente nil los deja apagados, y el log lo dice al arrancar.
func Nuevos(p *db.Pool, c *whatsapp.Cliente, sitio string, log *slog.Logger) *Trabajos {
	if log == nil {
		log = slog.Default()
	}
	t := &Trabajos{Pool: p, Sitio: sitio, Log: log}
	if c == nil {
		log.Info("whatsapp: sin configurar (TENGOLIBRE_WHATSAPP_*), no se mandan confirmaciones")
	} else {
		t.WhatsApp = c
	}
	return t
}

// Lista de trabajos (con los nombres que quedan en corridas).
func (t *Trabajos) Lista() []trabajos.Trabajo {
	if t == nil || t.WhatsApp == nil {
		return nil
	}
	return []trabajos.Trabajo{
		{Nombre: "whatsapp_envios", Cada: 5 * time.Second, Correr: func(ctx context.Context, ahora time.Time) (any, error) {
			return t.Enviar(ctx, ahora)
		}},
		{Nombre: "whatsapp_confirmaciones", Cada: 5 * time.Minute, Correr: t.Confirmaciones},
		{Nombre: "whatsapp_recordatorios", Cada: 5 * time.Minute, Correr: t.Recordatorios},
		{Nombre: "whatsapp_sin_respuesta", Cada: 5 * time.Minute, Correr: t.SinRespuesta},
		{Nombre: "whatsapp_purga", Cada: 24 * time.Hour, Correr: t.Purgar},
	}
}

// Hasta qué hora (la del negocio) se manda la confirmación de mañana: a la noche no se escribe. Un
// turno sacado más tarde para mañana se queda sin ella.
const confirmacionHastaHora = 21

// candidato a un mensaje: un turno con el teléfono del cliente.
type candidato struct {
	ID       string
	Inicio   time.Time
	Telefono string
}

// encolar deja el mensaje del turno en la cola con su clave (turno + horario: un turno movido vuelve
// a tener el suyo). Un teléfono que no se puede llevar a E.164 queda descartado, a la vista del panel.
func encolar(ctx context.Context, tx pgx.Tx, tipo string, c candidato, ahora time.Time) (bool, error) {
	clave := tipo + ":" + c.ID + ":" + strconv.FormatInt(c.Inicio.Unix(), 10)
	datos, _ := json.Marshal(map[string]string{"inicio": c.Inicio.UTC().Format(time.RFC3339)})
	para, err := whatsapp.E164(c.Telefono)
	estado, motivo := "pendiente", ""
	if err != nil {
		para, estado, motivo = "", "descartado", "El teléfono «"+c.Telefono+"» no es un celular válido."
	}
	tag, err := tx.Exec(ctx, `INSERT INTO whatsapp_mensajes (negocio_id, turno_id, tipo, para, token, clave, datos, estado,
		ultimo_error, proximo_intento) VALUES (app.negocio_actual(), $1, $2, $3, $4, $5, $6, $7, nullif($8, ''), $9)
		ON CONFLICT (negocio_id, clave) DO NOTHING`, c.ID, tipo, para, avisos.Token(), clave, datos, estado, motivo, ahora)
	return tag.RowsAffected() == 1 && estado == "pendiente", err
}

func candidatos(ctx context.Context, tx pgx.Tx, tipo string, desde, hasta, ahora time.Time, extra string) ([]candidato, error) {
	filas, err := tx.Query(ctx, `SELECT t.id::text, t.inicio, c.telefono FROM turnos t JOIN clientes c ON c.id = t.cliente_id
		LEFT JOIN preferencias p ON p.cliente_id = c.id
		WHERE t.estado IN ('pendiente', 'confirmado') AND t.inicio >= $1 AND t.inicio < $2 AND t.inicio > $3
			AND c.telefono <> '' AND coalesce(p.recordatorios, true) `+extra+`
			AND NOT EXISTS (SELECT 1 FROM whatsapp_mensajes m WHERE m.negocio_id = t.negocio_id
				AND m.clave = $4 || ':' || t.id || ':' || extract(epoch FROM t.inicio)::bigint)
		ORDER BY t.inicio LIMIT 500`, desde, hasta, ahora, tipo)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(filas, pgx.RowToStructByPos[candidato])
}

type ajustesNegocio struct {
	Ajustes
	zona *time.Location
}

func leerAjustesNegocio(ctx context.Context, tx pgx.Tx) (ajustesNegocio, error) {
	a, err := leerAjustes(ctx, tx)
	if err != nil {
		return ajustesNegocio{}, err
	}
	var zona string
	if err := tx.QueryRow(ctx, "SELECT zona_horaria FROM negocios WHERE id = app.negocio_actual()").Scan(&zona); err != nil {
		return ajustesNegocio{}, err
	}
	return ajustesNegocio{a, zonaDe(zona)}, nil
}

// Confirmaciones encola, en los negocios que la prendieron, la confirmación de los turnos de mañana
// desde la hora elegida (y hasta las 21). No se le pregunta a quien ya confirmó solo (por WhatsApp,
// desde su link o pagando la seña).
func (t *Trabajos) Confirmaciones(ctx context.Context, ahora time.Time) (any, error) {
	encolados := 0
	err := trabajos.PorNegocio(ctx, t.Pool, func(_ string, tx pgx.Tx) error {
		a, err := leerAjustesNegocio(ctx, tx)
		if err != nil || !a.Confirmacion {
			return err
		}
		local := ahora.In(a.zona)
		if local.Format("15:04") < a.Hora || local.Hour() >= confirmacionHastaHora {
			return nil
		}
		manana := time.Date(local.Year(), local.Month(), local.Day()+1, 0, 0, 0, 0, a.zona)
		cs, err := candidatos(ctx, tx, "confirmacion", manana, manana.AddDate(0, 0, 1), ahora,
			"AND t.confirmado_por IS DISTINCT FROM 'whatsapp' AND t.confirmado_por IS DISTINCT FROM 'web'")
		if err != nil {
			return err
		}
		for _, c := range cs {
			ok, err := encolar(ctx, tx, "confirmacion", c, ahora)
			if err != nil {
				return err
			}
			if ok {
				encolados++
			}
		}
		return nil
	})
	return map[string]int{"encolados": encolados}, err
}

// Ventana del recordatorio: la misma que la del mail (avisos.Trabajos.Recordatorios).
const (
	recordatorioDesde = 23 * time.Hour
	recordatorioHasta = 25 * time.Hour
)

// Recordatorios encola el recordatorio por WhatsApp de los turnos que empiezan en unas 24 horas, en
// los negocios que lo prendieron y no tienen la confirmación (que ya hace de recordatorio).
func (t *Trabajos) Recordatorios(ctx context.Context, ahora time.Time) (any, error) {
	encolados := 0
	err := trabajos.PorNegocio(ctx, t.Pool, func(_ string, tx pgx.Tx) error {
		a, err := leerAjustesNegocio(ctx, tx)
		if err != nil || !a.Recordatorio || a.Confirmacion {
			return err
		}
		cs, err := candidatos(ctx, tx, "recordatorio", ahora.Add(recordatorioDesde), ahora.Add(recordatorioHasta), ahora, "")
		if err != nil {
			return err
		}
		for _, c := range cs {
			ok, err := encolar(ctx, tx, "recordatorio", c, ahora)
			if err != nil {
				return err
			}
			if ok {
				encolados++
			}
		}
		return nil
	})
	return map[string]int{"encolados": encolados}, err
}

// SinRespuesta marca para llamar los turnos cuya confirmación salió hace más de las horas del
// negocio sin que el cliente tocara nada (ni confirmara desde su link), y avisa a la campanita.
func (t *Trabajos) SinRespuesta(ctx context.Context, ahora time.Time) (any, error) {
	marcados := 0
	err := trabajos.PorNegocio(ctx, t.Pool, func(_ string, tx pgx.Tx) error {
		a, err := leerAjustesNegocio(ctx, tx)
		if err != nil || !a.Confirmacion {
			return err
		}
		filas, err := tx.Query(ctx, `UPDATE whatsapp_mensajes m SET sin_respuesta_en = $1 FROM turnos t
			WHERE t.id = m.turno_id AND m.id IN (SELECT x.id FROM whatsapp_mensajes x WHERE x.tipo = 'confirmacion' AND x.estado = 'enviado'
				AND x.respuesta IS NULL AND x.sin_respuesta_en IS NULL AND x.enviado_en <= $1::timestamptz - make_interval(hours => $2::int)
				ORDER BY x.enviado_en LIMIT 200 FOR UPDATE SKIP LOCKED)
			RETURNING t.id::text, t.estado, t.inicio, t.confirmado_por`, ahora, a.SinRespuestaHoras)
		if err != nil {
			return err
		}
		type fila struct {
			ID, Estado    string
			Inicio        time.Time
			ConfirmadoPor *string
		}
		fs, err := pgx.CollectRows(filas, pgx.RowToStructByPos[fila])
		if err != nil {
			return err
		}
		for _, f := range fs {
			if (f.Estado != "pendiente" && f.Estado != "confirmado") || !f.Inicio.After(ahora) ||
				(f.ConfirmadoPor != nil && (*f.ConfirmadoPor == "whatsapp" || *f.ConfirmadoPor == "web")) {
				continue
			}
			var id string
			d, err := leerTurno(ctx, tx, "t.id = $1", f.ID, &id)
			if err != nil {
				return err
			}
			tag, err := tx.Exec(ctx, `UPDATE turnos SET llamar_desde = $2, llamar_motivo = 'sin_respuesta', actualizado_en = now()
				WHERE id = $1 AND llamar_desde IS NULL`, id, ahora)
			if err != nil {
				return err
			}
			if tag.RowsAffected() == 0 {
				continue
			}
			texto := "Llamalo para confirmar: " + conQuien(d) + ", el " + cuando(d) + "."
			if d.Telefono != "" {
				texto = "Llamalo al " + d.Telefono + " para confirmar: " + conQuien(d) + ", el " + cuando(d) + "."
			}
			if err := avisos.Notificar(ctx, tx, avisos.Notificacion{Tipo: NotiSinRespuesta, TurnoID: id,
				Titulo: d.Cliente + " no contestó el WhatsApp", Texto: texto}); err != nil {
				return err
			}
			marcados++
		}
		return nil
	})
	return map[string]int{"marcados": marcados}, err
}

// Cuánto se guarda la cola (como los mails: 60 días).
const retencion = 60 * 24 * time.Hour

// Purgar borra, por tandas, los mensajes viejos ya resueltos.
func (t *Trabajos) Purgar(ctx context.Context, ahora time.Time) (any, error) {
	var n int64
	err := t.Pool.EnTx(ctx, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, "SELECT app.purgar_whatsapp($1, 5000)", ahora.Add(-retencion)).Scan(&n)
	})
	return map[string]int64{"borrados": n}, err
}

// ResultadoEnvio de una pasada de la cola.
type ResultadoEnvio struct {
	Enviados    int `json:"enviados"`
	Reintentos  int `json:"reintentos"`
	Fallidos    int `json:"fallidos"`
	Descartados int `json:"descartados"`
	Errores     int `json:"errores"`
}

// Enviar manda lo que está pendiente a esa hora (como avisos.Enviador con los mails): cada mensaje
// en su transacción con el negocio fijado, bloqueado para que dos workers no lo manden dos veces.
func (t *Trabajos) Enviar(ctx context.Context, ahora time.Time) (ResultadoEnvio, error) {
	var res ResultadoEnvio
	type pendiente struct{ ID, Negocio string }
	var ps []pendiente
	err := t.Pool.EnTx(ctx, func(tx pgx.Tx) error {
		filas, err := tx.Query(ctx, "SELECT mensaje_id::text, negocio_id::text FROM app.whatsapp_pendientes($1, 100)", ahora)
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
		if err := t.Pool.EnNegocio(ctx, p.Negocio, func(tx pgx.Tx) error { return t.uno(ctx, tx, p.ID, ahora, &res) }); err != nil {
			res.Errores++
			t.Log.Error("whatsapp: no se pudo procesar", "mensaje", p.ID, "error", err)
		}
	}
	return res, nil
}

func (t *Trabajos) uno(ctx context.Context, tx pgx.Tx, id string, ahora time.Time, res *ResultadoEnvio) error {
	var tipo, para, token string
	var turnoID *string
	var datos map[string]string
	var intentos int
	err := tx.QueryRow(ctx, `SELECT tipo, para, token, turno_id::text, datos, intentos FROM whatsapp_mensajes
		WHERE id = $1 AND estado = 'pendiente' AND proximo_intento <= $2 FOR UPDATE SKIP LOCKED`, id, ahora).
		Scan(&tipo, &para, &token, &turnoID, &datos, &intentos)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	var wamid, descarte string
	var errEnvio error
	switch tipo {
	case "respuesta":
		wamid, errEnvio = t.WhatsApp.EnviarTexto(ctx, para, datos["texto"])
	default:
		var d datosTurno
		var tid string
		if turnoID == nil {
			descarte = "El turno ya no existe."
			break
		}
		d, err = leerTurno(ctx, tx, "t.id = $1", *turnoID, &tid)
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			descarte = "El turno ya no existe."
		case err != nil:
			return err
		case d.Estado != "pendiente" && d.Estado != "confirmado":
			descarte = "El turno está " + d.Estado + "."
		case !d.Inicio.After(ahora):
			descarte = "El turno ya pasó."
		case d.Inicio.UTC().Format(time.RFC3339) != datos["inicio"]:
			descarte = "El turno cambió de horario."
		case !d.Recordatorios:
			descarte = "El cliente no quiere recordatorios."
		case tipo == "confirmacion" && d.ConfirmadoPor != nil && (*d.ConfirmadoPor == "whatsapp" || *d.ConfirmadoPor == "web"):
			descarte = "El cliente ya confirmó."
		case tipo == "confirmacion":
			wamid, errEnvio = t.WhatsApp.EnviarPlantilla(ctx, PlantillaDeConfirmacion(para, token, d))
		default:
			wamid, errEnvio = t.WhatsApp.EnviarPlantilla(ctx, PlantillaDeRecordatorio(para, d))
		}
	}
	if descarte != "" {
		res.Descartados++
		_, err := tx.Exec(ctx, "UPDATE whatsapp_mensajes SET estado = 'descartado', ultimo_error = $2 WHERE id = $1", id, descarte)
		return err
	}
	if errEnvio == nil {
		res.Enviados++
		_, err := tx.Exec(ctx, `UPDATE whatsapp_mensajes SET estado = 'enviado', wamid = nullif($2, ''), enviado_en = $3,
			intentos = intentos + 1, ultimo_error = NULL WHERE id = $1`, id, wamid, ahora)
		return err
	}
	intentos++
	var e *whatsapp.Error
	reintentar := errors.As(errEnvio, &e) && e.Reintentable() || !errors.As(errEnvio, &e) && !errors.Is(errEnvio, whatsapp.ErrDestinoInvalido)
	t.Log.Warn("whatsapp: no salió", "mensaje", id, "tipo", tipo, "intento", intentos, "error", errEnvio)
	if !reintentar || intentos >= avisos.MaxIntentos {
		res.Fallidos++
		if _, err := tx.Exec(ctx, "UPDATE whatsapp_mensajes SET intentos = $2 WHERE id = $1", id, intentos); err != nil {
			return err
		}
		return fallido(ctx, tx, id, errEnvio, ahora)
	}
	res.Reintentos++
	msj := errEnvio.Error()
	if len(msj) > 1000 {
		msj = msj[:1000]
	}
	_, err = tx.Exec(ctx, "UPDATE whatsapp_mensajes SET intentos = $2, ultimo_error = $3, proximo_intento = $4 WHERE id = $1",
		id, intentos, msj, ahora.Add(avisos.EsperaReintento(intentos)))
	return err
}
