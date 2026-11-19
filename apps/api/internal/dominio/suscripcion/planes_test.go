package suscripcion_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/catalogo"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/cobros/cobrosprueba"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/planes"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/suscripcion"
	"github.com/nawesan12/tengolibre/apps/api/internal/integraciones/mercadopago"
	"github.com/nawesan12/tengolibre/apps/api/internal/integraciones/mercadopago/mpfalso"
)

// Los topes de cada plan se cumplen en el servidor y bajar de plan no borra nada.

type tope struct {
	Codigo  string `json:"code"`
	Plan    string `json:"plan"`
	Limite  int    `json:"limit"`
	Detalle string `json:"detail"`
}

func profesional(e *cobrosprueba.Entorno, estado int, nombre string, activo bool) (catalogo.Profesional, tope) {
	e.T.Helper()
	var p catalogo.Profesional
	var t tope
	r := e.Panel(estado, "POST", "/v1/panel/profesionales", map[string]any{"nombre": nombre, "activo": activo, "servicios": []string{e.Servicio},
		"horario": []map[string]any{{"dia": 1, "desde": "09:00", "hasta": "13:00"}}}, nil)
	if estado == 201 {
		r.Leer(e.T, &p)
	} else {
		r.Leer(e.T, &t)
	}
	return p, t
}

func listar(e *cobrosprueba.Entorno) []catalogo.Profesional {
	e.T.Helper()
	var l struct {
		Profesionales []catalogo.Profesional `json:"profesionales"`
	}
	e.Panel(200, "GET", "/v1/panel/profesionales", nil, &l)
	return l.Profesionales
}

func enPlan(ps []catalogo.Profesional) (n int) {
	for _, p := range ps {
		if p.EnPlan {
			n++
		}
	}
	return n
}

func publicos(e *cobrosprueba.Entorno) int {
	e.T.Helper()
	var pg struct {
		Profesionales []struct {
			ID string `json:"id"`
		} `json:"profesionales"`
	}
	e.Esperar(200, cobrosprueba.P{Metodo: "GET", Ruta: "/v1/p/el-corte"}, &pg)
	return len(pg.Profesionales)
}

func TestTopeDeProfesionalesYBajarSinPerder(t *testing.T) {
	e := cobrosprueba.Armar(t)
	// Inicial: hasta 5 activos (Ana es la primera). El sexto pide Equipo; uno inactivo entra igual.
	e.PonerPlan(e.Negocio, "inicial")
	for i := 2; i <= 5; i++ {
		profesional(e, 201, fmt.Sprintf("Pro %d", i), true)
	}
	_, tp := profesional(e, 402, "Pro 6", true)
	if tp.Codigo != "tope_plan" || tp.Plan != "equipo" || tp.Limite != 5 {
		t.Fatalf("el sexto con Inicial: %+v", tp)
	}
	inactivo, _ := profesional(e, 201, "Suplente", false)
	// Activar al inactivo también ocupa lugar.
	inactivo.Activo = true
	e.Panel(402, "PATCH", "/v1/panel/profesionales/"+inactivo.ID, inactivo, nil)

	// Equipo: hasta 12. Se suman 7 más (12 activos).
	e.PonerPlan(e.Negocio, "equipo")
	e.Panel(200, "PATCH", "/v1/panel/profesionales/"+inactivo.ID, inactivo, nil)
	for i := 7; i <= 12; i++ {
		profesional(e, 201, fmt.Sprintf("Pro %d", i), true)
	}
	_, tp = profesional(e, 402, "Pro 13", true)
	if tp.Plan != "centro" || tp.Limite != 12 {
		t.Fatalf("el 13 con Equipo: %+v", tp)
	}
	if ps := listar(e); len(ps) != 12 || enPlan(ps) != 12 || publicos(e) != 12 {
		t.Fatalf("con Equipo: %d, %d en el plan", len(ps), enPlan(ps))
	}

	// Bajar a Inicial no borra a nadie: los 12 siguen en el panel, 5 (los más antiguos) toman turnos y
	// la página pública muestra esos 5.
	e.PonerPlan(e.Negocio, "inicial")
	ps := listar(e)
	if len(ps) != 12 || enPlan(ps) != 5 || publicos(e) != 5 {
		t.Fatalf("después de bajar: %d en el panel, %d en el plan, %d en la página", len(ps), enPlan(ps), publicos(e))
	}
	var fuera catalogo.Profesional
	for _, p := range ps {
		if p.Activo && !p.EnPlan {
			fuera = p
			break
		}
	}
	// Uno fuera del plan no toma turnos (ni desde la página ni desde el panel).
	e.Esperar(409, cobrosprueba.P{Metodo: "POST", Ruta: "/v1/p/el-corte/turnos", Cuerpo: map[string]any{
		"servicioId": e.Servicio, "profesionalId": fuera.ID, "inicio": cobrosprueba.En("2026-11-02 09:00").Format(time.RFC3339),
		"cliente": map[string]string{"nombre": "Cliente", "telefono": "1111"}}}, nil)
	// Liberar lugar (desactivar a una de las de adentro) hace entrar a la siguiente.
	ana := ps[0]
	for _, p := range ps {
		if p.ID == e.Profesional {
			ana = p
		}
	}
	ana.Activo = false
	e.Panel(200, "PATCH", "/v1/panel/profesionales/"+ana.ID, ana, nil)
	if ps := listar(e); enPlan(ps) != 5 || len(ps) != 12 {
		t.Fatalf("liberando lugar: %d en el plan", enPlan(ps))
	}
	// Volver a subir: vuelven todas.
	e.PonerPlan(e.Negocio, "equipo")
	if ps := listar(e); enPlan(ps) != 11 || publicos(e) != 11 {
		t.Fatalf("subiendo de nuevo: %d en el plan", enPlan(ps))
	}
}

