package mcp

// OAuth 2.1 para el servidor MCP (spec de autorización de MCP 2025-06-18): los conectores que no
// dejan cargar encabezados (claude.ai, ChatGPT) descubren el authorization server por el 401 de
// /v1/mcp (RFC 9728), se registran solos (RFC 7591), mandan a la persona a /conectar del front a
// dar permiso y canjean el código con PKCE S256 (RFC 7636) por un token de acceso y uno de
// refresco que rota. El parámetro resource (RFC 8707) tiene que ser la URL del MCP.
//
// Una conexión OAuth es una fila de mcp_tokens con oauth_cliente: el token de acceso se resuelve
// igual que una clave del panel (mismo negocio, permisos y registro), sólo que vence.

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"

	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/auditoria"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/problema"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/sesion"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/web"
)

// Duraciones y topes del OAuth.
const (
	// DuracionAcceso del token de acceso.
	DuracionAcceso = time.Hour
	// DuracionRefresco del token de refresco: cada uso la renueva (una conexión que no se usa en
	// 60 días se cae sola).
	DuracionRefresco = 60 * 24 * time.Hour
	// DuracionSolicitud: lo que tiene la persona para ingresar y decidir en /conectar.
	DuracionSolicitud = 30 * time.Minute
	// DuracionCodigo: el código va y vuelve en segundos.
	DuracionCodigo = 2 * time.Minute
	// MaxConexiones OAuth vigentes por negocio (aparte de las MaxTokens claves).
	MaxConexiones = 10
	// MaxRegistrosPorHora: tope global del registro dinámico (es abierto).
	MaxRegistrosPorHora = 300
	// PrefijoRefresco de los tokens de refresco (los de acceso usan Prefijo, como las claves).
	PrefijoRefresco = "tlmcpr_"
)

var (
	reVerificador = regexp.MustCompile(`^[A-Za-z0-9._~-]{43,128}$`)
	reCliente     = regexp.MustCompile(`^tlcli_[A-Za-z0-9_-]{16,64}$`)
)

// emisor: el origen de la URL del MCP (https://api.tengolibre.com.ar), que es a la vez el
// authorization server.
func (s *Servicio) emisor() string {
	u, err := url.Parse(s.url())
	if err != nil {
		return strings.TrimSuffix(s.url(), "/v1/mcp")
	}
	return u.Scheme + "://" + u.Host
}

func (s *Servicio) sitio() string {
	if s.Sitio != "" {
		return strings.TrimSuffix(s.Sitio, "/")
	}
	return "https://tengolibre.com.ar"
}

// MetadatosDelRecurso: la URL del Protected Resource Metadata (la que va en el 401).
func (s *Servicio) metadatosDelRecurso() string {
	u, err := url.Parse(s.url())
	if err != nil {
		return s.emisor() + "/.well-known/oauth-protected-resource"
	}
	return s.emisor() + "/.well-known/oauth-protected-resource" + strings.TrimSuffix(u.Path, "/")
}

// mismoRecurso compara un resource con la URL del MCP (esquema y host sin mayúsculas, sin la barra
// final).
func (s *Servicio) mismoRecurso(r string) bool {
	norm := func(v string) string {
		u, err := url.Parse(v)
		if err != nil || u.Fragment != "" || u.RawQuery != "" {
			return "\x00"
		}
		return strings.ToLower(u.Scheme) + "://" + strings.ToLower(u.Host) + strings.TrimSuffix(u.Path, "/")
	}
	return norm(r) == norm(s.url())
}

func rutasOAuth(s *Servicio, mux *http.ServeMux) {
	abierto := func(h http.HandlerFunc) http.HandlerFunc { return conCORS(h) }
	for _, p := range []string{"/.well-known/oauth-protected-resource", "/.well-known/oauth-protected-resource/v1/mcp"} {
		mux.HandleFunc("GET "+p, abierto(s.metadatosRecurso))
		mux.HandleFunc("OPTIONS "+p, abierto(preflight))
	}
	mux.HandleFunc("GET /.well-known/oauth-authorization-server", abierto(s.metadatosServidor))
	mux.HandleFunc("OPTIONS /.well-known/oauth-authorization-server", abierto(preflight))
	mux.HandleFunc("POST /v1/oauth/register", abierto(s.registrarCliente))
	mux.HandleFunc("OPTIONS /v1/oauth/register", abierto(preflight))
	mux.HandleFunc("GET /v1/oauth/authorize", s.autorizar)
	mux.HandleFunc("POST /v1/oauth/token", abierto(s.token))
	mux.HandleFunc("OPTIONS /v1/oauth/token", abierto(preflight))
	mux.HandleFunc("POST /v1/oauth/revoke", abierto(s.revocar))
	mux.HandleFunc("OPTIONS /v1/oauth/revoke", abierto(preflight))
	// La pantalla de consentimiento (el front, con la sesión de la persona).
	mux.HandleFunc("GET /v1/oauth/solicitudes/{id}", s.Sesion.Persona(web.Envolver(s.verSolicitud)))
	mux.HandleFunc("POST /v1/oauth/solicitudes/{id}/permitir", s.Sesion.Persona(web.Envolver(s.permitir)))
	mux.HandleFunc("POST /v1/oauth/solicitudes/{id}/cancelar", s.Sesion.Persona(web.Envolver(s.cancelar)))
}

