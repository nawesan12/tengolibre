package mercadopago

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
)

// Pago es lo que interesa de GET /v1/payments/{id}: los pagos de preferencias viejas, que se
// siguen conciliando mientras existan.
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
