// Package grupos: las sucursales (BusinessGroup de Jiku). Una persona agrupa varios negocios de los
// que es dueña, los ve juntos y saca un reporte consolidado (turnos e ingresos por sucursal).
//
// El grupo es de la persona, no de un negocio: va por /v1/grupos con sesión de persona (sin
// X-Negocio). Cada sucursal se lee con su negocio fijado (EnNegocio), una por vez, y sólo entran
// las que la persona tiene como dueña ahora: si deja de serlo, sale del reporte aunque siga en el grupo.
package grupos

import (
	"context"
	"net/http"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"

	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/db"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/problema"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/sesion"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/web"
)

// Servicio de grupos.
type Servicio struct {
	Pool   *db.Pool
	Sesion *sesion.Servicio
	// Ahora: el reloj (nil: time.Now).
	Ahora func() time.Time
}

// Sucursal de un grupo (o un negocio propio todavía sin grupo).
type Sucursal struct {
	ID     string `json:"id"`
	Slug   string `json:"slug"`
	Nombre string `json:"nombre"`
	Ciudad string `json:"ciudad"`
}

// Grupo con sus sucursales.
type Grupo struct {
	ID         string     `json:"id"`
	Nombre     string     `json:"nombre"`
	Sucursales []Sucursal `json:"sucursales"`
}

// Numeros de una sucursal en el período.
type Numeros struct {
	Turnos     int   `json:"turnos"` // sin cancelados ni los que esperan el pago
	Atendidos  int   `json:"atendidos"`
	Ausentes   int   `json:"ausentes"`
	Cancelados int   `json:"cancelados"`
	Clientes   int   `json:"clientes"` // distintos
	Ingresos   int64 `json:"ingresos"` // precio de los atendidos, en pesos
}

// Rutas del dominio.
func (s *Servicio) Rutas(mux *http.ServeMux) {
	p := func(h web.Manejador) http.HandlerFunc { return s.Sesion.Persona(web.Envolver(h)) }
	mux.HandleFunc("GET /v1/grupos", p(s.listar))
	mux.HandleFunc("POST /v1/grupos", p(s.crear))
	mux.HandleFunc("PATCH /v1/grupos/{id}", p(s.renombrar))
	mux.HandleFunc("DELETE /v1/grupos/{id}", p(s.borrar))
	mux.HandleFunc("PUT /v1/grupos/{id}/sucursales/{negocio}", p(s.sumar))
	mux.HandleFunc("DELETE /v1/grupos/{id}/sucursales/{negocio}", p(s.sacar))
	mux.HandleFunc("GET /v1/grupos/{id}/reporte", p(s.reporte))
}

func (s *Servicio) ahora() time.Time {
	if s.Ahora == nil {
		return time.Now()
	}
	return s.Ahora()
}

// comoPersona corre fn con la persona fijada (la política de grupos compara app.clerk_id).
func comoPersona(ctx context.Context, tx pgx.Tx, clerkID string) error {
	_, err := tx.Exec(ctx, "SELECT set_config('app.clerk_id', $1, true)", clerkID)
	return err
}

func (s *Servicio) enPersona(ctx context.Context, fn func(pgx.Tx) error) error {
	return s.Pool.EnTx(ctx, func(tx pgx.Tx) error {
		if err := comoPersona(ctx, tx, sesion.De(ctx).ClerkID); err != nil {
			return err
		}
		return fn(tx)
	})
}

// propios: los negocios de los que la persona es dueña, con el grupo de cada uno.
func (s *Servicio) propios(ctx context.Context) ([]Sucursal, map[string]string, error) {
	ns, err := s.Sesion.Negocios(ctx, sesion.De(ctx).ClerkID)
	if err != nil {
		return nil, nil, err
	}
	var out []Sucursal
	grupo := map[string]string{}
	for _, n := range ns {
		if n.Rol != sesion.Duenio {
			continue
		}
		su := Sucursal{ID: n.ID, Slug: n.Slug, Nombre: n.Nombre}
		var g *string
		err := s.Pool.EnNegocio(ctx, n.ID, func(tx pgx.Tx) error {
			return tx.QueryRow(ctx, "SELECT ciudad, grupo_id::text FROM negocios WHERE id = app.negocio_actual()").Scan(&su.Ciudad, &g)
		})
		if err != nil {
			return nil, nil, err
		}
		if g != nil {
			grupo[n.ID] = *g
		}
		out = append(out, su)
	}
	return out, grupo, nil
}

func (s *Servicio) grupos(ctx context.Context) ([]Grupo, error) {
	var gs []Grupo
	err := s.enPersona(ctx, func(tx pgx.Tx) error {
		filas, err := tx.Query(ctx, "SELECT id::text, nombre FROM grupos ORDER BY nombre, creado_en")
		if err != nil {
			return err
		}
		gs, err = pgx.CollectRows(filas, func(f pgx.CollectableRow) (Grupo, error) {
			g := Grupo{Sucursales: []Sucursal{}}
			return g, f.Scan(&g.ID, &g.Nombre)
		})
		return err
	})
	return gs, err
}

