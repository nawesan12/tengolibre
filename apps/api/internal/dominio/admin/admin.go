// Package admin: la administración de TengoLibre (el equipo de la plataforma, PLATFORM_ADMIN en
// Jiku). Quién es admin sale de TENGOLIBRE_ADMINS (Clerk IDs), no de la base: así nadie se vuelve
// admin con un UPDATE.
//
// La RLS no deja leer entre negocios, así que todo pasa por funciones SECURITY DEFINER acotadas
// (app.admin_*, en 00006_plataforma.sql), que devuelven sólo agregados y columnas de listado; no hay
// un rol de base que lo vea todo.
package admin

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/disponibilidad"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/db"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/problema"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/sesion"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/web"
)

// Servicio de admin.
type Servicio struct {
	Pool   *db.Pool
	Sesion *sesion.Servicio
	// Admins: los Clerk IDs del equipo (TENGOLIBRE_ADMINS). Vacío: nadie.
	Admins []string
	// AdminsEmails: emails del equipo (TENGOLIBRE_ADMINS_EMAILS). Vale el email verificado de la
	// persona en Clerk, así alguien es admin desde su primer ingreso, sin conocer antes su ID.
	AdminsEmails []string
	// Ahora: el reloj (nil: time.Now).
	Ahora func() time.Time

	mu     sync.Mutex
	vistos map[string]visto
}

// visto: lo que dijo Clerk de una persona, para no preguntarle en cada pedido (limita por IP).
type visto struct {
	admin bool
	hasta time.Time
}

// esAdmin: por ID, o por email verificado (consultado a Clerk y guardado 10 minutos).
func (s *Servicio) esAdmin(ctx context.Context, clerkID string) bool {
	if slices.Contains(s.Admins, clerkID) {
		return true
	}
	if len(s.AdminsEmails) == 0 || s.Sesion.Verificador == nil {
		return false
	}
	ahora := s.ahora()
	s.mu.Lock()
	v, ok := s.vistos[clerkID]
	s.mu.Unlock()
	if ok && ahora.Before(v.hasta) {
		return v.admin
	}
	u, err := s.Sesion.Verificador.Usuario(ctx, clerkID)
	if err != nil {
		return false
	}
	// u.Email viene vacío si la persona no tiene ningún email verificado.
	es := u.Email != "" && slices.ContainsFunc(s.AdminsEmails, func(e string) bool { return strings.EqualFold(e, u.Email) })
	s.mu.Lock()
	if s.vistos == nil {
		s.vistos = map[string]visto{}
	}
	s.vistos[clerkID] = visto{admin: es, hasta: ahora.Add(10 * time.Minute)}
	s.mu.Unlock()
	return es
}

func (s *Servicio) ahora() time.Time {
	if s.Ahora == nil {
		return time.Now()
	}
	return s.Ahora()
}

// Rutas del dominio.
func (s *Servicio) Rutas(mux *http.ServeMux) {
	mux.HandleFunc("GET /v1/admin/estadisticas", s.admin(s.estadisticas))
	mux.HandleFunc("GET /v1/admin/negocios", s.admin(s.negocios))
	mux.HandleFunc("PATCH /v1/admin/negocios/{id}", s.admin(s.cambiar))
	mux.HandleFunc("POST /v1/admin/negocios/{id}/extender-prueba", s.admin(s.extender))
}

// admin exige sesión de una persona del equipo. A cualquier otra le responde 404, no 403: la
// administración no se anuncia.
func (s *Servicio) admin(h web.Manejador) http.HandlerFunc {
	return s.Sesion.Persona(web.Envolver(func(w http.ResponseWriter, r *http.Request) error {
		if !s.esAdmin(r.Context(), sesion.De(r.Context()).ClerkID) {
			return problema.NoEncontrado("Esa ruta no existe.")
		}
		return h(w, r)
	}))
}

// json crudo de una función de la base, tal cual.
func crudo(w http.ResponseWriter, b []byte) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(b)
}

