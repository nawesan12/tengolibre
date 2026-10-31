// Package resenas: el pedido de reseña después de un turno atendido (con un token por turno), la
// página pública para dejarla, la moderación del panel (ocultar, responder) y el promedio que muestra
// la página del negocio.
package resenas

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"

	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/avisos"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/auditoria"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/db"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/problema"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/sesion"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/trabajos"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/web"
)

// Servicio de reseñas. Ahora es el reloj (tests); nil: time.Now.
type Servicio struct {
	Pool   *db.Pool
	Sesion *sesion.Servicio
	Ahora  func() time.Time
}

func (s *Servicio) ahora() time.Time {
	if s.Ahora != nil {
		return s.Ahora()
	}
	return time.Now()
}

// Rutas del dominio.
func (s *Servicio) Rutas(mux *http.ServeMux) {
	// Link del pedido de reseña (el token es el secreto).
	mux.HandleFunc("GET /v1/resenas/{token}", web.Envolver(s.verPedido))
	mux.HandleFunc("POST /v1/resenas/{token}", web.Envolver(s.dejar))
	// Página pública del negocio.
	mux.HandleFunc("GET /v1/p/{slug}/resenas", web.Envolver(s.publicas))
	// Panel.
	mux.HandleFunc("GET /v1/panel/resenas", s.Sesion.Panel(sesion.Agenda, web.Envolver(s.listar)))
	mux.HandleFunc("PATCH /v1/panel/resenas/{id}", s.Sesion.Panel(sesion.Gestion, web.Envolver(s.moderar)))
}

// El link sirve una semana, y no se pide reseña de un turno de hace más de una semana (el cliente
// casi no se acuerda, y una primera corrida no le manda a nadie meses de turnos viejos).
const (
	vigencia  = 7 * 24 * time.Hour
	maxAntigu = 7 * 24 * time.Hour
)

// Pedir es el trabajo del worker: en cada negocio, a los turnos atendidos que terminaron hace más de
// ajustes.resena_demora_horas (y menos de una semana), sin pedido ni reseña y con email, les crea el
// token y encola el mail en la misma transacción.
func (s *Servicio) Pedir(ctx context.Context, ahora time.Time) (any, error) {
	pedidos := 0
	err := trabajos.PorNegocio(ctx, s.Pool, func(_ string, tx pgx.Tx) error {
		var demora int
		if err := tx.QueryRow(ctx, "SELECT resena_demora_horas FROM ajustes WHERE negocio_id = app.negocio_actual()").Scan(&demora); err != nil {
			return err
		}
		filas, err := tx.Query(ctx, `SELECT t.id::text FROM turnos t JOIN clientes c ON c.id = t.cliente_id
			WHERE t.estado = 'atendido' AND t.fin <= $1 AND t.fin >= $2 AND c.email <> ''
				AND NOT EXISTS (SELECT 1 FROM resena_tokens r WHERE r.turno_id = t.id)
				AND NOT EXISTS (SELECT 1 FROM resenas r WHERE r.turno_id = t.id)
			ORDER BY t.fin LIMIT 100`, ahora.Add(-time.Duration(demora)*time.Hour), ahora.Add(-maxAntigu))
		if err != nil {
			return err
		}
		ids, err := pgx.CollectRows(filas, pgx.RowTo[string])
		if err != nil {
			return err
		}
		for _, id := range ids {
			token := avisos.Token()
			// turno_id es único: si otra pasada llegó primero, no hace nada.
			tag, err := tx.Exec(ctx, `INSERT INTO resena_tokens (token, negocio_id, turno_id, vence_en) VALUES ($1, app.negocio_actual(), $2, $3)
				ON CONFLICT (turno_id) DO NOTHING`, token, id, ahora.Add(vigencia))
			if err != nil {
				return err
			}
			if tag.RowsAffected() == 0 {
				continue
			}
			if _, err := avisos.Encolar(ctx, tx, avisos.Aviso{Tipo: avisos.Resena, TurnoID: id, Clave: "resena:" + id,
				Datos: map[string]any{"token": token}}, ahora); err != nil {
				return err
			}
			pedidos++
		}
		return nil
	})
	return map[string]int{"pedidos": pedidos}, err
}

