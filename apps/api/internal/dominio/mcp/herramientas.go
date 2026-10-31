package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/avisos"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/auditoria"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/problema"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/sesion"
)

// Permisos de una clave: los interruptores de «Qué puede hacer» en el panel.
const (
	VerAgenda      = "ver_agenda"
	MoverTurnos    = "mover_turnos"
	CancelarTurnos = "cancelar_turnos"
	VerClientes    = "ver_clientes"
	VerNumeros     = "ver_numeros"
)

// Permiso como lo muestra el panel, con las herramientas que habilita.
type Permiso struct {
	Clave        string   `json:"clave"`
	Nombre       string   `json:"nombre"`
	Descripcion  string   `json:"descripcion"`
	Herramientas []string `json:"herramientas"`
}

var permisos = []Permiso{
	{Clave: VerAgenda, Nombre: "Ver la agenda", Descripcion: "Turnos, horarios libres, lista de espera, servicios y profesionales"},
	{Clave: MoverTurnos, Nombre: "Crear y mover turnos", Descripcion: "Cargar turnos, confirmarlos y cambiarlos de horario"},
	{Clave: CancelarTurnos, Nombre: "Cancelar turnos", Descripcion: "Cancelar un turno (al cliente le llega el aviso)"},
	{Clave: VerClientes, Nombre: "Ver clientes", Descripcion: "Nombre, contacto e historial"},
	{Clave: VerNumeros, Nombre: "Ver números", Descripcion: "Turnos, facturación y ausencias del período"},
}

// PermisosPorDefecto de una clave nueva (los encendidos del diseño).
var PermisosPorDefecto = []string{VerAgenda, MoverTurnos, VerClientes}

func permisoValido(p string) bool {
	for _, x := range permisos {
		if x.Clave == p {
			return true
		}
	}
	return false
}

// Permisos con sus herramientas (para el panel y la documentación).
func Permisos() []Permiso {
	out := make([]Permiso, len(permisos))
	for i, p := range permisos {
		p.Herramientas = []string{}
		for _, h := range herramientas {
			if h.permiso == p.Clave {
				p.Herramientas = append(p.Herramientas, h.nombre)
			}
		}
		out[i] = p
	}
	return out
}

// herramienta MCP. correr devuelve el resultado (un objeto) y una frase para el registro.
type herramienta struct {
	nombre, titulo, descripcion, permiso string
	esquema                              map[string]any
	// escribe: cambia algo (pasa por el sólo lectura y va a la auditoría y la campanita).
	escribe, destructiva bool
	correr               func(ctx context.Context, s *Servicio, c *conexion, args json.RawMessage) (any, string, error)
}

var herramientas []herramienta

