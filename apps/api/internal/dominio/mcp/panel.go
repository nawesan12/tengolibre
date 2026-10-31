package mcp

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"
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

// Prefijo de las claves del asistente (distinto del de las sesiones, tl_).
const Prefijo = "tlmcp_"

// MaxTokens por negocio (sin contar las revocadas).
const MaxTokens = 10

// Token: una clave del asistente como la ve el panel (la clave en sí nunca, salvo al crearla).
type Token struct {
	ID       string     `json:"id"`
	Nombre   string     `json:"nombre"`
	Pista    string     `json:"pista"`
	Permisos []string   `json:"permisos"`
	Cliente  string     `json:"cliente"`
	CreadoEn time.Time  `json:"creadoEn"`
	UsadoEn  *time.Time `json:"usadoEn"`
}

const columnasToken = "id::text, nombre, pista, permisos, cliente, creado_en, usado_en"

func escanearToken(f pgx.Row, t *Token) error {
	return f.Scan(&t.ID, &t.Nombre, &t.Pista, &t.Permisos, &t.Cliente, &t.CreadoEn, &t.UsadoEn)
}

// Conexion: un asistente conectado por OAuth (claude.ai, ChatGPT), como la ve el panel.
type Conexion struct {
	ID string `json:"id"`
	// Nombre: el de la app que se registró («Claude»); el registro y la campanita dicen ese.
	Nombre   string     `json:"nombre"`
	Permisos []string   `json:"permisos"`
	Cliente  string     `json:"cliente"`
	CreadoEn time.Time  `json:"creadoEn"`
	UsadoEn  *time.Time `json:"usadoEn"`
	// AutorizadoPor: el Clerk ID de quien dio el permiso.
	AutorizadoPor string `json:"autorizadoPor"`
}

const columnasConexion = "id::text, nombre, permisos, cliente, creado_en, usado_en, creado_por"

func escanearConexion(f pgx.Row, c *Conexion) error {
	return f.Scan(&c.ID, &c.Nombre, &c.Permisos, &c.Cliente, &c.CreadoEn, &c.UsadoEn, &c.AutorizadoPor)
}

// Una conexión vigente: sin revocar y con el refresco al día.
const conexionVigente = "oauth_cliente IS NOT NULL AND revocado_en IS NULL AND refresco_vence > now()"

func (s *Servicio) enNegocio(r *http.Request, fn func(context.Context, pgx.Tx) error) error {
	ctx := r.Context()
	return s.Pool.EnNegocio(ctx, sesion.De(ctx).NegocioID, func(tx pgx.Tx) error { return fn(ctx, tx) })
}

// leer: la URL del servidor, las claves y las conexiones OAuth vigentes y los permisos que existen.
func (s *Servicio) leer(w http.ResponseWriter, r *http.Request) error {
	out := struct {
		URL        string     `json:"url"`
		Tokens     []Token    `json:"tokens"`
		Conexiones []Conexion `json:"conexiones"`
		Permisos   []Permiso  `json:"permisos"`
	}{URL: s.url(), Tokens: []Token{}, Conexiones: []Conexion{}, Permisos: Permisos()}
	err := s.enNegocio(r, func(ctx context.Context, tx pgx.Tx) error {
		filas, err := tx.Query(ctx, "SELECT "+columnasToken+" FROM mcp_tokens WHERE revocado_en IS NULL AND oauth_cliente IS NULL ORDER BY creado_en, id")
		if err != nil {
			return err
		}
		out.Tokens, err = pgx.CollectRows(filas, func(f pgx.CollectableRow) (Token, error) {
			var t Token
			return t, escanearToken(f, &t)
		})
		if err != nil {
			return err
		}
		filas, err = tx.Query(ctx, "SELECT "+columnasConexion+" FROM mcp_tokens WHERE "+conexionVigente+" ORDER BY creado_en, id")
		if err != nil {
			return err
		}
		out.Conexiones, err = pgx.CollectRows(filas, func(f pgx.CollectableRow) (Conexion, error) {
			var c Conexion
			return c, escanearConexion(f, &c)
		})
		return err
	})
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusOK, out)
	return nil
}

func validarNombre(n string) (string, error) {
	n = strings.TrimSpace(n)
	if c := utf8.RuneCountInString(n); c < 1 || c > 60 {
		return n, problema.Invalido("El nombre va de 1 a 60 caracteres.", problema.Campo{Ruta: "nombre", Mensaje: "inválido"})
	}
	return n, nil
}

func validarPermisos(ps []string) ([]string, error) {
	out := []string{}
	for _, p := range ps {
		if !permisoValido(p) {
			return nil, problema.Invalido("No existe el permiso «"+p+"».", problema.Campo{Ruta: "permisos", Mensaje: "inválido"})
		}
		if !slices.Contains(out, p) {
			out = append(out, p)
		}
	}
	return out, nil
}

