// Package sesion identifica a quien llama: la persona por su token de Clerk y, en el panel, el
// negocio en el que trabaja (encabezado X-Negocio) con su rol, que se verifica contra miembros.
package sesion

import (
	"context"
	"errors"
	"net/http"
	"slices"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/clerk"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/db"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/problema"
)

// Roles del panel, de más a menos permisos.
const (
	Duenio      = "duenio"
	Encargado   = "encargado"
	Recepcion   = "recepcion"
	Profesional = "profesional"
)

// Gestion: los que administran el negocio (catálogo, equipo, ajustes).
var Gestion = []string{Duenio, Encargado}

// Agenda: los que manejan turnos.
var Agenda = []string{Duenio, Encargado, Recepcion, Profesional}

// Verificador valida un token y devuelve el id de la persona (clerk.Cliente lo cumple).
type Verificador interface {
	Verificar(ctx context.Context, token string) (string, error)
	Usuario(ctx context.Context, id string) (clerk.Usuario, error)
}

// Negocio es uno de los negocios de la persona.
type Negocio struct {
	ID     string `json:"id"`
	Slug   string `json:"slug"`
	Nombre string `json:"nombre"`
	Rol    string `json:"rol"`
}

// Sesion de un pedido.
type Sesion struct {
	ClerkID   string
	NegocioID string
	Rol       string
}

type clave struct{}

// De devuelve la sesión que dejó Persona o Panel.
func De(ctx context.Context) Sesion {
	s, _ := ctx.Value(clave{}).(Sesion)
	return s
}

// ConSesion deja s en el contexto (tests de los dominios).
func ConSesion(ctx context.Context, s Sesion) context.Context {
	return context.WithValue(ctx, clave{}, s)
}

// Servicio arma los middlewares. Sin Verificador, todo lo que pide sesión responde 401.
type Servicio struct {
	Pool        *db.Pool
	Verificador Verificador
}

func token(r *http.Request) string {
	a := r.Header.Get("Authorization")
	if t, ok := strings.CutPrefix(a, "Bearer "); ok {
		return strings.TrimSpace(t)
	}
	return ""
}

// Persona exige un token válido y deja el ClerkID.
func (s *Servicio) Persona(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		t := token(r)
		if s.Verificador == nil || t == "" {
			problema.Escribir(w, problema.SinSesion())
			return
		}
		id, err := s.Verificador.Verificar(r.Context(), t)
		if err != nil {
			problema.Escribir(w, problema.SinSesion())
			return
		}
		next(w, r.WithContext(ConSesion(r.Context(), Sesion{ClerkID: id})))
	}
}

// Panel exige persona, un X-Negocio del que sea miembro activo y uno de los roles.
func (s *Servicio) Panel(roles []string, next http.HandlerFunc) http.HandlerFunc {
	return s.Persona(func(w http.ResponseWriter, r *http.Request) {
		ses := De(r.Context())
		negocio := r.Header.Get("X-Negocio")
		if negocio == "" {
			problema.Escribir(w, problema.Invalido("Falta el encabezado X-Negocio."))
			return
		}
		ns, err := s.Negocios(r.Context(), ses.ClerkID)
		if err != nil {
			problema.Escribir(w, problema.Interno())
			return
		}
		i := slices.IndexFunc(ns, func(n Negocio) bool { return n.ID == negocio })
		if i < 0 {
			problema.Escribir(w, problema.SinPermiso("No sos parte de ese negocio."))
			return
		}
		if !slices.Contains(roles, ns[i].Rol) {
			problema.Escribir(w, problema.SinPermiso("Tu rol no puede hacer esto."))
			return
		}
		ses.NegocioID, ses.Rol = negocio, ns[i].Rol
		next(w, r.WithContext(ConSesion(r.Context(), ses)))
	})
}

// Negocios de una persona (activos, como miembro activo).
func (s *Servicio) Negocios(ctx context.Context, clerkID string) ([]Negocio, error) {
	out := []Negocio{}
	if s.Pool == nil {
		return out, errors.New("sesion: sin base")
	}
	err := s.Pool.EnTx(ctx, func(tx pgx.Tx) error {
		filas, err := tx.Query(ctx, "SELECT negocio_id::text, slug, nombre, rol FROM app.negocios_de($1)", clerkID)
		if err != nil {
			return err
		}
		out, err = pgx.CollectRows(filas, pgx.RowToStructByPos[Negocio])
		return err
	})
	return out, err
}
