package push

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestPayloadActividad(t *testing.T) {
	ahora := time.Date(2026, 11, 5, 15, 0, 0, 0, time.UTC)
	fin := ahora.Add(30 * time.Minute)
	var p struct {
		Aps map[string]any `json:"aps"`
	}
	if err := json.Unmarshal(PayloadActividad(ahora, false, fin, "16:00 · Sofía"), &p); err != nil {
		t.Fatal(err)
	}
	cs := p.Aps["content-state"].(map[string]any)
	if p.Aps["event"] != "update" || cs["siguiente"] != "16:00 · Sofía" || cs["fin"].(float64) != fin.Sub(referencia2001).Seconds() {
		t.Fatalf("update mal armado: %+v", p.Aps)
	}
	_ = json.Unmarshal(PayloadActividad(ahora, true, fin, ""), &p)
	if p.Aps["event"] != "end" || p.Aps["dismissal-date"] == nil {
		t.Fatalf("end mal armado: %+v", p.Aps)
	}
}

func TestTextoCliente(t *testing.T) {
	ahora := time.Date(2026, 11, 5, 13, 0, 0, 0, time.UTC) // 10:00 en Buenos Aires
	manana := time.Date(2026, 11, 6, 19, 30, 0, 0, time.UTC)
	ti, cu, ok := TextoCliente("recordatorio", "Lumi", "Corte", manana, "America/Argentina/Buenos_Aires", ahora)
	if !ok || ti != "Te esperan mañana a las 16:30" || !strings.Contains(cu, "Corte en Lumi") {
		t.Fatalf("recordatorio: %q %q", ti, cu)
	}
	if _, _, ok := TextoCliente("resumen", "Lumi", "", manana, "", ahora); ok {
		t.Fatal("el resumen del día no es para la clienta")
	}
	var p map[string]any
	_ = json.Unmarshal(PayloadCliente("t", "c", "ABC"), &p)
	if p["ruta"] != "turno-cliente/ABC" {
		t.Fatalf("ruta: %v", p["ruta"])
	}
}
