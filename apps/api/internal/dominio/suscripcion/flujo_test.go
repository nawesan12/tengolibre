package suscripcion_test

import (
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/cobros/cobrosprueba"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/negocios"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/suscripcion"
	"github.com/nawesan12/tengolibre/apps/api/internal/integraciones/mercadopago"
	"github.com/nawesan12/tengolibre/apps/api/internal/integraciones/mercadopago/mpfalso"
)

type P = cobrosprueba.P

type estado struct {
	suscripcion.Acceso
	NombrePlan  string                   `json:"nombrePlan"`
	Catalogo    []suscripcion.Plan       `json:"catalogo"`
	Suscripcion *suscripcion.Suscripcion `json:"suscripcion"`
}

func leer(e *cobrosprueba.Entorno) estado {
	e.T.Helper()
	var s estado
	e.Panel(200, "GET", "/v1/panel/suscripcion", nil, &s)
	return s
}

var cupones int

// escribe prueba las escrituras del panel de los dominios de plata: todas con el mismo estado.
func escribe(e *cobrosprueba.Entorno, esperado int) {
	e.T.Helper()
	cupones++
	e.Panel(esperado, "POST", "/v1/panel/cupones", map[string]any{"codigo": "CUPON" + strconv.Itoa(cupones), "tipo": "fijo", "valor": 100}, nil)
	e.Panel(map[int]int{201: 200, 402: 402}[esperado], "PUT", "/v1/panel/cobros", map[string]any{"modo": "sin_cobro"}, nil)
	e.Panel(map[int]int{201: 200, 402: 402}[esperado], "PUT", "/v1/panel/referidos/config", map[string]any{"activo": false}, nil)
}