// Trabajo para el planificador del worker.
func (s *Servicio) Trabajo() trabajos.Trabajo {
	return trabajos.Trabajo{Nombre: "resenas", Cada: 30 * time.Minute, Correr: s.Pedir}
}

// Resena como la ve el panel.
type Resena struct {
	ID           string     `json:"id"`
	TurnoID      string     `json:"turnoId"`
	Nombre       string     `json:"nombre"`
	Servicio     string     `json:"servicio"`
	Profesional  string     `json:"profesional"`
	Puntaje      int        `json:"puntaje"`
	Comentario   string     `json:"comentario"`
	Respuesta    string     `json:"respuesta"`
	RespondidaEn *time.Time `json:"respondidaEn"`
	Visible      bool       `json:"visible"`
	CreadoEn     time.Time  `json:"creadoEn"`
}

const columnas = `id::text, turno_id::text, nombre, servicio, profesional, puntaje, comentario, respuesta, respondida_en, visible, creado_en`

// Resumen de las reseñas visibles del negocio.
type Resumen struct {
	Promedio float64 `json:"promedio"`
	Cantidad int     `json:"cantidad"`
}

// Promedio de las reseñas visibles del negocio fijado en la transacción (redondeado a un decimal;
// 0 si no hay). Lo puede usar cualquier dominio que arme la página pública.
func Promedio(ctx context.Context, tx pgx.Tx) (Resumen, error) {
	var r Resumen
	err := tx.QueryRow(ctx, "SELECT coalesce(round(avg(puntaje), 1), 0)::float8, count(*) FROM resenas WHERE visible").Scan(&r.Promedio, &r.Cantidad)
	return r, err
}

// Pedido de reseña como lo ve el cliente.
type Pedido struct {
	Negocio     string    `json:"negocio"`
	Slug        string    `json:"slug"`
	Servicio    string    `json:"servicio"`
	Profesional string    `json:"profesional"`
	Inicio      time.Time `json:"inicio"`
	Cliente     string    `json:"cliente"`
	Vencido     bool      `json:"vencido"`
	Resena      *Resena   `json:"resena"`
}

func (s *Servicio) conPedido(ctx context.Context, token string, fn func(tx pgx.Tx, p *Pedido, turnoID, clienteID string, vence time.Time) error) (Pedido, error) {
	var p Pedido
	negocio, err := avisos.NegocioPorToken(ctx, s.Pool.EnTx, "resena", token)
	if err != nil {
		return p, err
	}
	err = s.Pool.EnNegocio(ctx, negocio, func(tx pgx.Tx) error {
		var turnoID, clienteID string
		var vence time.Time
		if err := tx.QueryRow(ctx, `SELECT n.nombre, n.slug, t.servicio, t.profesional, t.inicio, c.nombre, t.id::text, c.id::text, r.vence_en
			FROM resena_tokens r JOIN turnos t ON t.id = r.turno_id JOIN clientes c ON c.id = t.cliente_id
			JOIN negocios n ON n.id = r.negocio_id WHERE r.token = $1 FOR UPDATE OF r`, token).
			Scan(&p.Negocio, &p.Slug, &p.Servicio, &p.Profesional, &p.Inicio, &p.Cliente, &turnoID, &clienteID, &vence); err != nil {
			return err
		}
		var r Resena
		err := tx.QueryRow(ctx, "SELECT "+columnas+" FROM resenas WHERE turno_id = $1", turnoID).Scan(
			&r.ID, &r.TurnoID, &r.Nombre, &r.Servicio, &r.Profesional, &r.Puntaje, &r.Comentario, &r.Respuesta, &r.RespondidaEn, &r.Visible, &r.CreadoEn)
		if err == nil {
			p.Resena = &r
		} else if !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		p.Vencido = !vence.After(s.ahora())
		return fn(tx, &p, turnoID, clienteID, vence)
	})
	p.Cliente = primerNombre(p.Cliente)
	return p, err
}

func (s *Servicio) verPedido(w http.ResponseWriter, r *http.Request) error {
	p, err := s.conPedido(r.Context(), r.PathValue("token"), func(pgx.Tx, *Pedido, string, string, time.Time) error { return nil })
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusOK, p)
	return nil
}

