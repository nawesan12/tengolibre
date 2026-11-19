package asistente

import (
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"

	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/disponibilidad"
)

// El intérprete: lo más común, en castellano rioplatense, sin IA. Es puro (no toca la base): recibe
// el texto y el catálogo del negocio y dice qué se pidió. Si no está seguro, no adivina: la pregunta
// va a la IA.

// Nombrado: un id con su nombre.
type Nombrado struct{ ID, Nombre string }

// ServicioCat: un servicio del negocio.
type ServicioCat struct {
	ID, Nombre string
	Duracion   int
}

// Catalogo: lo que el intérprete necesita saber del negocio.
type Catalogo struct {
	// Ahora en la zona del negocio.
	Ahora         time.Time
	Profesionales []Nombrado
	Servicios     []ServicioCat
	// Atencion: el horario semanal del negocio (la unión de los profesionales).
	Atencion []disponibilidad.Franja
}

func (c Catalogo) hoy() time.Time {
	return time.Date(c.Ahora.Year(), c.Ahora.Month(), c.Ahora.Day(), 0, 0, 0, 0, c.Ahora.Location())
}

// Tipos de pedido que entiende.
const (
	Agenda    = "agenda"
	Quien     = "quien"
	Libres    = "libres"
	Mover     = "mover"
	Cancelar  = "cancelar"
	Confirmar = "confirmar"
	Anotar    = "anotar"
)

// HoraDicha: una hora como la dijo la persona («a las 5», «17:30», «5 de la tarde»). Sufijo:
// manana | tarde | noche | 24 (ya viene en 24 h) | "".
type HoraDicha struct {
	H, M   int
	Sufijo string
	Dicha  bool
}

// Intencion: lo que se pidió.
type Intencion struct {
	Tipo string
	// Fecha del pedido (en mover, la del turno de origen, si la dijo). Cero: no la dijo.
	Fecha time.Time
	// Hora del pedido (en mover, la del turno de origen: «el de las 3»).
	Hora   HoraDicha
	Franja string // manana | tarde | noche
	// Cliente: el nombre como lo dijo, normalizado («juana», «pedro gomez»).
	Cliente       string
	ProfesionalID string
	ServicioID    string
	// Destino de mover.
	Destino     time.Time
	DestinoHora HoraDicha
}

var (
	quitarTildes = transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	rePuntoHora  = regexp.MustCompile(`(\d)\.(\d\d)`)
	reHoraPegada = regexp.MustCompile(`^(\d{1,2})(?::(\d{2}))?(hs|h|hrs)?$`)
	reFechaNum   = regexp.MustCompile(`^(\d{1,2})/(\d{1,2})(?:/(\d{2,4}))?$`)
)

// normalizar: minúsculas, sin tildes, sin signos (quedan los dígitos, «:» y «/»).
func normalizar(s string) string {
	s = rePuntoHora.ReplaceAllString(s, "$1:$2")
	s, _, _ = transform.String(quitarTildes, strings.ToLower(s))
	var b strings.Builder
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == ':' || r == '/' {
			b.WriteRune(r)
		} else {
			b.WriteByte(' ')
		}
	}
	ws := strings.Fields(b.String())
	for i, w := range ws {
		if a, ok := abreviaturas[w]; ok {
			ws[i] = a
		}
	}
	return strings.Join(ws, " ")
}

