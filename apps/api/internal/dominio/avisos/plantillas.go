package avisos

import (
	"bytes"
	"fmt"
	"html/template"
	"strings"
	"time"

	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/correo"
)

// Fila de datos de un mail ("Servicio: Corte").
type Fila struct{ Etiqueta, Valor string }

// Carta es el contenido de un mail. Todos usan el mismo diseño (el de design/emails de Jiku): el
// negocio arriba, un título, el saludo, párrafos, los datos del turno, un botón y notas chicas.
type Carta struct {
	Asunto     string
	Encabezado string
	Titulo     string
	Saludo     string
	Parrafos   []string
	Filas      []Fila
	Lista      []string
	Boton      string
	Link       string
	Notas      []string
	// Baja: el link a las preferencias del cliente (va al pie y en List-Unsubscribe).
	Baja string
}

var diseno = template.Must(template.New("mail").Parse(`<!doctype html>
<html lang="es"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width"><title>{{.Asunto}}</title></head>
<body style="margin:0;padding:0;background:#f4f2ee;font-family:-apple-system,Segoe UI,Roboto,Helvetica,Arial,sans-serif;color:#1d1b18">
<table role="presentation" width="100%" cellpadding="0" cellspacing="0" style="background:#f4f2ee;padding:24px 12px"><tr><td align="center">
<table role="presentation" width="100%" cellpadding="0" cellspacing="0" style="max-width:520px;background:#ffffff;border-radius:12px;padding:28px">
<tr><td style="font-size:14px;font-weight:600;color:#6b665e;padding-bottom:16px">{{.Encabezado}}</td></tr>
<tr><td style="font-size:22px;font-weight:700;line-height:1.3;padding-bottom:16px">{{.Titulo}}</td></tr>
{{if .Saludo}}<tr><td style="font-size:16px;line-height:1.5;padding-bottom:12px">{{.Saludo}}</td></tr>{{end}}
{{range .Parrafos}}<tr><td style="font-size:16px;line-height:1.5;padding-bottom:12px">{{.}}</td></tr>{{end}}
{{if .Filas}}<tr><td style="padding:8px 0 16px"><table role="presentation" width="100%" cellpadding="0" cellspacing="0" style="background:#f8f7f4;border-radius:8px;padding:12px 16px">
{{range .Filas}}<tr><td style="font-size:14px;color:#6b665e;padding:4px 12px 4px 0;white-space:nowrap;vertical-align:top">{{.Etiqueta}}</td><td style="font-size:15px;padding:4px 0">{{.Valor}}</td></tr>{{end}}
</table></td></tr>{{end}}
{{if .Lista}}<tr><td style="padding:4px 0 16px">{{range .Lista}}<div style="font-size:15px;line-height:1.5;padding:6px 0;border-bottom:1px solid #eeebe5">{{.}}</div>{{end}}</td></tr>{{end}}
{{if .Link}}<tr><td style="padding:8px 0 20px"><a href="{{.Link}}" style="display:inline-block;background:#1d1b18;color:#ffffff;text-decoration:none;font-size:15px;font-weight:600;padding:12px 20px;border-radius:8px">{{.Boton}}</a></td></tr>{{end}}
{{range .Notas}}<tr><td style="font-size:13px;line-height:1.5;color:#6b665e;padding-bottom:8px">{{.}}</td></tr>{{end}}
</table>
<table role="presentation" width="100%" cellpadding="0" cellspacing="0" style="max-width:520px"><tr><td style="font-size:12px;color:#8a857c;padding:16px 8px;text-align:center">
Turnos online por TengoLibre{{if .Baja}} · <a href="{{.Baja}}" style="color:#8a857c">Elegir qué mails recibir</a>{{end}}
</td></tr></table>
</td></tr></table></body></html>`))

// HTML de la carta.
func (c Carta) HTML() string {
	var b bytes.Buffer
	_ = diseno.Execute(&b, c)
	return b.String()
}

