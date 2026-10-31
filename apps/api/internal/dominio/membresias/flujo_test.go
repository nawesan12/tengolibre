package membresias_test

import (
	"testing"
	"time"

	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/cobros/cobrosprueba"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/membresias"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/negocios"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/turnos"
)

type P = cobrosprueba.P

func reservarPanel(e *cobrosprueba.Entorno, hora, telefono string) turnos.Turno {
	e.T.Helper()
	var t turnos.Turno
	e.Panel(201, "POST", "/v1/panel/turnos", map[string]any{"servicioId": e.Servicio, "profesionalId": e.Profesional,
		"inicio": cobrosprueba.En("2026-11-02 " + hora).Format(time.RFC3339), "cliente": map[string]string{"nombre": "Socia", "telefono": telefono}}, &t)
	return t
}

func socio(e *cobrosprueba.Entorno, id string) membresias.SocioConCreditos {
	e.T.Helper()
	var s membresias.SocioConCreditos
	e.Panel(200, "GET", "/v1/panel/membresias/socios/"+id, nil, &s)
	return s
}

func TestCreditoDeMembresiaConsumido(t *testing.T) {
	e := cobrosprueba.Armar(t)
	var p membresias.Plan
	e.Panel(201, "POST", "/v1/panel/membresias/planes", map[string]any{"nombre": "Dos cortes", "precio": 15000, "visitas": 2,
		"duracionDias": 30, "beneficios": []string{"Prioridad para reservar"}, "servicios": []string{e.Servicio}, "topeSocios": 1}, &p)
	if !p.Activo || !p.Cubre(e.Servicio) || len(p.Beneficios) != 1 {
		t.Fatalf("plan: %+v", p)
	}
	e.Panel(400, "POST", "/v1/panel/membresias/planes", map[string]any{"nombre": "Malo", "precio": 100, "servicios": []string{"00000000-0000-0000-0000-000000000000"}}, nil)
	var pub struct {
		Planes []membresias.Plan `json:"planes"`
	}
	e.Esperar(200, P{Metodo: "GET", Ruta: "/v1/p/el-corte/membresias"}, &pub)
	if len(pub.Planes) != 1 {
		t.Fatalf("planes públicos: %+v", pub)
	}

	var s membresias.Socio
	e.Panel(201, "POST", "/v1/panel/membresias/socios", map[string]any{"planId": p.ID, "nombre": "Socia", "telefono": "11 1111"}, &s)
	if s.Quedan != 2 || s.Estado != "activa" || s.Telefono != "111111" {
		t.Fatalf("alta: %+v", s)
	}
	e.Panel(409, "POST", "/v1/panel/membresias/socios", map[string]any{"planId": p.ID, "nombre": "Socia", "telefono": "111111"}, nil)
	e.Panel(409, "POST", "/v1/panel/membresias/socios", map[string]any{"planId": p.ID, "nombre": "Otra", "telefono": "222222"}, nil) // cupo

	// Reservar en el panel gasta una visita; cancelar la devuelve (una sola vez).
	t1 := reservarPanel(e, "09:00", "111111")
	x := socio(e, s.ID)
	if x.Quedan != 1 || x.Creditos[0].Cantidad != -1 || x.Creditos[0].TurnoID == nil || *x.Creditos[0].TurnoID != t1.ID {
		t.Fatalf("después de reservar: %+v", x)
	}
	e.Panel(200, "PATCH", "/v1/panel/turnos/"+t1.ID, map[string]string{"estado": "cancelado"}, nil)
	e.Panel(200, "PATCH", "/v1/panel/turnos/"+t1.ID, map[string]string{"estado": "cancelado"}, nil)
	if x = socio(e, s.ID); x.Quedan != 2 {
		t.Fatalf("cancelar devuelve la visita una vez: %+v", x.Creditos)
	}

	// Una reserva desde la página no gasta visitas (todavía no se sabe quién es).
	e.Reservar(201, "09:00", "111111", nil)
	reservarPanel(e, "10:00", "111111")
	reservarPanel(e, "10:30", "111111")
	// Sin visitas, la reserva sale igual y el saldo no baja de cero.
	reservarPanel(e, "11:00", "111111")
	if x = socio(e, s.ID); x.Quedan != 0 {
		t.Fatalf("saldo: %d", x.Quedan)
	}

	// Ajustes a mano, sin quedar negativo.
	e.Panel(200, "PATCH", "/v1/panel/membresias/socios/"+s.ID, map[string]any{"ajustar": 3, "motivo": "Regalo"}, &x)
	if x.Quedan != 3 || x.Creditos[0].Motivo != "Regalo" || x.Creditos[0].ClerkID == nil {
		t.Fatalf("ajuste: %+v", x)
	}
	e.Panel(400, "PATCH", "/v1/panel/membresias/socios/"+s.ID, map[string]any{"ajustar": -10}, nil)

	// Vencido el período, no cubre; renovar arranca otro desde hoy con las visitas del plan.
	// (La prueba gratis se extiende, si no el panel queda de sólo lectura.)
	e.SQL("UPDATE negocios SET prueba_hasta = $1", e.Ahora.AddDate(1, 0, 0))
	e.Ahora = e.Ahora.AddDate(0, 0, 31)
	if x = socio(e, s.ID); x.Estado != "vencida" {
		t.Fatalf("estado: %s", x.Estado)
	}
	e.Panel(200, "PATCH", "/v1/panel/membresias/socios/"+s.ID, map[string]any{"renovar": true}, &x)
	if x.Estado != "activa" || x.Quedan != 5 || !x.Hasta.Equal(e.Ahora.AddDate(0, 0, 30)) {
		t.Fatalf("renovada: %+v", x.Socio)
	}
	var l struct {
		Socios []membresias.Socio `json:"socios"`
	}
	e.Panel(200, "GET", "/v1/panel/membresias/socios", nil, &l)
	if len(l.Socios) != 1 || l.Socios[0].Quedan != 5 {
		t.Fatalf("socios: %+v", l.Socios)
	}
	// Un plan con socios no se borra: se desactiva.
	e.Panel(204, "DELETE", "/v1/panel/membresias/planes/"+p.ID, nil, nil)
	e.Esperar(200, P{Metodo: "GET", Ruta: "/v1/p/el-corte/membresias"}, &pub)
	if len(pub.Planes) != 0 {
		t.Fatal("el plan desactivado sigue en la página")
	}

	// Otro negocio no ve a los socios de este.
	const otra = "user_otra000001"
	var n negocios.Negocio
	e.Esperar(201, P{Metodo: "POST", Ruta: "/v1/negocios", Quien: otra, Cuerpo: map[string]string{"slug": "otro", "nombre": "Otro"}}, &n)
	e.SQL("UPDATE negocios SET prueba_hasta = $1", e.Ahora.AddDate(1, 0, 0))
	e.Esperar(200, P{Metodo: "GET", Ruta: "/v1/panel/membresias/socios?estado=todas", Quien: otra, Negocio: n.ID}, &l)
	if len(l.Socios) != 0 {
		t.Fatalf("el otro negocio ve %d socios", len(l.Socios))
	}
	e.Esperar(404, P{Metodo: "GET", Ruta: "/v1/panel/membresias/socios/" + s.ID, Quien: otra, Negocio: n.ID}, nil)
	e.Esperar(404, P{Metodo: "PATCH", Ruta: "/v1/panel/membresias/socios/" + s.ID, Quien: otra, Negocio: n.ID,
		Cuerpo: map[string]any{"ajustar": 5}}, nil)
}
