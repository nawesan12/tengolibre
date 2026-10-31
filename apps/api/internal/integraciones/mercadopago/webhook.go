package mercadopago

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Aviso es el cuerpo de una notificación de Webhooks.
type Aviso struct {
	Type   string          `json:"type"`
	Action string          `json:"action"`
	UserID json.RawMessage `json:"user_id"`
	Data   struct {
		ID json.RawMessage `json:"id"`
	} `json:"data"`
}

// Crudo saca las comillas de un id que puede venir como número o como texto.
func Crudo(r json.RawMessage) string {
	return strings.Trim(strings.TrimSpace(string(r)), `"`)
}

// ID del recurso avisado ("" si no sirve).
func (a Aviso) ID() string {
	id := Crudo(a.Data.ID)
	if len(id) > 64 {
		return ""
	}
	return id
}

// Usuario de Mercado Pago del vendedor (0 si no vino).
func (a Aviso) Usuario() int64 {
	n, _ := strconv.ParseInt(Crudo(a.UserID), 10, 64)
	return n
}

var alfanumerico = regexp.MustCompile(`^[A-Za-z0-9]+$`)

// Firmar arma el x-signature de un aviso (lo usan los tests para hacerse pasar por Mercado Pago).
func Firmar(secreto, requestID, dataID string, ts int64) string {
	return "ts=" + strconv.FormatInt(ts, 10) + ",v1=" + firma(secreto, requestID, dataID, strconv.FormatInt(ts, 10))
}

func firma(secreto, requestID, dataID, ts string) string {
	if alfanumerico.MatchString(dataID) {
		dataID = strings.ToLower(dataID)
	}
	var b strings.Builder
	if dataID != "" {
		b.WriteString("id:" + dataID + ";")
	}
	if requestID != "" {
		b.WriteString("request-id:" + requestID + ";")
	}
	b.WriteString("ts:" + ts + ";")
	m := hmac.New(sha256.New, []byte(secreto))
	m.Write([]byte(b.String()))
	return hex.EncodeToString(m.Sum(nil))
}

// FirmaValida verifica x-signature (HMAC-SHA256 de "id:{data.id};request-id:{x-request-id};ts:{ts};"
// con la clave secreta de los webhooks), en tiempo constante. dataID es el de la URL; si es
// alfanumérico va en minúsculas. Además rechaza avisos de más de 10 minutos.
func FirmaValida(secreto, encabezado, requestID, dataID string, ahora time.Time) bool {
	if secreto == "" || encabezado == "" {
		return false
	}
	var ts, v1 string
	for _, parte := range strings.Split(encabezado, ",") {
		k, v, _ := strings.Cut(strings.TrimSpace(parte), "=")
		switch strings.TrimSpace(k) {
		case "ts":
			ts = strings.TrimSpace(v)
		case "v1":
			v1 = strings.TrimSpace(v)
		}
	}
	if ts == "" || v1 == "" {
		return false
	}
	if !hmac.Equal([]byte(firma(secreto, requestID, dataID, ts)), []byte(v1)) {
		return false
	}
	// ts viene en milisegundos (a veces en segundos).
	n, err := strconv.ParseInt(ts, 10, 64)
	if err != nil {
		return false
	}
	cuando := time.UnixMilli(n)
	if n < 1e12 {
		cuando = time.Unix(n, 0)
	}
	return ahora.Sub(cuando).Abs() <= 10*time.Minute
}

// Receptor procesa un aviso ya verificado. Si devuelve error, se responde 500 y Mercado Pago reintenta.
type Receptor func(ctx context.Context, a Aviso) error

// Webhook arma el handler: primero la firma, después todo lo demás. Sin firma válida, 401 y nada se
// procesa. Cada tipo de aviso ("payment", "subscription_preapproval"…) va a su receptor; los demás se
// contestan 200 y se ignoran.
func Webhook(secreto string, ahora func() time.Time, log *slog.Logger, receptores map[string]Receptor) http.HandlerFunc {
	if log == nil {
		log = slog.Default()
	}
	return func(w http.ResponseWriter, r *http.Request) {
		b, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 64<<10))
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		var a Aviso
		if json.Unmarshal(b, &a) != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		dataID := r.URL.Query().Get("data.id")
		if dataID == "" {
			dataID = Crudo(a.Data.ID)
		}
		if !FirmaValida(secreto, r.Header.Get("x-signature"), r.Header.Get("x-request-id"), dataID, ahora()) {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if rec, ok := receptores[a.Type]; ok && a.ID() != "" {
			if err := rec(r.Context(), a); err != nil {
				log.Error("mercadopago: webhook", "tipo", a.Type, "error", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
		}
		w.WriteHeader(http.StatusOK)
	}
}
