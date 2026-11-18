package ia

import (
	"strings"
	"sync"
	"time"
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

// Cache de respuestas: la misma pregunta (normalizada) de la misma persona en el mismo negocio,
// dentro de TTL, devuelve lo que ya se contestó sin llamar al modelo. Sólo se guardan respuestas
// que no propusieron cambios. En memoria: es un ahorro, no un dato (un reinicio lo vacía y listo).
type Cache struct {
	TTL   time.Duration
	Ahora func() time.Time

	mu sync.Mutex
	m  map[string]entrada
}

type entrada struct {
	texto string
	vence time.Time
}

// TTLRespuestas por defecto.
const TTLRespuestas = 10 * time.Minute

const maxEntradas = 2000

func (c *Cache) ahora() time.Time {
	if c.Ahora != nil {
		return c.Ahora()
	}
	return time.Now()
}

// Clave de una pregunta: negocio, quién (un profesional ve otra agenda que la dueña) y el texto
// normalizado.
func Clave(negocio, quien, pregunta string) string {
	return negocio + "|" + quien + "|" + Normalizar(pregunta)
}

var sinTildes = transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)

// Normalizar: minúsculas, sin tildes, sin signos y con un espacio entre palabras.
func Normalizar(s string) string {
	s, _, _ = transform.String(sinTildes, strings.ToLower(s))
	var b strings.Builder
	espacio := false
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == ':' {
			if espacio && b.Len() > 0 {
				b.WriteByte(' ')
			}
			b.WriteRune(r)
			espacio = false
		} else {
			espacio = true
		}
	}
	return b.String()
}

// Leer una respuesta guardada.
func (c *Cache) Leer(clave string) (string, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.m[clave]
	if !ok || !c.ahora().Before(e.vence) {
		return "", false
	}
	return e.texto, true
}

// Guardar una respuesta.
func (c *Cache) Guardar(clave, texto string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.m == nil {
		c.m = map[string]entrada{}
	}
	ahora := c.ahora()
	if len(c.m) >= maxEntradas {
		for k, e := range c.m {
			if !ahora.Before(e.vence) {
				delete(c.m, k)
			}
		}
		if len(c.m) >= maxEntradas {
			c.m = map[string]entrada{}
		}
	}
	ttl := c.TTL
	if ttl <= 0 {
		ttl = TTLRespuestas
	}
	c.m[clave] = entrada{texto: texto, vence: ahora.Add(ttl)}
}

// Olvidar lo guardado de un negocio (después de un cambio que hizo el propio asistente).
func (c *Cache) Olvidar(negocio string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for k := range c.m {
		if strings.HasPrefix(k, negocio+"|") {
			delete(c.m, k)
		}
	}
}