// conCORS: lo del OAuth no usa cookies, así que cualquier origen puede llamarlo (clientes MCP que
// corren en el navegador, como el inspector).
func conCORS(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Origin") != "" && w.Header().Get("Access-Control-Allow-Origin") == "" {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, MCP-Protocol-Version")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST")
			w.Header().Set("Access-Control-Max-Age", "600")
		}
		h(w, r)
	}
}

func preflight(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) }

func escribirOAuth(w http.ResponseWriter, estado int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
	w.WriteHeader(estado)
	_ = json.NewEncoder(w).Encode(v)
}

// errorOAuth: {error, error_description} (RFC 6749 §5.2).
func errorOAuth(w http.ResponseWriter, estado int, codigo, descripcion string) {
	escribirOAuth(w, estado, map[string]string{"error": codigo, "error_description": descripcion})
}

// ---- Metadatos

func (s *Servicio) metadatosRecurso(w http.ResponseWriter, _ *http.Request) {
	escribirOAuth(w, http.StatusOK, map[string]any{
		"resource":                 s.url(),
		"authorization_servers":    []string{s.emisor()},
		"bearer_methods_supported": []string{"header"},
		"resource_name":            "Tengo Libre",
		"resource_documentation":   s.sitio() + "/panel/integraciones",
	})
}

func (s *Servicio) metadatosServidor(w http.ResponseWriter, _ *http.Request) {
	e := s.emisor()
	escribirOAuth(w, http.StatusOK, map[string]any{
		"issuer":                                         e,
		"authorization_endpoint":                         e + "/v1/oauth/authorize",
		"token_endpoint":                                 e + "/v1/oauth/token",
		"registration_endpoint":                          e + "/v1/oauth/register",
		"revocation_endpoint":                            e + "/v1/oauth/revoke",
		"response_types_supported":                       []string{"code"},
		"response_modes_supported":                       []string{"query"},
		"grant_types_supported":                          []string{"authorization_code", "refresh_token"},
		"code_challenge_methods_supported":               []string{"S256"},
		"token_endpoint_auth_methods_supported":          []string{"none"},
		"revocation_endpoint_auth_methods_supported":     []string{"none"},
		"authorization_response_iss_parameter_supported": true,
		"service_documentation":                          s.sitio() + "/panel/integraciones",
	})
}

// ---- Registro dinámico (RFC 7591)

// redirectValido: https con host, o http a localhost/127.0.0.1/[::1] (clientes locales). Sin
// fragmento ni credenciales.
func redirectValido(v string) bool {
	if len(v) > 500 {
		return false
	}
	u, err := url.Parse(v)
	if err != nil || u.Host == "" || u.Fragment != "" || u.User != nil || strings.ContainsAny(v, " \t\r\n") {
		return false
	}
	switch u.Scheme {
	case "https":
		return true
	case "http":
		h := u.Hostname()
		return h == "localhost" || h == "127.0.0.1" || h == "::1"
	}
	return false
}

