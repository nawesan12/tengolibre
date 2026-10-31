package cobros_test

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/cobros"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/cobros/cobrosprueba"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/negocios"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/turnos"
	"github.com/nawesan12/tengolibre/apps/api/internal/integraciones/mercadopago"
	"github.com/nawesan12/tengolibre/apps/api/internal/integraciones/mercadopago/mpfalso"
)

type P = cobrosprueba.P

type pagos struct {
	Pagos []cobros.Pago `json:"pagos"`
}

type libres struct {
	Horarios []turnos.HorarioLibre `json:"horarios"`
}

func referencia(t *testing.T, tu turnos.Turno) string {
	t.Helper()
	if tu.Pago == nil || !strings.HasPrefix(tu.Pago.URL, "https://mp.falso/checkout/") {
		t.Fatalf("la reserva no trajo el link de pago: %+v", tu.Pago)
	}
	return strings.TrimPrefix(tu.Pago.URL, "https://mp.falso/checkout/")
}

func TestReservaConSenaYWebhook(t *testing.T) {
	e := cobrosprueba.Armar(t)

	// Configurado para cobrar pero sin la cuenta conectada: la reserva se toma sin cobrar.
	e.Panel(200, "PUT", "/v1/panel/cobros", map[string]any{"modo": "porcentaje", "senaPorcentaje": 50}, nil)
	var cfg cobros.Config
	e.Esperar(200, P{Metodo: "GET", Ruta: "/v1/p/el-corte/cobro"}, &cfg)
	if cfg.Modo != "sin_cobro" {
		t.Fatalf("sin cuenta conectada la página no tiene que pedir seña: %+v", cfg)
	}
	sin := e.Reservar(201, "12:00", "5555", nil)
	if sin.Estado != "pendiente" || sin.Pago != nil {
		t.Fatalf("sin Mercado Pago: %+v", sin)
	}

	e.ConectarMP()
	e.Esperar(200, P{Metodo: "GET", Ruta: "/v1/p/el-corte/cobro"}, &cfg)
	if cfg.Modo != "porcentaje" || cfg.SenaPorcentaje != 50 {
		t.Fatalf("config pública: %+v", cfg)
	}

	// La reserva queda esperando el pago por 15 minutos, con el link de Checkout Pro por la seña.
	tu := e.Reservar(201, "09:00", "1111", nil)
	if tu.Estado != "pendiente_pago" || tu.VenceEn == nil || !tu.VenceEn.Equal(e.Ahora.Add(15*time.Minute)) ||
		tu.Pago.Monto != 4000 || tu.Pago.Modo != "porcentaje" {
		t.Fatalf("reserva con seña: %+v %+v", tu, tu.Pago)
	}
	ref := referencia(t, tu)
	pref := e.MP.Preferencias[ref]
	if e.MP.TokenDePreferencia[ref] != "APP_USR-777" || pref.Items[0].UnitPrice != 4000 || !pref.Expires ||
		pref.ExpirationDateTo != mercadopago.Vencimiento(*tu.VenceEn) {
		t.Fatalf("preferencia: %+v con %q", pref, e.MP.TokenDePreferencia[ref])
	}
	// Pedir el link otra vez reusa la preferencia.
	var link struct {
		URL   string `json:"url"`
		Monto int64  `json:"monto"`
	}
	e.Esperar(200, P{Metodo: "POST", Ruta: "/v1/t/" + tu.Codigo + "/pagar"}, &link)
	if link.URL != tu.Pago.URL || link.Monto != 4000 || len(e.MP.Preferencias) != 1 {
		t.Fatalf("link de nuevo: %+v", link)
	}
	// Mientras espera el pago, el horario está tomado.
	var l libres
	e.Esperar(200, P{Metodo: "GET", Ruta: "/v1/p/el-corte/libres?servicio=" + e.Servicio + "&fecha=2026-11-02"}, &l)
	if l.Horarios[0].Hora != "09:30" {
		t.Fatalf("las 9 tendrían que estar tomadas: %+v", l.Horarios)
	}

	// Un aviso sin firma no entra.
	req := mpfalso.Aviso("payment", "1001", e.MP.Usuario, e.Ahora)
	req.Header.Set("x-signature", "ts=1,v1=00")
	rec := httptestRecorder(e, req)
	if rec != http.StatusUnauthorized {
		t.Fatalf("aviso con firma inválida: %d", rec)
	}

	// Rechazado: el turno sigue esperando (el comprador puede reintentar hasta que venza).
	e.MP.Pagar(1001, ref, "rejected", 4000)
	if c := e.Avisar("payment", "1001"); c != 200 {
		t.Fatalf("aviso: %d", c)
	}
	if x := e.Turno(tu.ID); x.Estado != "pendiente_pago" {
		t.Fatalf("rechazado no cambia el turno: %s", x.Estado)
	}
	// Aprobado: confirmado, sin vencimiento, y se avisa.
	e.MP.Pagar(1002, ref, "approved", 4000)
	if c := e.Avisar("payment", "1002"); c != 200 {
		t.Fatalf("aviso: %d", c)
	}
	if x := e.Turno(tu.ID); x.Estado != "confirmado" || x.VenceEn != nil {
		t.Fatalf("aprobado: %+v", x)
	}
	if ev := e.Avisos[tu.ID]; len(ev) != 2 || ev[1] != "confirmado" {
		t.Fatalf("avisos del turno: %v", ev)
	}
	// El mismo aviso otra vez no cambia nada; un rechazo posterior no pisa el aprobado.
	e.Avisar("payment", "1002")
	e.Avisar("payment", "1001")
	var ps pagos
	e.Panel(200, "GET", "/v1/panel/pagos", nil, &ps)
	if len(ps.Pagos) != 1 || ps.Pagos[0].Estado != "aprobado" || ps.Pagos[0].Monto != 4000 || *ps.Pagos[0].MPPago != 1002 {
		t.Fatalf("pagos: %+v", ps.Pagos)
	}
	if len(e.Avisos[tu.ID]) != 2 {
		t.Fatalf("se avisó de más: %v", e.Avisos[tu.ID])
	}

	// Devolución parcial y después el resto: el turno se cancela.
	var dev cobros.Pago
	e.Panel(200, "POST", "/v1/panel/pagos/"+ps.Pagos[0].ID+"/devolver", map[string]int64{"monto": 1000}, &dev)
	if dev.Devuelto != 1000 || dev.Estado != "aprobado" {
		t.Fatalf("devolución parcial: %+v", dev)
	}
	e.Panel(400, "POST", "/v1/panel/pagos/"+ps.Pagos[0].ID+"/devolver", map[string]int64{"monto": 5000}, nil)
	e.Panel(200, "POST", "/v1/panel/pagos/"+ps.Pagos[0].ID+"/devolver", nil, &dev)
	if dev.Devuelto != 4000 || dev.Estado != "devuelto" {
		t.Fatalf("devolución total: %+v", dev)
	}
	if d := e.MP.Devoluciones[1002]; len(d) != 2 || d[0] != 1000 || d[1] != 3000 {
		t.Fatalf("devoluciones en Mercado Pago: %v", d)
	}
	if x := e.Turno(tu.ID); x.Estado != "cancelado" {
		t.Fatalf("devuelto entero, el turno se cancela: %s", x.Estado)
	}
	e.Panel(409, "POST", "/v1/panel/pagos/"+ps.Pagos[0].ID+"/devolver", nil, nil)
	// El aviso de la devolución que manda Mercado Pago no lo revive.
	e.Avisar("payment", "1002")
	e.Panel(200, "GET", "/v1/panel/pagos", nil, &ps)
	if ps.Pagos[0].Estado != "devuelto" {
		t.Fatalf("después del aviso de devolución: %+v", ps.Pagos[0])
	}
}

