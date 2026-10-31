// Package negocios: el alta de un negocio, sus datos públicos, los ajustes de reserva y el equipo
// que entra al panel.
package negocios

import (
	"context"
	"net/http"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"

	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/db"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/problema"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/sesion"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/web"
)

// Servicio de negocios.
type Servicio struct {
	Pool   *db.Pool
	Sesion *sesion.Servicio
}

// Negocio son los datos del negocio que ve el panel y, salvo el id, la página pública.
type Negocio struct {
	ID          string  `json:"id"`
	Slug        string  `json:"slug"`
	Nombre      string  `json:"nombre"`
	Descripcion string  `json:"descripcion"`
	Direccion   string  `json:"direccion"`
	Ciudad      string  `json:"ciudad"`
	Telefono    string  `json:"telefono"`
	Whatsapp    string  `json:"whatsapp"`
	Instagram   string  `json:"instagram"`
	Logo        *string `json:"logo"`
	ZonaHoraria string  `json:"zonaHoraria"`
}

// Ajustes de reserva.
type Ajustes struct {
	Intervalo           int    `json:"intervalo"`
	AnticipacionMin     int    `json:"anticipacionMin"`
	DiasVista           int    `json:"diasVista"`
	MargenMin           int    `json:"margenMin"`
	CancelarHastaHoras  int    `json:"cancelarHastaHoras"`
	PoliticaCancelacion string `json:"politicaCancelacion"`
}

// Miembro del equipo.
type Miembro struct {
	ID      string    `json:"id"`
	ClerkID string    `json:"clerkId"`
	Email   string    `json:"email"`
	Nombre  string    `json:"nombre"`
	Rol     string    `json:"rol"`
	Activo  bool      `json:"activo"`
	Creado  time.Time `json:"creadoEn"`
}

// Rutas del dominio.
func (s *Servicio) Rutas(mux *http.ServeMux) {
	ses := s.Sesion
	mux.HandleFunc("GET /v1/yo", ses.Persona(web.Envolver(s.yo)))
	mux.HandleFunc("POST /v1/negocios", ses.Persona(web.Envolver(s.crear)))
	mux.HandleFunc("GET /v1/panel/negocio", ses.Panel(sesion.Agenda, web.Envolver(s.leer)))
	mux.HandleFunc("PATCH /v1/panel/negocio", ses.Panel(sesion.Gestion, web.Envolver(s.editar)))
	mux.HandleFunc("GET /v1/panel/ajustes", ses.Panel(sesion.Agenda, web.Envolver(s.leerAjustes)))
	mux.HandleFunc("PUT /v1/panel/ajustes", ses.Panel(sesion.Gestion, web.Envolver(s.guardarAjustes)))
	mux.HandleFunc("GET /v1/panel/equipo", ses.Panel(sesion.Gestion, web.Envolver(s.equipo)))
	mux.HandleFunc("POST /v1/panel/equipo", ses.Panel([]string{sesion.Duenio}, web.Envolver(s.sumar)))
	mux.HandleFunc("PATCH /v1/panel/equipo/{id}", ses.Panel([]string{sesion.Duenio}, web.Envolver(s.cambiarMiembro)))
}

func (s *Servicio) yo(w http.ResponseWriter, r *http.Request) error {
	ns, err := s.Sesion.Negocios(r.Context(), sesion.De(r.Context()).ClerkID)
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusOK, map[string]any{"negocios": ns})
	return nil
}

