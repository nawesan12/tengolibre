package main

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/cuenta"
)

func TestEscribirParaPurgar(t *testing.T) {
	var b bytes.Buffer
	EscribirParaPurgar(&b, nil)
	if !strings.Contains(b.String(), "no hay negocios cerrados hace más de 90 días") {
		t.Fatalf("vacío: %q", b.String())
	}
	b.Reset()
	cerrado := time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)
	EscribirParaPurgar(&b, []cuenta.NegocioParaPurgar{
		{ID: "a", Slug: "el-corte", Nombre: "El Corte", CerradoEn: cerrado, Clientes: 3, Turnos: 9, Cobros: 2},
		{ID: "b", Slug: "otro", Nombre: "Otro", CerradoEn: cerrado, Retenido: true},
	})
	s := b.String()
	for _, quiero := range []string{"el-corte", "2026-08-01", "retenido: suscripción que cobra", "2 negocios: 1 se borrarían, 1 retenidos"} {
		if !strings.Contains(s, quiero) {
			t.Errorf("falta %q en:\n%s", quiero, s)
		}
	}
}
