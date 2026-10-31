package caja

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/auditoria"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/problema"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/web"
)

// Producto del catálogo del local. Stock null: no se lleva la cuenta.
type Producto struct {
	ID     string `json:"id"`
	Nombre string `json:"nombre"`
	Precio int64  `json:"precio"`
	Stock  *int   `json:"stock"`
	Activo bool   `json:"activo"`
	Orden  int    `json:"orden"`
}

const columnasProducto = "id::text, nombre, precio, stock, activo, orden"

func escanearProducto(f pgx.Row, p *Producto) error {
	return f.Scan(&p.ID, &p.Nombre, &p.Precio, &p.Stock, &p.Activo, &p.Orden)
}

// EntradaProducto del panel. En PATCH, lo que no viene no cambia; SinStock deja de llevar la cuenta.
type EntradaProducto struct {
	Nombre   *string `json:"nombre"`
	Precio   *int64  `json:"precio"`
	Stock    *int    `json:"stock"`
	SinStock bool    `json:"sinStock"`
	Activo   *bool   `json:"activo"`
	Orden    *int    `json:"orden"`
}

func (e *EntradaProducto) validar(nuevo bool) error {
	if e.Nombre != nil {
		n := strings.TrimSpace(*e.Nombre)
		e.Nombre = &n
		if c := utf8.RuneCountInString(n); c < 2 || c > 80 {
			return problema.Invalido("El nombre va de 2 a 80 letras.", problema.Campo{Ruta: "nombre", Mensaje: "largo"})
		}
	} else if nuevo {
		return problema.Invalido("Falta el nombre.", problema.Campo{Ruta: "nombre", Mensaje: "requerido"})
	}
	if e.Precio != nil && (*e.Precio < 0 || *e.Precio > 100000000) {
		return problema.Invalido("Revisá el precio.", problema.Campo{Ruta: "precio", Mensaje: "fuera de rango"})
	} else if e.Precio == nil && nuevo {
		return problema.Invalido("Falta el precio.", problema.Campo{Ruta: "precio", Mensaje: "requerido"})
	}
	if e.Stock != nil && (*e.Stock < 0 || *e.Stock > 1000000) {
		return problema.Invalido("Revisá el stock.", problema.Campo{Ruta: "stock", Mensaje: "fuera de rango"})
	}
	if e.Stock != nil && e.SinStock {
		return problema.Invalido("Va stock o sinStock, no los dos.", problema.Campo{Ruta: "stock", Mensaje: "contradictorio"})
	}
	if e.Orden != nil && (*e.Orden < 0 || *e.Orden > 999) {
		return problema.Invalido("El orden va de 0 a 999.", problema.Campo{Ruta: "orden", Mensaje: "fuera de rango"})
	}
	return nil
}

func duplicado(err error) error {
	var pg *pgconn.PgError
	if errors.As(err, &pg) && pg.Code == "23505" {
		return problema.Conflicto("duplicado", "Ya hay un producto con ese nombre.")
	}
	return err
}

// productos del negocio: ?activos=1 deja sólo los que se pueden vender.
func (s *Servicio) productos(w http.ResponseWriter, r *http.Request) error {
	out := []Producto{}
	soloActivos := r.URL.Query().Get("activos") == "1"
	err := s.enNegocio(r, func(ctx context.Context, tx pgx.Tx) error {
		filas, err := tx.Query(ctx, "SELECT "+columnasProducto+" FROM productos WHERE activo OR NOT $1 ORDER BY orden, nombre LIMIT 500", soloActivos)
		if err != nil {
			return err
		}
		defer filas.Close()
		for filas.Next() {
			var p Producto
			if err := escanearProducto(filas, &p); err != nil {
				return err
			}
			out = append(out, p)
		}
		return filas.Err()
	})
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusOK, map[string]any{"productos": out})
	return nil
}

func (s *Servicio) crearProducto(w http.ResponseWriter, r *http.Request) error {
	var in EntradaProducto
	if err := web.Leer(w, r, &in); err != nil {
		return err
	}
	if err := in.validar(true); err != nil {
		return err
	}
	activo, orden := true, 0
	if in.Activo != nil {
		activo = *in.Activo
	}
	if in.Orden != nil {
		orden = *in.Orden
	}
	var p Producto
	err := s.enNegocio(r, func(ctx context.Context, tx pgx.Tx) error {
		err := escanearProducto(tx.QueryRow(ctx, `INSERT INTO productos (negocio_id, nombre, precio, stock, activo, orden)
			VALUES (app.negocio_actual(), $1, $2, $3, $4, $5) RETURNING `+columnasProducto, *in.Nombre, *in.Precio, in.Stock, activo, orden), &p)
		if err != nil {
			return duplicado(err)
		}
		return auditoria.Anotar(ctx, tx, auditoria.Registro{Accion: "producto.creado", Entidad: "producto", EntidadID: p.ID,
			Detalle: map[string]any{"nombre": p.Nombre, "precio": p.Precio}})
	})
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusCreated, p)
	return nil
}

func (s *Servicio) editarProducto(w http.ResponseWriter, r *http.Request) error {
	var in EntradaProducto
	if err := web.Leer(w, r, &in); err != nil {
		return err
	}
	if err := in.validar(false); err != nil {
		return err
	}
	var p Producto
	err := s.enNegocio(r, func(ctx context.Context, tx pgx.Tx) error {
		err := escanearProducto(tx.QueryRow(ctx, `UPDATE productos SET nombre = coalesce($2, nombre), precio = coalesce($3, precio),
			stock = CASE WHEN $4 THEN NULL ELSE coalesce($5, stock) END, activo = coalesce($6, activo), orden = coalesce($7, orden)
			WHERE id::text = $1 RETURNING `+columnasProducto, r.PathValue("id"), in.Nombre, in.Precio, in.SinStock, in.Stock, in.Activo, in.Orden), &p)
		if errors.Is(err, pgx.ErrNoRows) {
			return problema.NoEncontrado("No encontramos ese producto.")
		}
		if err != nil {
			return duplicado(err)
		}
		return auditoria.Anotar(ctx, tx, auditoria.Registro{Accion: "producto.editado", Entidad: "producto", EntidadID: p.ID,
			Detalle: map[string]any{"nombre": p.Nombre, "precio": p.Precio, "stock": p.Stock, "activo": p.Activo}})
	})
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusOK, p)
	return nil
}

// borrarProducto: lo vendido conserva nombre y precio (caja_items los copia).
func (s *Servicio) borrarProducto(w http.ResponseWriter, r *http.Request) error {
	err := s.enNegocio(r, func(ctx context.Context, tx pgx.Tx) error {
		var nombre string
		err := tx.QueryRow(ctx, "DELETE FROM productos WHERE id::text = $1 RETURNING nombre", r.PathValue("id")).Scan(&nombre)
		if errors.Is(err, pgx.ErrNoRows) {
			return problema.NoEncontrado("No encontramos ese producto.")
		}
		if err != nil {
			return err
		}
		return auditoria.Anotar(ctx, tx, auditoria.Registro{Accion: "producto.borrado", Entidad: "producto", EntidadID: r.PathValue("id"),
			Detalle: map[string]any{"nombre": nombre}})
	})
	if err != nil {
		return err
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}
