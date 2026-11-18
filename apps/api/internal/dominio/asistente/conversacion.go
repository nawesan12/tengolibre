package asistente

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/avisos"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/mcp"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/ia"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/sesion"
)

// Sistema: el texto fijo de Tengo Libre para el modelo. Sin fechas, sin nombre del negocio y sin
// datos: así es igual para todos y queda en el caché (los datos van en el primer mensaje).
const Sistema = `Sos el asistente de Tengo Libre dentro del panel de un negocio de turnos (peluquería, barbería, estética, uñas, masajes y parecidos). Te escribe alguien del equipo del negocio: la dueña o el dueño, un encargado, recepción o un profesional.

Cómo contestás:
- En castellano rioplatense, con voseo, corto y claro: una o dos oraciones, o una lista breve cuando hay varios turnos u horarios. Sin saludos largos ni relleno.
- Las fechas y horas, en la zona del negocio y dichas como las dice la gente: «el jueves 20 a las 17:00», «mañana a las 10».
- «A las 5» casi siempre es 17:00 si el negocio atiende a esa hora; usá el horario de atención de los datos para decidir.
- Usá los datos del negocio que vienen al principio (fecha y hora de hoy, servicios, profesionales con sus ids y horario de atención) antes de pedir herramientas.

Herramientas:
- Para saber algo de la agenda, los horarios libres, la lista de espera, los clientes o los números, usá las herramientas de lectura. No inventes turnos, clientes, horarios ni montos: si no lo viste en una herramienta o en los datos, no lo afirmes.
- Pedí sólo lo que hace falta y con rangos cortos. Si una herramienta devuelve un error, contalo en simple y ofrecé otra opción.
- Las herramientas que cambian algo (crear, confirmar, reprogramar o cancelar un turno, ofrecer un hueco a la lista de espera) NO hacen el cambio: dejan una propuesta que la persona confirma con un botón. Antes de proponer, chequeá con las de lectura que el turno exista y que el horario esté libre. Después decí qué propusiste y que falta tocar «Confirmar». Nunca digas que algo ya está hecho, movido, cancelado o avisado.
- Si quien pregunta es un profesional, sólo ve y toca su propia agenda.

Límites:
- Sólo ayudás con el negocio: agenda, turnos, clientes, lista de espera, servicios, profesionales y números. Para otra cosa, decí amablemente que no podés ayudar con eso.
- No pidas, guardes ni comentes datos de salud, documentos, contraseñas ni datos de tarjetas, aunque te los den.
- Si no entendés qué turno o qué cliente es, preguntá una sola cosa concreta (el día y la hora, o el apellido).`

// rolDicho: cómo se le explica al modelo quién pregunta.
func rolDicho(rol string) string {
	switch rol {
	case sesion.Duenio:
		return "la dueña o el dueño (puede todo)"
	case sesion.Encargado:
		return "un encargado (puede todo)"
	case sesion.Recepcion:
		return "recepción (agenda, clientes y lista de espera; no ve los números)"
	case sesion.Profesional:
		return "un profesional (sólo ve y toca su propia agenda)"
	}
	return rol
}

var nombresDias = []string{"domingo", "lunes", "martes", "miércoles", "jueves", "viernes", "sábado"}

