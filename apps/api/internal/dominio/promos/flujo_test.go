package promos_test

import (
	"strings"
	"testing"
	"time"

	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/cobros/cobrosprueba"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/negocios"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/promos"
)

type P = cobrosprueba.P

func TestCuponAplicadoEnLaReserva(t *testing.T) {
	e := cobrosprueba.Armar(t)
	e.ConectarMP()
	e.Panel(200, "PUT", "/v1/panel/cobros", map[string]any{"modo": "total"}, nil)

	var c promos.Cupon
	e.Panel(201, "POST", "/v1/panel/cupones", map[string]any{"codigo": "desc20", "tipo": "porcentaje", "valor": 20, "maxUsos": 1}, &c)
	if c.Codigo != "DESC20" || !c.Activo {
		t.Fatalf("cupón: %+v", c)
	}
	e.Panel(409, "POST", "/v1/panel/cupones", map[string]any{"codigo": "DESC20", "tipo": "fijo", "valor": 100}, nil)
	e.Panel(400, "POST", "/v1/panel/cupones", map[string]any{"codigo": "MAL", "tipo": "porcentaje", "valor": 150}, nil)

	// Validar desde la página: sirve y dice cuánto descuenta del servicio.
	var v promos.Validacion
	e.Esperar(200, P{Metodo: "POST", Ruta: "/v1/p/el-corte/cupones/validar", Cuerpo: map[string]string{"codigo": " Desc20 ", "servicioId": e.Servicio}}, &v)
	if !v.Valido || v.Descuento != 1600 {
		t.Fatalf("validación: %+v", v)
	}

	// Reservar con el cupón: el precio y el cobro bajan.
	tu := e.Reservar(201, "09:00", "1111", map[string]any{"cupon": "desc20"})
	if tu.Precio != 6400 || tu.Descuento != 1600 || tu.Pago == nil || tu.Pago.Monto != 6400 {
		t.Fatalf("reserva con cupón: %+v %+v", tu, tu.Pago)
	}
	if x := e.Turno(tu.ID); x.Precio != 6400 {
		t.Fatalf("el precio guardado: %d", x.Precio)
	}
	ref := strings.TrimPrefix(tu.Pago.URL, "https://mp.falso/checkout/")
	if e.MP.Ordenes[ref].Pedido.TotalAmount != "6400.00" {
		t.Fatalf("Mercado Pago cobra %s", e.MP.Ordenes[ref].Pedido.TotalAmount)
	}

	// Ya se usó la única vez: otra reserva con el cupón falla entera (el horario sigue libre).
	e.Reservar(422, "10:00", "2222", map[string]any{"cupon": "DESC20"})
	e.Esperar(200, P{Metodo: "POST", Ruta: "/v1/p/el-corte/cupones/validar", Cuerpo: map[string]string{"codigo": "DESC20"}}, &v)
	if v.Valido || v.Motivo != "Cupón agotado" {
		t.Fatalf("agotado: %+v", v)
	}
	if x := e.Reservar(201, "10:00", "2222", nil); x.Precio != 8000 {
		t.Fatalf("sin cupón: %d", x.Precio)
	}

	// Mínimo y vencimiento.
	e.Panel(201, "POST", "/v1/panel/cupones", map[string]any{"codigo": "GRANDE", "tipo": "fijo", "valor": 500, "minimo": 10000}, nil)
	e.Esperar(200, P{Metodo: "POST", Ruta: "/v1/p/el-corte/cupones/validar", Cuerpo: map[string]string{"codigo": "grande", "servicioId": e.Servicio}}, &v)
	if v.Valido || v.Motivo != "No aplica para este servicio" {
		t.Fatalf("mínimo: %+v", v)
	}
	var viejo promos.Cupon
	e.Panel(201, "POST", "/v1/panel/cupones", map[string]any{"codigo": "VIEJO", "tipo": "fijo", "valor": 500,
		"desde": e.Ahora.Add(-48 * time.Hour), "hasta": e.Ahora.Add(-time.Hour)}, &viejo)
	e.Reservar(422, "11:00", "3333", map[string]any{"cupon": "viejo"})
	// Se reactiva extendiéndolo.
	e.Panel(200, "PATCH", "/v1/panel/cupones/"+viejo.ID, map[string]any{"hasta": e.Ahora.Add(24 * time.Hour)}, nil)
	if x := e.Reservar(201, "11:00", "3333", map[string]any{"cupon": "viejo"}); x.Precio != 7500 {
		t.Fatalf("fijo: %d", x.Precio)
	}
	// Uno usado no se borra: se desactiva.
	e.Panel(204, "DELETE", "/v1/panel/cupones/"+viejo.ID, nil, nil)
	var l struct {
		Cupones []promos.Cupon `json:"cupones"`
	}
	e.Panel(200, "GET", "/v1/panel/cupones", nil, &l)
	for _, x := range l.Cupones {
		if x.ID == viejo.ID && (x.Activo || x.Usos != 1) {
			t.Fatalf("borrado de uno usado: %+v", x)
		}
	}

	// Otro negocio no ve ni valida los cupones de este.
	const otra = "user_otra000001"
	var n negocios.Negocio
	e.Esperar(201, P{Metodo: "POST", Ruta: "/v1/negocios", Quien: otra, Cuerpo: map[string]string{"slug": "otro", "nombre": "Otro"}}, &n)
	e.Esperar(200, P{Metodo: "GET", Ruta: "/v1/panel/cupones", Quien: otra, Negocio: n.ID}, &l)
	if len(l.Cupones) != 0 {
		t.Fatalf("el otro negocio ve %d cupones", len(l.Cupones))
	}
	e.Esperar(200, P{Metodo: "POST", Ruta: "/v1/p/otro/cupones/validar", Cuerpo: map[string]string{"codigo": "GRANDE"}}, &v)
	if v.Valido {
		t.Fatal("un cupón vale en otro negocio")
	}
	e.Esperar(404, P{Metodo: "PATCH", Ruta: "/v1/panel/cupones/" + viejo.ID, Quien: otra, Negocio: n.ID, Cuerpo: map[string]any{"activo": true}}, nil)
}

