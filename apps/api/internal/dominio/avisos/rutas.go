package avisos

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/auditoria"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/problema"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/sesion"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/web"
)

// Mostrador: los que ven y reenvían los mails (la recepción también: es la que atiende al cliente
// que dice que no le llegó).
var Mostrador = []string{sesion.Duenio, sesion.Encargado, sesion.Recepcion}

// Rutas del dominio.
func (s *Servicio) Rutas(mux *http.ServeMux) {
	ses := s.Sesion
	mux.HandleFunc("GET /v1/panel/avisos", ses.Panel(Mostrador, web.Envolver(s.listar)))
	mux.HandleFunc("POST /v1/panel/avisos/{id}/reenviar", ses.Panel(Mostrador, web.Envolver(s.reenviar)))
	mux.HandleFunc("GET /v1/panel/campanita", ses.Panel(sesion.Agenda, web.Envolver(s.campanita)))
	mux.HandleFunc("GET /v1/panel/campanita/sin-leer", ses.Panel(sesion.Agenda, web.Envolver(s.sinLeer)))
	mux.HandleFunc("POST /v1/panel/campanita/vista", ses.Panel(sesion.Agenda, web.Envolver(s.marcarVista)))
	mux.HandleFunc("GET /v1/panel/ajustes/avisos", ses.Panel(sesion.Agenda, web.Envolver(s.leerAjustes)))
	mux.HandleFunc("PUT /v1/panel/ajustes/avisos", ses.Panel(sesion.Gestion, web.Envolver(s.guardarAjustes)))
	// Link de baja del cliente (el token es el secreto).
	mux.HandleFunc("GET /v1/preferencias/{token}", web.Envolver(s.verPreferencias))
	mux.HandleFunc("PUT /v1/preferencias/{token}", web.Envolver(s.guardarPreferencias))
}

// Registro de un aviso como lo ve el panel.
type Registro struct {
	ID          string          `json:"id"`
	Tipo        string          `json:"tipo"`
	Para        string          `json:"para"`
	TurnoID     *string         `json:"turnoId"`
	Cliente     *string         `json:"cliente"`
	Estado      string          `json:"estado"`
	Intentos    int             `json:"intentos"`
	UltimoError *string         `json:"ultimoError"`
	Proximo     time.Time       `json:"proximoIntento"`
	EnviadoEn   *time.Time      `json:"enviadoEn"`
	CreadoEn    time.Time       `json:"creadoEn"`
	Datos       json.RawMessage `json:"-"`
}

const columnasRegistro = `a.id::text, a.tipo, a.para, a.turno_id::text, c.nombre, a.estado, a.intentos, a.ultimo_error,
	a.proximo_intento, a.enviado_en, a.creado_en`

const desdeRegistro = ` FROM avisos a LEFT JOIN turnos t ON t.id = a.turno_id LEFT JOIN clientes c ON c.id = t.cliente_id`

func escanear(f pgx.Row, r *Registro) error {
	return f.Scan(&r.ID, &r.Tipo, &r.Para, &r.TurnoID, &r.Cliente, &r.Estado, &r.Intentos, &r.UltimoError, &r.Proximo, &r.EnviadoEn, &r.CreadoEn)
}