func aleatorio(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

func (s *Servicio) registrarCliente(w http.ResponseWriter, r *http.Request) {
	var in struct {
		RedirectURIs            []string `json:"redirect_uris"`
		ClientName              string   `json:"client_name"`
		GrantTypes              []string `json:"grant_types"`
		ResponseTypes           []string `json:"response_types"`
		TokenEndpointAuthMethod string   `json:"token_endpoint_auth_method"`
	}
	// Los clientes mandan más metadatos (logo_uri, scope, software_id…): se ignoran.
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&in); err != nil {
		errorOAuth(w, http.StatusBadRequest, "invalid_client_metadata", "El cuerpo tiene que ser JSON.")
		return
	}
	if len(in.RedirectURIs) == 0 || len(in.RedirectURIs) > 10 {
		errorOAuth(w, http.StatusBadRequest, "invalid_redirect_uri", "Mandá de 1 a 10 redirect_uris.")
		return
	}
	uris := []string{}
	for _, u := range in.RedirectURIs {
		if !redirectValido(u) {
			errorOAuth(w, http.StatusBadRequest, "invalid_redirect_uri", "Los redirect_uris van con https (o http a localhost): "+u)
			return
		}
		if !slices.Contains(uris, u) {
			uris = append(uris, u)
		}
	}
	for _, g := range in.GrantTypes {
		if g != "authorization_code" && g != "refresh_token" {
			errorOAuth(w, http.StatusBadRequest, "invalid_client_metadata", "Sólo authorization_code y refresh_token.")
			return
		}
	}
	for _, t := range in.ResponseTypes {
		if t != "code" {
			errorOAuth(w, http.StatusBadRequest, "invalid_client_metadata", "Sólo response_type code.")
			return
		}
	}
	// Clientes públicos: sin secreto. Si pide otro método, se le contesta none (RFC 7591 §3.2.1).
	nombre := strings.TrimSpace(in.ClientName)
	if nombre == "" {
		nombre = "Asistente sin nombre"
	}
	if utf8.RuneCountInString(nombre) > 60 {
		nombre = string([]rune(nombre)[:60])
	}
	id := "tlcli_" + aleatorio(18)
	var creado time.Time
	err := s.Pool.EnTx(r.Context(), func(tx pgx.Tx) error {
		ctx := r.Context()
		// Los que nunca se usaron, a la semana se van.
		if _, err := tx.Exec(ctx, "DELETE FROM auth.oauth_clientes WHERE usado_en IS NULL AND creado_en < now() - interval '7 days'"); err != nil {
			return err
		}
		var n int
		if err := tx.QueryRow(ctx, "SELECT count(*) FROM auth.oauth_clientes WHERE creado_en > now() - interval '1 hour'").Scan(&n); err != nil {
			return err
		}
		if n >= MaxRegistrosPorHora {
			return errTope
		}
		return tx.QueryRow(ctx, "INSERT INTO auth.oauth_clientes (id, nombre, redirect_uris) VALUES ($1, $2, $3) RETURNING creado_en",
			id, nombre, uris).Scan(&creado)
	})
	if errors.Is(err, errTope) {
		w.Header().Set("Retry-After", "3600")
		errorOAuth(w, http.StatusTooManyRequests, "temporarily_unavailable", "Demasiados registros: probá en un rato.")
		return
	}
	if err != nil {
		slog.Error("oauth: no se pudo registrar el cliente", "error", err)
		errorOAuth(w, http.StatusInternalServerError, "server_error", "No pudimos registrar el cliente.")
		return
	}
	escribirOAuth(w, http.StatusCreated, map[string]any{
		"client_id":                  id,
		"client_id_issued_at":        creado.Unix(),
		"client_name":                nombre,
		"redirect_uris":              uris,
		"grant_types":                []string{"authorization_code", "refresh_token"},
		"response_types":             []string{"code"},
		"token_endpoint_auth_method": "none",
	})
}

var errTope = errors.New("tope")

// ---- Autorización

type cliente struct {
	id, nombre string
	redirects  []string
}

func (s *Servicio) cliente(ctx context.Context, id string) (*cliente, error) {
	if !reCliente.MatchString(id) {
		return nil, nil
	}
	c := &cliente{id: id}
	err := s.Pool.EnTx(ctx, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, "SELECT nombre, redirect_uris FROM auth.oauth_clientes WHERE id = $1", id).Scan(&c.nombre, &c.redirects)
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return c, err
}

// conParametros agrega los parámetros a la URL de vuelta (conserva su query).
func conParametros(base string, ps map[string]string) string {
	u, err := url.Parse(base)
	if err != nil {
		return base
	}
	q := u.Query()
	for k, v := range ps {
		if v != "" {
			q.Set(k, v)
		}
	}
	u.RawQuery = q.Encode()
	return u.String()
}

