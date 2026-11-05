// Package appstoreprueba firma como la App Store, con una cadena propia, para los tests de los
// dominios (el Verificador se arma con su raíz).
package appstoreprueba

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"time"

	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/appstore"
)

// Firmante: raíz, intermedio y hoja propios.
type Firmante struct {
	Raiz       *x509.Certificate
	hoja, inte []byte
	clave      *ecdsa.PrivateKey
}

func certificado(cn string, padre *x509.Certificate, clavePadre *ecdsa.PrivateKey, ca bool) (*x509.Certificate, *ecdsa.PrivateKey) {
	k, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	tpl := &x509.Certificate{SerialNumber: big.NewInt(time.Now().UnixNano()), Subject: pkix.Name{CommonName: cn},
		NotBefore: time.Now().AddDate(-1, 0, 0), NotAfter: time.Now().AddDate(5, 0, 0), IsCA: ca, BasicConstraintsValid: true,
		KeyUsage: x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign}
	if padre == nil {
		padre, clavePadre = tpl, k
	}
	der, err := x509.CreateCertificate(rand.Reader, tpl, padre, &k.PublicKey, clavePadre)
	if err != nil {
		panic(err)
	}
	c, _ := x509.ParseCertificate(der)
	return c, k
}

// Nuevo arma una cadena.
func Nuevo() *Firmante {
	raiz, kr := certificado("raiz", nil, nil, true)
	inte, ki := certificado("intermedio", raiz, kr, true)
	hoja, kh := certificado("hoja", inte, ki, false)
	return &Firmante{Raiz: raiz, hoja: hoja.Raw, inte: inte.Raw, clave: kh}
}

// Verificador que acepta lo que firma este Firmante.
func (f *Firmante) Verificador() *appstore.Verificador { return &appstore.Verificador{Raiz: f.Raiz} }

// Firmar devuelve el JWS (ES256, x5c) de payload.
func (f *Firmante) Firmar(payload any) string {
	cab, _ := json.Marshal(map[string]any{"alg": "ES256", "x5c": []string{
		base64.StdEncoding.EncodeToString(f.hoja), base64.StdEncoding.EncodeToString(f.inte), base64.StdEncoding.EncodeToString(f.Raiz.Raw)}})
	cuerpo, _ := json.Marshal(payload)
	firmado := base64.RawURLEncoding.EncodeToString(cab) + "." + base64.RawURLEncoding.EncodeToString(cuerpo)
	suma := sha256.Sum256([]byte(firmado))
	r, s, err := ecdsa.Sign(rand.Reader, f.clave, suma[:])
	if err != nil {
		panic(err)
	}
	firma := make([]byte, 64)
	r.FillBytes(firma[:32])
	s.FillBytes(firma[32:])
	return firmado + "." + base64.RawURLEncoding.EncodeToString(firma)
}

// Transaccion firma una compra de suscripción.
func (f *Firmante) Transaccion(original, producto, negocio string, vence time.Time) string {
	return f.Firmar(map[string]any{"originalTransactionId": original, "transactionId": original + "-1", "bundleId": "ar.com.tengolibre.app",
		"productId": producto, "appAccountToken": negocio, "expiresDate": vence.UnixMilli(), "purchaseDate": vence.AddDate(0, -1, 0).UnixMilli(),
		"environment": "Sandbox", "type": "Auto-Renewable Subscription"})
}

// Notificacion firma una notificación V2 con su transacción (y renovación, si no es "").
func (f *Firmante) Notificacion(tipo, subtipo, transaccion, renovacion string) string {
	data := map[string]any{"bundleId": "ar.com.tengolibre.app", "environment": "Sandbox", "signedTransactionInfo": transaccion}
	if renovacion != "" {
		data["signedRenewalInfo"] = renovacion
	}
	return f.Firmar(map[string]any{"notificationType": tipo, "subtype": subtipo, "data": data})
}