// armar devuelve los grupos con sus sucursales y los negocios propios sin grupo.
func (s *Servicio) armar(ctx context.Context) ([]Grupo, []Sucursal, error) {
	gs, err := s.grupos(ctx)
	if err != nil {
		return nil, nil, err
	}
	ps, grupo, err := s.propios(ctx)
	if err != nil {
		return nil, nil, err
	}
	sueltos := []Sucursal{}
	for _, p := range ps {
		i := slices.IndexFunc(gs, func(g Grupo) bool { return g.ID == grupo[p.ID] })
		if i < 0 {
			sueltos = append(sueltos, p)
			continue
		}
		gs[i].Sucursales = append(gs[i].Sucursales, p)
	}
	return gs, sueltos, nil
}

func (s *Servicio) listar(w http.ResponseWriter, r *http.Request) error {
	gs, sueltos, err := s.armar(r.Context())
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusOK, map[string]any{"grupos": gs, "sinGrupo": sueltos})
	return nil
}

func nombre(v string) (string, error) {
	v = strings.TrimSpace(v)
	if n := utf8.RuneCountInString(v); n < 1 || n > 80 {
		return "", problema.Invalido("Revisá el nombre.", problema.Campo{Ruta: "nombre", Mensaje: "de 1 a 80 letras"})
	}
	return v, nil
}

func (s *Servicio) crear(w http.ResponseWriter, r *http.Request) error {
	var in struct {
		Nombre string `json:"nombre"`
	}
	if err := web.Leer(w, r, &in); err != nil {
		return err
	}
	n, err := nombre(in.Nombre)
	if err != nil {
		return err
	}
	ctx := r.Context()
	g := Grupo{Nombre: n, Sucursales: []Sucursal{}}
	err = s.enPersona(ctx, func(tx pgx.Tx) error {
		var cuantos int
		if err := tx.QueryRow(ctx, "SELECT count(*) FROM grupos").Scan(&cuantos); err != nil {
			return err
		}
		if cuantos >= 20 {
			return problema.Conflicto("demasiados_grupos", "Ya tenés 20 grupos.")
		}
		return tx.QueryRow(ctx, "INSERT INTO grupos (duenio, nombre) VALUES ($1, $2) RETURNING id::text",
			sesion.De(ctx).ClerkID, n).Scan(&g.ID)
	})
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusCreated, g)
	return nil
}

func (s *Servicio) renombrar(w http.ResponseWriter, r *http.Request) error {
	var in struct {
		Nombre string `json:"nombre"`
	}
	if err := web.Leer(w, r, &in); err != nil {
		return err
	}
	n, err := nombre(in.Nombre)
	if err != nil {
		return err
	}
	ctx := r.Context()
	err = s.enPersona(ctx, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, "UPDATE grupos SET nombre = $2 WHERE id = $1", r.PathValue("id"), n)
		if err == nil && tag.RowsAffected() == 0 {
			return pgx.ErrNoRows
		}
		return err
	})
	if err != nil {
		return err
	}
	return s.uno(w, r)
}

// borrar el grupo: las sucursales quedan sueltas (la FK las pone en NULL), no se borran.
func (s *Servicio) borrar(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	err := s.enPersona(ctx, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, "DELETE FROM grupos WHERE id = $1", r.PathValue("id"))
		if err == nil && tag.RowsAffected() == 0 {
			return pgx.ErrNoRows
		}
		return err
	})
	if err != nil {
		return err
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}

// uno responde el grupo del path con sus sucursales.
func (s *Servicio) uno(w http.ResponseWriter, r *http.Request) error {
	gs, _, err := s.armar(r.Context())
	if err != nil {
		return err
	}
	i := slices.IndexFunc(gs, func(g Grupo) bool { return g.ID == r.PathValue("id") })
	if i < 0 {
		return pgx.ErrNoRows
	}
	web.JSON(w, http.StatusOK, gs[i])
	return nil
}

// duenia exige que la persona sea dueña del negocio.
func (s *Servicio) duenia(ctx context.Context, negocio string) error {
	ns, err := s.Sesion.Negocios(ctx, sesion.De(ctx).ClerkID)
	if err != nil {
		return err
	}
	if !slices.ContainsFunc(ns, func(n sesion.Negocio) bool { return n.ID == negocio && n.Rol == sesion.Duenio }) {
		return problema.SinPermiso("Sólo la dueña del negocio lo puede sumar o sacar de un grupo.")
	}
	return nil
}

