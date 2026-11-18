package ia

import (
	"context"
	"encoding/json"
	"errors"
	"sync"

	"github.com/anthropics/anthropic-sdk-go"
)

// Falso: un cliente sin red para las pruebas (como sesion.Falso). Devuelve las respuestas en orden
// y guarda los pedidos que recibió.
type Falso struct {
	mu         sync.Mutex
	Respuestas []string
	Recibidos  []anthropic.MessageNewParams
	// Err: si no es nil, todas las llamadas fallan con esto.
	Err error
}

// Crear devuelve la próxima respuesta.
func (f *Falso) Crear(_ context.Context, p anthropic.MessageNewParams) (*anthropic.Message, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Recibidos = append(f.Recibidos, p)
	if f.Err != nil {
		return nil, f.Err
	}
	if len(f.Respuestas) == 0 {
		return nil, errors.New("ia.Falso: no quedan respuestas")
	}
	var m anthropic.Message
	if err := json.Unmarshal([]byte(f.Respuestas[0]), &m); err != nil {
		return nil, err
	}
	f.Respuestas = f.Respuestas[1:]
	return &m, nil
}

// Llamadas que recibió.
func (f *Falso) Llamadas() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.Recibidos)
}

func usoFalso() map[string]any {
	return map[string]any{"input_tokens": 120, "output_tokens": 40, "cache_read_input_tokens": 900, "cache_creation_input_tokens": 0}
}

// RespuestaTexto: un mensaje del modelo que contesta con texto.
func RespuestaTexto(texto string) string {
	b, _ := json.Marshal(map[string]any{"id": "msg_falso", "type": "message", "role": "assistant", "model": Modelo,
		"stop_reason": "end_turn", "content": []any{map[string]any{"type": "text", "text": texto}}, "usage": usoFalso()})
	return string(b)
}

// RespuestaHerramienta: un mensaje del modelo que pide una herramienta.
func RespuestaHerramienta(id, nombre string, entrada any) string {
	b, _ := json.Marshal(map[string]any{"id": "msg_falso", "type": "message", "role": "assistant", "model": Modelo,
		"stop_reason": "tool_use", "content": []any{map[string]any{"type": "tool_use", "id": id, "name": nombre, "input": entrada}},
		"usage": usoFalso()})
	return string(b)
}

// RespuestaRechazo: el modelo no quiere contestar.
func RespuestaRechazo() string {
	b, _ := json.Marshal(map[string]any{"id": "msg_falso", "type": "message", "role": "assistant", "model": Modelo,
		"stop_reason": "refusal", "content": []any{}, "usage": usoFalso()})
	return string(b)
}
