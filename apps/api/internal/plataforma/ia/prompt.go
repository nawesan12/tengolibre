package ia

import (
	"encoding/json"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/anthropics/anthropic-sdk-go"
)

// Herramienta que el modelo puede pedir. Esquema: un JSON Schema de objeto (type, properties,
// required).
type Herramienta struct {
	Nombre, Descripcion string
	Esquema             map[string]any
}

// Turno de la conversación previa. Rol: "usuario" o "asistente".
type Turno struct {
	Rol   string `json:"rol"`
	Texto string `json:"texto"`
}

// Pedido al modelo.
type Pedido struct {
	// Sistema: el texto fijo de la app (sin fechas, sin nombre del negocio, sin datos).
	Sistema string
	// Datos del negocio armados de forma determinística (ver Redondear): primer bloque del primer
	// mensaje, con su propio breakpoint.
	Datos        string
	Historial    []Turno
	Pregunta     string
	Herramientas []Herramienta
	// MaxTokens de la ruta (chat 800).
	MaxTokens int64
}

// Historial: los últimos turnos, cada uno recortado.
const (
	MaxTurnos       = 8
	MaxLargoDeTurno = 600
)

// Armar los parámetros en el orden que lee el caché: herramientas (siempre iguales, por nombre) →
// system fijo (breakpoint 1) → datos del negocio (breakpoint 2) → conversación (caché automático).
func Armar(p Pedido) anthropic.MessageNewParams {
	hs := append([]Herramienta(nil), p.Herramientas...)
	sort.Slice(hs, func(i, j int) bool { return hs[i].Nombre < hs[j].Nombre })
	tools := make([]anthropic.ToolUnionParam, 0, len(hs))
	for _, h := range hs {
		esq := anthropic.ToolInputSchemaParam{Properties: h.Esquema["properties"]}
		if req, ok := h.Esquema["required"].([]string); ok {
			esq.Required = req
		}
		t := anthropic.ToolParam{Name: h.Nombre, Description: anthropic.String(h.Descripcion), InputSchema: esq}
		tools = append(tools, anthropic.ToolUnionParam{OfTool: &t})
	}
	max := p.MaxTokens
	if max <= 0 {
		max = 800
	}
	params := anthropic.MessageNewParams{
		Model:     Modelo,
		MaxTokens: max,
		System:    []anthropic.TextBlockParam{{Text: p.Sistema, CacheControl: anthropic.NewCacheControlEphemeralParam()}},
		Messages:  mensajes(p),
		// El resto de la conversación, con el caché automático.
		CacheControl: anthropic.NewCacheControlEphemeralParam(),
		OutputConfig: anthropic.OutputConfigParam{Effort: Esfuerzo},
	}
	if len(tools) > 0 {
		params.Tools = tools
		params.ToolChoice = anthropic.ToolChoiceUnionParam{OfAuto: &anthropic.ToolChoiceAutoParam{}}
	}
	return params
}

// mensajes: los datos van primero en el primer mensaje del usuario; los turnos se alternan (dos
// seguidos del mismo rol se juntan) y la conversación siempre empieza por el usuario.
func mensajes(p Pedido) []anthropic.MessageParam {
	turnos := recortar(p.Historial)
	turnos = append(turnos, Turno{Rol: "usuario", Texto: p.Pregunta})
	type grupo struct {
		usuario bool
		textos  []string
	}
	var gs []grupo
	for _, t := range turnos {
		u := t.Rol != "asistente"
		if len(gs) == 0 && !u {
			continue
		}
		if len(gs) > 0 && gs[len(gs)-1].usuario == u {
			gs[len(gs)-1].textos = append(gs[len(gs)-1].textos, t.Texto)
			continue
		}
		gs = append(gs, grupo{usuario: u, textos: []string{t.Texto}})
	}
	out := make([]anthropic.MessageParam, 0, len(gs))
	for i, g := range gs {
		var bloques []anthropic.ContentBlockParamUnion
		if i == 0 {
			datos := anthropic.TextBlockParam{Text: "<datos_del_negocio>\n" + p.Datos + "\n</datos_del_negocio>",
				CacheControl: anthropic.NewCacheControlEphemeralParam()}
			bloques = append(bloques, anthropic.ContentBlockParamUnion{OfText: &datos})
		}
		for _, t := range g.textos {
			bloques = append(bloques, anthropic.NewTextBlock(t))
		}
		if g.usuario {
			out = append(out, anthropic.NewUserMessage(bloques...))
		} else {
			out = append(out, anthropic.NewAssistantMessage(bloques...))
		}
	}
	return out
}

func recortar(h []Turno) []Turno {
	if len(h) > MaxTurnos {
		h = h[len(h)-MaxTurnos:]
	}
	out := make([]Turno, 0, len(h))
	for _, t := range h {
		x := strings.TrimSpace(t.Texto)
		if x == "" {
			continue
		}
		if utf8.RuneCountInString(x) > MaxLargoDeTurno {
			x = string([]rune(x)[:MaxLargoDeTurno-1]) + "…"
		}
		out = append(out, Turno{Rol: t.Rol, Texto: x})
	}
	return out
}

// Prefijo: los bytes que deciden el caché hasta el segundo breakpoint (modelo, esfuerzo, elección
// de herramientas, herramientas, system y el bloque de datos). Dos preguntas del mismo negocio en
// el mismo rato tienen que dar exactamente lo mismo; lo usan las pruebas.
func Prefijo(p anthropic.MessageNewParams) []byte {
	var datos any
	if len(p.Messages) > 0 && len(p.Messages[0].Content) > 0 {
		datos = p.Messages[0].Content[0]
	}
	b, _ := json.Marshal(struct {
		Modelo   any `json:"model"`
		Salida   any `json:"output_config"`
		Eleccion any `json:"tool_choice"`
		Tools    any `json:"tools"`
		System   any `json:"system"`
		Datos    any `json:"datos"`
	}{p.Model, p.OutputConfig, p.ToolChoice, p.Tools, p.System, datos})
	return b
}