// El tope de la base (app.tope_profesionales) es el mismo que el de planes.
func TestTopesDeLaBaseIgualesAPlanes(t *testing.T) {
	e := cobrosprueba.Armar(t)
	for _, p := range planes.Planes {
		e.PonerPlan(e.Negocio, p.Clave)
		e.SQL("UPDATE negocios SET profesionales_extra = 3 WHERE id = $1", e.Negocio)
		var base, cupo int
		var v planes.Vigente
		err := e.Bases.App.EnNegocio(context.Background(), e.Negocio, func(tx pgx.Tx) (err error) {
			if err = tx.QueryRow(context.Background(), "SELECT app.tope_profesionales(now())").Scan(&base); err != nil {
				return err
			}
			if cupo, _, err = planes.CupoIA(context.Background(), tx, time.Now()); err != nil {
				return err
			}
			v, err = planes.Leer(context.Background(), tx, time.Now())
			return err
		})
		if err != nil {
			t.Fatal(err)
		}
		if v.Clave != p.Clave || base != v.TopeProfesionales() || cupo != p.CupoIA {
			t.Fatalf("%s: base %d, planes %d (%s)", p.Clave, base, v.TopeProfesionales(), v.Clave)
		}
	}
}

func TestLaPruebaEsDeCatorceDiasConEquipo(t *testing.T) {
	e := cobrosprueba.Armar(t)
	var dias float64
	err := e.Bases.App.EnNegocio(context.Background(), e.Negocio, func(tx pgx.Tx) error {
		return tx.QueryRow(context.Background(), "SELECT extract(epoch FROM prueba_hasta - creado_en) / 86400 FROM negocios").Scan(&dias)
	})
	if err != nil || dias < 13.99 || dias > 14.01 {
		t.Fatalf("la prueba dura %v días (%v)", dias, err)
	}
	s := leer(e)
	if s.Plan != "equipo" || s.NombrePlan != "Equipo" {
		t.Fatalf("en prueba rige Equipo: %+v", s.Acceso)
	}
	// Lo de Equipo anda en la prueba; lo de Centro no.
	e.Panel(201, "POST", "/v1/panel/cupones", map[string]any{"codigo": "PRUEBA", "tipo": "fijo", "valor": 100}, nil)
	var tp tope
	e.Panel(402, "GET", "/v1/panel/reportes/turnos.csv", nil, &tp)
	if tp.Codigo != "tope_plan" || tp.Plan != "centro" {
		t.Fatalf("CSV en la prueba: %+v", tp)
	}
	// El catálogo: tres planes, Equipo destacado, anual de 10 meses.
	var cat struct {
		Catalogo []planes.Plan      `json:"catalogo"`
		Extras   suscripcion.Extras `json:"extras"`
		Uso      suscripcion.Uso    `json:"uso"`
	}
	e.Panel(200, "GET", "/v1/panel/suscripcion", nil, &cat)
	if len(cat.Catalogo) != 3 || cat.Catalogo[0].Precio != 6900 || cat.Catalogo[1].Precio != 14900 || !cat.Catalogo[1].Destacado ||
		cat.Catalogo[2].Precio != 29900 || cat.Catalogo[2].PrecioAnual != 299000 || cat.Extras.PackWhatsApp.Precio != 5900 ||
		cat.Extras.DiasDePrueba != 14 || cat.Uso.TopeProfesionales != 12 || cat.Uso.ProfesionalesActivos != 1 || cat.Uso.CupoIA != 400 {
		t.Fatalf("catálogo: %+v", cat)
	}
}

