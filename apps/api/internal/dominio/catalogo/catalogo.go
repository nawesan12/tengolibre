// Package catalogo: lo que el negocio ofrece (categorías y servicios) y quién lo atiende
// (profesionales, con sus servicios, su horario semanal, sus bloqueos y excepciones).
package catalogo

import (
	"context"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"

	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/disponibilidad"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/db"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/problema"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/sesion"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/web"
)

// Servicio del catálogo.
type Servicio struct {
	Pool   *db.Pool
	Sesion *sesion.Servicio
}

// Categoria de servicios.
type Categoria struct {
	ID     string `json:"id"`
	Nombre string `json:"nombre"`
	Orden  int    `json:"orden"`
}

// Prestacion es un servicio del negocio (Service en Jiku; "servicio" ya es el nombre de este tipo).
type Prestacion struct {
	ID          string  `json:"id"`
	CategoriaID *string `json:"categoriaId"`
	Nombre      string  `json:"nombre"`
	Descripcion string  `json:"descripcion"`
	Duracion    int     `json:"duracion"`
	Precio      int64   `json:"precio"`
	Foto        *string `json:"foto"`
	Orden       int     `json:"orden"`
	Activo      bool    `json:"activo"`
}

// Profesional con lo que hace y cuándo atiende.
type Profesional struct {
	ID           string                  `json:"id"`
	MiembroID    *string                 `json:"miembroId"`
	Nombre       string                  `json:"nombre"`
	Especialidad string                  `json:"especialidad"`
	Bio          string                  `json:"bio"`
	Foto         *string                 `json:"foto"`
	Email        string                  `json:"email"`
	Telefono     string                  `json:"telefono"`
	Orden        int                     `json:"orden"`
	Activo       bool                    `json:"activo"`
	Servicios    []string                `json:"servicios"`
	Horario      []disponibilidad.Franja `json:"horario"`
}

// Rutas del dominio.
func (s *Servicio) Rutas(mux *http.ServeMux) {
	g := func(h web.Manejador) http.HandlerFunc { return s.Sesion.Panel(sesion.Gestion, web.Envolver(h)) }
	a := func(h web.Manejador) http.HandlerFunc { return s.Sesion.Panel(sesion.Agenda, web.Envolver(h)) }
	mux.HandleFunc("GET /v1/panel/categorias", a(s.categorias))
	mux.HandleFunc("POST /v1/panel/categorias", g(s.crearCategoria))
	mux.HandleFunc("PATCH /v1/panel/categorias/{id}", g(s.editarCategoria))
	mux.HandleFunc("DELETE /v1/panel/categorias/{id}", g(s.borrar("categorias")))

	mux.HandleFunc("GET /v1/panel/servicios", a(s.servicios))
	mux.HandleFunc("POST /v1/panel/servicios", g(s.crearServicio))
	mux.HandleFunc("PATCH /v1/panel/servicios/{id}", g(s.editarServicio))
	mux.HandleFunc("DELETE /v1/panel/servicios/{id}", g(s.borrar("servicios")))

	mux.HandleFunc("GET /v1/panel/profesionales", a(s.profesionales))
	mux.HandleFunc("POST /v1/panel/profesionales", g(s.crearProfesional))
	mux.HandleFunc("PATCH /v1/panel/profesionales/{id}", g(s.editarProfesional))
	mux.HandleFunc("DELETE /v1/panel/profesionales/{id}", g(s.borrar("profesionales")))
	mux.HandleFunc("PUT /v1/panel/profesionales/{id}/servicios", g(s.asignarServicios))
	mux.HandleFunc("PUT /v1/panel/profesionales/{id}/horario", g(s.guardarHorario))
	mux.HandleFunc("GET /v1/panel/profesionales/{id}/bloqueos", a(s.bloqueos))
	mux.HandleFunc("POST /v1/panel/profesionales/{id}/bloqueos", a(s.crearBloqueo))
	mux.HandleFunc("DELETE /v1/panel/profesionales/{id}/bloqueos/{bloqueo}", a(s.borrarBloqueo))
	mux.HandleFunc("GET /v1/panel/profesionales/{id}/semanales", a(s.semanales))
	mux.HandleFunc("PUT /v1/panel/profesionales/{id}/semanales", g(s.guardarSemanales))
	mux.HandleFunc("GET /v1/panel/profesionales/{id}/excepciones", a(s.excepciones))
	mux.HandleFunc("PUT /v1/panel/profesionales/{id}/excepciones", a(s.guardarExcepcion))
	mux.HandleFunc("DELETE /v1/panel/profesionales/{id}/excepciones/{fecha}/{hora}", a(s.borrarExcepcion))
}