var (
	diasSemana = map[string]time.Weekday{"domingo": time.Sunday, "lunes": time.Monday, "martes": time.Tuesday,
		"miercoles": time.Wednesday, "jueves": time.Thursday, "viernes": time.Friday, "sabado": time.Saturday}
	meses = map[string]time.Month{"enero": 1, "febrero": 2, "marzo": 3, "abril": 4, "mayo": 5, "junio": 6, "julio": 7,
		"agosto": 8, "septiembre": 9, "setiembre": 9, "octubre": 10, "noviembre": 11, "diciembre": 12}
	numeros = map[string]int{"una": 1, "uno": 1, "dos": 2, "tres": 3, "cuatro": 4, "cinco": 5, "seis": 6, "siete": 7,
		"ocho": 8, "nueve": 9, "diez": 10, "once": 11, "doce": 12}

	verbosMover = []string{"pasa", "pasame", "pasala", "pasalo", "pasale", "pasar", "move", "movela", "movelo", "movele", "moveme",
		"mover", "cambia", "cambiala", "cambialo", "cambiale", "cambiar", "reprograma", "reprogramala", "reprogramalo",
		"reprogramale", "reprogramar", "corre", "correla", "correlo", "correle"}
	verbosCancelar  = []string{"cancela", "cancelala", "cancelalo", "cancelale", "cancelame", "cancelar", "anula", "anulala", "anulalo", "anular"}
	verbosConfirmar = []string{"confirma", "confirmala", "confirmalo", "confirmale", "confirmame", "confirmar"}
	verbosAnotar    = []string{"anota", "anotala", "anotalo", "anotame", "anotale", "anotar", "agendala", "agendalo",
		"agendame", "agendale", "agendar", "reserva", "reservala", "reservalo", "reservale", "reservame", "reservar", "carga",
		"cargala", "cargalo", "cargale", "cargame", "cargar"}

	// abreviaturas que se escriben en el teléfono.
	abreviaturas = map[string]string{"q": "que", "x": "por", "xq": "porque", "pq": "porque", "tmb": "tambien", "mnn": "manana", "mna": "manana"}

	// Palabras de relleno: pueden estar sin cambiar el pedido. Si queda alguna otra sin entender, el
	// intérprete no está seguro y la pregunta va a la IA.
	relleno = map[string]bool{"que": true, "tengo": true, "tenemos": true, "tiene": true, "tenes": true, "hay": true, "el": true,
		"la": true, "los": true, "las": true, "lo": true, "de": true, "del": true, "para": true, "turnos": true, "turno": true, "agenda": true,
		"a": true, "al": true, "y": true, "me": true, "mi": true, "mis": true, "en": true, "como": true, "viene": true, "vienen": true,
		"esta": true, "estoy": true, "cuantos": true, "cuantas": true, "quien": true, "quienes": true, "lugar": true, "libre": true,
		"libres": true, "hueco": true, "huecos": true, "horario": true, "horarios": true, "algun": true, "alguno": true, "alguna": true,
		"algo": true, "por": true, "favor": true, "porfa": true, "dale": true, "un": true, "una": true, "mostrame": true, "decime": true,
		"ver": true, "che": true, "hola": true, "con": true, "disponible": true, "disponibles": true, "queda": true, "quedan": true,
		"todavia": true, "se": true, "le": true, "puede": true, "podes": true, "porfavor": true, "gracias": true, "hoy": true,
		"es": true, "son": true, "otro": true, "anotado": true, "anotada": true, "anotados": true, "hora": true, "turnito": true,
		"agendado": true, "agendados": true, "pendiente": true, "pendientes": true, "cita": true, "citas": true, "tipo": true,
		"esa": true, "ese": true, "dia": true}
	paradasNombre = map[string]bool{"al": true, "para": true, "a": true, "el": true, "del": true, "de": true, "que": true,
		"hoy": true, "manana": true, "pasado": true, "las": true, "la": true, "y": true, "con": true, "turno": true, "por": true,
		"en": true, "porfa": true, "dale": true, "este": true, "esta": true, "proximo": true, "lo": true, "le": true, "mediodia": true}
)

// esc: los tokens del pedido y cuáles ya se entendieron.
type esc struct {
	t     []string
	usado []bool
}

func (e *esc) usar(i, n int) {
	for k := i; k < i+n && k < len(e.t); k++ {
		e.usado[k] = true
	}
}

func (e *esc) es(i int, ws ...string) bool {
	return i >= 0 && i < len(e.t) && slices.Contains(ws, e.t[i])
}

// sobrantes: lo que no se entendió ni es relleno.
func (e *esc) sobrantes() []string {
	var out []string
	for i, w := range e.t {
		if !e.usado[i] && !relleno[w] {
			out = append(out, w)
		}
	}
	return out
}

