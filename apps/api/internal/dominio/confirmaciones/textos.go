package confirmaciones

import (
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/avisos"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/whatsapp"
)

// Plantillas que hay que dar de alta en Meta (docs/api/whatsapp.md): nombre e idioma tienen que
// coincidir con lo aprobado, y los parámetros van en este orden.
const (
	// «Hola {{1}} 👋 Te esperamos mañana {{2}} a las {{3}} en {{4}} para tu {{5}} con {{6}}. ¿Venís?»
	// Botones de respuesta rápida: «Confirmo» y «Necesito cambiarlo».
	PlantillaConfirmacion = "tengolibre_confirmacion"
	// «Hola {{1}}, te recordamos tu turno en {{2}}: {{3}}, el {{4}} a las {{5}}.»
	// Botón de link «Ver mi turno»: https://tengolibre.com.ar/turno/{{1}} (el código del turno).
	PlantillaRecordatorio = "tengolibre_recordatorio"
	Idioma                = "es_AR"
)

// Lo que vuelve en el payload de cada botón: la acción y el token del mensaje.
const (
	Confirmo = "confirmo"
	Cambiar  = "cambiar"
)

var dias = [...]string{"domingo", "lunes", "martes", "miércoles", "jueves", "viernes", "sábado"}

// Dia: «lunes 2».
func Dia(t time.Time) string { return dias[t.Weekday()] + " " + strconv.Itoa(t.Day()) }

// enMinuscula baja la primera letra («Lifting» → «lifting») salvo que sea una sigla («HIFU»).
func enMinuscula(s string) string {
	r, n := utf8.DecodeRuneInString(s)
	if n == 0 {
		return s
	}
	if r2, _ := utf8.DecodeRuneInString(s[n:]); unicode.IsUpper(r2) {
		return s
	}
	return string(unicode.ToLower(r)) + s[n:]
}

// noVacio: Meta rechaza un parámetro vacío.
func noVacio(s, otro string) string {
	if s = strings.TrimSpace(s); s == "" {
		return otro
	}
	return s
}

// primerNombre: «Sofía Méndez» → «Sofía».
func primerNombre(n string) string {
	if f := strings.Fields(n); len(f) > 0 {
		return f[0]
	}
	return n
}

// datosTurno: lo que llevan los mensajes de un turno.
type datosTurno struct {
	Codigo, Estado        string
	Cliente, Telefono     string
	Servicio, Profesional string
	Inicio                time.Time
	ConfirmadoPor         *string
	Negocio, Slug         string
	Zona                  *time.Location
	Recordatorios         bool
}

// PlantillaDeConfirmacion arma el mensaje del día antes, con el token en los dos botones.
func PlantillaDeConfirmacion(para, token string, d datosTurno) whatsapp.Plantilla {
	l := d.Inicio.In(d.Zona)
	return whatsapp.Plantilla{Para: para, Nombre: PlantillaConfirmacion, Idioma: Idioma,
		Cuerpo: []string{noVacio(primerNombre(d.Cliente), "cliente"), Dia(l), avisos.Hora(l), noVacio(d.Negocio, "TengoLibre"),
			noVacio(enMinuscula(d.Servicio), "turno"), noVacio(d.Profesional, "el equipo")},
		Botones: []whatsapp.Boton{{Payload: Confirmo + ":" + token}, {Payload: Cambiar + ":" + token}}}
}

// PlantillaDeRecordatorio arma el recordatorio de 24 horas antes, con el link al turno.
func PlantillaDeRecordatorio(para string, d datosTurno) whatsapp.Plantilla {
	l := d.Inicio.In(d.Zona)
	return whatsapp.Plantilla{Para: para, Nombre: PlantillaRecordatorio, Idioma: Idioma,
		Cuerpo: []string{noVacio(primerNombre(d.Cliente), "cliente"), noVacio(d.Negocio, "TengoLibre"),
			noVacio(d.Servicio, "tu turno"), avisos.Fecha(l), avisos.Hora(l)},
		Botones: []whatsapp.Boton{{Sufijo: d.Codigo}}}
}

// Respuestas libres (dentro de las 24 horas que abre el botón que tocó).
func textoConfirmado(sitio, codigo string) string {
	return "¡Listo! Queda confirmado. Si te surge algo, cambialo acá: " + avisos.LinkTurno(sitio, codigo)
}

func textoCambio(sitio, negocio, codigo string) string {
	return "Dale. Desde " + noVacio(negocio, "el negocio") + " te van a contactar para buscarte otro horario. " +
		"Si preferís, elegilo vos acá: " + avisos.LinkTurno(sitio, codigo)
}

func textoNoVigente(sitio, slug string) string {
	return "Ese turno ya no está vigente. Si querés sacar otro: " + avisos.LinkReserva(sitio, slug)
}

// cuando: «lunes 2 a las 10:30».
func cuando(d datosTurno) string {
	l := d.Inicio.In(d.Zona)
	return Dia(l) + " a las " + avisos.Hora(l)
}

func conQuien(d datosTurno) string {
	if d.Profesional == "" {
		return d.Servicio
	}
	return d.Servicio + " con " + d.Profesional
}