// crearToken: la clave se devuelve una sola vez (en «clave»); después sólo queda la pista.
func (s *Servicio) crearToken(w http.ResponseWriter, r *http.Request) error {
	var in struct {
		Nombre   string    `json:"nombre"`
		Permisos *[]string `json:"permisos"`
	}
	if err := web.Leer(w, r, &in); err != nil {
		return err
	}
	nombre, err := validarNombre(in.Nombre)
	if err != nil {
		return err
	}
	ps := PermisosPorDefecto
	if in.Permisos != nil {
		if ps, err = validarPermisos(*in.Permisos); err != nil {
			return err
		}
	}
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	clave := Prefijo + base64.RawURLEncoding.EncodeToString(b)
	var t Token
	err = s.enNegocio(r, func(ctx context.Context, tx pgx.Tx) error {
		// El bloqueo del negocio pone en fila dos altas a la vez, así el tope se respeta.
		if _, err := tx.Exec(ctx, "SELECT 1 FROM negocios WHERE id = app.negocio_actual() FOR UPDATE"); err != nil {
			return err
		}
		var n int
		if err := tx.QueryRow(ctx, "SELECT count(*) FROM mcp_tokens WHERE revocado_en IS NULL AND oauth_cliente IS NULL").Scan(&n); err != nil {
			return err
		}
		if n >= MaxTokens {
			return problema.Conflicto("tope", "Ya hay "+strconv.Itoa(MaxTokens)+" claves: revocá alguna antes de crear otra.")
		}
		if err := escanearToken(tx.QueryRow(ctx, `INSERT INTO mcp_tokens (negocio_id, nombre, hash, pista, permisos, creado_por)
			VALUES (app.negocio_actual(), $1, $2, $3, $4, $5) RETURNING `+columnasToken,
			nombre, hashDe(clave), clave[len(clave)-4:], ps, sesion.De(ctx).ClerkID), &t); err != nil {
			return err
		}
		return auditoria.Anotar(ctx, tx, auditoria.Registro{Accion: "mcp.clave_creada", Entidad: "mcp_token", EntidadID: t.ID,
			Detalle: map[string]any{"nombre": t.Nombre, "permisos": t.Permisos}})
	})
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusCreated, map[string]any{"token": t, "clave": clave, "url": s.url()})
	return nil
}

// editarToken: renombrar o cambiar los permisos (la lista entera de los encendidos).
func (s *Servicio) editarToken(w http.ResponseWriter, r *http.Request) error {
	var in struct {
		Nombre   *string   `json:"nombre"`
		Permisos *[]string `json:"permisos"`
	}
	if err := web.Leer(w, r, &in); err != nil {
		return err
	}
	var nombre *string
	if in.Nombre != nil {
		n, err := validarNombre(*in.Nombre)
		if err != nil {
			return err
		}
		nombre = &n
	}
	var ps []string
	if in.Permisos != nil {
		var err error
		if ps, err = validarPermisos(*in.Permisos); err != nil {
			return err
		}
	}
	var t Token
	err := s.enNegocio(r, func(ctx context.Context, tx pgx.Tx) error {
		err := escanearToken(tx.QueryRow(ctx, `UPDATE mcp_tokens SET nombre = coalesce($2, nombre), permisos = coalesce($3, permisos)
			WHERE id::text = $1 AND revocado_en IS NULL AND oauth_cliente IS NULL RETURNING `+columnasToken, r.PathValue("id"), nombre, ps), &t)
		if err != nil {
			return err
		}
		return auditoria.Anotar(ctx, tx, auditoria.Registro{Accion: "mcp.clave_editada", Entidad: "mcp_token", EntidadID: t.ID,
			Detalle: map[string]any{"nombre": t.Nombre, "permisos": t.Permisos}})
	})
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusOK, t)
	return nil
}

// revocarToken: la clave deja de andar al instante. Su registro queda.
func (s *Servicio) revocarToken(w http.ResponseWriter, r *http.Request) error {
	err := s.enNegocio(r, func(ctx context.Context, tx pgx.Tx) error {
		var nombre string
		err := tx.QueryRow(ctx, "UPDATE mcp_tokens SET revocado_en = now() WHERE id::text = $1 AND revocado_en IS NULL AND oauth_cliente IS NULL RETURNING nombre",
			r.PathValue("id")).Scan(&nombre)
		if err != nil {
			return err
		}
		return auditoria.Anotar(ctx, tx, auditoria.Registro{Accion: "mcp.clave_revocada", Entidad: "mcp_token", EntidadID: r.PathValue("id"),
			Detalle: map[string]any{"nombre": nombre}})
	})
	if err != nil {
		return err
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}

// editarConexion: cambiar los permisos de un asistente conectado (la lista entera de los encendidos).
// Valen al instante, también con el token de acceso que ya tiene.
func (s *Servicio) editarConexion(w http.ResponseWriter, r *http.Request) error {
	var in struct {
		Permisos []string `json:"permisos"`
	}
	if err := web.Leer(w, r, &in); err != nil {
		return err
	}
	if in.Permisos == nil {
		return problema.Invalido("Mandá permisos (la lista de los encendidos).", problema.Campo{Ruta: "permisos", Mensaje: "falta"})
	}
	ps, err := validarPermisos(in.Permisos)
	if err != nil {
		return err
	}
	var c Conexion
	err = s.enNegocio(r, func(ctx context.Context, tx pgx.Tx) error {
		if err := escanearConexion(tx.QueryRow(ctx, "UPDATE mcp_tokens SET permisos = $2 WHERE id::text = $1 AND "+conexionVigente+
			" RETURNING "+columnasConexion, r.PathValue("id"), ps), &c); err != nil {
			return err
		}
		return auditoria.Anotar(ctx, tx, auditoria.Registro{Accion: "mcp.conexion_editada", Entidad: "mcp_token", EntidadID: c.ID,
			Detalle: map[string]any{"nombre": c.Nombre, "permisos": c.Permisos}})
	})
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusOK, c)
	return nil
}