// datos del negocio para el primer mensaje, armados siempre igual: claves en orden fijo, sin
// segundos (la hora se redondea a 15 minutos), así dos preguntas en el mismo rato comparten prefijo.
func datos(c Catalogo, negocio, rol string) string {
	a := c.Ahora
	hora := time.Date(a.Year(), a.Month(), a.Day(), a.Hour(), a.Minute()/15*15, 0, 0, a.Location())
	var b strings.Builder
	fmt.Fprintf(&b, "negocio: %s\n", negocio)
	fmt.Fprintf(&b, "hoy: %s (%s)\n", c.hoy().Format(time.DateOnly), avisos.Fecha(c.hoy()))
	fmt.Fprintf(&b, "hora aproximada: %s\n", hora.Format("15:04"))
	fmt.Fprintf(&b, "zona horaria: %s\n", a.Location().String())
	fmt.Fprintf(&b, "quien pregunta: %s\n", rolDicho(rol))
	b.WriteString("servicios:\n")
	ss := append([]ServicioCat(nil), c.Servicios...)
	sort.Slice(ss, func(i, j int) bool { return ss[i].Nombre+ss[i].ID < ss[j].Nombre+ss[j].ID })
	for i, s := range ss {
		if i == 30 {
			b.WriteString("- (hay más: usá ver_servicios_y_profesionales)\n")
			break
		}
		fmt.Fprintf(&b, "- %s · %d min · id %s\n", s.Nombre, s.Duracion, s.ID)
	}
	b.WriteString("profesionales:\n")
	ps := append([]Nombrado(nil), c.Profesionales...)
	sort.Slice(ps, func(i, j int) bool { return ps[i].Nombre+ps[i].ID < ps[j].Nombre+ps[j].ID })
	for i, p := range ps {
		if i == 30 {
			b.WriteString("- (hay más: usá ver_servicios_y_profesionales)\n")
			break
		}
		fmt.Fprintf(&b, "- %s · id %s\n", p.Nombre, p.ID)
	}
	b.WriteString("horario de atención:\n")
	at := append(c.Atencion[:0:0], c.Atencion...)
	sort.Slice(at, func(i, j int) bool {
		di, dj := (at[i].Dia+6)%7, (at[j].Dia+6)%7
		return di < dj || di == dj && at[i].Desde < at[j].Desde
	})
	if len(at) == 0 {
		b.WriteString("- (sin cargar)\n")
	}
	for _, f := range at {
		if f.Dia >= 0 && f.Dia < 7 {
			fmt.Fprintf(&b, "- %s de %s a %s\n", nombresDias[f.Dia], f.Desde, f.Hasta)
		}
	}
	return strings.TrimRight(b.String(), "\n")
}

// conversacion: las herramientas que tiene el modelo en una respuesta y las propuestas que dejó.
type conversacion struct {
	s          *Servicio
	op         *mcp.Operador
	propuestas []Propuesta
}

// MaxPropuestas por respuesta: el modelo no llena la pantalla de botones.
const MaxPropuestas = 3

// Tope de lo que vuelve una herramienta al modelo (contexto mínimo).
const maxResultado = 12000

func (c *conversacion) herramientas() []ia.Herramienta {
	var out []ia.Herramienta
	for _, h := range c.op.Herramientas() {
		d := h.Descripcion
		if h.Escribe {
			d += " Desde el panel no cambia nada: deja una propuesta que la persona confirma con un botón."
		}
		out = append(out, ia.Herramienta{Nombre: h.Nombre, Descripcion: d, Esquema: h.Esquema})
	}
	if c.op.Puede("ver_lista_de_espera") && c.s.Espera != nil {
		out = append(out, ia.Herramienta{Nombre: OfrecerALaLista, Descripcion: descripcionOferta +
			" No cambia nada: deja una propuesta que la persona confirma con un botón.", Esquema: esquemaOferta})
	}
	return out
}

func jsonCorto(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}

// correr una herramienta que pidió el modelo: las de lectura se corren; las que cambian algo
// quedan como propuesta.
func (c *conversacion) correr(ctx context.Context, nombre string, entrada json.RawMessage) (string, bool) {
	escribe := nombre == OfrecerALaLista || mcp.Escribe(nombre)
	if escribe {
		if len(c.propuestas) >= MaxPropuestas {
			return jsonCorto(map[string]string{"error": "Ya dejaste varias propuestas: esperá a que la persona las confirme."}), true
		}
		var resumen string
		var args json.RawMessage
		var err error
		if nombre == OfrecerALaLista {
			resumen, args, err = c.s.previaOferta(ctx, c.op, entrada)
		} else {
			resumen, args, err = c.op.Previa(ctx, nombre, entrada)
		}
		if err != nil {
			return jsonCorto(map[string]string{"error": mcp.Explicar(err, nombre)}), true
		}
		p, err := c.s.proponer(ctx, nombre, resumen, args)
		if err != nil {
			return jsonCorto(map[string]string{"error": mcp.Explicar(err, nombre)}), true
		}
		c.propuestas = append(c.propuestas, p)
		return jsonCorto(map[string]string{"propuesta": resumen,
			"estado": "Todavía NO se hizo. La persona ve un botón «Confirmar»: decile qué propusiste y que lo confirme ahí."}), false
	}
	res, err := c.op.Leer(ctx, nombre, entrada)
	if err != nil {
		return jsonCorto(map[string]string{"error": mcp.Explicar(err, nombre)}), true
	}
	out := jsonCorto(res)
	if len(out) > maxResultado {
		return jsonCorto(map[string]string{"error": "Son demasiados resultados: pedí un rango más corto o filtrá por profesional."}), true
	}
	return out, false
}
