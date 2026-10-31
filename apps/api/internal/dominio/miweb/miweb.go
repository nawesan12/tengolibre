// Package miweb: lo que la página pública del negocio muestra además de los datos básicos (Mi web
// en Jiku): colores, el texto largo, la portada, redes, el mapa, el rubro y la galería de fotos.
// Los datos básicos (nombre, descripción, dirección, logo…) siguen en negocios.
package miweb

import (
	"context"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"

	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/db"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/problema"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/sesion"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/web"
)

// MaxFotos de la galería.
const MaxFotos = 30

// Rubros del directorio. El orden es el de los filtros de explorar.
var Rubros = []string{"barberia", "peluqueria", "estetica", "unias", "cejas-pestanias", "maquillaje", "depilacion",
	"masajes", "spa", "tatuajes", "salud", "entrenamiento", "otro"}

// Web son los campos de la página que no están en negocios.Negocio.
type Web struct {
	Rubro         string   `json:"rubro"`
	Provincia     string   `json:"provincia"`
	Acerca        string   `json:"acerca"`
	Portada       *string  `json:"portada"`
	ColorPrimario string   `json:"colorPrimario"`
	ColorAcento   string   `json:"colorAcento"`
	Email         string   `json:"email"`
	SitioWeb      string   `json:"sitioWeb"`
	Facebook      string   `json:"facebook"`
	Tiktok        string   `json:"tiktok"`
	Latitud       *float64 `json:"latitud"`
	Longitud      *float64 `json:"longitud"`
}

// Foto de la galería.
type Foto struct {
	ID       string `json:"id"`
	URL      string `json:"url"`
	Epigrafe string `json:"epigrafe"`
	Orden    int    `json:"orden"`
}

// Servicio de Mi web.
type Servicio struct {
	Pool   *db.Pool
	Sesion *sesion.Servicio
}

// Rutas del dominio.
func (s *Servicio) Rutas(mux *http.ServeMux) {
	g := func(h web.Manejador) http.HandlerFunc { return s.Sesion.Panel(sesion.Gestion, web.Envolver(h)) }
	mux.HandleFunc("GET /v1/panel/mi-web", g(s.leer))
	mux.HandleFunc("PATCH /v1/panel/mi-web", g(s.editar))
	mux.HandleFunc("GET /v1/panel/mi-web/galeria", g(s.galeria))
	mux.HandleFunc("POST /v1/panel/mi-web/galeria", g(s.sumarFoto))
	mux.HandleFunc("PUT /v1/panel/mi-web/galeria/orden", g(s.ordenar))
	mux.HandleFunc("PATCH /v1/panel/mi-web/galeria/{id}", g(s.editarFoto))
	mux.HandleFunc("DELETE /v1/panel/mi-web/galeria/{id}", g(s.borrarFoto))
	// Lo mismo para la página pública, hasta que GET /v1/p/{slug} lo traiga junto.
	mux.HandleFunc("GET /v1/p/{slug}/web", web.Envolver(s.publico))
}

// Leer la web del negocio fijado en la transacción (lo usa la página pública).
func Leer(ctx context.Context, tx pgx.Tx) (Web, error) {
	var w Web
	err := tx.QueryRow(ctx, `SELECT rubro, provincia, acerca, portada, color_primario, color_acento, email, sitio_web,
		facebook, tiktok, latitud, longitud FROM negocios WHERE id = app.negocio_actual()`).Scan(
		&w.Rubro, &w.Provincia, &w.Acerca, &w.Portada, &w.ColorPrimario, &w.ColorAcento, &w.Email, &w.SitioWeb,
		&w.Facebook, &w.Tiktok, &w.Latitud, &w.Longitud)
	return w, err
}

// Galeria del negocio fijado, en orden.
func Galeria(ctx context.Context, tx pgx.Tx) ([]Foto, error) {
	filas, err := tx.Query(ctx, "SELECT id::text, url, epigrafe, orden FROM fotos ORDER BY orden, creado_en")
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(filas, pgx.RowToStructByPos[Foto])
}

func (s *Servicio) enNegocio(r *http.Request, fn func(context.Context, pgx.Tx) error) error {
	ctx := r.Context()
	return s.Pool.EnNegocio(ctx, sesion.De(ctx).NegocioID, func(tx pgx.Tx) error { return fn(ctx, tx) })
}

func (s *Servicio) leer(w http.ResponseWriter, r *http.Request) error {
	var out Web
	err := s.enNegocio(r, func(ctx context.Context, tx pgx.Tx) (err error) {
		out, err = Leer(ctx, tx)
		return err
	})
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusOK, out)
	return nil
}

var reColor = regexp.MustCompile(`^#[0-9a-f]{6}$`)

func campo(ruta, mensaje string) error {
	return problema.Invalido("Revisá "+ruta+".", problema.Campo{Ruta: ruta, Mensaje: mensaje})
}

