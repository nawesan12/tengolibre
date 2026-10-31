package membresias_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/cobros/cobrosprueba"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/membresias"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/sesion"
)

// sesionDe abre una sesión propia (la que guarda el front) para alguien con ese email verificado.
func sesionDe(e *cobrosprueba.Entorno, clerk, email string) string {
	e.T.Helper()
	token := sesion.Prefijo + clerk
	h := sha256.Sum256([]byte(token))
	e.SQL("INSERT INTO auth.sesiones (hash, clerk_id, email, vence_en) VALUES ($1, $2, $3, now() + interval '1 day')", h[:], clerk, email)
	return token
}

// conToken hace un pedido público con esa sesión y exige el estado.
func conToken(e *cobrosprueba.Entorno, estado int, metodo, ruta, token string, cuerpo, v any) {
	e.T.Helper()
	var b bytes.Buffer
	if cuerpo != nil {
		_ = json.NewEncoder(&b).Encode(cuerpo)
	}
	req := httptest.NewRequest(metodo, ruta, &b)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	e.Handler.ServeHTTP(rec, req)
	if rec.Code != estado {
		e.T.Fatalf("%s %s: %d, se esperaba %d: %s", metodo, ruta, rec.Code, estado, rec.Body)
	}
	if v != nil {
		if err := json.Unmarshal(rec.Body.Bytes(), v); err != nil {
			e.T.Fatal(err)
		}
	}
}

func reservaWeb(e *cobrosprueba.Entorno, hora, telefono, email string, usar bool) map[string]any {
	return map[string]any{"servicioId": e.Servicio, "profesionalId": e.Profesional, "inicio": cobrosprueba.En("2026-11-02 " + hora).Format(time.RFC3339),
		"cliente": map[string]string{"nombre": "Alguien", "telefono": telefono, "email": email}, "usarMembresia": usar}
}

func TestMembresiaDesdeLaPagina(t *testing.T) {
	e := cobrosprueba.Armar(t)
	var p membresias.Plan
	e.Panel(201, "POST", "/v1/panel/membresias/planes", map[string]any{"nombre": "Un corte", "precio": 9000, "visitas": 1,
		"duracionDias": 30, "servicios": []string{e.Servicio}}, &p)
	var s membresias.Socio
	e.Panel(201, "POST", "/v1/panel/membresias/socios", map[string]any{"planId": p.ID, "nombre": "Socia", "telefono": "333333", "email": "Socia@Mail.com"}, &s)
	socia := sesionDe(e, "user_socia000001", "socia@mail.com")
	ajena := sesionDe(e, "user_ajena000001", "ajena@mail.com")

	type mia struct {
		Membresia *membresias.MiMembresia `json:"membresia"`
	}
	var m mia
	conToken(e, 200, "GET", "/v1/p/el-corte/mi-membresia", "", nil, &m)
	if m.Membresia != nil {
		t.Fatalf("sin sesión: %+v", m.Membresia)
	}
	conToken(e, 200, "GET", "/v1/p/el-corte/mi-membresia", socia, nil, &m)
	if m.Membresia == nil || m.Membresia.Plan != "Un corte" || m.Membresia.Saldo != 1 {
		t.Fatalf("la socia: %+v", m.Membresia)
	}

	// Alguien que escribe el teléfono de la socia y su propio email no le pisa el email (que cargó el
	// negocio) ni puede usar la membresía con su cuenta.
	conToken(e, 201, "POST", "/v1/p/el-corte/turnos", "", reservaWeb(e, "09:00", "333333", "ajena@mail.com", false), nil)
	conToken(e, 200, "GET", "/v1/p/el-corte/mi-membresia", ajena, nil, &m)
	if m.Membresia != nil {
		t.Fatalf("la ajena ve la membresía: %+v", m.Membresia)
	}
	conToken(e, 409, "POST", "/v1/p/el-corte/turnos", ajena, reservaWeb(e, "09:30", "555555", "", true), nil)
	conToken(e, 401, "POST", "/v1/p/el-corte/turnos", "", reservaWeb(e, "09:30", "555555", "", true), nil)
	if x := socio(e, s.ID); x.Quedan != 1 {
		t.Fatalf("se gastó una visita sin pedirla: %d", x.Quedan)
	}

	// La socia, con su sesión y pidiéndolo, reserva cubierta (sin seña) y gasta la visita.
	conToken(e, 201, "POST", "/v1/p/el-corte/turnos", socia, reservaWeb(e, "10:00", "999999", "", true), nil)
	x := socio(e, s.ID)
	if x.Quedan != 0 || x.Creditos[0].TurnoID == nil {
		t.Fatalf("después de reservar: %+v", x)
	}
	// Sin visitas, no reserva pagando sin darse cuenta.
	conToken(e, 409, "POST", "/v1/p/el-corte/turnos", socia, reservaWeb(e, "10:30", "333333", "", true), nil)

	// Un email que vino de la página no es de confianza: aunque después le den una membresía por
	// teléfono, ingresar con ese email no la usa.
	conToken(e, 201, "POST", "/v1/p/el-corte/turnos", "", reservaWeb(e, "11:00", "444444", "nuevo@mail.com", false), nil)
	e.Panel(201, "POST", "/v1/panel/membresias/socios", map[string]any{"planId": p.ID, "nombre": "Nuevo", "telefono": "444444"}, nil)
	conToken(e, 200, "GET", "/v1/p/el-corte/mi-membresia", sesionDe(e, "user_nuevo000001", "nuevo@mail.com"), nil, &m)
	if m.Membresia != nil {
		t.Fatalf("email de la web tomado como de confianza: %+v", m.Membresia)
	}
}