// httptestRecorder manda un pedido armado a mano al handler.
func httptestRecorder(e *cobrosprueba.Entorno, r *http.Request) int {
	w := &grabador{h: http.Header{}}
	e.Handler.ServeHTTP(w, r)
	return w.estado
}

type grabador struct {
	h      http.Header
	estado int
}

func (g *grabador) Header() http.Header { return g.h }
func (g *grabador) Write(b []byte) (int, error) {
	if g.estado == 0 {
		g.estado = 200
	}
	return len(b), nil
}
func (g *grabador) WriteHeader(c int) { g.estado = c }

func TestVencidaLiberaYElPagoTardioNoPisa(t *testing.T) {
	e := cobrosprueba.Armar(t)
	e.ConectarMP()
	e.Panel(200, "PUT", "/v1/panel/cobros", map[string]any{"modo": "fija", "senaFija": 2000}, nil)

	primera := e.Reservar(201, "10:00", "1111", nil)
	if primera.Pago.Monto != 2000 {
		t.Fatalf("seña fija: %+v", primera.Pago)
	}
	// Pasan 16 minutos sin pagar: el horario se libera solo y el link ya no sirve.
	e.Ahora = e.Ahora.Add(16 * time.Minute)
	var l libres
	e.Esperar(200, P{Metodo: "GET", Ruta: "/v1/p/el-corte/libres?servicio=" + e.Servicio + "&fecha=2026-11-02"}, &l)
	libre := false
	for _, h := range l.Horarios {
		libre = libre || h.Hora == "10:00"
	}
	if !libre {
		t.Fatalf("las 10 tendrían que estar libres: %+v", l.Horarios)
	}
	e.Esperar(409, P{Metodo: "POST", Ruta: "/v1/t/" + primera.Codigo + "/pagar"}, nil)

	// Otra persona se lleva las 10; el pago tardío de la primera la cancela y queda para devolver.
	segunda := e.Reservar(201, "10:00", "2222", nil)
	e.MP.Pagar(2001, referencia(t, primera), "approved", 2000)
	if c := e.Avisar("payment", "2001"); c != 200 {
		t.Fatalf("aviso: %d", c)
	}
	if x := e.Turno(primera.ID); x.Estado != "cancelado" {
		t.Fatalf("pagó tarde y sin lugar: %s", x.Estado)
	}
	if x := e.Turno(segunda.ID); x.Estado != "pendiente_pago" {
		t.Fatalf("la segunda no se toca: %s", x.Estado)
	}
	var ps pagos
	e.Panel(200, "GET", "/v1/panel/pagos?estado=aprobado", nil, &ps)
	if len(ps.Pagos) != 1 || ps.Pagos[0].TurnoID != primera.ID || ps.Pagos[0].TurnoEstado != "cancelado" {
		t.Fatalf("el pago aprobado tiene que quedar a la vista para devolver: %+v", ps.Pagos)
	}

	// Si nadie tomó el horario, el pago tardío confirma igual.
	tercera := e.Reservar(201, "11:00", "3333", nil)
	e.Ahora = e.Ahora.Add(20 * time.Minute)
	e.MP.Pagar(2002, referencia(t, tercera), "approved", 2000)
	e.Avisar("payment", "2002")
	if x := e.Turno(tercera.ID); x.Estado != "confirmado" {
		t.Fatalf("tarde pero libre: %s", x.Estado)
	}

	// El cliente cancela una que espera el pago: el pago se anula.
	cuarta := e.Reservar(201, "12:00", "4444", nil)
	e.Esperar(200, P{Metodo: "POST", Ruta: "/v1/t/" + cuarta.Codigo + "/cancelar"}, nil)
	e.Panel(200, "GET", "/v1/panel/pagos?estado=cancelado", nil, &ps)
	if len(ps.Pagos) != 1 || ps.Pagos[0].TurnoID != cuarta.ID {
		t.Fatalf("cancelado por el cliente: %+v", ps.Pagos)
	}
}

