// Package whatsapp habla con la WhatsApp Cloud API de Meta (Graph API): manda plantillas aprobadas
// (con botones de respuesta rápida o de link) y textos libres dentro de la ventana de 24 horas que
// abre la respuesta del cliente, valida la firma de los webhooks (X-Hub-Signature-256) y los
// interpreta. Sólo biblioteca estándar.
//
// Sin configuración (token o id del número), Desde devuelve nil y los envíos sobre un cliente nil
// no hacen nada: queda apagado sin romper nada, como apns y fcm.
package whatsapp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Base de la Graph API y versión por defecto (se cambia con TENGOLIBRE_WHATSAPP_VERSION).
const (
	Base           = "https://graph.facebook.com"
	VersionDefecto = "v23.0"
)

// ErrDestinoInvalido: Meta dice que a ese número no se le puede mandar (no tiene WhatsApp, lo
// bloqueó, no está en la lista de prueba). No se reintenta: hay que llamarlo.
var ErrDestinoInvalido = errors.New("whatsapp: el número no recibe mensajes")

// Error es otra respuesta no exitosa de Meta (nunca lleva el token ni el número).
type Error struct {
	Estado  int
	Codigo  int
	Mensaje string
}

func (e *Error) Error() string {
	return fmt.Sprintf("whatsapp: HTTP %d, código %d: %s", e.Estado, e.Codigo, e.Mensaje)
}

// Reintentable dice si vale la pena volver a probar más tarde (Meta caído, límites de envío).
func (e *Error) Reintentable() bool {
	if e.Estado == http.StatusTooManyRequests || e.Estado >= 500 {
		return true
	}
	switch e.Codigo {
	case 1, 2, 4, 80007, 130429, 131000, 131016, 131048, 131056:
		return true
	}
	return false
}

// destinoInvalido: los códigos que dicen que el número no sirve (no del lado de TengoLibre).
func destinoInvalido(codigo int) bool {
	switch codigo {
	case 131026, 131030, 131021, 131050:
		return true
	}
	return false
}

// ErrorDeCodigo arma el error de un código de Meta (también lo usan los estados del webhook).
func ErrorDeCodigo(estado, codigo int, mensaje string) error {
	if destinoInvalido(codigo) {
		return ErrDestinoInvalido
	}
	return &Error{Estado: estado, Codigo: codigo, Mensaje: mensaje}
}

// Cliente manda mensajes desde el número de la plataforma. Usalo por puntero; es seguro entre
// goroutines.
type Cliente struct {
	Token, TelefonoID string
	// Version de la Graph API. Vacía: VersionDefecto.
	Version string
	// Base: en los tests, la URL del servidor falso. Vacía: la de Meta.
	Base string
	// HTTP: en los tests, el del servidor falso. Vacío: uno con 15 s de espera.
	HTTP *http.Client
}

// Desde arma el cliente con la configuración. Si falta el token o el id del número, devuelve nil:
// el envío queda apagado.
func Desde(token, telefonoID, version string) *Cliente {
	if token == "" || telefonoID == "" {
		return nil
	}
	return &Cliente{Token: token, TelefonoID: telefonoID, Version: version}
}

// Activo dice si hay cliente (nil: apagado).
func (c *Cliente) Activo() bool { return c != nil }

var clientePorDefecto = &http.Client{Timeout: 15 * time.Second}

func (c *Cliente) cli() *http.Client {
	if c.HTTP != nil {
		return c.HTTP
	}
	return clientePorDefecto
}

func (c *Cliente) url() string {
	base, v := c.Base, c.Version
	if base == "" {
		base = Base
	}
	if v == "" {
		v = VersionDefecto
	}
	return strings.TrimRight(base, "/") + "/" + v + "/" + c.TelefonoID + "/messages"
}

// Boton de una plantilla: de respuesta rápida (Payload, lo que vuelve en el webhook) o de link con
// sufijo variable (Sufijo, lo que completa la URL aprobada).
type Boton struct {
	Payload string
	Sufijo  string
}

// Plantilla a mandar: nombre e idioma como se aprobaron en Meta, los parámetros del cuerpo en orden
// ({{1}}, {{2}}…) y los botones en el orden en que están en la plantilla.
type Plantilla struct {
	Para    string // E.164 (+5491112345678)
	Nombre  string
	Idioma  string
	Cuerpo  []string
	Botones []Boton
}