func init() {
	herramientas = []herramienta{
		{nombre: "ver_agenda", titulo: "Ver la agenda", permiso: VerAgenda, correr: verAgenda,
			descripcion: "Muestra los turnos del negocio de un día o de un rango de días (hasta 31), con cliente, servicio, " +
				"profesional y estado. Sin fechas, los de hoy.",
			esquema: objeto(nil, map[string]any{
				"desde":             fecha("Primer día, AAAA-MM-DD. Por defecto, hoy."),
				"hasta":             fecha("Último día (inclusive), AAAA-MM-DD. Por defecto, el mismo que desde."),
				"profesionalId":     texto("Sólo los de este profesional (id o nombre)."),
				"incluirCancelados": map[string]any{"type": "boolean", "description": "Mostrar también los cancelados. Por defecto, no."},
			})},
		{nombre: "buscar_horarios_libres", titulo: "Buscar horarios libres", permiso: VerAgenda, correr: buscarLibres,
			descripcion: "Busca los horarios libres para un servicio, con cualquier profesional o con uno en particular, " +
				"en un día o en los días que siguen (hasta 7). Respeta los horarios, bloqueos y reglas de reserva del negocio.",
			esquema: objeto([]string{"servicioId", "fecha"}, map[string]any{
				"servicioId":    texto("El servicio (id o nombre exacto)."),
				"fecha":         fecha("Día, AAAA-MM-DD."),
				"dias":          map[string]any{"type": "integer", "minimum": 1, "maximum": 7, "description": "Cuántos días mirar desde la fecha. Por defecto, 1."},
				"profesionalId": texto("Sólo con este profesional (id o nombre)."),
			})},
		{nombre: "ver_lista_de_espera", titulo: "Ver la lista de espera", permiso: VerAgenda, correr: verEspera,
			descripcion: "Muestra quiénes están esperando que se libere un horario: para qué servicio, con quién y entre qué fechas.",
			esquema:     objeto(nil, map[string]any{})},
		{nombre: "ver_servicios_y_profesionales", titulo: "Ver servicios y profesionales", permiso: VerAgenda, correr: verCatalogo,
			descripcion: "Lista los servicios (duración y precio) y los profesionales del negocio (qué servicios hacen y en qué " +
				"horario atienden), con los ids que piden las demás herramientas.",
			esquema: objeto(nil, map[string]any{})},
		{nombre: "crear_turno", titulo: "Crear un turno", permiso: MoverTurnos, escribe: true, correr: crearTurno,
			descripcion: "Carga un turno nuevo, ya confirmado, como desde el panel. El horario tiene que estar libre (buscalo " +
				"antes con buscar_horarios_libres). Va a nombre de un cliente existente (clienteId) o de uno nuevo (cliente). " +
				"Al cliente le llega el aviso. Confirmá con la persona antes de usarla.",
			esquema: objeto([]string{"servicioId", "fecha", "hora"}, map[string]any{
				"servicioId":    texto("El servicio (id o nombre exacto)."),
				"fecha":         fecha("Día, AAAA-MM-DD."),
				"hora":          hora("Hora de inicio, HH:MM, en la hora del negocio."),
				"profesionalId": texto("Con quién (id o nombre). Sin esto, el primero libre."),
				"clienteId":     texto("Un cliente que ya existe (sale de buscar_clientes)."),
				"cliente": objeto([]string{"nombre"}, map[string]any{
					"nombre":   texto("Nombre y apellido."),
					"telefono": texto("Teléfono (con esto se lo reconoce la próxima vez)."),
					"email":    texto("Email."),
				}),
				"nota": texto("Nota para el turno (hasta 500 caracteres)."),
			})},
		{nombre: "confirmar_turno", titulo: "Confirmar un turno", permiso: MoverTurnos, escribe: true, correr: confirmarTurno,
			descripcion: "Pasa un turno pendiente a confirmado. Confirmá con la persona antes de usarla.",
			esquema:     objeto([]string{"turnoId"}, map[string]any{"turnoId": texto("El id del turno (sale de ver_agenda).")})},
		{nombre: "reprogramar_turno", titulo: "Reprogramar un turno", permiso: MoverTurnos, escribe: true, correr: reprogramarTurno,
			descripcion: "Mueve un turno pendiente o confirmado a otro día y hora (y, si se pide, a otro profesional). El horario " +
				"nuevo tiene que estar libre. Al cliente le llega el aviso. Confirmá con la persona antes de usarla.",
			esquema: objeto([]string{"turnoId", "fecha", "hora"}, map[string]any{
				"turnoId":       texto("El id del turno (sale de ver_agenda)."),
				"fecha":         fecha("Día nuevo, AAAA-MM-DD."),
				"hora":          hora("Hora nueva, HH:MM, en la hora del negocio."),
				"profesionalId": texto("Otro profesional (id o nombre). Sin esto, el mismo."),
			})},
		{nombre: "cancelar_turno", titulo: "Cancelar un turno", permiso: CancelarTurnos, escribe: true, destructiva: true, correr: cancelarTurno,
			descripcion: "Cancela un turno: el horario queda libre, se avisa a la lista de espera y al cliente le llega el aviso. " +
				"No se puede deshacer. Confirmá con la persona antes de usarla.",
			esquema: objeto([]string{"turnoId"}, map[string]any{"turnoId": texto("El id del turno (sale de ver_agenda).")})},
		{nombre: "buscar_clientes", titulo: "Buscar clientes", permiso: VerClientes, correr: buscarClientes,
			descripcion: "Busca clientes por nombre, teléfono o email, con sus visitas y su próximo turno.",
			esquema: objeto([]string{"texto"}, map[string]any{
				"texto":  texto("Parte del nombre, del teléfono o del email."),
				"limite": map[string]any{"type": "integer", "minimum": 1, "maximum": 50, "description": "Cuántos traer. Por defecto, 10."},
			})},
		{nombre: "ver_cliente", titulo: "Ver la ficha de un cliente", permiso: VerClientes, correr: verCliente,
			descripcion: "Muestra la ficha de un cliente: contacto, visitas, lo que gastó, ausencias, si está penalizado y sus " +
				"últimos turnos.",
			esquema: objeto([]string{"clienteId"}, map[string]any{"clienteId": texto("El id del cliente (sale de buscar_clientes).")})},
		{nombre: "resumen_de_numeros", titulo: "Resumen de números", permiso: VerNumeros, correr: resumenNumeros,
			descripcion: "Los números de un período: turnos, atendidos, ausencias, cancelaciones, lo facturado (turnos atendidos) " +
				"y clientes nuevos. Por defecto, el mes en curso hasta hoy.",
			esquema: objeto(nil, map[string]any{
				"desde": fecha("Primer día, AAAA-MM-DD. Por defecto, el 1 del mes."),
				"hasta": fecha("Último día (inclusive), AAAA-MM-DD. Por defecto, hoy."),
			})},
	}
}