func TestRenovacionDeTokens(t *testing.T) {
	e := cobrosprueba.Armar(t)
	e.ConectarMP()
	ctx := context.Background()
	// Faltan 10 días: la pasada de renovación lo renueva.
	e.SQL("UPDATE mp_conexiones SET vence_en = $1", e.Ahora.Add(10*24*time.Hour))
	if r, f, err := e.Cobros.RenovarPorVencer(ctx); err != nil || r != 1 || f != 0 || e.MP.Renovaciones != 1 {
		t.Fatalf("renovación: %d %d %v (%d)", r, f, err, e.MP.Renovaciones)
	}
	// Con el token nuevo se cobra.
	e.Panel(200, "PUT", "/v1/panel/cobros", map[string]any{"modo": "total"}, nil)
	tu := e.Reservar(201, "09:00", "1111", nil)
	if e.MP.TokenDePreferencia[referencia(t, tu)] != "APP_USR-renovado-1" || tu.Pago.Monto != 8000 {
		t.Fatalf("cobró con %q por %d", e.MP.TokenDePreferencia[referencia(t, tu)], tu.Pago.Monto)
	}
	// Si la renovación falla, queda para reconectar y no se cobra más (nunca con otro token).
	e.MP.FallarRenovacion = true
	e.SQL("UPDATE mp_conexiones SET vence_en = $1", e.Ahora.Add(2*24*time.Hour))
	if _, f, _ := e.Cobros.RenovarPorVencer(ctx); f != 1 {
		t.Fatalf("tendría que fallar una: %d", f)
	}
	var c cobros.Conexion
	e.Panel(200, "GET", "/v1/panel/mercadopago", nil, &c)
	if !c.Reautorizar || c.UltimoError == "" {
		t.Fatalf("conexión: %+v", c)
	}
	if x := e.Reservar(201, "10:00", "2222", nil); x.Estado != "pendiente" || x.Pago != nil {
		t.Fatalf("sin token usable no se cobra: %+v", x)
	}
	// Reconectar lo arregla; desconectar lo saca.
	e.ConectarMP()
	e.Panel(204, "DELETE", "/v1/panel/mercadopago", nil, nil)
	e.Panel(200, "GET", "/v1/panel/mercadopago", nil, &c)
	if c.Conectada {
		t.Fatal("sigue conectada")
	}
}

