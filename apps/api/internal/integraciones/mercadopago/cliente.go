// Package mercadopago habla con la API de Mercado Pago: el OAuth con que cada negocio conecta su
// cuenta (marketplace, con PKCE), las preferencias de Checkout Pro, los pagos y sus devoluciones, las
// suscripciones (preapproval) de la plataforma y la firma de los webhooks. No toca la base: lo que se
// guarda (tokens cifrados, cobros) es de los dominios. Calca la integración de Tu Local Web
// (mostrador-v2), ya probada en producción.
//
//   - Cada negocio cobra con su propio token. Nunca se cae al token de la plataforma: sin conexión,
//     no hay cobro.
//   - TengoLibre no cobra comisión: marketplace_fee no se manda.
//   - La suscripción del negocio a TengoLibre es un preapproval con el token de la plataforma.
package mercadopago

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// ErrorMP es una respuesta no exitosa de la API. Nunca se le muestra el cuerpo a nadie.
type ErrorMP struct {
	Estado int
	Cuerpo string
}

func (e *ErrorMP) Error() string { return fmt.Sprintf("mercadopago: HTTP %d", e.Estado) }

// Cliente habla con api.mercadopago.com (Base se cambia en los tests por un httptest.Server).
type Cliente struct {
	Base string
	// Autorizar es la página de autorización del OAuth (auth.mercadopago.com).
	Autorizar string
	HTTP      *http.Client
}

func (c *Cliente) base() string {
	if c != nil && c.Base != "" {
		return c.Base
	}
	return "https://api.mercadopago.com"
}

func (c *Cliente) autorizar() string {
	if c != nil && c.Autorizar != "" {
		return c.Autorizar
	}
	return "https://auth.mercadopago.com/authorization"
}

func (c *Cliente) http() *http.Client {
	if c != nil && c.HTTP != nil {
		return c.HTTP
	}
	return &http.Client{Timeout: 15 * time.Second}
}

// JSON hace un pedido con el token dado. idem, si no está vacío, va como X-Idempotency-Key.
func (c *Cliente) JSON(ctx context.Context, metodo, ruta, token, idem string, cuerpo, salida any) error {
	var r io.Reader
	if cuerpo != nil {
		b, err := json.Marshal(cuerpo)
		if err != nil {
			return err
		}
		r = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, metodo, c.base()+ruta, r)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	if cuerpo != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if idem != "" {
		req.Header.Set("X-Idempotency-Key", idem)
	}
	return c.hacer(req, salida)
}

// Formulario: /oauth/token va con application/x-www-form-urlencoded.
func (c *Cliente) Formulario(ctx context.Context, ruta string, v url.Values, salida any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.base()+ruta, strings.NewReader(v.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	return c.hacer(req, salida)
}

func (c *Cliente) hacer(req *http.Request, salida any) error {
	res, err := c.http().Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if res.StatusCode >= 300 {
		return &ErrorMP{Estado: res.StatusCode, Cuerpo: string(b)}
	}
	if salida == nil || len(b) == 0 {
		return nil
	}
	return json.Unmarshal(b, salida)
}

// Recortar a n runas (los textos que van a Mercado Pago tienen topes).
func Recortar(s string, n int) string {
	r := []rune(s)
	if len(r) > n {
		return string(r[:n])
	}
	return s
}
