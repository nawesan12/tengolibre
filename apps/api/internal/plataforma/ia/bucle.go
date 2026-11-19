package ia

import (
	"context"
	"encoding/json"
	"log/slog"
	"math"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
)

// MaxVueltas: llamadas al modelo por respuesta. En la última ya no se le ofrecen herramientas, así
// contesta con lo que juntó en vez de quedar a medias.
const MaxVueltas = 4

// Ejecutor corre una herramienta que pidió el modelo y devuelve el texto para el tool_result.
// esError: el resultado es un error que el modelo tiene que leer (no corta el bucle).
type Ejecutor func(ctx context.Context, nombre string, entrada json.RawMessage) (resultado string, esError bool)

// Uso de una respuesta (sumando las vueltas).
type Uso struct {
	Pedidos                                   int
	Entrada, CacheLeido, CacheEscrito, Salida int64
}

func (u *Uso) sumar(x anthropic.Usage) {
	u.Pedidos++
	u.Entrada += x.InputTokens
	u.CacheLeido += x.CacheReadInputTokens
	u.CacheEscrito += x.CacheCreationInputTokens
	u.Salida += x.OutputTokens
}

// Precios de Haiku 5.5 en micro-dólares por token: USD 0,10 / 0,50 por millón de entrada / salida;
// leer del caché, 0,1×; escribirlo (5 min), 1,25×.
const (
	precioEntrada      = 0.10
	precioCacheLeido   = 0.01
	precioCacheEscrito = 0.125
	precioSalida       = 0.50
)

// CostoMicro: lo que cuesta, en millonésimas de dólar.
func (u Uso) CostoMicro() int64 {
	return int64(math.Round(float64(u.Entrada)*precioEntrada + float64(u.CacheLeido)*precioCacheLeido +
		float64(u.CacheEscrito)*precioCacheEscrito + float64(u.Salida)*precioSalida))
}

// Resultado de una conversación.
type Resultado struct {
	Texto string
	Uso   Uso
	// Herramientas que pidió el modelo, en orden (para saber si escribió algo).
	Herramientas []string
	// Rechazo: el modelo no quiso contestar (stop_reason refusal).
	Rechazo bool
	// Cortada: la respuesta llegó al tope de tokens (stop_reason max_tokens).
	Cortada bool
}

// Frases fijas para lo que no sale del modelo.
const (
	FraseRechazo  = "No puedo ayudarte con eso."
	FraseSinVuelt = "No llegué a resolverlo. ¿Me lo pedís de otra forma, más concreto?"
)

// Conversar corre el bucle de herramientas: pregunta, el modelo pide herramientas, se corren (fuera
// de toda transacción) y se le devuelven, hasta que contesta o se acaban las vueltas.
func Conversar(ctx context.Context, cli Cliente, p Pedido, correr Ejecutor) (Resultado, error) {
	ctx, cancelar := context.WithTimeout(ctx, TopeRespuesta)
	defer cancelar()
	params := Armar(p)
	var r Resultado
	for vuelta := 1; vuelta <= MaxVueltas; vuelta++ {
		if vuelta == MaxVueltas && len(params.Tools) > 0 {
			// Última vuelta: que conteste. Cambiar tool_choice invalida sólo el caché de los mensajes.
			params.ToolChoice = anthropic.ToolChoiceUnionParam{OfNone: &anthropic.ToolChoiceNoneParam{}}
		}
		llamada, cortar := context.WithTimeout(ctx, TopePorLlamada)
		resp, err := cli.Crear(llamada, params)
		cortar()
		if err != nil {
			return r, err
		}
		r.Uso.sumar(resp.Usage)
		// Si el caché leído da 0 en pedidos seguidos, algo lo está invalidando.
		slog.Info("ia: respuesta", "vuelta", vuelta, "stop", resp.StopReason, "entrada", resp.Usage.InputTokens,
			"cache_leido", resp.Usage.CacheReadInputTokens, "cache_escrito", resp.Usage.CacheCreationInputTokens,
			"salida", resp.Usage.OutputTokens)
		if resp.StopReason == anthropic.StopReasonRefusal {
			r.Texto, r.Rechazo = FraseRechazo, true
			return r, nil
		}
		var textos []string
		var resultados []anthropic.ContentBlockParamUnion
		for _, b := range resp.Content {
			switch v := b.AsAny().(type) {
			case anthropic.TextBlock:
				if t := strings.TrimSpace(v.Text); t != "" {
					textos = append(textos, t)
				}
			case anthropic.ToolUseBlock:
				r.Herramientas = append(r.Herramientas, v.Name)
				entrada := v.Input
				if len(entrada) == 0 {
					entrada = json.RawMessage("{}")
				}
				res, esError := correr(ctx, v.Name, entrada)
				resultados = append(resultados, anthropic.NewToolResultBlock(v.ID, res, esError))
			}
		}
		if resp.StopReason == anthropic.StopReasonMaxTokens {
			// Se cortó por largo: se mide y, si no quedó texto, se dice en simple.
			slog.Warn("ia: cortada", "vuelta", vuelta, "salida", resp.Usage.OutputTokens)
			r.Cortada = true
		}
		if resp.StopReason != anthropic.StopReasonToolUse || len(resultados) == 0 {
			r.Texto = strings.Join(textos, "\n\n")
			if r.Texto == "" {
				r.Texto = FraseSinVuelt
			}
			return r, nil
		}
		// El turno del modelo va tal cual (con su razonamiento) para no romper el caché.
		params.Messages = append(params.Messages, resp.ToParam(), anthropic.NewUserMessage(resultados...))
	}
	r.Texto = FraseSinVuelt
	return r, nil
}