func TestPanelEnSoloLecturaConLaPruebaVencida(t *testing.T) {
	e := cobrosprueba.Armar(t)
	e.SQL("UPDATE negocios SET prueba_hasta = $1", e.Ahora.Add(60*time.Hour))
	s := leer(e)
	if s.Bloqueado || s.DiasDePrueba != 3 || s.Plan != "completo" || len(s.Catalogo) != 2 || s.Suscripcion != nil {
		t.Fatalf("en prueba: %+v", s)
	}
	escribe(e, 201)

	// Vencida la prueba y sin suscripción: escribir responde 402, leer sigue andando, y la página
	// pública sigue tomando reservas.
	e.SQL("UPDATE negocios SET prueba_hasta = $1", e.Ahora.Add(-time.Hour))
	if s = leer(e); !s.Bloqueado || s.Plan != "sin_plan" || s.PruebaHasta != nil {
		t.Fatalf("vencida: %+v", s)
	}
	escribe(e, 402)
	e.Panel(402, "POST", "/v1/panel/membresias/planes", map[string]any{"nombre": "Plan", "precio": 1000}, nil)
	e.Panel(402, "POST", "/v1/panel/mercadopago/conectar", nil, nil)
	e.Panel(200, "GET", "/v1/panel/cupones", nil, nil)
	e.Panel(200, "GET", "/v1/panel/membresias/planes", nil, nil)
	// El bloqueo vale para todo el panel (sesion.Panel), no sólo para los paquetes de plata.
	e.Panel(402, "POST", "/v1/panel/servicios", map[string]any{"nombre": "Barba", "duracion": 30, "precio": 5000}, nil)
	e.Panel(402, "PATCH", "/v1/panel/negocio", map[string]any{"nombre": "Otro"}, nil)
	e.Panel(200, "GET", "/v1/panel/servicios", nil, nil)
	e.Reservar(201, "09:00", "1111", nil)
	// El aviso de sólo lectura que muestra el panel a cualquier rol.
	var a suscripcion.Acceso
	if e.Panel(200, "GET", "/v1/panel/acceso", nil, &a); !a.Bloqueado || a.Plan != "sin_plan" {
		t.Fatalf("acceso: %+v", a)
	}

	// Suscribirse sí se puede (es lo que destraba). El link es de la cuenta de la plataforma.
	e.Panel(400, "POST", "/v1/panel/suscripcion", map[string]string{"plan": "oro"}, nil)
	var alta struct {
		ID  string `json:"suscripcionId"`
		URL string `json:"url"`
	}
	e.Panel(201, "POST", "/v1/panel/suscripcion", map[string]string{"plan": "completo"}, &alta)
	// Abandonar el checkout y volver a empezar no traba nada.
	e.Panel(201, "POST", "/v1/panel/suscripcion", map[string]string{"plan": "inicial"}, &alta)
	id := strings.TrimPrefix(alta.URL, "https://mp.falso/suscripcion/")
	pre := e.MP.Preaprobacion[id]
	if pre == nil || !strings.HasPrefix(pre.ExternalReference, e.Negocio+"-") || pre.PayerEmail != "duenia0001@prueba.tengolibre.com.ar" {
		t.Fatalf("preapproval: %+v", pre)
	}
	if s = leer(e); !s.Bloqueado || s.Suscripcion == nil || s.Suscripcion.Estado != "pendiente" || s.Suscripcion.Plan != "inicial" || s.Suscripcion.Monto != 7000 {
		t.Fatalf("pendiente no destraba: %+v %+v", s, s.Suscripcion)
	}

	// Mercado Pago avisa que quedó autorizada: el panel vuelve a escribir.
	proximo := e.Ahora.Add(30 * 24 * time.Hour)
	e.MP.Autorizar(id, "authorized", &proximo)
	if c := e.Avisar("subscription_preapproval", id); c != 200 {
		t.Fatalf("aviso: %d", c)
	}
	if s = leer(e); s.Bloqueado || s.Plan != "inicial" || s.NombrePlan != "Inicial" || s.Suscripcion.Estado != "autorizada" ||
		!s.Suscripcion.ProximoCobro.Equal(proximo) {
		t.Fatalf("autorizada: %+v %+v", s, s.Suscripcion)
	}
	escribe(e, 201)
	e.Panel(409, "POST", "/v1/panel/suscripcion", map[string]string{"plan": "completo"}, nil)
	acceso := func() suscripcion.Acceso {
		e.T.Helper()
		var a suscripcion.Acceso
		e.Panel(200, "GET", "/v1/panel/acceso", nil, &a)
		return a
	}
	if a := acceso(); a.ProximoCobro == nil || !a.ProximoCobro.Fecha.Equal(proximo) || a.ProximoCobro.Monto != 7000 || a.CobroFallidoHasta != nil {
		t.Fatalf("acceso autorizada: %+v", a)
	}

	// Un cobro mensual rechazado la pausa con 7 días de gracia; pasada la gracia, se bloquea.
	e.MP.Cobros["5001"] = mercadopago.CobroAutorizado{ID: 5001, PreapprovalID: id, Payment: struct {
		Status string `json:"status"`
	}{"rejected"}}
	e.Avisar("subscription_authorized_payment", "5001")
	if s = leer(e); s.Bloqueado || s.Suscripcion.Estado != "pausada" || !s.Suscripcion.GraciaHasta.Equal(e.Ahora.AddDate(0, 0, 7)) {
		t.Fatalf("pausada: %+v", s.Suscripcion)
	}
	if a := acceso(); a.ProximoCobro != nil || a.CobroFallidoHasta == nil || !a.CobroFallidoHasta.Equal(e.Ahora.AddDate(0, 0, 7)) {
		t.Fatalf("acceso con el cobro fallido: %+v", a)
	}
	e.Ahora = e.Ahora.AddDate(0, 0, 8)
	if s = leer(e); !s.Bloqueado {
		t.Fatal("pasada la gracia tiene que bloquear")
	}
	// El reintento pasa: vuelve.
	e.MP.Cobros["5002"] = mercadopago.CobroAutorizado{ID: 5002, PreapprovalID: id, Payment: struct {
		Status string `json:"status"`
	}{"approved"}}
	e.Avisar("subscription_authorized_payment", "5002")
	if s = leer(e); s.Bloqueado || s.Suscripcion.Estado != "autorizada" || s.Suscripcion.UltimoCobro == nil {
		t.Fatalf("cobro aprobado: %+v", s.Suscripcion)
	}

	// Otro negocio no ve esta suscripción, y una recepcionista no la toca.
	const otra = "user_otra000001"
	var n negocios.Negocio
	e.Esperar(201, P{Metodo: "POST", Ruta: "/v1/negocios", Quien: otra, Cuerpo: map[string]string{"slug": "otro", "nombre": "Otro"}}, &n)
	var ajena estado
	e.Esperar(200, P{Metodo: "GET", Ruta: "/v1/panel/suscripcion", Quien: otra, Negocio: n.ID}, &ajena)
	if ajena.Suscripcion != nil || ajena.ConSuscripcion {
		t.Fatalf("el otro negocio ve una suscripción: %+v", ajena)
	}
	e.Panel(201, "POST", "/v1/panel/equipo", map[string]string{"clerkId": "user_recepcion01", "rol": "recepcion"}, nil)
	e.Esperar(403, P{Metodo: "POST", Ruta: "/v1/panel/suscripcion/cancelar", Quien: "user_recepcion01", Negocio: e.Negocio}, nil)

	// Cancelar con el mes pago: sigue valiendo hasta el próximo cobro, y Mercado Pago la cancela.
	e.Panel(204, "POST", "/v1/panel/suscripcion/cancelar", map[string]string{"motivo": "Cierro por vacaciones"}, nil)
	if e.MP.Preaprobacion[id].Status != "cancelled" {
		t.Fatalf("en Mercado Pago: %s", e.MP.Preaprobacion[id].Status)
	}
	if s = leer(e); s.Bloqueado || s.Suscripcion.Estado != "pausada" || s.Suscripcion.CanceladaEn == nil {
		t.Fatalf("cancelada con el mes pago: %+v", s.Suscripcion)
	}
	if a := acceso(); a.ProximoCobro != nil || a.CobroFallidoHasta != nil {
		t.Fatalf("acceso dada de baja: %+v", a)
	}
	// El aviso de la cancelación no la corta antes.
	e.Avisar("subscription_preapproval", id)
	if s = leer(e); s.Bloqueado {
		t.Fatal("el aviso cortó el mes pago")
	}
	e.Ahora = proximo.Add(time.Hour)
	if s = leer(e); !s.Bloqueado {
		t.Fatal("terminado el mes pago, se bloquea")
	}
	escribe(e, 402)
}

