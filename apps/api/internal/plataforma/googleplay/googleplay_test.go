package googleplay

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestConsultaYReusaElToken(t *testing.T) {
	clave, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	var tokens, consultas atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/token":
			tokens.Add(1)
			_ = r.ParseForm()
			if r.Form.Get("grant_type") != "urn:ietf:params:oauth:grant-type:jwt-bearer" || strings.Count(r.Form.Get("assertion"), ".") != 2 {
				http.Error(w, "mal", 400)
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "acceso", "expires_in": 3600})
		case strings.HasSuffix(r.URL.Path, "/tokens/no-existe"):
			http.Error(w, "{}", 404)
		case r.URL.Path == "/androidpublisher/v3/applications/ar.com.tengolibre.app/purchases/subscriptionsv2/tokens/tok-1":
			consultas.Add(1)
			if r.Header.Get("authorization") != "Bearer acceso" {
				http.Error(w, "{}", 401)
				return
			}
			_, _ = w.Write([]byte(`{"subscriptionState":"SUBSCRIPTION_STATE_ACTIVE","startTime":"2026-11-05T10:00:00Z",
				"externalAccountIdentifiers":{"obfuscatedExternalAccountId":"negocio-1"},
				"lineItems":[{"productId":"ar.com.tengolibre.completo.mensual","expiryTime":"2026-12-05T10:00:00.123Z","autoRenewingPlan":{"autoRenewEnabled":true}}]}`))
		default:
			http.Error(w, r.URL.Path, 500)
		}
	}))
	defer srv.Close()
	c := &Cliente{Email: "play@x.iam.gserviceaccount.com", URLToken: srv.URL + "/token", Clave: clave, Host: srv.URL, HTTP: srv.Client(),
		Ahora: func() time.Time { return time.Date(2026, 11, 5, 12, 0, 0, 0, time.UTC) }}
	for range 2 {
		s, err := c.Consultar(context.Background(), "ar.com.tengolibre.app", "tok-1")
		if err != nil {
			t.Fatal(err)
		}
		if s.Estado != "SUBSCRIPTION_STATE_ACTIVE" || s.Producto() != "ar.com.tengolibre.completo.mensual" || s.Identificadores.Cuenta != "negocio-1" ||
			!s.Renueva() || s.Vence().Day() != 5 || s.Vence().Month() != 12 {
			t.Fatalf("suscripción: %+v", s)
		}
	}
	if tokens.Load() != 1 || consultas.Load() != 2 {
		t.Fatalf("tokens %d, consultas %d", tokens.Load(), consultas.Load())
	}
	if _, err := c.Consultar(context.Background(), "ar.com.tengolibre.app", "no-existe"); !errors.Is(err, ErrNoEncontrada) {
		t.Fatalf("no existe: %v", err)
	}
	var nada *Cliente
	if _, err := nada.Consultar(context.Background(), "x", "y"); err == nil {
		t.Fatal("un cliente nil no consulta")
	}
	if c, err := Desde(""); c != nil || err != nil {
		t.Fatal("sin ruta: apagado")
	}
}
