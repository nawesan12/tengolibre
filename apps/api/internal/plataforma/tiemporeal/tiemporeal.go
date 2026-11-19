// Package tiemporeal reparte por SSE (Server-Sent Events) los cambios que avisa Postgres en el
// canal `tengolibre_eventos` (ver 00008_tiempo_real.sql) a los paneles abiertos de cada negocio.
// Cada aviso dice sólo qué cambió; el panel vuelve a pedir lo que necesite.
//
// SSE y no WebSockets: el panel sólo necesita escuchar, pasa tal cual por Cloudflare y Caddy, y el
// navegador se reconecta solo (EventSource).
package tiemporeal

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"slices"
	"sync"
	"time"

	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/db"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/problema"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/sesion"
)

// Canal de Postgres.
const Canal = "tengolibre_eventos"

// MaxPorNegocio: paneles abiertos a la vez por negocio (tope de memoria y conexiones).
const MaxPorNegocio = 50

// MaxPorPersona: paneles abiertos a la vez por persona en un negocio (sin esto, un solo miembro podía
// ocupar los cupos del negocio entero).
const MaxPorPersona = 5

// Revalidar: cada cuánto un stream abierto vuelve a mirar que quien lo abrió siga en el negocio (y
// con un rol del panel): una baja o un cambio de rol lo corta.
var Revalidar = 3 * time.Minute

// Latido: cada cuánto se manda un comentario para que Cloudflare (corta a los 100 s sin datos) y
// los proxies no cierren la conexión.
var Latido = 25 * time.Second

// Evento es lo que avisa la base.
type Evento struct {
	Negocio string `json:"negocio"`
	Tipo    string `json:"tipo"`
	ID      string `json:"id"`
	Op      string `json:"op"`
}

// Servicio reparte los eventos.
type Servicio struct {
	Pool   *db.Pool
	Sesion *sesion.Servicio
	// AlPublicar, si no es nil, recibe cada evento (por ejemplo, para olvidar respuestas guardadas del
	// asistente cuando cambia la agenda del negocio).
	AlPublicar func(Evento)

	mu         sync.Mutex
	suscriptos map[string]map[chan Evento]string
}

// Escuchar arranca la escucha de Postgres (bloquea hasta que se cancele ctx).
func (s *Servicio) Escuchar(ctx context.Context) {
	s.Pool.Escuchar(ctx, Canal, func(carga string) {
		var e Evento
		if err := json.Unmarshal([]byte(carga), &e); err != nil || e.Negocio == "" {
			slog.Warn("tiempo real: aviso ilegible", "carga", carga)
			return
		}
		s.Publicar(e)
	})
}

// Publicar manda e a los suscriptos de su negocio. A quien no da abasto (canal lleno) se le saltea
// el evento: el panel igual se pone al día en el próximo.
func (s *Servicio) Publicar(e Evento) {
	if s.AlPublicar != nil {
		s.AlPublicar(e)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for c := range s.suscriptos[e.Negocio] {
		select {
		case c <- e:
		default:
		}
	}
}

func (s *Servicio) suscribir(negocio, quien string) (chan Evento, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.suscriptos == nil {
		s.suscriptos = map[string]map[chan Evento]string{}
	}
	if len(s.suscriptos[negocio]) >= MaxPorNegocio {
		return nil, false
	}
	propios := 0
	for _, q := range s.suscriptos[negocio] {
		if q == quien {
			propios++
		}
	}
	if propios >= MaxPorPersona {
		return nil, false
	}
	c := make(chan Evento, 32)
	if s.suscriptos[negocio] == nil {
		s.suscriptos[negocio] = map[chan Evento]string{}
	}
	s.suscriptos[negocio][c] = quien
	return c, true
}

// sigue: quien abrió el stream sigue siendo miembro activo del negocio con un rol del panel.
func (s *Servicio) sigue(ctx context.Context, negocio, quien string) bool {
	if s.Sesion == nil || s.Sesion.Pool == nil {
		return true
	}
	ns, err := s.Sesion.Negocios(ctx, quien)
	if err != nil {
		// Un error de la base no corta el stream: se vuelve a mirar en la próxima vuelta.
		return true
	}
	for _, n := range ns {
		if n.ID == negocio && slices.Contains(sesion.Agenda, n.Rol) {
			return true
		}
	}
	return false
}

func (s *Servicio) desuscribir(negocio string, c chan Evento) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.suscriptos[negocio], c)
	if len(s.suscriptos[negocio]) == 0 {
		delete(s.suscriptos, negocio)
	}
}

// Suscriptos de un negocio (tests y métricas).
func (s *Servicio) Suscriptos(negocio string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.suscriptos[negocio])
}

// Rutas del tiempo real.
func (s *Servicio) Rutas(mux *http.ServeMux) {
	mux.HandleFunc("GET /v1/panel/eventos", s.Sesion.Panel(sesion.Agenda, s.eventos))
}

func (s *Servicio) eventos(w http.ResponseWriter, r *http.Request) {
	ses := sesion.De(r.Context())
	negocio := ses.NegocioID
	c, ok := s.suscribir(negocio, ses.ClerkID)
	if !ok {
		w.Header().Set("Retry-After", "30")
		problema.Escribir(w, &problema.Problema{Estado: http.StatusServiceUnavailable, Codigo: "demasiados_paneles",
			Titulo: "Hay muchos paneles abiertos", Detalle: "Cerrá alguna pestaña y volvé a intentar."})
		return
	}
	defer s.desuscribir(negocio, c)
	rc := http.NewResponseController(w)
	// El servidor corta las respuestas a los 30 s (WriteTimeout): este stream vive lo que dure el panel.
	_ = rc.SetWriteDeadline(time.Time{})
	h := w.Header()
	h.Set("Content-Type", "text/event-stream")
	h.Set("Cache-Control", "no-store")
	h.Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)
	// retry: si se corta, el navegador vuelve a los 3 s.
	fmt.Fprint(w, "retry: 3000\n: conectado\n\n")
	if rc.Flush() != nil {
		return
	}
	latido := time.NewTicker(Latido)
	defer latido.Stop()
	revisar := time.NewTicker(Revalidar)
	defer revisar.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case <-revisar.C:
			if !s.sigue(r.Context(), negocio, ses.ClerkID) {
				return
			}
			continue
		case <-latido.C:
			fmt.Fprint(w, ": latido\n\n")
		case e := <-c:
			b, _ := json.Marshal(e)
			fmt.Fprintf(w, "event: %s\ndata: %s\n\n", e.Tipo, b)
		}
		if rc.Flush() != nil {
			return
		}
	}
}
