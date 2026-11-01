package admin_test

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/admin"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/arranque"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/catalogo"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/directorio"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/disponibilidad"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/negocios"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/turnos"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/prueba"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/servidor"
)

type P = prueba.Pedido

const (
	equipo = "user_equipotengolibre"
	ana    = "user_anaduenia01"
	beto   = "user_betoduenio1"
)

func armar(t *testing.T) *prueba.API {
	ahora, _ := time.ParseInLocation("2006-01-02 15:04", "2026-11-01 10:00", disponibilidad.ZonaPorDefecto)
	return prueba.Nueva(t, ahora, func(a *prueba.API) []servidor.Dominio {
		app := a.Bases.App
		return []servidor.Dominio{
			&negocios.Servicio{Pool: app, Sesion: a.Sesion},
			&catalogo.Servicio{Pool: app, Sesion: a.Sesion},
			&turnos.Servicio{Pool: app, Sesion: a.Sesion, Ahora: a.Reloj()},
			&directorio.Servicio{Pool: app, Sesion: a.Sesion, Ahora: a.Reloj()},
			&arranque.Servicio{Pool: app, Sesion: a.Sesion},
			&admin.Servicio{Pool: app, Sesion: a.Sesion, Admins: []string{equipo}, AdminsEmails: []string{"Socia0001@Prueba.TengoLibre.com.ar"}, Ahora: a.Reloj()},
		}
	})
}

func alta(a *prueba.API, quien, slug string) string {
	var n negocios.Negocio
	a.Esperar(201, P{Metodo: "POST", Ruta: "/v1/negocios", Quien: quien, Cuerpo: map[string]string{"slug": slug, "nombre": "Negocio " + slug}}, &n)
	return n.ID
}

// reservable le carga un servicio y un profesional con horario.
func reservable(a *prueba.API, quien, negocio string) {
	var s catalogo.Prestacion
	a.Esperar(201, P{Metodo: "POST", Ruta: "/v1/panel/servicios", Quien: quien, Negocio: negocio,
		Cuerpo: map[string]any{"nombre": "Corte", "duracion": 30, "precio": 8000}}, &s)
	a.Esperar(201, P{Metodo: "POST", Ruta: "/v1/panel/profesionales", Quien: quien, Negocio: negocio, Cuerpo: map[string]any{
		"nombre": "Ana", "servicios": []string{s.ID}, "horario": []disponibilidad.Franja{{Dia: 1, Desde: "09:00", Hasta: "11:00"}}}}, nil)
}

func TestAdminSoloParaElEquipo(t *testing.T) {
	a := armar(t)
	uno := alta(a, ana, "uno")
	alta(a, beto, "dos")
	for _, r := range []P{
		{Metodo: "GET", Ruta: "/v1/admin/estadisticas"},
		{Metodo: "GET", Ruta: "/v1/admin/negocios"},
		{Metodo: "PATCH", Ruta: "/v1/admin/negocios/" + uno, Cuerpo: map[string]bool{"activo": false}},
		{Metodo: "POST", Ruta: "/v1/admin/negocios/" + uno + "/extender-prueba", Cuerpo: map[string]int{"dias": 30}},
	} {
		a.Esperar(401, r, nil)
		// La dueña de un negocio tampoco: no es del equipo.
		r.Quien, r.Negocio = ana, uno
		a.Esperar(404, r, nil)
	}
}

