package ia

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
)

func pedidoDePrueba(pregunta string, historial ...Turno) Pedido {
	return Pedido{
		Sistema:  "Sos el asistente.",
		Datos:    "hoy: 2026-11-18\nhora: 10:15",
		Pregunta: pregunta, Historial: historial,
		Herramientas: []Herramienta{
			{Nombre: "ver_agenda", Descripcion: "agenda", Esquema: map[string]any{"type": "object", "properties": map[string]any{"desde": map[string]any{"type": "string"}}}},
			{Nombre: "buscar_horarios_libres", Descripcion: "libres", Esquema: map[string]any{"type": "object",
				"properties": map[string]any{"fecha": map[string]any{"type": "string"}}, "required": []string{"fecha"}}},
		},
	}
}

// Dos preguntas distintas del mismo negocio en el mismo rato comparten byte a byte todo hasta el
// segundo breakpoint (herramientas, system y datos): eso es lo que lee el caché.
func TestPrefijoIdenticoEntreDosPreguntas(t *testing.T) {
	a := Armar(pedidoDePrueba("¿qué tengo hoy?"))
	b := Armar(pedidoDePrueba("¿hay lugar el viernes?", Turno{Rol: "usuario", Texto: "hola"}, Turno{Rol: "asistente", Texto: "¡Hola!"}))
	if !bytes.Equal(Prefijo(a), Prefijo(b)) {
		t.Fatalf("el prefijo cambió:\n%s\n%s", Prefijo(a), Prefijo(b))
	}
	// Las herramientas van ordenadas por nombre, vengan como vengan.
	p := pedidoDePrueba("x")
	p.Herramientas[0], p.Herramientas[1] = p.Herramientas[1], p.Herramientas[0]
	if !bytes.Equal(Prefijo(Armar(p)), Prefijo(a)) {
		t.Fatal("el orden de las herramientas cambia el prefijo")
	}
	// Otros datos (otro negocio u otro cuarto de hora), otro prefijo.
	p = pedidoDePrueba("¿qué tengo hoy?")
	p.Datos = "hoy: 2026-11-18\nhora: 10:30"
	if bytes.Equal(Prefijo(Armar(p)), Prefijo(a)) {
		t.Fatal("datos distintos dieron el mismo prefijo")
	}
	// Breakpoints: system y datos explícitos, y el automático arriba.
	j, _ := json.Marshal(a)
	if n := strings.Count(string(j), `"cache_control"`); n != 3 {
		t.Fatalf("cache_control: %d, quería 3 (system, datos y automático): %s", n, j)
	}
	if a.Model != Modelo || a.OutputConfig.Effort != anthropic.OutputConfigEffortLow || a.ToolChoice.OfAuto == nil {
		t.Fatal("modelo, esfuerzo o tool_choice no son los fijos")
	}
}

func TestHistorialRecortadoYAlternado(t *testing.T) {
	var h []Turno
	for i := 0; i < 12; i++ {
		rol := "usuario"
		if i%2 == 1 {
			rol = "asistente"
		}
		h = append(h, Turno{Rol: rol, Texto: strings.Repeat("a", 900)})
	}
	// Empieza por el asistente: ese turno se cae.
	p := Armar(pedidoDePrueba("¿y mañana?", append([]Turno{{Rol: "asistente", Texto: "suelto"}}, h...)...))
	for i, m := range p.Messages {
		if (i%2 == 0) != (m.Role == anthropic.MessageParamRoleUser) {
			t.Fatalf("el mensaje %d no alterna: %s", i, m.Role)
		}
		for _, b := range m.Content {
			if b.OfText != nil && len([]rune(b.OfText.Text)) > MaxLargoDeTurno && !strings.Contains(b.OfText.Text, "datos_del_negocio") {
				t.Fatalf("turno sin recortar: %d", len(b.OfText.Text))
			}
		}
	}
	// 8 turnos de historial + la pregunta (que se junta con el último del usuario si hiciera falta).
	if len(p.Messages) != 9 {
		t.Fatalf("mensajes: %d, quería 9", len(p.Messages))
	}
}

