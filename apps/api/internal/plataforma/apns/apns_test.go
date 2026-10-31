package apns

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func claveDePrueba(t *testing.T) *ecdsa.PrivateKey {
	t.Helper()
	k, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	return k
}

// verificar comprueba un JWT ES256 con la clave pública y devuelve cabecera y cuerpo.
func verificar(t *testing.T, jwt string, pub *ecdsa.PublicKey) (map[string]any, map[string]any) {
	t.Helper()
	partes := strings.Split(jwt, ".")
	if len(partes) != 3 {
		t.Fatalf("el JWT tiene %d partes", len(partes))
	}
	enc := base64.RawURLEncoding
	firma, err := enc.DecodeString(partes[2])
	if err != nil || len(firma) != 64 {
		t.Fatalf("firma: %v, %d bytes (quería 64, r‖s)", err, len(firma))
	}
	h := sha256.Sum256([]byte(partes[0] + "." + partes[1]))
	if !ecdsa.Verify(pub, h[:], new(big.Int).SetBytes(firma[:32]), new(big.Int).SetBytes(firma[32:])) {
		t.Fatal("la firma no verifica con la clave pública")
	}
	var cab, cue map[string]any
	b, _ := enc.DecodeString(partes[0])
	_ = json.Unmarshal(b, &cab)
	b, _ = enc.DecodeString(partes[1])
	_ = json.Unmarshal(b, &cue)
	return cab, cue
}

func TestFirmarJWT(t *testing.T) {
	k := claveDePrueba(t)
	ahora := time.Unix(1_790_000_000, 0)
	jwt, err := FirmarJWT(k, "ABC123DEFG", "TEAM456789", ahora)
	if err != nil {
		t.Fatal(err)
	}
	cab, cue := verificar(t, jwt, &k.PublicKey)
	if cab["alg"] != "ES256" || cab["kid"] != "ABC123DEFG" {
		t.Errorf("cabecera: %v", cab)
	}
	if cue["iss"] != "TEAM456789" || cue["iat"] != float64(ahora.Unix()) {
		t.Errorf("cuerpo: %v", cue)
	}
	// Con otra clave no verifica.
	otra := claveDePrueba(t)
	partes := strings.Split(jwt, ".")
	firma, _ := base64.RawURLEncoding.DecodeString(partes[2])
	h := sha256.Sum256([]byte(partes[0] + "." + partes[1]))
	if ecdsa.Verify(&otra.PublicKey, h[:], new(big.Int).SetBytes(firma[:32]), new(big.Int).SetBytes(firma[32:])) {
		t.Fatal("verificó con otra clave")
	}
}

func TestElJWTSeReusa(t *testing.T) {
	k := claveDePrueba(t)
	ahora := time.Unix(1_790_000_000, 0)
	c := &Cliente{KeyID: "K", TeamID: "T", Clave: k, Ahora: func() time.Time { return ahora }}
	a, _ := c.Token()
	ahora = ahora.Add(39 * time.Minute)
	if b, _ := c.Token(); b != a {
		t.Fatal("a los 39 minutos firmó otro JWT")
	}
	ahora = ahora.Add(2 * time.Minute)
	if b, _ := c.Token(); b == a {
		t.Fatal("a los 41 minutos siguió con el mismo JWT")
	}
}

func TestLeerClaveYDesde(t *testing.T) {
	k := claveDePrueba(t)
	der, err := x509.MarshalPKCS8PrivateKey(k)
	if err != nil {
		t.Fatal(err)
	}
	ruta := filepath.Join(t.TempDir(), "AuthKey_PRUEBA.p8")
	if err := os.WriteFile(ruta, pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der}), 0o600); err != nil {
		t.Fatal(err)
	}
	c, err := Desde("K", "T", ruta, "ar.com.tulocalweb.app")
	if err != nil || c == nil || !c.Clave.Equal(k) {
		t.Fatalf("Desde: %v %v", c, err)
	}
	// Sin configuración: apagado, sin error.
	for _, faltante := range [][4]string{{"", "T", ruta, "x"}, {"K", "", ruta, "x"}, {"K", "T", "", "x"}, {"K", "T", ruta, ""}} {
		c, err := Desde(faltante[0], faltante[1], faltante[2], faltante[3])
		if c != nil || err != nil {
			t.Errorf("Desde(%q): %v %v, quería nil sin error", faltante, c, err)
		}
	}
	if _, err := Desde("K", "T", filepath.Join(t.TempDir(), "no-existe.p8"), "x"); err == nil {
		t.Error("una ruta que no existe tendría que ser error")
	}
	if _, err := LeerClave([]byte("no es pem")); err == nil {
		t.Error("una clave que no es PEM tendría que ser error")
	}
	// Un cliente nil no manda nada ni falla.
	var nada *Cliente
	if err := nada.Enviar(context.Background(), Envio{Token: "aa"}); err != nil || nada.Activo() {
		t.Fatalf("cliente nil: %v", err)
	}
}

// apnsFalso es un servidor HTTP/2 que responde como APNs según el token del dispositivo.
type apnsFalso struct {
	mu        sync.Mutex
	pedidos   []*http.Request
	cuerpos   [][]byte
	respuesta func(token string) (int, string)
}

