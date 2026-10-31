package mcp_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/avisos"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/catalogo"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/disponibilidad"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/mcp"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/negocios"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/suscripcion"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/turnos"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/auditoria"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/prueba"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/servidor"
)

type P = prueba.Pedido

const (
	duenia = "user_duenia0001"
	recep  = "user_recep00001"
	otra   = "user_otra000001"
)

func en(s string) time.Time {
	t, err := time.ParseInLocation("2006-01-02 15:04", s, disponibilidad.ZonaPorDefecto)
	if err != nil {
		panic(err)
	}
	return t
}

// entorno: un negocio con un servicio y una profesional que atiende los lunes de 9 a 11.
type entorno struct {
	*prueba.API
	t        *testing.T
	negocio  string
	servicio catalogo.Prestacion
	ana      catalogo.Profesional
}

func nuevo(t *testing.T) *entorno {
	a := prueba.Nueva(t, en("2026-11-01 10:00"), func(a *prueba.API) []servidor.Dominio {
		app := a.Bases.App
		av := &avisos.Servicio{Pool: app, Sesion: a.Sesion, Ahora: a.Reloj()}
		sus := &suscripcion.Servicio{Pool: app, Sesion: a.Sesion, Ahora: a.Reloj()}
		a.Sesion.Escritura = sus.Escritura
		tur := &turnos.Servicio{Pool: app, Sesion: a.Sesion, Ahora: a.Reloj(), Avisar: av.Avisar}
		return []servidor.Dominio{
			&negocios.Servicio{Pool: app, Sesion: a.Sesion},
			&catalogo.Servicio{Pool: app, Sesion: a.Sesion},
			tur, av,
			&auditoria.Servicio{Pool: app, Sesion: a.Sesion},
			&mcp.Servicio{Pool: app, Sesion: a.Sesion, Turnos: tur, Ahora: a.Reloj(), Origenes: []string{"https://tengolibre.com.ar"}},
		}
	})
	e := &entorno{API: a, t: t}
	var n negocios.Negocio
	a.Esperar(201, P{Metodo: "POST", Ruta: "/v1/negocios", Quien: duenia, Cuerpo: map[string]string{"slug": "el-corte", "nombre": "El Corte"}}, &n)
	e.negocio = n.ID
	e.panel(201, "POST", "/v1/panel/servicios", map[string]any{"nombre": "Corte", "duracion": 30, "precio": 8000}, &e.servicio)
	e.panel(201, "POST", "/v1/panel/profesionales", map[string]any{"nombre": "Ana", "servicios": []string{e.servicio.ID},
		"horario": []disponibilidad.Franja{{Dia: 1, Desde: "09:00", Hasta: "11:00"}}, "orden": 1}, &e.ana)
	return e
}

func (e *entorno) panel(estado int, metodo, ruta string, cuerpo, v any) prueba.Respuesta {
	e.t.Helper()
	return e.Esperar(estado, P{Metodo: metodo, Ruta: ruta, Quien: duenia, Negocio: e.negocio, Cuerpo: cuerpo}, v)
}

type creado struct {
	Token mcp.Token `json:"token"`
	Clave string    `json:"clave"`
	URL   string    `json:"url"`
}

func (e *entorno) clave(cuerpo any) creado {
	e.t.Helper()
	var c creado
	e.panel(201, "POST", "/v1/panel/mcp/tokens", cuerpo, &c)
	return c
}

