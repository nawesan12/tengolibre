package caja_test

import (
	"testing"

	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/caja"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/cobros/cobrosprueba"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/reportes"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/turnos"
	"github.com/nawesan12/tengolibre/apps/api/internal/integraciones/mercadopago/mpfalso"
)

type P = cobrosprueba.P

func TestCajaDelLocal(t *testing.T) {
	e := cobrosprueba.Armar(t)
	e.ConectarMP()
	e.Panel(200, "PUT", "/v1/panel/cobros", map[string]any{"modo": "porcentaje", "senaPorcentaje": 50}, nil)

	// Sofía reserva con seña ($ 4.000 de $ 8.000) y la paga.
	sofia := e.Reservar(201, "09:00", "1111", nil)
	ref := sofia.Pago.URL[len("https://mp.falso/checkout/"):]
	e.MP.Pagar(1001, ref, "approved", 4000)
	if c := mpfalso.Avisar(e.Handler, "payment", "1001", e.MP.Usuario, e.Ahora); c != 200 {
		t.Fatalf("aviso: %d", c)
	}
	// La seña entra el lunes 2 a la mañana (la base pone su propio reloj).
	e.SQL("UPDATE pagos SET creado_en = $1", cobrosprueba.En("2026-11-02 08:00"))

	// Catálogo de productos.
	var serum, cera caja.Producto
	stock := 2
	e.Panel(201, "POST", "/v1/panel/productos", map[string]any{"nombre": "Sérum", "precio": 9000, "stock": stock}, &serum)
	e.Panel(201, "POST", "/v1/panel/productos", map[string]any{"nombre": "Cera", "precio": 3000}, &cera)
	e.Panel(409, "POST", "/v1/panel/productos", map[string]any{"nombre": "Cera", "precio": 1}, nil)
	e.Panel(400, "POST", "/v1/panel/productos", map[string]any{"nombre": "X", "precio": 1}, nil)
	var vieja caja.Producto
	e.Panel(201, "POST", "/v1/panel/productos", map[string]any{"nombre": "Máscara", "precio": 5000, "activo": false}, &vieja)
	var lista struct{ Productos []caja.Producto }
	e.Panel(200, "GET", "/v1/panel/productos?activos=1", nil, &lista)
	if len(lista.Productos) != 2 || serum.Stock == nil || *serum.Stock != 2 || cera.Stock != nil {
		t.Fatalf("productos: %+v", lista.Productos)
	}

	// Antes de atenderla, la caja muestra lo que resta pero no cobra.
	ruta := "/v1/panel/turnos/" + sofia.ID + "/caja"
	var c caja.Cuenta
	e.Panel(200, "GET", ruta, nil, &c)
	if c.Estado != "sin_cobrar" || c.Total != 4000 || c.EstadoTurno != "confirmado" || len(c.Lineas) != 2 || c.Lineas[1].Monto != -4000 {
		t.Fatalf("caja antes: %+v", c)
	}
	e.Panel(409, "POST", ruta, map[string]any{"medio": "efectivo"}, nil)
	e.Panel(200, "PATCH", "/v1/panel/turnos/"+sofia.ID, map[string]any{"estado": "atendido"}, nil)

	// El lunes 2 al mediodía: se lleva un sérum y paga en efectivo.
	e.Ahora = cobrosprueba.En("2026-11-02 12:00")
	e.Panel(400, "POST", ruta, map[string]any{"medio": "cheque"}, nil)
	e.Panel(404, "POST", ruta, map[string]any{"medio": "efectivo", "productos": []map[string]any{{"productoId": vieja.ID, "cantidad": 1}}}, nil)
	e.Panel(201, "POST", ruta, map[string]any{"medio": "efectivo", "productos": []map[string]any{{"productoId": serum.ID, "cantidad": 1}}}, &c)
	if c.Estado != "cobrado" || c.Total != 13000 || *c.Medio != "efectivo" || c.CobradoEn == nil || len(c.Lineas) != 3 || c.Lineas[2].Tipo != "producto" {
		t.Fatalf("cobrado: %+v", c)
	}
	e.Panel(409, "POST", ruta, map[string]any{"medio": "efectivo"}, nil)
	e.Panel(200, "GET", "/v1/panel/productos", nil, &lista)
	for _, p := range lista.Productos {
		if p.ID == serum.ID && *p.Stock != 1 {
			t.Fatalf("stock del sérum: %d", *p.Stock)
		}
	}

	// Otro turno, cargado en el panel como sobreturno (sin seña): no hay stock para dos sérums; queda debiendo con descuento.
	var panelT turnos.Turno
	e.Panel(201, "POST", "/v1/panel/turnos", map[string]any{"servicioId": e.Servicio, "profesionalId": e.Profesional,
		"inicio": cobrosprueba.En("2026-11-02 10:00"), "forzar": true, "cliente": map[string]string{"nombre": "Nadia", "telefono": "2222"}}, &panelT)
	e.Panel(200, "PATCH", "/v1/panel/turnos/"+panelT.ID, map[string]any{"estado": "atendido"}, nil)
	ruta2 := "/v1/panel/turnos/" + panelT.ID + "/caja"
	e.Panel(409, "POST", ruta2, map[string]any{"medio": "efectivo", "productos": []map[string]any{{"productoId": serum.ID, "cantidad": 2}}}, nil)
	e.Panel(400, "POST", ruta2, map[string]any{"medio": "despues", "descuento": 9000}, nil)
	e.Panel(201, "POST", ruta2, map[string]any{"medio": "despues", "descuento": 1000}, &c)
	if c.Estado != "debe" || c.Total != 7000 || c.CobradoEn != nil {
		t.Fatalf("debe: %+v", c)
	}
	e.Panel(409, "POST", ruta2, map[string]any{"medio": "efectivo"}, nil)

	var r caja.Resumen
	e.Panel(200, "GET", "/v1/panel/caja?fecha=2026-11-02", nil, &r)
	if r.PorMedio["efectivo"] != 13000 || r.PorMedio["transferencia"] != 0 || r.Caja != 13000 || r.Senas != 4000 || r.Total != 17000 ||
		r.Productos != 9000 || r.Cobros != 1 || r.Debe != 7000 || len(r.Movimientos) != 1 || r.Movimientos[0].Cliente != "Cliente 1111" {
		t.Fatalf("resumen del día: %+v", r)
	}

	// Nadia paga por transferencia.
	e.Panel(200, "POST", ruta2+"/saldar", map[string]any{"medio": "transferencia"}, &c)
	if c.Estado != "cobrado" || *c.Medio != "transferencia" {
		t.Fatalf("saldado: %+v", c)
	}
	e.Panel(409, "POST", ruta2+"/saldar", map[string]any{"medio": "transferencia"}, nil)
	e.Panel(200, "GET", "/v1/panel/caja", nil, &r)
	if r.Fecha != "2026-11-02" || r.PorMedio["transferencia"] != 7000 || r.Debe != 0 || r.Total != 24000 {
		t.Fatalf("resumen después de saldar: %+v", r)
	}

	// Reportes: lo facturado (precio de los atendidos) y la plata real (seña + caja).
	var rep reportes.Reporte
	e.Panel(200, "GET", "/v1/panel/reportes?desde=2026-11-02&hasta=2026-11-02", nil, &rep)
	x := rep.Resumen
	if x.Ingresos != 16000 || x.Cobrado != 24000 || x.CobradoSenas != 4000 || x.CobradoCaja != 20000 || x.Productos != 9000 ||
		x.CobradoPorMedio["efectivo"] != 13000 || x.CobradoPorMedio["transferencia"] != 7000 || rep.PorDia[0].Cobrado != 24000 {
		t.Fatalf("reporte: %+v %+v", x, rep.PorDia)
	}

	// Anular el cobro de Sofía devuelve el sérum al stock.
	e.Panel(200, "DELETE", ruta, nil, &c)
	if c.Estado != "sin_cobrar" || c.Total != 4000 {
		t.Fatalf("anulado: %+v", c)
	}
	e.Panel(200, "GET", "/v1/panel/productos", nil, &lista)
	for _, p := range lista.Productos {
		if p.ID == serum.ID && *p.Stock != 2 {
			t.Fatalf("stock después de anular: %d", *p.Stock)
		}
	}
	e.Panel(404, "DELETE", ruta, nil, nil)

	// Editar y borrar productos.
	var ed caja.Producto
	e.Panel(200, "PATCH", "/v1/panel/productos/"+serum.ID, map[string]any{"precio": 9500, "sinStock": true}, &ed)
	if ed.Precio != 9500 || ed.Stock != nil {
		t.Fatalf("editado: %+v", ed)
	}
	e.Panel(204, "DELETE", "/v1/panel/productos/"+cera.ID, nil, nil)
	e.Panel(404, "DELETE", "/v1/panel/productos/"+cera.ID, nil, nil)
}
