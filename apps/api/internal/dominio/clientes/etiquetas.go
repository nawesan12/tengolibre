package clientes

import (
	"context"
	"net/http"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"

	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/problema"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/web"
)

// ColorPorDefecto de una etiqueta (el de Jiku).
const ColorPorDefecto = "#6366F1"

var reColor = regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`)

func validarEtiqueta(nombre, color string) error {
	if n := utf8.RuneCountInString(nombre); n < 1 || n > 40 {
		return problema.Invalido("El nombre de la etiqueta va de 1 a 40 letras.", problema.Campo{Ruta: "nombre", Mensaje: "largo inválido"})
	}
	if !reColor.MatchString(color) {
		return problema.Invalido("El color va como #RRGGBB.", problema.Campo{Ruta: "color", Mensaje: "formato inválido"})
	}
	return nil
}

// EtiquetaConUso: una etiqueta y a cuántos clientes se le puso.
type EtiquetaConUso struct {
	Etiqueta
	Clientes int `json:"clientes"`
}

func (s *Servicio) etiquetas(w http.ResponseWriter, r *http.Request) error {
	out := []EtiquetaConUso{}
	err := s.enNegocio(r, func(ctx context.Context, tx pgx.Tx) error {
		filas, err := tx.Query(ctx, `SELECT e.id::text, e.nombre, e.color, count(ce.cliente_id)
			FROM etiquetas e LEFT JOIN cliente_etiquetas ce ON ce.etiqueta_id = e.id
			GROUP BY e.id ORDER BY lower(e.nombre)`)
		if err != nil {
			return err
		}
		defer filas.Close()
		for filas.Next() {
			var e EtiquetaConUso
			if err := filas.Scan(&e.ID, &e.Nombre, &e.Color, &e.Clientes); err != nil {
				return err
			}
			out = append(out, e)
		}
		return filas.Err()
	})
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusOK, map[string]any{"etiquetas": out})
	return nil
}

func (s *Servicio) crearEtiqueta(w http.ResponseWriter, r *http.Request) error {
	var in Etiqueta
	if err := web.Leer(w, r, &in); err != nil {
		return err
	}
	in.Nombre = strings.TrimSpace(in.Nombre)
	if in.Color == "" {
		in.Color = ColorPorDefecto
	}
	if err := validarEtiqueta(in.Nombre, in.Color); err != nil {
		return err
	}
	err := s.enNegocio(r, func(ctx context.Context, tx pgx.Tx) error {
		return tx.QueryRow(ctx, `INSERT INTO etiquetas (negocio_id, nombre, color) VALUES (app.negocio_actual(), $1, upper($2))
			RETURNING id::text, nombre, color`, in.Nombre, in.Color).Scan(&in.ID, &in.Nombre, &in.Color)
	})
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusCreated, in)
	return nil
}

func (s *Servicio) editarEtiqueta(w http.ResponseWriter, r *http.Request) error {
	var in struct {
		Nombre *string `json:"nombre"`
		Color  *string `json:"color"`
	}
	if err := web.Leer(w, r, &in); err != nil {
		return err
	}
	var e Etiqueta
	err := s.enNegocio(r, func(ctx context.Context, tx pgx.Tx) error {
		if err := tx.QueryRow(ctx, "SELECT id::text, nombre, color FROM etiquetas WHERE id = $1 FOR UPDATE", r.PathValue("id")).
			Scan(&e.ID, &e.Nombre, &e.Color); err != nil {
			return err
		}
		if in.Nombre != nil {
			e.Nombre = strings.TrimSpace(*in.Nombre)
		}
		if in.Color != nil {
			e.Color = *in.Color
		}
		if err := validarEtiqueta(e.Nombre, e.Color); err != nil {
			return err
		}
		return tx.QueryRow(ctx, "UPDATE etiquetas SET nombre = $2, color = upper($3) WHERE id = $1 RETURNING color",
			e.ID, e.Nombre, e.Color).Scan(&e.Color)
	})
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusOK, e)
	return nil
}

// borrarEtiqueta: se va también de los clientes que la tenían (cascada).
func (s *Servicio) borrarEtiqueta(w http.ResponseWriter, r *http.Request) error {
	err := s.enNegocio(r, func(ctx context.Context, tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, "DELETE FROM etiquetas WHERE id = $1", r.PathValue("id"))
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return problema.NoEncontrado("Esa etiqueta no existe o no es de este negocio.")
		}
		return nil
	})
	if err != nil {
		return err
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}

// asignar la etiqueta al cliente. Ponerla dos veces no es error.
func (s *Servicio) asignar(w http.ResponseWriter, r *http.Request) error {
	err := s.enNegocio(r, func(ctx context.Context, tx pgx.Tx) error {
		if err := existe(ctx, tx, r.PathValue("id")); err != nil {
			return err
		}
		var hay bool
		if err := tx.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM etiquetas WHERE id = $1)", r.PathValue("etiqueta")).Scan(&hay); err != nil {
			return err
		}
		if !hay {
			return problema.NoEncontrado("Esa etiqueta no existe o no es de este negocio.")
		}
		_, err := tx.Exec(ctx, `INSERT INTO cliente_etiquetas (negocio_id, cliente_id, etiqueta_id) VALUES (app.negocio_actual(), $1, $2)
			ON CONFLICT DO NOTHING`, r.PathValue("id"), r.PathValue("etiqueta"))
		return err
	})
	if err != nil {
		return err
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}

// quitar la etiqueta. Quitar una que no tenía tampoco es error.
func (s *Servicio) quitar(w http.ResponseWriter, r *http.Request) error {
	err := s.enNegocio(r, func(ctx context.Context, tx pgx.Tx) error {
		if err := existe(ctx, tx, r.PathValue("id")); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, "DELETE FROM cliente_etiquetas WHERE cliente_id = $1 AND etiqueta_id = $2", r.PathValue("id"), r.PathValue("etiqueta"))
		return err
	})
	if err != nil {
		return err
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}