// rpc manda un mensaje al servidor MCP como lo haría un cliente (Claude, por ejemplo).
func (e *entorno) rpc(clave string, cuerpo any, cabeceras ...string) *httptest.ResponseRecorder {
	e.t.Helper()
	b, _ := json.Marshal(cuerpo)
	req := httptest.NewRequest("POST", "/v1/mcp", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	if clave != "" {
		req.Header.Set("Authorization", "Bearer "+clave)
	}
	for i := 0; i+1 < len(cabeceras); i += 2 {
		req.Header.Set(cabeceras[i], cabeceras[i+1])
	}
	rec := httptest.NewRecorder()
	e.Handler.ServeHTTP(rec, req)
	return rec
}

type respuesta struct {
	ID     int             `json:"id"`
	Result json.RawMessage `json:"result"`
	Error  *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func (e *entorno) pedir(clave, metodo string, params any) respuesta {
	e.t.Helper()
	rec := e.rpc(clave, map[string]any{"jsonrpc": "2.0", "id": 7, "method": metodo, "params": params}, "MCP-Protocol-Version", "2025-06-18")
	if rec.Code != 200 {
		e.t.Fatalf("%s: %d %s", metodo, rec.Code, rec.Body)
	}
	var r respuesta
	if err := json.Unmarshal(rec.Body.Bytes(), &r); err != nil || r.ID != 7 {
		e.t.Fatalf("%s: respuesta rara: %s", metodo, rec.Body)
	}
	return r
}

type resultado struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Structured json.RawMessage `json:"structuredContent"`
	IsError    bool            `json:"isError"`
}

// llamar a una herramienta; esError dice qué se espera. Decodifica structuredContent en v.
func (e *entorno) llamar(clave, nombre string, args any, esError bool, v any) resultado {
	e.t.Helper()
	r := e.pedir(clave, "tools/call", map[string]any{"name": nombre, "arguments": args})
	if r.Error != nil {
		e.t.Fatalf("%s: error de protocolo %d %s", nombre, r.Error.Code, r.Error.Message)
	}
	var res resultado
	if err := json.Unmarshal(r.Result, &res); err != nil {
		e.t.Fatal(err)
	}
	if res.IsError != esError || len(res.Content) != 1 || res.Content[0].Type != "text" || res.Content[0].Text != strings.TrimSpace(string(res.Structured)) {
		e.t.Fatalf("%s: isError %v (quería %v): %s", nombre, res.IsError, esError, r.Result)
	}
	if v != nil {
		if err := json.Unmarshal(res.Structured, v); err != nil {
			e.t.Fatal(err)
		}
	}
	return res
}

func (e *entorno) herramientas(clave string) []string {
	e.t.Helper()
	var l struct {
		Tools []struct {
			Name        string         `json:"name"`
			Description string         `json:"description"`
			InputSchema map[string]any `json:"inputSchema"`
		} `json:"tools"`
	}
	if err := json.Unmarshal(e.pedir(clave, "tools/list", map[string]any{}).Result, &l); err != nil {
		e.t.Fatal(err)
	}
	var out []string
	for _, h := range l.Tools {
		if h.Description == "" || h.InputSchema["type"] != "object" {
			e.t.Fatalf("herramienta sin descripción o esquema: %+v", h)
		}
		out = append(out, h.Name)
	}
	return out
}

func (e *entorno) sql(q string, args ...any) {
	e.t.Helper()
	ctx := context.Background()
	c, err := pgx.Connect(ctx, e.Bases.AdminURL)
	if err != nil {
		e.t.Fatal(err)
	}
	defer c.Close(ctx)
	if _, err := c.Exec(ctx, q, args...); err != nil {
		e.t.Fatal(err)
	}
}

type turnoMCP struct {
	Turno mcp.TurnoMCP `json:"turno"`
}

func TestAsistentePorMCP(t *testing.T) {
	e := nuevo(t)
	c := e.clave(map[string]string{"nombre": "Claude de Carla"})
	if !strings.HasPrefix(c.Clave, mcp.Prefijo) || c.URL != mcp.URLPorDefecto || c.Token.Pista != c.Clave[len(c.Clave)-4:] ||
		!slices.Equal(c.Token.Permisos, mcp.PermisosPorDefecto) {
		t.Fatalf("clave: %+v", c)
	}
	// El panel la lista sin la clave.
	var panel struct {
		URL      string        `json:"url"`
		Tokens   []mcp.Token   `json:"tokens"`
		Permisos []mcp.Permiso `json:"permisos"`
	}
	r := e.panel(200, "GET", "/v1/panel/mcp", nil, &panel)
	if len(panel.Tokens) != 1 || panel.Tokens[0].UsadoEn != nil || len(panel.Permisos) != 5 || strings.Contains(string(r.Cuerpo), c.Clave) {
		t.Fatalf("panel: %s", r.Cuerpo)
	}

	// Sin clave, con una inventada o desde un origen ajeno: nada.
	if rec := e.rpc("", map[string]any{"jsonrpc": "2.0", "id": 1, "method": "ping"}); rec.Code != 401 || rec.Header().Get("WWW-Authenticate") == "" {
		t.Fatalf("sin clave: %d", rec.Code)
	}
	if rec := e.rpc(mcp.Prefijo+"inventada", map[string]any{"jsonrpc": "2.0", "id": 1, "method": "ping"}); rec.Code != 401 {
		t.Fatalf("inventada: %d", rec.Code)
	}
	if rec := e.rpc(c.Clave, map[string]any{"jsonrpc": "2.0", "id": 1, "method": "ping"}, "Origin", "https://malo.example"); rec.Code != 403 {
		t.Fatalf("origen ajeno: %d", rec.Code)
	}

	// initialize → initialized → tools/list → tools/call, como un cliente MCP.
	var ini struct {
		ProtocolVersion string                    `json:"protocolVersion"`
		Capabilities    map[string]map[string]any `json:"capabilities"`
		ServerInfo      struct{ Name, Title string }
		Instructions    string `json:"instructions"`
	}
	r1 := e.pedir(c.Clave, "initialize", map[string]any{"protocolVersion": "2025-06-18", "capabilities": map[string]any{},
		"clientInfo": map[string]string{"name": "claude-ai", "version": "0.1.0"}})
	if err := json.Unmarshal(r1.Result, &ini); err != nil || ini.ProtocolVersion != "2025-06-18" || ini.Capabilities["tools"] == nil ||
		ini.ServerInfo.Name != "tengolibre" || !strings.Contains(ini.ServerInfo.Title, "El Corte") || ini.Instructions == "" {
		t.Fatalf("initialize: %s", r1.Result)
	}
	if rec := e.rpc(c.Clave, map[string]any{"jsonrpc": "2.0", "method": "notifications/initialized"}); rec.Code != 202 || rec.Body.Len() != 0 {
		t.Fatalf("initialized: %d %s", rec.Code, rec.Body)
	}
	if string(e.pedir(c.Clave, "ping", nil).Result) != "{}" {
		t.Fatal("ping")
	}
	if r := e.pedir(c.Clave, "resources/list", nil); r.Error == nil || r.Error.Code != -32601 {
		t.Fatalf("método desconocido: %+v", r)
	}
	if rec := e.rpc(c.Clave, map[string]any{"jsonrpc": "2.0", "id": 1, "method": "ping"}, "MCP-Protocol-Version", "1999-01-01"); rec.Code != 400 {
		t.Fatalf("versión desconocida: %d", rec.Code)
	}
	hs := e.herramientas(c.Clave)
	for _, h := range []string{"ver_agenda", "buscar_horarios_libres", "ver_lista_de_espera", "ver_servicios_y_profesionales",
		"crear_turno", "confirmar_turno", "reprogramar_turno", "buscar_clientes", "ver_cliente"} {
		if !slices.Contains(hs, h) {
			t.Fatalf("falta %s en %v", h, hs)
		}
	}
	if slices.Contains(hs, "cancelar_turno") || slices.Contains(hs, "resumen_de_numeros") {
		t.Fatalf("aparecen herramientas sin permiso: %v", hs)
	}

	// «¿Qué huecos tengo el lunes?»: por nombre de servicio.
	var libres struct {
		Servicio string `json:"servicio"`
		Dias     []struct {
			Fecha    string         `json:"fecha"`
			Horarios []mcp.LibreMCP `json:"horarios"`
		} `json:"dias"`
	}
	e.llamar(c.Clave, "buscar_horarios_libres", map[string]any{"servicioId": "corte", "fecha": "2026-11-02"}, false, &libres)
	if len(libres.Dias) != 1 || len(libres.Dias[0].Horarios) == 0 || libres.Dias[0].Horarios[0].Hora != "09:00" ||
		libres.Dias[0].Horarios[0].Profesionales[0].Nombre != "Ana" {
		t.Fatalf("libres: %+v", libres)
	}

	// «Cargá a María el lunes a las 9».
	var nt turnoMCP
	e.llamar(c.Clave, "crear_turno", map[string]any{"servicioId": e.servicio.ID, "fecha": "2026-11-02", "hora": "9:00",
		"cliente": map[string]string{"nombre": "María Gómez", "telefono": "11 5555-1234"}}, false, &nt)
	if nt.Turno.Estado != "confirmado" || nt.Turno.Profesional != "Ana" || nt.Turno.Inicio != "2026-11-02T09:00:00-03:00" ||
		nt.Turno.Cliente.Telefono != "1155551234" || nt.Turno.Dia == "" {
		t.Fatalf("turno: %+v", nt)
	}
	// El mismo horario ya no está libre: error de la herramienta, no del protocolo.
	res := e.llamar(c.Clave, "crear_turno", map[string]any{"servicioId": e.servicio.ID, "fecha": "2026-11-02", "hora": "09:00",
		"clienteId": nt.Turno.Cliente.ID}, true, nil)
	if !strings.Contains(res.Content[0].Text, "ya no está libre") {
		t.Fatalf("ocupado: %s", res.Content[0].Text)
	}
	// La agenda del día lo muestra; la clienta aparece en la búsqueda.
	var ag struct {
		Turnos []mcp.TurnoMCP `json:"turnos"`
	}
	e.llamar(c.Clave, "ver_agenda", map[string]any{"desde": "2026-11-02"}, false, &ag)
	if len(ag.Turnos) != 1 || ag.Turnos[0].ID != nt.Turno.ID {
		t.Fatalf("agenda: %+v", ag)
	}
	var bc struct {
		Clientes []mcp.ClienteMCP `json:"clientes"`
	}
	e.llamar(c.Clave, "buscar_clientes", map[string]any{"texto": "5555"}, false, &bc)
	if len(bc.Clientes) != 1 || bc.Clientes[0].Nombre != "María Gómez" || bc.Clientes[0].ProximoTurno == nil {
		t.Fatalf("clientes: %+v", bc)
	}
	var ficha struct {
		Nombre string         `json:"nombre"`
		Turnos []mcp.TurnoMCP `json:"ultimosTurnos"`
	}
	e.llamar(c.Clave, "ver_cliente", map[string]any{"clienteId": bc.Clientes[0].ID}, false, &ficha)
	if ficha.Nombre != "María Gómez" || len(ficha.Turnos) != 1 {
		t.Fatalf("ficha: %+v", ficha)
	}
	// «Pasala a las 10».
	e.llamar(c.Clave, "reprogramar_turno", map[string]any{"turnoId": nt.Turno.ID, "fecha": "2026-11-02", "hora": "10:00"}, false, &nt)
	if nt.Turno.Inicio != "2026-11-02T10:00:00-03:00" {
		t.Fatalf("reprogramado: %+v", nt)
	}
	e.llamar(c.Clave, "ver_servicios_y_profesionales", nil, false, nil)
	e.llamar(c.Clave, "ver_lista_de_espera", map[string]any{}, false, nil)

	// Lo que cambió algo va a la campanita y a la auditoría, en nombre del asistente.
	var camp struct {
		Notificaciones []struct{ Titulo, Texto string } `json:"notificaciones"`
	}
	e.panel(200, "GET", "/v1/panel/campanita", nil, &camp)
	titulos := ""
	for _, n := range camp.Notificaciones {
		titulos += n.Titulo + " | " + n.Texto + "\n"
	}
	if !strings.Contains(titulos, "El asistente cargó un turno | Corte para María Gómez con Ana") ||
		!strings.Contains(titulos, "El asistente movió un turno") || !strings.Contains(titulos, "«Claude de Carla»") {
		t.Fatalf("campanita:\n%s", titulos)
	}
	var aud struct {
		Entradas []auditoria.Entrada `json:"entradas"`
	}
	e.panel(200, "GET", "/v1/panel/auditoria", nil, &aud)
	acciones := map[string]string{}
	for _, x := range aud.Entradas {
		acciones[x.Accion] = x.ClerkID
	}
	if acciones["asistente.turno_creado"] != mcp.Quien(c.Token.ID) || acciones["turno.reservado"] != mcp.Quien(c.Token.ID) ||
		acciones["mcp.clave_creada"] != duenia {
		t.Fatalf("auditoría: %v", acciones)
	}

	// El registro: cada llamada, lo más nuevo primero, sin el teléfono.
	var reg struct {
		Acciones []mcp.Accion `json:"acciones"`
	}
	r = e.panel(200, "GET", "/v1/panel/mcp/registro", nil, &reg)
	if len(reg.Acciones) != 9 || reg.Acciones[0].Herramienta != "ver_lista_de_espera" || reg.Acciones[0].Token != "Claude de Carla" ||
		reg.Acciones[0].Cliente != "claude-ai 0.1.0" || strings.Contains(string(r.Cuerpo), "5555-1234") || strings.Contains(string(r.Cuerpo), "1155551234") {
		t.Fatalf("registro: %s", r.Cuerpo)
	}
	var cambios struct {
		Acciones []mcp.Accion `json:"acciones"`
	}
	e.panel(200, "GET", "/v1/panel/mcp/registro?soloCambios=1", nil, &cambios)
	if len(cambios.Acciones) != 3 || cambios.Acciones[0].Herramienta != "reprogramar_turno" || !cambios.Acciones[0].Ok ||
		cambios.Acciones[1].Ok || cambios.Acciones[1].Error == "" || !strings.HasPrefix(cambios.Acciones[2].Resumen, "cargó Corte para María") ||
		!strings.Contains(string(cambios.Acciones[2].Argumentos), `"telefono":"(dado)"`) || cambios.Acciones[2].Titulo != "Crear un turno" {
		t.Fatalf("cambios: %+v", cambios.Acciones)
	}
	e.panel(200, "GET", "/v1/panel/mcp", nil, &panel)
	if panel.Tokens[0].UsadoEn == nil || panel.Tokens[0].Cliente != "claude-ai 0.1.0" {
		t.Fatalf("uso: %+v", panel.Tokens[0])
	}
}

func TestPermisosDelAsistente(t *testing.T) {
	e := nuevo(t)
	solo := e.clave(map[string]any{"nombre": "Sólo mira", "permisos": []string{"ver_agenda"}})
	e.panel(400, "POST", "/v1/panel/mcp/tokens", map[string]any{"nombre": "x", "permisos": []string{"todo"}}, nil)
	e.panel(400, "POST", "/v1/panel/mcp/tokens", map[string]any{"nombre": ""}, nil)

	var t1 turnos.Turno
	e.panel(201, "POST", "/v1/panel/turnos", map[string]any{"servicioId": e.servicio.ID, "inicio": en("2026-11-02 09:00").Format(time.RFC3339),
		"cliente": map[string]string{"nombre": "Lola", "telefono": "1144443333"}}, &t1)

	hs := e.herramientas(solo.Clave)
	if !slices.Equal(hs, []string{"ver_agenda", "buscar_horarios_libres", "ver_lista_de_espera", "ver_servicios_y_profesionales"}) {
		t.Fatalf("herramientas: %v", hs)
	}
	// Lo que no tiene permiso no aparece y, si se llama igual, es un error.
	for _, h := range []string{"crear_turno", "cancelar_turno", "buscar_clientes", "resumen_de_numeros", "inventada"} {
		r := e.pedir(solo.Clave, "tools/call", map[string]any{"name": h, "arguments": map[string]any{"turnoId": t1.ID}})
		if r.Error == nil || r.Error.Code != -32602 {
			t.Fatalf("%s sin permiso: %s", h, r.Result)
		}
	}
	// Sin «Ver clientes», la agenda dice quién viene pero no su contacto.
	var ag struct {
		Turnos []mcp.TurnoMCP `json:"turnos"`
	}
	e.llamar(solo.Clave, "ver_agenda", map[string]any{"desde": "2026-11-02", "hasta": "2026-11-08"}, false, &ag)
	if len(ag.Turnos) != 1 || ag.Turnos[0].Cliente.Nombre != "Lola" || ag.Turnos[0].Cliente.Telefono != "" {
		t.Fatalf("agenda: %+v", ag)
	}
	e.llamar(solo.Clave, "ver_agenda", map[string]any{"desde": "2026-11-02", "hasta": "2026-12-30"}, true, nil)

	// Se prende «Cancelar turnos» y «Ver números» (y se renombra).
	var tk mcp.Token
	e.panel(200, "PATCH", "/v1/panel/mcp/tokens/"+solo.Token.ID, map[string]any{"nombre": "Claude",
		"permisos": []string{"ver_agenda", "cancelar_turnos", "ver_numeros"}}, &tk)
	if tk.Nombre != "Claude" || len(tk.Permisos) != 3 {
		t.Fatalf("editada: %+v", tk)
	}
	hs = e.herramientas(solo.Clave)
	if !slices.Contains(hs, "cancelar_turno") || !slices.Contains(hs, "resumen_de_numeros") || slices.Contains(hs, "crear_turno") {
		t.Fatalf("herramientas: %v", hs)
	}
	var num struct {
		Turnos    int   `json:"turnos"`
		PorVenir  int   `json:"porVenir"`
		Facturado int64 `json:"facturado"`
	}
	e.llamar(solo.Clave, "resumen_de_numeros", map[string]any{"desde": "2026-11-01", "hasta": "2026-11-30"}, false, &num)
	if num.Turnos != 1 || num.PorVenir != 1 || num.Facturado != 0 {
		t.Fatalf("números: %+v", num)
	}

	// Otro negocio, con su clave, no ve ni toca los turnos de este.
	var o negocios.Negocio
	e.Esperar(201, P{Metodo: "POST", Ruta: "/v1/negocios", Quien: otra, Cuerpo: map[string]string{"slug": "otro", "nombre": "Otro"}}, &o)
	var ajena creado
	e.Esperar(201, P{Metodo: "POST", Ruta: "/v1/panel/mcp/tokens", Quien: otra, Negocio: o.ID,
		Cuerpo: map[string]any{"nombre": "Ajena", "permisos": []string{"ver_agenda", "cancelar_turnos"}}}, &ajena)
	e.llamar(ajena.Clave, "ver_agenda", map[string]any{"desde": "2026-11-02"}, false, &ag)
	if len(ag.Turnos) != 0 {
		t.Fatalf("agenda ajena: %+v", ag)
	}
	e.llamar(ajena.Clave, "cancelar_turno", map[string]any{"turnoId": t1.ID}, true, nil)
	e.Esperar(404, P{Metodo: "PATCH", Ruta: "/v1/panel/mcp/tokens/" + solo.Token.ID, Quien: otra, Negocio: o.ID,
		Cuerpo: map[string]any{"nombre": "mía"}}, nil)

	// Con la prueba vencida, el asistente sólo mira (como el panel).
	e.sql("UPDATE negocios SET prueba_hasta = $1 WHERE id = $2", e.Ahora.Add(-time.Hour), e.negocio)
	res := e.llamar(solo.Clave, "cancelar_turno", map[string]any{"turnoId": t1.ID}, true, nil)
	if !strings.Contains(res.Content[0].Text, "Suscribite") {
		t.Fatalf("sólo lectura: %s", res.Content[0].Text)
	}
	e.llamar(solo.Clave, "ver_agenda", map[string]any{"desde": "2026-11-02"}, false, nil)
	e.sql("UPDATE negocios SET prueba_hasta = $1 WHERE id = $2", e.Ahora.Add(72*time.Hour), e.negocio)
	var ct turnoMCP
	e.llamar(solo.Clave, "cancelar_turno", map[string]any{"turnoId": t1.ID}, false, &ct)
	if ct.Turno.Estado != "cancelado" {
		t.Fatalf("cancelado: %+v", ct)
	}

	// La recepción no gestiona las claves.
	e.sql(`INSERT INTO miembros (negocio_id, clerk_id, email, nombre, rol) VALUES ($1, $2, 'r@mail.com', 'Rocío', 'recepcion')`, e.negocio, recep)
	e.Esperar(403, P{Metodo: "GET", Ruta: "/v1/panel/mcp", Quien: recep, Negocio: e.negocio}, nil)
	e.Esperar(403, P{Metodo: "GET", Ruta: "/v1/panel/mcp/registro", Quien: recep, Negocio: e.negocio}, nil)

	// Revocada, deja de andar al instante; su registro queda.
	e.panel(204, "DELETE", "/v1/panel/mcp/tokens/"+solo.Token.ID, nil, nil)
	if rec := e.rpc(solo.Clave, map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/list"}); rec.Code != http.StatusUnauthorized {
		t.Fatalf("revocada: %d", rec.Code)
	}
	e.panel(404, "DELETE", "/v1/panel/mcp/tokens/"+solo.Token.ID, nil, nil)
	var reg struct {
		Acciones []mcp.Accion `json:"acciones"`
	}
	e.panel(200, "GET", "/v1/panel/mcp/registro?token="+solo.Token.ID, nil, &reg)
	if len(reg.Acciones) == 0 || reg.Acciones[0].Herramienta != "cancelar_turno" || reg.Acciones[0].Token != "Claude" {
		t.Fatalf("registro: %+v", reg.Acciones)
	}
	var panel struct {
		Tokens []mcp.Token `json:"tokens"`
	}
	e.panel(200, "GET", "/v1/panel/mcp", nil, &panel)
	if len(panel.Tokens) != 0 {
		t.Fatalf("revocada sigue en la lista: %+v", panel.Tokens)
	}
}
