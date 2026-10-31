// Package disponibilidad es el motor de horarios libres: puro, sin base ni reloj (calca
// availability-engine.ts de Jiku). Es la regla central del producto; los tests lo cubren con datos
// armados a mano.
package disponibilidad

import (
	"sort"
	"time"

	// La zona de Argentina viaja en el binario: el VPS puede no tener tzdata.
	_ "time/tzdata"
)

// ZonaPorDefecto de los negocios (UTC−3, sin horario de verano).
var ZonaPorDefecto = func() *time.Location {
	z, err := time.LoadLocation("America/Argentina/Buenos_Aires")
	if err != nil {
		return time.FixedZone("ART", -3*3600)
	}
	return z
}()

// Franja del horario semanal. Dia 0 = domingo; Desde y Hasta "HH:MM" en la zona del negocio.
type Franja struct {
	Dia   int    `json:"dia"`
	Desde string `json:"desde"`
	Hasta string `json:"hasta"`
}

// Bloqueo de una fecha. Sin Desde/Hasta: el día entero.
type Bloqueo struct {
	ID     string  `json:"id,omitempty"`
	Fecha  string  `json:"fecha"`
	Desde  *string `json:"desde"`
	Hasta  *string `json:"hasta"`
	Motivo string  `json:"motivo,omitempty"`
}

// Semanal bloquea un horario puntual todas las semanas (el almuerzo, p. ej.).
type Semanal struct {
	Dia  int    `json:"dia"`
	Hora string `json:"hora"`
}

// Excepcion de un horario en una fecha: "bloqueado", o "libre" (gana sobre un Semanal).
type Excepcion struct {
	Fecha string `json:"fecha"`
	Hora  string `json:"hora"`
	Tipo  string `json:"tipo"`
}

// Ocupado es un turno que puede tapar un horario.
type Ocupado struct {
	Inicio  time.Time  `json:"inicio"`
	Fin     time.Time  `json:"fin"`
	Estado  string     `json:"estado"`
	VenceEn *time.Time `json:"venceEn"`
}

// Ocupa: pendiente y confirmado sí; pendiente de pago sólo mientras no venció (así no hace falta un
// cron que lo libere); cancelado, atendido y ausente no.
func (o Ocupado) Ocupa(ahora time.Time) bool {
	switch o.Estado {
	case "pendiente", "confirmado":
		return true
	case "pendiente_pago":
		return o.VenceEn == nil || o.VenceEn.After(ahora)
	}
	return false
}

// Profesional es todo lo de un profesional que mira el motor.
type Profesional struct {
	ID          string      `json:"id"`
	Horario     []Franja    `json:"horario"`
	Bloqueos    []Bloqueo   `json:"bloqueos"`
	Semanales   []Semanal   `json:"semanales"`
	Excepciones []Excepcion `json:"excepciones"`
	Turnos      []Ocupado   `json:"turnos"`
}

// Reglas del negocio (tabla ajustes), en minutos salvo DiasVista.
type Reglas struct {
	Intervalo       int `json:"intervalo"`
	AnticipacionMin int `json:"anticipacionMin"`
	DiasVista       int `json:"diasVista"`
	MargenMin       int `json:"margenMin"`
}

// Horario es un inicio posible del día. Libre: se puede reservar.
type Horario struct {
	Hora   string    `json:"hora"`
	Inicio time.Time `json:"inicio"`
	Libre  bool      `json:"libre"`
}

// Consulta: un profesional, una fecha ("YYYY-MM-DD") y un servicio de Duracion minutos.
type Consulta struct {
	Profesional Profesional
	Fecha       string
	Duracion    int
	Reglas      Reglas
	Ahora       time.Time
	Zona        *time.Location
}

// minutos de "HH:MM" (o "HH:MM:SS", como lo devuelve Postgres) desde la medianoche.
func minutos(s string) (int, bool) {
	if len(s) == 8 && s[5] == ':' {
		s = s[:5]
	}
	if len(s) != 5 || s[2] != ':' {
		return 0, false
	}
	for _, c := range []byte{s[0], s[1], s[3], s[4]} {
		if c < '0' || c > '9' {
			return 0, false
		}
	}
	h, m := int(s[0]-'0')*10+int(s[1]-'0'), int(s[3]-'0')*10+int(s[4]-'0')
	if h > 24 || m > 59 || (h == 24 && m != 0) {
		return 0, false
	}
	return h*60 + m, true
}

