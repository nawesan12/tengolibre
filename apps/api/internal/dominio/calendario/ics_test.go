package calendario

import (
	"strings"
	"testing"
	"time"
)

func TestICS(t *testing.T) {
	ini := time.Date(2026, 11, 2, 9, 0, 0, 0, time.FixedZone("ART", -3*3600))
	out := string(ICS("El Corte, Ana", []Evento{{UID: "x@tengolibre.com.ar", Titulo: "Corte · María; la de siempre",
		Descripcion: "Tel: 11 1111\nNota: " + strings.Repeat("ñandú ", 20), Inicio: ini, Fin: ini.Add(30 * time.Minute), Cambio: ini, Tentativo: true}}))
	for _, quiero := range []string{"BEGIN:VCALENDAR\r\n", "X-WR-CALNAME:El Corte\\, Ana\r\n", "DTSTART:20261102T120000Z\r\n",
		"DTEND:20261102T123000Z\r\n", `SUMMARY:Corte · María\; la de siempre` + "\r\n", "STATUS:TENTATIVE\r\n", "END:VCALENDAR\r\n"} {
		if !strings.Contains(out, quiero) {
			t.Errorf("falta %q en:\n%s", quiero, out)
		}
	}
	for l := range strings.SplitSeq(out, "\r\n") {
		if len(l) > 75 {
			t.Errorf("línea de %d octetos: %q", len(l), l)
		}
	}
	// Desplegar devuelve el texto escapado entero.
	plano := strings.ReplaceAll(out, "\r\n ", "")
	if !strings.Contains(plano, `DESCRIPTION:Tel: 11 1111\nNota: ñandú ñandú`) || strings.Count(plano, "ñandú") != 20 {
		t.Errorf("descripción plegada mal:\n%s", out)
	}
}