func TestAdmin(t *testing.T) {
	a := armar(t)
	uno, dos := alta(a, ana, "uno"), alta(a, beto, "dos")
	reservable(a, ana, uno)

	var est map[string]any
	a.Esperar(200, P{Metodo: "GET", Ruta: "/v1/admin/estadisticas", Quien: equipo}, &est)
	if est["negocios"] != 2.0 || est["activos"] != 2.0 || est["personas"] != 2.0 || est["enPrueba"] != 2.0 || len(est["recientes"].([]any)) != 2 {
		t.Fatalf("estadísticas: %v", est)
	}

	var lista struct {
		Total    int              `json:"total"`
		Negocios []map[string]any `json:"negocios"`
	}
	a.Esperar(200, P{Metodo: "GET", Ruta: "/v1/admin/negocios", Quien: equipo}, &lista)
	if lista.Total != 2 || len(lista.Negocios) != 2 {
		t.Fatalf("negocios: %+v", lista)
	}
	// Por email de la dueña (el falso de sesión arma <id>@prueba.tengolibre.com.ar).
	a.Esperar(200, P{Metodo: "GET", Ruta: "/v1/admin/negocios?q=betoduenio", Quien: equipo}, &lista)
	if lista.Total != 1 || lista.Negocios[0]["slug"] != "dos" || lista.Negocios[0]["reservable"] != false {
		t.Fatalf("búsqueda: %+v", lista)
	}

	// Desactivar: la página pública y el directorio lo dejan de mostrar, y el panel también.
	a.Esperar(200, P{Metodo: "GET", Ruta: "/v1/p/uno"}, nil)
	var pag directorio.Pagina
	a.Esperar(200, P{Metodo: "GET", Ruta: "/v1/explorar"}, &pag)
	if pag.Total != 1 {
		t.Fatalf("explorar antes: %+v", pag)
	}
	a.Esperar(200, P{Metodo: "PATCH", Ruta: "/v1/admin/negocios/" + uno, Quien: equipo, Cuerpo: map[string]bool{"activo": false}}, nil)
	a.Esperar(404, P{Metodo: "GET", Ruta: "/v1/p/uno"}, nil)
	a.Esperar(200, P{Metodo: "GET", Ruta: "/v1/explorar"}, &pag)
	if pag.Total != 0 {
		t.Fatalf("explorar después: %+v", pag)
	}
	a.Esperar(403, P{Metodo: "GET", Ruta: "/v1/panel/negocio", Quien: ana, Negocio: uno}, nil)
	a.Esperar(200, P{Metodo: "PATCH", Ruta: "/v1/admin/negocios/" + uno, Quien: equipo, Cuerpo: map[string]bool{"activo": true}}, nil)
	a.Esperar(200, P{Metodo: "GET", Ruta: "/v1/p/uno"}, nil)
	a.Esperar(404, P{Metodo: "PATCH", Ruta: "/v1/admin/negocios/00000000-0000-0000-0000-000000000000", Quien: equipo, Cuerpo: map[string]bool{"activo": true}}, nil)
	a.Esperar(400, P{Metodo: "PATCH", Ruta: "/v1/admin/negocios/" + uno, Quien: equipo, Cuerpo: map[string]any{}}, nil)

	// Extender la prueba: suma desde el fin (si todavía no venció) o desde hoy.
	var ext struct {
		PruebaHasta time.Time `json:"pruebaHasta"`
	}
	a.Esperar(200, P{Metodo: "POST", Ruta: "/v1/admin/negocios/" + dos + "/extender-prueba", Quien: equipo, Cuerpo: map[string]int{"dias": 30}}, &ext)
	primera := ext.PruebaHasta
	if !primera.After(a.Ahora.AddDate(0, 0, 30)) {
		t.Fatalf("la prueba vencía en 7 días reales: +30 tiene que pasar de hoy+30: %v", primera)
	}
	a.Esperar(200, P{Metodo: "POST", Ruta: "/v1/admin/negocios/" + dos + "/extender-prueba", Quien: equipo, Cuerpo: map[string]int{"dias": 10}}, &ext)
	if !ext.PruebaHasta.Equal(primera.AddDate(0, 0, 10)) {
		t.Fatalf("segunda extensión: %v, quería %v", ext.PruebaHasta, primera.AddDate(0, 0, 10))
	}
	a.Esperar(400, P{Metodo: "POST", Ruta: "/v1/admin/negocios/" + dos + "/extender-prueba", Quien: equipo, Cuerpo: map[string]int{"dias": 0}}, nil)
}