func (f *apnsFalso) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	b, _ := io.ReadAll(r.Body)
	f.mu.Lock()
	f.pedidos = append(f.pedidos, r)
	f.cuerpos = append(f.cuerpos, b)
	f.mu.Unlock()
	if r.ProtoMajor != 2 {
		w.WriteHeader(http.StatusHTTPVersionNotSupported)
		return
	}
	estado, motivo := 200, ""
	if f.respuesta != nil {
		estado, motivo = f.respuesta(strings.TrimPrefix(r.URL.Path, "/3/device/"))
	}
	w.Header().Set("apns-id", "00000000-0000-0000-0000-000000000000")
	w.WriteHeader(estado)
	if motivo != "" {
		_ = json.NewEncoder(w).Encode(map[string]string{"reason": motivo})
	}
}

func servidorFalso(t *testing.T, f *apnsFalso) (*Cliente, *ecdsa.PrivateKey) {
	t.Helper()
	srv := httptest.NewUnstartedServer(f)
	srv.EnableHTTP2 = true
	srv.StartTLS()
	t.Cleanup(srv.Close)
	k := claveDePrueba(t)
	return &Cliente{KeyID: "KEY1234567", TeamID: "TEAM123456", Topic: "ar.com.tulocalweb.app", Clave: k,
		HTTP: srv.Client(), Hosts: map[string]string{"sandbox": srv.URL, "produccion": srv.URL}}, k
}

func TestEnviarPorHTTP2(t *testing.T) {
	f := &apnsFalso{}
	c, k := servidorFalso(t, f)
	token := strings.Repeat("ab", 32)
	if err := c.Enviar(context.Background(), Envio{Token: token, Entorno: "sandbox", Payload: []byte(`{"aps":{}}`), Colapso: "venta-1"}); err != nil {
		t.Fatal(err)
	}
	if len(f.pedidos) != 1 {
		t.Fatalf("%d pedidos", len(f.pedidos))
	}
	r := f.pedidos[0]
	if r.ProtoMajor != 2 || r.Method != "POST" || r.URL.Path != "/3/device/"+token {
		t.Errorf("pedido: %s %s %s", r.Proto, r.Method, r.URL.Path)
	}
	for k, v := range map[string]string{"apns-topic": "ar.com.tulocalweb.app", "apns-push-type": "alert", "apns-priority": "10", "apns-collapse-id": "venta-1"} {
		if r.Header.Get(k) != v {
			t.Errorf("%s = %q, quería %q", k, r.Header.Get(k), v)
		}
	}
	jwt, ok := strings.CutPrefix(r.Header.Get("authorization"), "bearer ")
	if !ok {
		t.Fatal("sin bearer")
	}
	verificar(t, jwt, &k.PublicKey)
	if string(f.cuerpos[0]) != `{"aps":{}}` {
		t.Errorf("cuerpo: %s", f.cuerpos[0])
	}
	// El segundo envío reusa el JWT.
	_ = c.Enviar(context.Background(), Envio{Token: token, Entorno: "produccion", Payload: []byte(`{}`)})
	if f.pedidos[1].Header.Get("authorization") != r.Header.Get("authorization") {
		t.Error("el segundo envío firmó otro JWT")
	}
}

func TestRespuestasDeAPNs(t *testing.T) {
	f := &apnsFalso{respuesta: func(token string) (int, string) {
		switch token[:2] {
		case "aa":
			return http.StatusGone, "Unregistered"
		case "bb":
			return http.StatusBadRequest, "BadDeviceToken"
		case "cc":
			return http.StatusTooManyRequests, "TooManyRequests"
		case "dd":
			return http.StatusBadRequest, "PayloadTooLarge"
		}
		return 200, ""
	}}
	c, _ := servidorFalso(t, f)
	ctx := context.Background()
	for _, p := range []string{"aa", "bb"} {
		if err := c.Enviar(ctx, Envio{Token: strings.Repeat(p, 32)}); !errors.Is(err, ErrTokenInvalido) {
			t.Errorf("%s: %v, quería ErrTokenInvalido", p, err)
		}
	}
	var ea *Error
	if err := c.Enviar(ctx, Envio{Token: strings.Repeat("cc", 32)}); !errors.As(err, &ea) || !ea.Reintentable() {
		t.Errorf("429: %v, quería reintentable", err)
	}
	if err := c.Enviar(ctx, Envio{Token: strings.Repeat("dd", 32)}); !errors.As(err, &ea) || ea.Reintentable() || ea.Motivo != "PayloadTooLarge" {
		t.Errorf("400: %v", err)
	}
}

func TestJWTVencidoFirmaOtro(t *testing.T) {
	var vistos []string
	f := &apnsFalso{}
	f.respuesta = func(string) (int, string) {
		jwt := f.pedidos[len(f.pedidos)-1].Header.Get("authorization")
		vistos = append(vistos, jwt)
		if len(vistos) == 1 {
			return http.StatusForbidden, "ExpiredProviderToken"
		}
		return 200, ""
	}
	c, _ := servidorFalso(t, f)
	ahora := time.Unix(1_790_000_000, 0)
	c.Ahora = func() time.Time { ahora = ahora.Add(time.Second); return ahora }
	if err := c.Enviar(context.Background(), Envio{Token: strings.Repeat("ee", 32)}); err != nil {
		t.Fatal(err)
	}
	if len(vistos) != 2 || vistos[0] == vistos[1] {
		t.Fatalf("quería dos intentos con JWT distintos: %d", len(vistos))
	}
}
