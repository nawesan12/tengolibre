// Package googleplay consulta las suscripciones de Google Play (Google Play Developer API,
// purchases.subscriptionsv2) con la cuenta de servicio del producto: firma un JWT RS256, lo canjea por
// un token de acceso y lo reusa mientras vale, igual que fcm. Sólo biblioteca estándar.
//
// Sin configuración (la ruta del JSON de la cuenta), Desde devuelve nil: la suscripción por Google
// Play queda apagada sin romper nada.
package googleplay

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/fcm"
)

// Alcance que pide la API de Google Play.
const Alcance = "https://www.googleapis.com/auth/androidpublisher"

// Host de la API.
const Host = "https://androidpublisher.googleapis.com"

// ErrNoEncontrada: Google no conoce ese token de compra (o es de otra app).
var ErrNoEncontrada = errors.New("googleplay: compra no encontrada")

// Cliente habla con Google Play. Usalo por puntero; es seguro entre goroutines.
type Cliente struct {
	Email, URLToken string
	Clave           *rsa.PrivateKey
	// HTTP y Host: en los tests, los del servidor falso. Vacíos: los de Google.
	HTTP *http.Client
	Host string
	// Ahora: el reloj (tests). Vacío: time.Now.
	Ahora func() time.Time

	mu     sync.Mutex
	acceso string
	vence  time.Time
}

// Desde arma el cliente con el JSON de la cuenta de servicio. Sin ruta, nil y ningún error.
func Desde(ruta string) (*Cliente, error) {
	if ruta == "" {
		return nil, nil
	}
	b, err := os.ReadFile(ruta)
	if err != nil {
		return nil, fmt.Errorf("googleplay: no se pudo leer la cuenta de servicio: %w", err)
	}
	return DeCuenta(b)
}

// DeCuenta arma el cliente con el contenido del JSON de la cuenta de servicio.
func DeCuenta(b []byte) (*Cliente, error) {
	var c fcm.Cuenta
	if err := json.Unmarshal(b, &c); err != nil {
		return nil, errors.New("googleplay: la cuenta de servicio no es JSON")
	}
	if c.ClientEmail == "" || c.PrivateKey == "" {
		return nil, errors.New("googleplay: a la cuenta de servicio le falta client_email o private_key")
	}
	clave, err := fcm.LeerClave([]byte(c.PrivateKey))
	if err != nil {
		return nil, err
	}
	if c.TokenURI == "" {
		c.TokenURI = "https://oauth2.googleapis.com/token"
	}
	return &Cliente{Email: c.ClientEmail, URLToken: c.TokenURI, Clave: clave}, nil
}

func (c *Cliente) ahora() time.Time {
	if c.Ahora != nil {
		return c.Ahora()
	}
	return time.Now()
}

func (c *Cliente) cli() *http.Client {
	if c.HTTP != nil {
		return c.HTTP
	}
	return clientePorDefecto
}

var clientePorDefecto = &http.Client{Timeout: 15 * time.Second}

// firmar arma la aserción {"iss","scope","aud","iat","exp"} firmada con la clave de la cuenta.
func firmar(clave *rsa.PrivateKey, email, aud string, ahora time.Time) (string, error) {
	enc := base64.RawURLEncoding
	cab, _ := json.Marshal(map[string]string{"alg": "RS256", "typ": "JWT"})
	cue, _ := json.Marshal(map[string]any{"iss": email, "scope": Alcance, "aud": aud, "iat": ahora.Unix(), "exp": ahora.Add(time.Hour).Unix()})
	firmable := enc.EncodeToString(cab) + "." + enc.EncodeToString(cue)
	h := sha256.Sum256([]byte(firmable))
	firma, err := rsa.SignPKCS1v15(rand.Reader, clave, crypto.SHA256, h[:])
	if err != nil {
		return "", err
	}
	return firmable + "." + enc.EncodeToString(firma), nil
}

