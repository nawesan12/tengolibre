package asistente

import (
	"testing"
	"time"

	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/disponibilidad"
)

var zonaPrueba = func() *time.Location {
	z, _ := time.LoadLocation("America/Argentina/Buenos_Aires")
	return z
}()

// Miércoles 18 de noviembre de 2026, 10:20. Atiende de lunes a sábado de 9 a 20.
func catalogoDePrueba() Catalogo {
	var at []disponibilidad.Franja
	for d := 1; d <= 6; d++ {
		at = append(at, disponibilidad.Franja{Dia: d, Desde: "09:00", Hasta: "20:00"})
	}
	return Catalogo{
		Ahora:         time.Date(2026, 11, 18, 10, 20, 0, 0, zonaPrueba),
		Profesionales: []Nombrado{{"p1", "Martín López"}, {"p2", "Sofía"}},
		Servicios:     []ServicioCat{{"s1", "Corte", 30}, {"s2", "Color", 90}, {"s3", "Corte y barba", 45}},
		Atencion:      at,
	}
}

func dia(m time.Month, d int) time.Time { return time.Date(2026, m, d, 0, 0, 0, 0, zonaPrueba) }

func TestInterpretarConsultas(t *testing.T) {
	c := catalogoDePrueba()
	casos := []struct {
		texto  string
		tipo   string
		fecha  time.Time
		hora   int // en 24 h, resuelta con el horario (-1: sin hora)
		franja string
		prof   string
		srv    string
	}{
		{"¿Qué tengo hoy?", Agenda, dia(11, 18), -1, "", "", ""},
		{"que tengo mañana", Agenda, dia(11, 19), -1, "", "", ""},
		{"¿Qué tengo el jueves?", Agenda, dia(11, 19), -1, "", "", ""},
		{"¿qué tengo pasado mañana?", Agenda, dia(11, 20), -1, "", "", ""},
		{"¿Qué tengo el miércoles que viene?", Agenda, dia(11, 25), -1, "", "", ""},
		{"¿qué hay el lunes?", Agenda, dia(11, 23), -1, "", "", ""},
		{"turnos del 20/11", Agenda, dia(11, 20), -1, "", "", ""},
		{"¿qué tengo el 3?", Agenda, dia(12, 3), -1, "", "", ""},
		{"agenda del 2 de diciembre", Agenda, dia(12, 2), -1, "", "", ""},
		{"¿qué tiene Martín mañana?", Agenda, dia(11, 19), -1, "", "p1", ""},
		{"¿Quién viene a las 5?", Quien, time.Time{}, 17, "", "", ""},
		{"quien viene a las 10", Quien, time.Time{}, 10, "", "", ""},
		{"¿quién viene mañana a las 9 y media?", Quien, dia(11, 19), 9, "", "", ""},
		{"¿qué tengo a las 17:30?", Quien, time.Time{}, 17, "", "", ""},
		{"quien viene a las 8 de la noche", Quien, time.Time{}, 20, "", "", ""},
		{"¿Hay lugar el viernes a la tarde?", Libres, dia(11, 20), -1, "tarde", "", ""},
		{"hay algo libre mañana a la mañana para corte", Libres, dia(11, 19), -1, "manana", "", "s1"},
		{"¿hay lugar para corte y barba el sábado con Sofía?", Libres, dia(11, 21), -1, "", "p2", "s3"},
	}
	for _, k := range casos {
		in, ok := Interpretar(k.texto, c)
		if !ok {
			t.Errorf("%q: no lo entendió", k.texto)
			continue
		}
		if in.Tipo != k.tipo || !in.Fecha.Equal(k.fecha) || in.Franja != k.franja || in.ProfesionalID != k.prof || in.ServicioID != k.srv {
			t.Errorf("%q: %+v", k.texto, in)
		}
		if k.hora >= 0 {
			f := in.Fecha
			if f.IsZero() {
				f = c.hoy()
			}
			if h := c.Resolver(in.Hora, f).Hour(); !in.Hora.Dicha || h != k.hora {
				t.Errorf("%q: hora %d (%+v), quería %d", k.texto, h, in.Hora, k.hora)
			}
		}
	}
}