// autorizar: GET del navegador. Con un cliente o redirect_uri que no cierra no se vuelve al cliente
// (podría ser cualquiera): se muestra el error en /conectar. El resto de los errores vuelve al
// redirect_uri con error y state. Si está todo bien, el pedido queda guardado y la persona va a
// /conectar a decidir.
func (s *Servicio) autorizar(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	q := r.URL.Query()
	aConectar := func(codigo string) {
		http.Redirect(w, r, s.sitio()+"/conectar?error="+codigo, http.StatusFound)
	}
	c, err := s.cliente(ctx, q.Get("client_id"))
	if err != nil {
		slog.Error("oauth: no se pudo leer el cliente", "error", err)
		aConectar("interno")
		return
	}
	if c == nil {
		aConectar("cliente_desconocido")
		return
	}
	redirect := q.Get("redirect_uri")
	if redirect == "" && len(c.redirects) == 1 {
		redirect = c.redirects[0]
	}
	if !slices.Contains(c.redirects, redirect) {
		aConectar("redirect_invalido")
		return
	}
	state := q.Get("state")
	volver := func(codigo, descripcion string) {
		http.Redirect(w, r, conParametros(redirect, map[string]string{"error": codigo, "error_description": descripcion,
			"state": state, "iss": s.emisor()}), http.StatusFound)
	}
	switch {
	case len(state) > 500:
		volver("invalid_request", "state es demasiado largo.")
		return
	case q.Get("response_type") != "code":
		volver("unsupported_response_type", "Sólo response_type=code.")
		return
	case q.Get("code_challenge") == "" || q.Get("code_challenge_method") != "S256":
		volver("invalid_request", "PKCE es obligatorio: code_challenge con code_challenge_method=S256.")
		return
	case !reVerificador.MatchString(q.Get("code_challenge")):
		volver("invalid_request", "code_challenge no es válido.")
		return
	}
	recurso := s.url()
	if v, ok := q["resource"]; ok {
		if len(v) != 1 || !s.mismoRecurso(v[0]) {
			volver("invalid_target", "resource tiene que ser "+s.url()+".")
			return
		}
	}
	id := aleatorio(24)
	err = s.Pool.EnTx(ctx, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, "DELETE FROM auth.oauth_autorizaciones WHERE vence_en < now() - interval '1 day'"); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `INSERT INTO auth.oauth_autorizaciones (hash, cliente_id, redirect_uri, state, desafio, recurso, vence_en)
			VALUES ($1, $2, $3, $4, $5, $6, now() + $7::interval)`, hashDe(id), c.id, redirect, state, q.Get("code_challenge"),
			recurso, strconv.Itoa(int(DuracionSolicitud.Seconds()))+" seconds")
		return err
	})
	if err != nil {
		slog.Error("oauth: no se pudo guardar el pedido", "error", err)
		volver("server_error", "No pudimos guardar el pedido.")
		return
	}
	http.Redirect(w, r, s.sitio()+"/conectar?solicitud="+id, http.StatusFound)
}

// ---- Token

func (s *Servicio) token(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
	if err := r.ParseForm(); err != nil || !strings.HasPrefix(r.Header.Get("Content-Type"), "application/x-www-form-urlencoded") {
		errorOAuth(w, http.StatusBadRequest, "invalid_request", "El pedido va como application/x-www-form-urlencoded.")
		return
	}
	f := r.PostForm
	// Sin secreto: un cliente confidencial que manda Basic igual se identifica por client_id.
	clienteID := f.Get("client_id")
	if u, _, ok := r.BasicAuth(); ok && clienteID == "" {
		clienteID = u
	}
	if clienteID == "" {
		errorOAuth(w, http.StatusUnauthorized, "invalid_client", "Falta client_id.")
		return
	}
	if v, ok := f["resource"]; ok && (len(v) != 1 || !s.mismoRecurso(v[0])) {
		errorOAuth(w, http.StatusBadRequest, "invalid_target", "resource tiene que ser "+s.url()+".")
		return
	}
	switch f.Get("grant_type") {
	case "authorization_code":
		s.canjearCodigo(w, r, clienteID)
	case "refresh_token":
		s.refrescar(w, r, clienteID)
	case "":
		errorOAuth(w, http.StatusBadRequest, "invalid_request", "Falta grant_type.")
	default:
		errorOAuth(w, http.StatusBadRequest, "unsupported_grant_type", "Sólo authorization_code y refresh_token.")
	}
}

// errGrant: código o refresco que no sirve (invalid_grant) con su descripción.
type errGrant string

func (e errGrant) Error() string { return string(e) }

func desafioDe(verificador string) string {
	h := sha256.Sum256([]byte(verificador))
	return base64.RawURLEncoding.EncodeToString(h[:])
}

// tokens: un par nuevo (acceso con el prefijo de las claves, así /v1/mcp lo resuelve igual).
func tokensNuevos() (acceso, refresco string) {
	return Prefijo + aleatorio(32), PrefijoRefresco + aleatorio(32)
}

