// Package mcp: el servidor MCP del negocio. El dueño crea una clave en el panel y la carga en su
// asistente de IA (Claude, ChatGPT u otro cliente MCP); el asistente consulta y opera la agenda en
// lenguaje natural con las herramientas que esa clave tenga permitidas. El modelo lo pone el
// cliente: nosotros sólo servimos las herramientas.
//
// Transporte Streamable HTTP (spec MCP 2025-06-18): JSON-RPC 2.0 por POST a /v1/mcp, respuestas en
// JSON simple, sin sesión ni SSE. Cada tools/call queda en el registro; lo que cambia algo va además
// a la auditoría y a la campanita.
package mcp

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/turnos"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/db"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/problema"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/sesion"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/web"
)

// URLPorDefecto del servidor, la que se carga en el asistente.
const URLPorDefecto = "https://api.tengolibre.com.ar/v1/mcp"

// Versiones del protocolo que entendemos; la primera es la que proponemos.
var versiones = []string{"2025-06-18", "2025-11-25", "2025-03-26"}

// Servicio MCP. Ahora es el reloj (tests); nil: time.Now.
type Servicio struct {
	Pool   *db.Pool
	Sesion *sesion.Servicio
	Turnos *turnos.Servicio
	// URL pública del servidor (la que muestra el panel). Vacía: URLPorDefecto.
	URL string
	// Origenes: si el pedido trae Origin (un navegador), tiene que ser uno de estos (la spec pide
	// validarlo contra el DNS rebinding). Los clientes MCP de servidor no mandan Origin.
	Origenes []string
	Ahora    func() time.Time
}

func (s *Servicio) ahora() time.Time {
	if s.Ahora != nil {
		return s.Ahora()
	}
	return time.Now()
}

func (s *Servicio) url() string {
	if s.URL != "" {
		return s.URL
	}
	return URLPorDefecto
}

// Rutas del dominio: el servidor MCP y su gestión en el panel (sólo dueño y encargado).
func (s *Servicio) Rutas(mux *http.ServeMux) {
	mux.HandleFunc("POST /v1/mcp", s.servir)
	mux.HandleFunc("GET /v1/mcp", s.sinFlujo)
	mux.HandleFunc("DELETE /v1/mcp", s.sinFlujo)
	g := func(h web.Manejador) http.HandlerFunc { return s.Sesion.Panel(sesion.Gestion, web.Envolver(h)) }
	mux.HandleFunc("GET /v1/panel/mcp", g(s.leer))
	mux.HandleFunc("POST /v1/panel/mcp/tokens", g(s.crearToken))
	mux.HandleFunc("PATCH /v1/panel/mcp/tokens/{id}", g(s.editarToken))
	mux.HandleFunc("DELETE /v1/panel/mcp/tokens/{id}", g(s.revocarToken))
	mux.HandleFunc("GET /v1/panel/mcp/registro", g(s.registro))
}

// sinFlujo: no ofrecemos el stream SSE del GET ni sesiones para cerrar (la spec permite el 405).
func (s *Servicio) sinFlujo(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Allow", "POST")
	problema.Escribir(w, &problema.Problema{Estado: http.StatusMethodNotAllowed, Codigo: "metodo",
		Titulo: "Método no permitido", Detalle: "El servidor MCP de TengoLibre responde sólo por POST."})
}

// ---- JSON-RPC

type mensaje struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Metodo  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type respuesta struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  any             `json:"result,omitempty"`
	Error   *errorRPC       `json:"error,omitempty"`
}

type errorRPC struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// Códigos de JSON-RPC.
const (
	errParseo    = -32700
	errPedido    = -32600
	errMetodo    = -32601
	errParametro = -32602
	errInterno   = -32603
)

var nulo = json.RawMessage("null")

// conexion: la clave que llama, ya resuelta, y su negocio.
type conexion struct {
	negocio  string
	tokenID  string
	nombre   string
	permisos []string
	zona     *time.Location
}

func (c *conexion) puede(p string) bool { return slices.Contains(c.permisos, p) }

// claveDe: el Bearer del encabezado.
func claveDe(r *http.Request) string {
	if t, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer "); ok {
		return strings.TrimSpace(t)
	}
	return ""
}