func largo(ruta string, v *string, max int) error {
	if v == nil {
		return nil
	}
	*v = strings.TrimSpace(*v)
	if utf8.RuneCountInString(*v) > max {
		return campo(ruta, "demasiado largo")
	}
	return nil
}

// URLWeb valida un link que se va a mostrar en la página: http(s) y nada más (un javascript: en
// un href es un XSS).
func URLWeb(ruta, v string, max int) error {
	if v == "" {
		return nil
	}
	u, err := url.Parse(v)
	if err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Host == "" || len(v) > max {
		return campo(ruta, "tiene que ser un link http(s)")
	}
	return nil
}

func (s *Servicio) editar(w http.ResponseWriter, r *http.Request) error {
	var in struct {
		Rubro         *string  `json:"rubro"`
		Provincia     *string  `json:"provincia"`
		Acerca        *string  `json:"acerca"`
		Portada       *string  `json:"portada"`
		ColorPrimario *string  `json:"colorPrimario"`
		ColorAcento   *string  `json:"colorAcento"`
		Email         *string  `json:"email"`
		SitioWeb      *string  `json:"sitioWeb"`
		Facebook      *string  `json:"facebook"`
		Tiktok        *string  `json:"tiktok"`
		Latitud       *float64 `json:"latitud"`
		Longitud      *float64 `json:"longitud"`
		// SinUbicacion borra las coordenadas (null en el JSON no se distingue de "no vino").
		SinUbicacion bool `json:"sinUbicacion"`
	}
	if err := web.Leer(w, r, &in); err != nil {
		return err
	}
	for _, c := range []struct {
		ruta string
		v    *string
		max  int
	}{{"provincia", in.Provincia, 80}, {"acerca", in.Acerca, 5000}, {"email", in.Email, 120}, {"sitioWeb", in.SitioWeb, 200},
		{"facebook", in.Facebook, 100}, {"tiktok", in.Tiktok, 60}, {"portada", in.Portada, 2048}} {
		if err := largo(c.ruta, c.v, c.max); err != nil {
			return err
		}
	}
	if in.Rubro != nil && *in.Rubro != "" && !slices.Contains(Rubros, *in.Rubro) {
		return campo("rubro", "rubro desconocido")
	}
	for ruta, c := range map[string]*string{"colorPrimario": in.ColorPrimario, "colorAcento": in.ColorAcento} {
		if c != nil {
			*c = strings.ToLower(strings.TrimSpace(*c))
			if !reColor.MatchString(*c) {
				return campo(ruta, "un color como #1a2b3c")
			}
		}
	}
	if in.Email != nil && *in.Email != "" && !strings.Contains(*in.Email, "@") {
		return campo("email", "no parece un email")
	}
	if in.SitioWeb != nil {
		if err := URLWeb("sitioWeb", *in.SitioWeb, 200); err != nil {
			return err
		}
	}
	// Portada vacía: se saca.
	sinPortada := in.Portada != nil && *in.Portada == ""
	if in.Portada != nil {
		if err := URLWeb("portada", *in.Portada, 2048); err != nil {
			return err
		}
	}
	if (in.Latitud == nil) != (in.Longitud == nil) {
		return campo("latitud", "van latitud y longitud juntas")
	}
	if in.Latitud != nil && (*in.Latitud < -90 || *in.Latitud > 90 || *in.Longitud < -180 || *in.Longitud > 180) {
		return campo("latitud", "coordenadas fuera de rango")
	}
	var out Web
	err := s.enNegocio(r, func(ctx context.Context, tx pgx.Tx) (err error) {
		_, err = tx.Exec(ctx, `UPDATE negocios SET
			rubro = coalesce($1, rubro), provincia = coalesce($2, provincia), acerca = coalesce($3, acerca),
			color_primario = coalesce($4, color_primario), color_acento = coalesce($5, color_acento),
			email = coalesce($6, email), sitio_web = coalesce($7, sitio_web), facebook = coalesce($8, facebook),
			tiktok = coalesce($9, tiktok),
			latitud = CASE WHEN $12 THEN NULL ELSE coalesce($10, latitud) END,
			longitud = CASE WHEN $12 THEN NULL ELSE coalesce($11, longitud) END,
			portada = CASE WHEN $13 THEN NULL ELSE coalesce($14, portada) END
			WHERE id = app.negocio_actual()`,
			in.Rubro, in.Provincia, in.Acerca, in.ColorPrimario, in.ColorAcento, in.Email, in.SitioWeb, in.Facebook,
			in.Tiktok, in.Latitud, in.Longitud, in.SinUbicacion, sinPortada, in.Portada)
		if err != nil {
			return err
		}
		out, err = Leer(ctx, tx)
		return err
	})
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusOK, out)
	return nil
}
