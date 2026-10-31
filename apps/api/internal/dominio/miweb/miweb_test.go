package miweb_test

import (
	"testing"
	"time"

	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/miweb"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/negocios"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/prueba"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/servidor"
)

type P = prueba.Pedido

type galeria struct {
	Fotos []miweb.Foto `json:"fotos"`
}

func TestMiWebYGaleria(t *testing.T) {
	a := prueba.Nueva(t, time.Now(), func(a *prueba.API) []servidor.Dominio {
		return []servidor.Dominio{
			&negocios.Servicio{Pool: a.Bases.App, Sesion: a.Sesion},
			&miweb.Servicio{Pool: a.Bases.App, Sesion: a.Sesion},
		}
	})
	const ana, beto = "user_anaduenia01", "user_betoduenio1"
	var uno, dos negocios.Negocio
	a.Esperar(201, P{Metodo: "POST", Ruta: "/v1/negocios", Quien: ana, Cuerpo: map[string]string{"slug": "uno", "nombre": "Uno"}}, &uno)
	a.Esperar(201, P{Metodo: "POST", Ruta: "/v1/negocios", Quien: beto, Cuerpo: map[string]string{"slug": "dos", "nombre": "Dos"}}, &dos)
	panel := func(estado int, metodo, ruta string, cuerpo, v any) {
		t.Helper()
		a.Esperar(estado, P{Metodo: metodo, Ruta: ruta, Quien: ana, Negocio: uno.ID, Cuerpo: cuerpo}, v)
	}

	var w miweb.Web
	panel(200, "GET", "/v1/panel/mi-web", nil, &w)
	if w.ColorPrimario != "#6366f1" || w.Latitud != nil || w.Estilo != (miweb.Estilo{"carta", "foto", "crema", "grandes", "redondos"}) {
		t.Fatalf("por defecto: %+v", w)
	}
	panel(200, "PATCH", "/v1/panel/mi-web", map[string]any{"colorPrimario": "#1A2B3C", "rubro": "barberia", "acerca": "  Desde 1990.  ",
		"portada": "https://api.prueba/archivos/x.jpg", "latitud": -38.0, "longitud": -57.55, "sitioWeb": "https://uno.com"}, &w)
	if w.ColorPrimario != "#1a2b3c" || w.Rubro != "barberia" || w.Acerca != "Desde 1990." || w.Latitud == nil || *w.Latitud != -38 ||
		w.Portada == nil || w.SitioWeb != "https://uno.com" {
		t.Fatalf("guardado: %+v", w)
	}
	// Lo inválido no entra.
	for _, malo := range []map[string]any{
		{"colorAcento": "rojo"}, {"rubro": "carniceria"}, {"sitioWeb": "javascript:alert(1)"}, {"portada": "data:image/png;base64,AAAA"},
		{"latitud": 10.0}, {"latitud": 91.0, "longitud": 0.0}, {"estilo": map[string]string{"plantilla": "otra"}},
	} {
		panel(400, "PATCH", "/v1/panel/mi-web", malo, nil)
	}
	// Publicar el estilo: lo que no viene toma el de por defecto.
	panel(200, "PATCH", "/v1/panel/mi-web", map[string]any{"estilo": map[string]string{"plantilla": "galeria", "fondo": "noche"}}, &w)
	if w.Estilo != (miweb.Estilo{"galeria", "foto", "noche", "grandes", "redondos"}) || w.Rubro != "barberia" {
		t.Fatalf("estilo: %+v", w)
	}
	// Sacar portada y ubicación.
	panel(200, "PATCH", "/v1/panel/mi-web", map[string]any{"portada": "", "sinUbicacion": true}, &w)
	if w.Portada != nil || w.Latitud != nil || w.Longitud != nil || w.ColorPrimario != "#1a2b3c" {
		t.Fatalf("sacar: %+v", w)
	}

	// Galería: tres fotos, se reordenan; el otro negocio no las ve ni las toca.
	var f1, f2, f3 miweb.Foto
	panel(201, "POST", "/v1/panel/mi-web/galeria", map[string]string{"url": "https://api.prueba/archivos/1.jpg", "epigrafe": "Frente"}, &f1)
	panel(201, "POST", "/v1/panel/mi-web/galeria", map[string]string{"url": "https://api.prueba/archivos/2.jpg"}, &f2)
	panel(201, "POST", "/v1/panel/mi-web/galeria", map[string]string{"url": "https://api.prueba/archivos/3.jpg"}, &f3)
	panel(400, "POST", "/v1/panel/mi-web/galeria", map[string]string{"url": "javascript:alert(1)"}, nil)
	if f1.Orden != 0 || f3.Orden != 2 {
		t.Fatalf("orden al sumar: %d %d", f1.Orden, f3.Orden)
	}
	var g galeria
	panel(200, "PUT", "/v1/panel/mi-web/galeria/orden", map[string]any{"ids": []string{f3.ID, f1.ID, f2.ID}}, &g)
	if len(g.Fotos) != 3 || g.Fotos[0].ID != f3.ID || g.Fotos[1].ID != f1.ID || g.Fotos[2].Orden != 2 {
		t.Fatalf("reordenada: %+v", g.Fotos)
	}
	// Un orden que no tiene todas las fotos es un pedido viejo.
	panel(409, "PUT", "/v1/panel/mi-web/galeria/orden", map[string]any{"ids": []string{f1.ID, f2.ID}}, nil)
	panel(409, "PUT", "/v1/panel/mi-web/galeria/orden", map[string]any{"ids": []string{f1.ID, f1.ID, f2.ID}}, nil)

	otro := func(estado int, metodo, ruta string, cuerpo, v any) {
		t.Helper()
		a.Esperar(estado, P{Metodo: metodo, Ruta: ruta, Quien: beto, Negocio: dos.ID, Cuerpo: cuerpo}, v)
	}
	otro(200, "GET", "/v1/panel/mi-web/galeria", nil, &g)
	if len(g.Fotos) != 0 {
		t.Fatal("el otro negocio ve la galería")
	}
	otro(404, "DELETE", "/v1/panel/mi-web/galeria/"+f1.ID, nil, nil)
	otro(404, "PATCH", "/v1/panel/mi-web/galeria/"+f1.ID, map[string]string{"epigrafe": "mía"}, nil)
	otro(409, "PUT", "/v1/panel/mi-web/galeria/orden", map[string]any{"ids": []string{f3.ID, f1.ID, f2.ID}}, nil)
	// Y X-Negocio de un negocio ajeno no pasa.
	a.Esperar(403, P{Metodo: "GET", Ruta: "/v1/panel/mi-web", Quien: beto, Negocio: uno.ID}, nil)

	panel(200, "PATCH", "/v1/panel/mi-web/galeria/"+f2.ID, map[string]string{"epigrafe": "Adentro"}, nil)
	panel(204, "DELETE", "/v1/panel/mi-web/galeria/"+f3.ID, nil, nil)

	// La página pública.
	var pub struct {
		miweb.Web
		Galeria []miweb.Foto `json:"galeria"`
	}
	a.Esperar(200, P{Metodo: "GET", Ruta: "/v1/p/uno/web"}, &pub)
	if pub.Rubro != "barberia" || len(pub.Galeria) != 2 || pub.Galeria[0].ID != f1.ID || pub.Galeria[1].Epigrafe != "Adentro" {
		t.Fatalf("pública: %+v", pub)
	}
	a.Esperar(404, P{Metodo: "GET", Ruta: "/v1/p/no-existe/web"}, nil)
}
