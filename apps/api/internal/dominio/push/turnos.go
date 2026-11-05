package push

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/apns"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/problema"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/sesion"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/trabajos"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/web"
)

// Dos cosas del iPhone que siguen a un turno (no a una persona del equipo):
//   - la Live Activity del turno en curso: la app manda su token de push y el worker le manda cada
//     cambio del turno (atendido, cancelado, movido) y la termina cuando corresponde;
//   - los avisos a la clienta: su app registra el teléfono con el código del turno y el worker le
//     manda al iPhone lo mismo que le llega por mail (confirmación, recordatorio, cancelación…).

// referencia2001: ActivityKit decodifica las fechas del content-state como segundos desde 2001.
var referencia2001 = time.Date(2001, 1, 1, 0, 0, 0, 0, time.UTC)

func segundos2001(t time.Time) float64 { return t.Sub(referencia2001).Seconds() }

func (s *Servicio) rutasTurnos(mux *http.ServeMux) {
	mux.HandleFunc("POST /v1/panel/turnos/{id}/actividad", s.Sesion.Panel(sesion.Agenda, web.Envolver(s.registrarActividad)))
}

// registrarActividad: {"token": hex, "entorno": "sandbox|produccion"} de la Live Activity de un turno.
func (s *Servicio) registrarActividad(w http.ResponseWriter, r *http.Request) error {
	var in struct {
		Token   string `json:"token"`
		Entorno string `json:"entorno"`
	}
	if err := web.Leer(w, r, &in); err != nil {
		return err
	}
	in.Token = strings.ToLower(strings.TrimSpace(in.Token))
	if !tokenActividad.MatchString(in.Token) {
		return problema.Invalido("El token de la actividad no es válido.", problema.Campo{Ruta: "token", Mensaje: "hexadecimal"})
	}
	if in.Entorno != "sandbox" {
		in.Entorno = "produccion"
	}
	ctx := r.Context()
	ses := sesion.De(ctx)
	id := r.PathValue("id")
	if !esUUID(id) {
		return problema.NoEncontrado("No encontramos ese turno.")
	}
	err := s.Pool.EnNegocio(ctx, ses.NegocioID, func(tx pgx.Tx) error {
		var existe bool
		if err := tx.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM turnos WHERE id = $1)", id).Scan(&existe); err != nil {
			return err
		}
		if !existe {
			return problema.NoEncontrado("No encontramos ese turno.")
		}
		if _, err := tx.Exec(ctx, "DELETE FROM turno_actividades WHERE token = $1", in.Token); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `INSERT INTO turno_actividades (negocio_id, turno_id, token, entorno) VALUES (app.negocio_actual(), $1, $2, $3)`,
			id, in.Token, in.Entorno)
		return err
	})
	if err != nil {
		return err
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}

// TrabajoTurnos: la Live Activity y los avisos de la clienta, cada 10 segundos. Sin APNs, nada.
func (e *Envios) TrabajoTurnos() trabajos.Trabajo {
	return trabajos.Trabajo{Nombre: "push-turnos", Cada: 10 * time.Second, Correr: e.CorrerTurnos}
}

type envioTurno struct {
	envio  apns.Envio
	borrar string // turno_actividades.id a borrar después de mandar el «end»
	token  string
	tabla  string // turno_actividades | turno_dispositivos (para borrar tokens muertos)
}

// CorrerTurnos junta lo que hay que mandar en cada negocio (marcándolo como mandado en la misma
// transacción) y después lo manda, fuera de la transacción.
func (e *Envios) CorrerTurnos(ctx context.Context, ahora time.Time) (any, error) {
	if e == nil || e.APNs == nil {
		return nil, nil
	}
	var cola []envioTurno
	err := trabajos.PorNegocio(ctx, e.Pool, func(negocio string, tx pgx.Tx) error {
		act, err := actividades(ctx, tx, ahora)
		if err != nil {
			return err
		}
		cola = append(cola, act...)
		cli, err := avisosCliente(ctx, tx, ahora)
		if err != nil {
			return err
		}
		cola = append(cola, cli...)
		return nil
	})
	mandados, muertos := 0, 0
	var invalidos = map[string][]string{}
	for _, x := range cola {
		err := e.APNs.Enviar(ctx, x.envio)
		switch {
		case errors.Is(err, apns.ErrTokenInvalido):
			invalidos[x.tabla] = append(invalidos[x.tabla], x.token)
		case err != nil:
			e.Log.Warn("push: no salió un aviso de turno", "tabla", x.tabla, "error", err)
		default:
			mandados++
		}
	}
	if len(invalidos) > 0 {
		// Los tokens muertos se borran negocio por negocio (las tablas tienen RLS).
		_ = trabajos.PorNegocio(ctx, e.Pool, func(_ string, tx pgx.Tx) error {
			for tabla, ts := range invalidos {
				tag, err := tx.Exec(ctx, "DELETE FROM "+pgx.Identifier{tabla}.Sanitize()+" WHERE token = ANY($1)", ts)
				if err != nil {
					return err
				}
				muertos += int(tag.RowsAffected())
			}
			return nil
		})
	}
	return map[string]int{"mandados": mandados, "borrados": muertos}, err
}