func (s *Servicio) respuestaTokens(w http.ResponseWriter, acceso, refresco string, permisos []string) {
	escribirOAuth(w, http.StatusOK, map[string]any{
		"access_token":  acceso,
		"token_type":    "Bearer",
		"expires_in":    int(DuracionAcceso.Seconds()),
		"refresh_token": refresco,
		"scope":         strings.Join(permisos, " "),
	})
}

func intervalo(d time.Duration) string { return strconv.Itoa(int(d.Seconds())) + " seconds" }

func (s *Servicio) canjearCodigo(w http.ResponseWriter, r *http.Request, clienteID string) {
	ctx := r.Context()
	f := r.PostForm
	codigo, verificador := f.Get("code"), f.Get("code_verifier")
	if codigo == "" || verificador == "" {
		errorOAuth(w, http.StatusBadRequest, "invalid_request", "Faltan code y code_verifier.")
		return
	}
	var (
		negocio, clerkID, tokenID, nombreCliente string
		permisos                                 []string
		reusado                                  *string
		reusadoEn                                *string
	)
	err := s.Pool.EnTx(ctx, func(tx pgx.Tx) error {
		var cli, redirect, desafio, recurso string
		var vencido, canjeado bool
		var tokenPrevio *string
		err := tx.QueryRow(ctx, `SELECT a.cliente_id, a.redirect_uri, a.desafio, a.recurso, a.negocio_id::text, a.clerk_id, a.permisos,
				a.codigo_vence < now(), a.canjeada_en IS NOT NULL, a.token_id::text, c.nombre
			FROM auth.oauth_autorizaciones a JOIN auth.oauth_clientes c ON c.id = a.cliente_id
			WHERE a.codigo_hash = $1 FOR UPDATE OF a`, hashDe(codigo)).
			Scan(&cli, &redirect, &desafio, &recurso, &negocio, &clerkID, &permisos, &vencido, &canjeado, &tokenPrevio, &nombreCliente)
		if errors.Is(err, pgx.ErrNoRows) {
			return errGrant("El código no existe.")
		}
		if err != nil {
			return err
		}
		if canjeado {
			// Un código usado dos veces: alguien más lo tiene. Se cae lo que salió de él (RFC 6749 §4.1.2).
			reusado, reusadoEn = tokenPrevio, &negocio
			return errGrant("El código ya se usó.")
		}
		switch {
		case vencido:
			return errGrant("El código venció.")
		case cli != clienteID:
			return errGrant("El código es de otro cliente.")
		case f.Has("redirect_uri") && f.Get("redirect_uri") != redirect:
			return errGrant("redirect_uri no es el del pedido.")
		case !reVerificador.MatchString(verificador) ||
			subtle.ConstantTimeCompare([]byte(desafioDe(verificador)), []byte(desafio)) != 1:
			return errGrant("code_verifier no corresponde al code_challenge.")
		case recurso != "" && !s.mismoRecurso(recurso):
			return errGrant("El código es para otro recurso.")
		}
		if err := tx.QueryRow(ctx, `UPDATE auth.oauth_autorizaciones SET canjeada_en = now(), token_id = gen_random_uuid()
			WHERE codigo_hash = $1 RETURNING token_id::text`, hashDe(codigo)).Scan(&tokenID); err != nil {
			return err
		}
		_, err = tx.Exec(ctx, "UPDATE auth.oauth_clientes SET usado_en = now() WHERE id = $1", cli)
		return err
	})
	if reusado != nil {
		if err := s.revocarConexion(ctx, *reusadoEn, *reusado, "oauth:"+clienteID, "codigo_reusado"); err != nil {
			slog.Error("oauth: no se pudo revocar la conexión de un código reusado", "error", err)
		}
	}
	var eg errGrant
	if errors.As(err, &eg) {
		errorOAuth(w, http.StatusBadRequest, "invalid_grant", string(eg))
		return
	}
	if err != nil {
		slog.Error("oauth: no se pudo canjear el código", "error", err)
		errorOAuth(w, http.StatusInternalServerError, "server_error", "No pudimos canjear el código.")
		return
	}
	acceso, refresco := tokensNuevos()
	err = s.Pool.EnNegocio(ctx, negocio, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, "SELECT 1 FROM negocios WHERE id = app.negocio_actual() FOR UPDATE"); err != nil {
			return err
		}
		var n int
		if err := tx.QueryRow(ctx, `SELECT count(*) FROM mcp_tokens WHERE oauth_cliente IS NOT NULL AND revocado_en IS NULL
			AND refresco_vence > now()`).Scan(&n); err != nil {
			return err
		}
		if n >= MaxConexiones {
			return errGrant("El negocio ya tiene " + strconv.Itoa(MaxConexiones) + " conexiones: que revoque alguna en el panel.")
		}
		if _, err := tx.Exec(ctx, `INSERT INTO mcp_tokens (id, negocio_id, nombre, hash, pista, permisos, creado_por, oauth_cliente,
				acceso_vence, refresco_hash, refresco_vence)
			VALUES ($1, app.negocio_actual(), $2, $3, '', $4, $5, $6, now() + $7::interval, $8, now() + $9::interval)`,
			tokenID, nombreCliente, hashDe(acceso), permisos, clerkID, clienteID, intervalo(DuracionAcceso),
			hashDe(refresco), intervalo(DuracionRefresco)); err != nil {
			return err
		}
		return auditoria.Anotar(ctx, tx, auditoria.Registro{Quien: clerkID, Accion: "mcp.conexion_creada", Entidad: "mcp_token",
			EntidadID: tokenID, Detalle: map[string]any{"nombre": nombreCliente, "cliente": clienteID, "permisos": permisos}})
	})
	if errors.As(err, &eg) {
		errorOAuth(w, http.StatusBadRequest, "invalid_grant", string(eg))
		return
	}
	if err != nil {
		slog.Error("oauth: no se pudo crear la conexión", "error", err)
		errorOAuth(w, http.StatusInternalServerError, "server_error", "No pudimos crear la conexión.")
		return
	}
	s.respuestaTokens(w, acceso, refresco, permisos)
}