func (s *Servicio) dejar(w http.ResponseWriter, r *http.Request) error {
	var in struct {
		Puntaje    int    `json:"puntaje"`
		Comentario string `json:"comentario"`
	}
	if err := web.Leer(w, r, &in); err != nil {
		return err
	}
	in.Comentario = strings.TrimSpace(in.Comentario)
	if in.Puntaje < 1 || in.Puntaje > 5 {
		return problema.Invalido("Elegí de 1 a 5 estrellas.", problema.Campo{Ruta: "puntaje", Mensaje: "entre 1 y 5"})
	}
	if utf8.RuneCountInString(in.Comentario) > 1000 {
		return problema.Invalido("El comentario es muy largo (hasta 1000).", problema.Campo{Ruta: "comentario", Mensaje: "muy largo"})
	}
	ctx := r.Context()
	p, err := s.conPedido(ctx, r.PathValue("token"), func(tx pgx.Tx, p *Pedido, turnoID, clienteID string, _ time.Time) error {
		if p.Resena != nil {
			return problema.Conflicto("ya_dejada", "Ya dejaste tu reseña de este turno. ¡Gracias!")
		}
		if p.Vencido {
			return problema.Conflicto("vencido", "Este link ya venció.")
		}
		var x Resena
		if err := tx.QueryRow(ctx, `INSERT INTO resenas (negocio_id, turno_id, cliente_id, nombre, servicio, profesional, puntaje, comentario)
			VALUES (app.negocio_actual(), $1, $2, $3, $4, $5, $6, $7) RETURNING `+columnas,
			turnoID, clienteID, p.Cliente, p.Servicio, p.Profesional, in.Puntaje, in.Comentario).Scan(
			&x.ID, &x.TurnoID, &x.Nombre, &x.Servicio, &x.Profesional, &x.Puntaje, &x.Comentario, &x.Respuesta, &x.RespondidaEn, &x.Visible, &x.CreadoEn); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, "UPDATE resena_tokens SET usado_en = $2 WHERE token = $1", r.PathValue("token"), s.ahora()); err != nil {
			return err
		}
		p.Resena = &x
		texto := in.Comentario
		if utf8.RuneCountInString(texto) > 200 {
			texto = string([]rune(texto)[:200]) + "…"
		}
		return avisos.Notificar(ctx, tx, avisos.Notificacion{Tipo: avisos.NotiResena, TurnoID: turnoID,
			Titulo: "Nueva reseña de " + p.Cliente + ": " + strings.Repeat("★", in.Puntaje) + strings.Repeat("☆", 5-in.Puntaje), Texto: texto})
	})
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusCreated, p)
	return nil
}

// Publica: lo que muestra la página del negocio (el nombre, corto).
type Publica struct {
	Nombre     string    `json:"nombre"`
	Servicio   string    `json:"servicio"`
	Puntaje    int       `json:"puntaje"`
	Comentario string    `json:"comentario"`
	Respuesta  string    `json:"respuesta"`
	CreadoEn   time.Time `json:"creadoEn"`
}

func (s *Servicio) publicas(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	var id *string
	err := s.Pool.EnTx(ctx, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, "SELECT app.negocio_publico($1)::text", r.PathValue("slug")).Scan(&id)
	})
	if err != nil {
		return err
	}
	if id == nil {
		return problema.NoEncontrado("No encontramos ese negocio.")
	}
	pagina := 0
	if v, err := strconv.Atoi(r.URL.Query().Get("pagina")); err == nil && v > 0 {
		pagina = v
	}
	var out struct {
		Resumen
		Resenas []Publica `json:"resenas"`
	}
	out.Resenas = []Publica{}
	err = s.Pool.EnNegocio(ctx, *id, func(tx pgx.Tx) error {
		var err error
		if out.Resumen, err = Promedio(ctx, tx); err != nil {
			return err
		}
		filas, err := tx.Query(ctx, `SELECT nombre, servicio, puntaje, comentario, respuesta, creado_en FROM resenas WHERE visible
			ORDER BY creado_en DESC LIMIT 20 OFFSET $1`, pagina*20)
		if err != nil {
			return err
		}
		out.Resenas, err = pgx.CollectRows(filas, pgx.RowToStructByPos[Publica])
		return err
	})
	if err != nil {
		return err
	}
	for i := range out.Resenas {
		out.Resenas[i].Nombre = nombreCorto(out.Resenas[i].Nombre)
	}
	web.JSON(w, http.StatusOK, out)
	return nil
}

