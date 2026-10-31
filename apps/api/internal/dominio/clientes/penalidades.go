package clientes

import (
	"context"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"

	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/problema"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/web"
)

// Penalidad de un cliente. Las automáticas (motivo "ausencias") las crea la base cuando un turno
// pasa a ausente y el cliente llega al umbral; las manuales, alguien de gestión.
type Penalidad struct {
	ID          string     `json:"id"`
	ClienteID   string     `json:"clienteId"`
	Cliente     string     `json:"cliente"`
	Telefono    string     `json:"telefono"`
	Motivo      string     `json:"motivo"`
	Nota        string     `json:"nota"`
	Ausencias   int        `json:"ausencias"`
	Hasta       time.Time  `json:"hasta"`
	CreadoEn    time.Time  `json:"creadoEn"`
	LevantadaEn *time.Time `json:"levantadaEn"`
	Vigente     bool       `json:"vigente"`
}

const columnasPenalidad = `p.id::text, p.cliente_id::text, c.nombre, c.telefono, p.motivo, p.nota, p.ausencias, p.hasta,
	p.creado_en, p.levantada_en, (p.levantada_en IS NULL AND p.hasta > now())`

func leerPenalidades(ctx context.Context, tx pgx.Tx, donde string, args ...any) ([]Penalidad, error) {
	filas, err := tx.Query(ctx, "SELECT "+columnasPenalidad+" FROM penalidades p JOIN clientes c ON c.id = p.cliente_id WHERE "+donde, args...)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(filas, pgx.RowToStructByPos[Penalidad])
}

// penalidades del negocio: las vigentes, o todas con ?todas=1 (paginado).
func (s *Servicio) penalidades(w http.ResponseWriter, r *http.Request) error {
	pagina, porPagina := paginado(r, 50, 200)
	donde := "p.levantada_en IS NULL AND p.hasta > now()"
	if r.URL.Query().Get("todas") == "1" {
		donde = "true"
	}
	out := struct {
		Penalidades []Penalidad `json:"penalidades"`
		Total       int         `json:"total"`
		Pagina      int         `json:"pagina"`
		PorPagina   int         `json:"porPagina"`
	}{Pagina: pagina, PorPagina: porPagina}
	err := s.enNegocio(r, func(ctx context.Context, tx pgx.Tx) error {
		if err := tx.QueryRow(ctx, "SELECT count(*) FROM penalidades p WHERE "+donde).Scan(&out.Total); err != nil {
			return err
		}
		var err error
		out.Penalidades, err = leerPenalidades(ctx, tx, donde+" ORDER BY p.creado_en DESC, p.id LIMIT $1 OFFSET $2", porPagina, (pagina-1)*porPagina)
		return err
	})
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusOK, out)
	return nil
}

func (s *Servicio) penalidadesDe(w http.ResponseWriter, r *http.Request) error {
	var out []Penalidad
	err := s.enNegocio(r, func(ctx context.Context, tx pgx.Tx) error {
		if err := existe(ctx, tx, r.PathValue("id")); err != nil {
			return err
		}
		var err error
		out, err = leerPenalidades(ctx, tx, "p.cliente_id = $1 ORDER BY p.creado_en DESC, p.id", r.PathValue("id"))
		return err
	})
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusOK, map[string]any{"penalidades": out})
	return nil
}

// penalizar a mano: por dias (por defecto, los de los ajustes). Si ya tiene una vigente, 409.
func (s *Servicio) penalizar(w http.ResponseWriter, r *http.Request) error {
	var in struct {
		Dias int    `json:"dias"`
		Nota string `json:"nota"`
	}
	if err := web.Leer(w, r, &in); err != nil {
		return err
	}
	in.Nota = strings.TrimSpace(in.Nota)
	if utf8.RuneCountInString(in.Nota) > 500 {
		return problema.Invalido("La nota es muy larga (hasta 500).")
	}
	if in.Dias < 0 || in.Dias > 365 {
		return problema.Invalido("La penalidad va de 1 a 365 días.", problema.Campo{Ruta: "dias", Mensaje: "fuera de rango"})
	}
	var p Penalidad
	err := s.enNegocio(r, func(ctx context.Context, tx pgx.Tx) error {
		id := r.PathValue("id")
		if err := existe(ctx, tx, id); err != nil {
			return err
		}
		// El cliente bloqueado: dos penalidades a la vez no se cruzan.
		if _, err := tx.Exec(ctx, "SELECT 1 FROM clientes WHERE id = $1 FOR UPDATE", id); err != nil {
			return err
		}
		ya, err := Penalizado(ctx, tx, id)
		if err != nil {
			return err
		}
		if ya {
			return problema.Conflicto("ya_penalizado", "Este cliente ya tiene una penalidad vigente.")
		}
		if in.Dias == 0 {
			if err := tx.QueryRow(ctx, "SELECT dias_penalidad FROM ajustes WHERE negocio_id = app.negocio_actual()").Scan(&in.Dias); err != nil {
				return err
			}
		}
		miembro, _, err := miembroDeLaSesion(ctx, tx)
		if err != nil {
			return err
		}
		var nueva string
		if err := tx.QueryRow(ctx, `INSERT INTO penalidades (negocio_id, cliente_id, motivo, nota, hasta, creado_por)
			VALUES (app.negocio_actual(), $1, 'manual', $2, now() + make_interval(days => $3), $4) RETURNING id::text`,
			id, in.Nota, in.Dias, miembro).Scan(&nueva); err != nil {
			return err
		}
		ps, err := leerPenalidades(ctx, tx, "p.id = $1", nueva)
		if err != nil {
			return err
		}
		p = ps[0]
		return nil
	})
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusCreated, p)
	return nil
}