func TestArranque(t *testing.T) {
	a := armar(t)
	uno := alta(a, ana, "uno")
	var e arranque.Estado
	a.Esperar(200, P{Metodo: "GET", Ruta: "/v1/panel/arranque", Quien: ana, Negocio: uno}, &e)
	if e.Completo || e.Reservable || len(e.Pasos) == 0 {
		t.Fatalf("recién creado: %+v", e)
	}
	reservable(a, ana, uno)
	a.Esperar(200, P{Metodo: "GET", Ruta: "/v1/panel/arranque", Quien: ana, Negocio: uno}, &e)
	if e.Completo || !e.Reservable {
		t.Fatalf("con catálogo: %+v", e)
	}
	a.Esperar(200, P{Metodo: "PATCH", Ruta: "/v1/panel/negocio", Quien: ana, Negocio: uno, Cuerpo: map[string]string{
		"descripcion": "Barbería de barrio", "whatsapp": "2235000000", "direccion": "Güemes 1234", "ciudad": "Mar del Plata"}}, nil)
	a.Esperar(200, P{Metodo: "GET", Ruta: "/v1/panel/arranque", Quien: ana, Negocio: uno}, &e)
	if !e.Completo {
		t.Fatalf("completo: %+v", e)
	}
	for _, p := range e.Pasos {
		if (p.Clave == "logo" || p.Clave == "fotos") && (p.Listo || p.Obligatorio) {
			t.Fatalf("%s: %+v", p.Clave, p)
		}
	}
	// Otro negocio no ve el estado de este.
	dos := alta(a, beto, "dos")
	a.Esperar(403, P{Metodo: "GET", Ruta: "/v1/panel/arranque", Quien: beto, Negocio: uno}, nil)
	a.Esperar(200, P{Metodo: "GET", Ruta: "/v1/panel/arranque", Quien: beto, Negocio: dos}, &e)
	if e.Reservable {
		t.Fatal("dos no tiene nada cargado")
	}
}

func sql(t *testing.T, a *prueba.API, q string, args ...any) {
	t.Helper()
	conn, err := pgx.Connect(context.Background(), a.Bases.AdminURL)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close(context.Background())
	if _, err := conn.Exec(context.Background(), q, args...); err != nil {
		t.Fatal(err)
	}
}