// listar: el registro de envíos, del más nuevo al más viejo. Filtros: estado, tipo, turno y antes
// (cursor: el creadoEn del último de la página anterior).
func (s *Servicio) listar(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	q := r.URL.Query()
	cond := []string{"true"}
	var args []any
	agregar := func(sql string, v any) {
		args = append(args, v)
		cond = append(cond, strings.ReplaceAll(sql, "?", "$"+strconv.Itoa(len(args))))
	}
	if v := q.Get("estado"); v != "" {
		agregar("a.estado = ANY(?)", strings.Split(v, ","))
	}
	if v := q.Get("tipo"); v != "" {
		agregar("a.tipo = ANY(?)", strings.Split(v, ","))
	}
	if v := q.Get("turno"); v != "" {
		agregar("a.turno_id::text = ?", v)
	}
	if v := q.Get("antes"); v != "" {
		t, err := time.Parse(time.RFC3339Nano, v)
		if err != nil {
			return problema.Invalido("antes va como fecha RFC 3339.")
		}
		agregar("a.creado_en < ?", t)
	}
	out := []Registro{}
	err := s.Pool.EnNegocio(ctx, sesion.De(ctx).NegocioID, func(tx pgx.Tx) error {
		filas, err := tx.Query(ctx, "SELECT "+columnasRegistro+desdeRegistro+" WHERE "+strings.Join(cond, " AND ")+
			" ORDER BY a.creado_en DESC, a.id LIMIT 50", args...)
		if err != nil {
			return err
		}
		defer filas.Close()
		for filas.Next() {
			var x Registro
			if err := escanear(filas, &x); err != nil {
				return err
			}
			out = append(out, x)
		}
		return filas.Err()
	})
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusOK, map[string]any{"avisos": out})
	return nil
}

// reenviar vuelve a poner el aviso en la cola, para ya y con los intentos en cero: quien aprieta el
// botón suele haber arreglado lo que fallaba (el email del cliente, por ejemplo). Sirve también para
// mandar de nuevo uno que salió y el cliente dice que no le llegó.
func (s *Servicio) reenviar(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	var x Registro
	err := s.Pool.EnNegocio(ctx, sesion.De(ctx).NegocioID, func(tx pgx.Tx) error {
		var estado string
		if err := tx.QueryRow(ctx, "SELECT estado FROM avisos WHERE id = $1 FOR UPDATE", r.PathValue("id")).Scan(&estado); err != nil {
			return err
		}
		if estado == "pendiente" {
			return problema.Conflicto("en_cola", "Ese aviso ya está en la cola: sale en un momento.")
		}
		if _, err := tx.Exec(ctx, `UPDATE avisos SET estado = 'pendiente', intentos = 0, proximo_intento = $2, enviado_en = NULL
			WHERE id = $1`, r.PathValue("id"), s.ahora()); err != nil {
			return err
		}
		if err := escanear(tx.QueryRow(ctx, "SELECT "+columnasRegistro+desdeRegistro+" WHERE a.id = $1", r.PathValue("id")), &x); err != nil {
			return err
		}
		return auditoria.Anotar(ctx, tx, auditoria.Registro{Accion: "aviso.reenviado", Entidad: "aviso", EntidadID: x.ID,
			Detalle: map[string]any{"tipo": x.Tipo, "para": x.Para, "estadoAnterior": estado}})
	})
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusAccepted, x)
	return nil
}

// NotificacionPanel de la campanita.
type NotificacionPanel struct {
	ID       string    `json:"id"`
	Tipo     string    `json:"tipo"`
	Titulo   string    `json:"titulo"`
	Texto    string    `json:"texto"`
	TurnoID  *string   `json:"turnoId"`
	CreadoEn time.Time `json:"creadoEn"`
	Nueva    bool      `json:"nueva"`
}

// La campanita muestra los últimos siete días.
const ventanaCampanita = 7 * 24 * time.Hour

// vistaDesde: la última vez que la persona abrió la campanita; si nunca (o hace más de la ventana),
// el comienzo de la ventana, para no arrancar con un número inventado. Todo en la hora de la base,
// la misma con la que se crean las notificaciones.
func vistaDesde(ctx context.Context, tx pgx.Tx, clerkID string) (time.Time, error) {
	var v time.Time
	err := tx.QueryRow(ctx, `SELECT greatest(coalesce((SELECT vista_en FROM notificaciones_vistas WHERE clerk_id = $1), '-infinity'),
		now() - make_interval(secs => $2))`, clerkID, ventanaCampanita.Seconds()).Scan(&v)
	return v, err
}

