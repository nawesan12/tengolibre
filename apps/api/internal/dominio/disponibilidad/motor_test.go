package disponibilidad

import (
	"testing"
	"time"
)

var z = ZonaPorDefecto

func en(s string) time.Time {
	t, err := time.ParseInLocation("2006-01-02 15:04", s, z)
	if err != nil {
		panic(err)
	}
	return t
}

func str(s string) *string { return &s }

// El lunes 2 de noviembre de 2026, de 9 a 12, turnos de 30 cada 30.
func base() Consulta {
	return Consulta{
		Profesional: Profesional{ID: "p", Horario: []Franja{{Dia: 1, Desde: "09:00", Hasta: "12:00"}}},
		Fecha:       "2026-11-02",
		Duracion:    30,
		Reglas:      Reglas{Intervalo: 30, AnticipacionMin: 0, DiasVista: 30},
		Ahora:       en("2026-11-01 10:00"),
	}
}

func libres(hs []Horario) []string {
	out := []string{}
	for _, h := range hs {
		if h.Libre {
			out = append(out, h.Hora)
		}
	}
	return out
}

func igual(t *testing.T, got []string, want ...string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("libres %v, quería %v", got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("libres %v, quería %v", got, want)
		}
	}
}

func TestDiaSinNada(t *testing.T) {
	igual(t, libres(Dia(base())), "09:00", "09:30", "10:00", "10:30", "11:00", "11:30")
}

func TestServicioQueNoEntraAlCierre(t *testing.T) {
	c := base()
	c.Duracion = 60
	igual(t, libres(Dia(c)), "09:00", "09:30", "10:00", "10:30", "11:00")
}

func TestMargenSeSumaAlServicioYAlTurno(t *testing.T) {
	c := base()
	c.Reglas.MargenMin = 15
	c.Profesional.Turnos = []Ocupado{{Inicio: en("2026-11-02 10:00"), Fin: en("2026-11-02 10:30"), Estado: "confirmado"}}
	// 09:30 + 30 + 15 pisa las 10:00; el turno ocupa hasta 10:45, así que 10:30 tampoco; 11:30+45 no entra.
	igual(t, libres(Dia(c)), "09:00", "11:00")
}

func TestDiaQueNoAtiende(t *testing.T) {
	c := base()
	c.Fecha = "2026-11-03"
	igual(t, libres(Dia(c)))
}

func TestFueraDeLaVista(t *testing.T) {
	c := base()
	c.Reglas.DiasVista = 1 // sólo hoy (domingo 1)
	igual(t, libres(Dia(c)))
	c.Fecha = "2026-10-31"
	c.Reglas.DiasVista = 30
	igual(t, libres(Dia(c)))
}

func TestAnticipacion(t *testing.T) {
	c := base()
	c.Ahora = en("2026-11-02 08:00")
	c.Reglas.AnticipacionMin = 120
	igual(t, libres(Dia(c)), "10:00", "10:30", "11:00", "11:30")
}

func TestBloqueos(t *testing.T) {
	c := base()
	c.Profesional.Bloqueos = []Bloqueo{{Fecha: "2026-11-02", Desde: str("10:15"), Hasta: str("11:00")}}
	igual(t, libres(Dia(c)), "09:00", "09:30", "11:00", "11:30")
	c.Profesional.Bloqueos = []Bloqueo{{Fecha: "2026-11-02"}}
	if len(Dia(c)) != 0 {
		t.Fatal("un bloqueo del día entero no deja horarios")
	}
}

func TestSemanalYExcepciones(t *testing.T) {
	c := base()
	c.Profesional.Semanales = []Semanal{{Dia: 1, Hora: "10:00"}, {Dia: 1, Hora: "10:30:00"}}
	c.Profesional.Excepciones = []Excepcion{{Fecha: "2026-11-02", Hora: "10:30", Tipo: "libre"}, {Fecha: "2026-11-02", Hora: "09:00", Tipo: "bloqueado"}}
	igual(t, libres(Dia(c)), "09:30", "10:30", "11:00", "11:30")
}

func TestQueTurnosOcupan(t *testing.T) {
	c := base()
	vencio, vigente := en("2026-11-01 09:00"), en("2026-11-01 11:00")
	t0 := func(h, estado string, v *time.Time) Ocupado {
		return Ocupado{Inicio: en("2026-11-02 " + h), Fin: en("2026-11-02 " + h).Add(30 * time.Minute), Estado: estado, VenceEn: v}
	}
	c.Profesional.Turnos = []Ocupado{
		t0("09:00", "cancelado", nil), t0("09:30", "pendiente_pago", &vencio), t0("10:00", "pendiente_pago", &vigente),
		t0("10:30", "pendiente", nil), t0("11:00", "ausente", nil),
	}
	igual(t, libres(Dia(c)), "09:00", "09:30", "11:00", "11:30")
}

func TestHorarioCortadoYOtroIntervalo(t *testing.T) {
	c := base()
	c.Profesional.Horario = []Franja{{Dia: 1, Desde: "16:00", Hasta: "17:00"}, {Dia: 1, Desde: "09:00", Hasta: "10:00"}}
	c.Reglas.Intervalo = 20
	igual(t, libres(Dia(c)), "09:00", "09:20", "16:00", "16:20")
}

func TestLibre(t *testing.T) {
	c := base()
	if !Libre(c, en("2026-11-02 09:30")) || Libre(c, en("2026-11-02 09:40")) || Libre(c, en("2026-11-02 12:00")) {
		t.Fatal("Libre no coincide con Dia")
	}
}