// ponerGrupo fija el grupo del negocio (o lo saca con grupo nil), verificando en la misma
// transacción que el grupo sea de la persona.
func (s *Servicio) ponerGrupo(r *http.Request, grupo *string, soloSi string) error {
	ctx := r.Context()
	negocio := r.PathValue("negocio")
	if err := s.duenia(ctx, negocio); err != nil {
		return err
	}
	return s.Pool.EnNegocio(ctx, negocio, func(tx pgx.Tx) error {
		if err := comoPersona(ctx, tx, sesion.De(ctx).ClerkID); err != nil {
			return err
		}
		var mio bool
		if err := tx.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM grupos WHERE id = $1)", r.PathValue("id")).Scan(&mio); err != nil {
			return err
		}
		if !mio {
			return pgx.ErrNoRows
		}
		_, err := tx.Exec(ctx, "UPDATE negocios SET grupo_id = $1 WHERE id = app.negocio_actual() AND ($2 = '' OR grupo_id::text = $2)",
			grupo, soloSi)
		return err
	})
}

func (s *Servicio) sumar(w http.ResponseWriter, r *http.Request) error {
	id := r.PathValue("id")
	if err := s.ponerGrupo(r, &id, ""); err != nil {
		return err
	}
	return s.uno(w, r)
}

func (s *Servicio) sacar(w http.ResponseWriter, r *http.Request) error {
	if err := s.ponerGrupo(r, nil, r.PathValue("id")); err != nil {
		return err
	}
	return s.uno(w, r)
}

// reporte: desde y hasta (YYYY-MM-DD, inclusive, en la zona de cada sucursal); por defecto los
// últimos 30 días.
func (s *Servicio) reporte(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	q := r.URL.Query()
	hoy := s.ahora()
	desde, hasta := q.Get("desde"), q.Get("hasta")
	if hasta == "" {
		hasta = hoy.Format("2006-01-02")
	}
	if desde == "" {
		desde = hoy.AddDate(0, 0, -29).Format("2006-01-02")
	}
	d, err1 := time.Parse("2006-01-02", desde)
	h, err2 := time.Parse("2006-01-02", hasta)
	if err1 != nil || err2 != nil || h.Before(d) || h.Sub(d) > 366*24*time.Hour {
		return problema.Invalido("Revisá las fechas: desde y hasta en YYYY-MM-DD, hasta un año.")
	}
	gs, _, err := s.armar(ctx)
	if err != nil {
		return err
	}
	i := slices.IndexFunc(gs, func(g Grupo) bool { return g.ID == r.PathValue("id") })
	if i < 0 {
		return pgx.ErrNoRows
	}
	type fila struct {
		Sucursal
		Numeros
	}
	out := struct {
		Grupo      string  `json:"grupo"`
		Desde      string  `json:"desde"`
		Hasta      string  `json:"hasta"`
		Total      Numeros `json:"total"`
		Sucursales []fila  `json:"sucursales"`
	}{Grupo: gs[i].Nombre, Desde: desde, Hasta: hasta, Sucursales: []fila{}}
	for _, su := range gs[i].Sucursales {
		var n Numeros
		err := s.Pool.EnNegocio(ctx, su.ID, func(tx pgx.Tx) error {
			return tx.QueryRow(ctx, `SELECT
				count(*) FILTER (WHERE t.estado NOT IN ('cancelado', 'pendiente_pago')),
				count(*) FILTER (WHERE t.estado = 'atendido'),
				count(*) FILTER (WHERE t.estado = 'ausente'),
				count(*) FILTER (WHERE t.estado = 'cancelado'),
				count(DISTINCT t.cliente_id) FILTER (WHERE t.estado NOT IN ('cancelado', 'pendiente_pago')),
				coalesce(sum(t.precio) FILTER (WHERE t.estado = 'atendido'), 0)
				FROM negocios n LEFT JOIN turnos t ON t.negocio_id = n.id
					AND t.inicio >= ($1::date::timestamp AT TIME ZONE n.zona_horaria)
					AND t.inicio < (($2::date + 1)::timestamp AT TIME ZONE n.zona_horaria)
				WHERE n.id = app.negocio_actual()`, desde, hasta).Scan(
				&n.Turnos, &n.Atendidos, &n.Ausentes, &n.Cancelados, &n.Clientes, &n.Ingresos)
		})
		if err != nil {
			return err
		}
		out.Total.Turnos += n.Turnos
		out.Total.Atendidos += n.Atendidos
		out.Total.Ausentes += n.Ausentes
		out.Total.Cancelados += n.Cancelados
		out.Total.Clientes += n.Clientes
		out.Total.Ingresos += n.Ingresos
		out.Sucursales = append(out.Sucursales, fila{su, n})
	}
	web.JSON(w, http.StatusOK, out)
	return nil
}
