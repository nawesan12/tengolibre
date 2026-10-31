package turnos_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/catalogo"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/disponibilidad"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/negocios"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/turnos"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/prueba"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/servidor"
)

const duenia = "user_duenia0001"

type P = prueba.Pedido

var z = disponibilidad.ZonaPorDefecto

func en(s string) time.Time {
	t, err := time.ParseInLocation("2006-01-02 15:04", s, z)
	if err != nil {
		panic(err)
	}
	return t
}

// armar: un negocio con un servicio de 30' y dos profesionales que atienden los lunes de 9 a 11.
// Ahora: domingo 1/11/2026 a las 10.
func armar(t *testing.T) (a *prueba.API, negocio, servicio, ana, beto string) {
	a = prueba.Nueva(t, en("2026-11-01 10:00"), func(a *prueba.API) []servidor.Dominio {
		app := a.Bases.App
		return []servidor.Dominio{
			&negocios.Servicio{Pool: app, Sesion: a.Sesion},
			&catalogo.Servicio{Pool: app, Sesion: a.Sesion},
			&turnos.Servicio{Pool: app, Sesion: a.Sesion, Ahora: a.Reloj()},
		}
	})
	var n negocios.Negocio
	a.Esperar(http.StatusCreated, P{Metodo: "POST", Ruta: "/v1/negocios", Quien: duenia,
		Cuerpo: map[string]string{"slug": "el-corte", "nombre": "El Corte"}}, &n)
	negocio = n.ID
	panel := func(metodo, ruta string, cuerpo any, v any) {
		a.Esperar(map[string]int{"POST": 201, "PUT": 200, "PATCH": 200}[metodo], P{Metodo: metodo, Ruta: ruta, Quien: duenia, Negocio: negocio, Cuerpo: cuerpo}, v)
	}
	var s catalogo.Prestacion
	panel("POST", "/v1/panel/servicios", map[string]any{"nombre": "Corte", "duracion": 30, "precio": 8000}, &s)
	horario := []disponibilidad.Franja{{Dia: 1, Desde: "09:00", Hasta: "11:00"}}
	var pa, pb catalogo.Profesional
	panel("POST", "/v1/panel/profesionales", map[string]any{"nombre": "Ana", "servicios": []string{s.ID}, "horario": horario, "orden": 1}, &pa)
	panel("POST", "/v1/panel/profesionales", map[string]any{"nombre": "Beto", "servicios": []string{s.ID}, "horario": horario, "orden": 2}, &pb)
	if len(pa.Horario) != 1 || pa.Horario[0].Desde != "09:00" || len(pa.Servicios) != 1 {
		t.Fatalf("profesional mal guardado: %+v", pa)
	}
	return a, negocio, s.ID, pa.ID, pb.ID
}

type libresResp struct {
	Horarios []turnos.HorarioLibre `json:"horarios"`
}

func reservar(a *prueba.API, estado int, servicio, profesional, inicio, telefono string) turnos.Turno {
	var t turnos.Turno
	a.Esperar(estado, P{Metodo: "POST", Ruta: "/v1/p/el-corte/turnos", Cuerpo: map[string]any{
		"servicioId": servicio, "profesionalId": profesional, "inicio": en(inicio).Format(time.RFC3339),
		"cliente": map[string]string{"nombre": "Cliente " + telefono, "telefono": telefono}}}, func() any {
		if estado == http.StatusCreated {
			return &t
		}
		return nil
	}())
	return t
}

