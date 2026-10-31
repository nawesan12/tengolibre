package push

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/apns"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/db"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/fcm"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/trabajos"
)

// EnviadorAPNs es lo que hace falta de APNs (*apns.Cliente lo cumple; en los tests, uno falso).
type EnviadorAPNs interface {
	Enviar(context.Context, apns.Envio) error
}

// EnviadorFCM es lo que hace falta de FCM (*fcm.Cliente lo cumple; en los tests, uno falso).
type EnviadorFCM interface {
	Enviar(context.Context, fcm.Envio) error
}

// Frescura: una notificación con más de esto ya no se manda (el worker estuvo caído, o el push se
// prendió después): llega a la campanita, pero no suena tarde en el teléfono.
const Frescura = 30 * time.Minute

// Vida: cuánto guardan APNs y FCM el aviso si el teléfono está apagado.
const Vida = 2 * time.Hour

// Envios manda la campanita a los teléfonos. Sin APNs ni FCM (nil), el trabajo no hace nada.
type Envios struct {
	Pool *db.Pool
	APNs EnviadorAPNs
	FCM  EnviadorFCM
	Log  *slog.Logger
}

// Nuevo arma el envío. Un cliente nil deja apagada esa plataforma, y el log lo dice al arrancar.
func Nuevo(p *db.Pool, a *apns.Cliente, f *fcm.Cliente, log *slog.Logger) *Envios {
	if log == nil {
		log = slog.Default()
	}
	e := &Envios{Pool: p, Log: log}
	if a == nil {
		log.Info("push: APNs sin configurar (TENGOLIBRE_APNS_*), no se mandan avisos al iPhone")
	} else {
		e.APNs = a
	}
	if f == nil {
		log.Info("push: FCM sin configurar (TENGOLIBRE_FCM_CUENTA), no se mandan avisos a Android")
	} else {
		e.FCM = f
	}
	return e
}

func (e *Envios) activo() bool { return e != nil && (e.APNs != nil || e.FCM != nil) }

// Trabajo del worker: cada 10 segundos (el planificador pasa cada 5).
func (e *Envios) Trabajo() trabajos.Trabajo {
	return trabajos.Trabajo{Nombre: "push", Cada: 10 * time.Second, Correr: e.Correr}
}

// Aviso es una notificación ya decidida para un teléfono.
type Aviso struct {
	Titulo, Cuerpo string
	// Ruta de la app para abrir la pantalla (turnos/<id>, notificaciones).
	Ruta string
	// Negocio: el id, para que la app cambie de negocio antes de abrir la ruta si hace falta.
	Negocio string
	// Colapso: un id por notificación (los reintentos se ven como uno).
	Colapso string
}

// PayloadAPNs arma el JSON de APNs: aps.alert con título y cuerpo, sonido, thread-id por negocio y,
// fuera de aps, la ruta y el negocio para el enlace profundo.
func PayloadAPNs(a Aviso) []byte {
	type alerta struct {
		Title string `json:"title"`
		Body  string `json:"body,omitempty"`
	}
	type aps struct {
		Alert    alerta `json:"alert"`
		Sound    string `json:"sound"`
		ThreadID string `json:"thread-id"`
	}
	b, _ := json.Marshal(struct {
		Aps     aps    `json:"aps"`
		Ruta    string `json:"ruta"`
		Negocio string `json:"negocio"`
	}{aps{alerta{a.Titulo, a.Cuerpo}, "default", "negocio-" + a.Negocio}, a.Ruta, a.Negocio})
	return b
}

// destino es un teléfono de un miembro activo del negocio.
type destino struct {
	plataforma, token, entorno string
}

type pendiente struct {
	aviso   Aviso
	destino destino
}

