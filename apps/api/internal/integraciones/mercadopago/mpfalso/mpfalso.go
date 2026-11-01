// Package mpfalso es un Mercado Pago de mentira para los tests: un httptest.Server con lo que usa
// TengoLibre (OAuth, órdenes de Checkout Pro, pagos de preferencias viejas, devoluciones, preapprovals y
// cobros autorizados). Los tests
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
	// Ordenes creadas, por external_reference (la última de cada una) y por id.
	Ordenes   map[string]*Orden
	OrdenesID map[string]*Orden
	// Pagos de preferencias viejas (el tópico payment sigue andando mientras existan).
	Pagos map[int64]mercadopago.Pago
	// DevolucionesOrden pedidas: orden → reembolsos.
	DevolucionesOrden map[string][]ReembolsoOrden
	// Devoluciones pedidas: pago → montos (0 es total).
	Devoluciones  map[int64][]int64
	Preaprobacion map[string]*mercadopago.Preaprobacion
	Cobros        map[string]mercadopago.CobroAutorizado
	// Montos: el último precio que se le puso a cada preapproval con un PUT (cambio de plan).
	Montos map[string]int64
	// Cuenta: lo que devuelve /users/me con el token del vendedor (el nombre de la cuenta).
	Cuenta map[string]any
	// Renovaciones de tokens pedidas.
	Renovaciones int
	// FallarRenovacion hace que el refresh responda 400.
	FallarRenovacion bool
	n                int
	clavesOrden      map[string]*Orden
}

// Orden es una orden creada con POST /v1/orders.
type Orden struct {
	ID string
	// Token con el que se creó (para ver que se cobra con el del negocio) y X-Idempotency-Key.
	Token, Clave string
	Pedido       mercadopago.Orden
	Estado       string
	Detalle      string
	Pagos        []mercadopago.TransaccionPago
	URL          string
}

// ReembolsoOrden pedido a POST /v1/orders/{id}/refund: Transaccion vacía y Monto 0 es total.
type ReembolsoOrden struct {
	Transaccion string
	Monto       int64
}

// Nuevo levanta el falso y lo cierra al terminar el test.
func Nuevo(t testing.TB) *MP {
	m := &MP{Usuario: 777, Ordenes: map[string]*Orden{}, OrdenesID: map[string]*Orden{}, clavesOrden: map[string]*Orden{},
		DevolucionesOrden: map[string][]ReembolsoOrden{}, Pagos: map[int64]mercadopago.Pago{}, Devoluciones: map[int64][]int64{}, Preaprobacion: map[string]*mercadopago.Preaprobacion{},
		Cobros: map[string]mercadopago.CobroAutorizado{}, Montos: map[string]int64{},
		Cuenta: map[string]any{"id": 777, "nickname": "ELCORTE", "first_name": "Carla", "last_name": "Gómez",
			"company": map[string]any{"brand_name": "El Corte", "corporate_name": "EL CORTE SRL"}}}
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
	case r.Method == "GET" && ruta == "/users/me":
		if !strings.HasPrefix(token(r), "APP_USR-") || token(r) == TokenPlataforma {
			escribir(w, 401, nil)
			return
		}
		escribir(w, 200, m.Cuenta)
	case r.Method == "POST" && ruta == "/v1/orders":
		m.crearOrden(w, r)
	case strings.HasPrefix(ruta, "/v1/orders/"):
		m.orden(w, r, strings.TrimPrefix(ruta, "/v1/orders/"))
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
				Status        string `json:"status"`
				AutoRecurring *struct {
					TransactionAmount int64 `json:"transaction_amount"`
				} `json:"auto_recurring"`
			}
			_ = json.NewDecoder(r.Body).Decode(&b)
			if b.Status != "" {
				p.Status = b.Status
			}
			if b.AutoRecurring != nil {
				m.Montos[p.ID] = b.AutoRecurring.TransactionAmount
			}
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

// Pagar carga un pago de la API de Payments (los de una preferencia vieja) contra la referencia.
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

func montoOrden(s string) (int64, bool) {
	pesos, dec, ok := strings.Cut(s, ".")
	n, err := strconv.ParseInt(pesos, 10, 64)
	return n, ok && dec == "00" && err == nil && n > 0
}

// crearOrden valida lo que la API de verdad exige de una orden de Checkout Pro.
func (m *MP) crearOrden(w http.ResponseWriter, r *http.Request) {
	var p mercadopago.Orden
	_ = json.NewDecoder(r.Body).Decode(&p)
	clave := r.Header.Get("X-Idempotency-Key")
	total, okTotal := montoOrden(p.TotalAmount)
	var suma int64
	okItems := len(p.Items) > 0
	for _, it := range p.Items {
		n, ok := montoOrden(it.UnitPrice)
		okItems = okItems && ok && it.Title != "" && it.Quantity > 0
		suma += n * int64(it.Quantity)
	}
	if clave == "" || p.Type != "online" || p.ProcessingMode != "manual" || !okTotal || !okItems || suma != total || p.ExternalReference == "" ||
		!strings.HasPrefix(p.ExpirationTime, "PT") || p.Config == nil || p.Config.Online == nil || p.Config.Online.SuccessURL == "" {
		escribir(w, 400, map[string]string{"error": "orden inválida en el falso"})
		return
	}
	if o, ok := m.clavesOrden[clave]; ok {
		escribir(w, 200, o.respuesta())
		return
	}
	m.n++
	o := &Orden{ID: fmt.Sprintf("ORD%024d", m.n), Token: token(r), Clave: clave, Pedido: p, Estado: "created", Detalle: "created"}
	o.URL = "https://mp.falso/checkout/" + p.ExternalReference
	if _, hay := m.Ordenes[p.ExternalReference]; hay {
		o.URL += "?order_id=" + o.ID
	}
	m.Ordenes[p.ExternalReference], m.OrdenesID[o.ID], m.clavesOrden[clave] = o, o, o
	escribir(w, 201, o.respuesta())
}