func TestBajarDePlanDejaEnSoloLecturaLoQueNoIncluye(t *testing.T) {
	e := cobrosprueba.Armar(t)
	e.PonerPlan(e.Negocio, "equipo")
	var c struct {
		ID string `json:"id"`
	}
	e.Panel(201, "POST", "/v1/panel/cupones", map[string]any{"codigo": "HOLA", "tipo": "fijo", "valor": 100}, &c)
	e.Panel(200, "PUT", "/v1/panel/referidos/config", map[string]any{"activo": false}, nil)
	e.PonerPlan(e.Negocio, "inicial")
	// Nada se borra: el cupón se sigue viendo; editarlo o crear otro pide Equipo; borrarlo, sí.
	var l struct {
		Cupones []struct {
			ID string `json:"id"`
		} `json:"cupones"`
	}
	e.Panel(200, "GET", "/v1/panel/cupones", nil, &l)
	if len(l.Cupones) != 1 {
		t.Fatalf("el cupón sigue: %+v", l)
	}
	e.Panel(402, "PATCH", "/v1/panel/cupones/"+c.ID, map[string]any{"valor": 200}, nil)
	// En la página, el cupón no se aplica con Inicial.
	var v struct {
		Valido bool `json:"valido"`
	}
	e.Esperar(200, cobrosprueba.P{Metodo: "POST", Ruta: "/v1/p/el-corte/cupones/validar", Cuerpo: map[string]any{"codigo": "HOLA"}}, &v)
	if v.Valido {
		t.Fatal("con Inicial el cupón no vale")
	}
	e.Reservar(422, "09:00", "1111", map[string]any{"cupon": "HOLA"})
	e.Reservar(201, "09:00", "1111", nil)
	// Volver a Equipo: todo como estaba.
	e.PonerPlan(e.Negocio, "equipo")
	e.Esperar(200, cobrosprueba.P{Metodo: "POST", Ruta: "/v1/p/el-corte/cupones/validar", Cuerpo: map[string]any{"codigo": "HOLA"}}, &v)
	if !v.Valido {
		t.Fatal("con Equipo el cupón vuelve")
	}
	e.Panel(204, "DELETE", "/v1/panel/cupones/"+c.ID, nil, nil)
}

func TestSenaSoloConEquipo(t *testing.T) {
	e := cobrosprueba.Armar(t)
	e.ConectarMP()
	e.Panel(200, "PUT", "/v1/panel/cobros", map[string]any{"modo": "porcentaje", "senaPorcentaje": 50}, nil)
	if x := e.Reservar(201, "09:00", "1111", nil); x.Estado != "pendiente_pago" {
		t.Fatalf("con Equipo (prueba) cobra seña: %+v", x.Estado)
	}
	e.PonerPlan(e.Negocio, "inicial")
	if x := e.Reservar(201, "10:00", "2222", nil); x.Estado == "pendiente_pago" || x.Pago != nil {
		t.Fatalf("con Inicial no cobra seña: %+v", x)
	}
	var c struct {
		Modo string `json:"modo"`
	}
	e.Esperar(200, cobrosprueba.P{Metodo: "GET", Ruta: "/v1/p/el-corte/cobro"}, &c)
	if c.Modo != "sin_cobro" {
		t.Fatalf("la página no ofrece seña con Inicial: %+v", c)
	}
	// La configuración quedó guardada.
	e.Panel(200, "GET", "/v1/panel/cobros", nil, &c)
	if c.Modo != "porcentaje" {
		t.Fatalf("config guardada: %+v", c)
	}
}

