package reportes_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/avisos"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/catalogo"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/clientes"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/disponibilidad"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/negocios"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/promos"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/reportes"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/turnos"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/correo"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/prueba"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/servidor"
)

// Un negocio con Ana, que atiende los martes de 10 a 16. Cada martes de septiembre y octubre tiene
// un turno largo de 10 a 14 (y uno corto a las 14 el 13/10): de 14 a 16 queda casi siempre vacío.
func TestHorasMuertasCuponYReporteMensual(t *testing.T) {
	a := prueba.Nueva(t, en("2026-08-24 10:00"), func(a *prueba.API) []servidor.Dominio {
		app := a.Bases.App
		pro := &promos.Servicio{Pool: app, Sesion: a.Sesion, Ahora: a.Reloj()}
		return []servidor.Dominio{
			&negocios.Servicio{Pool: app, Sesion: a.Sesion},
			&catalogo.Servicio{Pool: app, Sesion: a.Sesion},
			&turnos.Servicio{Pool: app, Sesion: a.Sesion, Ahora: a.Reloj(), Extras: []turnos.Extra{pro}},
			&clientes.Servicio{Pool: app, Sesion: a.Sesion, Ahora: a.Reloj()},
			&reportes.Servicio{Pool: app, Sesion: a.Sesion, Ahora: a.Reloj()},
			pro,
		}
	})
	var n negocios.Negocio
	a.Esperar(201, P{Metodo: "POST", Ruta: "/v1/negocios", Quien: duenia, Cuerpo: map[string]string{"slug": "el-corte", "nombre": "El Corte"}}, &n)
	panel := func(estado int, metodo, ruta string, cuerpo, v any) prueba.Respuesta {
		t.Helper()
		return a.Esperar(estado, P{Metodo: metodo, Ruta: ruta, Quien: duenia, Negocio: n.ID, Cuerpo: cuerpo}, v)
	}
	var largo, corto catalogo.Prestacion
	panel(201, "POST", "/v1/panel/servicios", map[string]any{"nombre": "Alisado", "duracion": 240, "precio": 10000}, &largo)
	panel(201, "POST", "/v1/panel/servicios", map[string]any{"nombre": "Retoque", "duracion": 60, "precio": 5000}, &corto)
	var ana catalogo.Profesional
	panel(201, "POST", "/v1/panel/profesionales", map[string]any{"nombre": "Ana", "servicios": []string{largo.ID, corto.ID},
		"horario": []disponibilidad.Franja{{Dia: 2, Desde: "10:00", Hasta: "16:00"}}, "orden": 1}, &ana)
	reservar := func(servicio, inicio, nombre, telefono, cupon string) prueba.Respuesta {
		t.Helper()
		return a.Hacer(P{Metodo: "POST", Ruta: "/v1/p/el-corte/turnos", Cuerpo: map[string]any{
			"servicioId": servicio, "profesionalId": ana.ID, "inicio": en(inicio).Format(time.RFC3339), "cupon": cupon,
			"cliente": map[string]string{"nombre": nombre, "telefono": telefono, "email": strings.ToLower(nombre) + "@ejemplo.com"}}})
	}
	atendido := func(servicio, inicio, nombre, telefono string) {
		t.Helper()
		dia := en(inicio)
		a.Ahora = dia.AddDate(0, 0, -1)
		r := reservar(servicio, inicio, nombre, telefono, "")
		if r.Estado != 201 {
			t.Fatalf("reservar %s: %d %s", inicio, r.Estado, r.Cuerpo)
		}
		var x turnos.Turno
		r.Leer(t, &x)
		panel(200, "PATCH", "/v1/panel/turnos/"+x.ID, map[string]any{"estado": "atendido"}, nil)
	}
	// Juana viene desde agosto (más de dos meses); Pedro, desde septiembre.
	atendido(largo.ID, "2026-08-25 10:00", "Juana", "1111")
	for i, d := range []string{"09-08", "09-15", "09-22", "09-29", "10-06", "10-13", "10-20", "10-27"} {
		quien, tel := "Juana", "1111"
		if i%2 == 1 {
			quien, tel = "Pedro", "2222"
		}
		atendido(largo.ID, "2026-"+d+" 10:00", quien, tel)
	}
	atendido(corto.ID, "2026-10-13 14:00", "Pedro", "2222")

	// Domingo 1/11: las últimas 8 semanas van del 6/9 al 31/10.
	a.Ahora = en("2026-11-01 10:00")
	var hm reportes.HorasMuertas
	panel(200, "GET", "/v1/panel/reportes/horas-muertas", nil, &hm)
	if hm.Desde != "2026-09-06" || hm.Hasta != "2026-10-31" || hm.Semanas != 8 || len(hm.Mapa.Dias) != 1 || hm.Mapa.Dias[0] != 2 ||
		len(hm.Mapa.Horas) != 6 || hm.Mapa.Horas[0] != 10 || len(hm.Mapa.Celdas) != 6 {
		t.Fatalf("mapa: %+v", hm.Mapa)
	}
	if c := hm.Mapa.Celdas[0]; c.Capacidad != 8*60 || c.Ocupado != 8*60 || c.Estado != "ocupado" {
		t.Fatalf("martes 10: %+v", c)
	}
	if c := hm.Mapa.Celdas[4]; c.Hora != 14 || c.Ocupado != 60 || *c.Vacio != 87.5 || c.Estado != "suele_quedar_vacio" {
		t.Fatalf("martes 14: %+v", c)
	}
	if hm.Peor == nil || hm.Peor.Texto != "Martes de 14 a 16: vacío el 94% de las veces" || hm.Peor.LibreMinutos != 900 {
		t.Fatalf("peor franja: %+v", hm.Peor)
	}
	s := hm.Sugerencia
	if s == nil || s.Porcentaje != 20 || s.HoraDesde != "14:00" || s.HoraHasta != "16:00" || s.Codigo != "MARTES14" || s.Destinatarios != 1 ||
		s.Texto != "Probá un 20% sólo en esa franja, para clientes que vienen hace más de 2 meses." {
		t.Fatalf("sugerencia: %+v", s)
	}

	// El cupón listo: la franja, los clientes de más de 2 meses y 30 días de vigencia.
	var c promos.Cupon
	panel(201, "POST", "/v1/panel/reportes/horas-muertas/cupon", nil, &c)
	if c.Codigo != "MARTES14" || c.Valor != 20 || len(c.Dias) != 1 || c.Dias[0] != 2 || *c.HoraDesde != "14:00" || *c.AntiguedadDias != 60 ||
		c.Hasta == nil || !c.Hasta.Equal(a.Ahora.AddDate(0, 0, 30)) ||
		c.Condiciones != "Vale los martes de 14 a 16 h, para clientes que vienen hace más de 2 meses." {
		t.Fatalf("cupón: %+v", c)
	}
	var otro promos.Cupon
	panel(201, "POST", "/v1/panel/reportes/horas-muertas/cupon", map[string]any{"porcentaje": 10}, &otro)
	if otro.Codigo != "MARTES14-2" || otro.Valor != 10 {
		t.Fatalf("segundo cupón: %+v", otro)
	}
	panel(400, "POST", "/v1/panel/reportes/horas-muertas/cupon", map[string]any{"horaDesde": "16:00"}, nil)
	var lista struct {
		Cupones []promos.Cupon `json:"cupones"`
	}
	panel(200, "GET", "/v1/panel/cupones", nil, &lista)
	if len(lista.Cupones) != 2 || lista.Cupones[0].Condiciones == "" {
		t.Fatalf("cupones: %+v", lista.Cupones)
	}

	// La página pública avisa la franja; la reserva la exige.
	var v promos.Validacion
	a.Esperar(200, P{Metodo: "POST", Ruta: "/v1/p/el-corte/cupones/validar", Cuerpo: map[string]any{"codigo": "martes14",
		"servicioId": corto.ID, "inicio": en("2026-11-03 12:00").Format(time.RFC3339)}}, &v)
	if v.Valido || v.Motivo != "Este cupón vale sólo los martes de 14 a 16 h" || v.Condiciones == "" {
		t.Fatalf("validar fuera de franja: %+v", v)
	}
	a.Esperar(200, P{Metodo: "POST", Ruta: "/v1/p/el-corte/cupones/validar", Cuerpo: map[string]any{"codigo": "martes14",
		"servicioId": corto.ID, "inicio": en("2026-11-03 14:00").Format(time.RFC3339)}}, &v)
	if !v.Valido || v.Descuento != 1000 {
		t.Fatalf("validar en franja: %+v", v)
	}
	a.Ahora = en("2026-11-02 10:00")
	if r := reservar(corto.ID, "2026-11-03 12:00", "Juana", "1111", "MARTES14"); r.Estado != 422 || !strings.Contains(string(r.Cuerpo), "martes de 14 a 16") {
		t.Fatalf("fuera de franja: %d %s", r.Estado, r.Cuerpo)
	}
	if r := reservar(corto.ID, "2026-11-03 15:00", "Pedro", "2222", "MARTES14"); r.Estado != 422 || !strings.Contains(string(r.Cuerpo), "hace más de 2 meses") {
		t.Fatalf("cliente nuevo: %d %s", r.Estado, r.Cuerpo)
	}
	r := reservar(corto.ID, "2026-11-03 14:00", "Juana", "1111", "MARTES14")
	var tj turnos.Turno
	if r.Estado != 201 {
		t.Fatalf("en franja: %d %s", r.Estado, r.Cuerpo)
	}
	r.Leer(t, &tj)
	if tj.Precio != 4000 || tj.Descuento != 1000 {
		t.Fatalf("turno con cupón: %+v", tj)
	}
	// Las restricciones se editan y se sacan.
	panel(200, "PATCH", "/v1/panel/cupones/"+otro.ID, map[string]any{"dias": []int{}, "horaDesde": "", "horaHasta": "", "antiguedadDias": 0}, &otro)
	if len(otro.Dias) != 0 || otro.HoraDesde != nil || otro.AntiguedadDias != nil || otro.Condiciones != "" {
		t.Fatalf("sin restricciones: %+v", otro)
	}
	panel(400, "PATCH", "/v1/panel/cupones/"+otro.ID, map[string]any{"dias": []int{7}}, nil)

	// El reporte de octubre (el mes pasado, por defecto, el 1/11).
	a.Ahora = en("2026-11-01 10:00")
	var rm reportes.ReporteMensual
	panel(200, "GET", "/v1/panel/reportes/mensual", nil, &rm)
	if rm.Mes != "2026-10" || rm.Nombre != "octubre 2026" || rm.Periodo != "Del 1 al 31 de octubre de 2026 · comparado con septiembre" || !rm.Completo {
		t.Fatalf("encabezado: %+v", rm)
	}
	k := rm.Indicadores
	if len(k) != 4 || k[0].ValorTexto != "$ 45.000" || k[0].Texto != "13% más que septiembre" || !k[0].Bueno ||
		k[1].ValorTexto != "5" || k[1].Texto != "1 más que septiembre" || k[2].ValorTexto != "71%" || k[3].Texto != "Igual que septiembre" {
		t.Fatalf("indicadores: %+v", k)
	}
	if rm.Titulo != "Tu mejor mes del año." || !strings.HasPrefix(rm.Bajada, "Facturaste un 13% más que en septiembre y casi nadie faltó.") ||
		!strings.Contains(rm.Bajada, "los martes a la tarde") {
		t.Fatalf("titular: %q / %q", rm.Titulo, rm.Bajada)
	}
	ag := rm.Agenda
	if ag.Titulo != "Ocupaste el 71% de tus horas libres." || ag.HorasLibres != 7 || ag.FranjaLibre == nil ||
		ag.FranjaLibre.Texto != "Martes, 14 a 16 h" || ag.Sugerencia == nil || ag.Sugerencia.Titulo != "Para llenar los martes a la tarde" {
		t.Fatalf("agenda: %+v %+v", ag, ag.FranjaLibre)
	}
	pl := rm.Plata
	if pl.Titulo != "Entraron $ 45.000." || len(pl.Semanas) != 5 || pl.Semanas[0].Etiqueta != "1–7 oct" || pl.Semanas[1].Ingresos != 15000 ||
		len(pl.Servicios) != 2 || pl.Servicios[0].Nombre != "Alisado" || *pl.Servicios[0].Parte != 88.9 {
		t.Fatalf("plata: %+v", pl)
	}
	if len(rm.Equipo) != 1 || rm.Equipo[0].Turnos != 5 || rm.Clientes.Recurrentes != 2 || rm.Clientes.Nuevos != 0 ||
		len(rm.Destacados) < 2 || len(rm.Siguiente) == 0 || rm.Siguiente[0].Numero != "01" {
		t.Fatalf("equipo y clientes: %+v %+v %+v %+v", rm.Equipo, rm.Clientes, rm.Destacados, rm.Siguiente)
	}
	panel(400, "GET", "/v1/panel/reportes/mensual?mes=2026-12", nil, nil)
	panel(400, "GET", "/v1/panel/reportes/mensual?mes=octubre", nil, nil)
	var nov reportes.ReporteMensual
	panel(200, "GET", "/v1/panel/reportes/mensual?mes=2026-11", nil, &nov)
	if nov.Completo {
		t.Fatal("noviembre recién empieza")
	}

	// El mail del día 1: a la dueña, una vez, con el link a la versión completa.
	var aj reportes.AjustesMensual
	panel(200, "GET", "/v1/panel/reportes/mensual/ajustes", nil, &aj)
	if !aj.Activo || len(aj.Encargados) != 0 {
		t.Fatalf("ajustes: %+v", aj)
	}
	ctx := context.Background()
	for range 2 {
		if _, err := reportes.EncolarMensual(ctx, a.Bases.App, a.Ahora); err != nil {
			t.Fatal(err)
		}
	}
	mails := &correo.Memoria{}
	env := &avisos.Enviador{Pool: a.Bases.App, Correo: mails, Sitio: "https://tengolibre.com.ar", Armadores: reportes.Armadores()}
	if res, err := env.Pasada(ctx, a.Ahora); err != nil || res.Enviados != 1 {
		t.Fatalf("envío: %+v %v", res, err)
	}
	m := mails.Para("duenia0001@prueba.tengolibre.com.ar")
	if len(m) != 1 || m[0].Asunto != "Tu octubre 2026 en El Corte" || !strings.Contains(m[0].Texto, "Tu mejor mes del año.") ||
		!strings.Contains(m[0].Texto, "Facturado: $ 45.000 · 13% más que septiembre") ||
		!strings.Contains(m[0].Texto, "https://tengolibre.com.ar/panel/reportes/mensual?mes=2026-10") {
		t.Fatalf("mail mensual: %+v", m)
	}
	// Apagado no sale; y otro día que no es el 1 tampoco.
	panel(200, "PUT", "/v1/panel/reportes/mensual/ajustes", map[string]any{"activo": false}, &aj)
	if aj.Activo {
		t.Fatal("no se apagó")
	}
	res, err := reportes.EncolarMensual(ctx, a.Bases.App, en("2026-12-01 10:00"))
	if err != nil || res.(map[string]int)["encolados"] != 0 {
		t.Fatalf("apagado: %v %v", res, err)
	}
	panel(404, "PUT", "/v1/panel/reportes/mensual/ajustes", map[string]any{"encargados": []map[string]any{{"id": n.ID, "recibe": true}}}, nil)
}