func (s *Servicio) campanita(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	ses := sesion.De(ctx)
	out := []NotificacionPanel{}
	sinLeer := 0
	err := s.Pool.EnNegocio(ctx, ses.NegocioID, func(tx pgx.Tx) error {
		vista, err := vistaDesde(ctx, tx, ses.ClerkID)
		if err != nil {
			return err
		}
		filas, err := tx.Query(ctx, `SELECT id::text, tipo, titulo, texto, turno_id::text, creado_en, creado_en > $2 FROM notificaciones
			WHERE creado_en >= now() - make_interval(secs => $1) ORDER BY creado_en DESC LIMIT 30`, ventanaCampanita.Seconds(), vista)
		if err != nil {
			return err
		}
		if out, err = pgx.CollectRows(filas, pgx.RowToStructByPos[NotificacionPanel]); err != nil {
			return err
		}
		return tx.QueryRow(ctx, "SELECT count(*) FROM notificaciones WHERE creado_en > $1", vista).Scan(&sinLeer)
	})
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusOK, map[string]any{"notificaciones": out, "sinLeer": sinLeer})
	return nil
}

func (s *Servicio) sinLeer(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	ses := sesion.De(ctx)
	n := 0
	err := s.Pool.EnNegocio(ctx, ses.NegocioID, func(tx pgx.Tx) error {
		vista, err := vistaDesde(ctx, tx, ses.ClerkID)
		if err != nil {
			return err
		}
		return tx.QueryRow(ctx, "SELECT count(*) FROM notificaciones WHERE creado_en > $1", vista).Scan(&n)
	})
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusOK, map[string]int{"sinLeer": n})
	return nil
}

// marcarVista: abrió la campanita. Se marca con la hora de la base (la misma con la que se crean las
// notificaciones), así nada queda entre medio.
func (s *Servicio) marcarVista(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	ses := sesion.De(ctx)
	err := s.Pool.EnNegocio(ctx, ses.NegocioID, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `INSERT INTO notificaciones_vistas (negocio_id, clerk_id, vista_en) VALUES (app.negocio_actual(), $1, now())
			ON CONFLICT (negocio_id, clerk_id) DO UPDATE SET vista_en = excluded.vista_en`, ses.ClerkID)
		return err
	})
	if err != nil {
		return err
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}

// AjustesAvisos del negocio.
type AjustesAvisos struct {
	// Cuántas horas después de un turno atendido sale el pedido de reseña.
	ResenaDemoraHoras int `json:"resenaDemoraHoras"`
	// Mandarle a la dueña el resumen de mañana a la tarde.
	ResumenDiario bool `json:"resumenDiario"`
}

func leerAjustes(ctx context.Context, tx pgx.Tx) (AjustesAvisos, error) {
	var a AjustesAvisos
	err := tx.QueryRow(ctx, `SELECT resena_demora_horas, resumen_diario FROM ajustes WHERE negocio_id = app.negocio_actual()`).
		Scan(&a.ResenaDemoraHoras, &a.ResumenDiario)
	return a, err
}

func (s *Servicio) leerAjustes(w http.ResponseWriter, r *http.Request) error {
	var a AjustesAvisos
	err := s.Pool.EnNegocio(r.Context(), sesion.De(r.Context()).NegocioID, func(tx pgx.Tx) (err error) {
		a, err = leerAjustes(r.Context(), tx)
		return err
	})
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusOK, a)
	return nil
}

