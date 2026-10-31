package espera_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/cobros/cobrosprueba"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/espera"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/turnos"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/prueba"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/servidor"
)

// Tomar el lugar cobra la seña igual que la reserva pública: el turno queda esperando el pago y
// viene el link de Mercado Pago.
func TestTomarElLugarConSena(t *testing.T) {
	e := cobrosprueba.ArmarCon(t, func(a *prueba.API, tur *turnos.Servicio) []servidor.Dominio {
		return []servidor.Dominio{&espera.Servicio{Pool: a.Bases.App, Sesion: a.Sesion, Ahora: a.Reloj(), Turnos: tur}}
	})
	e.ConectarMP()
	e.Panel(200, "PUT", "/v1/panel/cobros", map[string]any{"modo": "porcentaje", "senaPorcentaje": 50}, nil)
	e.Esperar(201, P{Metodo: "POST", Ruta: "/v1/p/el-corte/espera", Cuerpo: map[string]any{"servicioId": e.Servicio,
		"nombre": "Martina Gómez", "email": "martina@ejemplo.com", "telefono": "1122334455", "desde": "2026-11-02"}}, nil)
	var o espera.Oferta
	e.Panel(201, "POST", "/v1/panel/espera/ofertas", map[string]any{"servicioId": e.Servicio, "profesionalId": e.Profesional,
		"inicio": cobrosprueba.En("2026-11-02 09:00").Format(time.RFC3339)}, &o)

	ctx := context.Background()
	c, err := pgx.Connect(ctx, e.Bases.AdminURL)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close(ctx)
	var token string
	if err := c.QueryRow(ctx, "SELECT token FROM oferta_intentos WHERE oferta_id = $1 AND estado = 'ofrecida'", o.ID).Scan(&token); err != nil {
		t.Fatal(err)
	}
	var tu turnos.Turno
	e.Esperar(201, P{Metodo: "POST", Ruta: "/v1/espera/" + token + "/tomar"}, &tu)
	if tu.Estado != "pendiente_pago" || tu.Pago == nil || tu.Pago.Monto != 4000 || !strings.HasPrefix(tu.Pago.URL, "https://mp.falso/checkout/") ||
		tu.Cliente.Nombre != "Martina Gómez" || tu.Cliente.ID != "" {
		t.Fatalf("turno: %+v %+v", tu, tu.Pago)
	}
	if ev := e.Avisos[tu.ID]; len(ev) != 1 || ev[0] != "reservado_espera" {
		t.Fatalf("avisos: %v", ev)
	}
}