type parametro struct {
	Type    string `json:"type"`
	Text    string `json:"text,omitempty"`
	Payload string `json:"payload,omitempty"`
}

type componente struct {
	Type       string      `json:"type"`
	SubType    string      `json:"sub_type,omitempty"`
	Index      string      `json:"index,omitempty"`
	Parameters []parametro `json:"parameters"`
}

// CuerpoPlantilla arma el JSON de /messages para la plantilla (exportado para los tests).
func CuerpoPlantilla(p Plantilla) []byte {
	var comps []componente
	if len(p.Cuerpo) > 0 {
		c := componente{Type: "body"}
		for _, v := range p.Cuerpo {
			c.Parameters = append(c.Parameters, parametro{Type: "text", Text: v})
		}
		comps = append(comps, c)
	}
	for i, b := range p.Botones {
		switch {
		case b.Payload != "":
			comps = append(comps, componente{Type: "button", SubType: "quick_reply", Index: fmt.Sprint(i),
				Parameters: []parametro{{Type: "payload", Payload: b.Payload}}})
		case b.Sufijo != "":
			comps = append(comps, componente{Type: "button", SubType: "url", Index: fmt.Sprint(i),
				Parameters: []parametro{{Type: "text", Text: b.Sufijo}}})
		}
	}
	idioma := p.Idioma
	if idioma == "" {
		idioma = "es_AR"
	}
	b, _ := json.Marshal(map[string]any{
		"messaging_product": "whatsapp",
		"recipient_type":    "individual",
		"to":                strings.TrimPrefix(p.Para, "+"),
		"type":              "template",
		"template": map[string]any{
			"name":       p.Nombre,
			"language":   map[string]string{"code": idioma},
			"components": comps,
		},
	})
	return b
}

// CuerpoTexto arma el JSON de un texto libre (sólo vale dentro de las 24 horas desde que el cliente
// escribió). Con link, WhatsApp muestra la vista previa.
func CuerpoTexto(para, texto string) []byte {
	b, _ := json.Marshal(map[string]any{
		"messaging_product": "whatsapp",
		"recipient_type":    "individual",
		"to":                strings.TrimPrefix(para, "+"),
		"type":              "text",
		"text":              map[string]any{"body": texto, "preview_url": strings.Contains(texto, "https://")},
	})
	return b
}

// EnviarPlantilla manda la plantilla y devuelve el id del mensaje (wamid). Sobre un cliente nil no
// hace nada.
func (c *Cliente) EnviarPlantilla(ctx context.Context, p Plantilla) (string, error) {
	if c == nil {
		return "", nil
	}
	return c.enviar(ctx, CuerpoPlantilla(p))
}

// EnviarTexto manda un texto libre y devuelve el wamid. Sobre un cliente nil no hace nada.
func (c *Cliente) EnviarTexto(ctx context.Context, para, texto string) (string, error) {
	if c == nil {
		return "", nil
	}
	return c.enviar(ctx, CuerpoTexto(para, texto))
}

func (c *Cliente) enviar(ctx context.Context, cuerpo []byte) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url(), bytes.NewReader(cuerpo))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Content-Type", "application/json")
	res, err := c.cli().Do(req)
	if err != nil {
		// El error de red puede traer la URL con el id del número: no se propaga.
		return "", errors.New("whatsapp: no se pudo conectar")
	}
	defer res.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(res.Body, 64<<10))
	var r struct {
		Messages []struct {
			ID string `json:"id"`
		} `json:"messages"`
		Error *struct {
			Message   string `json:"message"`
			Code      int    `json:"code"`
			ErrorData struct {
				Details string `json:"details"`
			} `json:"error_data"`
		} `json:"error"`
	}
	_ = json.Unmarshal(b, &r)
	if res.StatusCode >= 200 && res.StatusCode < 300 && len(r.Messages) > 0 && r.Messages[0].ID != "" {
		return r.Messages[0].ID, nil
	}
	if r.Error != nil {
		msj := r.Error.Message
		if r.Error.ErrorData.Details != "" {
			msj += " (" + r.Error.ErrorData.Details + ")"
		}
		return "", ErrorDeCodigo(res.StatusCode, r.Error.Code, msj)
	}
	return "", &Error{Estado: res.StatusCode, Mensaje: "respuesta sin id de mensaje"}
}