func TestReservaDePuntaAPunta(t *testing.T) {
	a, negocio, servicio, ana, beto := armar(t)

	// La página pública muestra el negocio, el servicio y los dos profesionales, sin datos de contacto.
	var pag struct {
		Negocio       negocios.Negocio      `json:"negocio"`
		Servicios     []catalogo.Prestacion `json:"servicios"`
		Profesionales []map[string]any      `json:"profesionales"`
	}
	a.Esperar(200, P{Metodo: "GET", Ruta: "/v1/p/el-corte"}, &pag)
	if pag.Negocio.Nombre != "El Corte" || len(pag.Servicios) != 1 || len(pag.Profesionales) != 2 {
		t.Fatalf("página: %+v", pag)
	}
	if _, ok := pag.Profesionales[0]["email"]; ok {
		t.Fatal("la página pública no muestra el email del profesional")
	}

	// Lunes 2: 4 horarios, cada uno con los dos libres.
	var l libresResp
	a.Esperar(200, P{Metodo: "GET", Ruta: "/v1/p/el-corte/libres?servicio=" + servicio + "&fecha=2026-11-02"}, &l)
	if len(l.Horarios) != 4 || len(l.Horarios[0].Profesionales) != 2 || l.Horarios[0].Hora != "09:00" {
		t.Fatalf("libres: %+v", l.Horarios)
	}

	// "Cualquiera" a las 9 se lo da a Ana (primera en orden); otra persona a las 9 va con Beto; una
	// tercera ya no entra.
	t1 := reservar(a, 201, servicio, "", "2026-11-02 09:00", "1111")
	if t1.ProfesionalID == nil || *t1.ProfesionalID != ana || t1.Estado != "pendiente" || t1.Precio != 8000 {
		t.Fatalf("primer turno: %+v", t1)
	}
	t2 := reservar(a, 201, servicio, "", "2026-11-02 09:00", "2222")
	if *t2.ProfesionalID != beto {
		t.Fatalf("segundo turno con %v, quería a Beto", *t2.ProfesionalID)
	}
	reservar(a, 409, servicio, "", "2026-11-02 09:00", "3333")
	reservar(a, 409, servicio, ana, "2026-11-02 09:00", "3333")
	// Un horario fuera de la grilla tampoco.
	reservar(a, 409, servicio, "", "2026-11-02 09:10", "3333")

	a.Esperar(200, P{Metodo: "GET", Ruta: "/v1/p/el-corte/libres?servicio=" + servicio + "&fecha=2026-11-02"}, &l)
	if l.Horarios[0].Hora != "09:30" {
		t.Fatalf("las 9 tendrían que estar ocupadas: %+v", l.Horarios)
	}

	// El mismo teléfono escrito distinto es el mismo cliente.
	reservar(a, 201, servicio, ana, "2026-11-02 10:00", "11-11")
	var ag struct {
		Turnos []turnos.Turno `json:"turnos"`
	}
	a.Esperar(200, P{Metodo: "GET", Ruta: "/v1/panel/turnos?desde=2026-11-02", Quien: duenia, Negocio: negocio}, &ag)
	if len(ag.Turnos) != 3 {
		t.Fatalf("agenda: %d turnos", len(ag.Turnos))
	}
	clientes := map[string]bool{}
	for _, x := range ag.Turnos {
		clientes[x.Cliente.ID] = true
	}
	if len(clientes) != 2 {
		t.Fatalf("1111 y 11-11 tendrían que ser el mismo cliente: %d clientes", len(clientes))
	}

	// El link del cliente: lo ve, lo reprograma y lo cancela.
	var tc turnos.TurnoCliente
	a.Esperar(200, P{Metodo: "GET", Ruta: "/v1/t/" + t1.Codigo}, &tc)
	if !tc.PuedeCancelar || tc.Negocio.Nombre != "El Corte" || tc.Cliente.ID != "" {
		t.Fatalf("link: %+v", tc)
	}
	a.Esperar(409, P{Metodo: "POST", Ruta: "/v1/t/" + t1.Codigo + "/reprogramar",
		Cuerpo: map[string]any{"inicio": en("2026-11-02 10:00").Format(time.RFC3339)}}, nil) // Ana ya tiene las 10
	a.Esperar(200, P{Metodo: "POST", Ruta: "/v1/t/" + t1.Codigo + "/reprogramar",
		Cuerpo: map[string]any{"inicio": en("2026-11-02 10:30").Format(time.RFC3339)}}, &tc)
	if !tc.Inicio.Equal(en("2026-11-02 10:30")) || *tc.ProfesionalID != ana {
		t.Fatalf("reprogramado: %+v", tc.Turno)
	}
	// Reprogramar al mismo horario no choca consigo mismo.
	a.Esperar(200, P{Metodo: "POST", Ruta: "/v1/t/" + t1.Codigo + "/reprogramar",
		Cuerpo: map[string]any{"inicio": en("2026-11-02 10:30").Format(time.RFC3339)}}, nil)
	a.Esperar(200, P{Metodo: "POST", Ruta: "/v1/t/" + t1.Codigo + "/cancelar"}, &tc)
	if tc.Estado != "cancelado" || tc.PuedeCancelar {
		t.Fatalf("cancelado: %+v", tc)
	}
	// Cancelado, el horario vuelve a estar libre.
	reservar(a, 201, servicio, ana, "2026-11-02 10:30", "4444")

	// Pasado el plazo (2 h antes por defecto), el cliente ya no cancela.
	a.Ahora = en("2026-11-02 08:30")
	a.Esperar(409, P{Metodo: "POST", Ruta: "/v1/t/" + t2.Codigo + "/cancelar"}, nil)

	// El panel marca atendido y no deja volver a pendiente_pago ni tocar un cancelado.
	var x turnos.Turno
	a.Esperar(200, P{Metodo: "PATCH", Ruta: "/v1/panel/turnos/" + t2.ID, Quien: duenia, Negocio: negocio,
		Cuerpo: map[string]any{"estado": "atendido", "nota": "todo bien"}}, &x)
	if x.Estado != "atendido" || x.Nota != "todo bien" {
		t.Fatalf("estado: %+v", x)
	}
	a.Esperar(409, P{Metodo: "PATCH", Ruta: "/v1/panel/turnos/" + t1.ID, Quien: duenia, Negocio: negocio,
		Cuerpo: map[string]any{"estado": "confirmado"}}, nil)

	// Sobreturno desde el panel: forzado entra aunque esté ocupado, y queda confirmado.
	var st turnos.Turno
	a.Esperar(201, P{Metodo: "POST", Ruta: "/v1/panel/turnos", Quien: duenia, Negocio: negocio, Cuerpo: map[string]any{
		"servicioId": servicio, "profesionalId": ana, "inicio": en("2026-11-02 10:00").Format(time.RFC3339), "forzar": true,
		"cliente": map[string]string{"nombre": "Sobreturno", "email": "sobre@turno.com"}}}, &st)
	if st.Estado != "confirmado" || st.Origen != "panel" {
		t.Fatalf("sobreturno: %+v", st)
	}
}

