// Package cuenta: borrar la cuenta (lo piden Google Play y la App Store). La baja desactiva la cuenta
// en el momento (sesiones, push, membresías y los negocios donde era la única dueña) y el worker
// borra sus datos personales a los DiasParaPurgar. Mientras tanto, el ingreso responde
// cuenta_borrada y la cuenta se puede recuperar a mano (tengolibrectl recuperar-cuenta). Todo lo
// que toca negocios ajenos a la sesión va por funciones de la base (ver 00024_cuentas_borradas).
package cuenta

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/db"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/sesion"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/trabajos"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/web"
)

// DiasParaPurgar: los días entre la baja y el borrado de los datos personales.
const DiasParaPurgar = 30

// Servicio de la baja.
type Servicio struct {
	Pool   *db.Pool
	Sesion *sesion.Servicio
	// CancelarSuscripcion da de baja la suscripción de un negocio que se cierra (para que Mercado
	// Pago no le siga cobrando). nil: no se cancela nada (tests).
	CancelarSuscripcion func(ctx context.Context, negocioID string) error
	Log                 *slog.Logger
}

func (s *Servicio) log() *slog.Logger {
	if s.Log != nil {
		return s.Log
	}
	return slog.Default()
}

// Rutas: la baja, con sesión (propia o de Clerk).
func (s *Servicio) Rutas(mux *http.ServeMux) {
	mux.HandleFunc("DELETE /v1/auth/cuenta", s.Sesion.Persona(web.Envolver(s.borrar)))
}

func (s *Servicio) borrar(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	ses := sesion.De(ctx)
	email := ses.Persona.Email
	if email == "" && s.Sesion.Verificador != nil {
		if u, err := s.Sesion.Verificador.Usuario(ctx, ses.ClerkID); err == nil {
			email = u.Email
		}
	}
	// Antes de cerrar nada, las suscripciones de los negocios que se van a cerrar: si Mercado Pago
	// no responde, la baja no se hace y la persona reintenta (mejor eso que seguir cobrándole).
	cerrar, err := s.unicaDuenia(ctx, ses.ClerkID)
	if err != nil {
		return err
	}
	if s.CancelarSuscripcion != nil {
		for _, n := range cerrar {
			if err := s.CancelarSuscripcion(ctx, n); err != nil {
				return err
			}
		}
	}
	var cerrados []string
	err = s.Pool.EnTx(ctx, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, "SELECT coalesce(app.borrar_cuenta($1, $2)::text[], '{}')", ses.ClerkID, email).Scan(&cerrados)
	})
	if err != nil {
		return err
	}
	s.log().Info("cuenta borrada", "clerk", ses.ClerkID, "negocios_cerrados", len(cerrados))
	w.WriteHeader(http.StatusNoContent)
	return nil
}

// unicaDuenia: los negocios activos donde la persona es la única dueña activa (los que la baja
// cierra).
func (s *Servicio) unicaDuenia(ctx context.Context, clerkID string) ([]string, error) {
	ns, err := s.Sesion.Negocios(ctx, clerkID)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, n := range ns {
		if n.Rol != sesion.Duenio {
			continue
		}
		var otras int
		err := s.Pool.EnNegocio(ctx, n.ID, func(tx pgx.Tx) error {
			return tx.QueryRow(ctx, "SELECT count(*) FROM miembros WHERE rol = 'duenio' AND activo AND clerk_id <> $1", clerkID).Scan(&otras)
		})
		if err != nil {
			return nil, err
		}
		if otras == 0 {
			out = append(out, n.ID)
		}
	}
	return out, nil
}

// Purgar borra los datos personales de las cuentas borradas hace más de DiasParaPurgar (de a
// tandas) y devuelve cuántas purgó.
func Purgar(ctx context.Context, pool *db.Pool, ahora time.Time) (int, error) {
	total := 0
	for {
		var n int
		err := pool.EnTx(ctx, func(tx pgx.Tx) error {
			return tx.QueryRow(ctx, "SELECT app.purgar_cuentas($1, 50)", ahora.AddDate(0, 0, -DiasParaPurgar)).Scan(&n)
		})
		if err != nil {
			return total, err
		}
		total += n
		if n < 50 {
			return total, nil
		}
	}
}

// Trabajo de la purga para el worker: una vez por día.
func Trabajo(pool *db.Pool) trabajos.Trabajo {
	return trabajos.Trabajo{Nombre: "purga_cuentas", Cada: 24 * time.Hour, Correr: func(ctx context.Context, ahora time.Time) (any, error) {
		n, err := Purgar(ctx, pool, ahora)
		return map[string]int{"purgadas": n}, err
	}}
}
