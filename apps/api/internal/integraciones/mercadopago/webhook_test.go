package mercadopago

import (
	"context"
	"net/http/httptest"
	"strings"
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

// TestWebhookIDFirmadoYProcesado: la firma cubre el data.id de la URL; si el del cuerpo es otro, no
// se procesa nada (con una firma de otro aviso se haría procesar cualquier id).
func TestWebhookIDFirmadoYProcesado(t *testing.T) {
	ahora := time.Date(2026, 11, 1, 10, 0, 0, 0, time.UTC)
	var procesados []string
	h := Webhook("secreto", func() time.Time { return ahora }, nil, map[string]Receptor{
		"payment": func(_ context.Context, a Aviso) error { procesados = append(procesados, a.ID()); return nil },
	})
	enviar := func(urlID, cuerpoID string) int {
		cuerpo := `{"type":"payment","data":{}}`
		if cuerpoID != "" {
			cuerpo = `{"type":"payment","data":{"id":"` + cuerpoID + `"}}`
		}
		ruta := "/v1/webhooks/mercadopago"
		if urlID != "" {
			ruta += "?data.id=" + urlID
		}
		r := httptest.NewRequest("POST", ruta, strings.NewReader(cuerpo))
		r.Header.Set("x-request-id", "req-1")
		r.Header.Set("x-signature", Firmar("secreto", "req-1", "111", ahora.UnixMilli()))
		rec := httptest.NewRecorder()
		h(rec, r)
		return rec.Code
	}
	if c := enviar("111", "222"); c != 401 {
		t.Fatalf("id firmado 111 y avisado 222: %d", c)
	}
	if c := enviar("111", "111"); c != 200 {
		t.Fatalf("mismo id: %d", c)
	}
	if c := enviar("111", ""); c != 200 {
		t.Fatalf("sólo en la URL: %d", c)
	}
	if c := enviar("", "111"); c != 200 {
		t.Fatalf("sólo en el cuerpo: %d", c)
	}
	if strings.Join(procesados, ",") != "111,111,111" {
		t.Fatalf("procesados: %v", procesados)
	}
}
