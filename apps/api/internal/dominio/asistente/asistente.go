// Package asistente: el asistente del panel («¿qué tengo mañana?», «pasá a Juana al jueves a las
// 5»). Primero contesta el intérprete propio (interprete.go), al instante y sin costo; lo que no
// entiende con certeza va a Claude Haiku (plataforma/ia) con las herramientas del MCP del negocio,
// con el rol de quien pregunta. Nada que cambie la agenda se hace directo: vuelve como propuesta y
// la persona la confirma con un botón (propuestas.go).
package asistente

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"

	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/catalogo"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/espera"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/mcp"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/turnos"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/db"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/ia"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/problema"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/sesion"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/web"
)

// Ruta con la que se mide el uso en ia_uso.
const Ruta = "asistente"

// Servicio del asistente. IA nil: sólo el intérprete. Ahora es el reloj (tests); nil: time.Now.
type Servicio struct {
	Pool    *db.Pool
	Sesion  *sesion.Servicio
	MCP     *mcp.Servicio
	Espera  *espera.Servicio
	IA      ia.Cliente
	Medidor *ia.Medidor
	Cache   *ia.Cache
	Ahora   func() time.Time
}

func (s *Servicio) ahora() time.Time {
	if s.Ahora != nil {
		return s.Ahora()
	}
	return time.Now()
}

// Rutas: cualquiera del equipo, con lo que su rol puede hacer.
func (s *Servicio) Rutas(mux *http.ServeMux) {
	a := func(h web.Manejador) http.HandlerFunc { return s.Sesion.Panel(sesion.Agenda, web.Envolver(h)) }
	mux.HandleFunc("GET /v1/panel/asistente", a(s.estado))
	mux.HandleFunc("POST /v1/panel/asistente", a(s.preguntar))
	mux.HandleFunc("POST /v1/panel/asistente/confirmar", a(s.confirmar))
	mux.HandleFunc("POST /v1/panel/asistente/descartar", a(s.descartar))
}

// SinChequeo: preguntar escribe (es un POST) pero no cambia nada; con el panel en sólo lectura se
// puede seguir preguntando. Confirmar una propuesta sí pasa por el chequeo (Operador.Hacer).
const SinChequeo = "/v1/panel/asistente"

// Sugerencias: lo que el intérprete entiende, para mostrar de ejemplo.
var Sugerencias = []string{"¿Qué tengo mañana?", "¿Hay lugar el viernes a la tarde?", "¿Quién viene a las 5?",
	"Pasá a Juana al jueves a las 5", "Cancelá el de las 3", "Confirmá a Pedro"}

// Respuesta a una pregunta.
type Respuesta struct {
	Respuesta  string      `json:"respuesta"`
	Propuestas []Propuesta `json:"propuestas"`
	// Origen: motor (el intérprete, sin IA), ia o cache.
	Origen string `json:"origen"`
	// Tope: no hay más IA por ahora (motivo y mensaje para mostrar).
	Tope *Tope `json:"tope,omitempty"`
}

// Tope de la IA alcanzado.
type Tope struct {
	Motivo  string `json:"motivo"`
	Mensaje string `json:"mensaje"`
}

func (s *Servicio) conIA() bool { return s.IA != nil && s.Medidor != nil }

// estado: si hay IA y cuánto queda del cupo del mes.
func (s *Servicio) estado(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	out := map[string]any{"ia": s.conIA(), "sugerencias": Sugerencias}
	if s.Medidor != nil {
		e, err := s.Medidor.Leer(ctx, sesion.De(ctx).NegocioID)
		if err != nil {
			return err
		}
		out["cupo"] = e
	}
	web.JSON(w, http.StatusOK, out)
	return nil
}

// pedido de una pregunta.
type pedido struct {
	Pregunta  string     `json:"pregunta"`
	Historial []ia.Turno `json:"historial"`
}

func (s *Servicio) preguntar(w http.ResponseWriter, r *http.Request) error {
	var in pedido
	if err := web.Leer(w, r, &in); err != nil {
		return err
	}
	in.Pregunta = strings.TrimSpace(in.Pregunta)
	if in.Pregunta == "" || utf8.RuneCountInString(in.Pregunta) > 500 {
		return problema.Invalido("Escribí la pregunta (hasta 500 caracteres).")
	}
	if len(in.Historial) > 20 {
		in.Historial = in.Historial[len(in.Historial)-20:]
	}
	res, err := s.Responder(r.Context(), in.Pregunta, in.Historial)
	if err != nil {
		return err
	}
	if res.Propuestas == nil {
		res.Propuestas = []Propuesta{}
	}
	web.JSON(w, http.StatusOK, res)
	return nil
}