func TestAislamientoEntreNegocios(t *testing.T) {
	e := cobrosprueba.Armar(t)
	e.ConectarMP()
	e.Panel(200, "PUT", "/v1/panel/cobros", map[string]any{"modo": "total"}, nil)
	tu := e.Reservar(201, "09:00", "1111", nil)
	e.MP.Pagar(3001, referencia(t, tu), "approved", 8000)
	e.Avisar("payment", "3001")

	const otra = "user_otra000001"
	var n negocios.Negocio
	e.Esperar(201, P{Metodo: "POST", Ruta: "/v1/negocios", Quien: otra, Cuerpo: map[string]string{"slug": "otro", "nombre": "Otro"}}, &n)
	var mio pagos
	e.Panel(200, "GET", "/v1/panel/pagos", nil, &mio)
	var ajeno pagos
	e.Esperar(200, P{Metodo: "GET", Ruta: "/v1/panel/pagos", Quien: otra, Negocio: n.ID}, &ajeno)
	if len(mio.Pagos) != 1 || len(ajeno.Pagos) != 0 {
		t.Fatalf("pagos: míos %d, del otro %d", len(mio.Pagos), len(ajeno.Pagos))
	}
	// El otro negocio no devuelve un pago ajeno ni ve la cuenta ajena.
	e.Esperar(404, P{Metodo: "POST", Ruta: "/v1/panel/pagos/" + mio.Pagos[0].ID + "/devolver", Quien: otra, Negocio: n.ID}, nil)
	var c cobros.Conexion
	e.Esperar(200, P{Metodo: "GET", Ruta: "/v1/panel/mercadopago", Quien: otra, Negocio: n.ID}, &c)
	if c.Conectada {
		t.Fatal("el otro negocio ve la cuenta de Mercado Pago ajena")
	}
	// Ni entra al panel del primero.
	e.Esperar(403, P{Metodo: "GET", Ruta: "/v1/panel/pagos", Quien: otra, Negocio: e.Negocio}, nil)
	// Un aviso de un vendedor que no es de TengoLibre se ignora (200, nada cambia).
	if c := mpfalso.Avisar(e.Handler, "payment", "3001", 999, e.Ahora); c != 200 {
		t.Fatalf("aviso ajeno: %d", c)
	}
	// Un recepcionista no configura cobros.
	e.Panel(201, "POST", "/v1/panel/equipo", map[string]string{"clerkId": "user_recepcion01", "rol": "recepcion"}, nil)
	e.Esperar(403, P{Metodo: "PUT", Ruta: "/v1/panel/cobros", Quien: "user_recepcion01", Negocio: e.Negocio,
		Cuerpo: map[string]any{"modo": "sin_cobro"}}, nil)
}