func objeto(requeridos []string, props map[string]any) map[string]any {
	m := map[string]any{"type": "object", "properties": props, "additionalProperties": false}
	if len(requeridos) > 0 {
		m["required"] = requeridos
	}
	return m
}

func texto(d string) map[string]any { return map[string]any{"type": "string", "description": d} }
func fecha(d string) map[string]any {
	return map[string]any{"type": "string", "description": d, "pattern": `^\d{4}-\d{2}-\d{2}$`}
}
func hora(d string) map[string]any {
	return map[string]any{"type": "string", "description": d, "pattern": `^\d{1,2}:\d{2}$`}
}

func buscarHerramienta(nombre string) *herramienta {
	for i := range herramientas {
		if herramientas[i].nombre == nombre {
			return &herramientas[i]
		}
	}
	return nil
}

// listar: sólo las herramientas que la clave tiene permitidas.
func listar(c *conexion) []map[string]any {
	out := []map[string]any{}
	for _, h := range herramientas {
		if !c.puede(h.permiso) {
			continue
		}
		out = append(out, map[string]any{
			"name": h.nombre, "title": h.titulo, "description": h.descripcion, "inputSchema": h.esquema,
			"annotations": map[string]any{"title": h.titulo, "readOnlyHint": !h.escribe, "destructiveHint": h.destructiva,
				"idempotentHint": !h.escribe || h.nombre == "confirmar_turno" || h.nombre == "cancelar_turno", "openWorldHint": false},
		})
	}
	return out
}

