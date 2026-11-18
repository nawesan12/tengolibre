package negocios

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/auditoria"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/problema"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/sesion"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/tope"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/web"
)

// TopeCambios: cuántas veces se puede cambiar el link cada 30 días (lo hace cumplir app.cambiar_slug).
const TopeCambios = 5

// Link del negocio como lo ve el panel: el de ahora, los anteriores que siguen redirigiendo y
// cuántos cambios le quedan este mes.
type Link struct {
	Slug       string     `json:"slug"`
	Anteriores []Anterior `json:"anteriores"`
	Restantes  int        `json:"restantes"`
}

// Anterior es un link que el negocio dejó y sigue llevando a su página.
type Anterior struct {
	Slug     string    `json:"slug"`
	DejadoEn time.Time `json:"dejadoEn"`
}

// Estado de un slug para quien lo pide: libre, propia (es su link), tomada o invalida.
type Estado struct {
	Estado string `json:"estado"`
}

func (s *Servicio) rutasLink(mux *http.ServeMux) {
	ses := s.Sesion
	mux.HandleFunc("GET /v1/slugs/{slug}", web.Envolver(s.estadoPublico))
	mux.HandleFunc("GET /v1/panel/negocio/link", ses.Panel(sesion.Agenda, web.Envolver(s.leerLink)))
	mux.HandleFunc("GET /v1/panel/negocio/link/{slug}", ses.Panel(sesion.Gestion, web.Envolver(s.estadoPanel)))
	mux.HandleFunc("PATCH /v1/panel/negocio/link", ses.Panel(sesion.Gestion, web.Envolver(s.cambiarLink)))
}

// estadoDe: el formato y los reservados los ve Go; lo que depende de los otros negocios, la base.
func estadoDe(ctx context.Context, tx pgx.Tx, slug string, negocio *string) (string, error) {
	if ValidarSlug(slug) != nil {
		return "invalida", nil
	}
	var e string
	err := tx.QueryRow(ctx, "SELECT app.estado_slug($1, $2::uuid)", slug, negocio).Scan(&e)
	return e, err
}

// estadoPublico lo usa el alta (antes de tener sesión) y el cierre de la landing. Un link anterior
// retenido de otro negocio también sale «tomada».
func (s *Servicio) estadoPublico(w http.ResponseWriter, r *http.Request) error {
	if !s.limite.Permitir("slugs:"+tope.IP(r), 120, time.Minute, time.Now()) {
		return tope.Demasiados("demasiados_pedidos", "Esperá un minuto y probá de nuevo.")
	}
	slug := strings.ToLower(r.PathValue("slug"))
	var e string
	err := s.Pool.EnTx(r.Context(), func(tx pgx.Tx) (err error) {
		e, err = estadoDe(r.Context(), tx, slug, nil)
		return err
	})
	if err != nil {
		return err
	}
	w.Header().Set("Cache-Control", "no-store")
	web.JSON(w, http.StatusOK, Estado{e})
	return nil
}

func (s *Servicio) estadoPanel(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	id := sesion.De(ctx).NegocioID
	slug := strings.ToLower(r.PathValue("slug"))
	var e string
	err := s.Pool.EnNegocio(ctx, id, func(tx pgx.Tx) (err error) {
		e, err = estadoDe(ctx, tx, slug, &id)
		return err
	})
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusOK, Estado{e})
	return nil
}

func leerLink(ctx context.Context, tx pgx.Tx) (Link, error) {
	l := Link{Anteriores: []Anterior{}}
	if err := tx.QueryRow(ctx, `SELECT slug, (SELECT count(*) FROM cambios_de_link
		WHERE negocio_id = app.negocio_actual() AND creado_en > now() - interval '30 days')
		FROM negocios WHERE id = app.negocio_actual()`).Scan(&l.Slug, &l.Restantes); err != nil {
		return l, err
	}
	l.Restantes = max(0, TopeCambios-l.Restantes)
	filas, err := tx.Query(ctx, "SELECT slug, dejado_en FROM slugs_anteriores ORDER BY dejado_en DESC")
	if err != nil {
		return l, err
	}
	as, err := pgx.CollectRows(filas, pgx.RowToStructByPos[Anterior])
	if err != nil {
		return l, err
	}
	l.Anteriores = append(l.Anteriores, as...)
	return l, nil
}

func (s *Servicio) leerLink(w http.ResponseWriter, r *http.Request) error {
	var l Link
	err := s.Pool.EnNegocio(r.Context(), sesion.De(r.Context()).NegocioID, func(tx pgx.Tx) (err error) {
		l, err = leerLink(r.Context(), tx)
		return err
	})
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusOK, l)
	return nil
}

// cambiarLink: el link viejo sigue andando (la web redirige con 301), así que los QR impresos no se
// pierden. Queda en la auditoría.
func (s *Servicio) cambiarLink(w http.ResponseWriter, r *http.Request) error {
	var in struct {
		Slug string `json:"slug"`
	}
	if err := web.Leer(w, r, &in); err != nil {
		return err
	}
	in.Slug = strings.ToLower(strings.TrimSpace(in.Slug))
	if err := ValidarSlug(in.Slug); err != nil {
		return err
	}
	ctx := r.Context()
	var l Link
	err := s.Pool.EnNegocio(ctx, sesion.De(ctx).NegocioID, func(tx pgx.Tx) error {
		var viejo, res string
		if err := tx.QueryRow(ctx, "SELECT slug FROM negocios WHERE id = app.negocio_actual()").Scan(&viejo); err != nil {
			return err
		}
		if err := tx.QueryRow(ctx, "SELECT app.cambiar_slug($1)", in.Slug).Scan(&res); err != nil {
			return err
		}
		switch res {
		case "tomada":
			return problema.Conflicto("slug_ocupado", "Esa dirección ya la usa otro negocio.")
		case "tope":
			return problema.Conflicto("tope_cambios", "Ya cambiaste el link 5 veces este mes. Probá más adelante.")
		case "ok":
			if err := auditoria.Anotar(ctx, tx, auditoria.Registro{Accion: "negocio.link_cambiado", Entidad: "negocio",
				EntidadID: sesion.De(ctx).NegocioID, Detalle: map[string]string{"de": viejo, "a": in.Slug}}); err != nil {
				return err
			}
		}
		var err error
		l, err = leerLink(ctx, tx)
		return err
	})
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusOK, l)
	return nil
}
