package tiemporeal_test

import (
	"bufio"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/catalogo"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/disponibilidad"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/negocios"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/turnos"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/prueba"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/servidor"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/tiemporeal"
)

type P = prueba.Pedido

func en(s string) time.Time {
	t, _ := time.ParseInLocation("2006-01-02 15:04", s, disponibilidad.ZonaPorDefecto)
	return t
}

// abrir conecta al stream como un panel y devuelve las líneas que van llegando.
func abrir(t *testing.T, srv *httptest.Server, quien, negocio string) <-chan string {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	req, _ := http.NewRequestWithContext(ctx, "GET", srv.URL+"/v1/panel/eventos", nil)
	req.Header.Set("Authorization", "Bearer dev:"+quien)
	req.Header.Set("X-Negocio", negocio)
	res, err := srv.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != 200 || res.Header.Get("Content-Type") != "text/event-stream" {
		t.Fatalf("stream: %d %s", res.StatusCode, res.Header.Get("Content-Type"))
	}
	lineas := make(chan string, 100)
	go func() {
		defer res.Body.Close()
		sc := bufio.NewScanner(res.Body)
		for sc.Scan() {
			lineas <- sc.Text()
		}
	}()
	return lineas
}

// esperar lee hasta encontrar una línea que contenga algo (o se vence).
func esperar(lineas <-chan string, algo string, dentro time.Duration) bool {
	fin := time.After(dentro)
	for {
		select {
		case l := <-lineas:
			if strings.Contains(l, algo) {
				return true
			}
		case <-fin:
			return false
		}
	}
}

func TestUnaReservaLlegaAlPanelDeEseNegocio(t *testing.T) {
	var tr *tiemporeal.Servicio
	a := prueba.Nueva(t, en("2026-11-01 10:00"), func(a *prueba.API) []servidor.Dominio {
		app := a.Bases.App
		tr = &tiemporeal.Servicio{Pool: app, Sesion: a.Sesion}
		return []servidor.Dominio{
			&negocios.Servicio{Pool: app, Sesion: a.Sesion},
			&catalogo.Servicio{Pool: app, Sesion: a.Sesion},
			&turnos.Servicio{Pool: app, Sesion: a.Sesion, Ahora: a.Reloj()},
			tr,
		}
	})
	ctx, cancel := context.WithCancel(context.Background())
	go tr.Escuchar(ctx)
	srv := httptest.NewServer(a.Handler)
	// Los Cleanup corren al revés: primero se cortan los streams (abrir), después el servidor y la
	// escucha. Con defer, srv.Close esperaría a streams que nunca terminan.
	t.Cleanup(func() { srv.Close(); cancel() })

	alta := func(quien, slug string) string {
		var n negocios.Negocio
		a.Esperar(201, P{Metodo: "POST", Ruta: "/v1/negocios", Quien: quien, Cuerpo: map[string]string{"slug": slug, "nombre": slug}}, &n)
		return n.ID
	}
	uno, otro := alta("user_duenia0001", "uno"), alta("user_duenio0002", "otro")
	var s catalogo.Prestacion
	a.Esperar(201, P{Metodo: "POST", Ruta: "/v1/panel/servicios", Quien: "user_duenia0001", Negocio: uno,
		Cuerpo: map[string]any{"nombre": "Corte", "duracion": 30, "precio": 1000}}, &s)
	a.Esperar(201, P{Metodo: "POST", Ruta: "/v1/panel/profesionales", Quien: "user_duenia0001", Negocio: uno,
		Cuerpo: map[string]any{"nombre": "Ana", "servicios": []string{s.ID}, "horario": []disponibilidad.Franja{{Dia: 1, Desde: "09:00", Hasta: "12:00"}}}}, nil)

	// Sin ser del negocio, no se escucha.
	req, _ := http.NewRequest("GET", srv.URL+"/v1/panel/eventos", nil)
	req.Header.Set("Authorization", "Bearer dev:user_duenio0002")
	req.Header.Set("X-Negocio", uno)
	if res, err := srv.Client().Do(req); err != nil || res.StatusCode != 403 {
		t.Fatalf("panel ajeno: %v %v", err, res)
	}

	panelUno := abrir(t, srv, "user_duenia0001", uno)
	panelOtro := abrir(t, srv, "user_duenio0002", otro)
	if !esperar(panelUno, ": conectado", 3*time.Second) || !esperar(panelOtro, ": conectado", 3*time.Second) {
		t.Fatal("no conectó")
	}
	for tr.Suscriptos(uno) != 1 {
		time.Sleep(10 * time.Millisecond)
	}
	// Una reserva desde la página pública de "uno".
	var turno turnos.Turno
	a.Esperar(201, P{Metodo: "POST", Ruta: "/v1/p/uno/turnos", Cuerpo: map[string]any{"servicioId": s.ID,
		"inicio": en("2026-11-02 09:00").Format(time.RFC3339), "cliente": map[string]string{"nombre": "Lu", "telefono": "1"}}}, &turno)
	if !esperar(panelUno, `"id":"`+turno.ID+`"`, 5*time.Second) {
		t.Fatal("el panel del negocio no se enteró de la reserva")
	}
	if esperar(panelOtro, turno.ID, 500*time.Millisecond) {
		t.Fatal("el panel de otro negocio se enteró de una reserva ajena")
	}
}
