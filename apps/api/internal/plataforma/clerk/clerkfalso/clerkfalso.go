// Package clerkfalso es una Backend API de Clerk para los tests (sin red): sirve el JWKS y las
// personas cargadas, y firma tokens de sesión con su propia clave.
package clerkfalso

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// Clave secreta y emisor que espera el servidor falso.
const (
	Secreto = "sk_test_falso"
	Emisor  = "https://clerk.falso.test"
)

// Usuario cargado en el servidor falso.
type Usuario struct {
	Nombre, Apellido, Email, Foto string
	Verificado                    bool
}

// Reclamos del token. Vacíos: emisor del falso y vence en un minuto.
type Reclamos struct {
	Sub, Iss, Azp string
	Exp           time.Time
}

// Servidor falso.
type Servidor struct {
	*httptest.Server
	Usuarios map[string]Usuario

	clave, ajena *rsa.PrivateKey
	mu           sync.Mutex
	jwks         int
	sinSecreto   bool
}

// Nuevo levanta el servidor (se cierra solo al terminar el test).
func Nuevo(t *testing.T) *Servidor {
	t.Helper()
	k, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	k2, _ := rsa.GenerateKey(rand.Reader, 2048)
	f := &Servidor{Usuarios: map[string]Usuario{}, clave: k, ajena: k2}
	f.Server = httptest.NewServer(http.HandlerFunc(f.servir))
	t.Cleanup(f.Close)
	return f
}

func b64(b []byte) string { return base64.RawURLEncoding.EncodeToString(b) }

func (f *Servidor) servir(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if r.Header.Get("Authorization") != "Bearer "+Secreto {
		f.sinSecreto = true
		http.Error(w, `{"errors":[{"code":"authentication_invalid"}]}`, http.StatusUnauthorized)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	switch {
	case r.URL.Path == "/v1/jwks":
		f.jwks++
		pub := f.clave.PublicKey
		_ = json.NewEncoder(w).Encode(map[string]any{"keys": []map[string]string{{
			"kid": "k1", "kty": "RSA", "alg": "RS256", "use": "sig",
			"n": b64(pub.N.Bytes()), "e": b64(big.NewInt(int64(pub.E)).Bytes()),
		}}})
	case strings.HasPrefix(r.URL.Path, "/v1/users/"):
		id := strings.TrimPrefix(r.URL.Path, "/v1/users/")
		u, ok := f.Usuarios[id]
		if !ok {
			http.Error(w, `{"errors":[{"code":"resource_not_found"}]}`, http.StatusNotFound)
			return
		}
		estado := "unverified"
		if u.Verificado {
			estado = "verified"
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": id, "first_name": u.Nombre, "last_name": u.Apellido,
			"image_url": u.Foto, "has_image": u.Foto != "",
			"primary_email_address_id": "idn_1",
			"email_addresses": []map[string]any{{
				"id": "idn_1", "email_address": u.Email, "verification": map[string]string{"status": estado},
			}},
		})
	default:
		http.NotFound(w, r)
	}
}

// PedidosJWKS cuenta cuántas veces se pidió el JWKS.
func (f *Servidor) PedidosJWKS() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.jwks
}

// SoloConSecreto dice si todos los pedidos trajeron la clave secreta.
func (f *Servidor) SoloConSecreto() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return !f.sinSecreto
}

// Token firma un token de sesión con la clave del JWKS.
func (f *Servidor) Token(t *testing.T, r Reclamos) string { return firmar(t, f.clave, "k1", r) }

// TokenAjeno lo firma con otra clave pero el mismo kid.
func (f *Servidor) TokenAjeno(t *testing.T, r Reclamos) string { return firmar(t, f.ajena, "k1", r) }

// TokenKid lo firma con un kid que el JWKS no tiene.
func (f *Servidor) TokenKid(t *testing.T, kid string, r Reclamos) string {
	return firmar(t, f.clave, kid, r)
}

func firmar(t *testing.T, k *rsa.PrivateKey, kid string, r Reclamos) string {
	t.Helper()
	if r.Iss == "" {
		r.Iss = Emisor
	}
	if r.Exp.IsZero() {
		r.Exp = time.Now().Add(time.Minute)
	}
	cab, _ := json.Marshal(map[string]string{"alg": "RS256", "kid": kid, "typ": "JWT"})
	rec := map[string]any{"sub": r.Sub, "iss": r.Iss, "exp": r.Exp.Unix(), "iat": time.Now().Unix(),
		"nbf": time.Now().Add(-10 * time.Second).Unix(), "sid": "sess_1"}
	if r.Azp != "" {
		rec["azp"] = r.Azp
	}
	cuerpo, _ := json.Marshal(rec)
	firmado := b64(cab) + "." + b64(cuerpo)
	h := sha256.Sum256([]byte(firmado))
	sig, err := rsa.SignPKCS1v15(rand.Reader, k, crypto.SHA256, h[:])
	if err != nil {
		t.Fatal(err)
	}
	return firmado + "." + b64(sig)
}