// Interpretar el pedido. ok: entendió con certeza.
func Interpretar(texto string, c Catalogo) (Intencion, bool) {
	n := normalizar(texto)
	e := &esc{t: strings.Fields(n)}
	e.usado = make([]bool, len(e.t))
	if len(e.t) == 0 || len(e.t) > 16 {
		return Intencion{}, false
	}
	var in Intencion
	verbo := -1
	for i := 0; i < len(e.t) && i < 3 && verbo < 0; i++ {
		switch {
		case slices.Contains(verbosMover, e.t[i]) && !e.es(i-1, "que"):
			in.Tipo = Mover
		case slices.Contains(verbosCancelar, e.t[i]):
			in.Tipo = Cancelar
		case slices.Contains(verbosConfirmar, e.t[i]):
			in.Tipo = Confirmar
		// «agenda» es también el sustantivo («agenda del jueves»): es verbo sólo con «a» (agendá a Marta).
		case slices.Contains(verbosAnotar, e.t[i]) || e.t[i] == "agenda" && e.es(i+1, "a"):
			in.Tipo = Anotar
		default:
			continue
		}
		verbo = i
		e.usar(i, 1)
	}
	// Los nombres del catálogo (servicios y profesionales) se reconocen en cualquier lugar.
	in.ServicioID = buscarNombre(e, servicios(c), false)
	if in.Tipo == "" {
		in.ProfesionalID = buscarNombre(e, c.Profesionales, true)
	}
	switch in.Tipo {
	case Mover:
		return interpretarMover(e, verbo, c, in)
	case Anotar:
		// «anotá a Lucía mañana a las 11 para corte»: cliente, día, hora y (si lo dijo) el servicio.
		in.Cliente = nombreCliente(e, verbo+1)
		in.Fecha, _ = fecha(e, 0, len(e.t), c)
		in.Hora = hora(e, 0, len(e.t))
		if in.Cliente == "" || in.Fecha.IsZero() || !in.Hora.Dicha {
			return in, false
		}
		in.ProfesionalID = buscarNombre(e, c.Profesionales, true)
		return in, len(e.sobrantes()) == 0
	case Cancelar, Confirmar:
		in.Cliente = nombreCliente(e, verbo+1)
		f, _ := fecha(e, 0, len(e.t), c)
		in.Fecha = f
		in.Hora = hora(e, 0, len(e.t))
		if in.Cliente == "" && !in.Hora.Dicha {
			return in, false
		}
		return in, len(e.sobrantes()) == 0
	}
	f, _ := fecha(e, 0, len(e.t), c)
	in.Fecha = f
	in.Hora = hora(e, 0, len(e.t))
	in.Franja = franja(e)
	switch {
	case e.tiene("quien", "quienes"):
		if !in.Hora.Dicha {
			return in, false
		}
		in.Tipo = Quien
	case e.tiene("lugar", "libre", "libres", "hueco", "huecos", "disponible", "disponibles"):
		in.Tipo = Libres
	case e.tiene("tengo", "tenemos", "tiene", "agenda", "turnos", "hay", "vienen", "viene"):
		if in.Hora.Dicha {
			// «¿qué tengo a las 5?» es «¿quién viene a las 5?».
			in.Tipo = Quien
		} else {
			in.Tipo = Agenda
		}
	default:
		return in, false
	}
	return in, len(e.sobrantes()) == 0
}

func (e *esc) tiene(ws ...string) bool {
	for _, w := range e.t {
		if slices.Contains(ws, w) {
			return true
		}
	}
	return false
}

func servicios(c Catalogo) []Nombrado {
	out := make([]Nombrado, len(c.Servicios))
	for i, s := range c.Servicios {
		out[i] = Nombrado{s.ID, s.Nombre}
	}
	return out
}

// buscarNombre: el nombre más largo del catálogo que aparece entero en el pedido (o, con
// primerNombre, sólo su primera palabra si es única).
func buscarNombre(e *esc, ns []Nombrado, primerNombre bool) string {
	mejor, largo, pos := "", 0, -1
	for _, x := range ns {
		ws := strings.Fields(normalizar(x.Nombre))
		if len(ws) == 0 {
			continue
		}
		if i := e.buscar(ws); i >= 0 && len(ws) > largo {
			mejor, largo, pos = x.ID, len(ws), i
		}
	}
	if mejor == "" && primerNombre {
		// Por la primera palabra, si un solo profesional la tiene.
		var ids []string
		var at int
		for _, x := range ns {
			ws := strings.Fields(normalizar(x.Nombre))
			if len(ws) > 0 {
				if i := e.buscar(ws[:1]); i >= 0 {
					ids, at = append(ids, x.ID), i
				}
			}
		}
		if len(ids) == 1 {
			mejor, largo, pos = ids[0], 1, at
		}
	}
	if pos >= 0 {
		e.usar(pos, largo)
	}
	return mejor
}

