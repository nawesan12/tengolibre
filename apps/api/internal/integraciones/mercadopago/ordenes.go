package mercadopago

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Cobros con la API de Orders: Checkout Pro vía Orders (POST /v1/orders, type online, processing_mode
// manual), que devuelve un checkout_url al que se redirige al comprador. Reemplaza a la API de
// Preferencias (/checkout/preferences), que Mercado Pago discontinúa. Guía vigente:
// https://www.mercadopago.com.ar/developers/es/docs/checkout-pro-orders/create-order

// Los montos de la API de Orders son texto con dos decimales ("4000.00").
func Importe(pesos int64) string { return strconv.FormatInt(pesos, 10) + ".00" }

// LeerImporte lee un importe de la API ("4000.00") en pesos; vacío o inválido es 0.
func LeerImporte(s string) float64 {
	f, _ := strconv.ParseFloat(strings.TrimSpace(s), 64)
	return f
}

// ItemOrden es un ítem de la orden. Todos los importes van como texto.
type ItemOrden struct {
	Title     string `json:"title"`
	Quantity  int    `json:"quantity"`
	UnitPrice string `json:"unit_price"`
}

// ConfigOnline son las URLs de retorno de Checkout Pro (config.online).
type ConfigOnline struct {
	SuccessURL string `json:"success_url,omitempty"`
	FailureURL string `json:"failure_url,omitempty"`
	PendingURL string `json:"pending_url,omitempty"`
	AutoReturn string `json:"auto_return,omitempty"`
}

// MediosOrden restringe los medios de pago (config.payment_method).
type MediosOrden struct {
	MaxInstallments int `json:"max_installments,omitempty"`
}

// ConfigOrden es el objeto config de la orden.
type ConfigOrden struct {
	Online        *ConfigOnline `json:"online,omitempty"`
	PaymentMethod *MediosOrden  `json:"payment_method,omitempty"`
	StatementDesc string        `json:"statement_descriptor,omitempty"`
}

// Orden es el cuerpo de POST /v1/orders para Checkout Pro.
type Orden struct {
	Type              string `json:"type"`
	ProcessingMode    string `json:"processing_mode"`
	TotalAmount       string `json:"total_amount"`
	ExternalReference string `json:"external_reference"`
	// ExpirationTime es una duración ISO 8601 desde la creación ("PT15M"): pasado ese tiempo la orden
	// vence y ya no se puede pagar.
	ExpirationTime string       `json:"expiration_time,omitempty"`
	Items          []ItemOrden  `json:"items"`
	Config         *ConfigOrden `json:"config,omitempty"`
}

// NuevaOrden de Checkout Pro: una orden online de modo manual con un solo ítem.
func NuevaOrden(referencia, titulo string, pesos int64) Orden {
	return Orden{Type: "online", ProcessingMode: "manual", TotalAmount: Importe(pesos), ExternalReference: referencia,
		Items: []ItemOrden{{Title: titulo, Quantity: 1, UnitPrice: Importe(pesos)}}}
}

// Duracion ISO 8601 en minutos, redondeada para arriba y de al menos uno ("PT15M").
func Duracion(d time.Duration) string {
	m := int64((d + time.Minute - 1) / time.Minute)
	return "PT" + strconv.FormatInt(max(m, 1), 10) + "M"
}

// OrdenCreada es lo que interesa de la respuesta de POST /v1/orders.
type OrdenCreada struct {
	ID          string
	CheckoutURL string
}

// CrearOrden con el token del vendedor. clave es la X-Idempotency-Key: un reintento con la misma no
// crea otra orden.
func (c *Cliente) CrearOrden(ctx context.Context, token, clave string, o Orden) (OrdenCreada, error) {
	var r struct {
		ID          string `json:"id"`
		CheckoutURL string `json:"checkout_url"`
	}
	if err := c.JSON(ctx, http.MethodPost, "/v1/orders", token, clave, o, &r); err != nil {
		return OrdenCreada{}, err
	}
	if r.ID == "" || r.CheckoutURL == "" {
		return OrdenCreada{}, errors.New("mercadopago: la orden no trajo id o checkout_url")
	}
	return OrdenCreada{ID: r.ID, CheckoutURL: r.CheckoutURL}, nil
}

// TransaccionPago es un pago (un intento) de una orden: transactions.payments[].
type TransaccionPago struct {
	ID            string `json:"id"`
	Amount        string `json:"amount"`
	PaidAmount    string `json:"paid_amount"`
	AttemptNumber int    `json:"attempt_number"`
	Status        string `json:"status"`
	StatusDetail  string `json:"status_detail"`
	// Reference.ID es el id numérico del pago en la API de Payments.
	Reference struct {
		ID string `json:"id"`
	} `json:"reference"`
}

// IDPago es el id numérico del pago (0 si la transacción todavía no lo tiene).
func (t TransaccionPago) IDPago() int64 {
	n, _ := strconv.ParseInt(strings.TrimSpace(t.Reference.ID), 10, 64)
	return n
}

// OrdenLeida es lo que interesa de GET /v1/orders/{id}.
type OrdenLeida struct {
	ID                string `json:"id"`
	Status            string `json:"status"`
	StatusDetail      string `json:"status_detail"`
	ExternalReference string `json:"external_reference"`
	TotalAmount       string `json:"total_amount"`
	TotalPaidAmount   string `json:"total_paid_amount"`
	Transactions      struct {
		Payments []TransaccionPago `json:"payments"`
	} `json:"transactions"`
}

// LeerOrden con el token del vendedor: el aviso no es la fuente de verdad.
func (c *Cliente) LeerOrden(ctx context.Context, token, id string) (OrdenLeida, error) {
	var o OrdenLeida
	err := c.JSON(ctx, http.MethodGet, "/v1/orders/"+url.PathEscape(id), token, "", nil, &o)
	return o, err
}

// Relevante es la transacción que cuenta de la orden: la que se acreditó (o se devolvió) si hay, y si
// no el último intento. ok es false si la orden todavía no tiene ninguna.
func (o OrdenLeida) Relevante() (t TransaccionPago, ok bool) {
	for _, p := range o.Transactions.Payments {
		switch p.Status {
		case "processed", "refunded", "charged_back":
			return p, true
		}
		if !ok || p.AttemptNumber >= t.AttemptNumber {
			t, ok = p, true
		}
	}
	return t, ok
}

// DevolverOrden reembolsa una orden: POST /v1/orders/{id}/refund. Total (monto 0) va sin cuerpo; uno
// parcial lleva la transacción y el importe. clave hace idempotente el reintento.
func (c *Cliente) DevolverOrden(ctx context.Context, token, orden, transaccion string, monto int64, clave string) error {
	var cuerpo any
	if monto > 0 {
		cuerpo = map[string]any{"transactions": []map[string]string{{"id": transaccion, "amount": Importe(monto)}}}
	}
	return c.JSON(ctx, http.MethodPost, fmt.Sprintf("/v1/orders/%s/refund", url.PathEscape(orden)), token, "devolucion:"+clave, cuerpo, nil)
}