func (s *Servicio) listar(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	q := r.URL.Query()
	cond := "true"
	switch q.Get("visible") {
	case "si":
		cond = "visible"
	case "no":
		cond = "NOT visible"
	}
	if q.Get("sinResponder") == "1" {
		cond += " AND respuesta = ''"
	}
	pagina := 0
	if v, err := strconv.Atoi(q.Get("pagina")); err == nil && v > 0 {
		pagina = v
	}
	var out struct {
		Resumen
		Resenas []Resena `json:"resenas"`
	}
	err := s.Pool.EnNegocio(ctx, sesion.De(ctx).NegocioID, func(tx pgx.Tx) error {
		var err error
		if out.Resumen, err = Promedio(ctx, tx); err != nil {
			return err
		}
		filas, err := tx.Query(ctx, "SELECT "+columnas+" FROM resenas WHERE "+cond+" ORDER BY creado_en DESC LIMIT 50 OFFSET $1", pagina*50)
		if err != nil {
			return err
		}
		out.Resenas, err = pgx.CollectRows(filas, pgx.RowToStructByPos[Resena])
		return err
	})
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusOK, out)
	return nil
}

// moderar: ocultar o volver a mostrar, y responder (la respuesta vacía la saca).
func (s *Servicio) moderar(w http.ResponseWriter, r *http.Request) error {
	var in struct {
		Visible   *bool   `json:"visible"`
		Respuesta *string `json:"respuesta"`
	}
	if err := web.Leer(w, r, &in); err != nil {
		return err
	}
	if in.Respuesta != nil {
		*in.Respuesta = strings.TrimSpace(*in.Respuesta)
		if utf8.RuneCountInString(*in.Respuesta) > 1000 {
			return problema.Invalido("La respuesta es muy larga (hasta 1000).")
		}
	}
	ctx := r.Context()
	var x Resena
	err := s.Pool.EnNegocio(ctx, sesion.De(ctx).NegocioID, func(tx pgx.Tx) error {
		var antes Resena
		if err := tx.QueryRow(ctx, "SELECT visible, respuesta FROM resenas WHERE id = $1 FOR UPDATE", r.PathValue("id")).
			Scan(&antes.Visible, &antes.Respuesta); err != nil {
			return err
		}
		if err := tx.QueryRow(ctx, `UPDATE resenas SET visible = coalesce($2, visible), respuesta = coalesce($3, respuesta),
			respondida_en = CASE WHEN $3::text IS NULL THEN respondida_en WHEN $3 = '' THEN NULL ELSE $4 END
			WHERE id = $1 RETURNING `+columnas, r.PathValue("id"), in.Visible, in.Respuesta, s.ahora()).Scan(
			&x.ID, &x.TurnoID, &x.Nombre, &x.Servicio, &x.Profesional, &x.Puntaje, &x.Comentario, &x.Respuesta, &x.RespondidaEn, &x.Visible, &x.CreadoEn); err != nil {
			return err
		}
		accion := "resena.respondida"
		if in.Visible != nil && *in.Visible != antes.Visible {
			accion = map[bool]string{true: "resena.mostrada", false: "resena.ocultada"}[*in.Visible]
		}
		return auditoria.Anotar(ctx, tx, auditoria.Registro{Accion: accion, Entidad: "resena", EntidadID: x.ID,
			Detalle: map[string]any{"visible": x.Visible, "respuesta": x.Respuesta}})
	})
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusOK, x)
	return nil
}

func primerNombre(nombre string) string {
	if f := strings.Fields(nombre); len(f) > 0 {
		return f[0]
	}
	return nombre
}

// nombreCorto: "Lucía Fernández" → "Lucía F." (la página pública no muestra apellidos enteros).
func nombreCorto(nombre string) string {
	f := strings.Fields(nombre)
	if len(f) < 2 {
		return nombre
	}
	inicial, _ := utf8.DecodeRuneInString(f[len(f)-1])
	return f[0] + " " + string(inicial) + "."
}
