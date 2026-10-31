package webpush

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdh"
	"crypto/ecdsa"
	"crypto/hkdf"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"
)

func dec(t *testing.T, s string) []byte {
	t.Helper()
	b, err := decodificar(s)
	if err != nil {
		t.Fatalf("%q: %v", s, err)
	}
	return b
}

// Vector de RFC 8291, apéndice A: con la clave efímera y la sal fijas, el mensaje tiene que salir
// byte a byte igual.
func TestVectorRFC8291(t *testing.T) {
	as, err := ecdh.P256().NewPrivateKey(dec(t, "yfWPiYE-n46HLnH0KqZOF1fJJU3MYrct3AELtAQ-oRw"))
	if err != nil {
		t.Fatal(err)
	}
	if got := b64.EncodeToString(as.PublicKey().Bytes()); got != "BP4z9KsN6nGRTbVYI_c7VJSPQTBtkgcy27mlmlMoZIIgDll6e3vCYLocInmYWAmS6TlzAC8wEqKK6PBru3jl7A8" {
		t.Fatalf("pública del servidor: %s", got)
	}
	got, err := CifrarCon([]byte("When I grow up, I want to be a watermelon"),
		"BCVxsr7N_eNgVRqvHtD0zTZsEc6-VV-JvLexhqUzORcxaOzi6-AYWXvTBHm4bjyPjs7Vd8pZGH6SRpkNtoIAiw4",
		"BTBZMqHH6r4Tts7J_aSIgg", as, dec(t, "DGv6ra1nlYgDCS1FRnbzlw"))
	if err != nil {
		t.Fatal(err)
	}
	const quiero = "DGv6ra1nlYgDCS1FRnbzlwAAEABBBP4z9KsN6nGRTbVYI_c7VJSPQTBtkgcy27mlmlMoZIIgDll6e3vCYLocInmYWAmS6TlzAC8wEqKK6PBru3jl7A_yl95bQpu6cVPTpK4Mqgkf1CXztLVBSt2Ks3oZwbuwXPXLWyouBWLVWGNWQexSgSxsj_Qulcy4a-fN"
	if b64.EncodeToString(got) != quiero {
		t.Fatalf("cifrado:\n got %s\nwant %s", b64.EncodeToString(got), quiero)
	}
	// Y el lado del navegador lo descifra.
	ua, _ := ecdh.P256().NewPrivateKey(dec(t, "q1dXpw3UpT5VOmu_cf_v6ih07Aems3njxI-JWgLcM94"))
	if p := descifrar(t, got, ua, dec(t, "BTBZMqHH6r4Tts7J_aSIgg")); string(p) != "When I grow up, I want to be a watermelon" {
		t.Fatalf("descifrado: %q", p)
	}
}

// descifrar hace lo del navegador (RFC 8291 desde el lado del agente de usuario).
func descifrar(t *testing.T, cuerpo []byte, ua *ecdh.PrivateKey, auth []byte) []byte {
	t.Helper()
	if len(cuerpo) < 21 {
		t.Fatal("cuerpo corto")
	}
	sal, rs, idlen := cuerpo[:16], binary.BigEndian.Uint32(cuerpo[16:20]), int(cuerpo[20])
	if rs != TamanoRegistro || idlen != 65 {
		t.Fatalf("encabezado: rs=%d idlen=%d", rs, idlen)
	}
	asPub := cuerpo[21 : 21+idlen]
	as, err := ecdh.P256().NewPublicKey(asPub)
	if err != nil {
		t.Fatal(err)
	}
	comp, err := ua.ECDH(as)
	if err != nil {
		t.Fatal(err)
	}
	info := append(append([]byte("WebPush: info\x00"), ua.PublicKey().Bytes()...), asPub...)
	prkClave, _ := hkdf.Extract(sha256.New, comp, auth)
	ikm, _ := hkdf.Expand(sha256.New, prkClave, string(info), 32)
	prk, _ := hkdf.Extract(sha256.New, ikm, sal)
	cek, _ := hkdf.Expand(sha256.New, prk, "Content-Encoding: aes128gcm\x00", 16)
	nonce, _ := hkdf.Expand(sha256.New, prk, "Content-Encoding: nonce\x00", 12)
	blq, _ := aes.NewCipher(cek)
	gcm, _ := cipher.NewGCM(blq)
	plano, err := gcm.Open(nil, nonce, cuerpo[21+idlen:], nil)
	if err != nil {
		t.Fatalf("no descifra: %v", err)
	}
	if len(plano) == 0 || plano[len(plano)-1] != 2 {
		t.Fatalf("falta el delimitador del último registro: %x", plano)
	}
	return plano[:len(plano)-1]
}

