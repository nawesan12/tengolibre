package grupos_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/disponibilidad"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/grupos"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/negocios"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/prueba"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/servidor"
)

type P = prueba.Pedido

const ana, beto = "user_anaduenia01", "user_betoduenio1"

type lista struct {
	Grupos   []grupos.Grupo    `json:"grupos"`
	SinGrupo []grupos.Sucursal `json:"sinGrupo"`
}

func TestSucursalesYReporte(t *testing.T) {
	ahora, _ := time.ParseInLocation("2006-01-02 15:04", "2026-11-01 10:00", disponibilidad.ZonaPorDefecto)
	a := prueba.Nueva(t, ahora, func(a *prueba.API) []servidor.Dominio {
		return []servidor.Dominio{
			&negocios.Servicio{Pool: a.Bases.App, Sesion: a.Sesion},
			&grupos.Servicio{Pool: a.Bases.App, Sesion: a.Sesion, Ahora: a.Reloj()},
		}
	})
	alta := func(quien, slug string) string {
		var n negocios.Negocio
		a.Esperar(201, P{Metodo: "POST", Ruta: "/v1/negocios", Quien: quien, Cuerpo: map[string]string{"slug": slug, "nombre": "Sucursal " + slug}}, &n)
		return n.ID
	}
	centro, norte, ajeno := alta(ana, "centro"), alta(ana, "norte"), alta(beto, "ajeno")

	var l lista
	a.Esperar(200, P{Metodo: "GET", Ruta: "/v1/grupos", Quien: ana}, &l)
	if len(l.Grupos) != 0 || len(l.SinGrupo) != 2 {
		t.Fatalf("al principio: %+v", l)
	}
	var g grupos.Grupo
	a.Esperar(201, P{Metodo: "POST", Ruta: "/v1/grupos", Quien: ana, Cuerpo: map[string]string{"nombre": "Cortes SRL"}}, &g)
	a.Esperar(200, P{Metodo: "PUT", Ruta: "/v1/grupos/" + g.ID + "/sucursales/" + centro, Quien: ana}, &g)
	a.Esperar(200, P{Metodo: "PUT", Ruta: "/v1/grupos/" + g.ID + "/sucursales/" + norte, Quien: ana}, &g)
	if len(g.Sucursales) != 2 {
		t.Fatalf("grupo: %+v", g)
	}
	// Un negocio ajeno no se suma; y el otro dueño ni ve el grupo ni le puede sumar el suyo.
	a.Esperar(403, P{Metodo: "PUT", Ruta: "/v1/grupos/" + g.ID + "/sucursales/" + ajeno, Quien: ana}, nil)
	a.Esperar(404, P{Metodo: "PUT", Ruta: "/v1/grupos/" + g.ID + "/sucursales/" + ajeno, Quien: beto}, nil)
	a.Esperar(404, P{Metodo: "GET", Ruta: "/v1/grupos/" + g.ID + "/reporte", Quien: beto}, nil)
	a.Esperar(404, P{Metodo: "PATCH", Ruta: "/v1/grupos/" + g.ID, Quien: beto, Cuerpo: map[string]string{"nombre": "Mío"}}, nil)
	a.Esperar(404, P{Metodo: "DELETE", Ruta: "/v1/grupos/" + g.ID, Quien: beto}, nil)
	a.Esperar(200, P{Metodo: "GET", Ruta: "/v1/grupos", Quien: beto}, &l)
	if len(l.Grupos) != 0 || len(l.SinGrupo) != 1 {
		t.Fatalf("beto ve: %+v", l)
	}
	a.Esperar(401, P{Metodo: "GET", Ruta: "/v1/grupos"}, nil)

	// Turnos: centro 2 atendidos (8000 + 6000) y un cancelado; norte uno atendido y un ausente;
	// uno de centro fuera del período.
	conn, err := pgx.Connect(context.Background(), a.Bases.AdminURL)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close(context.Background())
	i := 0
	turno := func(negocio, estado, cuando string, precio int) {
		t.Helper()
		i++
		cliente := fmt.Sprintf("00000000-0000-0000-0000-%012d", i)
		inicio, _ := time.ParseInLocation("2006-01-02 15:04", cuando, disponibilidad.ZonaPorDefecto)
		if _, err := conn.Exec(context.Background(), "INSERT INTO clientes (id, negocio_id, nombre) VALUES ($1, $2, 'Cliente')", cliente, negocio); err != nil {
			t.Fatal(err)
		}
		if _, err := conn.Exec(context.Background(), `INSERT INTO turnos (negocio_id, codigo, servicio, profesional, cliente_id, inicio, fin, precio, estado, origen)
			VALUES ($1, $2, 'Corte', 'Ana', $3, $4, $4::timestamptz + interval '30 minutes', $5, $6, 'panel')`,
			negocio, fmt.Sprintf("codigo-de-prueba-%04d", i), cliente, inicio, precio, estado); err != nil {
			t.Fatal(err)
		}
	}
	turno(centro, "atendido", "2026-10-20 10:00", 8000)
	turno(centro, "atendido", "2026-10-21 10:00", 6000)
	turno(centro, "cancelado", "2026-10-22 10:00", 8000)
	turno(centro, "atendido", "2026-09-01 10:00", 99999)
	turno(norte, "atendido", "2026-10-25 10:00", 7000)
	turno(norte, "ausente", "2026-10-26 10:00", 7000)
	turno(ajeno, "atendido", "2026-10-26 10:00", 1)

	var rep struct {
		Grupo      string         `json:"grupo"`
		Desde      string         `json:"desde"`
		Total      grupos.Numeros `json:"total"`
		Sucursales []struct {
			grupos.Sucursal
			grupos.Numeros
		} `json:"sucursales"`
	}
	a.Esperar(200, P{Metodo: "GET", Ruta: "/v1/grupos/" + g.ID + "/reporte", Quien: ana}, &rep)
	if rep.Desde != "2026-10-03" || rep.Total.Ingresos != 21000 || rep.Total.Turnos != 4 || rep.Total.Atendidos != 3 ||
		rep.Total.Ausentes != 1 || rep.Total.Cancelados != 1 || len(rep.Sucursales) != 2 {
		t.Fatalf("reporte: %+v", rep)
	}
	for _, s := range rep.Sucursales {
		if (s.ID == centro && s.Ingresos != 14000) || (s.ID == norte && s.Ingresos != 7000) {
			t.Fatalf("por sucursal: %+v", rep.Sucursales)
		}
	}
	a.Esperar(200, P{Metodo: "GET", Ruta: "/v1/grupos/" + g.ID + "/reporte?desde=2026-09-01&hasta=2026-09-01", Quien: ana}, &rep)
	if rep.Total.Ingresos != 99999 {
		t.Fatalf("septiembre: %+v", rep.Total)
	}
	a.Esperar(400, P{Metodo: "GET", Ruta: "/v1/grupos/" + g.ID + "/reporte?desde=2026-10-02&hasta=2026-10-01", Quien: ana}, nil)

	// Sacar una sucursal y borrar el grupo: los negocios quedan, sueltos.
	a.Esperar(200, P{Metodo: "DELETE", Ruta: "/v1/grupos/" + g.ID + "/sucursales/" + norte, Quien: ana}, &g)
	if len(g.Sucursales) != 1 || g.Sucursales[0].ID != centro {
		t.Fatalf("sin norte: %+v", g)
	}
	a.Esperar(200, P{Metodo: "PATCH", Ruta: "/v1/grupos/" + g.ID, Quien: ana, Cuerpo: map[string]string{"nombre": "Cortes SA"}}, &g)
	if g.Nombre != "Cortes SA" {
		t.Fatal("no se renombró")
	}
	a.Esperar(204, P{Metodo: "DELETE", Ruta: "/v1/grupos/" + g.ID, Quien: ana}, nil)
	a.Esperar(200, P{Metodo: "GET", Ruta: "/v1/grupos", Quien: ana}, &l)
	if len(l.Grupos) != 0 || len(l.SinGrupo) != 2 {
		t.Fatalf("después de borrar: %+v", l)
	}
}
