package avisos

import (
	"strings"
	"testing"
	"time"
	"unicode/utf8"
)

func TestFechaEnCastellano(t *testing.T) {
	z, _ := time.LoadLocation("America/Argentina/Buenos_Aires")
	f := time.Date(2026, 9, 12, 10, 30, 0, 0, z)
	if Fecha(f) != "sábado 12 de septiembre" || Hora(f) != "10:30" {
		t.Fatalf("%s %s", Fecha(f), Hora(f))
	}
}

// El .ics escapa lo que hay que escapar y pliega las líneas largas sin partir una tilde.
func TestICS(t *testing.T) {
	ahora := time.Date(2026, 11, 1, 13, 0, 0, 0, time.UTC)
	ics := string(ICS("t1", "Corte, barba; color — El Corte", strings.Repeat("ñ", 60), "Av. Colón 1234", ahora, ahora.Add(30*time.Minute), ahora))
	if !strings.Contains(ics, `SUMMARY:Corte\, barba\; color — El Corte`) || !strings.Contains(ics, "UID:t1@tengolibre.com.ar") {
		t.Fatal(ics)
	}
	for _, l := range strings.Split(ics, "\r\n") {
		if len(l) > 75 {
			t.Fatalf("línea de %d bytes: %q", len(l), l)
		}
		if !utf8.ValidString(l) {
			t.Fatalf("partió un carácter: %q", l)
		}
	}
}

func TestPesos(t *testing.T) {
	for n, quiero := range map[int64]string{0: "$ 0", 900: "$ 900", 26000: "$ 26.000", 1234567: "$ 1.234.567"} {
		if Pesos(n) != quiero {
			t.Errorf("Pesos(%d) = %q, quería %q", n, Pesos(n), quiero)
		}
	}
}

func TestEsperaCreciente(t *testing.T) {
	if EsperaReintento(1) != time.Minute || EsperaReintento(3) != 4*time.Minute || EsperaReintento(20) != 6*time.Hour {
		t.Fatal("espera")
	}
}

// El HTML escapa lo que viene de los datos (el nombre de un cliente no mete etiquetas).
func TestCartaEscapa(t *testing.T) {
	c := Carta{Encabezado: "El Corte", Titulo: "Hola", Saludo: "Hola <b>Lucía</b>,", Link: "https://x.y/z", Boton: "Ir"}
	if strings.Contains(c.HTML(), "<b>Lucía</b>") || !strings.Contains(c.Texto(), "Ir: https://x.y/z") {
		t.Fatal(c.HTML())
	}
}
