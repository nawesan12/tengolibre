// Package cifrado guarda secretos de terceros (los tokens de Mercado Pago) cifrados en la
// base: AES-256-GCM con una clave derivada por negocio (HKDF-SHA256) de la clave maestra. Un volcado
// de la base no alcanza para usarlos, y un token de un negocio no se descifra con la clave de otro.
package cifrado

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hkdf"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
)

// ErrClave: la clave maestra falta o no tiene 32 bytes.
var ErrClave = errors.New("cifrado: la clave maestra tiene que ser de 32 bytes en base64")

// Cifrador con la clave maestra (TENGOLIBRE_CLAVE_CIFRADO).
type Cifrador struct{ maestra []byte }

// Nuevo recibe la clave maestra en base64.
func Nuevo(b64 string) (*Cifrador, error) {
	k, err := base64.StdEncoding.DecodeString(b64)
	if err != nil || len(k) != 32 {
		return nil, ErrClave
	}
	return &Cifrador{maestra: k}, nil
}

func (c *Cifrador) aead(contexto string) (cipher.AEAD, error) {
	k, err := hkdf.Key(sha256.New, c.maestra, nil, "tengolibre:"+contexto, 32)
	if err != nil {
		return nil, err
	}
	b, err := aes.NewCipher(k)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(b)
}

// Cifrar devuelve nonce + texto cifrado. contexto es, por ejemplo, el id del negocio.
func (c *Cifrador) Cifrar(contexto string, plano []byte) ([]byte, error) {
	a, err := c.aead(contexto)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, a.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	return a.Seal(nonce, nonce, plano, []byte(contexto)), nil
}

// Descifrar falla si el dato es de otro contexto o se tocó.
func (c *Cifrador) Descifrar(contexto string, dato []byte) ([]byte, error) {
	a, err := c.aead(contexto)
	if err != nil {
		return nil, err
	}
	if len(dato) < a.NonceSize() {
		return nil, errors.New("cifrado: dato corto")
	}
	return a.Open(nil, dato[:a.NonceSize()], dato[a.NonceSize():], []byte(contexto))
}
