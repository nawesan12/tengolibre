package tope

import (
	"errors"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/problema"
)

func TestPermitirPublico(t *testing.T) {
	var c Contador
	ahora := time.Date(2026, 11, 1, 10, 0, 0, 0, time.UTC)
	p := Publico{PorIP: 2, PorNegocio: 3, Ventana: 10 * time.Minute}
	desde := func(ip string) error {
		r := httptest.NewRequest("POST", "/", nil)
		r.Header.Set("CF-Connecting-IP", ip)
		return c.PermitirPublico(r, "reservar", "n1", p, ahora)
	}
	if desde("1.1.1.1") != nil || desde("1.1.1.1") != nil {
		t.Fatal("las dos primeras entran")
	}
	var pr *problema.Problema
	if err := desde("1.1.1.1"); !errors.As(err, &pr) || pr.Estado != 429 {
		t.Fatalf("la tercera desde la misma IP: %v", err)
	}
	// Otra IP entra, hasta el tope del negocio (lo rechazado por IP no le suma).
	if desde("2.2.2.2") != nil {
		t.Fatal("otra IP entra")
	}
	if desde("3.3.3.3") == nil {
		t.Fatal("el negocio ya pasó su tope")
	}
	// Otro negocio y otra acción cuentan aparte; pasada la ventana, vuelve a entrar.
	r := httptest.NewRequest("POST", "/", nil)
	if c.PermitirPublico(r, "espera", "n1", p, ahora) != nil || c.PermitirPublico(r, "reservar", "n2", p, ahora) != nil {
		t.Fatal("cuentan aparte")
	}
	ahora = ahora.Add(11 * time.Minute)
	if desde("1.1.1.1") != nil {
		t.Fatal("pasada la ventana vuelve a entrar")
	}
}
