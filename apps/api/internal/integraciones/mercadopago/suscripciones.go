package mercadopago

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Preaprobacion es una suscripción (preapproval) de la cuenta de la plataforma.
type Preaprobacion struct {
	ID                string     `json:"id"`
	Status            string     `json:"status"`
	InitPoint         string     `json:"init_point"`
	ExternalReference string     `json:"external_reference"`
	PayerEmail        string     `json:"payer_email"`
	NextPaymentDate   *time.Time `json:"next_payment_date"`
}

// NuevaPreaprobacion: una suscripción sin plan, en estado pending (la guía de Subscriptions). La
// tarjeta la carga la persona en Mercado Pago (init_point): TengoLibre nunca la ve.
type NuevaPreaprobacion struct {
	Motivo     string
	Referencia string
	Email      string
	Monto      int64
	// DiasGratis: lo que queda de la prueba no se cobra.
	DiasGratis int
	Vuelta     string
}

// CrearPreaprobacion con el token de la plataforma.
func (c *Cliente) CrearPreaprobacion(ctx context.Context, token string, n NuevaPreaprobacion) (Preaprobacion, error) {
	recurrencia := map[string]any{"frequency": 1, "frequency_type": "months", "transaction_amount": n.Monto, "currency_id": "ARS"}
	if n.DiasGratis > 0 {
		recurrencia["free_trial"] = map[string]any{"frequency": n.DiasGratis, "frequency_type": "days"}
	}
	var p Preaprobacion
	err := c.JSON(ctx, http.MethodPost, "/preapproval", token, "", map[string]any{
		"reason":             n.Motivo,
		"external_reference": n.Referencia,
		"payer_email":        SinEtiqueta(n.Email),
		"auto_recurring":     recurrencia,
		"back_url":           n.Vuelta,
		"status":             "pending",
	}, &p)
	if err == nil && (p.ID == "" || p.InitPoint == "") {
		err = errors.New("mercadopago: la suscripción no trajo init_point")
	}
	return p, err
}

// LeerPreaprobacion con el token de la plataforma.
func (c *Cliente) LeerPreaprobacion(ctx context.Context, token, id string) (Preaprobacion, error) {
	var p Preaprobacion
	err := c.JSON(ctx, http.MethodGet, "/preapproval/"+url.PathEscape(id), token, "", nil, &p)
	return p, err
}

// CambiarPreaprobacion de estado (cancelled, paused).
func (c *Cliente) CambiarPreaprobacion(ctx context.Context, token, id, estado string) error {
	return c.JSON(ctx, http.MethodPut, "/preapproval/"+url.PathEscape(id), token, "", map[string]string{"status": estado}, nil)
}

// CambiarMontoPreaprobacion: el precio nuevo rige desde el próximo cobro (cambio de plan).
func (c *Cliente) CambiarMontoPreaprobacion(ctx context.Context, token, id string, monto int64) error {
	return c.JSON(ctx, http.MethodPut, "/preapproval/"+url.PathEscape(id), token, "", map[string]any{
		"auto_recurring": map[string]any{"transaction_amount": monto, "currency_id": "ARS"},
	}, nil)
}

// CobroAutorizado es el cobro mensual de una suscripción (/authorized_payments/{id}): la única señal
// de que una renovación se rechazó (el preapproval puede seguir "authorized" mientras reintenta).
type CobroAutorizado struct {
	ID            int64  `json:"id"`
	PreapprovalID string `json:"preapproval_id"`
	Status        string `json:"status"`
	Payment       struct {
		Status string `json:"status"`
	} `json:"payment"`
	DebitDate *time.Time `json:"debit_date"`
}

// LeerCobroAutorizado con el token de la plataforma.
func (c *Cliente) LeerCobroAutorizado(ctx context.Context, token, id string) (CobroAutorizado, error) {
	var a CobroAutorizado
	err := c.JSON(ctx, http.MethodGet, "/authorized_payments/"+url.PathEscape(id), token, "", nil, &a)
	return a, err
}

// SinEtiqueta saca la parte "+algo" del mail ("ana+negocio@x.com" → "ana@x.com"): Mercado Pago
// rechaza esos mails con "User bad request", y es la misma casilla.
func SinEtiqueta(email string) string {
	local, dominio, ok := strings.Cut(email, "@")
	if !ok {
		return email
	}
	if base, _, ok := strings.Cut(local, "+"); ok && base != "" {
		return base + "@" + dominio
	}
	return email
}