func TestPermisosYAislamiento(t *testing.T) {
	a, negocio, servicio, _, _ := armar(t)
	ajeno := "user_ajeno00001"
	// Sin sesión, sin X-Negocio, o de otro: no entra.
	a.Esperar(401, P{Metodo: "GET", Ruta: "/v1/panel/turnos"}, nil)
	a.Esperar(400, P{Metodo: "GET", Ruta: "/v1/panel/turnos", Quien: duenia}, nil)
	a.Esperar(403, P{Metodo: "GET", Ruta: "/v1/panel/turnos", Quien: ajeno, Negocio: negocio}, nil)

	// Otro negocio no ve ni toca lo de este.
	var otro negocios.Negocio
	a.Esperar(201, P{Metodo: "POST", Ruta: "/v1/negocios", Quien: ajeno, Cuerpo: map[string]string{"slug": "otro", "nombre": "Otro"}}, &otro)
	var ss struct {
		Servicios []catalogo.Prestacion `json:"servicios"`
	}
	a.Esperar(200, P{Metodo: "GET", Ruta: "/v1/panel/servicios", Quien: ajeno, Negocio: otro.ID}, &ss)
	if len(ss.Servicios) != 0 {
		t.Fatal("otro negocio ve servicios ajenos")
	}
	a.Esperar(404, P{Metodo: "PATCH", Ruta: "/v1/panel/servicios/" + servicio, Quien: ajeno, Negocio: otro.ID,
		Cuerpo: map[string]any{"precio": 1}}, nil)
	a.Esperar(404, P{Metodo: "DELETE", Ruta: "/v1/panel/servicios/" + servicio, Quien: ajeno, Negocio: otro.ID}, nil)

	// Slugs: ocupado, reservado y mal formado.
	a.Esperar(409, P{Metodo: "POST", Ruta: "/v1/negocios", Quien: ajeno, Cuerpo: map[string]string{"slug": "el-corte", "nombre": "X"}}, nil)
	a.Esperar(409, P{Metodo: "POST", Ruta: "/v1/negocios", Quien: ajeno, Cuerpo: map[string]string{"slug": "panel", "nombre": "X"}}, nil)
	a.Esperar(400, P{Metodo: "POST", Ruta: "/v1/negocios", Quien: ajeno, Cuerpo: map[string]string{"slug": "Con Espacios", "nombre": "X"}}, nil)

	// Equipo: la dueña suma a una recepcionista, que maneja turnos pero no el catálogo.
	recep := "user_recep00001"
	a.Esperar(201, P{Metodo: "POST", Ruta: "/v1/panel/equipo", Quien: duenia, Negocio: negocio,
		Cuerpo: map[string]string{"clerkId": recep, "rol": "recepcion"}}, nil)
	a.Esperar(200, P{Metodo: "GET", Ruta: "/v1/panel/turnos", Quien: recep, Negocio: negocio}, nil)
	a.Esperar(403, P{Metodo: "POST", Ruta: "/v1/panel/servicios", Quien: recep, Negocio: negocio,
		Cuerpo: map[string]any{"nombre": "X", "duracion": 30, "precio": 1}}, nil)
	var yo struct {
		Negocios []map[string]string `json:"negocios"`
	}
	a.Esperar(200, P{Metodo: "GET", Ruta: "/v1/yo", Quien: recep}, &yo)
	if len(yo.Negocios) != 1 || yo.Negocios[0]["rol"] != "recepcion" {
		t.Fatalf("yo: %+v", yo)
	}

	// La única dueña no puede quedarse afuera.
	var eq struct {
		Miembros []negocios.Miembro `json:"miembros"`
	}
	a.Esperar(200, P{Metodo: "GET", Ruta: "/v1/panel/equipo", Quien: duenia, Negocio: negocio}, &eq)
	for _, m := range eq.Miembros {
		if m.Rol == "duenio" {
			a.Esperar(409, P{Metodo: "PATCH", Ruta: "/v1/panel/equipo/" + m.ID, Quien: duenia, Negocio: negocio,
				Cuerpo: map[string]any{"activo": false}}, nil)
		}
	}
}