func (s *Servicio) enNegocio(r *http.Request, fn func(context.Context, pgx.Tx) error) error {
	ctx := r.Context()
	return s.Pool.EnNegocio(ctx, sesion.De(ctx).NegocioID, func(tx pgx.Tx) error { return fn(ctx, tx) })
}

func largo(campo string, v string, min, max int) error {
	if n := utf8.RuneCountInString(v); n < min || n > max {
		return problema.Invalido("Revisá "+campo+".", problema.Campo{Ruta: campo, Mensaje: "largo inválido"})
	}
	return nil
}

func noExiste(tag interface{ RowsAffected() int64 }) error {
	if tag.RowsAffected() == 0 {
		return problema.NoEncontrado("No existe o no es de este negocio.")
	}
	return nil
}

// borrar de una tabla con id: las FK deciden qué pasa con lo que cuelga (cascada o NULL).
func (s *Servicio) borrar(tabla string) web.Manejador {
	return func(w http.ResponseWriter, r *http.Request) error {
		err := s.enNegocio(r, func(ctx context.Context, tx pgx.Tx) error {
			tag, err := tx.Exec(ctx, "DELETE FROM "+tabla+" WHERE id = $1", r.PathValue("id"))
			if err != nil {
				return err
			}
			return noExiste(tag)
		})
		if err != nil {
			return err
		}
		w.WriteHeader(http.StatusNoContent)
		return nil
	}
}

// ---- Categorías

func (s *Servicio) categorias(w http.ResponseWriter, r *http.Request) error {
	var cs []Categoria
	err := s.enNegocio(r, func(ctx context.Context, tx pgx.Tx) (err error) {
		cs, err = Categorias(ctx, tx)
		return err
	})
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusOK, map[string]any{"categorias": cs})
	return nil
}

// Categorias del negocio fijado.
func Categorias(ctx context.Context, tx pgx.Tx) ([]Categoria, error) {
	filas, err := tx.Query(ctx, "SELECT id::text, nombre, orden FROM categorias ORDER BY orden, nombre")
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(filas, pgx.RowToStructByPos[Categoria])
}

func (s *Servicio) crearCategoria(w http.ResponseWriter, r *http.Request) error {
	var in Categoria
	if err := web.Leer(w, r, &in); err != nil {
		return err
	}
	in.Nombre = strings.TrimSpace(in.Nombre)
	if err := largo("nombre", in.Nombre, 1, 60); err != nil {
		return err
	}
	err := s.enNegocio(r, func(ctx context.Context, tx pgx.Tx) error {
		return tx.QueryRow(ctx, "INSERT INTO categorias (negocio_id, nombre, orden) VALUES (app.negocio_actual(), $1, $2) RETURNING id::text",
			in.Nombre, in.Orden).Scan(&in.ID)
	})
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusCreated, in)
	return nil
}

func (s *Servicio) editarCategoria(w http.ResponseWriter, r *http.Request) error {
	var in struct {
		Nombre *string `json:"nombre"`
		Orden  *int    `json:"orden"`
	}
	if err := web.Leer(w, r, &in); err != nil {
		return err
	}
	if in.Nombre != nil {
		*in.Nombre = strings.TrimSpace(*in.Nombre)
		if err := largo("nombre", *in.Nombre, 1, 60); err != nil {
			return err
		}
	}
	var c Categoria
	err := s.enNegocio(r, func(ctx context.Context, tx pgx.Tx) error {
		return tx.QueryRow(ctx, `UPDATE categorias SET nombre = coalesce($2, nombre), orden = coalesce($3, orden) WHERE id = $1
			RETURNING id::text, nombre, orden`, r.PathValue("id"), in.Nombre, in.Orden).Scan(&c.ID, &c.Nombre, &c.Orden)
	})
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusOK, c)
	return nil
}

// ---- Servicios

const columnasServicio = "id::text, categoria_id::text, nombre, descripcion, duracion, precio, foto, orden, activo"