// Responder una pregunta con la sesión del contexto: primero el intérprete; si no está seguro, la IA.
func (s *Servicio) Responder(ctx context.Context, pregunta string, historial []ia.Turno) (Respuesta, error) {
	op, err := s.MCP.Operador(ctx)
	if err != nil {
		return Respuesta{}, err
	}
	cat, nombre, err := s.catalogo(ctx, op)
	if err != nil {
		return Respuesta{}, err
	}
	if in, ok := Interpretar(pregunta, cat); ok {
		res, err := s.motor(ctx, op, cat, in)
		if err == nil {
			res.Origen = "motor"
			return res, nil
		}
		// Algo del motor falló (un turno que no se puede mover, un permiso): se explica tal cual.
		var p *problema.Problema
		if errors.As(err, &p) {
			return Respuesta{Respuesta: mcp.Explicar(err, "motor"), Origen: "motor"}, nil
		}
		return Respuesta{}, err
	}
	if !s.conIA() {
		return Respuesta{Origen: "motor", Respuesta: "Eso todavía no lo sé responder. Probá con algo como «¿Qué tengo mañana?», " +
			"«¿Hay lugar el viernes a la tarde?» o «Pasá a Juana al jueves a las 5»."}, nil
	}
	ses := sesion.De(ctx)
	clave := ia.Clave(ses.NegocioID, ses.ClerkID, pregunta)
	if len(historial) == 0 && s.Cache != nil {
		if txt, ok := s.Cache.Leer(clave); ok {
			return Respuesta{Respuesta: txt, Origen: "cache"}, nil
		}
	}
	if err := s.Medidor.Chequear(ctx, ses.NegocioID); err != nil {
		var a *ia.Agotado
		if errors.As(err, &a) {
			return Respuesta{Respuesta: a.Mensaje, Origen: "motor", Tope: &Tope{Motivo: a.Motivo, Mensaje: a.Mensaje}}, nil
		}
		return Respuesta{}, err
	}
	conv := &conversacion{s: s, op: op}
	p := ia.Pedido{Sistema: Sistema, Datos: datos(cat, nombre, ses.Rol), Historial: historial, Pregunta: pregunta,
		Herramientas: conv.herramientas(), MaxTokens: 800}
	r, err := ia.Conversar(ctx, s.IA, p, conv.correr)
	if r.Uso.Pedidos > 0 {
		if e := s.Medidor.Anotar(ctx, ses.NegocioID, Ruta, r.Uso, 1); e != nil {
			slog.Error("asistente: no se pudo anotar el uso de la IA", "error", e)
		}
	}
	if err != nil {
		slog.Warn("asistente: la IA no respondió", "error", err)
		return Respuesta{Origen: "motor", Respuesta: "No pude consultar a la IA, probá de nuevo en un rato. Mientras tanto entiendo " +
			"cosas como «¿Qué tengo mañana?» o «Pasá a Juana al jueves a las 5».", Propuestas: conv.propuestas}, nil
	}
	if len(conv.propuestas) == 0 && len(historial) == 0 && !r.Rechazo && s.Cache != nil {
		s.Cache.Guardar(clave, r.Texto)
	}
	return Respuesta{Respuesta: r.Texto, Propuestas: conv.propuestas, Origen: "ia"}, nil
}

// catalogo del negocio para el intérprete (un profesional, con sus profesionales nada más).
func (s *Servicio) catalogo(ctx context.Context, op *mcp.Operador) (Catalogo, string, error) {
	c := Catalogo{Ahora: s.ahora().In(op.Zona())}
	var nombre string
	err := s.Pool.EnNegocio(ctx, sesion.De(ctx).NegocioID, func(tx pgx.Tx) error {
		if err := tx.QueryRow(ctx, "SELECT nombre FROM negocios WHERE id = app.negocio_actual()").Scan(&nombre); err != nil {
			return err
		}
		srvs, err := catalogo.Servicios(ctx, tx, true)
		if err != nil {
			return err
		}
		for _, x := range srvs {
			c.Servicios = append(c.Servicios, ServicioCat{x.ID, x.Nombre, x.Duracion})
		}
		ps, err := catalogo.Profesionales(ctx, tx, true)
		if err != nil {
			return err
		}
		var mios []catalogo.Profesional
		for _, p := range ps {
			if op.Permitido(p.ID) {
				c.Profesionales = append(c.Profesionales, Nombrado{p.ID, p.Nombre})
				mios = append(mios, p)
			}
		}
		c.Atencion = turnos.HorarioAtencion(mios)
		return nil
	})
	return c, nombre, err
}
