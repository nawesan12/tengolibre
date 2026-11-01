package mcp_test

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/cobros/cobrosprueba"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/mcp"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/turnos"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/prueba"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/servidor"
)

// publico: la API con la plata enganchada (señas con el Mercado Pago falso) y el servidor MCP.
type publico struct {
	*cobrosprueba.Entorno
	t *testing.T
}

func nuevoPublico(t *testing.T) *publico {
	e := cobrosprueba.ArmarCon(t, func(a *prueba.API, tur *turnos.Servicio) []servidor.Dominio {
		return []servidor.Dominio{&mcp.Servicio{Pool: a.Bases.App, Sesion: a.Sesion, Turnos: tur, Ahora: a.Reloj(),
			Sitio: "https://tengolibre.com.ar", TopePedidos: 40, TopeReservas: 5}}
	})
	return &publico{Entorno: e, t: t}
}

// rpc al servidor público desde una IP.
func (p *publico) rpc(ip, metodo string, params any) (respuesta, int) {
	p.t.Helper()
	b, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 7, "method": metodo, "params": params})
	req := httptest.NewRequest("POST", "/v1/mcp/publico", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("CF-Connecting-IP", ip)
	req.Header.Set("Origin", "https://claude.ai")
	rec := httptest.NewRecorder()
	p.Handler.ServeHTTP(rec, req)
	var r respuesta
	if rec.Code == 200 {
		if err := json.Unmarshal(rec.Body.Bytes(), &r); err != nil {
			p.t.Fatalf("%s: %s", metodo, rec.Body)
		}
	}
	return r, rec.Code
}

func (p *publico) llamar(nombre string, args any, esError bool, v any) string {
	p.t.Helper()
	r, cod := p.rpc("200.1.1.1", "tools/call", map[string]any{"name": nombre, "arguments": args})
	if cod != 200 || r.Error != nil {
		p.t.Fatalf("%s: %d %+v", nombre, cod, r.Error)
	}
	var res resultado
	if err := json.Unmarshal(r.Result, &res); err != nil {
		p.t.Fatal(err)
	}
	if res.IsError != esError {
		p.t.Fatalf("%s: isError %v (quería %v): %s", nombre, res.IsError, esError, r.Result)
	}
	if v != nil {
		if err := json.Unmarshal(res.Structured, v); err != nil {
			p.t.Fatal(err)
		}
	}
	return string(res.Structured)
}

type reservaIA struct {
	Negocio string `json:"negocio"`
	Turno   struct {
		Codigo, Estado, Servicio, Profesional string
		Precio                                int64
	} `json:"turno"`
	Link string `json:"link"`
	Pago *struct {
		Monto int64  `json:"monto"`
		URL   string `json:"url"`
	} `json:"pago"`
	Mensaje string `json:"mensaje"`
}