type navegador struct {
	priv *ecdh.PrivateKey
	auth []byte
}

func nuevoNavegador(t *testing.T) navegador {
	t.Helper()
	k, err := ecdh.P256().GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	a := make([]byte, 16)
	_, _ = rand.Read(a)
	return navegador{k, a}
}

func (n navegador) suscripcion(endpoint string) Suscripcion {
	return Suscripcion{Endpoint: endpoint, P256dh: b64.EncodeToString(n.priv.PublicKey().Bytes()), Auth: b64.EncodeToString(n.auth)}
}

func TestCifrarIdaYVuelta(t *testing.T) {
	n := nuevoNavegador(t)
	s := n.suscripcion("https://fcm.googleapis.com/fcm/send/abc")
	msg := []byte(`{"title":"Nueva venta #12","body":"$ 24.020 · 3 productos"}`)
	a, err := Cifrar(msg, s.P256dh, s.Auth)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := Cifrar(msg, s.P256dh, s.Auth)
	if string(a) == string(b) {
		t.Fatal("dos cifrados iguales: la clave efímera o la sal se repiten")
	}
	if got := descifrar(t, a, n.priv, n.auth); string(got) != string(msg) {
		t.Fatalf("ida y vuelta: %q", got)
	}
	if _, err := Cifrar(make([]byte, MaxPayload+1), s.P256dh, s.Auth); err == nil {
		t.Fatal("cifró un aviso más largo que el máximo")
	}
	if _, err := Cifrar(msg, s.P256dh, "corto"); !errors.Is(err, ErrInvalida) {
		t.Fatalf("auth inválido: %v", err)
	}
}

func TestNormalizar(t *testing.T) {
	n := nuevoNavegador(t)
	ok := n.suscripcion("https://web.push.apple.com/QGuQyavXutnMH2")
	// Con relleno y base64 común también entra, y queda en base64url sin relleno.
	con := ok
	con.Auth += "=="
	got, err := Normalizar(con)
	if err != nil || got.Auth != ok.Auth || got.P256dh != ok.P256dh {
		t.Fatalf("%+v %v", got, err)
	}
	for _, ep := range []string{"https://fcm.googleapis.com/fcm/send/x", "https://updates.push.services.mozilla.com/wpush/v2/x",
		"https://wns2-bl2p.notify.windows.com/w/?token=x"} {
		if _, err := Normalizar(n.suscripcion(ep)); err != nil {
			t.Errorf("%s: %v", ep, err)
		}
	}
	malas := []Suscripcion{
		n.suscripcion("http://fcm.googleapis.com/x"),
		n.suscripcion("https://169.254.169.254/latest"),
		n.suscripcion("https://fcm.googleapis.com.evil.com/x"),
		n.suscripcion("https://evilpush.apple.com/x"),
		n.suscripcion("https://fcm.googleapis.com:8443/x"),
		{Endpoint: ok.Endpoint, P256dh: "AAAA", Auth: ok.Auth},
		{Endpoint: ok.Endpoint, P256dh: ok.P256dh, Auth: "AAAA"},
	}
	for _, s := range malas {
		if _, err := Normalizar(s); !errors.Is(err, ErrInvalida) {
			t.Errorf("%+v: %v", s, err)
		}
	}
}

