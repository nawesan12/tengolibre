package servidor

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSalud(t *testing.T) {
	h := Nuevo(Dependencias{Version: "prueba"})
	casos := []struct {
		ruta   string
		estado int
	}{
		{"/salud/vivo", http.StatusOK},
		{"/salud/listo", http.StatusInternalServerError}, // sin base
		{"/v1/version", http.StatusOK},
		{"/no-existe", http.StatusNotFound},
	}
	for _, c := range casos {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, c.ruta, nil))
		if rec.Code != c.estado {
			t.Errorf("%s: %d, quería %d", c.ruta, rec.Code, c.estado)
		}
	}
}