// Texto de la carta (para los clientes de mail que no muestran HTML).
func (c Carta) Texto() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s\n\n%s\n\n", c.Encabezado, c.Titulo)
	if c.Saludo != "" {
		b.WriteString(c.Saludo + "\n\n")
	}
	for _, p := range c.Parrafos {
		b.WriteString(p + "\n\n")
	}
	for _, f := range c.Filas {
		fmt.Fprintf(&b, "%s: %s\n", f.Etiqueta, f.Valor)
	}
	if len(c.Filas) > 0 {
		b.WriteString("\n")
	}
	for _, l := range c.Lista {
		b.WriteString(l + "\n")
	}
	if len(c.Lista) > 0 {
		b.WriteString("\n")
	}
	if c.Link != "" {
		fmt.Fprintf(&b, "%s: %s\n\n", c.Boton, c.Link)
	}
	for _, n := range c.Notas {
		b.WriteString(n + "\n\n")
	}
	b.WriteString("—\nTurnos online por TengoLibre\n")
	if c.Baja != "" {
		b.WriteString("Elegir qué mails recibir: " + c.Baja + "\n")
	}
	return b.String()
}

// Mensaje arma el mail para `para`.
func (c Carta) Mensaje(para string) correo.Mensaje {
	return correo.Mensaje{Para: para, Asunto: c.Asunto, HTML: c.HTML(), Texto: c.Texto(), Baja: c.Baja}
}

var (
	dias  = [...]string{"domingo", "lunes", "martes", "miércoles", "jueves", "viernes", "sábado"}
	meses = [...]string{"enero", "febrero", "marzo", "abril", "mayo", "junio", "julio", "agosto", "septiembre", "octubre",
		"noviembre", "diciembre"}
)

// Fecha como en los mails: "sábado 12 de septiembre" (en la zona de t).
func Fecha(t time.Time) string {
	return fmt.Sprintf("%s %d de %s", dias[t.Weekday()], t.Day(), meses[t.Month()-1])
}

// Hora: "10:30".
func Hora(t time.Time) string { return t.Format("15:04") }

// primerNombre: "Lucía Fernández" → "Lucía".
func primerNombre(nombre string) string {
	if f := strings.Fields(nombre); len(f) > 0 {
		return f[0]
	}
	return nombre
}

// ICS arma el evento de calendario del turno. El UID es el del turno: si se reprograma, el
// calendario actualiza el mismo evento (SEQUENCE crece con el tiempo).
func ICS(uid, titulo, descripcion, lugar string, inicio, fin, ahora time.Time) []byte {
	f := func(t time.Time) string { return t.UTC().Format("20060102T150405Z") }
	lineas := []string{
		"BEGIN:VCALENDAR", "VERSION:2.0", "PRODID:-//TengoLibre//Turnos//ES", "CALSCALE:GREGORIAN", "METHOD:PUBLISH",
		"BEGIN:VEVENT",
		"UID:" + uid + "@tengolibre.com.ar",
		fmt.Sprintf("SEQUENCE:%d", ahora.Unix()/60),
		"DTSTAMP:" + f(ahora),
		"DTSTART:" + f(inicio),
		"DTEND:" + f(fin),
		"SUMMARY:" + escaparICS(titulo),
	}
	if descripcion != "" {
		lineas = append(lineas, "DESCRIPTION:"+escaparICS(descripcion))
	}
	if lugar != "" {
		lineas = append(lineas, "LOCATION:"+escaparICS(lugar))
	}
	lineas = append(lineas, "STATUS:CONFIRMED",
		"BEGIN:VALARM", "TRIGGER:-PT1H", "ACTION:DISPLAY", "DESCRIPTION:Recordatorio de turno", "END:VALARM",
		"END:VEVENT", "END:VCALENDAR")
	var b strings.Builder
	for _, l := range lineas {
		b.WriteString(plegar(l))
		b.WriteString("\r\n")
	}
	return []byte(b.String())
}

func escaparICS(s string) string {
	return strings.NewReplacer(`\`, `\\`, ";", `\;`, ",", `\,`, "\n", `\n`, "\r", "").Replace(s)
}

// plegar corta las líneas de más de 75 bytes (RFC 5545) sin partir un carácter UTF-8.
func plegar(l string) string {
	if len(l) <= 75 {
		return l
	}
	var b strings.Builder
	n := 0
	for _, r := range l {
		tam := len(string(r))
		if n+tam > 75 {
			b.WriteString("\r\n ")
			n = 1
		}
		b.WriteRune(r)
		n += tam
	}
	return b.String()
}
