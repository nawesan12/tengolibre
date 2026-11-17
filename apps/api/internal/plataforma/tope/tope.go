// Package tope: límites de pedidos por clave (una IP, un negocio) en una ventana fija. En memoria:
// la API es un solo proceso. Lo usan las rutas públicas que se pueden abusar sin sesión (reservar,
// anotarse en la lista de espera, probar cupones y códigos de referido, el MCP público).
package tope

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/problema"
)

// Contador cuenta pedidos por clave. El valor cero sirve.
type Contador struct {
	mu      sync.Mutex
	cuentas map[string]*cuenta
}

type cuenta struct {
	desde time.Time
	n     int
}

// Permitir suma un pedido de clave y dice si entra en max por ventana.
func (l *Contador) Permitir(clave string, max int, ventana time.Duration, ahora time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.cuentas == nil {
		l.cuentas = map[string]*cuenta{}
	}
	// Limpieza de lo vencido cuando crece (no hace falta un barrido aparte).
	if len(l.cuentas) > 10000 {
		for k, c := range l.cuentas {
			if ahora.Sub(c.desde) > time.Hour {
				delete(l.cuentas, k)
			}
		}
	}
	c, ok := l.cuentas[clave]
	if !ok || ahora.Sub(c.desde) >= ventana {
		l.cuentas[clave] = &cuenta{desde: ahora, n: 1}
		return max > 0
	}
	c.n++
	return c.n <= max
}

// IP del cliente. La API sólo escucha detrás de Caddy, que sólo acepta a Cloudflare: el
// CF-Connecting-IP lo pone Cloudflare. Sin él (desarrollo, tests), la dirección de la conexión.
func IP(r *http.Request) string {
	if v := strings.TrimSpace(r.Header.Get("CF-Connecting-IP")); v != "" && net.ParseIP(v) != nil {
		return v
	}
	if h, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return h
	}
	return r.RemoteAddr
}

// Demasiados: el 429 de un tope.
func Demasiados(codigo, detalle string) *problema.Problema {
	return &problema.Problema{Estado: http.StatusTooManyRequests, Codigo: codigo, Titulo: "Demasiados intentos", Detalle: detalle}
}

// Publico: el tope de un pedido público que crea algo (reservar, anotarse en la lista de espera):
// por IP en cada negocio y, para lo que venga de muchas IPs, por negocio en total.
type Publico struct {
	PorIP, PorNegocio int
	Ventana           time.Duration
}

// PorDefecto: 10 cada 10 minutos por IP y 100 por negocio.
var PorDefecto = Publico{PorIP: 10, PorNegocio: 100, Ventana: 10 * time.Minute}

// PermitirPublico cuenta el pedido (accion: "reservar", "espera") y devuelve un 429 si se pasa. Un
// Publico vacío usa PorDefecto.
func (l *Contador) PermitirPublico(r *http.Request, accion, negocio string, p Publico, ahora time.Time) error {
	if p.Ventana <= 0 {
		p = PorDefecto
	}
	if !l.Permitir(accion+":"+negocio+":"+IP(r), p.PorIP, p.Ventana, ahora) ||
		!l.Permitir(accion+":"+negocio, p.PorNegocio, p.Ventana, ahora) {
		return Demasiados("demasiados_pedidos", "Hubo muchos pedidos seguidos desde esta conexión. Esperá unos minutos y volvé a probar.")
	}
	return nil
}
