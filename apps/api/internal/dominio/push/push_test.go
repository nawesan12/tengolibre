package push

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestNormalizar(t *testing.T) {
	hex := strings.Repeat("ab", 32)
	d, err := Normalizar(Dispositivo{Plataforma: "ios", Token: strings.ToUpper(hex), Modelo: " iPhone\n16 "})
	if err != nil || d.Token != hex || d.Entorno != "produccion" || d.Modelo != "iPhone16" {
		t.Fatalf("ios: %+v %v", d, err)
	}
	d, err = Normalizar(Dispositivo{Plataforma: "android", Token: "fcm:APA91b-token_de_registro", Entorno: "sandbox"})
	if err != nil || d.Entorno != "produccion" {
		t.Fatalf("android: %+v %v", d, err)
	}
	for _, mal := range []Dispositivo{
		{Plataforma: "ios", Token: "no-es-hex"},
		{Plataforma: "ios", Token: hex, Entorno: "otro"},
		{Plataforma: "android", Token: "corto"},
		{Plataforma: "android", Token: "con espacios y más de veinte letras"},
		{Plataforma: "windows", Token: hex},
	} {
		if _, err := Normalizar(mal); err == nil {
			t.Fatalf("%+v tenía que ser inválido", mal)
		}
	}
}

func TestPayloadAPNs(t *testing.T) {
	var p map[string]any
	_ = json.Unmarshal(PayloadAPNs(Aviso{Titulo: "Nueva reserva", Cuerpo: "Ana, corte", Ruta: "turnos/1", Negocio: "n1"}), &p)
	aps := p["aps"].(map[string]any)
	if aps["alert"].(map[string]any)["title"] != "Nueva reserva" || aps["thread-id"] != "negocio-n1" ||
		p["ruta"] != "turnos/1" || p["negocio"] != "n1" {
		t.Fatalf("payload: %v", p)
	}
}

func TestPayloadAPNsConCategoria(t *testing.T) {
	var p struct {
		Aps struct {
			Category string `json:"category"`
		} `json:"aps"`
		Tipo string `json:"tipo"`
	}
	_ = json.Unmarshal(PayloadAPNs(Aviso{Tipo: "reserva", Titulo: "Nueva reserva", Ruta: "turnos/1", Negocio: "n1"}), &p)
	if p.Aps.Category != "TL_RESERVA" || p.Tipo != "reserva" {
		t.Fatalf("la reserva lleva su categoría y su tipo: %+v", p)
	}
	var q struct {
		Aps map[string]any `json:"aps"`
	}
	_ = json.Unmarshal(PayloadAPNs(Aviso{Tipo: "equipo", Titulo: "x", Negocio: "n1"}), &q)
	if _, hay := q.Aps["category"]; hay {
		t.Fatal("un tipo sin acciones no lleva categoría")
	}
}
