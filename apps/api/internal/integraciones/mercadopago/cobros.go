package mercadopago

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

// Item de una preferencia.
type Item struct {
	Title      string `json:"title"`
	Quantity   int    `json:"quantity"`
	UnitPrice  int64  `json:"unit_price"`
	CurrencyID string `json:"currency_id"`
}

// Preferencia de Checkout Pro.
type Preferencia struct {
	Items             []Item            `json:"items"`
	ExternalReference string            `json:"external_reference"`
	BackURLs          map[string]string `json:"back_urls,omitempty"`
	AutoReturn        string            `json:"auto_return,omitempty"`
	PaymentMethods    map[string]any    `json:"payment_methods,omitempty"`
	StatementDesc     string            `json:"statement_descriptor,omitempty"`
	// La seña de un turno vence: después, Mercado Pago ya no deja pagarla.
	Expires          bool   `json:"expires,omitempty"`
	ExpirationDateTo string `json:"expiration_date_to,omitempty"`
}

// FormatoVencimiento de expiration_date_to.
const FormatoVencimiento = "2006-01-02T15:04:05.000-07:00"

// CrearPreferencia con el token del vendedor y devuelve el init_point. La referencia es la clave de
// idempotencia: un reintento no crea dos.
func (c *Cliente) CrearPreferencia(ctx context.Context, token string, p Preferencia) (string, error) {
	var r struct {
		ID        string `json:"id"`
		InitPoint string `json:"init_point"`
	}
	// /checkout/preferences, sin /v1 (la guía de Checkout Pro). Siempre init_point.
	if err := c.JSON(ctx, http.MethodPost, "/checkout/preferences", token, "pref:"+p.ExternalReference, p, &r); err != nil {
		return "", err
	}
	if r.InitPoint == "" {
		return "", errors.New("mercadopago: la preferencia no trajo init_point")
	}
	return r.InitPoint, nil
}

// Pago es lo que interesa de GET /v1/payments/{id}.
type Pago struct {
	ID                 int64   `json:"id"`
	Status             string  `json:"status"`
	StatusDetail       string  `json:"status_detail"`
	ExternalReference  string  `json:"external_reference"`
	TransactionAmount  float64 `json:"transaction_amount"`
	TransactionRefunds float64 `json:"transaction_amount_refunded"`
}

// Pesos del monto (los de TengoLibre son enteros).
func Pesos(f float64) int64 { return int64(f + 0.5) }

// LeerPago con el token del vendedor: el aviso no es la fuente de verdad.
func (c *Cliente) LeerPago(ctx context.Context, token string, id int64) (Pago, error) {
	var p Pago
	err := c.JSON(ctx, http.MethodGet, "/v1/payments/"+strconv.FormatInt(id, 10), token, "", nil, &p)
	return p, err
}

// Devolver un pago (total si monto es 0). clave hace idempotente el reintento.
func (c *Cliente) Devolver(ctx context.Context, token string, pago, monto int64, clave string) error {
	cuerpo := map[string]any{}
	if monto > 0 {
		cuerpo["amount"] = monto
	}
	return c.JSON(ctx, http.MethodPost, fmt.Sprintf("/v1/payments/%d/refunds", pago), token, "devolucion:"+clave, cuerpo, nil)
}

// Vencimiento en el formato de expiration_date_to.
func Vencimiento(t time.Time) string { return t.Format(FormatoVencimiento) }