// negocioDelRefresco resuelve el token de refresco a su negocio (nil: no existe, vencido o revocado).
func (s *Servicio) negocioDelRefresco(ctx context.Context, refresco string) (*string, error) {
	if !strings.HasPrefix(refresco, PrefijoRefresco) {
		return nil, nil
	}
	var n *string
	err := s.Pool.EnTx(ctx, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, "SELECT app.negocio_del_refresco($1)::text", hex.EncodeToString(hashDe(refresco))).Scan(&n)
	})
	return n, err
}

func (s *Servicio) refrescar(w http.ResponseWriter, r *http.Request, clienteID string) {
	ctx := r.Context()
	refresco := r.PostForm.Get("refresh_token")
	if refresco == "" {
		errorOAuth(w, http.StatusBadRequest, "invalid_request", "Falta refresh_token.")
		return
	}
	negocio, err := s.negocioDelRefresco(ctx, refresco)
	if err != nil {
		slog.Error("oauth: no se pudo resolver el refresco", "error", err)
		errorOAuth(w, http.StatusInternalServerError, "server_error", "No pudimos renovar el token.")
		return
	}
	if negocio == nil {
		errorOAuth(w, http.StatusBadRequest, "invalid_grant", "El refresh_token no sirve (vencido, revocado o ya usado).")
		return
	}
	acceso, nuevo := tokensNuevos()
	var permisos []string
	err = s.Pool.EnNegocio(ctx, *negocio, func(tx pgx.Tx) error {
		var cli string
		var activo bool
		err := tx.QueryRow(ctx, `SELECT k.oauth_cliente, k.permisos, n.activo FROM mcp_tokens k JOIN negocios n ON n.id = k.negocio_id
			WHERE k.refresco_hash = $1 AND k.revocado_en IS NULL AND k.refresco_vence > now() FOR UPDATE OF k`, hashDe(refresco)).
			Scan(&cli, &permisos, &activo)
		if errors.Is(err, pgx.ErrNoRows) {
			return errGrant("El refresh_token ya se usó.")
		}
		if err != nil {
			return err
		}
		if cli != clienteID {
			return errGrant("El refresh_token es de otro cliente.")
		}
		if !activo {
			return errGrant("El negocio está dado de baja.")
		}
		// Rota: el de refresco viejo deja de servir, y el de acceso anterior también.
		_, err = tx.Exec(ctx, `UPDATE mcp_tokens SET hash = $2, acceso_vence = now() + $3::interval, refresco_hash = $4,
			refresco_vence = now() + $5::interval WHERE refresco_hash = $1`,
			hashDe(refresco), hashDe(acceso), intervalo(DuracionAcceso), hashDe(nuevo), intervalo(DuracionRefresco))
		return err
	})
	var eg errGrant
	if errors.As(err, &eg) {
		errorOAuth(w, http.StatusBadRequest, "invalid_grant", string(eg))
		return
	}
	if err != nil {
		slog.Error("oauth: no se pudo renovar", "error", err)
		errorOAuth(w, http.StatusInternalServerError, "server_error", "No pudimos renovar el token.")
		return
	}
	s.respuestaTokens(w, acceso, nuevo, permisos)
}