// Correr toma las notificaciones nuevas de cada negocio (las baja de pendientes en la misma
// transacción, así dos workers no las mandan dos veces) y después las manda, fuera de la
// transacción. Un token que APNs o FCM dan por muerto se borra.
func (e *Envios) Correr(ctx context.Context, ahora time.Time) (any, error) {
	if !e.activo() {
		return nil, nil
	}
	var cola []pendiente
	err := trabajos.PorNegocio(ctx, e.Pool, func(negocio string, tx pgx.Tx) error {
		filas, err := tx.Query(ctx, `UPDATE notificaciones SET push = false WHERE id IN (
				SELECT id FROM notificaciones WHERE push ORDER BY creado_en LIMIT 100 FOR UPDATE SKIP LOCKED)
			RETURNING id::text, titulo, texto, coalesce(turno_id::text, ''), creado_en`)
		if err != nil {
			return err
		}
		var avisos []Aviso
		for filas.Next() {
			var id, titulo, texto, turno string
			var creado time.Time
			if err := filas.Scan(&id, &titulo, &texto, &turno, &creado); err != nil {
				filas.Close()
				return err
			}
			if ahora.Sub(creado) > Frescura {
				continue
			}
			ruta := "notificaciones"
			if turno != "" {
				ruta = "turnos/" + turno
			}
			avisos = append(avisos, Aviso{Titulo: titulo, Cuerpo: texto, Ruta: ruta, Negocio: negocio, Colapso: "noti-" + id})
		}
		filas.Close()
		if err := filas.Err(); err != nil || len(avisos) == 0 {
			return err
		}
		filas, err = tx.Query(ctx, `SELECT DISTINCT d.plataforma, d.token, d.entorno
			FROM miembros m
			JOIN auth.dispositivos d ON d.clerk_id = m.clerk_id
			JOIN auth.sesiones s ON s.hash = d.sesion_hash
			WHERE m.activo AND s.vence_en > $1`, ahora)
		if err != nil {
			return err
		}
		ds, err := pgx.CollectRows(filas, func(f pgx.CollectableRow) (destino, error) {
			var d destino
			err := f.Scan(&d.plataforma, &d.token, &d.entorno)
			return d, err
		})
		if err != nil {
			return err
		}
		for _, a := range avisos {
			for _, d := range ds {
				cola = append(cola, pendiente{a, d})
			}
		}
		return nil
	})
	mandados, muertos := e.mandar(ctx, cola, ahora)
	return map[string]int{"mandados": mandados, "borrados": muertos}, err
}

func (e *Envios) mandar(ctx context.Context, cola []pendiente, ahora time.Time) (mandados, muertos int) {
	var invalidos []string
	for _, p := range cola {
		var err error
		switch {
		case p.destino.plataforma == "ios" && e.APNs != nil:
			err = e.APNs.Enviar(ctx, apns.Envio{Token: p.destino.token, Entorno: p.destino.entorno,
				Payload: PayloadAPNs(p.aviso), Colapso: p.aviso.Colapso, Vence: ahora.Add(Vida)})
		case p.destino.plataforma == "android" && e.FCM != nil:
			err = e.FCM.Enviar(ctx, fcm.Envio{Token: p.destino.token, Titulo: p.aviso.Titulo, Cuerpo: p.aviso.Cuerpo,
				Datos: map[string]string{"ruta": p.aviso.Ruta, "negocio": p.aviso.Negocio}, Colapso: p.aviso.Colapso, Vida: Vida})
		default:
			continue
		}
		switch {
		case errors.Is(err, apns.ErrTokenInvalido) || errors.Is(err, fcm.ErrTokenInvalido):
			invalidos = append(invalidos, p.destino.token)
		case err != nil:
			// Sin reintento: un aviso que no salió sigue en la campanita, y uno que llega tarde
			// confunde más de lo que ayuda.
			e.Log.Warn("push: no salió un aviso", "plataforma", p.destino.plataforma, "error", err)
		default:
			mandados++
		}
	}
	if len(invalidos) == 0 {
		return mandados, 0
	}
	err := e.Pool.EnTx(ctx, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, "DELETE FROM auth.dispositivos WHERE token = ANY($1)", invalidos)
		muertos = int(tag.RowsAffected())
		return err
	})
	if err != nil {
		e.Log.Error("push: borrar tokens muertos", "error", err)
	}
	return mandados, muertos
}