func (s *Servicio) estadisticas(w http.ResponseWriter, r *http.Request) error {
	ahora := s.ahora().In(disponibilidad.ZonaPorDefecto)
	mes := time.Date(ahora.Year(), ahora.Month(), 1, 0, 0, 0, 0, ahora.Location())
	ctx := r.Context()
	var b []byte
	err := s.Pool.EnTx(ctx, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, "SELECT app.admin_estadisticas($1, $2)", mes, mes.AddDate(0, -1, 0)).Scan(&b)
	})
	if err != nil {
		return err
	}
	crudo(w, b)
	return nil
}

func (s *Servicio) negocios(w http.ResponseWriter, r *http.Request) error {
	q := r.URL.Query()
	texto := strings.TrimSpace(q.Get("q"))
	if len(texto) > 120 {
		return problema.Invalido("La búsqueda es demasiado larga.")
	}
	pagina, _ := strconv.Atoi(q.Get("pagina"))
	pagina = max(1, min(pagina, 10000))
	const limite = 25
	ctx := r.Context()
	var b []byte
	err := s.Pool.EnTx(ctx, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, "SELECT app.admin_negocios($1, $2, $3)", texto, limite, (pagina-1)*limite).Scan(&b)
	})
	if err != nil {
		return err
	}
	var out map[string]any
	if err := json.Unmarshal(b, &out); err != nil {
		return err
	}
	out["pagina"], out["limite"] = pagina, limite
	web.JSON(w, http.StatusOK, out)
	return nil
}

// cambiarNegocio llama a app.admin_cambiar_negocio y deja rastro en el log (quién y qué).
func (s *Servicio) cambiarNegocio(r *http.Request, activo *bool, prueba *time.Time) error {
	ctx := r.Context()
	id := r.PathValue("id")
	var hecho bool
	err := s.Pool.EnTx(ctx, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, "SELECT app.admin_cambiar_negocio($1, $2, $3)", id, activo, prueba).Scan(&hecho)
	})
	if err != nil {
		return err
	}
	if !hecho {
		return problema.NoEncontrado("No existe ese negocio.")
	}
	slog.Info("admin: negocio cambiado", "admin", sesion.De(ctx).ClerkID, "negocio", id, "activo", activo, "pruebaHasta", prueba)
	return nil
}

func (s *Servicio) cambiar(w http.ResponseWriter, r *http.Request) error {
	var in struct {
		Activo *bool `json:"activo"`
	}
	if err := web.Leer(w, r, &in); err != nil {
		return err
	}
	if in.Activo == nil {
		return problema.Invalido("Falta activo.", problema.Campo{Ruta: "activo", Mensaje: "true o false"})
	}
	if err := s.cambiarNegocio(r, in.Activo, nil); err != nil {
		return err
	}
	web.JSON(w, http.StatusOK, map[string]any{"id": r.PathValue("id"), "activo": *in.Activo})
	return nil
}

// extender suma días a la prueba, contados desde su fin o desde hoy si ya venció.
func (s *Servicio) extender(w http.ResponseWriter, r *http.Request) error {
	var in struct {
		Dias int `json:"dias"`
	}
	if err := web.Leer(w, r, &in); err != nil {
		return err
	}
	if in.Dias < 1 || in.Dias > 365 {
		return problema.Invalido("Los días van de 1 a 365.", problema.Campo{Ruta: "dias", Mensaje: "de 1 a 365"})
	}
	ctx := r.Context()
	var actual *time.Time
	err := s.Pool.EnNegocio(ctx, r.PathValue("id"), func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, "SELECT prueba_hasta FROM negocios WHERE id = app.negocio_actual()").Scan(&actual)
	})
	if err != nil {
		return err
	}
	base := s.ahora()
	if actual != nil && actual.After(base) {
		base = *actual
	}
	hasta := base.AddDate(0, 0, in.Dias)
	if err := s.cambiarNegocio(r, nil, &hasta); err != nil {
		return err
	}
	web.JSON(w, http.StatusOK, map[string]any{"id": r.PathValue("id"), "pruebaHasta": hasta})
	return nil
}
