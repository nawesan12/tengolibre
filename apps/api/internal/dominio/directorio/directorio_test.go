package directorio_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/catalogo"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/directorio"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/disponibilidad"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/miweb"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/negocios"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/prueba"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/servidor"
)

type P = prueba.Pedido

var z = disponibilidad.ZonaPorDefecto

func en(s string) time.Time {
	t, err := time.ParseInLocation("2006-01-02 15:04", s, z)
	if err != nil {
		panic(err)
	}
	return t
}

// negocio da de alta un negocio con un servicio y, si reservable, un profesional con horario que
// lo hace. Devuelve el id del negocio, del servicio y del profesional.
func negocio(t *testing.T, a *prueba.API, quien, slug, ciudad, rubro, servicio string, reservable bool) (string, string, string) {
	t.Helper()
	var n negocios.Negocio
	a.Esperar(201, P{Metodo: "POST", Ruta: "/v1/negocios", Quien: quien, Cuerpo: map[string]string{"slug": slug, "nombre": "Negocio " + slug}}, &n)
	panel := func(estado int, metodo, ruta string, cuerpo, v any) {
		a.Esperar(estado, P{Metodo: metodo, Ruta: ruta, Quien: quien, Negocio: n.ID, Cuerpo: cuerpo}, v)
	}
	panel(200, "PATCH", "/v1/panel/negocio", map[string]string{"ciudad": ciudad}, nil)
	panel(200, "PATCH", "/v1/panel/mi-web", map[string]string{"rubro": rubro, "provincia": "Buenos Aires"}, nil)
	var c catalogo.Categoria
	panel(201, "POST", "/v1/panel/categorias", map[string]string{"nombre": "Barba y bigote"}, &c)
	var s catalogo.Prestacion
	panel(201, "POST", "/v1/panel/servicios", map[string]any{"nombre": servicio, "duracion": 30, "precio": 9000, "categoriaId": c.ID}, &s)
	panel(201, "POST", "/v1/panel/servicios", map[string]any{"nombre": "Otro", "duracion": 30, "precio": 5000}, nil)
	if !reservable {
		return n.ID, s.ID, ""
	}
	var p catalogo.Profesional
	panel(201, "POST", "/v1/panel/profesionales", map[string]any{"nombre": "Ana", "servicios": []string{s.ID},
		"horario": []disponibilidad.Franja{{Dia: 2, Desde: "09:00", Hasta: "20:00"}}}, &p)
	return n.ID, s.ID, p.ID
}

func armar(t *testing.T) *prueba.API {
	return prueba.Nueva(t, en("2026-11-01 10:00"), func(a *prueba.API) []servidor.Dominio {
		app := a.Bases.App
		return []servidor.Dominio{
			&negocios.Servicio{Pool: app, Sesion: a.Sesion},
			&catalogo.Servicio{Pool: app, Sesion: a.Sesion},
			&miweb.Servicio{Pool: app, Sesion: a.Sesion},
			&directorio.Servicio{Pool: app, Sesion: a.Sesion, Ahora: a.Reloj()},
		}
	})
}

func admin(t *testing.T, a *prueba.API, sql string, args ...any) {
	t.Helper()
	conn, err := pgx.Connect(context.Background(), a.Bases.AdminURL)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close(context.Background())
	if _, err := conn.Exec(context.Background(), sql, args...); err != nil {
		t.Fatal(err)
	}
}

func TestExplorarSoloMuestraActivosReservables(t *testing.T) {
	a := armar(t)
	negocio(t, a, "user_aaaaaaaaaa", "la-barberia", "Mar del Plata", "barberia", "Corte y barba", true)
	negocio(t, a, "user_bbbbbbbbbb", "sin-horario", "Mar del Plata", "barberia", "Corte y barba", false)
	apagado, _, _ := negocio(t, a, "user_cccccccccc", "apagado", "Mar del Plata", "barberia", "Corte y barba", true)
	negocio(t, a, "user_dddddddddd", "el-salon", "Tandil", "peluqueria", "Color", true)
	admin(t, a, "UPDATE negocios SET activo = false WHERE id = $1", apagado)

	buscar := func(q string) directorio.Pagina {
		t.Helper()
		var p directorio.Pagina
		a.Esperar(200, P{Metodo: "GET", Ruta: "/v1/explorar" + q}, &p)
		return p
	}
	p := buscar("")
	if p.Total != 2 || len(p.Negocios) != 2 || p.Negocios[0].Slug != "el-salon" || p.Negocios[1].Slug != "la-barberia" {
		t.Fatalf("todos: %+v", p)
	}
	f := p.Negocios[1] // por nombre: "Negocio el-salon" va primero
	if f.Ciudad != "Mar del Plata" || f.Rubro != "barberia" || f.Desde == nil || *f.Desde != 5000 || len(f.Servicios) != 2 {
		t.Fatalf("ficha: %+v", f)
	}
	for q, quiero := range map[string]string{
		"?ciudad=mar+del":         "la-barberia",
		"?ciudad=TANDIL":          "el-salon",
		"?rubro=peluqueria":       "el-salon",
		"?q=barba":                "la-barberia", // por el servicio
		"?q=negocio+el":           "el-salon",    // por el nombre
		"?categoria=bigote&q=%25": "",            // el % no es comodín
		"?categoria=bigote":       "el-salon,la-barberia",
		"?provincia=buenos":       "el-salon,la-barberia",
	} {
		p := buscar(q)
		var slugs string
		for i, n := range p.Negocios {
			if i > 0 {
				slugs += ","
			}
			slugs += n.Slug
		}
		if slugs != quiero {
			t.Errorf("%s: %q, quería %q", q, slugs, quiero)
		}
	}
	// Paginado.
	p = buscar("?limite=1&pagina=2")
	if p.Total != 2 || p.Paginas != 2 || len(p.Negocios) != 1 || p.Negocios[0].Slug != "la-barberia" {
		t.Fatalf("página 2: %+v", p)
	}
	if p := buscar("?limite=1&pagina=9"); p.Total != 2 || len(p.Negocios) != 0 {
		t.Fatalf("página de más: %+v", p)
	}

	var cs struct {
		Ciudades []directorio.Ciudad `json:"ciudades"`
	}
	a.Esperar(200, P{Metodo: "GET", Ruta: "/v1/explorar/ciudades"}, &cs)
	if len(cs.Ciudades) != 2 || cs.Ciudades[0].Cantidad != 1 {
		t.Fatalf("ciudades: %+v", cs.Ciudades)
	}
}