// cobro aprueba un cobro del preapproval id por monto (lo que manda Mercado Pago cada período).
func cobro(e *cobrosprueba.Entorno, id, cobroID string, monto int64, proximo time.Time) {
	e.T.Helper()
	e.MP.Autorizar(id, "authorized", &proximo)
	debito := e.Ahora
	c := mercadopago.CobroAutorizado{PreapprovalID: id, TransactionAmount: float64(monto), DebitDate: &debito}
	c.Payment.Status = "approved"
	e.MP.Cobros[cobroID] = c
	if s := e.Avisar("subscription_authorized_payment", cobroID); s != 200 {
		e.T.Fatalf("aviso del cobro: %d", s)
	}
}

func preapproval(url string) string { return strings.TrimPrefix(url, "https://mp.falso/suscripcion/") }

type respuestaCambio struct {
	suscripcion.Suscripcion
	URL     string `json:"url"`
	Hoy     int64  `json:"hoy"`
	Despues int64  `json:"despues"`
}

func TestAnualCentroYExtras(t *testing.T) {
	e := cobrosprueba.Armar(t)
	var alta struct {
		URL string `json:"url"`
	}
	e.Panel(400, "POST", "/v1/panel/suscripcion", map[string]string{"plan": "centro", "periodo": "bienal"}, nil)
	e.Panel(400, "POST", "/v1/panel/suscripcion", map[string]string{"plan": "completo"}, nil)
	e.Panel(201, "POST", "/v1/panel/suscripcion", map[string]string{"plan": "centro", "periodo": "anual"}, &alta)
	id := preapproval(alta.URL)
	// Anual: 10 meses (299.000) cada 12.
	if e.MP.Montos[id] != 299000 || e.MP.Frecuencias[id] != 12 {
		t.Fatalf("anual: %d cada %d meses", e.MP.Montos[id], e.MP.Frecuencias[id])
	}
	finPrueba := e.Ahora.AddDate(0, 0, 14)
	e.MP.Autorizar(id, "authorized", &finPrueba)
	e.Avisar("subscription_preapproval", id)
	// Suscripto a Centro durante la prueba, sin nada cobrado: rige Equipo (la prueba es con Equipo).
	if s := leer(e); s.Plan != "equipo" || s.Suscripcion.Plan != "centro" || s.Suscripcion.Periodo != "anual" {
		t.Fatalf("centro anual en la prueba: %+v %+v", s.Acceso, s.Suscripcion)
	}
	// Cancelarla antes del primer cobro no deja Centro gratis.
	// (Con el primer cobro, Centro rige aunque quede prueba.)
	proximo := e.Ahora.AddDate(1, 0, 0)
	cobro(e, id, "7001", 299000, proximo)
	if s := leer(e); s.Plan != "centro" || s.Suscripcion.Monto != 299000 {
		t.Fatalf("centro anual cobrado: %+v %+v", s.Acceso, s.Suscripcion)
	}
	// Extras: 2 profesionales más ($1.900 c/u por mes). Con el año recién pagado se cobra ya la
	// diferencia por lo que queda (2 × 1.900 × 10 = 38.000) y el tope sube recién cuando se acredita.
	var x respuestaCambio
	e.Panel(400, "PUT", "/v1/panel/suscripcion/extras", map[string]int{"cantidad": -1}, nil)
	e.Panel(200, "PUT", "/v1/panel/suscripcion/extras", map[string]int{"cantidad": 2}, &x)
	if x.URL == "" || x.Hoy != 38000 || x.Despues != 337000 || e.MP.Montos[id] != 299000 {
		t.Fatalf("extras: %+v %d", x, e.MP.Montos[id])
	}
	uso := func() suscripcion.Uso {
		var u struct {
			Uso suscripcion.Uso `json:"uso"`
		}
		e.Panel(200, "GET", "/v1/panel/suscripcion", nil, &u)
		return u.Uso
	}
	if u := uso(); u.TopeProfesionales != 25 {
		t.Fatalf("sin pagar la diferencia el tope no sube: %+v", u)
	}
	ref := strings.TrimPrefix(x.URL, "https://mp.falso/checkout/")
	orden := e.MP.PagarOrden(ref, "processed", "accredited", 38000, 31)
	mpfalso.Avisar(e.Handler, "order", orden, 999, e.Ahora)
	if u := uso(); u.TopeProfesionales != 27 || u.WhatsAppIncluidos != 150 || u.CupoIA != 1500 || e.MP.Montos[id] != 337000 {
		t.Fatalf("uso con Centro y extras: %+v %d", u, e.MP.Montos[id])
	}
	// Bajar a Equipo (mismo período): el monto del preapproval baja ya, pero Centro (pagado) sigue
	// hasta el próximo cobro.
	e.Panel(200, "POST", "/v1/panel/suscripcion/cambiar", map[string]string{"plan": "equipo"}, &x)
	if x.Plan != "centro" || x.PlanSiguiente == nil || *x.PlanSiguiente != "equipo" || x.Monto != 149000 || e.MP.Montos[id] != 149000 {
		t.Fatalf("a equipo: %+v", x)
	}
	if s := leer(e); s.Plan != "centro" {
		t.Fatalf("lo pagado se usa entero: %+v", s.Acceso)
	}
	// Arrepentirse: volver a Centro deshace la baja (sin cobrar nada).
	x = respuestaCambio{}
	e.Panel(200, "POST", "/v1/panel/suscripcion/cambiar", map[string]string{"plan": "centro"}, &x)
	if x.PlanSiguiente != nil || x.Monto != 337000 || e.MP.Montos[id] != 337000 || x.URL != "" {
		t.Fatalf("deshacer la baja: %+v", x)
	}
	e.Panel(200, "POST", "/v1/panel/suscripcion/cambiar", map[string]string{"plan": "equipo"}, &x)
	// El cobro del año siguiente es el de Equipo: ahí rige.
	e.Ahora = proximo.Add(time.Hour)
	cobro(e, id, "7002", 149000, proximo.AddDate(1, 0, 0))
	if s := leer(e); s.Plan != "equipo" || s.Suscripcion.Plan != "equipo" || s.Suscripcion.PlanSiguiente != nil || s.Suscripcion.ProfesionalesExtra != 0 {
		t.Fatalf("equipo desde el cobro: %+v %+v", s.Acceso, s.Suscripcion)
	}
	e.Panel(409, "PUT", "/v1/panel/suscripcion/extras", map[string]int{"cantidad": 1}, nil)
	// Pasar a mensual: un alta nueva que empieza cuando vence lo pagado y reemplaza a la anual. Su
	// plan rige desde su primer cobro.
	var nueva respuestaCambio
	e.Panel(200, "POST", "/v1/panel/suscripcion/cambiar", map[string]string{"plan": "centro", "periodo": "mensual"}, &nueva)
	nid := preapproval(nueva.URL)
	if e.MP.Montos[nid] != 29900 || e.MP.Frecuencias[nid] != 1 || nueva.Hoy != 0 {
		t.Fatalf("mensual nueva: %d cada %d", e.MP.Montos[nid], e.MP.Frecuencias[nid])
	}
	fin := proximo.AddDate(1, 0, 0)
	e.MP.Autorizar(nid, "authorized", &fin)
	e.Avisar("subscription_preapproval", nid)
	if s := leer(e); s.Plan != "equipo" || e.MP.Preaprobacion[id].Status != "cancelled" {
		t.Fatalf("pasar a Centro mensual no da Centro antes de cobrarlo: %+v", s.Acceso)
	}
	// Cancelar antes del primer cobro: sigue lo pagado (Equipo) hasta que vence; nunca Centro.
	e.Panel(204, "POST", "/v1/panel/suscripcion/cancelar", nil, nil)
	if s := leer(e); s.Plan != "equipo" || s.Bloqueado {
		t.Fatalf("cancelada antes del primer cobro: %+v", s.Acceso)
	}
}

