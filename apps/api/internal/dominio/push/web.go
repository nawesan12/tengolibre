package push

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/problema"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/web"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/webpush"
)

// Push web (Propuesta 10): el panel abierto o instalado en el navegador recibe la campanita como
// los teléfonos, con Web Push (VAPID) en vez de APNs o FCM. Se pide desde la tarjeta «Enterate en el
// momento» de Inicio, nunca con el cartel del navegador de entrada: el front llama a
// Notification.requestPermission() recién cuando la persona toca «Activar», se suscribe con la
// clave pública de GET /v1/push/vapid y manda la suscripción a POST /v1/yo/suscripciones-web.

// SuscripcionWeb es lo que manda el front: PushSubscription.toJSON() tal cual (endpoint y keys) o
// con las claves sueltas, más cómo se ve el aparato en la lista.
type SuscripcionWeb struct {
	Endpoint string `json:"endpoint"`
	// ExpirationTime viene en toJSON(); no se usa (el servicio avisa con 404/410 cuando vence).
	ExpirationTime *float64 `json:"expirationTime"`
	Keys           *struct {
		P256dh string `json:"p256dh"`
		Auth   string `json:"auth"`
	} `json:"keys"`
	P256dh  string `json:"p256dh"`
	Auth    string `json:"auth"`
	Aparato string `json:"aparato"`
}

// EstadoWeb: lo que necesita la tarjeta de Inicio. Clave vacía: el push web está apagado.
type EstadoWeb struct {
	Clave     string `json:"clave"`
	Suscripta bool   `json:"suscripta"`
}

func (s *Servicio) rutasWeb(mux *http.ServeMux) {
	// La clave pública no es secreta: la pide también el service worker al renovar la suscripción.
	mux.HandleFunc("GET /v1/push/vapid", web.Envolver(s.vapid))
	mux.HandleFunc("GET /v1/yo/suscripciones-web", s.Sesion.Persona(web.Envolver(s.estadoWeb)))
	mux.HandleFunc("POST /v1/yo/suscripciones-web", s.Sesion.Persona(web.Envolver(s.suscribirWeb)))
	mux.HandleFunc("DELETE /v1/yo/suscripciones-web", s.Sesion.Persona(web.Envolver(s.bajaWeb)))
}

func (s *Servicio) vapid(w http.ResponseWriter, r *http.Request) error {
	web.JSON(w, http.StatusOK, map[string]any{"clave": s.ClaveVAPID, "activo": s.ClaveVAPID != ""})
	return nil
}

func (s *Servicio) estadoWeb(w http.ResponseWriter, r *http.Request) error {
	ses, err := sesionPropia(r.Context())
	if err != nil {
		return err
	}
	var n int
	err = s.Pool.EnTx(r.Context(), func(tx pgx.Tx) error {
		return tx.QueryRow(r.Context(), "SELECT count(*) FROM auth.suscripciones_web WHERE sesion_hash = $1", ses.Hash).Scan(&n)
	})
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusOK, EstadoWeb{Clave: s.ClaveVAPID, Suscripta: n > 0})
	return nil
}

// NormalizarWeb valida la suscripción (un servicio de push conocido, con sus claves).
func NormalizarWeb(d SuscripcionWeb) (webpush.Suscripcion, string, error) {
	p, a := d.P256dh, d.Auth
	if d.Keys != nil {
		p, a = d.Keys.P256dh, d.Keys.Auth
	}
	n, err := webpush.Normalizar(webpush.Suscripcion{Endpoint: d.Endpoint, P256dh: p, Auth: a})
	if err != nil {
		return n, "", problema.Invalido("La suscripción del navegador no es válida.",
			problema.Campo{Ruta: "endpoint", Mensaje: "Tiene que ser de un servicio de push conocido, con sus claves."})
	}
	return n, limpiar(d.Aparato, 60), nil
}

// suscribirWeb ata la suscripción a la sesión. Si el endpoint ya estaba con otra (volvió a entrar),
// se mueve a esta.
func (s *Servicio) suscribirWeb(w http.ResponseWriter, r *http.Request) error {
	ses, err := sesionPropia(r.Context())
	if err != nil {
		return err
	}
	if s.ClaveVAPID == "" {
		return problema.Conflicto("push_apagado", "Los avisos en el navegador todavía no están disponibles.")
	}
	var d SuscripcionWeb
	if err := web.Leer(w, r, &d); err != nil {
		return err
	}
	n, aparato, err := NormalizarWeb(d)
	if err != nil {
		return err
	}
	ctx := r.Context()
	err = s.Pool.EnTx(ctx, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, "DELETE FROM auth.suscripciones_web WHERE endpoint = $1", n.Endpoint); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `INSERT INTO auth.suscripciones_web (clerk_id, sesion_hash, endpoint, p256dh, auth, aparato)
			VALUES ($1, $2, $3, $4, $5, $6)`, ses.ClerkID, ses.Hash, n.Endpoint, n.P256dh, n.Auth, aparato)
		return err
	})
	if err != nil {
		return err
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}

// bajaWeb borra las suscripciones de esta sesión (o sólo la del endpoint, si viene).
func (s *Servicio) bajaWeb(w http.ResponseWriter, r *http.Request) error {
	ses, err := sesionPropia(r.Context())
	if err != nil {
		return err
	}
	var in struct {
		Endpoint string `json:"endpoint"`
	}
	if r.ContentLength > 0 {
		if err := web.Leer(w, r, &in); err != nil {
			return err
		}
	}
	ctx := r.Context()
	err = s.Pool.EnTx(ctx, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, "DELETE FROM auth.suscripciones_web WHERE sesion_hash = $1 AND ($2 = '' OR endpoint = $2)",
			ses.Hash, strings.TrimSpace(in.Endpoint))
		return err
	})
	if err != nil {
		return err
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}

// URLPanel traduce la ruta de la app a la página del panel web que abre el aviso.
func URLPanel(ruta string) string {
	switch {
	case strings.HasPrefix(ruta, "turnos/"):
		return "/panel/turnos/" + strings.TrimPrefix(ruta, "turnos/")
	case ruta == "notificaciones":
		return "/panel/notificaciones"
	}
	return "/panel"
}

// PayloadWeb arma lo que lee el service worker del panel: título, cuerpo, la página que abre, el
// negocio (para cambiar a ese antes de abrirla), el tipo de la campanita y la etiqueta (dos avisos
// con la misma se ven como uno).
func PayloadWeb(a Aviso) []byte {
	b, _ := json.Marshal(struct {
		Title   string `json:"title"`
		Body    string `json:"body"`
		URL     string `json:"url"`
		Negocio string `json:"negocio"`
		Tipo    string `json:"tipo,omitempty"`
		Tag     string `json:"tag,omitempty"`
	}{a.Titulo, a.Cuerpo, URLPanel(a.Ruta), a.Negocio, a.Tipo, a.Colapso})
	return b
}

// esVencida: la suscripción ya no sirve (el servicio la dio de baja, o no es de un servicio
// permitido): se borra.
func esVencida(err error) bool {
	return errors.Is(err, webpush.ErrVencida) || errors.Is(err, webpush.ErrInvalida)
}