func (s *Servicio) guardarAjustes(w http.ResponseWriter, r *http.Request) error {
	var a AjustesAvisos
	if err := web.Leer(w, r, &a); err != nil {
		return err
	}
	if a.ResenaDemoraHoras < 0 || a.ResenaDemoraHoras > 168 {
		return problema.Invalido("El pedido de reseña sale entre 0 y 168 horas después.", problema.Campo{Ruta: "resenaDemoraHoras", Mensaje: "fuera de rango"})
	}
	ctx := r.Context()
	err := s.Pool.EnNegocio(ctx, sesion.De(ctx).NegocioID, func(tx pgx.Tx) error {
		antes, err := leerAjustes(ctx, tx)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `UPDATE ajustes SET resena_demora_horas = $1, resumen_diario = $2, actualizado_en = now()
			WHERE negocio_id = app.negocio_actual()`, a.ResenaDemoraHoras, a.ResumenDiario); err != nil {
			return err
		}
		return auditoria.Anotar(ctx, tx, auditoria.Registro{Accion: "ajustes.avisos", Entidad: "ajustes",
			Detalle: map[string]any{"antes": antes, "despues": a}})
	})
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusOK, a)
	return nil
}

// PreferenciasPublicas: lo que ve el cliente en el link de baja.
type PreferenciasPublicas struct {
	Negocio       string `json:"negocio"`
	Slug          string `json:"slug"`
	Cliente       string `json:"cliente"`
	Email         bool   `json:"email"`
	Recordatorios bool   `json:"recordatorios"`
}

func (s *Servicio) conPreferencias(ctx context.Context, token string, fn func(tx pgx.Tx, p *PreferenciasPublicas) error) (PreferenciasPublicas, error) {
	var p PreferenciasPublicas
	negocio, err := NegocioPorToken(ctx, s.Pool.EnTx, "preferencias", token)
	if err != nil {
		return p, err
	}
	err = s.Pool.EnNegocio(ctx, negocio, func(tx pgx.Tx) error {
		if err := tx.QueryRow(ctx, `SELECT n.nombre, n.slug, c.nombre, p.email, p.recordatorios FROM preferencias p
			JOIN clientes c ON c.id = p.cliente_id JOIN negocios n ON n.id = p.negocio_id WHERE p.token = $1 FOR UPDATE OF p`, token).
			Scan(&p.Negocio, &p.Slug, &p.Cliente, &p.Email, &p.Recordatorios); err != nil {
			return err
		}
		return fn(tx, &p)
	})
	return p, err
}

func (s *Servicio) verPreferencias(w http.ResponseWriter, r *http.Request) error {
	p, err := s.conPreferencias(r.Context(), r.PathValue("token"), func(pgx.Tx, *PreferenciasPublicas) error { return nil })
	if err != nil {
		return err
	}
	p.Cliente = primerNombre(p.Cliente)
	web.JSON(w, http.StatusOK, p)
	return nil
}

func (s *Servicio) guardarPreferencias(w http.ResponseWriter, r *http.Request) error {
	var in struct {
		Email         *bool `json:"email"`
		Recordatorios *bool `json:"recordatorios"`
	}
	if err := web.Leer(w, r, &in); err != nil {
		return err
	}
	ctx := r.Context()
	p, err := s.conPreferencias(ctx, r.PathValue("token"), func(tx pgx.Tx, p *PreferenciasPublicas) error {
		return tx.QueryRow(ctx, `UPDATE preferencias SET email = coalesce($2, email), recordatorios = coalesce($3, recordatorios),
			actualizado_en = now() WHERE token = $1 RETURNING email, recordatorios`, r.PathValue("token"), in.Email, in.Recordatorios).
			Scan(&p.Email, &p.Recordatorios)
	})
	if err != nil {
		return err
	}
	p.Cliente = primerNombre(p.Cliente)
	web.JSON(w, http.StatusOK, p)
	return nil
}

// NegocioPorToken: de un token público (preferencias, resena, invitacion) a su negocio, o 404.
// enTx es Pool.EnTx (sin negocio todavía).
func NegocioPorToken(ctx context.Context, enTx func(context.Context, func(pgx.Tx) error) error, tipo, token string) (string, error) {
	var id *string
	err := enTx(ctx, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, "SELECT app.negocio_por_token($1, $2)::text", tipo, token).Scan(&id)
	})
	if err != nil {
		return "", err
	}
	if id == nil {
		return "", problema.NoEncontrado("Ese link no existe o ya no sirve.")
	}
	return *id, nil
}
