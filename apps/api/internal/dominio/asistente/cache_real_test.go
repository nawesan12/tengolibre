package asistente

import (
	"context"
	"os"
	"testing"

	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/ia"
)

// TestCacheReal manda dos preguntas distintas del mismo negocio a Claude de verdad y comprueba que la
// segunda lea el prefijo del caché. Se corre a mano antes de prender la IA:
// TENGOLIBRE_IA_PRUEBA_REAL=1 ANTHROPIC_API_KEY=… go test ./internal/dominio/asistente -run CacheReal -v
func TestCacheReal(t *testing.T) {
	if os.Getenv("TENGOLIBRE_IA_PRUEBA_REAL") == "" {
		t.Skip("sólo a mano, con la clave")
	}
	c := ia.NuevoCliente(os.Getenv("ANTHROPIC_API_KEY"))
	datos := `{"negocio":"Peluquería de prueba","rol":"duenio","hoy":"2026-11-18 16:00","servicios":["Corte","Color","Brushing"],"profesionales":["Ana","Beto"]}`
	esquema := map[string]any{"type": "object", "properties": map[string]any{"fecha": map[string]any{"type": "string"}}}
	hs := []ia.Herramienta{{Nombre: "ver_agenda", Descripcion: "Los turnos de un día.", Esquema: esquema},
		{Nombre: "buscar_horarios_libres", Descripcion: "Los horarios libres de un día.", Esquema: esquema}}
	for i, q := range []string{"¿Cuál es el servicio más caro?", "¿Qué profesionales hay?"} {
		r, err := c.Crear(context.Background(), ia.Armar(ia.Pedido{Sistema: Sistema, Datos: datos, Pregunta: q, Herramientas: hs, MaxTokens: 300}))
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("pedido %d: entrada %d, caché leído %d, caché escrito %d, salida %d", i+1, r.Usage.InputTokens,
			r.Usage.CacheReadInputTokens, r.Usage.CacheCreationInputTokens, r.Usage.OutputTokens)
		if i == 1 && r.Usage.CacheReadInputTokens == 0 {
			t.Fatal("la segunda pregunta no leyó nada del caché: hay un invalidador en el prefijo")
		}
	}
}