func (o *Orden) respuesta() map[string]any {
	var pagado int64
	for _, t := range o.Pagos {
		if t.Status == "processed" {
			n, _ := montoOrden(t.PaidAmount)
			pagado += n
		}
	}
	return map[string]any{"id": o.ID, "type": "online", "processing_mode": "manual", "status": o.Estado, "status_detail": o.Detalle,
		"external_reference": o.Pedido.ExternalReference, "total_amount": o.Pedido.TotalAmount, "total_paid_amount": mercadopago.Importe(pagado),
		"checkout_url": o.URL, "transactions": map[string]any{"payments": o.Pagos}}
}

// orden atiende GET /v1/orders/{id} y POST /v1/orders/{id}/refund con el token con que se creó.
func (m *MP) orden(w http.ResponseWriter, r *http.Request, resto string) {
	id, accion, _ := strings.Cut(resto, "/")
	o, ok := m.OrdenesID[id]
	if !ok || o.Token != token(r) {
		escribir(w, 404, nil)
		return
	}
	switch {
	case r.Method == "GET" && accion == "":
		escribir(w, 200, o.respuesta())
	case r.Method == "POST" && accion == "refund":
		var b struct {
			Transactions []struct {
				ID     string `json:"id"`
				Amount string `json:"amount"`
			} `json:"transactions"`
		}
		if r.ContentLength > 0 {
			_ = json.NewDecoder(r.Body).Decode(&b)
		}
		if r.Header.Get("X-Idempotency-Key") == "" || len(o.Pagos) == 0 {
			escribir(w, 400, nil)
			return
		}
		t := &o.Pagos[len(o.Pagos)-1]
		var monto int64
		pagado, _ := montoOrden(t.PaidAmount)
		if len(b.Transactions) == 0 {
			m.DevolucionesOrden[id] = append(m.DevolucionesOrden[id], ReembolsoOrden{})
			monto = pagado
		} else {
			x := b.Transactions[0]
			n, okMonto := montoOrden(x.Amount)
			if x.ID != t.ID || !okMonto || n > pagado {
				escribir(w, 400, map[string]string{"error": "reembolso inválido en el falso"})
				return
			}
			m.DevolucionesOrden[id] = append(m.DevolucionesOrden[id], ReembolsoOrden{Transaccion: x.ID, Monto: n})
			monto = n
		}
		if resta := pagado - monto; resta > 0 {
			t.PaidAmount, t.StatusDetail = mercadopago.Importe(resta), "partially_refunded"
			o.Detalle = "partially_refunded"
		} else {
			t.PaidAmount, t.Status, t.StatusDetail = "0.00", "refunded", "refunded"
			o.Estado, o.Detalle = "refunded", "refunded"
		}
		escribir(w, 201, o.respuesta())
	default:
		escribir(w, 404, nil)
	}
}

// PagarOrden carga un intento de pago contra la última orden de la referencia: estado y detalle son
// los de la transacción (processed/accredited, failed/rejected_by_issuer…). idPago es el id numérico
// que Mercado Pago pone en payments[].reference.id. Devuelve el id de la orden, para mandar el aviso.
func (m *MP) PagarOrden(referencia, estado, detalle string, monto, idPago int64) string {
	m.mu.Lock()
	defer m.mu.Unlock()
	o := m.Ordenes[referencia]
	if o == nil {
		panic("mpfalso: no hay orden para " + referencia)
	}
	t := mercadopago.TransaccionPago{ID: fmt.Sprintf("PAY%s%d", o.ID[3:], len(o.Pagos)+1), Amount: mercadopago.Importe(monto), PaidAmount: mercadopago.Importe(monto),
		AttemptNumber: len(o.Pagos) + 1, Status: estado, StatusDetail: detalle}
	t.Reference.ID = strconv.FormatInt(idPago, 10)
	if estado != "processed" {
		t.PaidAmount = "0.00"
	}
	o.Pagos = append(o.Pagos, t)
	o.Estado, o.Detalle = estado, detalle
	return o.ID
}

// Expirar vence la orden sin que nadie la pague.
func (m *MP) Expirar(orden string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if o := m.OrdenesID[orden]; o != nil {
		o.Estado, o.Detalle = "canceled", "canceled"
	}
}