func TestCanceladaDesdeMercadoPago(t *testing.T) {
	e := cobrosprueba.Armar(t)
	e.SQL("UPDATE negocios SET prueba_hasta = NULL")
	var alta struct {
		URL string `json:"url"`
	}
	e.Panel(201, "POST", "/v1/panel/suscripcion", map[string]string{"plan": "completo"}, &alta)
	id := strings.TrimPrefix(alta.URL, "https://mp.falso/suscripcion/")
	e.MP.Autorizar(id, "authorized", nil)
	e.Avisar("subscription_preapproval", id)
	if s := leer(e); s.Bloqueado || s.Plan != "completo" {
		t.Fatalf("autorizada: %+v", s)
	}
	// La persona la cancela desde Mercado Pago (sin pasar por el panel): se corta.
	e.MP.Autorizar(id, "cancelled", nil)
	e.Avisar("subscription_preapproval", id)
	if s := leer(e); !s.Bloqueado || s.Suscripcion != nil {
		t.Fatalf("cancelada: %+v", s)
	}
	// Un aviso de algo que no existe se contesta 200 (reintentar no lo arregla); uno sin firma, 401.
	if c := e.Avisar("subscription_preapproval", "nada"); c != 200 {
		t.Fatalf("aviso desconocido: %d", c)
	}
	r := mpfalso.Aviso("subscription_preapproval", id, 0, e.Ahora.Add(-time.Hour))
	w := &grabador{}
	e.Handler.ServeHTTP(w, r)
	if w.estado != 401 {
		t.Fatalf("aviso viejo: %d", w.estado)
	}
}

type grabador struct{ estado int }

func (g *grabador) Header() http.Header         { return http.Header{} }
func (g *grabador) Write(b []byte) (int, error) { return len(b), nil }
func (g *grabador) WriteHeader(c int)           { g.estado = c }
