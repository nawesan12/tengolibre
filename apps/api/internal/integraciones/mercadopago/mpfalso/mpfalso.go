// Package mpfalso es un Mercado Pago de mentira para los tests: un httptest.Server con lo que usa
// TengoLibre (OAuth, preferencias, pagos, devoluciones, preapprovals y cobros autorizados). Los tests
// cargan los pagos a mano y mandan avisos firmados como los de verdad. Nunca se llama a la API real.
package mpfalso

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/nawesan12/tengolibre/apps/api/internal/integraciones/mercadopago"
)

// Secreto de los webhooks del falso.
const Secreto = "secreto-webhooks"

// TokenPlataforma: el token de la cuenta de TengoLibre (suscripciones).
const TokenPlataforma = "APP_USR-plataforma"

// MP falso.
type MP struct {
	*httptest.Server
	mu sync.Mutex
	// Usuario es el vendedor que devuelve el próximo canje de OAuth.
	Usuario int64
	// Preferencias creadas, por external_reference.
	Preferencias map[string]mercadopago.Preferencia
	// TokenDePreferencia: con qué token se creó cada una (para ver que se cobra con el del negocio).
	TokenDePreferencia map[string]string
	Pagos              map[int64]mercadopago.Pago
	// Devoluciones pedidas: pago → montos (0 es total).
	Devoluciones  map[int64][]int64
	Preaprobacion map[string]*mercadopago.Preaprobacion
	Cobros        map[string]mercadopago.CobroAutorizado
	// Renovaciones de tokens pedidas.
	Renovaciones int
	// FallarRenovacion hace que el refresh responda 400.
	FallarRenovacion bool
	n                int
}

// Nuevo levanta el falso y lo cierra al terminar el test.
func Nuevo(t testing.TB) *MP {
	m := &MP{Usuario: 777, Preferencias: map[string]mercadopago.Preferencia{}, TokenDePreferencia: map[string]string{},
		Pagos: map[int64]mercadopago.Pago{}, Devoluciones: map[int64][]int64{}, Preaprobacion: map[string]*mercadopago.Preaprobacion{},
		Cobros: map[string]mercadopago.CobroAutorizado{}}
	m.Server = httptest.NewServer(http.HandlerFunc(m.servir))
	t.Cleanup(m.Close)
	return m
}

// Cliente apuntado al falso.
func (m *MP) Cliente() *mercadopago.Cliente {
	return &mercadopago.Cliente{Base: m.URL, Autorizar: m.URL + "/authorization"}
}

func escribir(w http.ResponseWriter, estado int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(estado)
	_ = json.NewEncoder(w).Encode(v)
}

func token(r *http.Request) string {
	return strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
}

