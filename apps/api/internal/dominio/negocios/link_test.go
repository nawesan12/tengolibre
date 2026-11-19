package negocios_test

import (
	"testing"
	"time"

	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/negocios"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/prueba"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/servidor"
)

type P = prueba.Pedido

func TestCambiarLink(t *testing.T) {
	a := prueba.Nueva(t, time.Now(), func(a *prueba.API) []servidor.Dominio {
		return []servidor.Dominio{&negocios.Servicio{Pool: a.Bases.App, Sesion: a.Sesion}}
	})
	const ana, beto = "user_anaduenia01", "user_betoduenio1"
	var uno, dos negocios.Negocio
	a.Esperar(201, P{Metodo: "POST", Ruta: "/v1/negocios", Quien: ana, Cuerpo: map[string]string{"slug": "uno", "nombre": "Uno"}}, &uno)
	a.Esperar(201, P{Metodo: "POST", Ruta: "/v1/negocios", Quien: beto, Cuerpo: map[string]string{"slug": "dos", "nombre": "Dos"}}, &dos)
	cambiar := func(estado int, quien, negocio, slug string) negocios.Link {
		t.Helper()
		var l negocios.Link
		var v any
		if estado == 200 {
			v = &l
		}
		a.Esperar(estado, P{Metodo: "PATCH", Ruta: "/v1/panel/negocio/link", Quien: quien, Negocio: negocio, Cuerpo: map[string]string{"slug": slug}}, v)
		return l
	}
	publico := func(slug string) string {
		t.Helper()
		var id *string
		a.Fila("SELECT app.negocio_publico($1)::text", []any{slug}, &id)
		if id == nil {
			return ""
		}
		return *id
	}
	estado := func(slug string) string {
		t.Helper()
		var e negocios.Estado
		a.Esperar(200, P{Metodo: "GET", Ruta: "/v1/slugs/" + slug}, &e)
		return e.Estado
	}

	// Reservado, mal formado u ocupado: no.
	cambiar(409, ana, uno.ID, "panel")
	cambiar(400, ana, uno.ID, "Con Espacios")
	cambiar(409, ana, uno.ID, "dos")
	if estado("dos") != "tomada" || estado("libre-ya") != "libre" || estado("panel") != "invalida" {
		t.Fatal("estado público")
	}

	// Cambiar: el viejo sigue llevando al negocio.
	l := cambiar(200, ana, uno.ID, "uno-nuevo")
	if l.Slug != "uno-nuevo" || len(l.Anteriores) != 1 || l.Anteriores[0].Slug != "uno" || l.Restantes != negocios.TopeCambios-1 {
		t.Fatalf("cambio: %+v", l)
	}
	if publico("uno") != uno.ID || publico("uno-nuevo") != uno.ID {
		t.Fatal("el link viejo tiene que seguir andando")
	}
	// En el panel, el propio anterior se puede volver a usar; para otro, está tomado.
	var e negocios.Estado
	a.Esperar(200, P{Metodo: "GET", Ruta: "/v1/panel/negocio/link/uno", Quien: ana, Negocio: uno.ID}, &e)
	if e.Estado != "libre" {
		t.Fatalf("propio anterior: %s", e.Estado)
	}
	a.Esperar(200, P{Metodo: "GET", Ruta: "/v1/panel/negocio/link/uno-nuevo", Quien: ana, Negocio: uno.ID}, &e)
	if e.Estado != "propia" {
		t.Fatalf("el de ahora: %s", e.Estado)
	}
	// Nadie se queda con los QR impresos de otro: ni al cambiar ni al crear un negocio.
	cambiar(409, beto, dos.ID, "uno")
	a.Esperar(409, P{Metodo: "POST", Ruta: "/v1/negocios", Quien: beto, Cuerpo: map[string]string{"slug": "uno", "nombre": "Otro"}}, nil)
	if estado("uno") != "tomada" {
		t.Fatal("el anterior retenido sale tomado")
	}

	// Volver al anterior propio: «uno-nuevo» pasa a ser el anterior.
	l = cambiar(200, ana, uno.ID, "uno")
	if l.Slug != "uno" || len(l.Anteriores) != 1 || l.Anteriores[0].Slug != "uno-nuevo" {
		t.Fatalf("volver: %+v", l)
	}
	// El mismo link no cuenta como cambio.
	if l2 := cambiar(200, ana, uno.ID, "uno"); l2.Restantes != l.Restantes {
		t.Fatal("el mismo no suma")
	}

	// Tope: 5 cada 30 días.
	cambiar(200, ana, uno.ID, "uno-3")
	cambiar(200, ana, uno.ID, "uno-4")
	l = cambiar(200, ana, uno.ID, "uno-5")
	if l.Restantes != 0 {
		t.Fatalf("restantes: %+v", l)
	}
	cambiar(409, ana, uno.ID, "uno-6")
	// Todos los anteriores siguen andando.
	for _, s := range []string{"uno-nuevo", "uno-3", "uno-4"} {
		if publico(s) != uno.ID {
			t.Fatalf("%s dejó de andar", s)
		}
	}

	// Mientras uno siga activo, su link anterior no se libera aunque pasen 90 días (los QR impresos
	// duran más); de un negocio inactivo, sí.
	a.Fila("UPDATE slugs_anteriores SET dejado_en = now() - interval '91 days' WHERE slug = 'uno-3'", nil)
	cambiar(409, beto, dos.ID, "uno-3")
	a.Fila("UPDATE negocios SET activo = false WHERE id = $1", []any{uno.ID})
	cambiar(200, beto, dos.ID, "uno-3")
	if publico("uno-3") != dos.ID || publico("dos") != dos.ID {
		t.Fatal("uno-3 ahora es de dos (y dos sigue andando)")
	}

	// El alta con un link anterior ya liberado borra la fila vieja: no vuelve a resolver al negocio viejo.
	a.Fila("UPDATE slugs_anteriores SET dejado_en = now() - interval '91 days' WHERE slug = 'uno-4'", nil)
	var tres negocios.Negocio
	a.Esperar(201, P{Metodo: "POST", Ruta: "/v1/negocios", Quien: beto, Cuerpo: map[string]string{"slug": "uno-4", "nombre": "Tres"}}, &tres)
	var quedan int
	a.Fila("SELECT count(*) FROM slugs_anteriores WHERE slug = 'uno-4'", nil, &quedan)
	if quedan != 0 || publico("uno-4") != tres.ID {
		t.Fatalf("la fila vieja quedó: %d", quedan)
	}

	// Recepción no cambia el link; sólo dueño o encargado.
	cambiar(403, beto, uno.ID, "uno-7")
}
