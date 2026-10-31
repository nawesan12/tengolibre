// Package webpush manda notificaciones Web Push (el estándar de los navegadores, sin Firebase) al
// panel abierto o instalado en el navegador: el cuerpo va cifrado como pide RFC 8291 (aes128gcm de
// RFC 8188, con ECDH P-256 y HKDF) y el servidor se identifica con VAPID (RFC 8292: un JWT ES256 por
// servicio de push, reusado mientras vale). Sólo biblioteca estándar.
//
// Sin configuración (clave pública, privada o sujeto), Desde devuelve nil y Enviar sobre un cliente
// nil no hace nada: el envío queda apagado sin romper nada. Calcado del de Tu Local Web (mostrador-v2).
package webpush

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdh"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/hkdf"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

// TamanoRegistro: el rs del encabezado aes128gcm. Un solo registro (los avisos son chicos).
const TamanoRegistro = 4096

// MaxPayload: lo más largo que se cifra. El cuerpo cifrado (86 de encabezado + 16 de etiqueta + 1
// de delimitador) tiene que entrar en los 4096 bytes que aceptan todos los servicios.
const MaxPayload = TamanoRegistro - 86 - 16 - 1

// VidaJWT: cuánto se reusa el JWT de VAPID para un servicio; vence a las 12 h (el máximo es 24).
const VidaJWT = 10 * time.Hour

// ErrVencida: el servicio de push dice que la suscripción ya no existe (404 o 410). Hay que borrarla.
var ErrVencida = errors.New("webpush: suscripción vencida")

// ErrInvalida: la suscripción no tiene la forma esperada (endpoint, p256dh o auth).
var ErrInvalida = errors.New("webpush: suscripción inválida")

// Error es otra respuesta no exitosa del servicio de push (nunca lleva el endpoint).
type Error struct {
	Estado int
}

func (e *Error) Error() string { return fmt.Sprintf("webpush: HTTP %d", e.Estado) }

// Reintentable dice si vale la pena volver a probar más tarde (servicio caído, demasiados pedidos).
func (e *Error) Reintentable() bool {
	return e.Estado == http.StatusTooManyRequests || e.Estado >= 500
}

var b64 = base64.RawURLEncoding

// decodificar acepta base64url con o sin relleno (y el base64 común, por las dudas).
func decodificar(s string) ([]byte, error) {
	s = strings.TrimRight(strings.TrimSpace(s), "=")
	s = strings.NewReplacer("+", "-", "/", "_").Replace(s)
	return b64.DecodeString(s)
}

// Suscripcion es lo que da PushSubscription.toJSON() en el navegador.
type Suscripcion struct {
	Endpoint string `json:"endpoint"`
	P256dh   string `json:"p256dh"`
	Auth     string `json:"auth"`
}

// Servicios de push conocidos (por sufijo del host). El worker hace un POST al endpoint que manda el
// navegador: sólo a estos, para que nadie use la suscripción para hacerle pegar a otro lado.
var Servicios = []string{
	"fcm.googleapis.com",        // Chrome, Edge en Android, Samsung Internet, Opera
	".push.apple.com",           // Safari en iPhone (16.4+, instalada) y en Mac
	"push.services.mozilla.com", // Firefox
	".notify.windows.com",       // Edge en Windows
}

func servicioConocido(host string) bool {
	host = strings.ToLower(host)
	for _, s := range Servicios {
		if host == strings.TrimPrefix(s, ".") || strings.HasSuffix(host, "."+strings.TrimPrefix(s, ".")) {
			return true
		}
	}
	return false
}