// llamar corre una herramienta (tools/call) y la deja en el registro.
func (s *Servicio) llamar(ctx context.Context, c *conexion, params json.RawMessage) (any, *errorRPC) {
	var in struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	}
	if err := json.Unmarshal(params, &in); err != nil || in.Name == "" {
		return nil, &errorRPC{errParametro, "tools/call necesita name."}
	}
	if len(in.Arguments) == 0 || string(in.Arguments) == "null" {
		in.Arguments = json.RawMessage("{}")
	}
	h := buscarHerramienta(in.Name)
	if h == nil || !c.puede(h.permiso) {
		msj := "La herramienta " + in.Name + " no existe o esta clave no tiene permiso para usarla (se cambia en el panel, en Integraciones e IA)."
		s.registrar(ctx, c, in.Name, in.Arguments, false, "", "sin permiso o desconocida", false)
		return nil, &errorRPC{errParametro, msj}
	}
	// La sesión de quien opera: el asistente de esa clave, con lo que puede un encargado.
	ctx = sesion.ConSesion(ctx, sesion.Sesion{ClerkID: Quien(c.tokenID), NegocioID: c.negocio, Rol: sesion.Encargado})
	var res any
	var resumen string
	err := s.puedeEscribir(ctx, h)
	if err == nil {
		res, resumen, err = h.correr(ctx, s, c, in.Arguments)
	}
	if err != nil {
		msj := explicar(err, h.nombre)
		s.registrar(ctx, c, h.nombre, in.Arguments, false, "", msj, h.escribe)
		return resultado(map[string]any{"error": msj}, true), nil
	}
	s.registrar(ctx, c, h.nombre, in.Arguments, true, resumen, "", h.escribe)
	return resultado(res, false), nil
}

// Quien: cómo figura el asistente de una clave en la auditoría (clerk_id).
func Quien(tokenID string) string { return "mcp:" + tokenID }

func resultado(v any, esError bool) map[string]any {
	b, _ := json.Marshal(v)
	return map[string]any{"content": []map[string]any{{"type": "text", "text": string(b)}}, "structuredContent": v, "isError": esError}
}

// puedeEscribir: con la prueba vencida el panel es de sólo lectura, y el asistente también. Usa el
// mismo chequeo que sesion.Panel le pone a las rutas que escriben.
func (s *Servicio) puedeEscribir(ctx context.Context, h *herramienta) error {
	if !h.escribe || s.Sesion == nil || s.Sesion.Escritura == nil {
		return nil
	}
	r, err := http.NewRequestWithContext(ctx, http.MethodPost, "/v1/mcp", nil)
	if err != nil {
		return err
	}
	c := &captura{cabeceras: http.Header{}}
	paso := false
	s.Sesion.Escritura(func(http.ResponseWriter, *http.Request) { paso = true })(c, r)
	if paso {
		return nil
	}
	var p problema.Problema
	if json.Unmarshal(c.cuerpo, &p) == nil && p.Estado != 0 {
		return &p
	}
	return problema.Interno()
}

// captura es un ResponseWriter que guarda lo que escribió el chequeo.
type captura struct {
	cabeceras http.Header
	cuerpo    []byte
}

func (c *captura) Header() http.Header         { return c.cabeceras }
func (c *captura) WriteHeader(int)             {}
func (c *captura) Write(b []byte) (int, error) { c.cuerpo = append(c.cuerpo, b...); return len(b), nil }

// explicar un error en una frase para el asistente (y el registro).
func explicar(err error, herramienta string) string {
	var p *problema.Problema
	var pg *pgconn.PgError
	switch {
	case errors.As(err, &p):
		if p.Detalle != "" {
			return p.Detalle
		}
		return p.Titulo
	case errors.Is(err, pgx.ErrNoRows):
		return "No existe o no es de este negocio."
	case errors.As(err, &pg) && (pg.Code == "22P02" || pg.Code == "22007" || pg.Code == "22008" || pg.Code == "23514"):
		return "Algún dato tiene un formato inválido: revisá los ids, las fechas y las horas."
	case errors.As(err, &pg) && pg.Code == "23503":
		return "Hace referencia a algo que no existe."
	default:
		slog.Error("mcp: error interno en una herramienta", "herramienta", herramienta, "error", err)
		return "Algo salió mal de nuestro lado. Probá de nuevo en un rato."
	}
}

// ---- Registro

// sensibles: lo que no se guarda tal cual en el registro.
var sensibles = map[string]bool{"telefono": true, "email": true, "nota": true}