func TestReferidos(t *testing.T) {
	e := cobrosprueba.Armar(t)
	ana := e.Reservar(201, "09:00", "1111", nil)
	// Sin programa no hay código.
	e.Esperar(409, P{Metodo: "POST", Ruta: "/v1/t/" + ana.Codigo + "/referido"}, nil)
	e.Panel(200, "PUT", "/v1/panel/referidos/config", map[string]any{"activo": true, "tipo": "fijo", "valor": 1000, "tope": 2}, nil)
	e.Panel(400, "PUT", "/v1/panel/referidos/config", map[string]any{"activo": true, "tipo": "fijo"}, nil)

	var mio promos.MiReferido
	e.Esperar(200, P{Metodo: "POST", Ruta: "/v1/t/" + ana.Codigo + "/referido"}, &mio)
	if mio.Codigo == nil || len(*mio.Codigo) != promos.LargoReferido {
		t.Fatalf("código: %+v", mio)
	}
	codigo := *mio.Codigo
	// Pedirlo de nuevo devuelve el mismo.
	e.Esperar(200, P{Metodo: "POST", Ruta: "/v1/t/" + ana.Codigo + "/referido"}, &mio)
	if *mio.Codigo != codigo {
		t.Fatal("cambió el código")
	}
	var ok map[string]bool
	e.Esperar(200, P{Metodo: "POST", Ruta: "/v1/p/el-corte/referidos/validar", Cuerpo: map[string]string{"codigo": strings.ToLower(codigo)}}, &ok)
	if !ok["valido"] {
		t.Fatal("el código no valida")
	}

	// Beto reserva con el código de Ana: Ana gana un cupón cuando el turno de Beto queda confirmado (no
	// al reservar). Ana con su propio código, o Beto otra vez, no cuentan.
	beto := e.Reservar(201, "09:30", "2222", map[string]any{"referido": codigo})
	e.Reservar(201, "10:00", "1111", map[string]any{"referido": codigo})
	e.Reservar(201, "10:30", "2222", map[string]any{"referido": codigo})
	e.Esperar(200, P{Metodo: "GET", Ruta: "/v1/t/" + ana.Codigo + "/referido"}, &mio)
	if mio.Referidos != 1 || len(mio.Recompensas) != 0 {
		t.Fatalf("con el turno de Beto pendiente no hay premio: %+v", mio)
	}
	e.Panel(200, "PATCH", "/v1/panel/turnos/"+beto.ID, map[string]string{"estado": "confirmado"}, nil)
	e.Esperar(200, P{Metodo: "GET", Ruta: "/v1/t/" + ana.Codigo + "/referido"}, &mio)
	if mio.Referidos != 1 || len(mio.Recompensas) != 1 {
		t.Fatalf("después de Beto: %+v", mio)
	}
	premio := mio.Recompensas[0]
	if !strings.HasPrefix(premio.Codigo, "REF-"+codigo+"-") || premio.Valor != 1000 || premio.MaxUsos == nil || *premio.MaxUsos != 1 ||
		premio.Hasta == nil || !premio.Hasta.Equal(e.Ahora.AddDate(0, 0, 90)) {
		t.Fatalf("recompensa: %+v", premio)
	}
	// Carla suma la segunda (el premio, cuando venga); Dani ya pasa el tope.
	carla := e.Reservar(201, "11:00", "3333", map[string]any{"referido": codigo})
	e.Reservar(201, "11:30", "4444", map[string]any{"referido": codigo})
	e.Esperar(200, P{Metodo: "GET", Ruta: "/v1/t/" + ana.Codigo + "/referido"}, &mio)
	if mio.Referidos != 2 || len(mio.Recompensas) != 1 {
		t.Fatalf("tope: %+v", mio)
	}
	// Un código inventado se ignora (la reserva sale igual).
	e.Reservar(201, "12:00", "5555", map[string]any{"referido": "NOEXISTE"})

	// Ana usa su premio.
	conPremio := e.Reservar(201, "12:30", "1111", map[string]any{"cupon": premio.Codigo})
	if conPremio.Precio != 7000 {
		t.Fatalf("premio: %d", conPremio.Precio)
	}
	var l struct {
		Referidos []promos.Referido `json:"referidos"`
	}
	e.Panel(200, "GET", "/v1/panel/referidos", nil, &l)
	if len(l.Referidos) != 1 || l.Referidos[0].Usos != 2 || l.Referidos[0].Cliente != "Cliente 1111" {
		t.Fatalf("panel: %+v", l.Referidos)
	}
	// Cada uso con su premio: el de Beto ya se usó; Carla tiene el turno por delante.
	var u struct {
		Usos []promos.UsoReferido `json:"usos"`
	}
	e.Panel(200, "GET", "/v1/panel/referidos/usos", nil, &u)
	if len(u.Usos) != 2 || u.Usos[0].Referido != "Cliente 3333" || u.Usos[0].Estado != "turno" || u.Usos[0].TurnoInicio == nil ||
		u.Usos[1].Referido != "Cliente 2222" || u.Usos[1].Refiere != "Cliente 1111" || u.Usos[1].Estado != "premio_usado" {
		t.Fatalf("usos: %+v", u.Usos)
	}

	// Carla cancela: su lugar vuelve (Eva entra en el tope) y no hay premio. Ana cancela el turno con
	// el premio: el cupón vuelve a servir.
	e.Esperar(200, P{Metodo: "POST", Ruta: "/v1/t/" + carla.Codigo + "/cancelar"}, nil)
	e.Esperar(200, P{Metodo: "POST", Ruta: "/v1/t/" + conPremio.Codigo + "/cancelar"}, nil)
	e.Esperar(200, P{Metodo: "GET", Ruta: "/v1/t/" + ana.Codigo + "/referido"}, &mio)
	if mio.Referidos != 1 || len(mio.Recompensas) != 1 || mio.Recompensas[0].Usos != 0 {
		t.Fatalf("después de cancelar: %+v", mio)
	}
	eva := e.Reservar(201, "11:00", "6666", map[string]any{"referido": codigo})
	e.Panel(200, "PATCH", "/v1/panel/turnos/"+eva.ID, map[string]string{"estado": "atendido"}, nil)
	e.Esperar(200, P{Metodo: "GET", Ruta: "/v1/t/" + ana.Codigo + "/referido"}, &mio)
	if mio.Referidos != 2 || len(mio.Recompensas) != 2 {
		t.Fatalf("Eva entra en el lugar de Carla: %+v", mio)
	}
	if x := e.Reservar(201, "12:30", "1111", map[string]any{"cupon": premio.Codigo}); x.Precio != 7000 {
		t.Fatalf("el premio devuelto vuelve a servir: %d", x.Precio)
	}
}