// El bucle con el cliente falso: pide una herramienta, se le devuelve y contesta.
func TestBucleConHerramientas(t *testing.T) {
	f := &Falso{Respuestas: []string{
		RespuestaHerramienta("tu_1", "ver_agenda", map[string]any{"desde": "2026-11-18"}),
		RespuestaTexto("Hoy tenés 3 turnos."),
	}}
	var corridas []string
	r, err := Conversar(context.Background(), f, pedidoDePrueba("¿qué tengo hoy?"), func(_ context.Context, n string, e json.RawMessage) (string, bool) {
		corridas = append(corridas, n+string(e))
		return `{"turnos":[]}`, false
	})
	if err != nil {
		t.Fatal(err)
	}
	if r.Texto != "Hoy tenés 3 turnos." || len(corridas) != 1 || corridas[0] != `ver_agenda{"desde":"2026-11-18"}` {
		t.Fatalf("resultado: %+v, corridas: %v", r, corridas)
	}
	if r.Uso.Pedidos != 2 || r.Uso.CacheLeido != 1800 {
		t.Fatalf("uso: %+v", r.Uso)
	}
	// La segunda vuelta lleva el turno del modelo y el tool_result, con el mismo prefijo.
	seg := f.Recibidos[1]
	if len(seg.Messages) != 3 || !bytes.Equal(Prefijo(seg), Prefijo(f.Recibidos[0])) {
		t.Fatalf("segunda vuelta: %d mensajes", len(seg.Messages))
	}
}

// Tope de vueltas: si el modelo sigue pidiendo herramientas, la última va sin ellas.
func TestBucleTopeDeVueltas(t *testing.T) {
	var rs []string
	for i := 0; i < MaxVueltas-1; i++ {
		rs = append(rs, RespuestaHerramienta("tu", "ver_agenda", map[string]any{}))
	}
	rs = append(rs, RespuestaTexto("Listo."))
	f := &Falso{Respuestas: rs}
	r, err := Conversar(context.Background(), f, pedidoDePrueba("x"), func(context.Context, string, json.RawMessage) (string, bool) { return "{}", false })
	if err != nil {
		t.Fatal(err)
	}
	if f.Llamadas() != MaxVueltas || r.Texto != "Listo." {
		t.Fatalf("llamadas %d, texto %q", f.Llamadas(), r.Texto)
	}
	if f.Recibidos[MaxVueltas-1].ToolChoice.OfNone == nil {
		t.Fatal("la última vuelta tenía que ir con tool_choice none")
	}
}

func TestBucleRechazo(t *testing.T) {
	f := &Falso{Respuestas: []string{RespuestaRechazo()}}
	r, err := Conversar(context.Background(), f, pedidoDePrueba("x"), nil)
	if err != nil || !r.Rechazo || r.Texto != FraseRechazo {
		t.Fatalf("%+v %v", r, err)
	}
}

func TestCostoMicro(t *testing.T) {
	// 1M de entrada = USD 0,10 = 100.000 micro; 1M de salida = 500.000.
	u := Uso{Entrada: 1_000_000, Salida: 1_000_000, CacheLeido: 1_000_000, CacheEscrito: 1_000_000}
	if c := u.CostoMicro(); c != 100_000+500_000+10_000+125_000 {
		t.Fatalf("costo: %d", c)
	}
}

func TestCacheDeRespuestas(t *testing.T) {
	ahora := time.Date(2026, 11, 18, 10, 0, 0, 0, time.UTC)
	c := &Cache{Ahora: func() time.Time { return ahora }}
	c.Guardar(Clave("n1", "u1", "¿Qué tengo HOY?"), "3 turnos")
	if v, ok := c.Leer(Clave("n1", "u1", "que tengo hoy")); !ok || v != "3 turnos" {
		t.Fatal("la misma pregunta normalizada no salió del caché")
	}
	if _, ok := c.Leer(Clave("n1", "u2", "que tengo hoy")); ok {
		t.Fatal("otra persona leyó el caché de otra")
	}
	ahora = ahora.Add(TTLRespuestas)
	if _, ok := c.Leer(Clave("n1", "u1", "que tengo hoy")); ok {
		t.Fatal("pasados 10 minutos sigue en el caché")
	}
	c.Guardar(Clave("n1", "u1", "a"), "x")
	c.Olvidar("n1")
	if _, ok := c.Leer(Clave("n1", "u1", "a")); ok {
		t.Fatal("Olvidar no borró")
	}
}