// Subir de plan a mitad de mes cobra ya la diferencia; sin pagarla no cambia nada. Subir, usar y
// bajar antes del cobro ya no deja usar Centro pagando Inicial.
func TestSubirCobraLaDiferencia(t *testing.T) {
	e := cobrosprueba.Armar(t)
	e.SQL("UPDATE negocios SET prueba_hasta = NULL")
	var alta struct {
		URL string `json:"url"`
	}
	e.Panel(201, "POST", "/v1/panel/suscripcion", map[string]string{"plan": "inicial"}, &alta)
	id := preapproval(alta.URL)
	cobro(e, id, "8001", 6900, e.Ahora.AddDate(0, 1, 0))
	// Pasaron 10 días de 30: quedan 20 (2/3 del mes). Inicial → Centro: (29.900 − 6.900) × 2/3.
	e.Ahora = e.Ahora.AddDate(0, 0, 10)
	var x respuestaCambio
	e.Panel(200, "POST", "/v1/panel/suscripcion/cambiar", map[string]string{"plan": "centro"}, &x)
	if want := int64(15334); x.Hoy != want || x.Despues != 29900 || x.URL == "" {
		t.Fatalf("subida: %+v (esperado hoy %d)", x, want)
	}
	if s := leer(e); s.Plan != "inicial" || e.MP.Montos[id] != 6900 {
		t.Fatalf("sin pagar la diferencia: %+v %d", s.Acceso, e.MP.Montos[id])
	}
	// Bajar sin haber pagado: no hay nada que bajar (sigue Inicial) y la orden vieja queda cancelada.
	e.Panel(409, "POST", "/v1/panel/suscripcion/cambiar", map[string]string{"plan": "inicial"}, nil)
	// Dos pedidos a la vez no se pisan: mientras uno está en curso, el otro espera.
	e.SQL("UPDATE suscripciones SET cambiando_hasta = $1 WHERE mp_id = $2", e.Ahora.Add(time.Minute), id)
	var p tope
	e.Panel(409, "POST", "/v1/panel/suscripcion/cambiar", map[string]string{"plan": "equipo"}, &p)
	if p.Codigo != "cambio_en_curso" {
		t.Fatalf("en curso: %+v", p)
	}
	e.SQL("UPDATE suscripciones SET cambiando_hasta = NULL WHERE mp_id = $1", id)
	// Pagar la diferencia: Centro rige ya y el preapproval cobra Centro desde el próximo mes.
	ref := strings.TrimPrefix(x.URL, "https://mp.falso/checkout/")
	orden := e.MP.PagarOrden(ref, "processed", "accredited", x.Hoy, 41)
	mpfalso.Avisar(e.Handler, "order", orden, 999, e.Ahora)
	mpfalso.Avisar(e.Handler, "order", orden, 999, e.Ahora)
	if s := leer(e); s.Plan != "centro" || s.Suscripcion.Monto != 29900 || e.MP.Montos[id] != 29900 {
		t.Fatalf("subida pagada: %+v %+v", s.Acceso, s.Suscripcion)
	}
	// Mercado Pago reenvía el aviso del cobro de Inicial (de antes de la subida): no vuelve a Inicial.
	if s := e.Avisar("subscription_authorized_payment", "8001"); s != 200 {
		t.Fatalf("aviso repetido: %d", s)
	}
	if s := leer(e); s.Plan != "centro" {
		t.Fatalf("el aviso repetido pisó la subida: %+v", s.Acceso)
	}
	// Bajar a Inicial: Centro pagado sigue hasta el cobro; el cobro es de Inicial y ahí baja.
	e.Panel(200, "POST", "/v1/panel/suscripcion/cambiar", map[string]string{"plan": "inicial"}, &x)
	if s := leer(e); s.Plan != "centro" || e.MP.Montos[id] != 6900 {
		t.Fatalf("baja programada: %+v %d", s.Acceso, e.MP.Montos[id])
	}
	e.Ahora = e.Ahora.AddDate(0, 0, 21)
	cobro(e, id, "8002", 6900, e.Ahora.AddDate(0, 1, 0))
	if s := leer(e); s.Plan != "inicial" {
		t.Fatalf("después del cobro: %+v", s.Acceso)
	}
	// Si Mercado Pago cobra menos que el plan (un cambio de monto que no llegó), rige lo que se pagó.
	e.SQL("UPDATE suscripciones SET plan = 'centro', monto = 29900 WHERE mp_id = $1", id)
	e.SQL("UPDATE negocios SET plan = 'centro' WHERE id = $1", e.Negocio)
	e.Ahora = e.Ahora.AddDate(0, 1, 0)
	cobro(e, id, "8003", 14900, e.Ahora.AddDate(0, 2, 0))
	if s := leer(e); s.Plan != "equipo" {
		t.Fatalf("cobró menos: %+v", s.Acceso)
	}
}