// TestTelefonoDeOtroCliente: quien reserva desde la web con el teléfono de una clienta que ya tiene su
// email de confianza no ve sus datos (ni su id) en la respuesta ni en el link, y el link no muestra ni
// da su código de referidos. La dueña real, desde su propio turno, sí.
func TestTelefonoDeOtroCliente(t *testing.T) {
	e := cobrosprueba.Armar(t)
	e.Panel(200, "PUT", "/v1/panel/referidos/config", map[string]any{"activo": true, "tipo": "fijo", "valor": 1000}, nil)
	var suyo struct {
		ID     string `json:"id"`
		Codigo string `json:"codigo"`
	}
	e.Panel(201, "POST", "/v1/panel/turnos", map[string]any{"servicioId": e.Servicio, "profesionalId": e.Profesional,
		"inicio":  cobrosprueba.En("2026-11-02 09:00").Format(time.RFC3339),
		"cliente": map[string]string{"nombre": "Verónica Paz", "telefono": "7777", "email": "vero@ejemplo.com"}}, &suyo)
	var mio promos.MiReferido
	e.Esperar(200, P{Metodo: "POST", Ruta: "/v1/t/" + suyo.Codigo + "/referido"}, &mio)
	if mio.Codigo == nil || mio.Oculto {
		t.Fatalf("la dueña ve su código: %+v", mio)
	}

	var otro map[string]any
	e.Esperar(201, P{Metodo: "POST", Ruta: "/v1/p/el-corte/turnos", Cuerpo: map[string]any{"servicioId": e.Servicio,
		"profesionalId": e.Profesional, "inicio": cobrosprueba.En("2026-11-02 10:00").Format(time.RFC3339),
		"cliente": map[string]string{"nombre": "Intruso", "telefono": "7777", "email": "intruso@ejemplo.com"}}}, &otro)
	c := otro["cliente"].(map[string]any)
	if c["nombre"] != "Intruso" || c["email"] != "intruso@ejemplo.com" || c["id"] != nil {
		t.Fatalf("la reserva devolvió datos que no tipeó: %+v", c)
	}
	var link map[string]any
	e.Esperar(200, P{Metodo: "GET", Ruta: "/v1/t/" + otro["codigo"].(string)}, &link)
	if c := link["cliente"].(map[string]any); c["nombre"] != "Verónica" || c["email"] != "ve***@ejemplo.com" || c["telefono"] != "****" ||
		c["id"] != nil {
		t.Fatalf("el link muestra los datos de la clienta: %+v", c)
	}
	e.Esperar(200, P{Metodo: "GET", Ruta: "/v1/t/" + otro["codigo"].(string) + "/referido"}, &mio)
	if mio.Codigo != nil || !mio.Oculto || len(mio.Recompensas) != 0 {
		t.Fatalf("el link ajeno muestra el código: %+v", mio)
	}
	e.Esperar(409, P{Metodo: "POST", Ruta: "/v1/t/" + otro["codigo"].(string) + "/referido"}, nil)
	// El email de confianza no se pisó.
	var email string
	e.Fila("SELECT email FROM clientes WHERE telefono = '7777'", nil, &email)
	if email != "vero@ejemplo.com" {
		t.Fatalf("email: %q", email)
	}
}

// TestTopeDeValidaciones: probar códigos desde la página tiene tope por IP (cupones y referidos suman).
func TestTopeDeValidaciones(t *testing.T) {
	e := cobrosprueba.Armar(t)
	for i := range promos.ValidarPorIP {
		ruta := "/v1/p/el-corte/cupones/validar"
		if i%2 == 1 {
			ruta = "/v1/p/el-corte/referidos/validar"
		}
		e.Esperar(200, P{Metodo: "POST", Ruta: ruta, Cuerpo: map[string]string{"codigo": "PRUEBA" + strings.Repeat("X", i%5)}}, nil)
	}
	e.Esperar(429, P{Metodo: "POST", Ruta: "/v1/p/el-corte/referidos/validar", Cuerpo: map[string]string{"codigo": "OTRO"}}, nil)
	e.Esperar(429, P{Metodo: "POST", Ruta: "/v1/p/el-corte/cupones/validar", Cuerpo: map[string]string{"codigo": "OTRO"}}, nil)
}
