package clientes

import (
	"context"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"

	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/sesion"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/web"
)

// Resultados de la búsqueda global del panel.
type Resultados struct {
	Clientes  []ClienteEncontrado  `json:"clientes"`
	Turnos    []TurnoEncontrado    `json:"turnos"`
	Servicios []ServicioEncontrado `json:"servicios"`
}

// ClienteEncontrado por nombre, email o teléfono.
type ClienteEncontrado struct {
	ID       string `json:"id"`
	Nombre   string `json:"nombre"`
	Telefono string `json:"telefono"`
	Email    string `json:"email"`
}

// TurnoEncontrado por el cliente (nombre o teléfono) o el servicio.
type TurnoEncontrado struct {
	ID          string    `json:"id"`
	Inicio      time.Time `json:"inicio"`
	Estado      string    `json:"estado"`
	Servicio    string    `json:"servicio"`
	Profesional string    `json:"profesional"`
	ClienteID   string    `json:"clienteId"`
	Cliente     string    `json:"cliente"`
}

// ServicioEncontrado por nombre.
type ServicioEncontrado struct {
	ID       string `json:"id"`
	Nombre   string `json:"nombre"`
	Precio   int64  `json:"precio"`
	Duracion int    `json:"duracion"`
	Activo   bool   `json:"activo"`
}

// buscar ?q= (desde 2 letras): hasta 8 clientes, 8 turnos (los más nuevos primero) y 5 servicios.
// Un profesional (rol) ve sólo sus turnos, como en la agenda.
func (s *Servicio) buscar(w http.ResponseWriter, r *http.Request) error {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	out := Resultados{Clientes: []ClienteEncontrado{}, Turnos: []TurnoEncontrado{}, Servicios: []ServicioEncontrado{}}
	if utf8.RuneCountInString(q) < 2 {
		web.JSON(w, http.StatusOK, out)
		return nil
	}
	p := patron(q)
	// Con menos de 3 dígitos no se busca por teléfono: "12" aparece en casi todos.
	tel := ""
	if d := soloDigitos(q); len(d) >= 3 {
		tel = patron(d)
	}
	ses := sesion.De(r.Context())
	err := s.enNegocio(r, func(ctx context.Context, tx pgx.Tx) error {
		filas, err := tx.Query(ctx, `SELECT id::text, nombre, telefono, email FROM clientes
			WHERE nombre ILIKE $1 OR email ILIKE $1 OR ($2 <> '' AND telefono LIKE $2) ORDER BY lower(nombre), id LIMIT 8`, p, tel)
		if err != nil {
			return err
		}
		if out.Clientes, err = pgx.CollectRows(filas, pgx.RowToStructByPos[ClienteEncontrado]); err != nil {
			return err
		}
		propios := ""
		args := []any{p, tel}
		if ses.Rol == sesion.Profesional {
			args = append(args, ses.ClerkID)
			propios = ` AND t.profesional_id IN (SELECT pr.id FROM profesionales pr JOIN miembros m ON m.id = pr.miembro_id
				WHERE m.clerk_id = $3)`
		}
		filas, err = tx.Query(ctx, `SELECT t.id::text, t.inicio, t.estado, t.servicio, t.profesional, c.id::text, c.nombre
			FROM turnos t JOIN clientes c ON c.id = t.cliente_id
			WHERE (c.nombre ILIKE $1 OR ($2 <> '' AND c.telefono LIKE $2) OR t.servicio ILIKE $1)`+propios+`
			ORDER BY t.inicio DESC LIMIT 8`, args...)
		if err != nil {
			return err
		}
		if out.Turnos, err = pgx.CollectRows(filas, pgx.RowToStructByPos[TurnoEncontrado]); err != nil {
			return err
		}
		filas, err = tx.Query(ctx, `SELECT id::text, nombre, precio, duracion, activo FROM servicios
			WHERE nombre ILIKE $1 ORDER BY activo DESC, orden, nombre LIMIT 5`, p)
		if err != nil {
			return err
		}
		out.Servicios, err = pgx.CollectRows(filas, pgx.RowToStructByPos[ServicioEncontrado])
		return err
	})
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusOK, out)
	return nil
}