// Un checkout abandonado se cancela también en Mercado Pago, y si igual lo autorizan no revive.
func TestCheckoutAbandonadoNoQuedaCobrando(t *testing.T) {
	e := cobrosprueba.Armar(t)
	e.SQL("UPDATE negocios SET prueba_hasta = NULL")
	var a, b struct {
		URL string `json:"url"`
	}
	e.Panel(201, "POST", "/v1/panel/suscripcion", map[string]string{"plan": "equipo"}, &a)
	e.Panel(201, "POST", "/v1/panel/suscripcion", map[string]string{"plan": "centro", "periodo": "anual"}, &b)
	viejo, nuevo := preapproval(a.URL), preapproval(b.URL)
	if e.MP.Preaprobacion[viejo].Status != "cancelled" {
		t.Fatalf("el abandonado sigue en Mercado Pago: %s", e.MP.Preaprobacion[viejo].Status)
	}
	proximo := e.Ahora.AddDate(1, 0, 0)
	cobro(e, nuevo, "9101", 299000, proximo)
	// Alguien igual autoriza el link viejo: no revive y se cancela en Mercado Pago.
	e.MP.Autorizar(viejo, "authorized", &proximo)
	e.Avisar("subscription_preapproval", viejo)
	var n int
	err := e.Bases.App.EnNegocio(context.Background(), e.Negocio, func(tx pgx.Tx) error {
		return tx.QueryRow(context.Background(), "SELECT count(*) FROM suscripciones WHERE estado = 'autorizada'").Scan(&n)
	})
	if err != nil || n != 1 || e.MP.Preaprobacion[viejo].Status != "cancelled" {
		t.Fatalf("autorizadas: %d, vieja en MP: %s (%v)", n, e.MP.Preaprobacion[viejo].Status, err)
	}
}

