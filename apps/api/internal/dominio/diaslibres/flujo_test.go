package diaslibres_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/avisos"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/catalogo"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/diaslibres"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/disponibilidad"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/espera"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/negocios"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/turnos"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/prueba"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/servidor"
)

type P = prueba.Pedido

const (
	duenia = "user_duenia0001"
	ana    = "user_ana00000001"
	recep  = "user_recep000001"
)

func en(s string) time.Time {
	t, err := time.ParseInLocation("2006-01-02 15:04", s, disponibilidad.ZonaPorDefecto)
	if err != nil {
		panic(err)
	}
	return t
}

func sql(t *testing.T, a *prueba.API, q string, args ...any) {
	t.Helper()
	ctx := context.Background()
	c, err := pgx.Connect(ctx, a.Bases.AdminURL)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close(ctx)
	if _, err := c.Exec(ctx, q, args...); err != nil {
		t.Fatal(err)
	}
}

type pedidos struct {
	Pedidos []diaslibres.Pedido `json:"pedidos"`
}

type paraMover struct {
	Turnos []diaslibres.Reprogramar `json:"turnos"`
}

func TestPedirDiasLibres(t *testing.T) {
	a := prueba.Nueva(t, en("2026-11-01 10:00"), func(a *prueba.API) []servidor.Dominio {
		app := a.Bases.App
		av := &avisos.Servicio{Pool: app, Sesion: a.Sesion, Ahora: a.Reloj()}
		tur := &turnos.Servicio{Pool: app, Sesion: a.Sesion, Ahora: a.Reloj(), Avisar: av.Avisar}
		return []servidor.Dominio{
			&negocios.Servicio{Pool: app, Sesion: a.Sesion},
			&catalogo.Servicio{Pool: app, Sesion: a.Sesion},
			tur, av,
			&espera.Servicio{Pool: app, Sesion: a.Sesion, Ahora: a.Reloj(), Turnos: tur},
			&diaslibres.Servicio{Pool: app, Sesion: a.Sesion, Ahora: a.Reloj()},
		}
	})
	var n negocios.Negocio
	a.Esperar(201, P{Metodo: "POST", Ruta: "/v1/negocios", Quien: duenia, Cuerpo: map[string]string{"slug": "el-corte", "nombre": "El Corte"}}, &n)
	panel := func(estado int, quien, metodo, ruta string, cuerpo, v any) prueba.Respuesta {
		t.Helper()
		return a.Esperar(estado, P{Metodo: metodo, Ruta: ruta, Quien: quien, Negocio: n.ID, Cuerpo: cuerpo}, v)
	}
	var s catalogo.Prestacion
	panel(201, duenia, "POST", "/v1/panel/servicios", map[string]any{"nombre": "Corte", "duracion": 30, "precio": 8000}, &s)
	horario := []disponibilidad.Franja{{Dia: 1, Desde: "09:00", Hasta: "11:00"}}
	var pa, pb catalogo.Profesional
	panel(201, duenia, "POST", "/v1/panel/profesionales", map[string]any{"nombre": "Ana", "servicios": []string{s.ID}, "horario": horario, "orden": 1}, &pa)
	panel(201, duenia, "POST", "/v1/panel/profesionales", map[string]any{"nombre": "Beto", "servicios": []string{s.ID}, "horario": horario, "orden": 2}, &pb)
	// Ana entra como profesional (con su usuario vinculado) y hay una recepcionista.
	sql(t, a, `WITH m AS (INSERT INTO miembros (negocio_id, clerk_id, email, nombre, rol) VALUES ($1, $2, 'ana@mail.com', 'Ana', 'profesional') RETURNING id)
		UPDATE profesionales SET miembro_id = (SELECT id FROM m) WHERE id = $3`, n.ID, ana, pa.ID)
	sql(t, a, `INSERT INTO miembros (negocio_id, clerk_id, email, nombre, rol) VALUES ($1, $2, 'recep@mail.com', 'Rocío', 'recepcion')`, n.ID, recep)
	reservar := func(prof, cuando, tel string) turnos.Turno {
		var x turnos.Turno
		a.Esperar(201, P{Metodo: "POST", Ruta: "/v1/p/el-corte/turnos", Cuerpo: map[string]any{"servicioId": s.ID, "profesionalId": prof,
			"inicio": en(cuando).Format(time.RFC3339), "cliente": map[string]string{"nombre": "Cliente " + tel, "telefono": tel}}}, &x)
		return x
	}
	t1 := reservar(pa.ID, "2026-11-02 09:00", "111")
	t2 := reservar(pa.ID, "2026-11-02 09:30", "222")
	reservar(pa.ID, "2026-11-09 09:00", "333")
	reservar(pb.ID, "2026-11-02 10:00", "444")

	// Ana ve cuántos turnos tiene esos días antes de pedir.
	var ts struct {
		Cantidad int                        `json:"cantidad"`
		Turnos   []diaslibres.TurnoAfectado `json:"turnos"`
	}
	panel(200, ana, "GET", "/v1/panel/dias-libres/turnos?desde=2026-11-02&hasta=2026-11-04", nil, &ts)
	if ts.Cantidad != 2 || ts.Turnos[0].ID != t1.ID || ts.Turnos[1].Cliente != "Cliente 222" {
		t.Fatalf("turnos en el rango: %+v", ts)
	}
	panel(200, duenia, "GET", "/v1/panel/dias-libres/turnos?desde=2026-11-02&hasta=2026-11-04&profesional="+pb.ID, nil, &ts)
	if ts.Cantidad != 1 {
		t.Fatalf("los de Beto: %+v", ts)
	}

	// Pedir: sólo quien tiene un profesional vinculado, con un motivo válido y desde hoy.
	pedido := map[string]any{"desde": "2026-11-02", "hasta": "2026-11-04", "motivo": "vacaciones", "nota": "Me voy a la costa"}
	panel(409, duenia, "POST", "/v1/panel/dias-libres", pedido, nil)
	panel(400, ana, "POST", "/v1/panel/dias-libres", map[string]any{"desde": "2026-11-02", "motivo": "porque sí"}, nil)
	panel(400, ana, "POST", "/v1/panel/dias-libres", map[string]any{"desde": "2026-10-30", "hasta": "2026-11-02", "motivo": "salud"}, nil)
	var p diaslibres.Pedido
	panel(201, ana, "POST", "/v1/panel/dias-libres", pedido, &p)
	if p.Estado != "pendiente" || p.Turnos != 2 || p.Afectados != 2 || p.Profesional != "Ana" || p.Desde != "2026-11-02" || p.Hasta != "2026-11-04" {
		t.Fatalf("pedido: %+v", p)
	}
	panel(409, ana, "POST", "/v1/panel/dias-libres", map[string]any{"desde": "2026-11-04", "hasta": "2026-11-06", "motivo": "tramite"}, nil)

	// La dueña lo ve en la campanita y en la lista; Ana y recepción no pueden aprobar.
	var c struct {
		Notificaciones []avisos.NotificacionPanel `json:"notificaciones"`
	}
	panel(200, duenia, "GET", "/v1/panel/campanita", nil, &c)
	if c.Notificaciones[0].Titulo != "Ana pide días libres" || c.Notificaciones[0].Tipo != avisos.NotiEquipo ||
		c.Notificaciones[0].Texto != "Del lunes 2 de noviembre al miércoles 4 de noviembre · vacaciones · 2 turnos para mover." {
		t.Fatalf("campanita: %+v", c.Notificaciones[0])
	}
	var l pedidos
	panel(200, duenia, "GET", "/v1/panel/dias-libres?estado=pendiente", nil, &l)
	if len(l.Pedidos) != 1 || l.Pedidos[0].ID != p.ID {
		t.Fatalf("lista: %+v", l)
	}
	panel(403, ana, "POST", "/v1/panel/dias-libres/"+p.ID+"/aprobar", nil, nil)
	panel(403, recep, "POST", "/v1/panel/dias-libres/"+p.ID+"/aprobar", nil, nil)

	// Mientras tanto se libera las 10:30 de Ana y se le ofrece a la lista.
	a.Esperar(201, P{Metodo: "POST", Ruta: "/v1/p/el-corte/espera", Cuerpo: map[string]any{"servicioId": s.ID, "profesionalId": pa.ID,
		"nombre": "Martina", "email": "martina@ejemplo.com", "desde": "2026-11-02"}}, nil)
	var o espera.Oferta
	panel(201, duenia, "POST", "/v1/panel/espera/ofertas", map[string]any{"servicioId": s.ID, "profesionalId": pa.ID,
		"inicio": en("2026-11-02 10:30").Format(time.RFC3339)}, &o)

	// Aprobar: Ana queda bloqueada esos días, sus turnos pasan a la lista para reprogramar y el hueco
	// ofrecido se cae.
	var ap struct {
		Pedido      diaslibres.Pedido        `json:"pedido"`
		Reprogramar []diaslibres.Reprogramar `json:"reprogramar"`
	}
	panel(200, duenia, "POST", "/v1/panel/dias-libres/"+p.ID+"/aprobar", map[string]string{"respuesta": "¡Disfrutá!"}, &ap)
	if ap.Pedido.Estado != "aprobado" || ap.Pedido.Respuesta != "¡Disfrutá!" || ap.Pedido.RespondidoEn == nil || len(ap.Reprogramar) != 2 ||
		ap.Reprogramar[0].Turno.ID != t1.ID || ap.Reprogramar[0].Turno.Cliente.Telefono != "111" || ap.Reprogramar[0].Motivo != "vacaciones" {
		t.Fatalf("aprobar: %+v", ap)
	}
	panel(409, duenia, "POST", "/v1/panel/dias-libres/"+p.ID+"/aprobar", nil, nil)
	var bs struct {
		Bloqueos []disponibilidad.Bloqueo `json:"bloqueos"`
	}
	panel(200, duenia, "GET", "/v1/panel/profesionales/"+pa.ID+"/bloqueos?desde=2026-11-02&hasta=2026-11-04", nil, &bs)
	if len(bs.Bloqueos) != 3 || !strings.Contains(bs.Bloqueos[0].Motivo, "vacaciones") {
		t.Fatalf("bloqueos: %+v", bs)
	}
	var libres struct {
		Horarios []turnos.HorarioLibre `json:"horarios"`
	}
	a.Esperar(200, P{Metodo: "GET", Ruta: "/v1/p/el-corte/libres?servicio=" + s.ID + "&profesional=" + pa.ID + "&fecha=2026-11-02"}, &libres)
	if len(libres.Horarios) != 0 {
		t.Fatalf("Ana sigue libre: %+v", libres)
	}
	panel(200, duenia, "GET", "/v1/panel/espera/ofertas/"+o.ID, nil, &o)
	if o.Estado != "caida" {
		t.Fatalf("la oferta de esos días: %+v", o)
	}

	// Recepción va resolviendo: uno lo pasa a Beto (sale solo de la lista), el otro lo marca a mano.
	var pm paraMover
	panel(200, recep, "GET", "/v1/panel/reprogramar", nil, &pm)
	if len(pm.Turnos) != 2 {
		t.Fatalf("para mover: %+v", pm)
	}
	panel(403, ana, "GET", "/v1/panel/reprogramar", nil, nil)
	panel(200, recep, "POST", "/v1/panel/turnos/"+t1.ID+"/reprogramar", map[string]any{"inicio": en("2026-11-02 09:00").Format(time.RFC3339),
		"profesionalId": pb.ID}, nil)
	panel(200, recep, "GET", "/v1/panel/reprogramar", nil, &pm)
	if len(pm.Turnos) != 1 || pm.Turnos[0].Turno.ID != t2.ID {
		t.Fatalf("después de mover uno: %+v", pm)
	}
	var r diaslibres.Reprogramar
	panel(200, recep, "POST", "/v1/panel/reprogramar/"+pm.Turnos[0].ID+"/resolver", nil, &r)
	if r.ResueltoEn == nil || r.ResueltoPor == nil || *r.ResueltoPor != recep {
		t.Fatalf("resolver: %+v", r)
	}
	panel(200, recep, "GET", "/v1/panel/reprogramar", nil, &pm)
	if len(pm.Turnos) != 0 {
		t.Fatalf("quedan: %+v", pm)
	}
	panel(200, recep, "GET", "/v1/panel/reprogramar?todos=1&pedido="+p.ID, nil, &pm)
	if len(pm.Turnos) != 2 || *pm.Turnos[0].ResueltoPor != "automatico" {
		t.Fatalf("todos: %+v", pm)
	}
	panel(404, recep, "POST", "/v1/panel/reprogramar/00000000-0000-0000-0000-000000000000/resolver", nil, nil)

	// Rechazar y retirar.
	var p2, p3 diaslibres.Pedido
	panel(201, ana, "POST", "/v1/panel/dias-libres", map[string]any{"desde": "2026-11-09", "motivo": "tramite"}, &p2)
	if p2.Turnos != 1 || p2.Hasta != "2026-11-09" {
		t.Fatalf("p2: %+v", p2)
	}
	panel(200, duenia, "POST", "/v1/panel/dias-libres/"+p2.ID+"/rechazar", map[string]string{"respuesta": "Ese día no"}, &p2)
	if p2.Estado != "rechazado" {
		t.Fatalf("rechazar: %+v", p2)
	}
	panel(200, ana, "GET", "/v1/panel/campanita", nil, &c)
	if c.Notificaciones[0].Titulo != "No se aprobaron los días libres de Ana" {
		t.Fatalf("campanita de Ana: %+v", c.Notificaciones[0])
	}
	panel(201, ana, "POST", "/v1/panel/dias-libres", map[string]any{"desde": "2026-11-16", "motivo": "salud"}, &p3)
	panel(403, recep, "POST", "/v1/panel/dias-libres/"+p3.ID+"/cancelar", nil, nil)
	panel(200, ana, "POST", "/v1/panel/dias-libres/"+p3.ID+"/cancelar", nil, &p3)
	if p3.Estado != "cancelado" {
		t.Fatalf("cancelar: %+v", p3)
	}
	var ver struct {
		diaslibres.Pedido
		TurnosAfectados []diaslibres.TurnoAfectado `json:"turnosAfectados"`
	}
	panel(200, ana, "GET", "/v1/panel/dias-libres/"+p2.ID, nil, &ver)
	if ver.Estado != "rechazado" || len(ver.TurnosAfectados) != 1 {
		t.Fatalf("ver: %+v", ver)
	}
	panel(200, ana, "GET", "/v1/panel/dias-libres", nil, &l)
	if len(l.Pedidos) != 3 {
		t.Fatalf("los de Ana: %+v", l)
	}

	// Otro negocio no los ve.
	const otra = "user_otra000001"
	var o2 negocios.Negocio
	a.Esperar(201, P{Metodo: "POST", Ruta: "/v1/negocios", Quien: otra, Cuerpo: map[string]string{"slug": "otro", "nombre": "Otro"}}, &o2)
	a.Esperar(200, P{Metodo: "GET", Ruta: "/v1/panel/dias-libres", Quien: otra, Negocio: o2.ID}, &l)
	if len(l.Pedidos) != 0 {
		t.Fatal("otro negocio ve pedidos ajenos")
	}
	a.Esperar(404, P{Metodo: "POST", Ruta: "/v1/panel/dias-libres/" + p.ID + "/rechazar", Quien: otra, Negocio: o2.ID}, nil)
}