func TestLaIADeTusClientes(t *testing.T) {
	p := nuevoPublico(t)
	p.SQL("UPDATE negocios SET ciudad = 'Rosario', rubro = 'barberia'")

	// Sin clave: initialize y la lista de herramientas.
	r, cod := p.rpc("200.1.1.1", "initialize", map[string]any{"protocolVersion": "2025-06-18", "clientInfo": map[string]string{"name": "claude-ai"}})
	if cod != 200 || r.Error != nil || !strings.Contains(string(r.Result), "tengolibre-publico") {
		t.Fatalf("initialize: %d %s", cod, r.Result)
	}
	r, _ = p.rpc("200.1.1.1", "tools/list", map[string]any{})
	for _, h := range []string{"buscar_negocios", "ver_servicios_y_profesionales", "buscar_horarios_libres", "reservar_turno"} {
		if !strings.Contains(string(r.Result), `"`+h+`"`) {
			t.Fatalf("falta %s: %s", h, r.Result)
		}
	}

	// Apagada (por defecto), el negocio no aparece ni se puede usar.
	var ia mcp.IAPublica
	if p.Panel(200, "GET", "/v1/panel/ia/publico", nil, &ia); ia.Activo || ia.ReservasMes != 0 || !strings.HasSuffix(ia.URL, "/v1/mcp/publico") {
		t.Fatalf("panel apagada: %+v", ia)
	}
	var lista struct {
		Negocios []mcp.FichaIA `json:"negocios"`
	}
	if p.llamar("buscar_negocios", map[string]any{"texto": "corte"}, false, &lista); len(lista.Negocios) != 0 {
		t.Fatalf("apagada aparece: %+v", lista)
	}
	p.llamar("ver_servicios_y_profesionales", map[string]any{"negocio": "el-corte"}, true, nil)

	// Prendida: la encuentra por texto, rubro y ciudad.
	p.Panel(400, "PUT", "/v1/panel/ia/publico", map[string]any{}, nil)
	p.Panel(204, "PUT", "/v1/panel/ia/publico", map[string]any{"activo": true}, nil)
	for _, q := range []map[string]any{{"texto": "corte"}, {"rubro": "barberia", "ciudad": "rosa"}} {
		if p.llamar("buscar_negocios", q, false, &lista); len(lista.Negocios) != 1 || lista.Negocios[0].Slug != "el-corte" ||
			lista.Negocios[0].Pagina != "https://tengolibre.com.ar/el-corte" {
			t.Fatalf("buscar %v: %+v", q, lista)
		}
	}
	if p.llamar("buscar_negocios", map[string]any{"ciudad": "Salta"}, false, &lista); len(lista.Negocios) != 0 {
		t.Fatalf("otra ciudad: %+v", lista)
	}

	// Servicios, profesionales (sin datos de contacto) y horarios libres del lunes.
	cat := p.llamar("ver_servicios_y_profesionales", map[string]any{"negocio": "el-corte"}, false, nil)
	if !strings.Contains(cat, `"Corte"`) || !strings.Contains(cat, `"Ana"`) || !strings.Contains(cat, "politicaCancelacion") {
		t.Fatalf("catálogo: %s", cat)
	}
	libres := p.llamar("buscar_horarios_libres", map[string]any{"negocio": "el-corte", "servicioId": "corte", "fecha": "2026-11-02"}, false, nil)
	if !strings.Contains(libres, `"09:00"`) {
		t.Fatalf("libres: %s", libres)
	}

	// Reservar sin seña: queda pendiente, con origen ia y el link del turno.
	p.llamar("reservar_turno", map[string]any{"negocio": "el-corte", "servicioId": "Corte", "fecha": "2026-11-02", "hora": "9:00",
		"nombre": "María López"}, true, nil) // sin teléfono ni email
	var res reservaIA
	p.llamar("reservar_turno", map[string]any{"negocio": "el-corte", "servicioId": "Corte", "fecha": "2026-11-02", "hora": "9:00",
		"nombre": "María López", "telefono": "341 555-1111"}, false, &res)
	if res.Turno.Estado != "pendiente" || res.Pago != nil || res.Link != "https://tengolibre.com.ar/turno/"+res.Turno.Codigo || res.Negocio != "El Corte" {
		t.Fatalf("reserva: %+v", res)
	}
	var agenda struct {
		Turnos []turnos.Turno `json:"turnos"`
	}
	p.Panel(200, "GET", "/v1/panel/turnos?desde=2026-11-02", nil, &agenda)
	if len(agenda.Turnos) != 1 || agenda.Turnos[0].Origen != turnos.OrigenIA || agenda.Turnos[0].Cliente.Nombre != "María López" {
		t.Fatalf("agenda: %+v", agenda.Turnos)
	}
	// El mismo horario ya no está.
	if out := p.llamar("reservar_turno", map[string]any{"negocio": "el-corte", "servicioId": "Corte", "fecha": "2026-11-02", "hora": "09:00",
		"nombre": "Otra", "email": "otra@x.com"}, true, nil); !strings.Contains(out, "ya no está libre") {
		t.Fatalf("ocupado: %s", out)
	}

	// Con seña: el link de Mercado Pago que abre la persona.
	p.ConectarMP()
	p.Panel(200, "PUT", "/v1/panel/cobros", map[string]any{"modo": "porcentaje", "senaPorcentaje": 50}, nil)
	p.llamar("reservar_turno", map[string]any{"negocio": "el-corte", "servicioId": "Corte", "fecha": "2026-11-02", "hora": "10:00",
		"nombre": "Juana Pérez", "email": "juana@x.com"}, false, &res)
	if res.Turno.Estado != "pendiente_pago" || res.Pago == nil || res.Pago.Monto != 4000 || res.Pago.URL == "" || !strings.Contains(res.Mensaje, "seña") {
		t.Fatalf("con seña: %+v", res)
	}

	// Una clienta penalizada no reserva por la IA (como en la web).
	p.SQL(`INSERT INTO penalidades (negocio_id, cliente_id, hasta, motivo)
		SELECT negocio_id, id, now() + interval '30 days', 'manual' FROM clientes WHERE telefono = '3415551111'`)
	if out := p.llamar("reservar_turno", map[string]any{"negocio": "el-corte", "servicioId": "Corte", "fecha": "2026-11-02", "hora": "11:00",
		"nombre": "María López", "telefono": "3415551111"}, true, nil); !strings.Contains(out, "ausencias") {
		t.Fatalf("penalizada: %s", out)
	}

	// El panel cuenta las reservas del mes y muestra lo último.
	if p.Panel(200, "GET", "/v1/panel/ia/publico", nil, &ia); !ia.Activo || ia.ReservasMes != 2 || len(ia.Ultimas) != 2 ||
		!strings.HasPrefix(ia.Ultimas[0].Resumen, "reservó Corte para Juana el lunes") {
		t.Fatalf("panel: %+v", ia)
	}

	// Tope de reservas por IP (5 en la hora, contando las que no pasaron: van 4; los datos
	// incompletos no cuentan).
	p.llamar("reservar_turno", map[string]any{"negocio": "el-corte", "servicioId": "Corte", "fecha": "2026-11-02", "hora": "11:30",
		"nombre": "Rosa", "email": "rosa@x.com"}, false, nil)
	if out := p.llamar("reservar_turno", map[string]any{"negocio": "el-corte", "servicioId": "Corte", "fecha": "2026-11-02", "hora": "12:00",
		"nombre": "Rosa", "email": "rosa@x.com"}, true, nil); !strings.Contains(out, "muchas reservas") {
		t.Fatalf("tope de reservas: %s", out)
	}
	// Tope de pedidos por IP: 40 por minuto; otra IP sigue andando.
	for i := 0; i < 40; i++ {
		p.rpc("10.0.0.9", "ping", map[string]any{})
	}
	if _, cod := p.rpc("10.0.0.9", "ping", map[string]any{}); cod != 429 {
		t.Fatalf("tope de pedidos: %d", cod)
	}
	if _, cod := p.rpc("10.0.0.10", "ping", map[string]any{}); cod != 200 {
		t.Fatalf("otra ip: %d", cod)
	}

	// Apagada otra vez: deja de aparecer.
	p.Panel(204, "PUT", "/v1/panel/ia/publico", map[string]any{"activo": false}, nil)
	p.llamar("buscar_horarios_libres", map[string]any{"negocio": "el-corte", "servicioId": "corte", "fecha": "2026-11-02"}, true, nil)
}