func escanearServicio(f pgx.Row, p *Prestacion) error {
	return f.Scan(&p.ID, &p.CategoriaID, &p.Nombre, &p.Descripcion, &p.Duracion, &p.Precio, &p.Foto, &p.Orden, &p.Activo)
}

// Servicios del negocio fijado; soloActivos para la página pública.
func Servicios(ctx context.Context, tx pgx.Tx, soloActivos bool) ([]Prestacion, error) {
	q := "SELECT " + columnasServicio + " FROM servicios"
	if soloActivos {
		q += " WHERE activo"
	}
	filas, err := tx.Query(ctx, q+" ORDER BY orden, nombre")
	if err != nil {
		return nil, err
	}
	defer filas.Close()
	out := []Prestacion{}
	for filas.Next() {
		var p Prestacion
		if err := escanearServicio(filas, &p); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, filas.Err()
}

// LeerServicio por id en el negocio fijado.
func LeerServicio(ctx context.Context, tx pgx.Tx, id string) (Prestacion, error) {
	var p Prestacion
	err := escanearServicio(tx.QueryRow(ctx, "SELECT "+columnasServicio+" FROM servicios WHERE id = $1", id), &p)
	return p, err
}

func (s *Servicio) servicios(w http.ResponseWriter, r *http.Request) error {
	var ps []Prestacion
	err := s.enNegocio(r, func(ctx context.Context, tx pgx.Tx) (err error) {
		ps, err = Servicios(ctx, tx, false)
		return err
	})
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusOK, map[string]any{"servicios": ps})
	return nil
}

func validarServicio(p *Prestacion) error {
	p.Nombre = strings.TrimSpace(p.Nombre)
	if err := largo("nombre", p.Nombre, 2, 80); err != nil {
		return err
	}
	if p.Duracion < 5 || p.Duracion > 480 || p.Duracion%5 != 0 {
		return problema.Invalido("La duración va de 5 a 480 minutos, de a 5.", problema.Campo{Ruta: "duracion", Mensaje: "fuera de rango"})
	}
	if p.Precio < 0 {
		return problema.Invalido("El precio no puede ser negativo.", problema.Campo{Ruta: "precio", Mensaje: "negativo"})
	}
	if p.CategoriaID != nil && *p.CategoriaID == "" {
		p.CategoriaID = nil
	}
	if p.Foto != nil && *p.Foto == "" {
		p.Foto = nil
	}
	return nil
}

func (s *Servicio) crearServicio(w http.ResponseWriter, r *http.Request) error {
	p := Prestacion{Activo: true}
	if err := web.Leer(w, r, &p); err != nil {
		return err
	}
	if err := validarServicio(&p); err != nil {
		return err
	}
	err := s.enNegocio(r, func(ctx context.Context, tx pgx.Tx) error {
		return escanearServicio(tx.QueryRow(ctx, `INSERT INTO servicios (negocio_id, categoria_id, nombre, descripcion, duracion, precio, foto, orden, activo)
			VALUES (app.negocio_actual(), $1, $2, $3, $4, $5, $6, $7, $8) RETURNING `+columnasServicio,
			p.CategoriaID, p.Nombre, p.Descripcion, p.Duracion, p.Precio, p.Foto, p.Orden, p.Activo), &p)
	})
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusCreated, p)
	return nil
}

// editarServicio: PATCH sobre lo que hay (se lee, se pisa con lo que vino y se valida entero).
func (s *Servicio) editarServicio(w http.ResponseWriter, r *http.Request) error {
	var p Prestacion
	err := s.enNegocio(r, func(ctx context.Context, tx pgx.Tx) error {
		var err error
		if p, err = LeerServicio(ctx, tx, r.PathValue("id")); err != nil {
			return err
		}
		id := p.ID
		if err := web.Leer(w, r, &p); err != nil {
			return err
		}
		p.ID = id
		if err := validarServicio(&p); err != nil {
			return err
		}
		return escanearServicio(tx.QueryRow(ctx, `UPDATE servicios SET categoria_id = $2, nombre = $3, descripcion = $4, duracion = $5,
			precio = $6, foto = $7, orden = $8, activo = $9 WHERE id = $1 RETURNING `+columnasServicio,
			p.ID, p.CategoriaID, p.Nombre, p.Descripcion, p.Duracion, p.Precio, p.Foto, p.Orden, p.Activo), &p)
	})
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusOK, p)
	return nil
}
