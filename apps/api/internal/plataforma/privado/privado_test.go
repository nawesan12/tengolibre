package privado

import "testing"

func TestEnmascarar(t *testing.T) {
	casos := map[string]string{
		"550 5.1.1 <rocio.paz@gmail.com>: no existe": "550 5.1.1 <***@gmail.com>: no existe",
		"destino +54 9 11 5555-1234 inválido":        "destino …1234 inválido",
		"intento 3 de 5":                             "intento 3 de 5",
		"turno 2026-11-02":                           "turno 2026-11-02",
	}
	for in, quiero := range casos {
		if got := Enmascarar(in); got != quiero {
			t.Errorf("%q → %q, quería %q", in, got, quiero)
		}
	}
}
