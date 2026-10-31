package reportes_test

import (
	"strings"
	"testing"
	"time"

	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/catalogo"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/clientes"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/disponibilidad"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/negocios"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/reportes"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/turnos"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/prueba"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/servidor"
)

const duenia = "user_duenia0001"

type P = prueba.Pedido

func en(s string) time.Time {
	t, err := time.ParseInLocation("2006-01-02 15:04", s, disponibilidad.ZonaPorDefecto)
	if err != nil {
		panic(err)
	}
	return t
}

func TestReportesYExportaciones(t *testing.T) {
	a := prueba.Nueva(t, en("2026-11-01 10:00"), func(a *prueba.API) []servidor.Dominio {
		app := a.Bases.App
		return []servidor.Dominio{
			&negocios.Servicio{Pool: app, Sesion: a.Sesion},
			&catalogo.Servicio{Pool: app, Sesion: a.Sesion},
			&turnos.Servicio{Pool: app, Sesion: a.Sesion, Ahora: a.Reloj()},
			&clientes.Servicio{Pool: app, Sesion: a.Sesion, Ahora: a.Reloj()},
			&reportes.Servicio{Pool: app, Sesion: a.Sesion, Ahora: a.Reloj()},
		}
	})
	var n negocios.Negocio
	a.Esperar(201, P{Metodo: "POST", Ruta: "/v1/negocios", Quien: duenia, Cuerpo: map[string]string{"slug": "el-corte", "nombre": "El Corte"}}, &n)
	panel := func(estado int, metodo, ruta string, cuerpo, v any) prueba.Respuesta {
		t.Helper()
		return a.Esperar(estado, P{Metodo: metodo, Ruta: ruta, Quien: duenia, Negocio: n.ID, Cuerpo: cuerpo}, v)
	}
	var s catalogo.Prestacion
	panel(201, "POST", "/v1/panel/servicios", map[string]any{"nombre": "Corte", "duracion": 30, "precio": 8000}, &s)
	horario := []disponibilidad.Franja{{Dia: 1, Desde: "09:00", Hasta: "11:00"}}
	var ana, beto catalogo.Profesional
	panel(201, "POST", "/v1/panel/profesionales", map[string]any{"nombre": "Ana", "servicios": []string{s.ID}, "horario": horario, "orden": 1}, &ana)
	panel(201, "POST", "/v1/panel/profesionales", map[string]any{"nombre": "Beto", "servicios": []string{s.ID}, "horario": horario, "orden": 2}, &beto)
	reservar := func(profesional, inicio, nombre, telefono string) turnos.Turno {
		t.Helper()
		var x turnos.Turno
		a.Esperar(201, P{Metodo: "POST", Ruta: "/v1/p/el-corte/turnos", Cuerpo: map[string]any{
			"servicioId": s.ID, "profesionalId": profesional, "inicio": en(inicio).Format(time.RFC3339),
			"cliente": map[string]string{"nombre": nombre, "telefono": telefono}}}, &x)
		return x
	}
	estado := func(id, e string) { panel(200, "PATCH", "/v1/panel/turnos/"+id, map[string]any{"estado": e}, nil) }

	// Lunes 2: Ana atiende a Juana a las 9 (8000), Beto tiene una ausencia a las 9, Ana un cancelado a
	// las 9:30 y un confirmado a las 10. El lunes 9 Juana vuelve.
	t1 := reservar(ana.ID, "2026-11-02 09:00", "Juana", "1111")
	t2 := reservar(beto.ID, "2026-11-02 09:00", "Pedro", "2222")
	t3 := reservar(ana.ID, "2026-11-02 09:30", "Rita", "3333")
	reservar(ana.ID, "2026-11-02 10:00", "Sole", "4444")
	reservar(ana.ID, "2026-11-09 09:00", "Juana", "1111")
	estado(t1.ID, "atendido")
	estado(t2.ID, "ausente")
	estado(t3.ID, "cancelado")
	// Un bloqueo de una hora a Beto le saca disponibilidad.
	panel(201, "POST", "/v1/panel/profesionales/"+beto.ID+"/bloqueos", map[string]any{"fecha": "2026-11-02", "desde": "10:00", "hasta": "11:00"}, nil)

	var rep reportes.Reporte
	panel(200, "GET", "/v1/panel/reportes?desde=2026-11-02&hasta=2026-11-02", nil, &rep)
	r := rep.Resumen
	if r.Turnos != 3 || r.Atendidos != 1 || r.Ingresos != 8000 || r.TicketPromedio != 8000 || r.Clientes != 3 {
		t.Fatalf("resumen: %+v", r)
	}
	if rep.PorEstado["atendido"] != 1 || rep.PorEstado["ausente"] != 1 || rep.PorEstado["cancelado"] != 1 || rep.PorEstado["pendiente"] != 1 {
		t.Fatalf("por estado: %+v", rep.PorEstado)
	}
	// Ausencias: 1 de 2 cerrados. Ocupación: Ana 60' de 120, Beto 30' de 60 → 90 de 180.
	if r.TasaAusencias == nil || *r.TasaAusencias != 50 || r.Ocupacion == nil || *r.Ocupacion != 50 {
		t.Fatalf("tasas: ausencias %v ocupación %v", r.TasaAusencias, r.Ocupacion)
	}
	if len(rep.PorProfesional) != 2 || rep.PorProfesional[0].Nombre != "Ana" || rep.PorProfesional[0].Ingresos != 8000 ||
		rep.PorProfesional[0].MinutosReservados != 60 || *rep.PorProfesional[1].MinutosDisponibles != 60 {
		t.Fatalf("por profesional: %+v", rep.PorProfesional)
	}
	if len(rep.PorServicio) != 1 || rep.PorServicio[0].Turnos != 3 || rep.PorServicio[0].Ausentes != 1 {
		t.Fatalf("por servicio: %+v", rep.PorServicio)
	}
	if rep.ClientesNuevos != 3 || rep.ClientesRecurrentes != 0 || len(rep.PorDia) != 1 || rep.PorDia[0].Fecha != "2026-11-02" {
		t.Fatalf("clientes y días: %+v", rep)
	}

	// El lunes 9 Juana es recurrente; el período entero tiene 8 días.
	panel(200, "GET", "/v1/panel/reportes?desde=2026-11-09&hasta=2026-11-09", nil, &rep)
	if rep.ClientesNuevos != 0 || rep.ClientesRecurrentes != 1 || rep.Resumen.Ocupacion == nil || *rep.Resumen.Ocupacion != 12.5 {
		t.Fatalf("lunes 9: %+v", rep)
	}
	panel(200, "GET", "/v1/panel/reportes?desde=2026-11-02&hasta=2026-11-09", nil, &rep)
	if len(rep.PorDia) != 8 || rep.PorDia[3].Turnos != 0 || rep.Resumen.Turnos != 4 {
		t.Fatalf("semana: %+v", rep.PorDia)
	}
	// Al profesional sin turnos en el período también se lo lista, en cero.
	panel(200, "GET", "/v1/panel/reportes?desde=2026-11-09&hasta=2026-11-09", nil, &rep)
	if len(rep.PorProfesional) != 2 || rep.PorProfesional[1].Nombre != "Beto" || rep.PorProfesional[1].Turnos != 0 {
		t.Fatalf("Beto en cero: %+v", rep.PorProfesional)
	}
	panel(400, "GET", "/v1/panel/reportes?desde=2026-11-09&hasta=2026-11-02", nil, nil)
	panel(400, "GET", "/v1/panel/reportes?desde=2025-01-01&hasta=2026-11-02", nil, nil)
	panel(400, "GET", "/v1/panel/reportes?desde=ayer", nil, nil)
	// Por defecto, los últimos 30 días.
	panel(200, "GET", "/v1/panel/reportes", nil, &rep)
	if rep.Hasta != "2026-11-01" || rep.Desde != "2026-10-03" {
		t.Fatalf("rango por defecto: %s a %s", rep.Desde, rep.Hasta)
	}

	// Tablero del lunes 2 a las 8.
	a.Ahora = en("2026-11-02 08:00")
	var tb reportes.Tablero
	panel(200, "GET", "/v1/panel/estadisticas", nil, &tb)
	if tb.TurnosHoy != 3 || tb.TurnosAyer != 0 || tb.IngresosMes != 8000 || tb.OcupacionHoy == nil || *tb.OcupacionHoy != 50 ||
		len(tb.Ultimos7Dias) != 7 || len(tb.Ultimos6Meses) != 6 || tb.Ultimos6Meses[5].Mes != "2026-11" || len(tb.Proximos) != 2 {
		t.Fatalf("tablero: %+v", tb)
	}

	// CSV: con BOM, encabezado y una fila por turno; las fórmulas no pasan.
	panel(200, "PATCH", "/v1/panel/turnos/"+t1.ID, map[string]any{"nota": "=HYPERLINK(\"x\")"}, nil)
	csv := string(panel(200, "GET", "/v1/panel/reportes/turnos.csv?desde=2026-11-02&hasta=2026-11-09", nil, nil).Cuerpo)
	lineas := strings.Split(strings.TrimSpace(csv), "\n")
	if !strings.HasPrefix(csv, "\xEF\xBB\xBFFecha,Hora,Cliente") || len(lineas) != 6 ||
		!strings.Contains(csv, "2026-11-02,09:00,Juana,1111") || !strings.Contains(csv, `'=HYPERLINK`) {
		t.Fatalf("turnos.csv:\n%s", csv)
	}
	csv = string(panel(200, "GET", "/v1/panel/reportes/clientes.csv", nil, nil).Cuerpo)
	lineas = strings.Split(strings.TrimSpace(csv), "\n")
	if len(lineas) != 5 || !strings.HasPrefix(lineas[1], "Juana,1111,,") {
		t.Fatalf("clientes.csv:\n%s", csv)
	}
}