func hhmm(m int) string {
	return string([]byte{byte('0' + m/600), byte('0' + m/60%10), ':', byte('0' + m%60/10), byte('0' + m%10)})
}

func (c Consulta) zona() *time.Location {
	if c.Zona != nil {
		return c.Zona
	}
	return ZonaPorDefecto
}

// fecha de la consulta, si está en la vista: de hoy a DiasVista−1 días adelante.
func (c Consulta) fecha() (time.Time, bool) {
	z := c.zona()
	dia, err := time.ParseInLocation("2006-01-02", c.Fecha, z)
	if err != nil || c.Duracion <= 0 {
		return time.Time{}, false
	}
	a := c.Ahora.In(z)
	hoy := time.Date(a.Year(), a.Month(), a.Day(), 0, 0, 0, 0, z)
	vista := c.Reglas.DiasVista
	if vista <= 0 {
		vista = 30
	}
	if dia.Before(hoy) || !dia.Before(hoy.AddDate(0, 0, vista)) {
		return time.Time{}, false
	}
	return dia, true
}

// Dia devuelve los horarios del día del profesional, libres u ocupados, ordenados.
func Dia(c Consulta) []Horario {
	out := []Horario{}
	dia, ok := c.fecha()
	if !ok {
		return out
	}
	p := c.Profesional
	semana := int(dia.Weekday())
	intervalo := c.Reglas.Intervalo
	if intervalo <= 0 {
		intervalo = 30
	}
	// Un bloqueo del día entero tapa todo; los parciales se miran horario por horario.
	var parciales [][2]int
	for _, b := range p.Bloqueos {
		if b.Fecha != c.Fecha {
			continue
		}
		if b.Desde == nil || b.Hasta == nil {
			return out
		}
		d, ok1 := minutos(*b.Desde)
		h, ok2 := minutos(*b.Hasta)
		if ok1 && ok2 {
			parciales = append(parciales, [2]int{d, h})
		}
	}
	semanales := map[int]bool{}
	for _, s := range p.Semanales {
		if m, ok := minutos(s.Hora); ok && s.Dia == semana {
			semanales[m] = true
		}
	}
	excepciones := map[int]string{}
	for _, e := range p.Excepciones {
		if m, ok := minutos(e.Hora); ok && e.Fecha == c.Fecha {
			excepciones[m] = e.Tipo
		}
	}
	margen := time.Duration(c.Reglas.MargenMin) * time.Minute
	minimo := c.Ahora.Add(time.Duration(c.Reglas.AnticipacionMin) * time.Minute)
	total := c.Duracion + c.Reglas.MargenMin
	vistos := map[int]bool{}
	for _, f := range p.Horario {
		if f.Dia != semana {
			continue
		}
		desde, ok1 := minutos(f.Desde)
		hasta, ok2 := minutos(f.Hasta)
		if !ok1 || !ok2 {
			continue
		}
		for m := desde; m+total <= hasta; m += intervalo {
			if vistos[m] {
				continue
			}
			vistos[m] = true
			ini := time.Date(dia.Year(), dia.Month(), dia.Day(), 0, m, 0, 0, dia.Location())
			fin := ini.Add(time.Duration(total) * time.Minute)
			h := Horario{Hora: hhmm(m), Inicio: ini, Libre: true}
			switch {
			case ini.Before(minimo):
				h.Libre = false
			case pisaParcial(parciales, m, m+total):
				h.Libre = false
			case excepciones[m] == "bloqueado":
				h.Libre = false
			case excepciones[m] != "libre" && semanales[m]:
				h.Libre = false
			default:
				// Un turno ocupa [inicio, fin + margen): se superponen si cada uno empieza antes de
				// que termine el otro.
				for _, o := range p.Turnos {
					if o.Ocupa(c.Ahora) && ini.Before(o.Fin.Add(margen)) && fin.After(o.Inicio) {
						h.Libre = false
						break
					}
				}
			}
			out = append(out, h)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Inicio.Before(out[j].Inicio) })
	return out
}

func pisaParcial(parciales [][2]int, ini, fin int) bool {
	for _, b := range parciales {
		if ini < b[1] && fin > b[0] {
			return true
		}
	}
	return false
}

// Libre dice si inicio es un horario libre del profesional (para reservar: se recalcula adentro de
// la transacción, con el profesional bloqueado).
func Libre(c Consulta, inicio time.Time) bool {
	for _, h := range Dia(c) {
		if h.Inicio.Equal(inicio) {
			return h.Libre
		}
	}
	return false
}
