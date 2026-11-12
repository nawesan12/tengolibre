package sesion

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/clerk"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/problema"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/web"
)

// Sesiones propias: el token de Clerk (dura un minuto) se canjea una vez por una sesión de
// TengoLibre de Duracion, que es la que mandan después la web y la app.

// Prefijo de los tokens propios (así Persona los distingue de un JWT de Clerk).
const Prefijo = "tl_"

// Duracion de una sesión.
const Duracion = 30 * 24 * time.Hour

// Persona de la sesión, como la ve el front.
type Persona struct {
	ClerkID string `json:"clerkId"`
	Email   string `json:"email"`
	Nombre  string `json:"nombre"`
	Foto    string `json:"foto"`
}

func hashDe(token string) []byte {
	h := sha256.Sum256([]byte(token))
	return h[:]
}

// propia busca la sesión del token (vigente) y la marca como usada (a lo sumo una vez por hora).
func (s *Servicio) propia(ctx context.Context, token string) (Persona, error) {
	var p Persona
	var usada time.Time
	err := s.Pool.EnTx(ctx, func(tx pgx.Tx) error {
		err := tx.QueryRow(ctx, `SELECT clerk_id, email, nombre, foto, usada_en FROM auth.sesiones
			WHERE hash = $1 AND vence_en > now()`, hashDe(token)).Scan(&p.ClerkID, &p.Email, &p.Nombre, &p.Foto, &usada)
		if err != nil {
			return err
		}
		if time.Since(usada) > time.Hour {
			_, err = tx.Exec(ctx, "UPDATE auth.sesiones SET usada_en = now() WHERE hash = $1", hashDe(token))
		}
		return err
	})
	return p, err
}

// Rutas de ingreso y salida.
func (s *Servicio) Rutas(mux *http.ServeMux) {
	mux.HandleFunc("POST /v1/auth/clerk", web.Envolver(s.ingresarConClerk))
	mux.HandleFunc("POST /v1/auth/salir", web.Envolver(s.salir))
	mux.HandleFunc("GET /v1/auth/persona", s.Persona(web.Envolver(s.persona)))
}

// ingresarConClerk verifica el token de Clerk, lee a la persona (su email tiene que estar
// verificado) y abre una sesión propia.
func (s *Servicio) ingresarConClerk(w http.ResponseWriter, r *http.Request) error {
	var in struct {
		Token string `json:"token"`
	}
	if err := web.Leer(w, r, &in); err != nil {
		return err
	}
	if s.Verificador == nil {
		return problema.Conflicto("ingreso_apagado", "El ingreso no está configurado.")
	}
	ctx := r.Context()
	id, err := s.Verificador.Verificar(ctx, strings.TrimSpace(in.Token))
	if errors.Is(err, clerk.ErrToken) {
		return &problema.Problema{Estado: http.StatusUnauthorized, Codigo: "token_invalido", Titulo: "No pudimos verificar tu ingreso",
			Detalle: "Volvé a intentar."}
	}
	if err != nil {
		return err
	}
	if borrada, err := s.Borrada(ctx, id); err != nil {
		return err
	} else if borrada {
		return ProblemaCuentaBorrada()
	}
	u, err := s.Verificador.Usuario(ctx, id)
	if err != nil {
		return err
	}
	if u.Email == "" {
		return problema.NoProcesable("sin_email", "Tu cuenta no tiene un email verificado.")
	}
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	token := Prefijo + base64.RawURLEncoding.EncodeToString(b)
	p := Persona{ClerkID: id, Email: u.Email, Nombre: strings.TrimSpace(u.Nombre + " " + u.Apellido), Foto: u.Foto}
	vence := time.Now().Add(Duracion)
	err = s.Pool.EnTx(ctx, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `INSERT INTO auth.sesiones (hash, clerk_id, email, nombre, foto, vence_en) VALUES ($1, $2, $3, $4, $5, $6)`,
			hashDe(token), p.ClerkID, p.Email, p.Nombre, p.Foto, vence)
		if err != nil {
			return err
		}
		// De paso, se limpian las vencidas de esta persona.
		_, err = tx.Exec(ctx, "DELETE FROM auth.sesiones WHERE clerk_id = $1 AND vence_en <= now()", p.ClerkID)
		return err
	})
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusCreated, map[string]any{"token": token, "venceEn": vence, "persona": p})
	return nil
}

// ProblemaCuentaBorrada: la persona borró su cuenta y todavía está en los 30 días antes de la purga.
func ProblemaCuentaBorrada() *problema.Problema {
	return &problema.Problema{Estado: http.StatusConflict, Codigo: "cuenta_borrada", Titulo: "Esta cuenta se borró",
		Detalle: "Esta cuenta se borró. Si fue un error, escribinos a hola@tengolibre.com.ar dentro de los 30 días y la recuperamos."}
}

// Borrada dice si la persona borró su cuenta y la baja no se purgó todavía (después de la purga
// vuelve a entrar como una cuenta nueva).
func (s *Servicio) Borrada(ctx context.Context, clerkID string) (bool, error) {
	var borrada bool
	if s.Pool == nil {
		return false, nil
	}
	err := s.Pool.EnTx(ctx, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM auth.cuentas_borradas WHERE clerk_id = $1 AND purgada_en IS NULL)",
			clerkID).Scan(&borrada)
	})
	return borrada, err
}

func (s *Servicio) salir(w http.ResponseWriter, r *http.Request) error {
	if t := token(r); strings.HasPrefix(t, Prefijo) {
		err := s.Pool.EnTx(r.Context(), func(tx pgx.Tx) error {
			_, err := tx.Exec(r.Context(), "DELETE FROM auth.sesiones WHERE hash = $1", hashDe(t))
			return err
		})
		if err != nil {
			return err
		}
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}

func (s *Servicio) persona(w http.ResponseWriter, r *http.Request) error {
	ses := De(r.Context())
	if ses.Persona.ClerkID == "" {
		u, err := s.Verificador.Usuario(r.Context(), ses.ClerkID)
		if err != nil {
			return err
		}
		ses.Persona = Persona{ClerkID: u.ID, Email: u.Email, Nombre: strings.TrimSpace(u.Nombre + " " + u.Apellido), Foto: u.Foto}
	}
	web.JSON(w, http.StatusOK, ses.Persona)
	return nil
}