func (e *esc) buscar(ws []string) int {
	for i := 0; i+len(ws) <= len(e.t); i++ {
		if !e.usado[i] && slices.Equal(e.t[i:i+len(ws)], ws) {
			return i
		}
	}
	return -1
}

// nombreCliente: las palabras después del verbo («a Juana», «el turno de Pedro Gómez»), hasta una
// palabra de corte. Hasta tres palabras.
func nombreCliente(e *esc, desde int) string {
	i := desde
	for i < len(e.t) {
		// «el de las 3», «a las 5»: no hay nombre.
		if e.es(i, "de", "a", "el", "la") && e.es(i+1, "las", "la") {
			break
		}
		if !e.es(i, "a", "al", "el", "la", "turno", "de", "del", "lo", "le", "me", "porfa") {
			break
		}
		i++
	}
	var ws []string
	for k := i; k < len(e.t) && len(ws) < 3; k++ {
		w := e.t[k]
		if e.usado[k] || paradasNombre[w] || esNumero(w) || diasSemana[w] != 0 || w == "domingo" || relleno[w] && len(ws) == 0 {
			break
		}
		ws = append(ws, w)
		e.usar(k, 1)
	}
	return strings.Join(ws, " ")
}

func esNumero(w string) bool {
	_, ok := numeros[w]
	return ok || reHoraPegada.MatchString(w) || reFechaNum.MatchString(w)
}

// interpretarMover: «pasá a Juana al jueves a las 5», «mové el de las 3 al viernes», «cambiá a
// Pedro de las 3 a las 5». Lo que va después de «al» / «para» es el destino.
func interpretarMover(e *esc, verbo int, c Catalogo, in Intencion) (Intencion, bool) {
	in.Cliente = nombreCliente(e, verbo+1)
	corte := -1
	for i := verbo + 1; i < len(e.t); i++ {
		if !e.usado[i] && (e.es(i, "al", "para") && !e.es(i+1, "mediodia")) {
			corte = i
			break
		}
	}
	if corte >= 0 {
		e.usar(corte, 1)
		in.Fecha, _ = fecha(e, verbo+1, corte, c)
		in.Hora = hora(e, verbo+1, corte)
		in.Destino, _ = fecha(e, corte+1, len(e.t), c)
		in.DestinoHora = hora(e, corte+1, len(e.t))
	} else {
		// Sin «al»: con dos horas («de las 3 a las 5»), la primera es la de origen.
		in.Fecha, _ = fecha(e, verbo+1, len(e.t), c)
		h1 := hora(e, verbo+1, len(e.t))
		h2 := hora(e, verbo+1, len(e.t))
		if h2.Dicha {
			in.Hora, in.DestinoHora = h1, h2
		} else {
			in.DestinoHora = h1
		}
		// Una sola fecha, sin «al»: es el destino («cambiá a Juana el jueves a las 5»).
		in.Destino, in.Fecha = in.Fecha, time.Time{}
	}
	if in.Cliente == "" && !in.Hora.Dicha {
		return in, false
	}
	if in.Destino.IsZero() && !in.DestinoHora.Dicha {
		return in, false
	}
	return in, len(e.sobrantes()) == 0
}