func TestPackDeWhatsApp(t *testing.T) {
	e := cobrosprueba.Armar(t)
	var c struct {
		URL string `json:"url"`
	}
	e.Panel(201, "POST", "/v1/panel/suscripcion/whatsapp", nil, &c)
	ref := strings.TrimPrefix(c.URL, "https://mp.falso/checkout/")
	if !strings.HasPrefix(ref, e.Negocio+"-wa-") {
		t.Fatalf("referencia: %q", c.URL)
	}
	o := e.MP.Ordenes[ref]
	if o.Token != mpfalso.TokenPlataforma || o.Pedido.TotalAmount != "5900.00" {
		t.Fatalf("orden: %+v", o)
	}
	saldo := func() int {
		var u struct {
			Uso suscripcion.Uso `json:"uso"`
		}
		e.Panel(200, "GET", "/v1/panel/suscripcion", nil, &u)
		return u.Uso.WhatsAppSaldo
	}
	// Rechazada: nada. Pagada: +100, una sola vez aunque el aviso se repita.
	orden := e.MP.PagarOrden(ref, "failed", "rejected_by_issuer", 5900, 1)
	if s := mpfalso.Avisar(e.Handler, "order", orden, 999, e.Ahora); s != 200 || saldo() != 0 {
		t.Fatalf("rechazada: %d %d", s, saldo())
	}
	// Un doble clic no arma otra orden: devuelve la pendiente.
	var c2 struct {
		URL string `json:"url"`
	}
	e.Panel(201, "POST", "/v1/panel/suscripcion/whatsapp", nil, &c2)
	if c2.URL != c.URL {
		t.Fatalf("otra orden: %q %q", c2.URL, c.URL)
	}
	// Si la pendiente quedó sin link (Mercado Pago no respondió), la orden nueva lleva su misma
	// referencia: si no, el pago no encontraba el pack.
	e.SQL("UPDATE whatsapp_packs SET link = NULL WHERE referencia = $1", ref)
	e.Panel(201, "POST", "/v1/panel/suscripcion/whatsapp", nil, &c2)
	if c2.URL != c.URL {
		t.Fatalf("sin link, otra referencia: %q %q", c2.URL, c.URL)
	}
	// Pagada sin que llegue el aviso: la vuelta de Mercado Pago la concilia igual.
	e.MP.PagarOrden(ref, "processed", "accredited", 5900, 2)
	e.Panel(204, "POST", "/v1/panel/suscripcion/conciliar", nil, nil)
	mpfalso.Avisar(e.Handler, "order", orden, 999, e.Ahora)
	mpfalso.Avisar(e.Handler, "order", orden, 999, e.Ahora)
	if saldo() != 100 {
		t.Fatalf("pagada: %d", saldo())
	}
	// Un recordatorio del saldo que no sale (el turno se cancela) vuelve al saldo.
	turno := e.Reservar(201, "09:00", "1155550000", nil)
	e.SQL(`INSERT INTO whatsapp_mensajes (negocio_id, turno_id, tipo, para, token, clave, datos, estado, proximo_intento, desde_saldo)
		VALUES ($1, $2, 'recordatorio', '+5491155550000', 'tokendeprueba000001', 'r:1', '{}', 'pendiente', now(), true)`, e.Negocio, turno.ID)
	e.SQL("UPDATE negocios SET whatsapp_saldo = whatsapp_saldo - 1 WHERE id = $1", e.Negocio)
	e.SQL("UPDATE whatsapp_mensajes SET estado = 'descartado' WHERE clave = 'r:1'")
	if saldo() != 100 {
		t.Fatalf("el descartado vuelve al saldo: %d", saldo())
	}
	// Una devolución (o un contracargo) saca el pack del saldo.
	e.MP.OrdenesID[orden].Pagos[len(e.MP.OrdenesID[orden].Pagos)-1].Status = "refunded"
	mpfalso.Avisar(e.Handler, "order", orden, 999, e.Ahora)
	if saldo() != 0 {
		t.Fatalf("devuelto: %d", saldo())
	}
}

