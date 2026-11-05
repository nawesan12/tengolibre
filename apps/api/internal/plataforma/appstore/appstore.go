// Package appstore verifica lo que firma la App Store: las transacciones de StoreKit 2 que manda la
// app y las notificaciones del servidor (App Store Server Notifications V2). Las dos son JWS (ES256)
// con la cadena de certificados en x5c, que tiene que terminar en Apple Root CA - G3.
package appstore

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/sha256"
	"crypto/x509"
	_ "embed"
	"encoding/asn1"
	"encoding/base64"
	"encoding/json"
	"errors"
	"math/big"
	"strings"
	"time"
)

//go:embed AppleRootCA-G3.cer
var raizDER []byte

// ErrFirma: el JWS no lo firmó Apple (o está roto).
var ErrFirma = errors.New("appstore: la firma no es de Apple")

// OIDs que Apple pone en el certificado que firma (hoja) y en el intermedio (WWDR).
var (
	oidHoja       = asn1.ObjectIdentifier{1, 2, 840, 113635, 100, 6, 11, 1}
	oidIntermedio = asn1.ObjectIdentifier{1, 2, 840, 113635, 100, 6, 2, 1}
)

// Verificador de JWS de la App Store. Raiz nil: Apple Root CA - G3 (los tests ponen otra).
type Verificador struct {
	Raiz  *x509.Certificate
	Ahora func() time.Time
}

func (v *Verificador) raiz() (*x509.Certificate, error) {
	if v != nil && v.Raiz != nil {
		return v.Raiz, nil
	}
	return x509.ParseCertificate(raizDER)
}

// Verificar valida la cadena y la firma y devuelve el payload (JSON).
func (v *Verificador) Verificar(jws string) ([]byte, error) {
	partes := strings.Split(strings.TrimSpace(jws), ".")
	if len(partes) != 3 {
		return nil, ErrFirma
	}
	cab, err := base64.RawURLEncoding.DecodeString(partes[0])
	if err != nil {
		return nil, ErrFirma
	}
	var h struct {
		Alg string   `json:"alg"`
		X5c []string `json:"x5c"`
	}
	if json.Unmarshal(cab, &h) != nil || h.Alg != "ES256" || len(h.X5c) < 2 {
		return nil, ErrFirma
	}
	var certs []*x509.Certificate
	for _, c := range h.X5c {
		der, err := base64.StdEncoding.DecodeString(c)
		if err != nil {
			return nil, ErrFirma
		}
		x, err := x509.ParseCertificate(der)
		if err != nil {
			return nil, ErrFirma
		}
		certs = append(certs, x)
	}
	raiz, err := v.raiz()
	if err != nil {
		return nil, err
	}
	// La raíz de la cadena tiene que ser la de Apple (si viene en x5c, igual a la nuestra).
	if len(certs) >= 3 && !bytes.Equal(certs[len(certs)-1].Raw, raiz.Raw) {
		return nil, ErrFirma
	}
	raices := x509.NewCertPool()
	raices.AddCert(raiz)
	inter := x509.NewCertPool()
	inter.AddCert(certs[1])
	cuando := time.Now()
	if v != nil && v.Ahora != nil {
		cuando = v.Ahora()
	}
	if _, err := certs[0].Verify(x509.VerifyOptions{Roots: raices, Intermediates: inter, CurrentTime: cuando,
		KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageAny}}); err != nil {
		return nil, ErrFirma
	}
	if v == nil || v.Raiz == nil {
		if !tieneExtension(certs[0], oidHoja) || !tieneExtension(certs[1], oidIntermedio) {
			return nil, ErrFirma
		}
	}
	pub, ok := certs[0].PublicKey.(*ecdsa.PublicKey)
	if !ok {
		return nil, ErrFirma
	}
	firma, err := base64.RawURLEncoding.DecodeString(partes[2])
	if err != nil || len(firma) != 64 {
		return nil, ErrFirma
	}
	suma := sha256.Sum256([]byte(partes[0] + "." + partes[1]))
	r, s := new(big.Int).SetBytes(firma[:32]), new(big.Int).SetBytes(firma[32:])
	if !ecdsa.Verify(pub, suma[:], r, s) {
		return nil, ErrFirma
	}
	return base64.RawURLEncoding.DecodeString(partes[1])
}