// fecha: hoy, mañana, pasado mañana, un día de la semana («el jueves», «el jueves que viene»), «el
// 20», «20 de noviembre» o «20/11», entre los tokens [desde, hasta).
func fecha(e *esc, desde, hasta int, c Catalogo) (time.Time, bool) {
	hoy := c.hoy()
	for i := desde; i < hasta; i++ {
		if e.usado[i] {
			continue
		}
		w := e.t[i]
		switch {
		case w == "hoy":
			e.usar(i, 1)
			return hoy, true
		case w == "pasado" && e.es(i+1, "manana"):
			e.usar(i, 2)
			return hoy.AddDate(0, 0, 2), true
		// «mañana» es el día salvo en «a la mañana» o «a las 9 de la mañana» («el de las 4 de
		// mañana» es el de mañana).
		case w == "manana" && !e.es(i-1, "la"):
			e.usar(i, 1)
			return hoy.AddDate(0, 0, 1), true
		case w == "domingo" || diasSemana[w] != 0:
			d := diasSemana[w]
			off := (int(d) - int(hoy.Weekday()) + 7) % 7
			e.usar(i, 1)
			if e.es(i-1, "proximo") {
				e.usar(i-1, 1)
				if off == 0 {
					off = 7
				}
			}
			if e.es(i-1, "este") {
				e.usar(i-1, 1)
			}
			if e.es(i+1, "que") && e.es(i+2, "viene") {
				e.usar(i+1, 2)
				if off == 0 {
					off = 7
				}
			}
			// «el jueves 20»: el número es del mismo día.
			if i+1 < hasta {
				if n, err := strconv.Atoi(e.t[i+1]); err == nil && n == hoy.AddDate(0, 0, off).Day() {
					e.usar(i+1, 1)
				}
			}
			return hoy.AddDate(0, 0, off), true
		case reFechaNum.MatchString(w):
			m := reFechaNum.FindStringSubmatch(w)
			d, _ := strconv.Atoi(m[1])
			mes, _ := strconv.Atoi(m[2])
			anio := hoy.Year()
			if m[3] != "" {
				anio, _ = strconv.Atoi(m[3])
				if anio < 100 {
					anio += 2000
				}
			} else if time.Month(mes) < hoy.Month() {
				anio++
			}
			if f, ok := armarFecha(anio, time.Month(mes), d, hoy.Location()); ok {
				e.usar(i, 1)
				return f, true
			}
		case e.es(i+1, "de") && i+2 < hasta && meses[e.t[i+2]] != 0:
			d, err := strconv.Atoi(w)
			if err != nil {
				continue
			}
			mes := meses[e.t[i+2]]
			anio := hoy.Year()
			if mes < hoy.Month() {
				anio++
			}
			if f, ok := armarFecha(anio, mes, d, hoy.Location()); ok {
				e.usar(i, 3)
				return f, true
			}
		case w == "el" && i+1 < hasta && !e.usado[i+1]:
			// «el 20»: un número suelto que no es una hora.
			d, err := strconv.Atoi(e.t[i+1])
			if err != nil || d < 1 || d > 31 || e.es(i+2, "hs", "h", "horas", "y") {
				continue
			}
			anio, mes := hoy.Year(), hoy.Month()
			if d < hoy.Day() {
				mes++
				if mes > 12 {
					mes, anio = 1, anio+1
				}
			}
			if f, ok := armarFecha(anio, mes, d, hoy.Location()); ok {
				e.usar(i, 2)
				return f, true
			}
		}
	}
	return time.Time{}, false
}

func armarFecha(anio int, mes time.Month, dia int, z *time.Location) (time.Time, bool) {
	if mes < 1 || mes > 12 || dia < 1 || dia > 31 {
		return time.Time{}, false
	}
	f := time.Date(anio, mes, dia, 0, 0, 0, 0, z)
	return f, f.Day() == dia
}

// hora: la primera hora entre [desde, hasta) que no se haya usado.
func hora(e *esc, desde, hasta int) HoraDicha {
	for i := desde; i < hasta; i++ {
		if e.usado[i] {
			continue
		}
		w := e.t[i]
		if w == "mediodia" {
			e.usar(i, 1)
			if e.es(i-1, "al") {
				e.usar(i-1, 1)
			}
			return HoraDicha{H: 12, Sufijo: "24", Dicha: true}
		}
		// Después de «a las», «las», «a la» o pegada con hs / dos puntos.
		previa := e.es(i-1, "las", "la") && !e.usado[i-1]
		h, m, suf, ok := numeroHora(w)
		// «el jueves 5 de la tarde»: sin «a las», la hora la marca «de la tarde».
		conFranja := e.es(i+1, "de") && (e.es(i+2, "la") && e.es(i+3, "manana", "tarde", "noche") || e.es(i+2, "tarde", "noche"))
		if !ok || (!previa && suf == "" && !conFranja && !strings.Contains(w, ":") && !e.es(i+1, "hs", "h", "horas", "hrs")) {
			continue
		}
		usados := 1
		if previa {
			e.usar(i-1, 1)
			if e.es(i-2, "a") {
				e.usar(i-2, 1)
			}
		}
		k := i + 1
		switch {
		case e.es(k, "y") && e.es(k+1, "media"):
			m, usados, k = 30, usados+2, k+2
		case e.es(k, "y") && e.es(k+1, "cuarto"):
			m, usados, k = 15, usados+2, k+2
		case e.es(k, "menos") && e.es(k+1, "cuarto"):
			h, m, usados, k = h-1, 45, usados+2, k+2
		}
		if e.es(k, "hs", "h", "horas", "hrs") {
			usados, k = usados+1, k+1
		}
		if e.es(k, "de") && e.es(k+1, "la") && e.es(k+2, "manana", "tarde", "noche") {
			suf, usados = e.t[k+2], usados+3
		} else if e.es(k, "de") && e.es(k+1, "tarde", "noche") {
			suf, usados = e.t[k+1], usados+2
		}
		if h < 0 || h > 23 || m > 59 {
			continue
		}
		e.usar(i, usados)
		return HoraDicha{H: h, M: m, Sufijo: suf, Dicha: true}
	}
	return HoraDicha{}
}