func TestInterpretarAcciones(t *testing.T) {
	c := catalogoDePrueba()
	casos := []struct {
		texto          string
		tipo, cliente  string
		fecha          time.Time
		hora           int
		destino        time.Time
		destinoHora    int
		destinoMinutos int
	}{
		{"Pasá a Juana al jueves a las 5", Mover, "juana", time.Time{}, -1, dia(11, 19), 17, 0},
		{"mové a Juana Pérez para el viernes a las 10", Mover, "juana perez", time.Time{}, -1, dia(11, 20), 10, 0},
		{"pasame a Pedro del jueves al sábado a las 4 y media", Mover, "pedro", dia(11, 19), -1, dia(11, 21), 16, 30},
		{"mové el de las 3 al viernes", Mover, "", time.Time{}, 15, dia(11, 20), -1, 0},
		{"cambiá a Pedro de las 3 a las 5", Mover, "pedro", time.Time{}, 15, time.Time{}, 17, 0},
		{"reprogramá a Lu a las 18hs", Mover, "lu", time.Time{}, -1, time.Time{}, 18, 0},
		{"Cancelá el de las 3", Cancelar, "", time.Time{}, 15, time.Time{}, -1, 0},
		{"cancelá el turno de Juana", Cancelar, "juana", time.Time{}, -1, time.Time{}, -1, 0},
		{"cancelale a Pedro el de mañana", Cancelar, "pedro", dia(11, 19), -1, time.Time{}, -1, 0},
		{"Confirmá a Pedro", Confirmar, "pedro", time.Time{}, -1, time.Time{}, -1, 0},
		{"confirmá el de las 11", Confirmar, "", time.Time{}, 11, time.Time{}, -1, 0},
	}
	for _, k := range casos {
		in, ok := Interpretar(k.texto, c)
		if !ok {
			t.Errorf("%q: no lo entendió (%+v)", k.texto, in)
			continue
		}
		if in.Tipo != k.tipo || in.Cliente != k.cliente || !in.Fecha.Equal(k.fecha) || !in.Destino.Equal(k.destino) {
			t.Errorf("%q: %+v", k.texto, in)
			continue
		}
		if k.hora >= 0 && (!in.Hora.Dicha || c.Resolver(in.Hora, c.hoy()).Hour() != k.hora) {
			t.Errorf("%q: hora de origen %+v", k.texto, in.Hora)
		}
		if k.destinoHora >= 0 {
			f := in.Destino
			if f.IsZero() {
				f = c.hoy()
			}
			r := c.Resolver(in.DestinoHora, f)
			if !in.DestinoHora.Dicha || r.Hour() != k.destinoHora || r.Minute() != k.destinoMinutos {
				t.Errorf("%q: hora de destino %v (%+v)", k.texto, r, in.DestinoHora)
			}
		}
	}
}

// Lo que no entiende con certeza va a la IA: no adivina.
func TestInterpretarNoAdivina(t *testing.T) {
	c := catalogoDePrueba()
	for _, x := range []string{
		"¿qué tengo que hacer para cambiar el logo?",
		"¿cuánto facturé este mes?",
		"avisale a la lista de espera",
		"pasá a Juana",                    // sin destino
		"cancelá",                         // sin a quién
		"quién viene",                     // sin hora
		"¿qué tengo la semana que viene?", // un rango: lo resuelve la IA
		"hola",
		"",
	} {
		if in, ok := Interpretar(x, c); ok {
			t.Errorf("%q: lo entendió como %+v", x, in)
		}
	}
}

// «A las 5» depende del horario: si atiende a las 5 de la mañana (y no a las 17), son las 5.
func TestResolverHoraConElHorario(t *testing.T) {
	c := catalogoDePrueba()
	c.Atencion = []disponibilidad.Franja{{Dia: 3, Desde: "05:00", Hasta: "12:00"}}
	if h := c.Resolver(HoraDicha{H: 5, Dicha: true}, dia(11, 18)).Hour(); h != 5 {
		t.Fatalf("hora %d, quería 5", h)
	}
	if h := c.Resolver(HoraDicha{H: 10, Dicha: true}, dia(11, 18)).Hour(); h != 10 {
		t.Fatalf("hora %d, quería 10", h)
	}
	// Un día sin atención: de 1 a 7 es de tarde.
	if h := c.Resolver(HoraDicha{H: 3, Dicha: true}, dia(11, 22)).Hour(); h != 15 {
		t.Fatalf("hora %d, quería 15", h)
	}
	if h := c.Resolver(HoraDicha{H: 9, Sufijo: "noche", Dicha: true}, dia(11, 22)).Hour(); h != 21 {
		t.Fatalf("hora %d, quería 21", h)
	}
}

func TestCoincideNombre(t *testing.T) {
	if !coincideNombre("juana", "Juana Pérez") || !coincideNombre("juana p", "Juana Pérez") || coincideNombre("juan", "Pedro") ||
		coincideNombre("", "Pedro") || !coincideNombre("perez", "Juana Pérez") {
		t.Fatal("coincideNombre")
	}
}