// Normalizar valida la suscripción y deja las claves en base64url sin relleno.
func Normalizar(s Suscripcion) (Suscripcion, error) {
	s.Endpoint = strings.TrimSpace(s.Endpoint)
	u, err := url.Parse(s.Endpoint)
	if err != nil || u.Scheme != "https" || u.User != nil || len(s.Endpoint) > 1000 || !servicioConocido(u.Hostname()) {
		return s, ErrInvalida
	}
	if u.Port() != "" && u.Port() != "443" {
		return s, ErrInvalida
	}
	p, err := decodificar(s.P256dh)
	if err != nil {
		return s, ErrInvalida
	}
	if _, err := ecdh.P256().NewPublicKey(p); err != nil {
		return s, ErrInvalida
	}
	a, err := decodificar(s.Auth)
	if err != nil || len(a) != 16 {
		return s, ErrInvalida
	}
	s.P256dh, s.Auth = b64.EncodeToString(p), b64.EncodeToString(a)
	return s, nil
}

// Cifrar arma el cuerpo aes128gcm (RFC 8291 §3.4) para la suscripción, con una clave efímera y una
// sal nuevas en cada llamada.
func Cifrar(payload []byte, p256dh, auth string) ([]byte, error) {
	efimera, err := ecdh.P256().GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	sal := make([]byte, 16)
	if _, err := rand.Read(sal); err != nil {
		return nil, err
	}
	return CifrarCon(payload, p256dh, auth, efimera, sal)
}

// CifrarCon es Cifrar con la clave efímera y la sal dadas (los vectores de prueba del RFC).
func CifrarCon(payload []byte, p256dh, auth string, efimera *ecdh.PrivateKey, sal []byte) ([]byte, error) {
	if len(payload) > MaxPayload {
		return nil, fmt.Errorf("webpush: el aviso pesa %d bytes (máximo %d)", len(payload), MaxPayload)
	}
	if len(sal) != 16 {
		return nil, errors.New("webpush: la sal tiene que tener 16 bytes")
	}
	uaPub, err := decodificar(p256dh)
	if err != nil {
		return nil, ErrInvalida
	}
	ua, err := ecdh.P256().NewPublicKey(uaPub)
	if err != nil {
		return nil, ErrInvalida
	}
	secretoAuth, err := decodificar(auth)
	if err != nil || len(secretoAuth) != 16 {
		return nil, ErrInvalida
	}
	compartido, err := efimera.ECDH(ua)
	if err != nil {
		return nil, err
	}
	asPub := efimera.PublicKey().Bytes()
	// IKM = HKDF(auth, ecdh, "WebPush: info" ‖ 0 ‖ ua_public ‖ as_public, 32)
	info := append(append([]byte("WebPush: info\x00"), uaPub...), asPub...)
	prkClave, err := hkdf.Extract(sha256.New, compartido, secretoAuth)
	if err != nil {
		return nil, err
	}
	ikm, err := hkdf.Expand(sha256.New, prkClave, string(info), 32)
	if err != nil {
		return nil, err
	}
	// CEK y nonce de RFC 8188 con la sal.
	prk, err := hkdf.Extract(sha256.New, ikm, sal)
	if err != nil {
		return nil, err
	}
	cek, err := hkdf.Expand(sha256.New, prk, "Content-Encoding: aes128gcm\x00", 16)
	if err != nil {
		return nil, err
	}
	nonce, err := hkdf.Expand(sha256.New, prk, "Content-Encoding: nonce\x00", 12)
	if err != nil {
		return nil, err
	}
	bloque, err := aes.NewCipher(cek)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(bloque)
	if err != nil {
		return nil, err
	}
	// Un solo registro: el texto y el delimitador 0x02 (último registro), sin relleno.
	plano := append(append(make([]byte, 0, len(payload)+1), payload...), 2)
	// Encabezado: sal ‖ rs (4 bytes) ‖ largo del keyid (1) ‖ keyid (la clave pública efímera).
	out := make([]byte, 0, 16+4+1+len(asPub)+len(plano)+gcm.Overhead())
	out = append(out, sal...)
	out = binary.BigEndian.AppendUint32(out, TamanoRegistro)
	out = append(out, byte(len(asPub)))
	out = append(out, asPub...)
	return gcm.Seal(out, nonce, plano, nil), nil
}

