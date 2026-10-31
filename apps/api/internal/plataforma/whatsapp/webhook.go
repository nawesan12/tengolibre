package whatsapp

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strconv"
	"strings"
	"time"
)

// FirmaValida compara X-Hub-Signature-256 («sha256=<hex>») con el HMAC-SHA256 del cuerpo crudo con
// el secreto de la app de Meta. Sin secreto, nada es válido.
func FirmaValida(secreto string, cuerpo []byte, cabecera string) bool {
	if secreto == "" {
		return false
	}
	hexa, ok := strings.CutPrefix(strings.TrimSpace(cabecera), "sha256=")
	if !ok {
		return false
	}
	recibida, err := hex.DecodeString(hexa)
	if err != nil {
		return false
	}
	m := hmac.New(sha256.New, []byte(secreto))
	m.Write(cuerpo)
	return hmac.Equal(recibida, m.Sum(nil))
}

// Firmar devuelve la cabecera X-Hub-Signature-256 del cuerpo (para los tests y el servidor falso).
func Firmar(secreto string, cuerpo []byte) string {
	m := hmac.New(sha256.New, []byte(secreto))
	m.Write(cuerpo)
	return "sha256=" + hex.EncodeToString(m.Sum(nil))
}

// Respuesta es un botón que tocó el cliente (de una plantilla o de un mensaje interactivo).
type Respuesta struct {
	// TelefonoID: el número de la plataforma que recibió (metadata.phone_number_id).
	TelefonoID string
	// De: el número del cliente, como lo manda Meta (sin +).
	De string
	// ID: el wamid de la respuesta (Meta puede mandar el mismo webhook más de una vez).
	ID string
	// Contexto: el wamid del mensaje que se respondió.
	Contexto string
	// Payload: el que se mandó en el botón (o el id del botón interactivo); Texto, lo que dice.
	Payload, Texto string
	Cuando         time.Time
}

// Estado es un cambio de estado de un mensaje que mandamos (sent, delivered, read, failed).
type Estado struct {
	TelefonoID string
	ID         string // wamid
	Estado     string
	Cuando     time.Time
	// Error: con failed, lo que dijo Meta (ErrDestinoInvalido o *Error).
	Error error
}

type notificacion struct {
	Object string `json:"object"`
	Entry  []struct {
		Changes []struct {
			Field string `json:"field"`
			Value struct {
				Metadata struct {
					PhoneNumberID string `json:"phone_number_id"`
				} `json:"metadata"`
				Messages []struct {
					From      string `json:"from"`
					ID        string `json:"id"`
					Timestamp string `json:"timestamp"`
					Type      string `json:"type"`
					Context   struct {
						ID string `json:"id"`
					} `json:"context"`
					Button struct {
						Payload string `json:"payload"`
						Text    string `json:"text"`
					} `json:"button"`
					Interactive struct {
						Type        string `json:"type"`
						ButtonReply struct {
							ID    string `json:"id"`
							Title string `json:"title"`
						} `json:"button_reply"`
					} `json:"interactive"`
				} `json:"messages"`
				Statuses []struct {
					ID        string `json:"id"`
					Status    string `json:"status"`
					Timestamp string `json:"timestamp"`
					Errors    []struct {
						Code    int    `json:"code"`
						Title   string `json:"title"`
						Message string `json:"message"`
					} `json:"errors"`
				} `json:"statuses"`
			} `json:"value"`
		} `json:"changes"`
	} `json:"entry"`
}

func cuando(ts string) time.Time {
	n, err := strconv.ParseInt(ts, 10, 64)
	if err != nil {
		return time.Time{}
	}
	return time.Unix(n, 0)
}

// Interpretar saca del webhook los botones tocados y los estados de los mensajes. Lo demás (textos
// sueltos, imágenes, otros campos) se ignora.
func Interpretar(cuerpo []byte) ([]Respuesta, []Estado, error) {
	var n notificacion
	if err := json.Unmarshal(cuerpo, &n); err != nil {
		return nil, nil, err
	}
	var rs []Respuesta
	var es []Estado
	for _, e := range n.Entry {
		for _, c := range e.Changes {
			if c.Field != "messages" {
				continue
			}
			v := c.Value
			for _, m := range v.Messages {
				r := Respuesta{TelefonoID: v.Metadata.PhoneNumberID, De: m.From, ID: m.ID, Contexto: m.Context.ID, Cuando: cuando(m.Timestamp)}
				switch {
				case m.Type == "button":
					r.Payload, r.Texto = m.Button.Payload, m.Button.Text
				case m.Type == "interactive" && m.Interactive.Type == "button_reply":
					r.Payload, r.Texto = m.Interactive.ButtonReply.ID, m.Interactive.ButtonReply.Title
				default:
					continue
				}
				rs = append(rs, r)
			}
			for _, s := range v.Statuses {
				x := Estado{TelefonoID: v.Metadata.PhoneNumberID, ID: s.ID, Estado: s.Status, Cuando: cuando(s.Timestamp)}
				if s.Status == "failed" {
					x.Error = &Error{Mensaje: "falló sin detalle"}
					if len(s.Errors) > 0 {
						msj := s.Errors[0].Title
						if s.Errors[0].Message != "" && s.Errors[0].Message != msj {
							msj += ": " + s.Errors[0].Message
						}
						x.Error = ErrorDeCodigo(0, s.Errors[0].Code, msj)
					}
				}
				es = append(es, x)
			}
		}
	}
	return rs, es, nil
}
