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
			if cupo, err = planes.CupoIA(context.Background(), tx, time.Now()); err != nil {
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

func TestAnualCentroYExtras(t *testing.T) {
	e := cobrosprueba.Armar(t)
	var alta struct {
		URL string `json:"url"`
	}
	e.Panel(400, "POST", "/v1/panel/suscripcion", map[string]string{"plan": "centro", "periodo": "bienal"}, nil)
	e.Panel(400, "POST", "/v1/panel/suscripcion", map[string]string{"plan": "completo"}, nil)
	e.Panel(201, "POST", "/v1/panel/suscripcion", map[string]string{"plan": "centro", "periodo": "anual"}, &alta)
	id := strings.TrimPrefix(alta.URL, "https://mp.falso/suscripcion/")
	// Anual: 10 meses (299.000) cada 12.
	if e.MP.Montos[id] != 299000 || e.MP.Frecuencias[id] != 12 {
		t.Fatalf("anual: %d cada %d meses", e.MP.Montos[id], e.MP.Frecuencias[id])
	}
	proximo := e.Ahora.AddDate(1, 0, 0)
	e.MP.Autorizar(id, "authorized", &proximo)
	e.Avisar("subscription_preapproval", id)
	s := leer(e)
	if s.Plan != "centro" || s.Suscripcion.Periodo != "anual" || s.Suscripcion.Monto != 299000 {
		t.Fatalf("centro anual: %+v %+v", s.Acceso, s.Suscripcion)
	}
	// Extras: 2 profesionales más ($1.900 c/u por mes): el anual pasa a 10 × 33.700.
	var x suscripcion.Suscripcion
	e.Panel(400, "PUT", "/v1/panel/suscripcion/extras", map[string]int{"cantidad": -1}, nil)
	e.Panel(200, "PUT", "/v1/panel/suscripcion/extras", map[string]int{"cantidad": 2}, &x)
	if x.ProfesionalesExtra != 2 || x.Monto != 337000 || e.MP.Montos[id] != 337000 {
		t.Fatalf("extras: %+v %d", x, e.MP.Montos[id])
	}
	var u struct {
		Uso suscripcion.Uso `json:"uso"`
	}
	e.Panel(200, "GET", "/v1/panel/suscripcion", nil, &u)
	if u.Uso.TopeProfesionales != 27 || u.Uso.WhatsAppIncluidos != 150 || u.Uso.CupoIA != 1500 {
		t.Fatalf("uso con Centro y extras: %+v", u.Uso)
	}
	// Bajar a Equipo (mismo período): el monto baja, los extras se van y el tope vuelve a 12.
	e.Panel(200, "POST", "/v1/panel/suscripcion/cambiar", map[string]string{"plan": "equipo"}, &x)
	if x.Plan != "equipo" || x.Monto != 149000 || x.ProfesionalesExtra != 0 || e.MP.Montos[id] != 149000 {
		t.Fatalf("a equipo: %+v", x)
	}
	e.Panel(409, "PUT", "/v1/panel/suscripcion/extras", map[string]int{"cantidad": 1}, nil)
	// Pasar a mensual: un alta nueva que empieza cuando vence lo pagado y reemplaza a la anual.
	var nueva struct {
		URL string `json:"url"`
	}
	e.Panel(200, "POST", "/v1/panel/suscripcion/cambiar", map[string]string{"plan": "equipo", "periodo": "mensual"}, &nueva)
	nid := strings.TrimPrefix(nueva.URL, "https://mp.falso/suscripcion/")
	if e.MP.Montos[nid] != 14900 || e.MP.Frecuencias[nid] != 1 {
		t.Fatalf("mensual nueva: %d cada %d", e.MP.Montos[nid], e.MP.Frecuencias[nid])
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
	e.MP.PagarOrden(ref, "processed", "accredited", 5900, 2)
	mpfalso.Avisar(e.Handler, "order", orden, 999, e.Ahora)
	mpfalso.Avisar(e.Handler, "order", orden, 999, e.Ahora)
	if saldo() != 100 {
		t.Fatalf("pagada: %d", saldo())
	}
}
