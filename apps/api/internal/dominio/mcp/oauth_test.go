package mcp_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http/httptest"
	"net/url"
	"slices"
	"strings"
	"testing"

	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/mcp"
)

const vuelta = "https://claude.ai/api/mcp/auth_callback"

// form manda un POST application/x-www-form-urlencoded (como el token endpoint lo espera).
func (e *entorno) form(ruta string, v url.Values) (int, map[string]any) {
	e.t.Helper()
	req := httptest.NewRequest("POST", ruta, strings.NewReader(v.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	e.Handler.ServeHTTP(rec, req)
	out := map[string]any{}
	if rec.Body.Len() > 0 {
		_ = json.Unmarshal(rec.Body.Bytes(), &out)
	}
	return rec.Code, out
}

func (e *entorno) obtener(ruta string) *httptest.ResponseRecorder {
	e.t.Helper()
	rec := httptest.NewRecorder()
	e.Handler.ServeHTTP(rec, httptest.NewRequest("GET", ruta, nil))
	return rec
}

func (e *entorno) registrarCliente(cuerpo any) (int, map[string]any) {
	e.t.Helper()
	b, _ := json.Marshal(cuerpo)
	req := httptest.NewRequest("POST", "/v1/oauth/register", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	e.Handler.ServeHTTP(rec, req)
	out := map[string]any{}
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	return rec.Code, out
}

func desafio(v string) string {
	h := sha256.Sum256([]byte(v))
	return base64.RawURLEncoding.EncodeToString(h[:])
}

// autorizar hace el GET del navegador y devuelve a dónde lo manda la API.
func (e *entorno) autorizar(ps url.Values) *url.URL {
	e.t.Helper()
	rec := e.obtener("/v1/oauth/authorize?" + ps.Encode())
	if rec.Code != 302 {
		e.t.Fatalf("authorize: %d %s", rec.Code, rec.Body)
	}
	u, err := url.Parse(rec.Header().Get("Location"))
	if err != nil {
		e.t.Fatal(err)
	}
	return u
}

func parametros(cliente, verificador string) url.Values {
	return url.Values{"response_type": {"code"}, "client_id": {cliente}, "redirect_uri": {vuelta},
		"code_challenge": {desafio(verificador)}, "code_challenge_method": {"S256"}, "state": {"st-123"},
		"resource": {mcp.URLPorDefecto}, "scope": {"mcp"}}
}

// solicitud: authorize → /conectar?solicitud=… (lo que ve el front).
func (e *entorno) solicitud(cliente, verificador string) string {
	e.t.Helper()
	u := e.autorizar(parametros(cliente, verificador))
	if u.Host != "tengolibre.com.ar" || u.Path != "/conectar" || u.Query().Get("solicitud") == "" {
		e.t.Fatalf("authorize no va a /conectar: %s", u)
	}
	return u.Query().Get("solicitud")
}

// codigo: la persona permite y el front la manda al redirect_uri con el código.
func (e *entorno) codigo(solicitud string, permisos []string) string {
	e.t.Helper()
	var out struct{ Redirect string }
	e.Esperar(200, P{Metodo: "POST", Ruta: "/v1/oauth/solicitudes/" + solicitud + "/permitir", Quien: duenia,
		Cuerpo: map[string]any{"negocioId": e.negocio, "permisos": permisos}}, &out)
	u, _ := url.Parse(out.Redirect)
	q := u.Query()
	if !strings.HasPrefix(out.Redirect, vuelta+"?") || q.Get("state") != "st-123" || q.Get("iss") != "https://api.tengolibre.com.ar" || q.Get("code") == "" {
		e.t.Fatalf("vuelta: %s", out.Redirect)
	}
	return q.Get("code")
}

func (e *entorno) clienteNuevo() string {
	e.t.Helper()
	code, c := e.registrarCliente(map[string]any{"client_name": "Claude", "redirect_uris": []string{vuelta},
		"grant_types": []string{"authorization_code", "refresh_token"}, "token_endpoint_auth_method": "none", "logo_uri": "https://claude.ai/x.png"})
	if code != 201 || c["token_endpoint_auth_method"] != "none" {
		e.t.Fatalf("registro: %d %v", code, c)
	}
	return c["client_id"].(string)
}

func canje(cliente, code, verificador string) url.Values {
	return url.Values{"grant_type": {"authorization_code"}, "code": {code}, "client_id": {cliente}, "redirect_uri": {vuelta},
		"code_verifier": {verificador}, "resource": {mcp.URLPorDefecto}}
}

func ping(e *entorno, token string) int {
	return e.rpc(token, map[string]any{"jsonrpc": "2.0", "id": 1, "method": "ping"}).Code
}

func TestOAuthDePuntaAPunta(t *testing.T) {
	e := nuevo(t)

	// 1. Sin token, el 401 dice dónde está el Protected Resource Metadata.
	rec := e.rpc("", map[string]any{"jsonrpc": "2.0", "id": 1, "method": "initialize"})
	prm := "https://api.tengolibre.com.ar/.well-known/oauth-protected-resource/v1/mcp"
	if rec.Code != 401 || !strings.Contains(rec.Header().Get("WWW-Authenticate"), `resource_metadata="`+prm+`"`) {
		t.Fatalf("401: %d %q", rec.Code, rec.Header().Get("WWW-Authenticate"))
	}
	for _, p := range []string{"/.well-known/oauth-protected-resource/v1/mcp", "/.well-known/oauth-protected-resource"} {
		var m struct {
			Resource string   `json:"resource"`
			AS       []string `json:"authorization_servers"`
		}
		r := e.obtener(p)
		if err := json.Unmarshal(r.Body.Bytes(), &m); err != nil || r.Code != 200 || m.Resource != mcp.URLPorDefecto ||
			!slices.Equal(m.AS, []string{"https://api.tengolibre.com.ar"}) {
			t.Fatalf("%s: %d %s", p, r.Code, r.Body)
		}
	}
	var as map[string]any
	r := e.obtener("/.well-known/oauth-authorization-server")
	if err := json.Unmarshal(r.Body.Bytes(), &as); err != nil || as["issuer"] != "https://api.tengolibre.com.ar" ||
		as["authorization_endpoint"] != "https://api.tengolibre.com.ar/v1/oauth/authorize" ||
		as["token_endpoint"] != "https://api.tengolibre.com.ar/v1/oauth/token" ||
		as["registration_endpoint"] != "https://api.tengolibre.com.ar/v1/oauth/register" ||
		as["revocation_endpoint"] != "https://api.tengolibre.com.ar/v1/oauth/revoke" ||
		!strings.Contains(r.Body.String(), `"code_challenge_methods_supported":["S256"]`) ||
		!strings.Contains(r.Body.String(), `"token_endpoint_auth_methods_supported":["none"]`) {
		t.Fatalf("metadatos: %s", r.Body)
	}

	// 2. Registro dinámico, authorize y la pantalla de /conectar.
	cliente := e.clienteNuevo()
	verificador := strings.Repeat("v3rificador-", 5)
	sol := e.solicitud(cliente, verificador)
	var s mcp.Solicitud
	e.Esperar(200, P{Metodo: "GET", Ruta: "/v1/oauth/solicitudes/" + sol, Quien: duenia}, &s)
	if s.Cliente != "Claude" || s.Destino != "claude.ai" || len(s.Negocios) != 1 || s.Negocios[0].ID != e.negocio ||
		len(s.Permisos) != 5 || !slices.Equal(s.PorDefecto, mcp.PermisosPorDefecto) {
		t.Fatalf("solicitud: %+v", s)
	}
	// Sin sesión no se ve.
	e.Esperar(401, P{Metodo: "GET", Ruta: "/v1/oauth/solicitudes/" + sol}, nil)
	code := e.codigo(sol, []string{mcp.VerAgenda, mcp.MoverTurnos})
	// Ya decidida: no se puede volver a permitir.
	e.Esperar(404, P{Metodo: "POST", Ruta: "/v1/oauth/solicitudes/" + sol + "/permitir", Quien: duenia,
		Cuerpo: map[string]any{"negocioId": e.negocio, "permisos": []string{}}}, nil)

	// 3. Canje con PKCE.
	st, tok := e.form("/v1/oauth/token", canje(cliente, code, verificador))
	acceso, _ := tok["access_token"].(string)
	refresco, _ := tok["refresh_token"].(string)
	if st != 200 || tok["token_type"] != "Bearer" || tok["expires_in"] != float64(3600) || !strings.HasPrefix(acceso, mcp.Prefijo) ||
		!strings.HasPrefix(refresco, mcp.PrefijoRefresco) || tok["scope"] != "ver_agenda mover_turnos" {
		t.Fatalf("token: %d %v", st, tok)
	}

	// 4. El token anda en /v1/mcp con sus permisos, y el registro dice «Claude».
	hs := e.herramientas(acceso)
	if !slices.Contains(hs, "ver_agenda") || !slices.Contains(hs, "crear_turno") || slices.Contains(hs, "buscar_clientes") {
		t.Fatalf("herramientas: %v", hs)
	}
	e.pedir(acceso, "initialize", map[string]any{"protocolVersion": "2025-06-18", "clientInfo": map[string]string{"name": "claude-ai", "version": "0.1.0"}})
	var nt turnoMCP
	e.llamar(acceso, "crear_turno", map[string]any{"servicioId": "Corte", "fecha": "2026-11-02", "hora": "09:00",
		"cliente": map[string]string{"nombre": "María Gómez", "telefono": "11 5555-1234"}}, false, &nt)
	var reg struct {
		Acciones []mcp.Accion `json:"acciones"`
	}
	e.panel(200, "GET", "/v1/panel/mcp/registro", nil, &reg)
	if len(reg.Acciones) != 1 || reg.Acciones[0].Token != "Claude" || reg.Acciones[0].Tipo != "oauth" || reg.Acciones[0].Cliente != "claude-ai 0.1.0" {
		t.Fatalf("registro: %+v", reg.Acciones)
	}
	var panel struct {
		Tokens     []mcp.Token    `json:"tokens"`
		Conexiones []mcp.Conexion `json:"conexiones"`
	}
	e.panel(200, "GET", "/v1/panel/mcp", nil, &panel)
	if len(panel.Tokens) != 0 || len(panel.Conexiones) != 1 || panel.Conexiones[0].Nombre != "Claude" || panel.Conexiones[0].UsadoEn == nil ||
		panel.Conexiones[0].AutorizadoPor != duenia || !slices.Equal(panel.Conexiones[0].Permisos, []string{mcp.VerAgenda, mcp.MoverTurnos}) {
		t.Fatalf("panel: %+v", panel)
	}
	// La conexión no es una clave: no se edita ni se revoca por /tokens.
	e.panel(404, "DELETE", "/v1/panel/mcp/tokens/"+panel.Conexiones[0].ID, nil, nil)
	// Los permisos se cambian desde el panel y valen al instante.
	e.panel(200, "PATCH", "/v1/panel/mcp/conexiones/"+panel.Conexiones[0].ID, map[string]any{"permisos": []string{mcp.VerAgenda}}, nil)
	if hs := e.herramientas(acceso); slices.Contains(hs, "crear_turno") {
		t.Fatalf("sigue con crear_turno: %v", hs)
	}

	// 5. Refresco: rota los dos; el refresco viejo y el acceso viejo dejan de servir.
	st, tok2 := e.form("/v1/oauth/token", url.Values{"grant_type": {"refresh_token"}, "refresh_token": {refresco}, "client_id": {cliente},
		"resource": {mcp.URLPorDefecto}})
	acceso2, _ := tok2["access_token"].(string)
	refresco2, _ := tok2["refresh_token"].(string)
	if st != 200 || acceso2 == "" || refresco2 == "" || acceso2 == acceso || refresco2 == refresco || tok2["scope"] != "ver_agenda" {
		t.Fatalf("refresco: %d %v", st, tok2)
	}
	if st, out := e.form("/v1/oauth/token", url.Values{"grant_type": {"refresh_token"}, "refresh_token": {refresco}, "client_id": {cliente}}); st != 400 || out["error"] != "invalid_grant" {
		t.Fatalf("refresco reusado: %d %v", st, out)
	}
	if c := ping(e, acceso); c != 401 {
		t.Fatalf("acceso viejo: %d", c)
	}
	if c := ping(e, acceso2); c != 200 {
		t.Fatalf("acceso nuevo: %d", c)
	}
	// Otro cliente no puede usar este refresco.
	otro := e.clienteNuevo()
	if st, out := e.form("/v1/oauth/token", url.Values{"grant_type": {"refresh_token"}, "refresh_token": {refresco2}, "client_id": {otro}}); st != 400 || out["error"] != "invalid_grant" {
		t.Fatalf("refresco de otro cliente: %d %v", st, out)
	}

	// 6. El token de acceso vencido: 401 con invalid_token.
	e.sql("UPDATE mcp_tokens SET acceso_vence = now() - interval '1 second' WHERE oauth_cliente IS NOT NULL")
	rec = e.rpc(acceso2, map[string]any{"jsonrpc": "2.0", "id": 1, "method": "ping"})
	if rec.Code != 401 || !strings.Contains(rec.Header().Get("WWW-Authenticate"), `error="invalid_token"`) {
		t.Fatalf("vencido: %d %q", rec.Code, rec.Header().Get("WWW-Authenticate"))
	}
	st, tok3 := e.form("/v1/oauth/token", url.Values{"grant_type": {"refresh_token"}, "refresh_token": {refresco2}, "client_id": {cliente}})
	acceso3, _ := tok3["access_token"].(string)
	refresco3, _ := tok3["refresh_token"].(string)
	if st != 200 || ping(e, acceso3) != 200 {
		t.Fatalf("refresco después de vencer: %d %v", st, tok3)
	}

	// 7. Revocación (RFC 7009): siempre 200; la conexión se cae entera.
	if st, _ := e.form("/v1/oauth/revoke", url.Values{"token": {"tlmcpr_inventado"}, "client_id": {cliente}}); st != 200 {
		t.Fatalf("revocar inventado: %d", st)
	}
	if st, _ := e.form("/v1/oauth/revoke", url.Values{"token": {refresco3}, "client_id": {cliente}, "token_type_hint": {"refresh_token"}}); st != 200 {
		t.Fatalf("revocar: %d", st)
	}
	if c := ping(e, acceso3); c != 401 {
		t.Fatalf("acceso tras revocar: %d", c)
	}
	if st, out := e.form("/v1/oauth/token", url.Values{"grant_type": {"refresh_token"}, "refresh_token": {refresco3}, "client_id": {cliente}}); st != 400 || out["error"] != "invalid_grant" {
		t.Fatalf("refresco tras revocar: %d %v", st, out)
	}
	e.panel(200, "GET", "/v1/panel/mcp", nil, &panel)
	if len(panel.Conexiones) != 0 {
		t.Fatalf("revocada sigue en el panel: %+v", panel.Conexiones)
	}
	// El registro queda, con el nombre del cliente.
	e.panel(200, "GET", "/v1/panel/mcp/registro", nil, &reg)
	if len(reg.Acciones) != 1 || reg.Acciones[0].Token != "Claude" {
		t.Fatalf("registro tras revocar: %+v", reg.Acciones)
	}

	// 8. Una segunda conexión se desconecta desde el panel.
	v2 := strings.Repeat("otro-verificador_", 3)
	code = e.codigo(e.solicitud(cliente, v2), []string{})
	st, tok = e.form("/v1/oauth/token", canje(cliente, code, v2))
	if st != 200 {
		t.Fatalf("segunda conexión: %d %v", st, tok)
	}
	e.panel(200, "GET", "/v1/panel/mcp", nil, &panel)
	if len(panel.Conexiones) != 1 || len(panel.Conexiones[0].Permisos) != 0 {
		t.Fatalf("segunda en el panel: %+v", panel.Conexiones)
	}
	if hs := e.herramientas(tok["access_token"].(string)); len(hs) != 0 {
		t.Fatalf("sin permisos ve herramientas: %v", hs)
	}
	e.panel(204, "DELETE", "/v1/panel/mcp/conexiones/"+panel.Conexiones[0].ID, nil, nil)
	if c := ping(e, tok["access_token"].(string)); c != 401 {
		t.Fatalf("desconectada: %d", c)
	}
	e.panel(404, "DELETE", "/v1/panel/mcp/conexiones/"+panel.Conexiones[0].ID, nil, nil)
}

func TestOAuthErrores(t *testing.T) {
	e := nuevo(t)

	// Registro: redirect_uris inválidos.
	for _, u := range []string{"http://claude.ai/cb", "javascript:alert(1)", "https://claude.ai/cb#x", "ftp://claude.ai/cb", "/relativa"} {
		if code, out := e.registrarCliente(map[string]any{"client_name": "Malo", "redirect_uris": []string{u}}); code != 400 || out["error"] != "invalid_redirect_uri" {
			t.Fatalf("redirect %q: %d %v", u, code, out)
		}
	}
	if code, out := e.registrarCliente(map[string]any{"client_name": "Sin"}); code != 400 || out["error"] != "invalid_redirect_uri" {
		t.Fatalf("sin redirect: %d %v", code, out)
	}
	if code, out := e.registrarCliente(map[string]any{"redirect_uris": []string{vuelta}, "grant_types": []string{"client_credentials"}}); code != 400 || out["error"] != "invalid_client_metadata" {
		t.Fatalf("grant ajeno: %d %v", code, out)
	}
	// Un cliente local (Claude Code, el inspector) sí.
	if code, _ := e.registrarCliente(map[string]any{"client_name": "Local", "redirect_uris": []string{"http://localhost:33418/callback", "http://127.0.0.1:6274/oauth/callback"}}); code != 201 {
		t.Fatalf("local: %d", code)
	}
	cliente := e.clienteNuevo()
	verificador := strings.Repeat("verificador.", 5)

	// Authorize: cliente desconocido o redirect no registrado → error en /conectar, nunca al redirect.
	ps := parametros("tlcli_noexisteNoexisteNo", verificador)
	if u := e.autorizar(ps); u.Host != "tengolibre.com.ar" || u.Query().Get("error") != "cliente_desconocido" {
		t.Fatalf("cliente desconocido: %s", u)
	}
	ps = parametros(cliente, verificador)
	ps.Set("redirect_uri", "https://malo.example/cb")
	if u := e.autorizar(ps); u.Host != "tengolibre.com.ar" || u.Query().Get("error") != "redirect_invalido" {
		t.Fatalf("redirect no registrado: %s", u)
	}
	// Sin PKCE, con plain o con resource ajeno → vuelve al cliente con el error y el state.
	for nombre, cambiar := range map[string]func(url.Values){
		"sin pkce":      func(v url.Values) { v.Del("code_challenge") },
		"plain":         func(v url.Values) { v.Set("code_challenge_method", "plain") },
		"response_type": func(v url.Values) { v.Set("response_type", "token") },
		"resource":      func(v url.Values) { v.Set("resource", "https://otro.example/mcp") },
	} {
		ps := parametros(cliente, verificador)
		cambiar(ps)
		u := e.autorizar(ps)
		want := map[string]string{"sin pkce": "invalid_request", "plain": "invalid_request", "response_type": "unsupported_response_type", "resource": "invalid_target"}[nombre]
		if !strings.HasPrefix(u.String(), vuelta+"?") || u.Query().Get("error") != want || u.Query().Get("state") != "st-123" {
			t.Fatalf("%s: %s", nombre, u)
		}
	}

	// /conectar: alguien que no es del negocio no puede dar permiso; cancelar vuelve con access_denied.
	sol := e.solicitud(cliente, verificador)
	e.Esperar(403, P{Metodo: "POST", Ruta: "/v1/oauth/solicitudes/" + sol + "/permitir", Quien: otra,
		Cuerpo: map[string]any{"negocioId": e.negocio, "permisos": []string{mcp.VerAgenda}}}, nil)
	var s mcp.Solicitud
	e.Esperar(200, P{Metodo: "GET", Ruta: "/v1/oauth/solicitudes/" + sol, Quien: otra}, &s)
	if len(s.Negocios) != 0 {
		t.Fatalf("otra ve negocios: %+v", s.Negocios)
	}
	e.Esperar(400, P{Metodo: "POST", Ruta: "/v1/oauth/solicitudes/" + sol + "/permitir", Quien: duenia,
		Cuerpo: map[string]any{"negocioId": e.negocio, "permisos": []string{"todo"}}}, nil)
	var c struct{ Redirect string }
	e.Esperar(200, P{Metodo: "POST", Ruta: "/v1/oauth/solicitudes/" + sol + "/cancelar", Quien: duenia}, &c)
	if u, _ := url.Parse(c.Redirect); u.Query().Get("error") != "access_denied" || u.Query().Get("state") != "st-123" {
		t.Fatalf("cancelar: %s", c.Redirect)
	}
	e.Esperar(404, P{Metodo: "GET", Ruta: "/v1/oauth/solicitudes/" + sol, Quien: duenia}, nil)
	e.Esperar(404, P{Metodo: "GET", Ruta: "/v1/oauth/solicitudes/inventada", Quien: duenia}, nil)

	// Token: PKCE mal, resource ajeno, redirect distinto, otro cliente, sin form.
	code := e.codigo(e.solicitud(cliente, verificador), mcp.PermisosPorDefecto)
	malo := func(nombre string, v url.Values, estado int, error string) {
		t.Helper()
		if st, out := e.form("/v1/oauth/token", v); st != estado || out["error"] != error {
			t.Fatalf("%s: %d %v", nombre, st, out)
		}
	}
	v := canje(cliente, code, strings.Repeat("equivocado-", 5))
	malo("pkce mal", v, 400, "invalid_grant")
	v = canje(cliente, code, verificador)
	v.Del("code_verifier")
	malo("sin verificador", v, 400, "invalid_request")
	v = canje(cliente, code, verificador)
	v.Set("resource", "https://otro.example/mcp")
	malo("resource ajeno", v, 400, "invalid_target")
	v = canje(cliente, code, verificador)
	v.Set("redirect_uri", "https://claude.ai/otra")
	malo("redirect distinto", v, 400, "invalid_grant")
	malo("otro cliente", canje(e.clienteNuevo(), code, verificador), 400, "invalid_grant")
	v = canje(cliente, code, verificador)
	v.Del("client_id")
	malo("sin cliente", v, 401, "invalid_client")
	malo("grant ajeno", url.Values{"grant_type": {"password"}, "client_id": {cliente}}, 400, "unsupported_grant_type")
	req := httptest.NewRequest("POST", "/v1/oauth/token", strings.NewReader(`{"grant_type":"authorization_code"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	e.Handler.ServeHTTP(rec, req)
	if rec.Code != 400 {
		t.Fatalf("json: %d", rec.Code)
	}

	// Los errores no gastaron el código: el canje bueno anda una vez.
	st, tok := e.form("/v1/oauth/token", canje(cliente, code, verificador))
	if st != 200 {
		t.Fatalf("canje bueno: %d %v", st, tok)
	}
	acceso := tok["access_token"].(string)
	if ping(e, acceso) != 200 {
		t.Fatal("el acceso no anda")
	}
	// Reusar el código: invalid_grant, y la conexión que salió de él se cae.
	malo("código reusado", canje(cliente, code, verificador), 400, "invalid_grant")
	if c := ping(e, acceso); c != 401 {
		t.Fatalf("tras reusar el código, el acceso sigue: %d", c)
	}

	// Código vencido.
	code = e.codigo(e.solicitud(cliente, verificador), mcp.PermisosPorDefecto)
	e.sql("UPDATE auth.oauth_autorizaciones SET codigo_vence = now() - interval '1 second' WHERE canjeada_en IS NULL AND codigo_hash IS NOT NULL")
	malo("código vencido", canje(cliente, code, verificador), 400, "invalid_grant")
	malo("código inventado", canje(cliente, "inventado", verificador), 400, "invalid_grant")

	// Solicitud vencida: /conectar dice que no existe.
	sol = e.solicitud(cliente, verificador)
	e.sql("UPDATE auth.oauth_autorizaciones SET vence_en = now() - interval '1 second' WHERE decidida_en IS NULL")
	e.Esperar(404, P{Metodo: "GET", Ruta: "/v1/oauth/solicitudes/" + sol, Quien: duenia}, nil)

	// Una clave del panel no se revoca por el endpoint de OAuth.
	k := e.clave(map[string]string{"nombre": "Claude Code"})
	if st, _ := e.form("/v1/oauth/revoke", url.Values{"token": {k.Clave}}); st != 200 || ping(e, k.Clave) != 200 {
		t.Fatalf("revocar una clave por OAuth: %d", st)
	}
	// CORS abierto en los metadatos (clientes en el navegador).
	req = httptest.NewRequest("GET", "/.well-known/oauth-authorization-server", nil)
	req.Header.Set("Origin", "https://inspector.example")
	rec = httptest.NewRecorder()
	e.Handler.ServeHTTP(rec, req)
	if rec.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Fatalf("cors: %v", rec.Header())
	}
}
