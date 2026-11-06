// Package push: la campanita del panel también en el teléfono de quienes atienden el negocio.
//
// La app (iOS o Android) registra el token de cada teléfono atado a la sesión con que entró
// (POST /v1/yo/dispositivos) y lo da de baja al salir (DELETE); cerrar o vencer la sesión también lo
// borra. El worker toma cada notificación nueva de la campanita y la manda a los teléfonos de los
// miembros activos del negocio: por APNs al iPhone, por FCM a Android y por Web Push al panel abierto
// en el navegador (web.go). Cada persona elige qué tipos de la campanita le llegan (preferencias.go).
package push

import (
	"context"
	"net/http"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"

	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/db"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/problema"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/sesion"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/web"
)

// Dispositivo es lo que manda la app al registrarse.
type Dispositivo struct {
	Plataforma string `json:"plataforma"` // ios | android
	Token      string `json:"token"`
	// Entorno: sólo iOS (sandbox desde Xcode; produccion en TestFlight y App Store). Vacío: produccion.
	Entorno string `json:"entorno"`
	Modelo  string `json:"modelo"`
	Version string `json:"version"`
}

var (
	tokenAPNs = regexp.MustCompile(`^[0-9a-f]{64,200}$`)
	tokenFCM  = regexp.MustCompile(`^[A-Za-z0-9_:-]{20,}$`)
	// El token de push de una Live Activity (hexadecimal, más largo que el del dispositivo).
	tokenActividad = regexp.MustCompile(`^[0-9a-f]{64,400}$`)
	uuidRE         = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
)

func limpiar(s string, max int) string {
	s = strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7f {
			return -1
		}
		return r
	}, strings.TrimSpace(s))
	for utf8.RuneCountInString(s) > max {
		_, n := utf8.DecodeLastRuneInString(s)
		s = s[:len(s)-n]
	}
	return s
}

// Normalizar valida y limpia lo que mandó la app.
func Normalizar(d Dispositivo) (Dispositivo, error) {
	d.Token = strings.TrimSpace(d.Token)
	switch d.Plataforma {
	case "ios":
		d.Token = strings.ToLower(d.Token)
		if !tokenAPNs.MatchString(d.Token) {
			return d, problema.Invalido("El token del iPhone no es válido.",
				problema.Campo{Ruta: "token", Mensaje: "Tiene que ser el token de APNs en hexadecimal."})
		}
		switch d.Entorno {
		case "sandbox", "produccion":
		case "":
			d.Entorno = "produccion"
		default:
			return d, problema.Invalido("El entorno tiene que ser sandbox o produccion.",
				problema.Campo{Ruta: "entorno", Mensaje: "sandbox o produccion"})
		}
	case "android":
		if !tokenFCM.MatchString(d.Token) || len(d.Token) > 4096 {
			return d, problema.Invalido("El token del teléfono no es válido.",
				problema.Campo{Ruta: "token", Mensaje: "Tiene que ser el token de registro de FCM."})
		}
		d.Entorno = "produccion"
	default:
		return d, problema.Invalido("La plataforma tiene que ser ios o android.",
			problema.Campo{Ruta: "plataforma", Mensaje: "ios o android"})
	}
	d.Modelo, d.Version = limpiar(d.Modelo, 60), limpiar(d.Version, 30)
	return d, nil
}

// Servicio de las rutas de dispositivos.
type Servicio struct {
	Pool   *db.Pool
	Sesion *sesion.Servicio
	// ClaveVAPID: la pública del push web (base64url). Vacía: el push web está apagado.
	ClaveVAPID string
}

// Rutas: POST y DELETE /v1/yo/dispositivos, con la sesión propia de la persona (la de la app); las
// del push web y las preferencias.
func (s *Servicio) Rutas(mux *http.ServeMux) {
	mux.HandleFunc("POST /v1/yo/dispositivos", s.Sesion.Persona(web.Envolver(s.registrar)))
	mux.HandleFunc("DELETE /v1/yo/dispositivos", s.Sesion.Persona(web.Envolver(s.baja)))
	s.rutasWeb(mux)
	s.rutasTurnos(mux)
	s.rutasPreferencias(mux)
}

func sesionPropia(ctx context.Context) (sesion.Sesion, error) {
	ses := sesion.De(ctx)
	if len(ses.Hash) == 0 {
		return ses, problema.SinPermiso("Los avisos al teléfono necesitan la sesión de la app.")
	}
	return ses, nil
}

// registrar ata el teléfono a la sesión. Si el token ya estaba con otra (la misma persona que volvió
// a entrar, u otra cuenta en ese teléfono), se mueve a esta.
func (s *Servicio) registrar(w http.ResponseWriter, r *http.Request) error {
	ses, err := sesionPropia(r.Context())
	if err != nil {
		return err
	}
	var d Dispositivo
	if err := web.Leer(w, r, &d); err != nil {
		return err
	}
	if d, err = Normalizar(d); err != nil {
		return err
	}
	ctx := r.Context()
	err = s.Pool.EnTx(ctx, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, "DELETE FROM auth.dispositivos WHERE token = $1", d.Token); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `INSERT INTO auth.dispositivos (clerk_id, sesion_hash, plataforma, token, entorno, modelo, version_app)
			VALUES ($1, $2, $3, $4, $5, $6, $7)`, ses.ClerkID, ses.Hash, d.Plataforma, d.Token, d.Entorno, d.Modelo, d.Version)
		return err
	})
	if err != nil {
		return err
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}

// baja borra los teléfonos de esta sesión (la app lo llama al salir; cerrar la sesión también).
func (s *Servicio) baja(w http.ResponseWriter, r *http.Request) error {
	ses, err := sesionPropia(r.Context())
	if err != nil {
		return err
	}
	ctx := r.Context()
	err = s.Pool.EnTx(ctx, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, "DELETE FROM auth.dispositivos WHERE sesion_hash = $1", ses.Hash)
		return err
	})
	if err != nil {
		return err
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}

// TokenFCM dice si el texto tiene forma de token de registro de FCM (el de un teléfono Android).
func TokenFCM(t string) bool { return len(t) <= 4096 && tokenFCM.MatchString(t) }