// resumirArgs deja los argumentos para el registro: sin teléfonos, emails ni notas, y con los textos
// cortados.
func resumirArgs(raw json.RawMessage) any {
	var v any
	if json.Unmarshal(raw, &v) != nil {
		return nil
	}
	var limpiar func(v any, prof int) any
	limpiar = func(v any, prof int) any {
		switch x := v.(type) {
		case map[string]any:
			if prof > 3 {
				return "…"
			}
			out := map[string]any{}
			for k, val := range x {
				if len(out) >= 20 {
					break
				}
				if sensibles[k] {
					if s, ok := val.(string); ok && s == "" {
						continue
					}
					out[k] = "(dado)"
					continue
				}
				out[k] = limpiar(val, prof+1)
			}
			return out
		case []any:
			return "(lista de " + strconv.Itoa(len(x)) + ")"
		case string:
			return recortar(x, 80)
		default:
			return x
		}
	}
	return limpiar(v, 0)
}

func recortar(s string, n int) string {
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	r := []rune(s)
	return string(r[:n-1]) + "…"
}

func (s *Servicio) registrar(ctx context.Context, c *conexion, nombre string, args json.RawMessage, ok bool, resumen, msj string, escribe bool) {
	nombre = recortar(strings.TrimSpace(nombre), 60)
	if nombre == "" {
		return
	}
	a, _ := json.Marshal(resumirArgs(args))
	err := s.Pool.EnNegocio(context.WithoutCancel(ctx), c.negocio, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `INSERT INTO mcp_registro (negocio_id, token_id, herramienta, argumentos, ok, resumen, error, escribe)
			VALUES (app.negocio_actual(), $1, $2, $3, $4, $5, $6, $7)`, c.tokenID, nombre, a, ok, recortar(resumen, 300), recortar(msj, 300), escribe)
		return err
	})
	if err != nil {
		slog.Error("mcp: no se pudo anotar en el registro", "herramienta", nombre, "error", err)
	}
}

// anotarCambio: lo que cambió algo va a la campanita y a la auditoría, en nombre del asistente.
func (s *Servicio) anotarCambio(ctx context.Context, c *conexion, tipo, titulo, texto, turnoID, accion string) {
	err := s.Pool.EnNegocio(context.WithoutCancel(ctx), c.negocio, func(tx pgx.Tx) error {
		if err := avisos.Notificar(ctx, tx, avisos.Notificacion{Tipo: tipo, Titulo: titulo, Texto: texto + " Lo pidió «" + c.nombre + "».",
			TurnoID: turnoID}); err != nil {
			return err
		}
		return auditoria.Anotar(ctx, tx, auditoria.Registro{Quien: Quien(c.tokenID), Accion: "asistente." + accion, Entidad: "turno",
			EntidadID: turnoID, Detalle: map[string]any{"clave": c.nombre, "resumen": texto}})
	})
	if err != nil {
		slog.Error("mcp: no se pudo avisar el cambio", "accion", accion, "error", err)
	}
}

// ---- Fechas en la zona del negocio

func (c *conexion) dia(v string) (time.Time, error) {
	t, err := time.ParseInLocation(time.DateOnly, strings.TrimSpace(v), c.zona)
	if err != nil {
		return t, problema.Invalido("La fecha va como AAAA-MM-DD.")
	}
	return t, nil
}

func (c *conexion) momento(f, h string) (time.Time, error) {
	h = strings.TrimSpace(h)
	if len(h) == 4 {
		h = "0" + h
	}
	t, err := time.ParseInLocation("2006-01-02 15:04", strings.TrimSpace(f)+" "+h, c.zona)
	if err != nil {
		return t, problema.Invalido("La fecha va como AAAA-MM-DD y la hora como HH:MM.")
	}
	return t, nil
}

func (c *conexion) hoy(ahora time.Time) time.Time {
	h := ahora.In(c.zona)
	return time.Date(h.Year(), h.Month(), h.Day(), 0, 0, 0, 0, c.zona)
}

// cuando: «lunes 2 de noviembre a las 10:00».
func (c *conexion) cuando(t time.Time) string {
	t = t.In(c.zona)
	return avisos.Fecha(t) + " a las " + avisos.Hora(t)
}