func TestSugerir(t *testing.T) {
	ahora := en("2026-11-01 10:00") // domingo
	var ps []directorio.Pasado
	// Cuatro martes a las 18 (uno a las 17:50, que redondea a las 18) con Ana, uno un jueves.
	for _, h := range []string{"2026-10-27 18:00", "2026-10-20 17:50", "2026-10-13 18:00", "2026-10-06 18:10"} {
		ps = append(ps, directorio.Pasado{Inicio: en(h), ProfesionalID: "ana", Profesional: "Ana", ServicioID: "corte"})
	}
	ps = append(ps, directorio.Pasado{Inicio: en("2026-10-01 11:00"), ProfesionalID: "beto", Profesional: "Beto", ServicioID: "barba"})
	ss := directorio.Sugerir(ps, "", 30, ahora, z)
	if len(ss) != 2 {
		t.Fatalf("sugerencias: %+v", ss)
	}
	if s := ss[0]; s.Dia != 2 || s.Hora != "18:00" || s.ProfesionalID != "ana" || s.Confianza != 0.8 || s.Proxima != "2026-11-03" ||
		s.Motivo != "Suele reservar los martes a las 18:00 con Ana" {
		t.Fatalf("primera: %+v", s)
	}
	if ss[1].Dia != 4 || ss[1].Hora != "11:00" {
		t.Fatalf("segunda: %+v", ss[1])
	}
	// Con un solo turno no hay patrón.
	if ss := directorio.Sugerir(ps[:1], "", 30, ahora, z); len(ss) != 0 {
		t.Fatal("un turno no es un patrón")
	}
	// Redondeo con arrastre: 23:50 en grilla de 30 es la medianoche del día siguiente (no "23:60").
	tarde := []directorio.Pasado{{Inicio: en("2026-10-27 23:50"), ProfesionalID: "ana"}, {Inicio: en("2026-10-20 23:55"), ProfesionalID: "ana"}}
	if ss := directorio.Sugerir(tarde, "", 30, ahora, z); len(ss) != 1 || ss[0].Hora != "00:00" || ss[0].Dia != 3 {
		t.Fatalf("arrastre: %+v", ss)
	}
	// Hoy, si la hora todavía no pasó, es la próxima.
	hoy := []directorio.Pasado{{Inicio: en("2026-10-25 18:00")}, {Inicio: en("2026-10-18 18:00")}}
	if ss := directorio.Sugerir(hoy, "", 30, ahora, z); ss[0].Proxima != "2026-11-01" {
		t.Fatalf("hoy: %+v", ss)
	}
}

func TestSugerenciasDelCliente(t *testing.T) {
	a := armar(t)
	n, s, p := negocio(t, a, "user_aaaaaaaaaa", "la-barberia", "Mar del Plata", "barberia", "Corte", true)
	cliente := "00000000-0000-0000-0000-00000000c001"
	admin(t, a, "INSERT INTO clientes (id, negocio_id, nombre, telefono) VALUES ($1, $2, 'Caro', '2231')", cliente, n)
	for i, h := range []string{"2026-10-27 18:00", "2026-10-20 18:00", "2026-10-13 18:00"} {
		admin(t, a, `INSERT INTO turnos (negocio_id, codigo, servicio_id, servicio, profesional_id, profesional, cliente_id, inicio, fin, precio, estado, origen)
			VALUES ($1, $2, $3, 'Corte', $4, 'Ana', $5, $6, $6::timestamptz + interval '30 minutes', 9000, 'atendido', 'web')`,
			n, fmt.Sprintf("codigo-del-cliente-%04d", i), s, p, cliente, en(h))
	}
	var r struct {
		Sugerencias []directorio.Sugerencia `json:"sugerencias"`
	}
	a.Esperar(200, P{Metodo: "GET", Ruta: "/v1/t/codigo-del-cliente-0000/sugerencias"}, &r)
	if len(r.Sugerencias) != 1 || r.Sugerencias[0].Hora != "18:00" || r.Sugerencias[0].ProfesionalID != p || r.Sugerencias[0].Proxima != "2026-11-03" {
		t.Fatalf("cliente: %+v", r.Sugerencias)
	}
	a.Esperar(404, P{Metodo: "GET", Ruta: "/v1/t/no-existe-este-codigo/sugerencias"}, nil)
	a.Esperar(200, P{Metodo: "GET", Ruta: "/v1/panel/clientes/" + cliente + "/sugerencias", Quien: "user_aaaaaaaaaa", Negocio: n}, &r)
	if len(r.Sugerencias) != 1 {
		t.Fatalf("panel: %+v", r.Sugerencias)
	}
	// Otro negocio no ve al cliente.
	otro, _, _ := negocio(t, a, "user_bbbbbbbbbb", "otro", "Tandil", "barberia", "Corte", false)
	a.Esperar(404, P{Metodo: "GET", Ruta: "/v1/panel/clientes/" + cliente + "/sugerencias", Quien: "user_bbbbbbbbbb", Negocio: otro}, nil)
}