// ---- Revocación (RFC 7009)

// revocar: siempre 200 (un token desconocido no es un error). Revoca la conexión entera; sólo
// conexiones OAuth (una clave del panel se revoca desde el panel).
func (s *Servicio) revocar(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
	if err := r.ParseForm(); err != nil || r.PostForm.Get("token") == "" {
		errorOAuth(w, http.StatusBadRequest, "invalid_request", "Falta token.")
		return
	}
	tok := r.PostForm.Get("token")
	clienteID := r.PostForm.Get("client_id")
	if u, _, ok := r.BasicAuth(); ok && clienteID == "" {
		clienteID = u
	}
	var negocio *string
	var err error
	col := "hash"
	switch {
	case strings.HasPrefix(tok, PrefijoRefresco):
		negocio, err = s.negocioDelRefresco(ctx, tok)
		col = "refresco_hash"
	case strings.HasPrefix(tok, Prefijo):
		err = s.Pool.EnTx(ctx, func(tx pgx.Tx) error {
			return tx.QueryRow(ctx, "SELECT app.negocio_del_mcp($1)::text", hex.EncodeToString(hashDe(tok))).Scan(&negocio)
		})
	}
	if err == nil && negocio != nil {
		err = s.Pool.EnNegocio(ctx, *negocio, func(tx pgx.Tx) error {
			var id, nombre string
			err := tx.QueryRow(ctx, `UPDATE mcp_tokens SET revocado_en = now() WHERE `+col+` = $1 AND revocado_en IS NULL
				AND oauth_cliente IS NOT NULL AND ($2 = '' OR oauth_cliente = $2) RETURNING id::text, nombre`, hashDe(tok), clienteID).
				Scan(&id, &nombre)
			if errors.Is(err, pgx.ErrNoRows) {
				return nil
			}
			if err != nil {
				return err
			}
			return auditoria.Anotar(ctx, tx, auditoria.Registro{Quien: "oauth:" + clienteID, Accion: "mcp.conexion_revocada",
				Entidad: "mcp_token", EntidadID: id, Detalle: map[string]any{"nombre": nombre, "por": "cliente"}})
		})
	}
	if err != nil {
		slog.Error("oauth: no se pudo revocar", "error", err)
		errorOAuth(w, http.StatusServiceUnavailable, "server_error", "No pudimos revocar el token: probá de nuevo.")
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
}

// revocarConexion: la conexión deja de andar (desde el panel, o por un código reusado).
func (s *Servicio) revocarConexion(ctx context.Context, negocio, id, quien, motivo string) error {
	return s.Pool.EnNegocio(ctx, negocio, func(tx pgx.Tx) error {
		var nombre string
		err := tx.QueryRow(ctx, `UPDATE mcp_tokens SET revocado_en = now() WHERE id::text = $1 AND revocado_en IS NULL
			AND oauth_cliente IS NOT NULL RETURNING nombre`, id).Scan(&nombre)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		return auditoria.Anotar(ctx, tx, auditoria.Registro{Quien: quien, Accion: "mcp.conexion_revocada", Entidad: "mcp_token",
			EntidadID: id, Detalle: map[string]any{"nombre": nombre, "por": motivo}})
	})
}

// ---- La pantalla de consentimiento (/conectar del front)

// Solicitud: lo que muestra /conectar.
type Solicitud struct {
	// Cliente: el nombre con que se registró la app («Claude»).
	Cliente string `json:"cliente"`
	// Destino: el host al que vuelve (para que la persona vea a dónde va el permiso).
	Destino    string           `json:"destino"`
	Negocios   []sesion.Negocio `json:"negocios"`
	Permisos   []Permiso        `json:"permisos"`
	PorDefecto []string         `json:"porDefecto"`
	VenceEn    time.Time        `json:"venceEn"`
}

type pedido struct {
	clienteID, cliente, redirect, state string
	venceEn                             time.Time
}

// leerPedido: el pedido vigente y sin decidir (pgx.ErrNoRows si no; el front dice «venció»).
func leerPedido(ctx context.Context, tx pgx.Tx, id string, bloquear bool) (pedido, error) {
	var p pedido
	q := `SELECT a.cliente_id, c.nombre, a.redirect_uri, a.state, a.vence_en FROM auth.oauth_autorizaciones a
		JOIN auth.oauth_clientes c ON c.id = a.cliente_id WHERE a.hash = $1 AND a.decidida_en IS NULL AND a.vence_en > now()`
	if bloquear {
		q += " FOR UPDATE OF a"
	}
	err := tx.QueryRow(ctx, q, hashDe(id)).Scan(&p.clienteID, &p.cliente, &p.redirect, &p.state, &p.venceEn)
	return p, err
}

func (s *Servicio) negociosDeGestion(ctx context.Context) ([]sesion.Negocio, error) {
	ns, err := s.Sesion.Negocios(ctx, sesion.De(ctx).ClerkID)
	if err != nil {
		return nil, err
	}
	out := []sesion.Negocio{}
	for _, n := range ns {
		if slices.Contains(sesion.Gestion, n.Rol) {
			out = append(out, n)
		}
	}
	return out, nil
}

func hostDe(v string) string {
	if u, err := url.Parse(v); err == nil {
		return u.Host
	}
	return v
}

func (s *Servicio) verSolicitud(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	var p pedido
	if err := s.Pool.EnTx(ctx, func(tx pgx.Tx) error {
		var err error
		p, err = leerPedido(ctx, tx, r.PathValue("id"), false)
		return err
	}); err != nil {
		return err
	}
	ns, err := s.negociosDeGestion(ctx)
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusOK, Solicitud{Cliente: p.cliente, Destino: hostDe(p.redirect), Negocios: ns, Permisos: Permisos(),
		PorDefecto: PermisosPorDefecto, VenceEn: p.venceEn})
	return nil
}

