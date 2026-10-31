package clientes_test

import (
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

const (
	duenia = "user_duenia0001"
	recep  = "user_recep00001"
	profe  = "user_profe00001"
	ajeno  = "user_ajeno00001"
)

type P = prueba.Pedido

func en(s string) time.Time {
	t, err := time.ParseInLocation("2006-01-02 15:04", s, disponibilidad.ZonaPorDefecto)
	if err != nil {
		panic(err)
	}
	return t
}

type mundo struct {
	*prueba.API
	t                            *testing.T
	negocio, servicio, ana, beto string
}

// armar: un negocio con un servicio de 30' a $8000, Ana y Beto los lunes de 9 a 11, y en el equipo
// una recepcionista y una profesional. Ahora: domingo 1/11/2026 a las 10.
func armar(t *testing.T) *mundo {
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
	m := &mundo{API: a, t: t}
	var n negocios.Negocio
	a.Esperar(201, P{Metodo: "POST", Ruta: "/v1/negocios", Quien: duenia, Cuerpo: map[string]string{"slug": "el-corte", "nombre": "El Corte"}}, &n)
	m.negocio = n.ID
	var s catalogo.Prestacion
	m.panel(201, "POST", "/v1/panel/servicios", map[string]any{"nombre": "Corte", "duracion": 30, "precio": 8000}, &s)
	m.servicio = s.ID
	horario := []disponibilidad.Franja{{Dia: 1, Desde: "09:00", Hasta: "11:00"}}
	var pa, pb catalogo.Profesional
	m.panel(201, "POST", "/v1/panel/profesionales", map[string]any{"nombre": "Ana", "servicios": []string{s.ID}, "horario": horario, "orden": 1}, &pa)
	m.panel(201, "POST", "/v1/panel/profesionales", map[string]any{"nombre": "Beto", "servicios": []string{s.ID}, "horario": horario, "orden": 2}, &pb)
	m.ana, m.beto = pa.ID, pb.ID
	m.panel(201, "POST", "/v1/panel/equipo", map[string]string{"clerkId": recep, "rol": "recepcion"}, nil)
	m.panel(201, "POST", "/v1/panel/equipo", map[string]string{"clerkId": profe, "rol": "profesional"}, nil)
	return m
}

// panel hace un pedido como la dueña.
func (m *mundo) panel(estado int, metodo, ruta string, cuerpo, v any) prueba.Respuesta {
	m.t.Helper()
	return m.Esperar(estado, P{Metodo: metodo, Ruta: ruta, Quien: duenia, Negocio: m.negocio, Cuerpo: cuerpo}, v)
}

func (m *mundo) como(quien string, estado int, metodo, ruta string, cuerpo, v any) prueba.Respuesta {
	m.t.Helper()
	return m.Esperar(estado, P{Metodo: metodo, Ruta: ruta, Quien: quien, Negocio: m.negocio, Cuerpo: cuerpo}, v)
}

func (m *mundo) reservar(estado int, profesional, inicio, nombre, telefono string) turnos.Turno {
	m.t.Helper()
	var t turnos.Turno
	r := m.Esperar(estado, P{Metodo: "POST", Ruta: "/v1/p/el-corte/turnos", Cuerpo: map[string]any{
		"servicioId": m.servicio, "profesionalId": profesional, "inicio": en(inicio).Format(time.RFC3339),
		"cliente": map[string]string{"nombre": nombre, "telefono": telefono}}}, nil)
	if estado == 201 {
		r.Leer(m.t, &t)
	}
	return t
}

func (m *mundo) estado(turno, estado string) {
	m.t.Helper()
	m.panel(200, "PATCH", "/v1/panel/turnos/"+turno, map[string]any{"estado": estado}, nil)
}

type listado struct {
	Clientes []clientes.Cliente `json:"clientes"`
	Total    int                `json:"total"`
}

func codigo(t *testing.T, r prueba.Respuesta) string {
	var p struct {
		Code string `json:"code"`
	}
	r.Leer(t, &p)
	return p.Code
}

func TestClientesDePuntaAPunta(t *testing.T) {
	m := armar(t)
	t1 := m.reservar(201, m.ana, "2026-11-02 09:00", "Juana Pérez", "11 1111-1111")
	m.reservar(201, m.ana, "2026-11-02 09:30", "Juana Pérez", "1111111111")
	m.reservar(201, m.beto, "2026-11-02 10:00", "Pedro Gómez", "2222222222")
	m.estado(t1.ID, "atendido")

	// El listado: dos clientes, ordenados por nombre, con lo que hicieron.
	var l listado
	m.panel(200, "GET", "/v1/panel/clientes", nil, &l)
	if l.Total != 2 || len(l.Clientes) != 2 || l.Clientes[0].Nombre != "Juana Pérez" {
		t.Fatalf("listado: %+v", l)
	}
	j := l.Clientes[0]
	if j.Visitas != 1 || j.Gastado != 8000 || j.UltimaVisita == nil || !j.UltimaVisita.Equal(en("2026-11-02 09:00")) ||
		j.ProximoTurno == nil || !j.ProximoTurno.Equal(en("2026-11-02 09:30")) || j.Ausencias != 0 || j.Penalizado {
		t.Fatalf("resumen de Juana: %+v", j)
	}

	// Búsqueda por nombre (sin importar mayúsculas), por teléfono escrito con guiones, y paginado.
	m.panel(200, "GET", "/v1/panel/clientes?q=juana", nil, &l)
	if l.Total != 1 {
		t.Fatalf("q=juana: %+v", l)
	}
	m.panel(200, "GET", "/v1/panel/clientes?q=222-222", nil, &l)
	if l.Total != 1 || l.Clientes[0].Nombre != "Pedro Gómez" {
		t.Fatalf("por teléfono: %+v", l)
	}
	m.panel(200, "GET", "/v1/panel/clientes?q=%25", nil, &l)
	if l.Total != 0 {
		t.Fatal("un % no es comodín")
	}
	m.panel(200, "GET", "/v1/panel/clientes?porPagina=1&pagina=2", nil, &l)
	if l.Total != 2 || len(l.Clientes) != 1 || l.Clientes[0].Nombre != "Pedro Gómez" {
		t.Fatalf("página 2: %+v", l)
	}
	m.panel(200, "GET", "/v1/panel/clientes?orden=gastado", nil, &l)
	if l.Clientes[0].Nombre != "Juana Pérez" {
		t.Fatal("orden por gastado")
	}
	m.panel(400, "GET", "/v1/panel/clientes?orden=cualquiera", nil, nil)

	// Alta manual: un teléfono que ya está es un 409 que dice de quién es.
	var c clientes.Cliente
	m.como(recep, 201, "POST", "/v1/panel/clientes", map[string]string{"nombre": "Walk In", "telefono": "33 33"}, &c)
	if c.Telefono != "3333" || c.Visitas != 0 || len(c.Etiquetas) != 0 {
		t.Fatalf("alta: %+v", c)
	}
	if r := m.panel(409, "POST", "/v1/panel/clientes", map[string]string{"nombre": "Otra", "telefono": "1111-111111"}, nil); codigo(t, r) != "cliente_existente" {
		t.Fatalf("teléfono repetido: %s", r.Cuerpo)
	}
	m.panel(400, "POST", "/v1/panel/clientes", map[string]string{"nombre": " "}, nil)
	// La reserva con ese teléfono cae en el mismo cliente.
	m.reservar(201, m.beto, "2026-11-02 10:30", "Walter", "3333")

	// Edición de nombre y email.
	m.panel(200, "PATCH", "/v1/panel/clientes/"+c.ID, map[string]any{"nombre": "Walter In", "email": "WALTER@mail.com"}, &c)
	if c.Nombre != "Walter In" || c.Email != "walter@mail.com" || c.ProximoTurno == nil {
		t.Fatalf("editado: %+v", c)
	}
	m.panel(400, "PATCH", "/v1/panel/clientes/"+c.ID, map[string]any{"email": "sin-arroba"}, nil)
	m.panel(400, "PATCH", "/v1/panel/clientes/"+c.ID, map[string]any{"telefono": "1"}, nil) // el teléfono no se edita

	// Ficha con historial.
	var f clientes.Ficha
	m.panel(200, "GET", "/v1/panel/clientes/"+j.ID, nil, &f)
	if len(f.Turnos) != 2 || !f.Turnos[0].Inicio.After(f.Turnos[1].Inicio) || f.Penalidad != nil || f.Gastado != 8000 {
		t.Fatalf("ficha: %+v", f)
	}

	// Etiquetas: color en mayúsculas, sin repetir el nombre, asignar es idempotente y filtra.
	var e clientes.Etiqueta
	m.panel(201, "POST", "/v1/panel/etiquetas", map[string]string{"nombre": "VIP", "color": "#ff0000"}, &e)
	if e.Color != "#FF0000" {
		t.Fatalf("etiqueta: %+v", e)
	}
	m.panel(409, "POST", "/v1/panel/etiquetas", map[string]string{"nombre": "vip"}, nil)
	m.panel(400, "POST", "/v1/panel/etiquetas", map[string]string{"nombre": "Mal", "color": "rojo"}, nil)
	var e2 clientes.Etiqueta
	m.panel(201, "POST", "/v1/panel/etiquetas", map[string]string{"nombre": "Nueva"}, &e2)
	if e2.Color != clientes.ColorPorDefecto {
		t.Fatalf("color por defecto: %+v", e2)
	}
	m.como(recep, 204, "PUT", "/v1/panel/clientes/"+j.ID+"/etiquetas/"+e.ID, nil, nil)
	m.como(recep, 204, "PUT", "/v1/panel/clientes/"+j.ID+"/etiquetas/"+e.ID, nil, nil)
	m.panel(200, "GET", "/v1/panel/clientes?etiqueta="+e.ID, nil, &l)
	if l.Total != 1 || len(l.Clientes[0].Etiquetas) != 1 || l.Clientes[0].Etiquetas[0].Nombre != "VIP" {
		t.Fatalf("filtro por etiqueta: %+v", l)
	}
	var es struct {
		Etiquetas []clientes.EtiquetaConUso `json:"etiquetas"`
	}
	m.como(profe, 200, "GET", "/v1/panel/etiquetas", nil, &es)
	if len(es.Etiquetas) != 2 || es.Etiquetas[1].Nombre != "VIP" || es.Etiquetas[1].Clientes != 1 {
		t.Fatalf("etiquetas: %+v", es)
	}
	m.panel(200, "PATCH", "/v1/panel/etiquetas/"+e.ID, map[string]string{"nombre": "Premium"}, &e)
	if e.Nombre != "Premium" || e.Color != "#FF0000" {
		t.Fatalf("etiqueta editada: %+v", e)
	}
	m.panel(204, "DELETE", "/v1/panel/clientes/"+j.ID+"/etiquetas/"+e2.ID, nil, nil) // no la tenía: no es error
	m.panel(204, "DELETE", "/v1/panel/etiquetas/"+e.ID, nil, nil)
	m.panel(200, "GET", "/v1/panel/clientes/"+j.ID, nil, &f)
	if len(f.Etiquetas) != 0 {
		t.Fatal("la etiqueta borrada sigue en el cliente")
	}
	m.panel(404, "PUT", "/v1/panel/clientes/"+j.ID+"/etiquetas/"+e.ID, nil, nil)

	// Notas: el autor es quien la escribió; la recepcionista no borra la de la dueña, la dueña sí la de ella.
	var n1, n2 clientes.Nota
	m.panel(201, "POST", "/v1/panel/clientes/"+j.ID+"/notas", map[string]string{"texto": "Prefiere a la mañana"}, &n1)
	m.como(recep, 201, "POST", "/v1/panel/clientes/"+j.ID+"/notas", map[string]string{"texto": "Alérgica al tinte"}, &n2)
	if n1.Autor != "Prueba" || n1.MiembroID == nil || n2.MiembroID == nil || *n1.MiembroID == *n2.MiembroID {
		t.Fatalf("autores: %+v %+v", n1, n2)
	}
	m.panel(400, "POST", "/v1/panel/clientes/"+j.ID+"/notas", map[string]string{"texto": "  "}, nil)
	m.como(recep, 403, "DELETE", "/v1/panel/clientes/"+j.ID+"/notas/"+n1.ID, nil, nil)
	m.panel(204, "DELETE", "/v1/panel/clientes/"+j.ID+"/notas/"+n2.ID, nil, nil)
	var ns struct {
		Notas []clientes.Nota `json:"notas"`
	}
	m.como(profe, 200, "GET", "/v1/panel/clientes/"+j.ID+"/notas", nil, &ns)
	if len(ns.Notas) != 1 || ns.Notas[0].Texto != "Prefiere a la mañana" {
		t.Fatalf("notas: %+v", ns)
	}

	// Búsqueda global: clientes, turnos (por cliente o servicio) y servicios.
	var b clientes.Resultados
	m.panel(200, "GET", "/v1/panel/buscar?q=corte", nil, &b)
	if len(b.Servicios) != 1 || len(b.Turnos) != 4 || len(b.Clientes) != 0 {
		t.Fatalf("buscar corte: %+v", b)
	}
	m.panel(200, "GET", "/v1/panel/buscar?q=1111", nil, &b)
	if len(b.Clientes) != 1 || len(b.Turnos) != 2 || len(b.Servicios) != 0 {
		t.Fatalf("buscar 1111: %+v", b)
	}
	m.panel(200, "GET", "/v1/panel/buscar?q=j", nil, &b)
	if len(b.Clientes)+len(b.Turnos)+len(b.Servicios) != 0 {
		t.Fatal("con una letra no se busca")
	}
	// La profesional sin profesional asociado no ve turnos en la búsqueda (como en la agenda).
	m.como(profe, 200, "GET", "/v1/panel/buscar?q=corte", nil, &b)
	if len(b.Turnos) != 0 || len(b.Servicios) != 1 {
		t.Fatalf("buscar como profesional: %+v", b)
	}
}

func TestPenalidadesPorAusencias(t *testing.T) {
	m := armar(t)
	var aj clientes.AjustesAusencias
	m.panel(200, "GET", "/v1/panel/ausencias/ajustes", nil, &aj)
	if aj.UmbralAusencias != 3 || aj.DiasPenalidad != 30 || aj.MarcarAusentes {
		t.Fatalf("ajustes por defecto: %+v", aj)
	}
	m.panel(400, "PUT", "/v1/panel/ausencias/ajustes", map[string]any{"umbralAusencias": 2, "diasPenalidad": 0, "marcarAusentes": false}, nil)
	m.panel(200, "PUT", "/v1/panel/ausencias/ajustes", map[string]any{"umbralAusencias": 2, "diasPenalidad": 30, "marcarAusentes": false}, &aj)
	m.como(recep, 403, "PUT", "/v1/panel/ausencias/ajustes", map[string]any{"umbralAusencias": 0, "diasPenalidad": 1, "marcarAusentes": false}, nil)
	// Los ajustes de reserva siguen intactos.
	var ar negocios.Ajustes
	m.panel(200, "GET", "/v1/panel/ajustes", nil, &ar)
	if ar.Intervalo != 30 {
		t.Fatalf("ajustes de reserva: %+v", ar)
	}

	// Dos ausencias: la segunda penaliza sola.
	a1 := m.reservar(201, m.ana, "2026-11-02 09:00", "Juana", "1111")
	a2 := m.reservar(201, m.ana, "2026-11-02 09:30", "Juana", "1111")
	m.estado(a1.ID, "ausente")
	var ps struct {
		Penalidades []clientes.Penalidad `json:"penalidades"`
		Total       int                  `json:"total"`
	}
	m.panel(200, "GET", "/v1/panel/penalidades", nil, &ps)
	if ps.Total != 0 {
		t.Fatal("con una ausencia no se penaliza")
	}
	m.estado(a2.ID, "ausente")
	m.como(profe, 200, "GET", "/v1/panel/penalidades", nil, &ps)
	if ps.Total != 1 || ps.Penalidades[0].Motivo != "ausencias" || ps.Penalidades[0].Ausencias != 2 || !ps.Penalidades[0].Vigente ||
		ps.Penalidades[0].Cliente != "Juana" {
		t.Fatalf("penalidad automática: %+v", ps)
	}
	pen := ps.Penalidades[0]

	// Penalizada: desde la web no reserva; desde el panel sí.
	if r := m.Hacer(P{Metodo: "POST", Ruta: "/v1/p/el-corte/turnos", Cuerpo: map[string]any{
		"servicioId": m.servicio, "profesionalId": m.ana, "inicio": en("2026-11-02 10:00").Format(time.RFC3339),
		"cliente": map[string]string{"nombre": "Juana", "telefono": "11-11"}}}); r.Estado != 409 || codigo(t, r) != "cliente_penalizado" {
		t.Fatalf("reserva web penalizada: %d %s", r.Estado, r.Cuerpo)
	}
	m.panel(201, "POST", "/v1/panel/turnos", map[string]any{"servicioId": m.servicio, "profesionalId": m.ana,
		"inicio": en("2026-11-02 10:00").Format(time.RFC3339), "cliente": map[string]string{"nombre": "Juana", "telefono": "1111"}}, nil)
	var l listado
	m.panel(200, "GET", "/v1/panel/clientes?penalizados=1", nil, &l)
	if l.Total != 1 || !l.Clientes[0].Penalizado || l.Clientes[0].Ausencias != 2 {
		t.Fatalf("filtro penalizados: %+v", l)
	}
	var f clientes.Ficha
	m.panel(200, "GET", "/v1/panel/clientes/"+pen.ClienteID, nil, &f)
	if f.Penalidad == nil || f.Penalidad.ID != pen.ID {
		t.Fatalf("ficha sin la penalidad: %+v", f.Penalidad)
	}

	// Levantarla: sólo gestión, y una sola vez. Después reserva de nuevo.
	m.como(recep, 403, "POST", "/v1/panel/penalidades/"+pen.ID+"/levantar", nil, nil)
	var lev clientes.Penalidad
	m.panel(200, "POST", "/v1/panel/penalidades/"+pen.ID+"/levantar", nil, &lev)
	if lev.Vigente || lev.LevantadaEn == nil {
		t.Fatalf("levantada: %+v", lev)
	}
	m.panel(409, "POST", "/v1/panel/penalidades/"+pen.ID+"/levantar", nil, nil)
	a3 := m.reservar(201, m.ana, "2026-11-02 10:30", "Juana", "1111")

	// Una ausencia más no la vuelve a penalizar: las dos de antes ya se pagaron.
	m.estado(a3.ID, "ausente")
	m.panel(200, "GET", "/v1/panel/penalidades", nil, &ps)
	if ps.Total != 0 {
		t.Fatalf("re-penalizada con las ausencias viejas: %+v", ps)
	}
	m.panel(200, "GET", "/v1/panel/penalidades?todas=1", nil, &ps)
	if ps.Total != 1 {
		t.Fatalf("todas: %+v", ps)
	}

	// Penalidad manual: por los días de los ajustes, y no dos a la vez.
	m.como(recep, 403, "POST", "/v1/panel/clientes/"+pen.ClienteID+"/penalidades", map[string]any{"nota": "x"}, nil)
	var man clientes.Penalidad
	m.panel(201, "POST", "/v1/panel/clientes/"+pen.ClienteID+"/penalidades", map[string]any{"nota": "Llegó tarde tres veces"}, &man)
	if man.Motivo != "manual" || !man.Vigente || man.Hasta.Sub(man.CreadoEn) < 29*24*time.Hour {
		t.Fatalf("manual: %+v", man)
	}
	if r := m.panel(409, "POST", "/v1/panel/clientes/"+pen.ClienteID+"/penalidades", map[string]any{"dias": 3}, nil); codigo(t, r) != "ya_penalizado" {
		t.Fatalf("doble: %s", r.Cuerpo)
	}
	var del struct {
		Penalidades []clientes.Penalidad `json:"penalidades"`
	}
	m.panel(200, "GET", "/v1/panel/clientes/"+pen.ClienteID+"/penalidades", nil, &del)
	if len(del.Penalidades) != 2 {
		t.Fatalf("penalidades del cliente: %+v", del)
	}

	// Marcado automático: apagado no toca nada; prendido, pasadas dos horas del fin, marca ausente.
	b1 := m.reservar(201, m.beto, "2026-11-02 09:00", "Pedro", "2222")
	m.API.Ahora = en("2026-11-02 12:00")
	var mk struct {
		Marcados int `json:"marcados"`
	}
	m.panel(200, "POST", "/v1/panel/ausencias/marcar", nil, &mk)
	if mk.Marcados != 0 {
		t.Fatal("marcó con el ajuste apagado")
	}
	m.panel(200, "PUT", "/v1/panel/ausencias/ajustes", map[string]any{"umbralAusencias": 1, "diasPenalidad": 7, "marcarAusentes": true}, nil)
	// El listado lo hace al pasar: el de Pedro (9 a 9:30) pasa a ausente, el de Juana de las 10 (fin 10:30) todavía no.
	m.panel(200, "GET", "/v1/panel/clientes", nil, &l)
	var tb turnos.Turno
	m.panel(200, "GET", "/v1/panel/turnos/"+b1.ID, nil, &tb)
	if tb.Estado != "ausente" {
		t.Fatalf("el turno de Pedro tendría que estar ausente: %s", tb.Estado)
	}
	// Con umbral 1, a Pedro lo penaliza el trigger.
	m.panel(200, "GET", "/v1/panel/penalidades", nil, &ps)
	if ps.Total != 2 {
		t.Fatalf("vigentes: %+v", ps)
	}
	m.API.Ahora = en("2026-11-02 12:31")
	m.panel(200, "POST", "/v1/panel/ausencias/marcar", nil, &mk)
	if mk.Marcados != 1 {
		t.Fatalf("marcados a las 12:31: %d", mk.Marcados)
	}

	var au struct {
		Total     int `json:"total"`
		Ultimos30 int `json:"ultimos30"`
		Vigentes  int `json:"penalidadesVigentes"`
		Reinc     int `json:"reincidentes"`
	}
	m.como(recep, 200, "GET", "/v1/panel/ausencias", nil, &au)
	if au.Total != 5 || au.Ultimos30 != 5 || au.Vigentes != 2 || au.Reinc != 1 {
		t.Fatalf("resumen de ausencias: %+v", au)
	}
}

func TestAislamientoYPermisos(t *testing.T) {
	m := armar(t)
	m.reservar(201, m.ana, "2026-11-02 09:00", "Juana", "1111")
	var l listado
	m.panel(200, "GET", "/v1/panel/clientes", nil, &l)
	juana := l.Clientes[0].ID
	var e clientes.Etiqueta
	m.panel(201, "POST", "/v1/panel/etiquetas", map[string]string{"nombre": "VIP"}, &e)

	// Sin sesión, sin negocio, o de otro negocio: no entra.
	m.Esperar(401, P{Metodo: "GET", Ruta: "/v1/panel/clientes"}, nil)
	m.Esperar(400, P{Metodo: "GET", Ruta: "/v1/panel/clientes", Quien: duenia}, nil)
	m.Esperar(403, P{Metodo: "GET", Ruta: "/v1/panel/clientes", Quien: ajeno, Negocio: m.negocio}, nil)

	// Otro negocio no ve ni toca a los clientes de este.
	var otro negocios.Negocio
	m.Esperar(201, P{Metodo: "POST", Ruta: "/v1/negocios", Quien: ajeno, Cuerpo: map[string]string{"slug": "otro", "nombre": "Otro"}}, &otro)
	o := func(estado int, metodo, ruta string, cuerpo, v any) {
		t.Helper()
		m.Esperar(estado, P{Metodo: metodo, Ruta: ruta, Quien: ajeno, Negocio: otro.ID, Cuerpo: cuerpo}, v)
	}
	o(200, "GET", "/v1/panel/clientes", nil, &l)
	if l.Total != 0 {
		t.Fatal("otro negocio ve clientes ajenos")
	}
	o(404, "GET", "/v1/panel/clientes/"+juana, nil, nil)
	o(404, "PATCH", "/v1/panel/clientes/"+juana, map[string]string{"nombre": "Hackeada"}, nil)
	o(404, "GET", "/v1/panel/clientes/"+juana+"/notas", nil, nil)
	o(404, "POST", "/v1/panel/clientes/"+juana+"/notas", map[string]string{"texto": "x"}, nil)
	o(404, "POST", "/v1/panel/clientes/"+juana+"/penalidades", map[string]any{}, nil)
	o(404, "PATCH", "/v1/panel/etiquetas/"+e.ID, map[string]string{"nombre": "x"}, nil)
	o(404, "DELETE", "/v1/panel/etiquetas/"+e.ID, nil, nil)
	var suya clientes.Cliente
	o(201, "POST", "/v1/panel/clientes", map[string]string{"nombre": "Suya", "telefono": "1111"}, &suya) // el mismo teléfono, en otro negocio, es otro cliente
	o(404, "PUT", "/v1/panel/clientes/"+suya.ID+"/etiquetas/"+e.ID, nil, nil)                            // la etiqueta de este negocio
	o(404, "PUT", "/v1/panel/clientes/"+juana+"/etiquetas/"+e.ID, nil, nil)
	var b clientes.Resultados
	o(200, "GET", "/v1/panel/buscar?q=juana", nil, &b)
	if len(b.Clientes)+len(b.Turnos) != 0 {
		t.Fatalf("la búsqueda de otro negocio ve clientes ajenos: %+v", b)
	}
	var rep reportes.Reporte
	o(200, "GET", "/v1/panel/reportes?desde=2026-11-01&hasta=2026-11-30", nil, &rep)
	if rep.Resumen.Turnos != 0 || len(rep.PorProfesional) != 0 {
		t.Fatalf("el reporte de otro negocio ve turnos ajenos: %+v", rep)
	}
	m.panel(200, "GET", "/v1/panel/clientes/"+juana, nil, nil) // y a Juana no le pasó nada
	m.panel(400, "GET", "/v1/panel/clientes/no-es-un-id", nil, nil)

	// Roles: la profesional ve clientes y escribe notas, pero no da de alta ni edita ni etiqueta;
	// recepción da de alta pero no maneja etiquetas, penalidades ni reportes.
	m.como(profe, 200, "GET", "/v1/panel/clientes/"+juana, nil, nil)
	m.como(profe, 201, "POST", "/v1/panel/clientes/"+juana+"/notas", map[string]string{"texto": "ok"}, nil)
	m.como(profe, 403, "POST", "/v1/panel/clientes", map[string]string{"nombre": "X"}, nil)
	m.como(profe, 403, "PATCH", "/v1/panel/clientes/"+juana, map[string]string{"nombre": "X"}, nil)
	m.como(profe, 403, "PUT", "/v1/panel/clientes/"+juana+"/etiquetas/"+e.ID, nil, nil)
	m.como(recep, 403, "POST", "/v1/panel/etiquetas", map[string]string{"nombre": "X"}, nil)
	m.como(recep, 403, "DELETE", "/v1/panel/etiquetas/"+e.ID, nil, nil)
	for _, ruta := range []string{"/v1/panel/reportes", "/v1/panel/estadisticas", "/v1/panel/reportes/clientes.csv", "/v1/panel/reportes/turnos.csv"} {
		m.como(recep, 403, "GET", ruta, nil, nil)
		m.como(profe, 403, "GET", ruta, nil, nil)
	}
	m.como(recep, 403, "POST", "/v1/panel/ausencias/marcar", nil, nil)
}
