package clientes_test

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"

	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/clientes"
)

// sql corre una consulta como superusuario (sin RLS) y devuelve un valor.
func uno[T any](t *testing.T, m *mundo, consulta string, args ...any) T {
	t.Helper()
	conn, err := pgx.Connect(context.Background(), m.Bases.AdminURL)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close(context.Background())
	var v T
	if err := conn.QueryRow(context.Background(), consulta, args...).Scan(&v); err != nil {
		t.Fatalf("%s: %v", consulta, err)
	}
	return v
}

func TestCorregirElTelefono(t *testing.T) {
	m := armar(t)
	m.reservar(201, m.ana, "2026-11-02 09:00", "Juana Pérez", "1111111111")
	m.reservar(201, m.beto, "2026-11-02 10:00", "Pedro Gómez", "2222222222")
	var l listado
	m.panel(200, "GET", "/v1/panel/clientes", nil, &l)
	juana := l.Clientes[0]

	// El teléfono se limpia como en la reserva y queda único en el negocio.
	var c clientes.Cliente
	m.panel(200, "PATCH", "/v1/panel/clientes/"+juana.ID, map[string]any{"telefono": "+54 9 11 3333-3333", "nombre": "Juana Paz"}, &c)
	if c.Telefono != "+5491133333333" || c.Nombre != "Juana Paz" || c.Email != juana.Email {
		t.Fatalf("corregido: %+v", c)
	}
	if got := codigo(t, m.panel(409, "PATCH", "/v1/panel/clientes/"+juana.ID, map[string]any{"telefono": "2222222222"}, nil)); got != "cliente_existente" {
		t.Fatalf("teléfono de otro: %s", got)
	}
	m.panel(400, "PATCH", "/v1/panel/clientes/"+juana.ID, map[string]any{"telefono": "1234567890123456789012345678901"}, nil)
	// El profesional no edita clientes.
	m.como(profe, 403, "PATCH", "/v1/panel/clientes/"+juana.ID, map[string]any{"telefono": "4444444444"}, nil)
	// Reservar con el teléfono nuevo es la misma clienta.
	m.reservar(201, m.ana, "2026-11-02 10:00", "Juana", "+54 9 11 3333-3333")
	m.panel(200, "GET", "/v1/panel/clientes", nil, &l)
	if l.Total != 2 {
		t.Fatalf("con el teléfono corregido no tendría que nacer otro cliente: %+v", l)
	}
}

func TestSuprimirUnCliente(t *testing.T) {
	m := armar(t)
	// Sin turnos: se borra entero.
	var solo clientes.Cliente
	m.panel(201, "POST", "/v1/panel/clientes", map[string]any{"nombre": "Sin Turnos", "telefono": "5555555555", "email": "sin@x.com"}, &solo)
	var r struct {
		Resultado string `json:"resultado"`
	}
	m.como(profe, 403, "DELETE", "/v1/panel/clientes/"+solo.ID, nil, nil)
	m.como(recep, 200, "DELETE", "/v1/panel/clientes/"+solo.ID, nil, &r)
	if r.Resultado != "borrado" || uno[int](t, m, "SELECT count(*) FROM clientes WHERE id = $1", solo.ID) != 0 {
		t.Fatalf("sin turnos: %+v", r)
	}
	m.panel(404, "DELETE", "/v1/panel/clientes/"+solo.ID, nil, nil)

	// Con un turno por venir no se puede: primero se cancela.
	t1 := m.reservar(201, m.ana, "2026-11-02 09:00", "Juana Pérez", "1111111111")
	t2 := m.reservar(201, m.ana, "2026-11-02 09:30", "Juana Pérez", "1111111111")
	m.estado(t1.ID, "atendido")
	var l listado
	m.panel(200, "GET", "/v1/panel/clientes", nil, &l)
	juana := l.Clientes[0].ID
	m.panel(200, "PATCH", "/v1/panel/clientes/"+juana, map[string]any{"email": "juana@x.com", "nacimiento": "1990-05-04"}, nil)
	m.panel(201, "POST", "/v1/panel/clientes/"+juana+"/notas", map[string]any{"texto": "Le gusta corto"}, nil)
	if got := codigo(t, m.panel(409, "DELETE", "/v1/panel/clientes/"+juana, nil, nil)); got != "cliente_con_turnos" {
		t.Fatalf("con turnos por venir: %s", got)
	}
	m.estado(t2.ID, "cancelado")

	// Con turnos pasados: se anonimiza y los turnos se quedan, sin nota y con otro código.
	m.panel(200, "DELETE", "/v1/panel/clientes/"+juana, nil, &r)
	if r.Resultado != "anonimizado" {
		t.Fatalf("con turnos: %+v", r)
	}
	if n := uno[string](t, m, "SELECT nombre || '|' || telefono || '|' || email || '|' || coalesce(nacimiento::text, '') FROM clientes WHERE id = $1", juana); n != "Cliente borrado|||" {
		t.Fatalf("cliente anonimizado: %q", n)
	}
	if n := uno[int](t, m, "SELECT count(*) FROM turnos WHERE cliente_id = $1", juana); n != 2 {
		t.Fatalf("los turnos se conservan: %d", n)
	}
	if n := uno[int](t, m, "SELECT count(*) FROM notas_cliente WHERE cliente_id = $1", juana); n != 0 {
		t.Fatalf("las notas se borran: %d", n)
	}
	if n := uno[int](t, m, "SELECT count(*) FROM turnos WHERE codigo = $1", t1.Codigo); n != 0 {
		t.Fatal("el link viejo de Mi turno tendría que dejar de andar")
	}
	m.Esperar(404, P{Metodo: "GET", Ruta: "/v1/t/" + t1.Codigo}, nil)

	// No aparece en el listado ni en la búsqueda, no se edita y no se vuelve a suprimir.
	m.panel(200, "GET", "/v1/panel/clientes", nil, &l)
	if l.Total != 0 {
		t.Fatalf("el suprimido no se lista: %+v", l)
	}
	var b struct {
		Clientes []any `json:"clientes"`
	}
	m.panel(200, "GET", "/v1/panel/buscar?q=Cliente", nil, &b)
	if len(b.Clientes) != 0 {
		t.Fatalf("el suprimido no se busca: %+v", b)
	}
	m.panel(404, "PATCH", "/v1/panel/clientes/"+juana, map[string]any{"nombre": "Otra"}, nil)
	m.panel(404, "DELETE", "/v1/panel/clientes/"+juana, nil, nil)

	// Si vuelve a reservar con su teléfono, es una clienta nueva.
	m.reservar(201, m.beto, "2026-11-02 10:00", "Juana Pérez", "1111111111")
	m.panel(200, "GET", "/v1/panel/clientes", nil, &l)
	if l.Total != 1 || l.Clientes[0].ID == juana {
		t.Fatalf("la reserva nueva: %+v", l)
	}
}