func TestDesde(t *testing.T) {
	if c, err := Desde("", "", ""); c != nil || err != nil {
		t.Fatal("sin configuración tiene que quedar apagado sin error")
	}
	pub, priv, err := GenerarClaves()
	if err != nil {
		t.Fatal(err)
	}
	c, err := Desde(pub, priv, "mailto:hola@tulocalweb.com.ar")
	if err != nil || c.ClavePublica() != pub {
		t.Fatalf("%v %v", c, err)
	}
	otraPub, _, _ := GenerarClaves()
	for _, caso := range [][3]string{{otraPub, priv, "mailto:a@b.c"}, {pub, "xx", "mailto:a@b.c"}, {pub, priv, "hola"}} {
		if _, err := Desde(caso[0], caso[1], caso[2]); err == nil {
			t.Errorf("aceptó %v", caso)
		}
	}
	var nada *Cliente
	if nada.ClavePublica() != "" || nada.Enviar(context.Background(), Envio{}) != nil {
		t.Fatal("un cliente nil no hace nada")
	}
}

// verificar comprueba el JWT ES256 con la clave pública y devuelve el cuerpo.
func verificar(t *testing.T, jwt string, pub *ecdsa.PublicKey) map[string]any {
	t.Helper()
	p := strings.Split(jwt, ".")
	if len(p) != 3 {
		t.Fatalf("JWT con %d partes", len(p))
	}
	firma, err := b64.DecodeString(p[2])
	if err != nil || len(firma) != 64 {
		t.Fatalf("firma: %v %d", err, len(firma))
	}
	h := sha256.Sum256([]byte(p[0] + "." + p[1]))
	if !ecdsa.Verify(pub, h[:], new(big.Int).SetBytes(firma[:32]), new(big.Int).SetBytes(firma[32:])) {
		t.Fatal("la firma no verifica")
	}
	var cab, cue map[string]any
	b, _ := b64.DecodeString(p[0])
	_ = json.Unmarshal(b, &cab)
	b, _ = b64.DecodeString(p[1])
	_ = json.Unmarshal(b, &cue)
	if cab["alg"] != "ES256" || cab["typ"] != "JWT" {
		t.Fatalf("cabecera: %v", cab)
	}
	return cue
}

