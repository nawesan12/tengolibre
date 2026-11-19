// Package ia: la pieza común para hablar con Claude (ver docs-ia/asistente-ia.md de Grupo Vimo).
// Cliente detrás de una interfaz (en las pruebas, uno falso y sin red), armado del prompt pensado
// para el caché, bucle de herramientas con tope de vueltas, medición y topes en la base, y caché de
// respuestas. Lo que se le pregunta y qué herramientas tiene lo decide cada dominio.
//
// Regla: nunca se llama al modelo con una transacción de base abierta. Las herramientas abren y
// cierran la suya; el bucle sólo las llama entre una vuelta y otra.
package ia

import (
	"context"
	"errors"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

// Modelo: uno solo para todo (Claude Haiku 5.5), con esfuerzo bajo fijo. Cambiar cualquiera de los
// dos invalida el caché de prompts.
const (
	Modelo   = "claude-haiku-5-5"
	Esfuerzo = anthropic.OutputConfigEffortLow
)

// Tiempos: cada llamada tiene su tope y la respuesta entera otro, con margen dentro del WriteTimeout
// del servidor (30 s) para el catálogo, la reserva del cupo y la respuesta «probá de nuevo».
const (
	TopePorLlamada = 15 * time.Second
	TopeRespuesta  = 22 * time.Second
)

// Permanente: el error del modelo no se arregla reintentando (clave revocada, modelo retirado,
// facturación, pedido mal armado). Quien llama apaga la IA un rato y avisa en el log como error.
func Permanente(err error) (int, bool) {
	var e *anthropic.Error
	if errors.As(err, &e) {
		switch e.StatusCode {
		case 400, 401, 403, 404:
			return e.StatusCode, true
		}
	}
	return 0, false
}

// Cliente es la única puerta al modelo.
type Cliente interface {
	Crear(ctx context.Context, p anthropic.MessageNewParams) (*anthropic.Message, error)
}

type clienteAnthropic struct{ c anthropic.Client }

// NuevoCliente con la clave de la API (ANTHROPIC_API_KEY). Sin clave, nil: la app anda sólo con su
// motor propio. Un solo reintento (el SDK reintenta 429, 5xx y errores de conexión).
func NuevoCliente(clave string) Cliente {
	if clave == "" {
		return nil
	}
	return &clienteAnthropic{c: anthropic.NewClient(option.WithAPIKey(clave), option.WithMaxRetries(1),
		option.WithRequestTimeout(TopePorLlamada))}
}

func (c *clienteAnthropic) Crear(ctx context.Context, p anthropic.MessageNewParams) (*anthropic.Message, error) {
	return c.c.Messages.New(ctx, p)
}