// actividades: las Live Activities cuyo turno cambió (o ya terminó). Un turno atendido, cancelado,
// ausente o que terminó hace más de 15 minutos manda «end» y se borra; si no, «update» con el fin y
// quién sigue.
func actividades(ctx context.Context, tx pgx.Tx, ahora time.Time) ([]envioTurno, error) {
	filas, err := tx.Query(ctx, `SELECT a.id::text, a.token, a.entorno, t.estado, t.fin, coalesce(t.profesional_id::text, ''), t.actualizado_en
		FROM turno_actividades a JOIN turnos t ON t.id = a.turno_id
		WHERE t.actualizado_en > a.enviado_en OR t.fin < $1
		FOR UPDATE OF a SKIP LOCKED`, ahora.Add(-15*time.Minute))
	if err != nil {
		return nil, err
	}
	type fila struct {
		id, token, entorno, estado, profesional string
		fin, actualizado                        time.Time
	}
	fs, err := pgx.CollectRows(filas, func(r pgx.CollectableRow) (fila, error) {
		var f fila
		err := r.Scan(&f.id, &f.token, &f.entorno, &f.estado, &f.fin, &f.profesional, &f.actualizado)
		return f, err
	})
	if err != nil {
		return nil, err
	}
	var out []envioTurno
	for _, f := range fs {
		fin := f.estado == "atendido" || f.estado == "cancelado" || f.estado == "ausente" || f.fin.Before(ahora.Add(-15*time.Minute))
		siguiente := ""
		if !fin && f.profesional != "" {
			var inicio time.Time
			var nombre string
			err := tx.QueryRow(ctx, `SELECT t.inicio, c.nombre FROM turnos t JOIN clientes c ON c.id = t.cliente_id
				WHERE t.profesional_id = $1 AND t.inicio >= $2 AND t.estado IN ('pendiente', 'confirmado', 'pendiente_pago')
				ORDER BY t.inicio LIMIT 1`, f.profesional, f.fin).Scan(&inicio, &nombre)
			if err == nil {
				var zona string
				_ = tx.QueryRow(ctx, "SELECT zona_horaria FROM negocios WHERE id = app.negocio_actual()").Scan(&zona)
				siguiente = horaEn(inicio, zona) + " · " + primerNombre(nombre)
			} else if !errors.Is(err, pgx.ErrNoRows) {
				return nil, err
			}
		}
		out = append(out, envioTurno{envio: apns.Envio{Token: f.token, Entorno: f.entorno, Tipo: apns.TipoLiveActivity,
			Payload: PayloadActividad(ahora, fin, f.fin, siguiente), Vence: ahora.Add(time.Hour)}, token: f.token, tabla: "turno_actividades"})
		if fin {
			if _, err := tx.Exec(ctx, "DELETE FROM turno_actividades WHERE id = $1", f.id); err != nil {
				return nil, err
			}
		} else if _, err := tx.Exec(ctx, "UPDATE turno_actividades SET enviado_en = $2 WHERE id = $1", f.id, f.actualizado); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// PayloadActividad arma el push de una Live Activity (content-state = AtributosTurno.ContentState de
// la app: fin en segundos desde 2001 y quién sigue).
func PayloadActividad(ahora time.Time, terminar bool, fin time.Time, siguiente string) []byte {
	aps := map[string]any{
		"timestamp":     ahora.Unix(),
		"event":         "update",
		"content-state": map[string]any{"fin": segundos2001(fin), "siguiente": siguiente},
		"stale-date":    fin.Unix(),
	}
	if terminar {
		aps["event"] = "end"
		aps["dismissal-date"] = ahora.Add(2 * time.Minute).Unix()
	}
	b, _ := json.Marshal(map[string]any{"aps": aps})
	return b
}

// avisosCliente: lo que se le avisó a la clienta por un turno (confirmación, recordatorio,
// cancelación, cambio, reseña), también a los iPhone que siguen ese turno. Sale cuando el aviso se
// mandó o se resolvió (los recordatorios se encolan antes), y sólo si es reciente.
func avisosCliente(ctx context.Context, tx pgx.Tx, ahora time.Time) ([]envioTurno, error) {
	filas, err := tx.Query(ctx, `UPDATE avisos SET push_cliente = false WHERE id IN (
			SELECT id FROM avisos WHERE push_cliente AND (turno_id IS NULL OR estado <> 'pendiente' OR creado_en < $1)
			ORDER BY creado_en LIMIT 200 FOR UPDATE SKIP LOCKED)
		RETURNING id::text, tipo, coalesce(turno_id::text, ''), coalesce(enviado_en, creado_en)`, ahora.Add(-Frescura))
	if err != nil {
		return nil, err
	}
	type fila struct {
		id, tipo, turno string
		cuando          time.Time
	}
	fs, err := pgx.CollectRows(filas, func(r pgx.CollectableRow) (fila, error) {
		var f fila
		err := r.Scan(&f.id, &f.tipo, &f.turno, &f.cuando)
		return f, err
	})
	if err != nil {
		return nil, err
	}
	var out []envioTurno
	for _, f := range fs {
		if f.turno == "" || ahora.Sub(f.cuando) > Frescura {
			continue
		}
		var codigo, servicio, negocio, zona, estado string
		var inicio time.Time
		err := tx.QueryRow(ctx, `SELECT t.codigo, t.servicio, t.inicio, t.estado, n.nombre, n.zona_horaria
			FROM turnos t JOIN negocios n ON n.id = t.negocio_id WHERE t.id = $1`, f.turno).
			Scan(&codigo, &servicio, &inicio, &estado, &negocio, &zona)
		if errors.Is(err, pgx.ErrNoRows) {
			continue
		}
		if err != nil {
			return nil, err
		}
		titulo, cuerpo, ok := TextoCliente(f.tipo, negocio, servicio, inicio, zona, ahora)
		if !ok {
			continue
		}
		ds, err := tx.Query(ctx, "SELECT token, entorno FROM turno_dispositivos WHERE turno_id = $1", f.turno)
		if err != nil {
			return nil, err
		}
		type disp struct{ token, entorno string }
		lista, err := pgx.CollectRows(ds, func(r pgx.CollectableRow) (disp, error) {
			var d disp
			err := r.Scan(&d.token, &d.entorno)
			return d, err
		})
		if err != nil {
			return nil, err
		}
		for _, d := range lista {
			out = append(out, envioTurno{envio: apns.Envio{Token: d.token, Entorno: d.entorno,
				Payload: PayloadCliente(titulo, cuerpo, codigo), Colapso: "aviso-" + f.id, Vence: ahora.Add(Vida)},
				token: d.token, tabla: "turno_dispositivos"})
		}
	}
	return out, nil
}

// TextoCliente: el aviso al iPhone de la clienta según el tipo de aviso del turno.
func TextoCliente(tipo, negocio, servicio string, inicio time.Time, zona string, ahora time.Time) (titulo, cuerpo string, ok bool) {
	cuando := cuandoEn(inicio, zona, ahora)
	switch tipo {
	case "reservado":
		return "Reservaste en " + negocio, fmt.Sprintf("%s · %s", servicio, cuando), true
	case "recordatorio":
		return "Te esperan " + cuando, fmt.Sprintf("%s en %s. Si no podés ir, avisá desde la app.", servicio, negocio), true
	case "cancelado":
		return "Se canceló tu turno", fmt.Sprintf("%s en %s, %s.", servicio, negocio, cuando), true
	case "reprogramado":
		return "Cambió tu turno", fmt.Sprintf("Ahora es %s: %s en %s.", cuando, servicio, negocio), true
	case "resena":
		return "¿Cómo te fue en " + negocio + "?", "Contanos en un toque: tu reseña ayuda a otras personas.", true
	}
	return "", "", false
}

// PayloadCliente: el aviso de la clienta, que abre su turno en la app (ruta turno-cliente/<código>).
func PayloadCliente(titulo, cuerpo, codigo string) []byte {
	b, _ := json.Marshal(map[string]any{
		"aps":  map[string]any{"alert": map[string]string{"title": titulo, "body": cuerpo}, "sound": "default", "thread-id": "turno-" + codigo},
		"ruta": "turno-cliente/" + codigo,
	})
	return b
}

var diasSemana = []string{"domingo", "lunes", "martes", "miércoles", "jueves", "viernes", "sábado"}
var meses = []string{"enero", "febrero", "marzo", "abril", "mayo", "junio", "julio", "agosto", "septiembre", "octubre", "noviembre", "diciembre"}

func zonaDe(zona string) *time.Location {
	if l, err := time.LoadLocation(zona); err == nil {
		return l
	}
	l, _ := time.LoadLocation("America/Argentina/Buenos_Aires")
	if l == nil {
		return time.UTC
	}
	return l
}

func horaEn(t time.Time, zona string) string { return t.In(zonaDe(zona)).Format("15:04") }

// cuandoEn: «hoy a las 16:00», «mañana a las 10:30», «el jueves 12 de noviembre a las 9:00».
func cuandoEn(t time.Time, zona string, ahora time.Time) string {
	l := zonaDe(zona)
	t, ahora = t.In(l), ahora.In(l)
	h := t.Format("15:04")
	dia := func(x time.Time) time.Time { return time.Date(x.Year(), x.Month(), x.Day(), 0, 0, 0, 0, l) }
	switch int(dia(t).Sub(dia(ahora)).Hours() / 24) {
	case 0:
		return "hoy a las " + h
	case 1:
		return "mañana a las " + h
	}
	return fmt.Sprintf("el %s %d de %s a las %s", diasSemana[t.Weekday()], t.Day(), meses[t.Month()-1], h)
}

func primerNombre(s string) string {
	if i := strings.IndexByte(strings.TrimSpace(s), ' '); i > 0 {
		return strings.TrimSpace(s)[:i]
	}
	return strings.TrimSpace(s)
}

func esUUID(s string) bool { return uuidRE.MatchString(s) }
