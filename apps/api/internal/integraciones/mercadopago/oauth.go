package mercadopago

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"net/url"
)

// Aplicacion son las credenciales de la aplicación de TengoLibre en Mercado Pago (OAuth).
type Aplicacion struct {
	ID, Secreto string
	// Vuelta del OAuth: tiene que coincidir exacto con la configurada en la aplicación.
	Vuelta string
}

// Tokens del canje o de la renovación.
type Tokens struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
	Scope        string `json:"scope"`
	UserID       int64  `json:"user_id"`
}

// Azar devuelve un texto aleatorio para URL (state, verificador PKCE).
func Azar() string {
	b := make([]byte, 24)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

// Hash de un secreto de un solo uso (el state se guarda hasheado).
func Hash(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}

// desafio PKCE S256 del verificador.
func desafio(verificador string) string {
	h := sha256.Sum256([]byte(verificador))
	return base64.RawURLEncoding.EncodeToString(h[:])
}

// URLAutorizacion arma la página a la que va la dueña para autorizar a TengoLibre.
func (c *Cliente) URLAutorizacion(a Aplicacion, estado, verificador string) string {
	q := url.Values{
		"client_id":     {a.ID},
		"response_type": {"code"},
		"platform_id":   {"mp"},
		"redirect_uri":  {a.Vuelta},
		"state":         {estado},
		// PKCE S256 (la aplicación lo exige).
		"code_challenge":        {desafio(verificador)},
		"code_challenge_method": {"S256"},
	}
	return c.autorizar() + "?" + q.Encode()
}

// Canjear el código de la vuelta por los tokens del vendedor.
func (c *Cliente) Canjear(ctx context.Context, a Aplicacion, codigo, verificador string) (Tokens, error) {
	var tk Tokens
	err := c.Formulario(ctx, "/oauth/token", url.Values{
		"grant_type":    {"authorization_code"},
		"client_id":     {a.ID},
		"client_secret": {a.Secreto},
		"code":          {codigo},
		"redirect_uri":  {a.Vuelta},
		"code_verifier": {verificador},
	}, &tk)
	if err == nil && (tk.AccessToken == "" || tk.RefreshToken == "" || tk.UserID == 0) {
		err = errors.New("mercadopago: el canje no trajo tokens")
	}
	return tk, err
}

// Renovar con el refresh token. Si Mercado Pago no rota el refresh, se conserva el anterior.
func (c *Cliente) Renovar(ctx context.Context, a Aplicacion, renovacion string) (Tokens, error) {
	var tk Tokens
	err := c.Formulario(ctx, "/oauth/token", url.Values{
		"grant_type":    {"refresh_token"},
		"client_id":     {a.ID},
		"client_secret": {a.Secreto},
		"refresh_token": {renovacion},
	}, &tk)
	if err == nil && tk.AccessToken == "" {
		err = errors.New("mercadopago: la renovación no trajo token")
	}
	if tk.RefreshToken == "" {
		tk.RefreshToken = renovacion
	}
	return tk, err
}
