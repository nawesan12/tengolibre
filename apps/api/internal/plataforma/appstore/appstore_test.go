package appstore

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
	"strings"
	"testing"
	"time"
)

type cadena struct {
	raiz       *x509.Certificate
	hoja, inte []byte
	clave      *ecdsa.PrivateKey
}

func certificado(t *testing.T, cn string, padre *x509.Certificate, clavePadre *ecdsa.PrivateKey, ca bool) (*x509.Certificate, *ecdsa.PrivateKey) {
	t.Helper()
	k, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	tpl := &x509.Certificate{SerialNumber: big.NewInt(time.Now().UnixNano()), Subject: pkix.Name{CommonName: cn},
		NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour), IsCA: ca, BasicConstraintsValid: true,
		KeyUsage: x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign}
	if padre == nil {
		padre, clavePadre = tpl, k
	}
	der, err := x509.CreateCertificate(rand.Reader, tpl, padre, &k.PublicKey, clavePadre)
	if err != nil {
		t.Fatal(err)
	}
	c, _ := x509.ParseCertificate(der)
	return c, k
}

func nuevaCadena(t *testing.T) cadena {
	raiz, kr := certificado(t, "raiz", nil, nil, true)
	inte, ki := certificado(t, "intermedio", raiz, kr, true)
	hoja, kh := certificado(t, "hoja", inte, ki, false)
	return cadena{raiz: raiz, hoja: hoja.Raw, inte: inte.Raw, clave: kh}
}

func (c cadena) firmar(t *testing.T, payload any) string {
	t.Helper()
	cab, _ := json.Marshal(map[string]any{"alg": "ES256", "x5c": []string{
		base64.StdEncoding.EncodeToString(c.hoja), base64.StdEncoding.EncodeToString(c.inte), base64.StdEncoding.EncodeToString(c.raiz.Raw)}})
	cuerpo, _ := json.Marshal(payload)
	firmado := base64.RawURLEncoding.EncodeToString(cab) + "." + base64.RawURLEncoding.EncodeToString(cuerpo)
	suma := sha256.Sum256([]byte(firmado))
	r, s, err := ecdsa.Sign(rand.Reader, c.clave, suma[:])
	if err != nil {
		t.Fatal(err)
	}
	firma := make([]byte, 64)
	r.FillBytes(firma[:32])
	s.FillBytes(firma[32:])
	return firmado + "." + base64.RawURLEncoding.EncodeToString(firma)
}

func TestTransaccionFirmada(t *testing.T) {
	c := nuevaCadena(t)
	v := &Verificador{Raiz: c.raiz}
	jws := c.firmar(t, map[string]any{"originalTransactionId": "1000", "productId": "ar.com.tengolibre.completo.mensual",
		"bundleId": "ar.com.tengolibre.app", "expiresDate": time.Now().Add(720 * time.Hour).UnixMilli(), "environment": "Sandbox"})
	tr, err := v.LeerTransaccion(jws)
	if err != nil {
		t.Fatal(err)
	}
	if tr.OriginalTransactionID != "1000" || tr.Vence().Before(time.Now()) {
		t.Fatalf("transacción mal leída: %+v", tr)
	}
}

func TestFirmaDeOtraRaiz(t *testing.T) {
	c, otra := nuevaCadena(t), nuevaCadena(t)
	v := &Verificador{Raiz: otra.raiz}
	if _, err := v.LeerTransaccion(c.firmar(t, map[string]any{"originalTransactionId": "1"})); err == nil {
		t.Fatal("una cadena que no termina en la raíz tiene que fallar")
	}
}

func TestPayloadTocado(t *testing.T) {
	c := nuevaCadena(t)
	v := &Verificador{Raiz: c.raiz}
	jws := c.firmar(t, map[string]any{"originalTransactionId": "1", "expiresDate": 1})
	p := strings.Split(jws, ".")
	p[1] = base64.RawURLEncoding.EncodeToString([]byte(`{"originalTransactionId":"1","expiresDate":99999999999999}`))
	if _, err := v.LeerTransaccion(strings.Join(p, ".")); err == nil {
		t.Fatal("un payload cambiado tiene que fallar")
	}
}

func TestNotificacion(t *testing.T) {
	c := nuevaCadena(t)
	v := &Verificador{Raiz: c.raiz}
	tr := c.firmar(t, map[string]any{"originalTransactionId": "77", "productId": "p", "expiresDate": time.Now().Add(time.Hour).UnixMilli()})
	ren := c.firmar(t, map[string]any{"originalTransactionId": "77", "autoRenewStatus": 0})
	n, err := v.LeerNotificacion(c.firmar(t, map[string]any{"notificationType": "DID_CHANGE_RENEWAL_STATUS", "subtype": "AUTO_RENEW_DISABLED",
		"data": map[string]any{"bundleId": "ar.com.tengolibre.app", "environment": "Sandbox", "signedTransactionInfo": tr, "signedRenewalInfo": ren}}))
	if err != nil {
		t.Fatal(err)
	}
	if n.Tipo != "DID_CHANGE_RENEWAL_STATUS" || n.Transaccion == nil || n.Transaccion.OriginalTransactionID != "77" || n.Renovacion == nil {
		t.Fatalf("notificación mal leída: %+v", n)
	}
}

func TestRaizDeApple(t *testing.T) {
	r, err := (&Verificador{}).raiz()
	if err != nil || r.Subject.CommonName != "Apple Root CA - G3" {
		t.Fatalf("la raíz embebida no es la de Apple: %v %v", r, err)
	}
}
