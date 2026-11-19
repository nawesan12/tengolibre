package espera_test

import (
	"testing"
	"time"

	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/espera"
)

func TestDespuesDeDormir(t *testing.T) {
	z, _ := time.LoadLocation("America/Argentina/Buenos_Aires")
	en := func(s string) time.Time { x, _ := time.ParseInLocation("2006-01-02 15:04", s, z); return x }
	casos := []struct{ ahora, inicio, quiero string }{
		{"2026-11-02 15:00", "2026-11-02 18:00", "2026-11-02 15:00"}, // de día: ya
		{"2026-11-02 23:30", "2026-11-03 11:00", "2026-11-03 08:00"}, // de noche: a las 8
		{"2026-11-03 03:00", "2026-11-03 10:00", "2026-11-03 08:00"}, // de madrugada: a las 8
		{"2026-11-03 03:00", "2026-11-03 08:30", "2026-11-03 03:00"}, // el turno es temprano: no se espera
	}
	for _, c := range casos {
		if got := espera.DespuesDeDormir(en(c.ahora), en(c.inicio), z); !got.Equal(en(c.quiero)) {
			t.Errorf("%s → %s: %s, quería %s", c.ahora, c.inicio, got.In(z), c.quiero)
		}
	}
}