// El estado de cada cuenta (prueba, plan, cobro fallido, sólo lectura), el pulso de hoy y la plata.
// Ahora: domingo 1/11/2026 a las 10.
func TestAdminEstadosYPlata(t *testing.T) {
	a := armar(t)
	uno, dos := alta(a, ana, "uno"), alta(a, beto, "dos")
	tres, cuatro := alta(a, "user_carla00001", "tres"), alta(a, "user_dani000001", "cuatro")
	dia := 24 * time.Hour
	prueba := func(id string, d time.Duration) {
		sql(t, a, "UPDATE negocios SET prueba_hasta = $2 WHERE id = $1", id, a.Ahora.Add(d))
	}
	prueba(uno, 2*dia)   // en prueba, por vencer
	prueba(dos, -10*dia) // pagando
	prueba(tres, -5*dia) // el cobro no pasó
	prueba(cuatro, -dia) // sólo lectura
	sql(t, a, `INSERT INTO suscripciones (negocio_id, referencia, plan, estado, monto, proximo_cobro) VALUES ($1, md5(random()::text), 'completo', 'autorizada', 15000, $2)`,
		dos, a.Ahora.Add(20*dia))
	sql(t, a, `INSERT INTO suscripciones (negocio_id, referencia, plan, estado, monto, gracia_hasta) VALUES ($1, md5(random()::text), 'inicial', 'pausada', 7000, $2)`,
		tres, a.Ahora.Add(3*dia))
	// Hoy: dos turnos por la web y uno del panel (y uno cancelado, que no cuenta).
	sql(t, a, `WITH c AS (INSERT INTO clientes (negocio_id, nombre, telefono) VALUES ($1, 'Cliente', '1111') RETURNING id, negocio_id)
		INSERT INTO turnos (negocio_id, codigo, servicio, profesional, cliente_id, inicio, fin, precio, estado, origen)
		SELECT c.negocio_id, md5(random()::text), 'Corte', 'Ana', c.id, $2::timestamptz, $2::timestamptz + interval '30 minutes', 1000, x.estado, x.origen
		FROM c, (VALUES ('confirmado', 'web'), ('pendiente', 'web'), ('confirmado', 'panel'), ('cancelado', 'web')) x (estado, origen)`,
		uno, a.Ahora.Add(5*time.Hour))

	var est struct {
		TurnosHoy          int  `json:"turnosHoy"`
		PorcentajeWeb      *int `json:"porcentajeWeb"`
		SuscripcionesPagas int  `json:"suscripcionesPagas"`
		Mrr                int  `json:"mrr"`
		PruebasPorVencer   int  `json:"pruebasPorVencer"`
		Conversion         *int `json:"conversion"`
	}
	a.Esperar(200, P{Metodo: "GET", Ruta: "/v1/admin/estadisticas", Quien: equipo}, &est)
	if est.TurnosHoy != 3 || est.PorcentajeWeb == nil || *est.PorcentajeWeb != 67 || est.SuscripcionesPagas != 1 || est.Mrr != 15000 ||
		est.PruebasPorVencer != 1 || est.Conversion == nil || *est.Conversion != 67 {
		t.Fatalf("estadísticas: %+v", est)
	}

	type fila struct {
		Slug       string `json:"slug"`
		Rubro      string `json:"rubro"`
		Estado     string `json:"estado"`
		GraciaDias *int   `json:"graciaDias"`
	}
	var l struct {
		Total    int            `json:"total"`
		Conteos  map[string]int `json:"conteos"`
		Negocios []fila         `json:"negocios"`
	}
	a.Esperar(200, P{Metodo: "GET", Ruta: "/v1/admin/negocios", Quien: equipo}, &l)
	estados := map[string]fila{}
	for _, n := range l.Negocios {
		estados[n.Slug] = n
	}
	if estados["uno"].Estado != "prueba" || estados["dos"].Estado != "completo" || estados["tres"].Estado != "cobro_fallido" ||
		estados["cuatro"].Estado != "solo_lectura" || estados["tres"].GraciaDias == nil || *estados["tres"].GraciaDias != 3 ||
		estados["dos"].GraciaDias != nil {
		t.Fatalf("estados: %+v", estados)
	}
	if l.Conteos["prueba"] != 1 || l.Conteos["completo"] != 1 || l.Conteos["cobro_fallido"] != 1 || l.Conteos["solo_lectura"] != 1 {
		t.Fatalf("conteos: %+v", l.Conteos)
	}
	// El filtro deja los conteos como están (son las pastillas de la búsqueda).
	a.Esperar(200, P{Metodo: "GET", Ruta: "/v1/admin/negocios?estado=cobro_fallido", Quien: equipo}, &l)
	if l.Total != 1 || len(l.Negocios) != 1 || l.Negocios[0].Slug != "tres" || l.Conteos["prueba"] != 1 {
		t.Fatalf("filtro por estado: %+v", l)
	}
	l.Conteos = nil // el JSON se suma a un mapa que ya existe
	a.Esperar(200, P{Metodo: "GET", Ruta: "/v1/admin/negocios?estado=prueba&q=dos", Quien: equipo}, &l)
	if l.Total != 0 || l.Conteos["completo"] != 1 || len(l.Conteos) != 1 {
		t.Fatalf("filtro con búsqueda: %+v", l)
	}
	a.Esperar(400, P{Metodo: "GET", Ruta: "/v1/admin/negocios?estado=oro", Quien: equipo}, nil)

	// Pasada la gracia, el cobro fallido queda en sólo lectura.
	a.Ahora = a.Ahora.Add(4 * dia)
	a.Esperar(200, P{Metodo: "GET", Ruta: "/v1/admin/negocios?q=tres", Quien: equipo}, &l)
	if l.Negocios[0].Estado != "solo_lectura" || l.Negocios[0].GraciaDias != nil {
		t.Fatalf("sin gracia: %+v", l.Negocios[0])
	}
}

// Por email verificado en Clerk (sin importar mayúsculas) también es admin; otro email, no.
func TestAdminPorEmail(t *testing.T) {
	a := armar(t)
	a.Esperar(200, P{Metodo: "GET", Ruta: "/v1/admin/estadisticas", Quien: "user_socia0001"}, nil)
	a.Esperar(404, P{Metodo: "GET", Ruta: "/v1/admin/estadisticas", Quien: "user_otrapersona"}, nil)
}
