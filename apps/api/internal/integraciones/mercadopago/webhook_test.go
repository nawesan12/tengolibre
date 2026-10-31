package mercadopago

import (
	"testing"
	"time"
)

func TestFirmaDelWebhook(t *testing.T) {
	ahora := time.Date(2026, 11, 1, 10, 0, 0, 0, time.UTC)
	f := Firmar("secreto", "req-1", "ABC123", ahora.UnixMilli())
	if !FirmaValida("secreto", f, "req-1", "ABC123", ahora) {
		t.Fatal("la firma propia no valida")
	}
	casos := map[string]bool{
		"otro secreto":   FirmaValida("otro", f, "req-1", "ABC123", ahora),
		"otro request":   FirmaValida("secreto", f, "req-2", "ABC123", ahora),
		"otro recurso":   FirmaValida("secreto", f, "req-1", "XYZ", ahora),
		"vieja":          FirmaValida("secreto", f, "req-1", "ABC123", ahora.Add(11*time.Minute)),
		"sin encabezado": FirmaValida("secreto", "", "req-1", "ABC123", ahora),
		"sin secreto":    FirmaValida("", f, "req-1", "ABC123", ahora),
	}
	for caso, valida := range casos {
		if valida {
			t.Errorf("%s: aceptó la firma", caso)
		}
	}
	// El id alfanumérico va en minúsculas en el manifiesto (así lo firma Mercado Pago).
	if !FirmaValida("secreto", f, "req-1", "abc123", ahora) {
		t.Error("el id en minúsculas tiene que validar igual")
	}
}