func (m *MP) servir(w http.ResponseWriter, r *http.Request) {
	m.mu.Lock()
	defer m.mu.Unlock()
	ruta := r.URL.Path
	switch {
	case r.Method == "POST" && ruta == "/oauth/token":
		_ = r.ParseForm()
		switch r.Form.Get("grant_type") {
		case "authorization_code":
			if r.Form.Get("code") == "" || r.Form.Get("code_verifier") == "" {
				escribir(w, 400, map[string]string{"error": "invalid_grant"})
				return
			}
			escribir(w, 200, mercadopago.Tokens{AccessToken: fmt.Sprintf("APP_USR-%d", m.Usuario), RefreshToken: "TG-renovar",
				ExpiresIn: 180 * 24 * 3600, Scope: "offline_access read write", UserID: m.Usuario})
		case "refresh_token":
			if m.FallarRenovacion {
				escribir(w, 400, map[string]string{"error": "invalid_grant"})
				return
			}
			m.Renovaciones++
			escribir(w, 200, mercadopago.Tokens{AccessToken: fmt.Sprintf("APP_USR-renovado-%d", m.Renovaciones),
				RefreshToken: "TG-rotado", ExpiresIn: 180 * 24 * 3600})
		default:
			escribir(w, 400, nil)
		}
	case r.Method == "POST" && ruta == "/checkout/preferences":
		var p mercadopago.Preferencia
		_ = json.NewDecoder(r.Body).Decode(&p)
		m.Preferencias[p.ExternalReference] = p
		m.TokenDePreferencia[p.ExternalReference] = token(r)
		escribir(w, 201, map[string]string{"id": "pref-" + p.ExternalReference, "init_point": "https://mp.falso/checkout/" + p.ExternalReference})
	case r.Method == "GET" && strings.HasPrefix(ruta, "/v1/payments/"):
		id, _ := strconv.ParseInt(strings.TrimPrefix(ruta, "/v1/payments/"), 10, 64)
		p, ok := m.Pagos[id]
		if !ok {
			escribir(w, 404, nil)
			return
		}
		escribir(w, 200, p)
	case r.Method == "POST" && strings.HasPrefix(ruta, "/v1/payments/") && strings.HasSuffix(ruta, "/refunds"):
		id, _ := strconv.ParseInt(strings.TrimSuffix(strings.TrimPrefix(ruta, "/v1/payments/"), "/refunds"), 10, 64)
		p, ok := m.Pagos[id]
		if !ok {
			escribir(w, 404, nil)
			return
		}
		var b struct {
			Amount int64 `json:"amount"`
		}
		_ = json.NewDecoder(r.Body).Decode(&b)
		m.Devoluciones[id] = append(m.Devoluciones[id], b.Amount)
		monto := float64(b.Amount)
		if b.Amount == 0 {
			monto = p.TransactionAmount - p.TransactionRefunds
		}
		p.TransactionRefunds += monto
		if p.TransactionRefunds >= p.TransactionAmount {
			p.Status = "refunded"
		}
		m.Pagos[id] = p
		escribir(w, 201, map[string]any{"id": len(m.Devoluciones[id]), "amount": monto})
	case r.Method == "POST" && ruta == "/preapproval":
		if token(r) != TokenPlataforma {
			escribir(w, 401, nil)
			return
		}
		var b struct {
			ExternalReference string `json:"external_reference"`
			PayerEmail        string `json:"payer_email"`
		}
		_ = json.NewDecoder(r.Body).Decode(&b)
		m.n++
		id := fmt.Sprintf("pre%04d", m.n)
		m.Preaprobacion[id] = &mercadopago.Preaprobacion{ID: id, Status: "pending", InitPoint: "https://mp.falso/suscripcion/" + id,
			ExternalReference: b.ExternalReference, PayerEmail: b.PayerEmail}
		escribir(w, 201, m.Preaprobacion[id])
	case strings.HasPrefix(ruta, "/preapproval/"):
		p, ok := m.Preaprobacion[strings.TrimPrefix(ruta, "/preapproval/")]
		if !ok || token(r) != TokenPlataforma {
			escribir(w, 404, nil)
			return
		}
		if r.Method == "PUT" {
			var b struct {
				Status string `json:"status"`
			}
			_ = json.NewDecoder(r.Body).Decode(&b)
			p.Status = b.Status
		}
		escribir(w, 200, p)
	case r.Method == "GET" && strings.HasPrefix(ruta, "/authorized_payments/"):
		c, ok := m.Cobros[strings.TrimPrefix(ruta, "/authorized_payments/")]
		if !ok {
			escribir(w, 404, nil)
			return
		}
		escribir(w, 200, c)
	default:
		escribir(w, 404, map[string]string{"error": "ruta del falso: " + r.Method + " " + ruta})
	}
}

// Pagar carga un pago del comprador contra la preferencia de referencia.
func (m *MP) Pagar(id int64, referencia, estado string, monto int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Pagos[id] = mercadopago.Pago{ID: id, Status: estado, ExternalReference: referencia, TransactionAmount: float64(monto)}
}

// Autorizar cambia el estado de un preapproval (lo que pasa cuando la persona carga la tarjeta).
func (m *MP) Autorizar(id, estado string, proximo *time.Time) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if p, ok := m.Preaprobacion[id]; ok {
		p.Status, p.NextPaymentDate = estado, proximo
	}
}

// Aviso firmado como los de Mercado Pago, listo para mandar al handler.
func Aviso(tipo, id string, usuario int64, ahora time.Time) *http.Request {
	cuerpo := fmt.Sprintf(`{"type":%q,"action":"updated","user_id":%d,"data":{"id":%q}}`, tipo, usuario, id)
	r := httptest.NewRequest("POST", "/v1/webhooks/mercadopago?data.id="+id+"&type="+tipo, strings.NewReader(cuerpo))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("x-request-id", "req-"+id)
	r.Header.Set("x-signature", mercadopago.Firmar(Secreto, "req-"+id, id, ahora.UnixMilli()))
	return r
}

// Avisar manda el aviso al handler y devuelve el estado.
func Avisar(h http.Handler, tipo, id string, usuario int64, ahora time.Time) int {
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, Aviso(tipo, id, usuario, ahora))
	return rec.Code
}