// Cliente manda los avisos con las claves VAPID. Usalo por puntero; es seguro entre goroutines.
type Cliente struct {
	// Publica: la clave pública VAPID (punto P-256 sin comprimir, 65 bytes), la misma que usa el
	// navegador como applicationServerKey.
	Publica []byte
	Privada *ecdsa.PrivateKey
	// Sujeto: un mailto: o https: de contacto para los servicios de push.
	Sujeto string
	// HTTP: en los tests, el del servidor falso. Vacío: uno con 15 s de espera.
	HTTP *http.Client
	// Ahora: el reloj (tests). Vacío: time.Now.
	Ahora func() time.Time
	// Hosts: permite otros servicios además de los conocidos (los tests, con su servidor falso).
	Hosts []string

	mu   sync.Mutex
	jwts map[string]jwtGuardado
}

type jwtGuardado struct {
	jwt     string
	firmado time.Time
}

// Desde arma el cliente con la configuración (las dos claves en base64url, como las da
// `go run ./cmd/vapid` o `npx web-push generate-vapid-keys`). Si falta algo, devuelve nil y ningún
// error: el envío queda apagado. Claves que no se pueden leer sí son error.
func Desde(publica, privada, sujeto string) (*Cliente, error) {
	if publica == "" || privada == "" || sujeto == "" {
		return nil, nil
	}
	if !strings.HasPrefix(sujeto, "mailto:") && !strings.HasPrefix(sujeto, "https://") {
		return nil, errors.New("webpush: el sujeto tiene que empezar con mailto: o https://")
	}
	pub, err := decodificar(publica)
	if err != nil {
		return nil, errors.New("webpush: la clave pública no es base64url")
	}
	priv, err := decodificar(privada)
	if err != nil || len(priv) != 32 {
		return nil, errors.New("webpush: la clave privada tiene que ser de 32 bytes en base64url")
	}
	k, err := ecdh.P256().NewPrivateKey(priv)
	if err != nil {
		return nil, fmt.Errorf("webpush: la clave privada no sirve: %w", err)
	}
	if !bytes.Equal(k.PublicKey().Bytes(), pub) {
		return nil, errors.New("webpush: la clave pública no corresponde a la privada")
	}
	return &Cliente{Publica: pub, Privada: aECDSA(priv, pub), Sujeto: sujeto}, nil
}

// aECDSA arma la clave de firma con los bytes ya validados por crypto/ecdh.
func aECDSA(priv, pub []byte) *ecdsa.PrivateKey {
	k := &ecdsa.PrivateKey{D: new(big.Int).SetBytes(priv)}
	k.Curve = elliptic.P256()
	k.X, k.Y = new(big.Int).SetBytes(pub[1:33]), new(big.Int).SetBytes(pub[33:65])
	return k
}

// GenerarClaves devuelve un par VAPID nuevo en base64url (pública de 65 bytes, privada de 32).
func GenerarClaves() (publica, privada string, err error) {
	k, err := ecdh.P256().GenerateKey(rand.Reader)
	if err != nil {
		return "", "", err
	}
	return b64.EncodeToString(k.PublicKey().Bytes()), b64.EncodeToString(k.Bytes()), nil
}

// ClavePublica en base64url, para el navegador (applicationServerKey). Vacía si está apagado.
func (c *Cliente) ClavePublica() string {
	if c == nil {
		return ""
	}
	return b64.EncodeToString(c.Publica)
}

func (c *Cliente) ahora() time.Time {
	if c.Ahora != nil {
		return c.Ahora()
	}
	return time.Now()
}

// Token devuelve el JWT VAPID para el origen del servicio de push (aud), firmando uno nuevo si el
// guardado tiene más de VidaJWT.
func (c *Cliente) Token(aud string) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	ahora := c.ahora()
	if g, ok := c.jwts[aud]; ok && ahora.Sub(g.firmado) < VidaJWT {
		return g.jwt, nil
	}
	t, err := FirmarJWT(c.Privada, aud, c.Sujeto, ahora)
	if err != nil {
		return "", err
	}
	if c.jwts == nil {
		c.jwts = map[string]jwtGuardado{}
	}
	c.jwts[aud] = jwtGuardado{t, ahora}
	return t, nil
}