func hashDe(clave string) []byte {
	h := sha256.Sum256([]byte(clave))
	return h[:]
}

func sinClave(w http.ResponseWriter) {
	w.Header().Set("WWW-Authenticate", `Bearer realm="TengoLibre", error="invalid_token"`)
	problema.Escribir(w, &problema.Problema{Estado: http.StatusUnauthorized, Codigo: "sin_clave", Titulo: "Falta la clave",
		Detalle: "Mandá la clave del asistente (la del panel, en Integraciones e IA) como Authorization: Bearer <clave>."})
}

// conectar resuelve la clave al negocio, carga sus permisos y anota que se usó.
func (s *Servicio) conectar(ctx context.Context, clave string) (*conexion, error) {
	if !strings.HasPrefix(clave, Prefijo) {
		return nil, nil
	}
	h := hashDe(clave)
	var negocio *string
	if err := s.Pool.EnTx(ctx, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, "SELECT app.negocio_del_mcp($1)::text", hex.EncodeToString(h)).Scan(&negocio)
	}); err != nil {
		return nil, err
	}
	if negocio == nil {
		return nil, nil
	}
	c := &conexion{negocio: *negocio}
	err := s.Pool.EnNegocio(ctx, *negocio, func(tx pgx.Tx) error {
		var activo bool
		var zona string
		err := tx.QueryRow(ctx, `SELECT k.id::text, k.nombre, k.permisos, n.activo, n.zona_horaria FROM mcp_tokens k
			JOIN negocios n ON n.id = k.negocio_id WHERE k.hash = $1 AND k.revocado_en IS NULL`, h).
			Scan(&c.tokenID, &c.nombre, &c.permisos, &activo, &zona)
		if err != nil {
			return err
		}
		if !activo {
			c = nil
			return nil
		}
		if c.zona, err = time.LoadLocation(zona); err != nil {
			c.zona = time.FixedZone("UTC-3", -3*3600)
		}
		_, err = tx.Exec(ctx, "UPDATE mcp_tokens SET usado_en = now() WHERE id = $1 AND (usado_en IS NULL OR usado_en < now() - interval '1 minute')", c.tokenID)
		return err
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return c, err
}

func (s *Servicio) origenPermitido(r *http.Request) bool {
	o := r.Header.Get("Origin")
	return o == "" || slices.Contains(s.Origenes, o)
}

// servir atiende un POST del cliente MCP: un mensaje JSON-RPC (o un lote, como en 2025-03-26).
func (s *Servicio) servir(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if !s.origenPermitido(r) {
		problema.Escribir(w, problema.SinPermiso("Ese origen no puede hablar con el servidor MCP."))
		return
	}
	if v := r.Header.Get("MCP-Protocol-Version"); v != "" && !slices.Contains(versiones, v) {
		problema.Escribir(w, problema.Invalido("No conocemos la versión "+v+" del protocolo MCP."))
		return
	}
	c, err := s.conectar(ctx, claveDe(r))
	if err != nil {
		slog.Error("mcp: no se pudo resolver la clave", "error", err)
		problema.Escribir(w, problema.Interno())
		return
	}
	if c == nil {
		sinClave(w)
		return
	}
	cuerpo, err := io.ReadAll(http.MaxBytesReader(w, r.Body, web.MaxCuerpo))
	if err != nil {
		escribirRPC(w, http.StatusRequestEntityTooLarge, respuesta{JSONRPC: "2.0", ID: nulo, Error: &errorRPC{errPedido, "El pedido es muy grande."}})
		return
	}
	cuerpo = bytes.TrimSpace(cuerpo)
	var lote []mensaje
	esLote := len(cuerpo) > 0 && cuerpo[0] == '['
	if esLote {
		err = json.Unmarshal(cuerpo, &lote)
	} else {
		var m mensaje
		err = json.Unmarshal(cuerpo, &m)
		lote = []mensaje{m}
	}
	if err != nil || len(lote) == 0 {
		escribirRPC(w, http.StatusBadRequest, respuesta{JSONRPC: "2.0", ID: nulo, Error: &errorRPC{errParseo, "El cuerpo no es JSON-RPC válido."}})
		return
	}
	var out []respuesta
	for _, m := range lote {
		if res := s.atender(ctx, c, m); res != nil {
			out = append(out, *res)
		}
	}
	// Sólo notificaciones o respuestas: 202 sin cuerpo.
	if len(out) == 0 {
		w.WriteHeader(http.StatusAccepted)
		return
	}
	if esLote {
		escribirRPC(w, http.StatusOK, out)
		return
	}
	escribirRPC(w, http.StatusOK, out[0])
}

func escribirRPC(w http.ResponseWriter, estado int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(estado)
	_ = json.NewEncoder(w).Encode(v)
}

// atender un mensaje. nil: no lleva respuesta (notificación o respuesta del cliente).
func (s *Servicio) atender(ctx context.Context, c *conexion, m mensaje) *respuesta {
	if m.Metodo == "" {
		return nil
	}
	if len(m.ID) == 0 || string(m.ID) == "null" {
		// Notificaciones (notifications/initialized, cancelled…): no hay nada que contestar.
		return nil
	}
	res := &respuesta{JSONRPC: "2.0", ID: m.ID}
	if m.JSONRPC != "2.0" {
		res.Error = &errorRPC{errPedido, "Falta jsonrpc: \"2.0\"."}
		return res
	}
	var err *errorRPC
	switch m.Metodo {
	case "initialize":
		res.Result, err = s.inicializar(ctx, c, m.Params)
	case "ping":
		res.Result = map[string]any{}
	case "tools/list":
		res.Result = map[string]any{"tools": listar(c)}
	case "tools/call":
		res.Result, err = s.llamar(ctx, c, m.Params)
	default:
		err = &errorRPC{errMetodo, "Método desconocido: " + m.Metodo}
	}
	if err != nil {
		res.Result, res.Error = nil, err
	}
	return res
}

func (s *Servicio) inicializar(ctx context.Context, c *conexion, params json.RawMessage) (any, *errorRPC) {
	var in struct {
		ProtocolVersion string `json:"protocolVersion"`
		ClientInfo      struct {
			Name    string `json:"name"`
			Title   string `json:"title"`
			Version string `json:"version"`
		} `json:"clientInfo"`
	}
	if len(params) > 0 {
		if err := json.Unmarshal(params, &in); err != nil {
			return nil, &errorRPC{errParametro, "Los parámetros de initialize no son válidos."}
		}
	}
	version := versiones[0]
	if slices.Contains(versiones, in.ProtocolVersion) {
		version = in.ProtocolVersion
	}
	cliente := strings.TrimSpace(strings.TrimSpace(in.ClientInfo.Name) + " " + strings.TrimSpace(in.ClientInfo.Version))
	if cliente != "" {
		if len(cliente) > 80 {
			cliente = cliente[:80]
		}
		if err := s.Pool.EnNegocio(ctx, c.negocio, func(tx pgx.Tx) error {
			_, err := tx.Exec(ctx, "UPDATE mcp_tokens SET cliente = $2 WHERE id = $1", c.tokenID, cliente)
			return err
		}); err != nil {
			slog.Warn("mcp: no se pudo anotar el cliente", "error", err)
		}
	}
	var nombre string
	_ = s.Pool.EnNegocio(ctx, c.negocio, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, "SELECT nombre FROM negocios WHERE id = app.negocio_actual()").Scan(&nombre)
	})
	return map[string]any{
		"protocolVersion": version,
		"capabilities":    map[string]any{"tools": map[string]any{"listChanged": false}},
		"serverInfo":      map[string]any{"name": "tengolibre", "title": "Tengo Libre · " + nombre, "version": "1.0.0"},
		"instructions": "Sos el asistente de " + nombre + " en Tengo Libre, el sistema de turnos del negocio. " +
			"Las fechas van como AAAA-MM-DD y las horas como HH:MM, en la hora del negocio. Antes de crear, mover, " +
			"confirmar o cancelar un turno, mostrale a la persona lo que vas a hacer y esperá que te diga que sí. " +
			"Para crear un turno, buscá antes los horarios libres; los ids salen de ver_servicios_y_profesionales, " +
			"ver_agenda y buscar_clientes.",
	}, nil
}
