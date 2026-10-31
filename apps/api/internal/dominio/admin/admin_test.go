package admin_test

import (
	"testing"
	"time"

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

// Por email verificado en Clerk (sin importar mayúsculas) también es admin; otro email, no.
func TestAdminPorEmail(t *testing.T) {
	a := armar(t)
	a.Esperar(200, P{Metodo: "GET", Ruta: "/v1/admin/estadisticas", Quien: "user_socia0001"}, nil)
	a.Esperar(404, P{Metodo: "GET", Ruta: "/v1/admin/estadisticas", Quien: "user_otrapersona"}, nil)
}