// permitir: la persona elige el negocio (del que es dueña o encargada) y los permisos. Sale el
// código y la URL a la que el front manda el navegador.
func (s *Servicio) permitir(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	var in struct {
		NegocioID string   `json:"negocioId"`
		Permisos  []string `json:"permisos"`
	}
	if err := web.Leer(w, r, &in); err != nil {
		return err
	}
	ps, err := validarPermisos(in.Permisos)
	if err != nil {
		return err
	}
	ns, err := s.negociosDeGestion(ctx)
	if err != nil {
		return err
	}
	if !slices.ContainsFunc(ns, func(n sesion.Negocio) bool { return n.ID == in.NegocioID }) {
		return problema.SinPermiso("Sólo el dueño o un encargado del negocio puede conectar un asistente.")
	}
	var n int
	if err := s.Pool.EnNegocio(ctx, in.NegocioID, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `SELECT count(*) FROM mcp_tokens WHERE oauth_cliente IS NOT NULL AND revocado_en IS NULL
			AND refresco_vence > now()`).Scan(&n)
	}); err != nil {
		return err
	}
	if n >= MaxConexiones {
		return problema.Conflicto("tope", "Ese negocio ya tiene "+strconv.Itoa(MaxConexiones)+
			" asistentes conectados: desconectá alguno en Integraciones e IA.")
	}
	codigo := aleatorio(32)
	var p pedido
	err = s.Pool.EnTx(ctx, func(tx pgx.Tx) error {
		var err error
		if p, err = leerPedido(ctx, tx, r.PathValue("id"), true); err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `UPDATE auth.oauth_autorizaciones SET decidida_en = now(), clerk_id = $2, negocio_id = $3, permisos = $4,
			codigo_hash = $5, codigo_vence = now() + $6::interval WHERE hash = $1`,
			hashDe(r.PathValue("id")), sesion.De(ctx).ClerkID, in.NegocioID, ps, hashDe(codigo), intervalo(DuracionCodigo))
		return err
	})
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusOK, map[string]string{"redirect": conParametros(p.redirect,
		map[string]string{"code": codigo, "state": p.state, "iss": s.emisor()})})
	return nil
}

// cancelar: vuelve al cliente con access_denied.
func (s *Servicio) cancelar(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	var p pedido
	err := s.Pool.EnTx(ctx, func(tx pgx.Tx) error {
		var err error
		if p, err = leerPedido(ctx, tx, r.PathValue("id"), true); err != nil {
			return err
		}
		_, err = tx.Exec(ctx, "UPDATE auth.oauth_autorizaciones SET decidida_en = now(), clerk_id = $2 WHERE hash = $1",
			hashDe(r.PathValue("id")), sesion.De(ctx).ClerkID)
		return err
	})
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusOK, map[string]string{"redirect": conParametros(p.redirect, map[string]string{"error": "access_denied",
		"error_description": "La persona no dio permiso.", "state": p.state, "iss": s.emisor()})})
	return nil
}