// token devuelve el token de acceso, pidiendo otro si no hay o vence en menos de 5 minutos.
func (c *Cliente) token(ctx context.Context) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	ahora := c.ahora()
	if c.acceso != "" && ahora.Add(5*time.Minute).Before(c.vence) {
		return c.acceso, nil
	}
	jwt, err := firmar(c.Clave, c.Email, c.URLToken, ahora)
	if err != nil {
		return "", err
	}
	form := url.Values{"grant_type": {"urn:ietf:params:oauth:grant-type:jwt-bearer"}, "assertion": {jwt}}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.URLToken, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("content-type", "application/x-www-form-urlencoded")
	res, err := c.cli().Do(req)
	if err != nil {
		return "", errors.New("googleplay: no se pudo pedir el token de acceso")
	}
	defer res.Body.Close()
	var r struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if res.StatusCode != http.StatusOK || json.NewDecoder(io.LimitReader(res.Body, 64<<10)).Decode(&r) != nil || r.AccessToken == "" {
		return "", fmt.Errorf("googleplay: el token de acceso respondió HTTP %d", res.StatusCode)
	}
	if r.ExpiresIn <= 0 {
		r.ExpiresIn = 3600
	}
	c.acceso, c.vence = r.AccessToken, ahora.Add(time.Duration(r.ExpiresIn)*time.Second)
	return c.acceso, nil
}

// Suscripcion es lo que hace falta de SubscriptionPurchaseV2.
type Suscripcion struct {
	Estado          string    `json:"subscriptionState"` // SUBSCRIPTION_STATE_ACTIVE, _IN_GRACE_PERIOD, _ON_HOLD, _PAUSED, _CANCELED, _EXPIRED, _PENDING
	Inicio          string    `json:"startTime"`
	TokenAnterior   string    `json:"linkedPurchaseToken"`
	Prueba          *struct{} `json:"testPurchase"`
	Identificadores struct {
		Cuenta string `json:"obfuscatedExternalAccountId"`
	} `json:"externalAccountIdentifiers"`
	Items []struct {
		Producto string `json:"productId"`
		Vence    string `json:"expiryTime"`
		Renueva  *struct {
			Activa bool `json:"autoRenewEnabled"`
		} `json:"autoRenewingPlan"`
	} `json:"lineItems"`
}

// Producto del primer ítem (las de Tengo Libre tienen uno solo).
func (s Suscripcion) Producto() string {
	if len(s.Items) == 0 {
		return ""
	}
	return s.Items[0].Producto
}

// Vence: el último vencimiento de los ítems (cero si no vino).
func (s Suscripcion) Vence() time.Time {
	var v time.Time
	for _, i := range s.Items {
		if t, err := time.Parse(time.RFC3339Nano, i.Vence); err == nil && t.After(v) {
			v = t
		}
	}
	return v
}

// Renueva dice si la renovación automática sigue prendida.
func (s Suscripcion) Renueva() bool {
	for _, i := range s.Items {
		if i.Renueva != nil && i.Renueva.Activa {
			return true
		}
	}
	return false
}

// Consultar trae la suscripción de ese token de compra.
func (c *Cliente) Consultar(ctx context.Context, paquete, tokenCompra string) (*Suscripcion, error) {
	if c == nil {
		return nil, errors.New("googleplay: apagado")
	}
	acceso, err := c.token(ctx)
	if err != nil {
		return nil, err
	}
	host := c.Host
	if host == "" {
		host = Host
	}
	u := strings.TrimRight(host, "/") + "/androidpublisher/v3/applications/" + url.PathEscape(paquete) + "/purchases/subscriptionsv2/tokens/" + url.PathEscape(tokenCompra)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("authorization", "Bearer "+acceso)
	res, err := c.cli().Do(req)
	if err != nil {
		return nil, errors.New("googleplay: no se pudo conectar")
	}
	defer res.Body.Close()
	switch {
	case res.StatusCode == http.StatusNotFound || res.StatusCode == http.StatusGone || res.StatusCode == http.StatusBadRequest:
		return nil, ErrNoEncontrada
	case res.StatusCode != http.StatusOK:
		if res.StatusCode == http.StatusUnauthorized {
			c.mu.Lock()
			c.acceso = ""
			c.mu.Unlock()
		}
		return nil, fmt.Errorf("googleplay: HTTP %d", res.StatusCode)
	}
	var s Suscripcion
	if err := json.NewDecoder(io.LimitReader(res.Body, 256<<10)).Decode(&s); err != nil {
		return nil, errors.New("googleplay: respuesta ilegible")
	}
	return &s, nil
}