func numeroHora(w string) (h, m int, suf string, ok bool) {
	if n, esPalabra := numeros[w]; esPalabra {
		return n, 0, "", true
	}
	x := reHoraPegada.FindStringSubmatch(w)
	if x == nil {
		return 0, 0, "", false
	}
	h, _ = strconv.Atoi(x[1])
	if x[2] != "" {
		m, _ = strconv.Atoi(x[2])
	}
	// «17hs» o «17:30» ya vienen en 24 h; «5hs» no dice si es de mañana o de tarde.
	if h > 12 || h == 0 {
		suf = "24"
	}
	return h, m, suf, true
}

// franja: «a la mañana», «por la tarde», «a la noche».
func franja(e *esc) string {
	for i, w := range e.t {
		if e.usado[i] || (w != "manana" && w != "tarde" && w != "noche") {
			continue
		}
		if e.es(i-1, "la") && e.es(i-2, "a", "por", "de") {
			e.usar(i-2, 3)
			return w
		}
		if e.es(i-1, "de") && w != "manana" {
			e.usar(i-1, 2)
			return w
		}
	}
	return ""
}

// EnFranja: la hora (en minutos desde las 0) cae en la franja.
func EnFranja(f string, minutos int) bool {
	switch f {
	case "manana":
		return minutos < 13*60
	case "tarde":
		return minutos >= 13*60 && minutos < 20*60
	case "noche":
		return minutos >= 19*60
	}
	return true
}

// Resolver una hora dicha en un día: «a las 5» es 17 h si el negocio atiende a las 17 y no a las 5.
func (c Catalogo) Resolver(h HoraDicha, dia time.Time) time.Time {
	hh := h.H
	switch {
	case h.Sufijo == "24":
	case h.Sufijo == "tarde" || h.Sufijo == "noche":
		if hh < 12 {
			hh += 12
		}
	case h.Sufijo == "manana":
		if hh == 12 {
			hh = 0
		}
	case hh >= 1 && hh <= 11:
		a, b := c.atiende(dia, hh*60+h.M), c.atiende(dia, (hh+12)*60+h.M)
		switch {
		case b && !a:
			hh += 12
		case a && !b:
		case hh <= 7:
			// Sin pistas (o con las dos posibles): de 1 a 7 es de tarde, como se dice.
			hh += 12
		}
	}
	return time.Date(dia.Year(), dia.Month(), dia.Day(), hh, h.M, 0, 0, dia.Location())
}

func (c Catalogo) atiende(dia time.Time, minutos int) bool {
	for _, f := range c.Atencion {
		if f.Dia != int(dia.Weekday()) {
			continue
		}
		d, h := minutosDe(f.Desde), minutosDe(f.Hasta)
		if minutos >= d && minutos < h {
			return true
		}
	}
	return false
}

func minutosDe(hhmm string) int {
	p := strings.SplitN(hhmm, ":", 2)
	if len(p) != 2 {
		return -1
	}
	h, _ := strconv.Atoi(p[0])
	m, _ := strconv.Atoi(p[1])
	return h*60 + m
}

// coincideNombre: todas las palabras dichas son el principio de alguna palabra del nombre («juana»
// con «Juana Pérez», «juana p» también).
func coincideNombre(dicho, nombre string) bool {
	ws := strings.Fields(normalizar(nombre))
	for _, d := range strings.Fields(dicho) {
		ok := false
		for _, w := range ws {
			if strings.HasPrefix(w, d) {
				ok = true
				break
			}
		}
		if !ok {
			return false
		}
	}
	return dicho != ""
}