// levantar una penalidad vigente antes de tiempo.
func (s *Servicio) levantar(w http.ResponseWriter, r *http.Request) error {
	var p Penalidad
	err := s.enNegocio(r, func(ctx context.Context, tx pgx.Tx) error {
		ps, err := leerPenalidades(ctx, tx, "p.id = $1 FOR UPDATE OF p", r.PathValue("id"))
		if err != nil {
			return err
		}
		if len(ps) == 0 {
			return problema.NoEncontrado("Esa penalidad no existe o no es de este negocio.")
		}
		if !ps[0].Vigente {
			return problema.Conflicto("no_vigente", "Esa penalidad ya se levantó o venció.")
		}
		miembro, _, err := miembroDeLaSesion(ctx, tx)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, "UPDATE penalidades SET levantada_en = now(), levantada_por = $2 WHERE id = $1", ps[0].ID, miembro); err != nil {
			return err
		}
		ps, err = leerPenalidades(ctx, tx, "p.id = $1", ps[0].ID)
		if err != nil {
			return err
		}
		p = ps[0]
		return nil
	})
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusOK, p)
	return nil
}

// ---- Ausencias

// AjustesAusencias del negocio.
type AjustesAusencias struct {
	// UmbralAusencias en 90 días que penaliza solo. 0: no se penaliza solo.
	UmbralAusencias int  `json:"umbralAusencias"`
	DiasPenalidad   int  `json:"diasPenalidad"`
	MarcarAusentes  bool `json:"marcarAusentes"`
}

func leerAjustes(ctx context.Context, tx pgx.Tx) (AjustesAusencias, error) {
	var a AjustesAusencias
	err := tx.QueryRow(ctx, "SELECT umbral_ausencias, dias_penalidad, marcar_ausentes FROM ajustes WHERE negocio_id = app.negocio_actual()").
		Scan(&a.UmbralAusencias, &a.DiasPenalidad, &a.MarcarAusentes)
	return a, err
}

func (s *Servicio) leerAjustes(w http.ResponseWriter, r *http.Request) error {
	var a AjustesAusencias
	err := s.enNegocio(r, func(ctx context.Context, tx pgx.Tx) (err error) {
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
	var a AjustesAusencias
	if err := web.Leer(w, r, &a); err != nil {
		return err
	}
	if a.UmbralAusencias < 0 || a.UmbralAusencias > 20 {
		return problema.Invalido("El umbral va de 0 (apagado) a 20 ausencias.", problema.Campo{Ruta: "umbralAusencias", Mensaje: "fuera de rango"})
	}
	if a.DiasPenalidad < 1 || a.DiasPenalidad > 365 {
		return problema.Invalido("La penalidad va de 1 a 365 días.", problema.Campo{Ruta: "diasPenalidad", Mensaje: "fuera de rango"})
	}
	err := s.enNegocio(r, func(ctx context.Context, tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `UPDATE ajustes SET umbral_ausencias = $1, dias_penalidad = $2, marcar_ausentes = $3, actualizado_en = now()
			WHERE negocio_id = app.negocio_actual()`, a.UmbralAusencias, a.DiasPenalidad, a.MarcarAusentes); err != nil {
			return err
		}
		var err error
		a, err = leerAjustes(ctx, tx)
		return err
	})
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusOK, a)
	return nil
}

// marcar ausentes ahora (lo mismo que hacen el listado y los reportes al pasar).
func (s *Servicio) marcar(w http.ResponseWriter, r *http.Request) error {
	var n int64
	err := s.enNegocio(r, func(ctx context.Context, tx pgx.Tx) (err error) {
		n, err = MarcarAusentes(ctx, tx, s.ahora())
		return err
	})
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusOK, map[string]int64{"marcados": n})
	return nil
}

// ausencias: el resumen de Jiku (no-shows): total, los últimos 30 días, penalidades vigentes y
// clientes con dos o más ausencias en 90 días.
func (s *Servicio) ausencias(w http.ResponseWriter, r *http.Request) error {
	var out struct {
		Total       int `json:"total"`
		Ultimos30   int `json:"ultimos30"`
		Vigentes    int `json:"penalidadesVigentes"`
		Reincidente int `json:"reincidentes"`
	}
	ahora := s.ahora()
	err := s.enNegocio(r, func(ctx context.Context, tx pgx.Tx) error {
		return tx.QueryRow(ctx, `SELECT
			(SELECT count(*) FROM turnos WHERE estado = 'ausente'),
			(SELECT count(*) FROM turnos WHERE estado = 'ausente' AND inicio >= $1 AND inicio <= $3),
			(SELECT count(*) FROM penalidades WHERE levantada_en IS NULL AND hasta > now()),
			(SELECT count(*) FROM (SELECT cliente_id FROM turnos WHERE estado = 'ausente' AND inicio >= $2 AND inicio <= $3
				GROUP BY cliente_id HAVING count(*) >= 2) x)`,
			ahora.AddDate(0, 0, -30), ahora.AddDate(0, 0, -90), ahora).Scan(&out.Total, &out.Ultimos30, &out.Vigentes, &out.Reincidente)
	})
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusOK, out)
	return nil
}