func tieneExtension(c *x509.Certificate, oid asn1.ObjectIdentifier) bool {
	for _, e := range c.Extensions {
		if e.Id.Equal(oid) {
			return true
		}
	}
	return false
}

// Transaccion (JWSTransactionDecodedPayload): lo que importa de una compra de suscripción.
type Transaccion struct {
	OriginalTransactionID string `json:"originalTransactionId"`
	TransactionID         string `json:"transactionId"`
	BundleID              string `json:"bundleId"`
	ProductID             string `json:"productId"`
	// AppAccountToken: la app lo pone al comprar (el id del negocio).
	AppAccountToken string `json:"appAccountToken"`
	ExpiresDate     int64  `json:"expiresDate"` // milisegundos
	PurchaseDate    int64  `json:"purchaseDate"`
	RevocationDate  int64  `json:"revocationDate"`
	Environment     string `json:"environment"` // Production | Sandbox
	Type            string `json:"type"`
}

// Vence es expiresDate como time.Time (cero si no tiene).
func (t Transaccion) Vence() time.Time {
	if t.ExpiresDate == 0 {
		return time.Time{}
	}
	return time.UnixMilli(t.ExpiresDate)
}

// Renovacion (JWSRenewalInfoDecodedPayload).
type Renovacion struct {
	OriginalTransactionID  string `json:"originalTransactionId"`
	AutoRenewProductID     string `json:"autoRenewProductId"`
	AutoRenewStatus        int    `json:"autoRenewStatus"`
	GracePeriodExpiresDate int64  `json:"gracePeriodExpiresDate"`
	IsInBillingRetryPeriod bool   `json:"isInBillingRetryPeriod"`
}

// Notificacion (responseBodyV2DecodedPayload), con la transacción y la renovación ya verificadas.
type Notificacion struct {
	Tipo        string
	Subtipo     string
	Entorno     string
	BundleID    string
	Transaccion *Transaccion
	Renovacion  *Renovacion
}

// LeerTransaccion verifica y decodifica una transacción firmada.
func (v *Verificador) LeerTransaccion(jws string) (Transaccion, error) {
	var t Transaccion
	b, err := v.Verificar(jws)
	if err != nil {
		return t, err
	}
	if err := json.Unmarshal(b, &t); err != nil || t.OriginalTransactionID == "" {
		return t, ErrFirma
	}
	return t, nil
}

// LeerNotificacion verifica el signedPayload de una notificación y lo que trae adentro.
func (v *Verificador) LeerNotificacion(signedPayload string) (Notificacion, error) {
	var n Notificacion
	b, err := v.Verificar(signedPayload)
	if err != nil {
		return n, err
	}
	var p struct {
		NotificationType string `json:"notificationType"`
		Subtype          string `json:"subtype"`
		Data             struct {
			BundleID              string `json:"bundleId"`
			Environment           string `json:"environment"`
			SignedTransactionInfo string `json:"signedTransactionInfo"`
			SignedRenewalInfo     string `json:"signedRenewalInfo"`
		} `json:"data"`
	}
	if err := json.Unmarshal(b, &p); err != nil {
		return n, ErrFirma
	}
	n = Notificacion{Tipo: p.NotificationType, Subtipo: p.Subtype, Entorno: p.Data.Environment, BundleID: p.Data.BundleID}
	if p.Data.SignedTransactionInfo != "" {
		t, err := v.LeerTransaccion(p.Data.SignedTransactionInfo)
		if err != nil {
			return n, err
		}
		n.Transaccion = &t
	}
	if p.Data.SignedRenewalInfo != "" {
		rb, err := v.Verificar(p.Data.SignedRenewalInfo)
		if err != nil {
			return n, err
		}
		var r Renovacion
		if json.Unmarshal(rb, &r) == nil {
			n.Renovacion = &r
		}
	}
	return n, nil
}
