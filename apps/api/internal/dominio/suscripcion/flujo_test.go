package suscripcion_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
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
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/auditoria"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/googleplay"
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
	// Se sigue viendo, cancelada (para reactivarla).
	if s := leer(e); !s.Bloqueado || s.Suscripcion == nil || s.Suscripcion.Estado != "cancelada" {
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

func TestCambiarPagarTarjetaYReactivar(t *testing.T) {
	e := cobrosprueba.Armar(t)
	e.SQL("UPDATE negocios SET prueba_hasta = NULL")
	var url struct {
		URL string `json:"url"`
	}
	// Sin suscripción no hay nada que cambiar.
	e.Panel(409, "POST", "/v1/panel/suscripcion/cambiar", map[string]string{"plan": "completo"}, nil)
	e.Panel(404, "POST", "/v1/panel/suscripcion/pagar", nil, nil)
	e.Panel(201, "POST", "/v1/panel/suscripcion", map[string]string{"plan": "inicial"}, &url)
	id := strings.TrimPrefix(url.URL, "https://mp.falso/suscripcion/")
	proximo := e.Ahora.Add(20 * 24 * time.Hour)
	e.MP.Autorizar(id, "authorized", &proximo)
	e.Avisar("subscription_preapproval", id)

	// Cambiar de plan: el precio nuevo va al preapproval y el plan cambia ya.
	e.Panel(400, "POST", "/v1/panel/suscripcion/cambiar", map[string]string{"plan": "oro"}, nil)
	e.Panel(409, "POST", "/v1/panel/suscripcion/cambiar", map[string]string{"plan": "inicial"}, nil)
	var x suscripcion.Suscripcion
	e.Panel(200, "POST", "/v1/panel/suscripcion/cambiar", map[string]string{"plan": "completo"}, &x)
	if x.Plan != "completo" || x.Monto != 15000 || e.MP.Montos[id] != 15000 {
		t.Fatalf("cambiar: %+v %v", x, e.MP.Montos)
	}
	if s := leer(e); s.Plan != "completo" || s.NombrePlan != "Completo" {
		t.Fatalf("plan: %+v", s)
	}
	// Va a la auditoría, entre los cambios de plata; lo del equipo no.
	e.Panel(200, "PUT", "/v1/panel/cobros", map[string]any{"modo": "total"}, nil)
	var aud struct {
		Entradas []auditoria.Entrada `json:"entradas"`
	}
	e.Panel(200, "GET", "/v1/panel/auditoria?plata=1", nil, &aud)
	if len(aud.Entradas) != 2 || aud.Entradas[0].Accion != "cobros.config" || aud.Entradas[1].Accion != "suscripcion.plan_cambiado" {
		t.Fatalf("auditoría de plata: %+v", aud.Entradas)
	}
	// Una recepcionista no toca nada de esto.
	e.Panel(201, "POST", "/v1/panel/equipo", map[string]string{"clerkId": "user_recepcion01", "rol": "recepcion"}, nil)
	e.Esperar(403, P{Metodo: "POST", Ruta: "/v1/panel/suscripcion/tarjeta", Quien: "user_recepcion01", Negocio: e.Negocio}, nil)
	// Con todo al día no hay cuota para pagar; la tarjeta sí se cambia (y no se cobra dos veces el mes).
	e.Panel(409, "POST", "/v1/panel/suscripcion/pagar", map[string]any{}, nil)
	e.Panel(409, "POST", "/v1/panel/suscripcion/reactivar", nil, nil)
	e.Panel(200, "POST", "/v1/panel/suscripcion/tarjeta", nil, &url)
	nueva := strings.TrimPrefix(url.URL, "https://mp.falso/suscripcion/")
	if nueva == id || e.MP.Preaprobacion[nueva] == nil {
		t.Fatalf("tarjeta: %s", url.URL)
	}
	// Mientras no se autoriza, se sigue viendo la de antes.
	if s := leer(e); s.Suscripcion.Estado != "autorizada" || s.Suscripcion.Plan != "completo" {
		t.Fatalf("antes de autorizar la nueva: %+v", s.Suscripcion)
	}
	e.MP.Autorizar(nueva, "authorized", &proximo)
	e.Avisar("subscription_preapproval", nueva)
	if e.MP.Preaprobacion[id].Status != "cancelled" {
		t.Fatalf("la vieja en Mercado Pago: %s", e.MP.Preaprobacion[id].Status)
	}
	if s := leer(e); s.Bloqueado || s.Suscripcion.Estado != "autorizada" || !strings.HasSuffix(*s.Suscripcion.Link, nueva) || s.Plan != "completo" {
		t.Fatalf("con la tarjeta nueva: %+v", s.Suscripcion)
	}

	// El cobro falla: pagar ahora es un alta que cobra ya y reemplaza a la pausada.
	e.MP.Cobros["9001"] = mercadopago.CobroAutorizado{ID: 9001, PreapprovalID: nueva, Payment: struct {
		Status string `json:"status"`
	}{"rejected"}}
	e.Avisar("subscription_authorized_payment", "9001")
	e.Panel(200, "POST", "/v1/panel/suscripcion/pagar", nil, &url)
	otra := strings.TrimPrefix(url.URL, "https://mp.falso/suscripcion/")
	e.MP.Autorizar(otra, "authorized", &proximo)
	e.Avisar("subscription_preapproval", otra)
	if s := leer(e); s.Suscripcion.Estado != "autorizada" || !strings.HasSuffix(*s.Suscripcion.Link, otra) || e.MP.Preaprobacion[nueva].Status != "cancelled" {
		t.Fatalf("después de pagar: %+v", s.Suscripcion)
	}

	// Cancelada del todo: reactivar con el mismo plan.
	e.MP.Autorizar(otra, "cancelled", nil)
	e.Avisar("subscription_preapproval", otra)
	if s := leer(e); s.Suscripcion.Estado != "cancelada" {
		t.Fatalf("cancelada: %+v", s.Suscripcion)
	}
	e.Panel(200, "POST", "/v1/panel/suscripcion/reactivar", nil, &url)
	if p := e.MP.Preaprobacion[strings.TrimPrefix(url.URL, "https://mp.falso/suscripcion/")]; p == nil {
		t.Fatalf("reactivar: %s", url.URL)
	}
	if s := leer(e); s.Suscripcion.Estado != "pendiente" || s.Suscripcion.Plan != "completo" {
		t.Fatalf("reactivada: %+v", s.Suscripcion)
	}
}

type grabador struct{ estado int }

func (g *grabador) Header() http.Header         { return http.Header{} }
func (g *grabador) Write(b []byte) (int, error) { return len(b), nil }
func (g *grabador) WriteHeader(c int)           { g.estado = c }

// TestSuscripcionDesdeElIPhone: la compra en la App Store abre el panel, Mercado Pago no la pisa, las
// notificaciones la renuevan, la apagan y la vencen, y una compra de otro negocio no sirve.
func TestSuscripcionDesdeElIPhone(t *testing.T) {
	e := cobrosprueba.Armar(t)
	e.SQL("UPDATE negocios SET prueba_hasta = $1", e.Ahora.Add(-time.Hour))
	if s := leer(e); !s.Bloqueado {
		t.Fatalf("sin prueba ni suscripción tendría que estar bloqueado: %+v", s)
	}
	vence := e.Ahora.AddDate(0, 1, 0)
	compra := e.Apple.Transaccion("2000001", "ar.com.tengolibre.completo.mensual", e.Negocio, vence)

	// Una firma que no es de la App Store, y una compra hecha para otro negocio, no sirven.
	e.Panel(400, "POST", "/v1/panel/suscripcion/apple", map[string]string{"transaccion": "a.b.c"}, nil)
	otro := e.Apple.Transaccion("2000002", "ar.com.tengolibre.completo.mensual", "00000000-0000-0000-0000-000000000000", vence)
	e.Panel(409, "POST", "/v1/panel/suscripcion/apple", map[string]string{"transaccion": otro}, nil)

	var x suscripcion.Suscripcion
	e.Panel(200, "POST", "/v1/panel/suscripcion/apple", map[string]string{"transaccion": compra}, &x)
	if x.Origen != "apple" || x.Estado != "autorizada" || x.Plan != "completo" {
		t.Fatalf("la compra no quedó: %+v", x)
	}
	if s := leer(e); s.Bloqueado || !s.ConSuscripcion || s.Plan != "completo" || s.Suscripcion == nil || s.Suscripcion.Origen != "apple" {
		t.Fatalf("con la compra tendría que estar abierto: %+v", s)
	}
	// Mandarla dos veces (restaurar compras) no duplica.
	e.Panel(200, "POST", "/v1/panel/suscripcion/apple", map[string]string{"transaccion": compra}, nil)
	// Mercado Pago no la pisa: suscribir, cancelar y cambiar responden que se maneja desde el iPhone.
	e.Panel(409, "POST", "/v1/panel/suscripcion", map[string]string{"plan": "inicial"}, nil)
	e.Panel(409, "POST", "/v1/panel/suscripcion/cancelar", nil, nil)
	e.Panel(409, "POST", "/v1/panel/suscripcion/cambiar", map[string]string{"plan": "inicial"}, nil)

	webhook := func(cuerpo string) {
		e.Esperar(200, P{Metodo: "POST", Ruta: "/v1/webhooks/appstore", Cuerpo: map[string]string{"signedPayload": cuerpo}}, nil)
	}
	// Apagó la renovación: sigue valiendo hasta que vence.
	ren := e.Apple.Firmar(map[string]any{"originalTransactionId": "2000001", "autoRenewStatus": 0, "autoRenewProductId": "ar.com.tengolibre.completo.mensual"})
	webhook(e.Apple.Notificacion("DID_CHANGE_RENEWAL_STATUS", "AUTO_RENEW_DISABLED", compra, ren))
	if s := leer(e); s.Bloqueado || s.Suscripcion.CanceladaEn == nil {
		t.Fatalf("con la renovación apagada sigue valiendo hasta el vencimiento: %+v", s)
	}
	// Renovó (otro mes).
	renovo := e.Apple.Transaccion("2000001", "ar.com.tengolibre.completo.mensual", e.Negocio, vence.AddDate(0, 1, 0))
	webhook(e.Apple.Notificacion("DID_RENEW", "", renovo, ""))
	if s := leer(e); s.Bloqueado || s.Suscripcion.ProximoCobro == nil || !s.Suscripcion.ProximoCobro.After(vence) {
		t.Fatalf("renovada: %+v", s)
	}
	// Una notificación firmada por otro no hace nada (400).
	e.Esperar(400, P{Metodo: "POST", Ruta: "/v1/webhooks/appstore", Cuerpo: map[string]string{"signedPayload": "a.b.c"}}, nil)
	// Venció: el panel vuelve a sólo lectura.
	vencida := e.Apple.Transaccion("2000001", "ar.com.tengolibre.completo.mensual", e.Negocio, e.Ahora.Add(-2*time.Hour))
	webhook(e.Apple.Notificacion("EXPIRED", "VOLUNTARY", vencida, ""))
	if s := leer(e); !s.Bloqueado || s.Plan != "sin_plan" {
		t.Fatalf("vencida tendría que bloquear: %+v", s)
	}
}

// sus arma lo que devuelve Google Play para una suscripción de Tengo Libre.
func sus(estado, negocio string, vence time.Time, renueva bool) googleplay.Suscripcion {
	var g googleplay.Suscripcion
	b, _ := json.Marshal(map[string]any{"subscriptionState": estado, "startTime": vence.AddDate(0, -1, 0).Format(time.RFC3339),
		"externalAccountIdentifiers": map[string]string{"obfuscatedExternalAccountId": negocio},
		"lineItems": []map[string]any{{"productId": "ar.com.tengolibre.completo.mensual", "expiryTime": vence.Format(time.RFC3339),
			"autoRenewingPlan": map[string]bool{"autoRenewEnabled": renueva}}}})
	_ = json.Unmarshal(b, &g)
	return g
}

// TestSuscripcionDesdeAndroid: la compra en Google Play abre el panel, Mercado Pago no la pisa, las
// notificaciones de RTDN la apagan y la vencen, y una compra de otro negocio o sin la clave no sirve.
func TestSuscripcionDesdeAndroid(t *testing.T) {
	e := cobrosprueba.Armar(t)
	e.SQL("UPDATE negocios SET prueba_hasta = $1", e.Ahora.Add(-time.Hour))
	vence := e.Ahora.AddDate(0, 1, 0)
	e.Google.Cargar("tok-1", sus("SUBSCRIPTION_STATE_ACTIVE", e.Negocio, vence, true))
	e.Google.Cargar("tok-otro", sus("SUBSCRIPTION_STATE_ACTIVE", "00000000-0000-0000-0000-000000000000", vence, true))

	e.Panel(400, "POST", "/v1/panel/suscripcion/google", map[string]string{"producto": "x", "token": "no-existe"}, nil)
	e.Panel(409, "POST", "/v1/panel/suscripcion/google", map[string]string{"producto": "x", "token": "tok-otro"}, nil)
	var x suscripcion.Suscripcion
	e.Panel(200, "POST", "/v1/panel/suscripcion/google", map[string]string{"producto": "ar.com.tengolibre.completo.mensual", "token": "tok-1"}, &x)
	if x.Origen != "google" || x.Estado != "autorizada" || x.Plan != "completo" {
		t.Fatalf("la compra no quedó: %+v", x)
	}
	if s := leer(e); s.Bloqueado || s.Suscripcion == nil || s.Suscripcion.Origen != "google" {
		t.Fatalf("con la compra tendría que estar abierto: %+v", s)
	}
	e.Panel(200, "POST", "/v1/panel/suscripcion/google", map[string]string{"producto": "ar.com.tengolibre.completo.mensual", "token": "tok-1"}, nil)
	e.Panel(409, "POST", "/v1/panel/suscripcion", map[string]string{"plan": "inicial"}, nil)
	e.Panel(409, "POST", "/v1/panel/suscripcion/cancelar", nil, nil)

	webhook := func(clave string, esperado int) {
		d, _ := json.Marshal(map[string]any{"packageName": suscripcion.PaqueteAndroid, "subscriptionNotification": map[string]any{"notificationType": 3, "purchaseToken": "tok-1"}})
		e.Esperar(esperado, P{Metodo: "POST", Ruta: "/v1/webhooks/google-play?clave=" + clave,
			Cuerpo: map[string]any{"message": map[string]string{"data": base64.StdEncoding.EncodeToString(d)}}}, nil)
	}
	webhook("otra", 404)
	// Apagó la renovación: sigue valiendo hasta que vence.
	e.Google.Cargar("tok-1", sus("SUBSCRIPTION_STATE_CANCELED", e.Negocio, vence, false))
	webhook(cobrosprueba.ClaveGoogle, 200)
	if s := leer(e); s.Bloqueado || s.Suscripcion.CanceladaEn == nil {
		t.Fatalf("con la renovación apagada sigue valiendo hasta el vencimiento: %+v", s)
	}
	// Venció: el panel vuelve a sólo lectura.
	e.Google.Cargar("tok-1", sus("SUBSCRIPTION_STATE_EXPIRED", e.Negocio, e.Ahora.Add(-2*time.Hour), false))
	webhook(cobrosprueba.ClaveGoogle, 200)
	if s := leer(e); !s.Bloqueado || s.Plan != "sin_plan" {
		t.Fatalf("vencida tendría que bloquear: %+v", s)
	}
}

// TestSondeoGooglePlay: sin RTDN, el worker vuelve a consultar las suscripciones de Play cerca del
// vencimiento. Una renovación cobrada sigue abierta; una que venció cierra el panel; una lejos del
// vencimiento ni se consulta.
func TestSondeoGooglePlay(t *testing.T) {
	e := cobrosprueba.Armar(t)
	e.SQL("UPDATE negocios SET prueba_hasta = $1", e.Ahora.Add(-time.Hour))
	vence := e.Ahora.AddDate(0, 1, 0)
	e.Google.Cargar("tok-1", sus("SUBSCRIPTION_STATE_ACTIVE", e.Negocio, vence, true))
	e.Panel(200, "POST", "/v1/panel/suscripcion/google", map[string]string{"producto": "ar.com.tengolibre.completo.mensual", "token": "tok-1"}, nil)
	ctx := context.Background()

	if n, err := e.Suscripcion.SondearGoogle(ctx, e.Ahora); err != nil || n != 0 {
		t.Fatalf("lejos del vencimiento no se consulta: %d %v", n, err)
	}
	// Google cobró la renovación: vence un mes más tarde.
	e.Google.Cargar("tok-1", sus("SUBSCRIPTION_STATE_ACTIVE", e.Negocio, vence.AddDate(0, 1, 0), true))
	if n, err := e.Suscripcion.SondearGoogle(ctx, vence.Add(-24*time.Hour)); err != nil || n != 1 {
		t.Fatalf("cerca del vencimiento se consulta: %d %v", n, err)
	}
	if s := leer(e); s.Bloqueado || s.Suscripcion == nil || s.Suscripcion.ProximoCobro == nil || !s.Suscripcion.ProximoCobro.Equal(vence.AddDate(0, 1, 0)) {
		t.Fatalf("la renovación tendría que correr el vencimiento: %+v", s.Suscripcion)
	}
	// Se canceló y venció.
	vence = vence.AddDate(0, 1, 0)
	e.Google.Cargar("tok-1", sus("SUBSCRIPTION_STATE_EXPIRED", e.Negocio, vence, false))
	if _, err := e.Suscripcion.SondearGoogle(ctx, vence.Add(2*time.Hour)); err != nil {
		t.Fatal(err)
	}
	if s := leer(e); !s.Bloqueado || s.Plan != "sin_plan" {
		t.Fatalf("vencida tendría que bloquear: %+v", s)
	}
}