// desconectar: la conexión OAuth deja de andar al instante (acceso y refresco). Su registro queda.
func (s *Servicio) desconectar(w http.ResponseWriter, r *http.Request) error {
	err := s.enNegocio(r, func(ctx context.Context, tx pgx.Tx) error {
		var nombre string
		err := tx.QueryRow(ctx, "UPDATE mcp_tokens SET revocado_en = now() WHERE id::text = $1 AND "+conexionVigente+" RETURNING nombre",
			r.PathValue("id")).Scan(&nombre)
		if err != nil {
			return err
		}
		return auditoria.Anotar(ctx, tx, auditoria.Registro{Accion: "mcp.conexion_revocada", Entidad: "mcp_token", EntidadID: r.PathValue("id"),
			Detalle: map[string]any{"nombre": nombre, "por": "panel"}})
	})
	if err != nil {
		return err
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}

// Accion del registro, como la ve el panel («Lo último que hizo la IA»).
type Accion struct {
	ID      string `json:"id"`
	TokenID string `json:"tokenId"`
	Token   string `json:"token"`
	Cliente string `json:"cliente"`
	// Tipo: «clave» (una clave del panel) u «oauth» (un asistente conectado, p. ej. Claude).
	Tipo        string          `json:"tipo"`
	Herramienta string          `json:"herramienta"`
	Titulo      string          `json:"titulo"`
	Argumentos  json.RawMessage `json:"argumentos"`
	Ok          bool            `json:"ok"`
	Escribe     bool            `json:"escribe"`
	Resumen     string          `json:"resumen"`
	Error       string          `json:"error"`
	CreadoEn    time.Time       `json:"creadoEn"`
}

// registro: lo más nuevo primero. Filtros: token, soloCambios=1, antes (cursor: el creadoEn del
// último de la página anterior), limite (hasta 200; por defecto 50).
func (s *Servicio) registro(w http.ResponseWriter, r *http.Request) error {
	q := r.URL.Query()
	limite := 50
	if v, err := strconv.Atoi(q.Get("limite")); err == nil && v > 0 {
		limite = min(v, 200)
	}
	antes := time.Now().Add(time.Hour)
	if v := q.Get("antes"); v != "" {
		t, err := time.Parse(time.RFC3339Nano, v)
		if err != nil {
			return problema.Invalido("antes va como fecha RFC 3339.")
		}
		antes = t
	}
	out := []Accion{}
	err := s.enNegocio(r, func(ctx context.Context, tx pgx.Tx) error {
		filas, err := tx.Query(ctx, `SELECT g.id::text, g.token_id::text, k.nombre, k.cliente,
				CASE WHEN k.oauth_cliente IS NULL THEN 'clave' ELSE 'oauth' END, g.herramienta,
				coalesce(g.argumentos, 'null'::jsonb), g.ok, g.escribe, g.resumen, g.error, g.creado_en
			FROM mcp_registro g JOIN mcp_tokens k ON k.id = g.token_id
			WHERE g.creado_en < $1 AND ($2 = '' OR g.token_id::text = $2) AND (NOT $3 OR g.escribe)
			ORDER BY g.creado_en DESC, g.id DESC LIMIT $4`, antes, q.Get("token"), q.Get("soloCambios") == "1", limite)
		if err != nil {
			return err
		}
		out, err = pgx.CollectRows(filas, func(f pgx.CollectableRow) (Accion, error) {
			var a Accion
			var args []byte
			err := f.Scan(&a.ID, &a.TokenID, &a.Token, &a.Cliente, &a.Tipo, &a.Herramienta, &args, &a.Ok, &a.Escribe, &a.Resumen, &a.Error, &a.CreadoEn)
			a.Argumentos = args
			if h := buscarHerramienta(a.Herramienta); h != nil {
				a.Titulo = h.titulo
			}
			return a, err
		})
		return err
	})
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusOK, map[string]any{"acciones": out})
	return nil
}