// FirmarJWT arma el JWT de VAPID: {"typ":"JWT","alg":"ES256"} . {"aud","exp","sub"}, con la firma
// r‖s de 64 bytes (JWS, no DER). Vence a las 12 h.
func FirmarJWT(clave *ecdsa.PrivateKey, aud, sujeto string, ahora time.Time) (string, error) {
	if clave == nil {
		return "", errors.New("webpush: sin clave")
	}
	cab, _ := json.Marshal(map[string]string{"typ": "JWT", "alg": "ES256"})
	cue, _ := json.Marshal(map[string]any{"aud": aud, "exp": ahora.Add(12 * time.Hour).Unix(), "sub": sujeto})
	firmable := b64.EncodeToString(cab) + "." + b64.EncodeToString(cue)
	h := sha256.Sum256([]byte(firmable))
	r, s, err := ecdsa.Sign(rand.Reader, clave, h[:])
	if err != nil {
		return "", err
	}
	firma := make([]byte, 64)
	r.FillBytes(firma[:32])
	s.FillBytes(firma[32:])
	return firmable + "." + b64.EncodeToString(firma), nil
}

// Envio es un aviso para una suscripción.
type Envio struct {
	Suscripcion
	Payload []byte
	// TTL: cuánto lo guarda el servicio si el celular está apagado. Cero: una hora.
	TTL time.Duration
	// Tema (Topic): un aviso pendiente con el mismo tema se reemplaza (los reintentos del outbox).
	// Hasta 32 caracteres de base64url; lo demás se descarta.
	Tema string
}

func (c *Cliente) cli() *http.Client {
	if c.HTTP != nil {
		return c.HTTP
	}
	return clientePorDefecto
}

var clientePorDefecto = &http.Client{Timeout: 15 * time.Second}

// tema deja sólo lo que acepta el encabezado Topic.
func tema(s string) string {
	var b strings.Builder
	for _, r := range s {
		if b.Len() == 32 {
			break
		}
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func (c *Cliente) permitido(host string) bool {
	if servicioConocido(host) {
		return true
	}
	for _, h := range c.Hosts {
		if strings.EqualFold(h, host) {
			return true
		}
	}
	return false
}

// Enviar cifra y manda el aviso. Sobre un cliente nil no hace nada. Devuelve ErrVencida si hay que
// borrar la suscripción, o *Error con lo que respondió el servicio.
func (c *Cliente) Enviar(ctx context.Context, e Envio) error {
	if c == nil {
		return nil
	}
	u, err := url.Parse(e.Endpoint)
	if err != nil || u.Scheme != "https" || !c.permitido(u.Hostname()) {
		return ErrInvalida
	}
	cuerpo, err := Cifrar(e.Payload, e.P256dh, e.Auth)
	if err != nil {
		return err
	}
	jwt, err := c.Token(u.Scheme + "://" + u.Host)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, e.Endpoint, bytes.NewReader(cuerpo))
	if err != nil {
		return ErrInvalida
	}
	ttl := e.TTL
	if ttl <= 0 {
		ttl = time.Hour
	}
	req.Header.Set("Authorization", "vapid t="+jwt+", k="+c.ClavePublica())
	req.Header.Set("Content-Encoding", "aes128gcm")
	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set("TTL", strconv.Itoa(int(ttl.Seconds())))
	req.Header.Set("Urgency", "high")
	if t := tema(e.Tema); t != "" {
		req.Header.Set("Topic", t)
	}
	res, err := c.cli().Do(req)
	if err != nil {
		// El error de red trae el endpoint (que identifica al celular): no se propaga.
		return errors.New("webpush: no se pudo conectar")
	}
	defer res.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(res.Body, 4<<10))
	switch {
	case res.StatusCode >= 200 && res.StatusCode < 300:
		return nil
	case res.StatusCode == http.StatusNotFound || res.StatusCode == http.StatusGone:
		return ErrVencida
	}
	return &Error{Estado: res.StatusCode}
}