// Lo que trae el plan se ve antes de completar un formulario: el acceso dice qué funciones rigen y
// con qué plan vienen las que faltan, y la página pública no ofrece lista de espera ni cupón.
func TestElPlanSeVeAntesDelTope(t *testing.T) {
	e := cobrosprueba.Armar(t)
	e.PonerPlan(e.Negocio, "inicial")
	var a suscripcion.Acceso
	e.Panel(200, "GET", "/v1/panel/acceso", nil, &a)
	if a.Plan != "inicial" || len(a.Funciones) != 0 || a.Viene["cupones"] != "equipo" || a.Viene["csv"] != "centro" {
		t.Fatalf("acceso con Inicial: %+v", a)
	}
	var pg struct {
		Espera  bool `json:"espera"`
		Cupones bool `json:"cupones"`
	}
	e.Esperar(200, cobrosprueba.P{Metodo: "GET", Ruta: "/v1/p/el-corte"}, &pg)
	if pg.Espera || pg.Cupones {
		t.Fatalf("página con Inicial: %+v", pg)
	}
	e.PonerPlan(e.Negocio, "equipo")
	e.Esperar(200, cobrosprueba.P{Metodo: "GET", Ruta: "/v1/p/el-corte"}, &pg)
	if !pg.Espera || !pg.Cupones {
		t.Fatalf("página con Equipo: %+v", pg)
	}
	a = suscripcion.Acceso{}
	e.Panel(200, "GET", "/v1/panel/acceso", nil, &a)
	if _, ok := a.Viene["cupones"]; ok || a.Viene["dominio"] != "centro" {
		t.Fatalf("acceso con Equipo: %+v", a)
	}
}

func TestBotonDeArrepentimiento(t *testing.T) {
	e := cobrosprueba.Armar(t)
	e.Esperar(400, cobrosprueba.P{Metodo: "POST", Ruta: "/v1/arrepentimiento", Cuerpo: map[string]string{"email": "no-es-mail"}}, nil)
	var r struct {
		Codigo string `json:"codigo"`
	}
	e.Esperar(201, cobrosprueba.P{Metodo: "POST", Ruta: "/v1/arrepentimiento", Cuerpo: map[string]string{"email": "ana@ejemplo.com",
		"negocio": "el-corte", "detalle": "Me arrepentí del pack"}}, &r)
	if len(r.Codigo) != 8 {
		t.Fatalf("código: %+v", r)
	}
}
