package calendario

import (
	"strings"
	"time"
	"unicode/utf8"
)

// Evento de un turno, ya listo para el calendario.
type Evento struct {
	UID, Titulo, Descripcion, URL string
	Inicio, Fin, Cambio            time.Time
	// Tentativo: todavía no está confirmado (pendiente o esperando el pago).
	Tentativo bool
}

// ICS arma el VCALENDAR (RFC 5545). Las horas van en UTC: cada calendario las muestra en la zona de
// quien mira, que para un negocio es la misma.
func ICS(nombre string, eventos []Evento) []byte {
	var b strings.Builder
	l := func(s string) { linea(&b, s) }
	l("BEGIN:VCALENDAR")
	l("VERSION:2.0")
	l("PRODID:-//Tengo Libre//Turnos//ES")
	l("CALSCALE:GREGORIAN")
	l("METHOD:PUBLISH")
	l("X-WR-CALNAME:" + texto(nombre))
	// Cada cuánto conviene releerlo (Google lo ignora y lee cuando quiere; Apple y Outlook lo usan).
	l("REFRESH-INTERVAL;VALUE=DURATION:PT15M")
	l("X-PUBLISHED-TTL:PT15M")
	for _, e := range eventos {
		l("BEGIN:VEVENT")
		l("UID:" + e.UID)
		l("DTSTAMP:" + utc(e.Cambio))
		l("LAST-MODIFIED:" + utc(e.Cambio))
		l("DTSTART:" + utc(e.Inicio))
		l("DTEND:" + utc(e.Fin))
		l("SUMMARY:" + texto(e.Titulo))
		if e.Descripcion != "" {
			l("DESCRIPTION:" + texto(e.Descripcion))
		}
		if e.URL != "" {
			l("URL:" + e.URL)
		}
		if e.Tentativo {
			l("STATUS:TENTATIVE")
		} else {
			l("STATUS:CONFIRMED")
		}
		l("END:VEVENT")
	}
	l("END:VCALENDAR")
	return []byte(b.String())
}

func utc(t time.Time) string { return t.UTC().Format("20060102T150405Z") }

// texto escapa un valor TEXT: barra, coma, punto y coma y saltos de línea.
func texto(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	r := strings.NewReplacer(`\`, `\\`, ";", `\;`, ",", `\,`, "\n", `\n`, "\r", "")
	return r.Replace(s)
}

// linea escribe una línea plegada a 75 octetos (sin cortar un carácter) y terminada en CRLF.
func linea(b *strings.Builder, s string) {
	max := 75
	for len(s) > max {
		corte := max
		for corte > 0 && !utf8.RuneStart(s[corte]) {
			corte--
		}
		b.WriteString(s[:corte])
		b.WriteString("\r\n ")
		s = s[corte:]
		max = 74 // la continuación ya lleva el espacio
	}
	b.WriteString(s)
	b.WriteString("\r\n")
}
