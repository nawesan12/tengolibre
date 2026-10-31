package cobros_test

import (
	"testing"
	"time"

	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/cobros/cobrosprueba"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/turnos"
)

// Volver a reservar desde Mi turno atendido (propuesta 04): vive con los tests de cobros porque la
// reserva cobra la seña igual que la pública.
func TestVolverAReservar(t *testing.T) {
	e := cobrosprueba.Armar(t)
	e.SQL("UPDATE ajustes SET dias_vista = 60")
	tu := e.Reservar(201, "09:00", "1111", nil)
	ruta := "/v1/t/" + tu.Codigo + "/repetir"
	e.Esperar(409, P{Metodo: "GET", Ruta: ruta}, nil)
	e.Panel(200, "PATCH", "/v1/panel/turnos/"+tu.ID, map[string]any{"estado": "atendido"}, nil)

	// Sin historia: a las 4 semanas, el lunes 30 a la misma hora, y los lunes de alrededor.
	var rp turnos.Repetir
	e.Esperar(200, P{Metodo: "GET", Ruta: ruta}, &rp)
	if rp.Semanas != 4 || rp.Habitual || rp.Sugerencia == nil || rp.Sugerencia.Fecha != "2026-11-30" || rp.Sugerencia.Hora != "09:00" ||
		!rp.Sugerencia.MismoHorario || rp.Pago != nil || rp.Profesional.Nombre != "Ana" || rp.Servicio.Precio != 8000 {
		t.Fatalf("sugerencia: %+v %+v", rp, rp.Sugerencia)
	}
	if len(rp.Alternativas) != 3 || rp.Alternativas[0].Fecha != "2026-11-16" || rp.Alternativas[1].Fecha != "2026-11-23" ||
		rp.Alternativas[2].Fecha != "2026-12-07" {
		t.Fatalf("alternativas: %+v", rp.Alternativas)
	}

	// Alguien toma el 30 a las 9: lo más parecido pasa a ser las 9:30. Con seña, se muestra cuánto.
	e.Esperar(201, P{Metodo: "POST", Ruta: "/v1/p/el-corte/turnos", Cuerpo: map[string]any{"servicioId": e.Servicio,
		"profesionalId": e.Profesional, "inicio": cobrosprueba.En("2026-11-30 09:00").Format(time.RFC3339),
		"cliente": map[string]string{"nombre": "Otra", "telefono": "3333"}}}, nil)
	e.ConectarMP()
	e.Panel(200, "PUT", "/v1/panel/cobros", map[string]any{"modo": "porcentaje", "senaPorcentaje": 50}, nil)
	e.Esperar(200, P{Metodo: "GET", Ruta: ruta}, &rp)
	if rp.Sugerencia == nil || rp.Sugerencia.Hora != "09:30" || rp.Sugerencia.MismoHorario || rp.Pago == nil || rp.Pago.Monto != 4000 {
		t.Fatalf("con el 30 a las 9 tomado: %+v %+v", rp.Sugerencia, rp.Pago)
	}

	// Reservar: mismo cliente, seña pendiente con el link de Mercado Pago.
	var nuevo turnos.Turno
	e.Esperar(201, P{Metodo: "POST", Ruta: ruta, Cuerpo: map[string]any{"inicio": rp.Sugerencia.Inicio}}, &nuevo)
	if nuevo.Estado != "pendiente_pago" || nuevo.Pago == nil || nuevo.Pago.Monto != 4000 || nuevo.Pago.URL == "" || nuevo.Cliente.ID != "" {
		t.Fatalf("reserva repetida: %+v %+v", nuevo, nuevo.Pago)
	}
	if x := e.Turno(nuevo.ID); x.Cliente.ID != e.Turno(tu.ID).Cliente.ID || x.Origen != "web" || !x.Inicio.Equal(cobrosprueba.En("2026-11-30 09:30")) {
		t.Fatalf("turno nuevo en el panel: %+v", x)
	}
	if ev := e.Avisos[nuevo.ID]; len(ev) != 1 || ev[0] != "reservado" {
		t.Fatalf("avisos: %v", ev)
	}
	// El mismo horario otra vez, no.
	e.Esperar(409, P{Metodo: "POST", Ruta: ruta, Cuerpo: map[string]any{"inicio": rp.Sugerencia.Inicio}}, nil)
	e.Esperar(400, P{Metodo: "POST", Ruta: ruta, Cuerpo: map[string]any{}}, nil)

	// Con historia (viene cada 2 semanas), la sugerencia sigue su frecuencia.
	h1 := e.Reservar(201, "11:00", "4444", nil)
	var h2 turnos.Turno
	e.Esperar(201, P{Metodo: "POST", Ruta: "/v1/p/el-corte/turnos", Cuerpo: map[string]any{"servicioId": e.Servicio,
		"profesionalId": e.Profesional, "inicio": cobrosprueba.En("2026-11-16 11:00").Format(time.RFC3339),
		"cliente": map[string]string{"nombre": "Cliente 4444", "telefono": "4444"}}}, &h2)
	for _, id := range []string{h1.ID, h2.ID} {
		e.SQL("UPDATE turnos SET estado = 'atendido' WHERE id = $1", id)
	}
	e.Esperar(200, P{Metodo: "GET", Ruta: "/v1/t/" + h2.Codigo + "/repetir"}, &rp)
	if rp.Semanas != 2 || !rp.Habitual || rp.Sugerencia == nil || rp.Sugerencia.Fecha != "2026-11-30" || rp.Sugerencia.Hora != "11:00" {
		t.Fatalf("con historia: %+v %+v", rp, rp.Sugerencia)
	}
	e.Esperar(404, P{Metodo: "GET", Ruta: "/v1/t/noexisteestecodigo123/repetir"}, nil)
}