func clienteDePrueba(t *testing.T) *Cliente {
	t.Helper()
	pub, priv, _ := GenerarClaves()
	c, err := Desde(pub, priv, "mailto:hola@tulocalweb.com.ar")
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestJWTReusadoPorServicio(t *testing.T) {
	c := clienteDePrueba(t)
	ahora := time.Unix(1_790_000_000, 0)
	c.Ahora = func() time.Time { return ahora }
	a, _ := c.Token("https://fcm.googleapis.com")
	cue := verificar(t, a, &c.Privada.PublicKey)
	if cue["aud"] != "https://fcm.googleapis.com" || cue["sub"] != "mailto:hola@tulocalweb.com.ar" ||
		cue["exp"] != float64(ahora.Add(12*time.Hour).Unix()) {
		t.Fatalf("cuerpo: %v", cue)
	}
	if b, _ := c.Token("https://fcm.googleapis.com"); b != a {
		t.Fatal("no reusó el JWT")
	}
	if b, _ := c.Token("https://web.push.apple.com"); b == a {
		t.Fatal("el mismo JWT para otro servicio")
	}
	ahora = ahora.Add(VidaJWT)
	if b, _ := c.Token("https://fcm.googleapis.com"); b == a {
		t.Fatal("no firmó otro al vencer")
	}
}

// Contra un servicio de push falso: encabezados, cifrado que el navegador abre, y las respuestas.
func TestEnviar(t *testing.T) {
	n := nuevoNavegador(t)
	var mu sync.Mutex
	var recibidos []*http.Request
	var cuerpos [][]byte
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		mu.Lock()
		recibidos, cuerpos = append(recibidos, r), append(cuerpos, b)
		mu.Unlock()
		switch r.URL.Path {
		case "/vencida":
			w.WriteHeader(http.StatusGone)
		case "/no-existe":
			w.WriteHeader(http.StatusNotFound)
		case "/lleno":
			w.WriteHeader(http.StatusTooManyRequests)
		case "/grande":
			w.WriteHeader(http.StatusRequestEntityTooLarge)
		default:
			w.WriteHeader(http.StatusCreated)
		}
	}))
	defer srv.Close()
	u, _ := url.Parse(srv.URL)
	c := clienteDePrueba(t)
	c.HTTP = srv.Client()
	ctx := context.Background()

	// Un host que no es un servicio de push no recibe nada.
	if err := c.Enviar(ctx, Envio{Suscripcion: n.suscripcion(srv.URL + "/ok"), Payload: []byte("{}")}); !errors.Is(err, ErrInvalida) {
		t.Fatalf("host desconocido: %v", err)
	}
	if len(recibidos) != 0 {
		t.Fatal("le pegó a un host desconocido")
	}
	c.Hosts = []string{u.Hostname()}

	msg := []byte(`{"title":"Nueva venta #3"}`)
	if err := c.Enviar(ctx, Envio{Suscripcion: n.suscripcion(srv.URL + "/ok"), Payload: msg, TTL: 24 * time.Hour, Tema: "venta-3 ñ/+"}); err != nil {
		t.Fatal(err)
	}
	r := recibidos[0]
	if r.Method != http.MethodPost || r.Header.Get("Content-Encoding") != "aes128gcm" || r.Header.Get("TTL") != "86400" ||
		r.Header.Get("Urgency") != "high" || r.Header.Get("Topic") != "venta-3" || r.Header.Get("Content-Type") != "application/octet-stream" {
		t.Fatalf("encabezados: %v", r.Header)
	}
	auth := r.Header.Get("Authorization")
	partes := strings.SplitN(strings.TrimPrefix(auth, "vapid t="), ", k=", 2)
	if !strings.HasPrefix(auth, "vapid t=") || len(partes) != 2 || partes[1] != c.ClavePublica() {
		t.Fatalf("Authorization: %q", auth)
	}
	if cue := verificar(t, partes[0], &c.Privada.PublicKey); cue["aud"] != srv.URL {
		t.Fatalf("aud: %v (quería %s)", cue["aud"], srv.URL)
	}
	if got := descifrar(t, cuerpos[0], n.priv, n.auth); string(got) != string(msg) {
		t.Fatalf("el navegador leyó %q", got)
	}

	for ruta, quiero := range map[string]func(error) bool{
		"/vencida":   func(e error) bool { return errors.Is(e, ErrVencida) },
		"/no-existe": func(e error) bool { return errors.Is(e, ErrVencida) },
		"/lleno": func(e error) bool {
			var we *Error
			return errors.As(e, &we) && we.Reintentable()
		},
		"/grande": func(e error) bool {
			var we *Error
			return errors.As(e, &we) && !we.Reintentable() && we.Estado == 413
		},
	} {
		err := c.Enviar(ctx, Envio{Suscripcion: n.suscripcion(srv.URL + ruta), Payload: msg})
		if !quiero(err) {
			t.Errorf("%s: %v", ruta, err)
		}
		if err != nil && strings.Contains(err.Error(), srv.URL) {
			t.Errorf("%s: el error trae el endpoint: %v", ruta, err)
		}
	}
	// Sin conexión: el error no trae el endpoint.
	srv.Close()
	if err := c.Enviar(ctx, Envio{Suscripcion: n.suscripcion(srv.URL + "/ok"), Payload: msg}); err == nil || strings.Contains(err.Error(), u.Host) {
		t.Fatalf("sin conexión: %v", err)
	}
}