func TestBloqueosYAjustesCambianLosLibres(t *testing.T) {
	a, negocio, servicio, ana, _ := armar(t)
	libres := func() int {
		var l libresResp
		a.Esperar(200, P{Metodo: "GET", Ruta: "/v1/p/el-corte/libres?servicio=" + servicio + "&profesional=" + ana + "&fecha=2026-11-02"}, &l)
		return len(l.Horarios)
	}
	if libres() != 4 {
		t.Fatal("arranca con 4")
	}
	quien := func(metodo, ruta string, cuerpo any, estado int) {
		a.Esperar(estado, P{Metodo: metodo, Ruta: ruta, Quien: duenia, Negocio: negocio, Cuerpo: cuerpo}, nil)
	}
	quien("POST", "/v1/panel/profesionales/"+ana+"/bloqueos", map[string]any{"fecha": "2026-11-02", "desde": "09:00", "hasta": "10:00"}, 201)
	if libres() != 2 {
		t.Fatal("el bloqueo parcial saca dos")
	}
	quien("PUT", "/v1/panel/profesionales/"+ana+"/semanales", map[string]any{"semanales": []map[string]any{{"dia": 1, "hora": "10:30"}}}, 200)
	if libres() != 1 {
		t.Fatal("el semanal saca uno")
	}
	quien("PUT", "/v1/panel/profesionales/"+ana+"/excepciones", map[string]any{"fecha": "2026-11-02", "hora": "10:30", "tipo": "libre"}, 200)
	if libres() != 2 {
		t.Fatal("la excepción libre lo devuelve")
	}
	quien("PUT", "/v1/panel/ajustes", map[string]any{"intervalo": 15, "anticipacionMin": 0, "diasVista": 30, "margenMin": 0,
		"cancelarHastaHoras": 2, "politicaCancelacion": ""}, 200)
	if libres() != 3 { // 10:00 10:15 10:30 (10:45 + 30 pasa del cierre); el semanal de 10:30 lo libera la excepción
		t.Fatalf("con intervalo 15: %d", libres())
	}
	var d struct {
		Dias []turnos.Dia `json:"dias"`
	}
	a.Esperar(200, P{Metodo: "GET", Ruta: "/v1/p/el-corte/dias?servicio=" + servicio + "&dias=8"}, &d)
	if len(d.Dias) != 8 || d.Dias[1].Fecha != "2026-11-02" || d.Dias[1].Libres == 0 || d.Dias[0].Libres != 0 {
		t.Fatalf("días: %+v", d.Dias)
	}
	// Horarios que se pisan no se guardan.
	quien("PUT", "/v1/panel/profesionales/"+ana+"/horario", map[string]any{"horario": []map[string]any{
		{"dia": 1, "desde": "09:00", "hasta": "12:00"}, {"dia": 1, "desde": "11:00", "hasta": "13:00"}}}, 400)
}
