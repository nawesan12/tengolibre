package calendario_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/calendario"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/catalogo"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/disponibilidad"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/negocios"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/turnos"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/prueba"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/servidor"
)

type P = prueba.Pedido

const (
	duenia = "user_duenia0001"
	ana    = "user_ana00000001"
)

func en(s string) time.Time {
	t, err := time.ParseInLocation("2006-01-02 15:04", s, disponibilidad.ZonaPorDefecto)
	if err != nil {
		panic(err)
	}
	return t
}

type lista struct {
	Calendarios []calendario.Calendario `json:"calendarios"`
}

func TestCalendarioParaSuscribirse(t *testing.T) {
	a := prueba.Nueva(t, en("2026-11-01 10:00"), func(a *prueba.API) []servidor.Dominio {
		app := a.Bases.App
		return []servidor.Dominio{
			&negocios.Servicio{Pool: app, Sesion: a.Sesion},
			&catalogo.Servicio{Pool: app, Sesion: a.Sesion},
			&turnos.Servicio{Pool: app, Sesion: a.Sesion, Ahora: a.Reloj()},
			&calendario.Servicio{Pool: app, Sesion: a.Sesion, Sitio: "https://tengolibre.com.ar", Ahora: a.Reloj()},
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
	reservar := func(prof, hora, tel, nota string) turnos.Turno {
		var x turnos.Turno
		a.Esperar(201, P{Metodo: "POST", Ruta: "/v1/p/el-corte/turnos", Cuerpo: map[string]any{"servicioId": s.ID, "profesionalId": prof,
			"inicio": en("2026-11-02 " + hora).Format(time.RFC3339), "nota": nota,
			"cliente": map[string]string{"nombre": "María, la de siempre", "telefono": tel}}}, &x)
		return x
	}
	t1 := reservar(pa.ID, "09:00", "111111", "Trae foto")
	reservar(pb.ID, "09:00", "222222", "")
	t3 := reservar(pa.ID, "10:00", "333333", "")
	panel(200, duenia, "PATCH", "/v1/panel/turnos/"+t3.ID, map[string]string{"estado": "cancelado"}, nil)

	// La dueña ve el del negocio y uno por profesional, todavía sin crear.
	var l lista
	panel(200, duenia, "GET", "/v1/panel/calendarios", nil, &l)
	if len(l.Calendarios) != 3 || l.Calendarios[0].ProfesionalID != nil || l.Calendarios[0].URL != nil || l.Calendarios[1].Nombre != "Ana" {
		t.Fatalf("lista: %+v", l.Calendarios)
	}
	var todo, deAna calendario.Calendario
	panel(201, duenia, "POST", "/v1/panel/calendarios", map[string]string{}, &todo)
	panel(201, duenia, "POST", "/v1/panel/calendarios", map[string]string{"profesionalId": pa.ID}, &deAna)
	if todo.URL == nil || !strings.HasPrefix(*todo.URL, "https://tengolibre.com.ar/cal/") || !strings.HasSuffix(*todo.URL, ".ics") {
		t.Fatalf("url: %+v", todo)
	}
	ruta := func(c calendario.Calendario) string { return "/v1" + strings.TrimPrefix(*c.URL, "https://tengolibre.com.ar") }

	ics := string(a.Esperar(200, P{Metodo: "GET", Ruta: ruta(todo)}, nil).Cuerpo)
	if strings.Count(ics, "BEGIN:VEVENT") != 2 || !strings.Contains(ics, "X-WR-CALNAME:El Corte") ||
		!strings.Contains(ics, `SUMMARY:Corte · María\, la de siempre`) || !strings.Contains(ics, "DTSTART:20261102T120000Z") ||
		!strings.Contains(ics, "Con Ana") || !strings.Contains(ics, "UID:"+t1.ID+"@tengolibre.com.ar") {
		t.Fatalf("el del negocio:\n%s", ics)
	}
	ics = string(a.Esperar(200, P{Metodo: "GET", Ruta: ruta(deAna)}, nil).Cuerpo)
	if strings.Count(ics, "BEGIN:VEVENT") != 1 || !strings.Contains(ics, "X-WR-CALNAME:El Corte · Ana") || !strings.Contains(ics, "Trae foto") {
		t.Fatalf("el de Ana:\n%s", ics)
	}

	// Regenerar invalida el link viejo; un token inventado da 404.
	var nuevo calendario.Calendario
	panel(201, duenia, "POST", "/v1/panel/calendarios", map[string]string{"profesionalId": pa.ID}, &nuevo)
	a.Esperar(404, P{Metodo: "GET", Ruta: ruta(deAna)}, nil)
	a.Esperar(200, P{Metodo: "GET", Ruta: ruta(nuevo)}, nil)
	a.Esperar(404, P{Metodo: "GET", Ruta: "/v1/cal/inventado0000000000000000000000000.ics"}, nil)

	// Ana entra como profesional: ve y maneja sólo el suyo.
	sql(t, a, `WITH m AS (INSERT INTO miembros (negocio_id, clerk_id, email, nombre, rol) VALUES ($1, $2, 'ana@mail.com', 'Ana', 'profesional') RETURNING id)
		UPDATE profesionales SET miembro_id = (SELECT id FROM m) WHERE id = $3`, n.ID, ana, pa.ID)
	panel(200, ana, "GET", "/v1/panel/calendarios", nil, &l)
	if len(l.Calendarios) != 1 || l.Calendarios[0].Nombre != "Ana" || l.Calendarios[0].URL == nil {
		t.Fatalf("lista de Ana: %+v", l.Calendarios)
	}
	panel(403, ana, "POST", "/v1/panel/calendarios", map[string]string{}, nil)
	panel(403, ana, "POST", "/v1/panel/calendarios", map[string]string{"profesionalId": pb.ID}, nil)
	panel(403, ana, "DELETE", "/v1/panel/calendarios/"+*todo.ID, nil, nil)
	panel(204, ana, "DELETE", "/v1/panel/calendarios/"+*nuevo.ID, nil, nil)
	a.Esperar(404, P{Metodo: "GET", Ruta: ruta(nuevo)}, nil)

	// Otro negocio no ve ni borra los de este.
	const otra = "user_otra000001"
	var o negocios.Negocio
	a.Esperar(201, P{Metodo: "POST", Ruta: "/v1/negocios", Quien: otra, Cuerpo: map[string]string{"slug": "otro", "nombre": "Otro"}}, &o)
	a.Esperar(404, P{Metodo: "DELETE", Ruta: "/v1/panel/calendarios/" + *todo.ID, Quien: otra, Negocio: o.ID}, nil)
	a.Esperar(200, P{Metodo: "GET", Ruta: ruta(todo)}, nil)
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