var reSlug = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,46}[a-z0-9])?$`)

// Reservados: la página de un negocio vive en la raíz (/{slug}) y comparte espacio con las rutas
// del sitio (reserved-slugs.ts de Jiku).
var Reservados = map[string]bool{
	"admin": true, "api": true, "explorar": true, "iniciar-sesion": true, "invitacion": true, "mi-cuenta": true,
	"mis-turnos": true, "panel": true, "presentacion": true, "recuperar-contrasena": true, "registrarse": true,
	"review": true, "resena": true, "sin-conexion": true, "turno": true, "ingresar": true, "precios": true,
	"ayuda": true, "legal": true, "terminos": true, "privacidad": true, "www": true,
}

// ValidarSlug devuelve el problema si el slug no sirve.
func ValidarSlug(slug string) error {
	if !reSlug.MatchString(slug) || strings.Contains(slug, "--") {
		return problema.Invalido("La dirección va en minúsculas, números y guiones (hasta 48).",
			problema.Campo{Ruta: "slug", Mensaje: "formato inválido"})
	}
	if Reservados[slug] {
		return problema.Conflicto("slug_reservado", "Esa dirección no está disponible.")
	}
	return nil
}

func largo(campo, v string, min, max int) error {
	if n := utf8.RuneCountInString(v); n < min || n > max {
		return problema.Invalido("Revisá "+campo+".", problema.Campo{Ruta: campo, Mensaje: "largo inválido"})
	}
	return nil
}

func (s *Servicio) crear(w http.ResponseWriter, r *http.Request) error {
	var in struct {
		Slug   string `json:"slug"`
		Nombre string `json:"nombre"`
	}
	if err := web.Leer(w, r, &in); err != nil {
		return err
	}
	in.Slug, in.Nombre = strings.ToLower(strings.TrimSpace(in.Slug)), strings.TrimSpace(in.Nombre)
	if err := ValidarSlug(in.Slug); err != nil {
		return err
	}
	if err := largo("nombre", in.Nombre, 1, 80); err != nil {
		return err
	}
	ctx := r.Context()
	clerkID := sesion.De(ctx).ClerkID
	u, err := s.Sesion.Verificador.Usuario(ctx, clerkID)
	if err != nil {
		return err
	}
	persona := strings.TrimSpace(u.Nombre + " " + u.Apellido)
	var n Negocio
	err = s.Pool.EnTx(ctx, func(tx pgx.Tx) error {
		var ocupado bool
		if err := tx.QueryRow(ctx, "SELECT app.negocio_publico($1) IS NOT NULL", in.Slug).Scan(&ocupado); err != nil {
			return err
		}
		if ocupado {
			return problema.Conflicto("slug_ocupado", "Esa dirección ya la usa otro negocio.")
		}
		if err := tx.QueryRow(ctx, "SELECT app.crear_negocio($1, $2, $3, $4, $5)", in.Slug, in.Nombre, clerkID, u.Email, persona).Scan(&n.ID); err != nil {
			return err
		}
		return leerNegocio(ctx, tx, &n)
	})
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusCreated, n)
	return nil
}

const columnasNegocio = "id::text, slug, nombre, descripcion, direccion, ciudad, telefono, whatsapp, instagram, logo, zona_horaria"

func leerNegocio(ctx context.Context, tx pgx.Tx, n *Negocio) error {
	return tx.QueryRow(ctx, "SELECT "+columnasNegocio+" FROM negocios WHERE id = app.negocio_actual()").Scan(
		&n.ID, &n.Slug, &n.Nombre, &n.Descripcion, &n.Direccion, &n.Ciudad, &n.Telefono, &n.Whatsapp, &n.Instagram, &n.Logo, &n.ZonaHoraria)
}

// Leer el negocio fijado en la transacción (lo usa también la página pública).
func Leer(ctx context.Context, tx pgx.Tx) (Negocio, error) {
	var n Negocio
	err := leerNegocio(ctx, tx, &n)
	return n, err
}

func (s *Servicio) leer(w http.ResponseWriter, r *http.Request) error {
	var n Negocio
	err := s.Pool.EnNegocio(r.Context(), sesion.De(r.Context()).NegocioID, func(tx pgx.Tx) error {
		return leerNegocio(r.Context(), tx, &n)
	})
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusOK, n)
	return nil
}

func (s *Servicio) editar(w http.ResponseWriter, r *http.Request) error {
	var in struct {
		Nombre      *string `json:"nombre"`
		Descripcion *string `json:"descripcion"`
		Direccion   *string `json:"direccion"`
		Ciudad      *string `json:"ciudad"`
		Telefono    *string `json:"telefono"`
		Whatsapp    *string `json:"whatsapp"`
		Instagram   *string `json:"instagram"`
		Logo        *string `json:"logo"`
	}
	if err := web.Leer(w, r, &in); err != nil {
		return err
	}
	if in.Nombre != nil {
		*in.Nombre = strings.TrimSpace(*in.Nombre)
		if err := largo("nombre", *in.Nombre, 1, 80); err != nil {
			return err
		}
	}
	if in.Logo != nil && *in.Logo == "" {
		in.Logo = nil
	}
	ctx := r.Context()
	var n Negocio
	err := s.Pool.EnNegocio(ctx, sesion.De(ctx).NegocioID, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `UPDATE negocios SET
			nombre = coalesce($1, nombre), descripcion = coalesce($2, descripcion), direccion = coalesce($3, direccion),
			ciudad = coalesce($4, ciudad), telefono = coalesce($5, telefono), whatsapp = coalesce($6, whatsapp),
			instagram = coalesce($7, instagram), logo = coalesce($8, logo)
			WHERE id = app.negocio_actual()`,
			in.Nombre, in.Descripcion, in.Direccion, in.Ciudad, in.Telefono, in.Whatsapp, in.Instagram, in.Logo)
		if err != nil {
			return err
		}
		return leerNegocio(ctx, tx, &n)
	})
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusOK, n)
	return nil
}

// LeerAjustes del negocio fijado (los usa la agenda).
func LeerAjustes(ctx context.Context, tx pgx.Tx) (Ajustes, error) {
	var a Ajustes
	err := tx.QueryRow(ctx, `SELECT intervalo, anticipacion_min, dias_vista, margen_min, cancelar_hasta_horas, politica_cancelacion
		FROM ajustes WHERE negocio_id = app.negocio_actual()`).Scan(
		&a.Intervalo, &a.AnticipacionMin, &a.DiasVista, &a.MargenMin, &a.CancelarHastaHoras, &a.PoliticaCancelacion)
	return a, err
}

func (s *Servicio) leerAjustes(w http.ResponseWriter, r *http.Request) error {
	var a Ajustes
	err := s.Pool.EnNegocio(r.Context(), sesion.De(r.Context()).NegocioID, func(tx pgx.Tx) (err error) {
		a, err = LeerAjustes(r.Context(), tx)
		return err
	})
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusOK, a)
	return nil
}

func (s *Servicio) guardarAjustes(w http.ResponseWriter, r *http.Request) error {
	var a Ajustes
	if err := web.Leer(w, r, &a); err != nil {
		return err
	}
	ctx := r.Context()
	err := s.Pool.EnNegocio(ctx, sesion.De(ctx).NegocioID, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `UPDATE ajustes SET intervalo = $1, anticipacion_min = $2, dias_vista = $3, margen_min = $4,
			cancelar_hasta_horas = $5, politica_cancelacion = $6, actualizado_en = now() WHERE negocio_id = app.negocio_actual()`,
			a.Intervalo, a.AnticipacionMin, a.DiasVista, a.MargenMin, a.CancelarHastaHoras, a.PoliticaCancelacion)
		if err != nil {
			return err
		}
		a, err = LeerAjustes(ctx, tx)
		return err
	})
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusOK, a)
	return nil
}

const columnasMiembro = "id::text, clerk_id, email, nombre, rol, activo, creado_en"

func (s *Servicio) equipo(w http.ResponseWriter, r *http.Request) error {
	var ms []Miembro
	err := s.Pool.EnNegocio(r.Context(), sesion.De(r.Context()).NegocioID, func(tx pgx.Tx) error {
		filas, err := tx.Query(r.Context(), "SELECT "+columnasMiembro+" FROM miembros ORDER BY creado_en")
		if err != nil {
			return err
		}
		ms, err = pgx.CollectRows(filas, pgx.RowToStructByPos[Miembro])
		return err
	})
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusOK, map[string]any{"miembros": ms})
	return nil
}

var rolesValidos = map[string]bool{sesion.Duenio: true, sesion.Encargado: true, sesion.Recepcion: true, sesion.Profesional: true}

// sumar agrega a alguien que ya tiene cuenta en Clerk (el front lo busca por email con Clerk y manda
// su id). La invitación por mail a quien todavía no tiene cuenta va por el dominio de avisos.
func (s *Servicio) sumar(w http.ResponseWriter, r *http.Request) error {
	var in struct {
		ClerkID string `json:"clerkId"`
		Rol     string `json:"rol"`
	}
	if err := web.Leer(w, r, &in); err != nil {
		return err
	}
	if !rolesValidos[in.Rol] {
		return problema.Invalido("Rol desconocido.", problema.Campo{Ruta: "rol", Mensaje: "duenio, encargado, recepcion o profesional"})
	}
	ctx := r.Context()
	u, err := s.Sesion.Verificador.Usuario(ctx, in.ClerkID)
	if err != nil {
		return problema.NoEncontrado("No encontramos esa cuenta.")
	}
	var m Miembro
	err = s.Pool.EnNegocio(ctx, sesion.De(ctx).NegocioID, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `INSERT INTO miembros (negocio_id, clerk_id, email, nombre, rol) VALUES (app.negocio_actual(), $1, $2, $3, $4)
			RETURNING `+columnasMiembro, u.ID, u.Email, strings.TrimSpace(u.Nombre+" "+u.Apellido), in.Rol).Scan(
			&m.ID, &m.ClerkID, &m.Email, &m.Nombre, &m.Rol, &m.Activo, &m.Creado)
	})
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusCreated, m)
	return nil
}

func (s *Servicio) cambiarMiembro(w http.ResponseWriter, r *http.Request) error {
	var in struct {
		Rol    *string `json:"rol"`
		Activo *bool   `json:"activo"`
	}
	if err := web.Leer(w, r, &in); err != nil {
		return err
	}
	if in.Rol != nil && !rolesValidos[*in.Rol] {
		return problema.Invalido("Rol desconocido.")
	}
	ctx := r.Context()
	var m Miembro
	err := s.Pool.EnNegocio(ctx, sesion.De(ctx).NegocioID, func(tx pgx.Tx) error {
		if err := tx.QueryRow(ctx, `UPDATE miembros SET rol = coalesce($2, rol), activo = coalesce($3, activo) WHERE id = $1
			RETURNING `+columnasMiembro, r.PathValue("id"), in.Rol, in.Activo).Scan(
			&m.ID, &m.ClerkID, &m.Email, &m.Nombre, &m.Rol, &m.Activo, &m.Creado); err != nil {
			return err
		}
		// Siempre queda al menos una dueña activa.
		var duenias int
		if err := tx.QueryRow(ctx, "SELECT count(*) FROM miembros WHERE rol = 'duenio' AND activo").Scan(&duenias); err != nil {
			return err
		}
		if duenias == 0 {
			return problema.Conflicto("sin_duenio", "El negocio tiene que quedar con al menos una persona dueña.")
		}
		return nil
	})
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusOK, m)
	return nil
}
