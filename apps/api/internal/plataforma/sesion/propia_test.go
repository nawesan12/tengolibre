package sesion_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/negocios"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/prueba"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/servidor"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/sesion"
)

type P = prueba.Pedido

// con manda un pedido con un Bearer arbitrario (prueba.Hacer arma "dev:…").
func con(a *prueba.API, metodo, ruta, token, negocio string, cuerpo string) *httptest.ResponseRecorder {
	var body *strings.Reader
	if cuerpo != "" {
		body = strings.NewReader(cuerpo)
	} else {
		body = strings.NewReader("")
	}
	req := httptest.NewRequest(metodo, ruta, body)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if negocio != "" {
		req.Header.Set("X-Negocio", negocio)
	}
	rec := httptest.NewRecorder()
	a.Handler.ServeHTTP(rec, req)
	return rec
}

func TestSesionPropia(t *testing.T) {
	a := prueba.Nueva(t, time.Now(), func(a *prueba.API) []servidor.Dominio {
		return []servidor.Dominio{a.Sesion, &negocios.Servicio{Pool: a.Bases.App, Sesion: a.Sesion}}
	})
	// El token de Clerk (acá, el falso de desarrollo) se canjea por una sesión propia.
	var r struct {
		Token   string         `json:"token"`
		Persona sesion.Persona `json:"persona"`
	}
	a.Esperar(201, P{Metodo: "POST", Ruta: "/v1/auth/clerk", Cuerpo: map[string]string{"token": "dev:user_propia0001"}}, &r)
	if !strings.HasPrefix(r.Token, sesion.Prefijo) || r.Persona.ClerkID != "user_propia0001" || r.Persona.Email == "" {
		t.Fatalf("canje: %+v", r)
	}
	a.Esperar(401, P{Metodo: "POST", Ruta: "/v1/auth/clerk", Cuerpo: map[string]string{"token": "cualquiera"}}, nil)

	// Con la sesión propia se crea un negocio y se entra al panel.
	if rec := con(a, "POST", "/v1/negocios", r.Token, "", `{"slug":"propia","nombre":"Propia"}`); rec.Code != http.StatusCreated {
		t.Fatalf("alta con sesión propia: %d %s", rec.Code, rec.Body)
	}
	if rec := con(a, "GET", "/v1/auth/persona", r.Token, "", ""); rec.Code != 200 || !strings.Contains(rec.Body.String(), "user_propia0001") {
		t.Fatalf("persona: %d %s", rec.Code, rec.Body)
	}
	// Un token propio inventado no entra; salir la cierra.
	if rec := con(a, "GET", "/v1/yo", sesion.Prefijo+"inventado", "", ""); rec.Code != 401 {
		t.Fatalf("token inventado: %d", rec.Code)
	}
	if rec := con(a, "POST", "/v1/auth/salir", r.Token, "", ""); rec.Code != 204 {
		t.Fatalf("salir: %d", rec.Code)
	}
	if rec := con(a, "GET", "/v1/yo", r.Token, "", ""); rec.Code != 401 {
		t.Fatalf("después de salir: %d", rec.Code)
	}
}
