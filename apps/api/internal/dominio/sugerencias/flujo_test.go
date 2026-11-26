package sugerencias_test

import (
	"testing"
	"time"

	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/cuenta"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/negocios"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/sugerencias"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/db/pruebadb"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/prueba"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/servidor"
)

type P = prueba.Pedido

const (
	duena     = "user_sugeduenia0001"
	empleada  = "user_sugeempleada01"
	encargada = "user_sugeencargada1"
)

type respuesta struct {
	Sugerencia *sugerencias.Sugerencia `json:"sugerencia"`
}

func armar(t *testing.T, activas bool) *prueba.API {
	return prueba.Nueva(t, time.Now(), func(a *prueba.API) []servidor.Dominio {
		return []servidor.Dominio{a.Sesion, &negocios.Servicio{Pool: a.Bases.App, Sesion: a.Sesion},
			&cuenta.Servicio{Pool: a.Bases.App, Sesion: a.Sesion},
			&sugerencias.Servicio{Pool: a.Bases.App, Sesion: a.Sesion, Activas: activas, Sitio: "https://tengolibre.com.ar/", Ahora: a.Reloj()}}
	})
}

// negocioListo: una barbería de 20 días con plan, el alta terminada (servicio, profesional con
// horario y asignado) y un turno. Le toca la sugerencia.
func negocioListo(t *testing.T, a *prueba.API, slug string) string {
	t.Helper()
	n := pruebadb.CrearNegocio(t, a.Bases.App, slug, duena)
	a.PonerPlan(n, "equipo")
	a.Fila("UPDATE negocios SET rubro = 'barberia', creado_en = now() - interval '20 days' WHERE id = $1", []any{n})
	var s, p string
	a.Fila("INSERT INTO servicios (negocio_id, nombre, duracion, precio) VALUES ($1, 'Corte', 30, 8000) RETURNING id::text", []any{n}, &s)
	a.Fila("INSERT INTO profesionales (negocio_id, nombre) VALUES ($1, 'Ana') RETURNING id::text", []any{n}, &p)
	a.Fila("INSERT INTO horarios (negocio_id, profesional_id, dia, desde, hasta) VALUES ($1, $2, 1, '09:00', '18:00')", []any{n, p})
	a.Fila("INSERT INTO profesional_servicios (negocio_id, profesional_id, servicio_id) VALUES ($1, $2, $3)", []any{n, p, s})
	var cli string
	a.Fila("INSERT INTO clientes (negocio_id, nombre, telefono) VALUES ($1, 'Juana', '1111111111') RETURNING id::text", []any{n}, &cli)
	a.Fila(`INSERT INTO turnos (negocio_id, codigo, servicio, profesional, cliente_id, inicio, fin, precio, estado, origen)
		VALUES ($1, replace(gen_random_uuid()::text, '-', ''), 'Corte', 'Ana', $2, now() - interval '2 days',
			now() - interval '2 days' + interval '30 minutes', 8000, 'atendido', 'web')`, []any{n, cli})
	return n
}

func ver(t *testing.T, a *prueba.API, quien, negocio string) *sugerencias.Sugerencia {
	t.Helper()
	var r respuesta
	a.Esperar(200, P{Metodo: "GET", Ruta: "/v1/panel/sugerencia", Quien: quien, Negocio: negocio}, &r)
	return r.Sugerencia
}

func entero(t *testing.T, a *prueba.API, q string, args ...any) int {
	t.Helper()
	var n int
	a.Fila(q, args, &n)
	return n
}

func TestSugerenciaDePuntaAPunta(t *testing.T) {
	a := armar(t, true)
	n := negocioListo(t, a, "barberia-sur")

	s := ver(t, a, duena, n)
	if s == nil || s.ID != "tl-tlw" || s.Marca != "tulocalweb" || s.Sello != "De Grupo Vimo" {
		t.Fatalf("la dueña no ve la sugerencia: %+v", s)
	}
	if s.Imagen != "https://tengolibre.com.ar/sugerencias/tulocalweb-mini.jpg" || s.Banner != "https://tengolibre.com.ar/sugerencias/tulocalweb-banner.jpg" {
		t.Errorf("imágenes: %s %s", s.Imagen, s.Banner)
	}
	if s.URL != "https://tulocalweb.com.ar/empezar?utm_source=tengolibre&utm_medium=sugerencia&utm_content=tl-tlw" {
		t.Errorf("url: %s", s.URL)
	}
	if m := entero(t, a, "SELECT mostradas FROM sugerencias_metricas WHERE sugerencia_id = 'tl-tlw'"); m != 1 {
		t.Errorf("mostradas = %d", m)
	}

	// La encargada también; la profesional (empleada) recibe null y no puede cerrarla.
	a.Fila("INSERT INTO miembros (negocio_id, clerk_id, rol) VALUES ($1, $2, 'encargado'), ($1, $3, 'profesional')", []any{n, encargada, empleada})
	if ver(t, a, encargada, n) == nil {
		t.Error("la encargada no la ve")
	}
	if ver(t, a, empleada, n) != nil {
		t.Error("la empleada la ve")
	}
	a.Esperar(403, P{Metodo: "POST", Ruta: "/v1/panel/sugerencia/tl-tlw/cerrar", Cuerpo: map[string]string{"hasta": "30d"}, Quien: empleada, Negocio: n}, nil)
	a.Esperar(403, P{Metodo: "PUT", Ruta: "/v1/panel/ajustes/sugerencias", Cuerpo: map[string]bool{"activas": false}, Quien: empleada, Negocio: n}, nil)
	// Sin sesión: 401.
	a.Esperar(401, P{Metodo: "GET", Ruta: "/v1/panel/sugerencia", Negocio: n}, nil)
	// Otra persona, de otro negocio: 403.
	a.Esperar(403, P{Metodo: "GET", Ruta: "/v1/panel/sugerencia", Quien: "user_sugeotra000001", Negocio: n}, nil)

	// El clic suma en el contador agregado.
	a.Esperar(204, P{Metodo: "POST", Ruta: "/v1/panel/sugerencia/tl-tlw/clic", Quien: duena, Negocio: n}, nil)
	if c := entero(t, a, "SELECT clics FROM sugerencias_metricas WHERE sugerencia_id = 'tl-tlw'"); c != 1 {
		t.Errorf("clics = %d", c)
	}
	a.Esperar(404, P{Metodo: "POST", Ruta: "/v1/panel/sugerencia/otramas/clic", Quien: duena, Negocio: n}, nil)

	// Ocultar por 30 días: no vuelve hasta pasados los 30.
	a.Esperar(400, P{Metodo: "POST", Ruta: "/v1/panel/sugerencia/tl-tlw/cerrar", Cuerpo: map[string]string{"hasta": "1d"}, Quien: duena, Negocio: n}, nil)
	a.Esperar(204, P{Metodo: "POST", Ruta: "/v1/panel/sugerencia/tl-tlw/cerrar", Cuerpo: map[string]string{"hasta": "30d"}, Quien: duena, Negocio: n}, nil)
	if ver(t, a, duena, n) != nil {
		t.Fatal("cerrada, se sigue viendo")
	}
	inicio := a.Ahora
	a.Ahora = inicio.AddDate(0, 0, 29)
	if ver(t, a, duena, n) != nil {
		t.Error("vuelve antes de los 30 días")
	}
	a.Ahora = inicio.AddDate(0, 0, 31)
	if ver(t, a, duena, n) == nil {
		t.Fatal("no vuelve con el reloj +31 días")
	}

	// «Ya la uso»: para siempre (y un «30 días» después no la acorta).
	a.Esperar(204, P{Metodo: "POST", Ruta: "/v1/panel/sugerencia/tl-tlw/cerrar", Cuerpo: map[string]string{"hasta": "siempre"}, Quien: duena, Negocio: n}, nil)
	a.Esperar(204, P{Metodo: "POST", Ruta: "/v1/panel/sugerencia/tl-tlw/cerrar", Cuerpo: map[string]string{"hasta": "30d"}, Quien: duena, Negocio: n}, nil)
	a.Ahora = inicio.AddDate(3, 0, 0)
	if ver(t, a, duena, n) != nil {
		t.Error("«Ya la uso» volvió")
	}
	a.Ahora = inicio
	a.Fila("DELETE FROM sugerencias_cerradas WHERE negocio_id = $1", []any{n})

	// El interruptor de Ajustes.
	var aj sugerencias.Ajuste
	a.Esperar(200, P{Metodo: "GET", Ruta: "/v1/panel/ajustes/sugerencias", Quien: duena, Negocio: n}, &aj)
	if !aj.Activas {
		t.Error("el interruptor arranca apagado")
	}
	a.Esperar(200, P{Metodo: "PUT", Ruta: "/v1/panel/ajustes/sugerencias", Cuerpo: map[string]bool{"activas": false}, Quien: duena, Negocio: n}, &aj)
	if ver(t, a, duena, n) != nil {
		t.Error("con el interruptor apagado se ve")
	}
	a.Esperar(200, P{Metodo: "PUT", Ruta: "/v1/panel/ajustes/sugerencias", Cuerpo: map[string]bool{"activas": true}, Quien: duena, Negocio: n}, &aj)
	if ver(t, a, duena, n) == nil {
		t.Error("prendida otra vez no se ve")
	}

	// La prueba vencida sin plan (sólo lectura): nada, pero cerrar y apagar siguen andando.
	a.Fila("UPDATE suscripciones SET estado = 'cancelada', cancelada_en = now() WHERE negocio_id = $1", []any{n})
	a.Fila("UPDATE negocios SET prueba_hasta = now() - interval '1 day' WHERE id = $1", []any{n})
	if ver(t, a, duena, n) != nil {
		t.Error("con el panel bloqueado se ve")
	}

	// Borrar la cuenta (única dueña): el negocio se cierra y sus sugerencias cerradas se van.
	a.Esperar(204, P{Metodo: "POST", Ruta: "/v1/panel/sugerencia/tl-tlw/cerrar", Cuerpo: map[string]string{"hasta": "siempre"}, Quien: duena, Negocio: n}, nil)
	a.Fila("DELETE FROM miembros WHERE negocio_id = $1 AND rol = 'encargado'", []any{n})
	a.Esperar(204, P{Metodo: "DELETE", Ruta: "/v1/auth/cuenta", Quien: duena}, nil)
	if c := entero(t, a, "SELECT count(*) FROM sugerencias_cerradas WHERE negocio_id = $1", n); c != 0 {
		t.Errorf("quedaron %d sugerencias cerradas", c)
	}
}

// Las condiciones del negocio, una por una, contra la base (las reglas finas están en reglas_test).
func TestCuandoNoAparece(t *testing.T) {
	a := armar(t, true)
	n := negocioListo(t, a, "peluqueria-norte")
	if ver(t, a, duena, n) == nil {
		t.Fatal("el negocio listo no la ve")
	}
	casos := []struct{ nombre, rompe, arregla string }{
		{"sin rubro", "UPDATE negocios SET rubro = '' WHERE id = $1", "UPDATE negocios SET rubro = 'barberia' WHERE id = $1"},
		{"cuenta de 13 días", "UPDATE negocios SET creado_en = now() - interval '13 days' WHERE id = $1",
			"UPDATE negocios SET creado_en = now() - interval '20 days' WHERE id = $1"},
		{"alta sin terminar", "DELETE FROM profesional_servicios WHERE negocio_id = $1", ""},
	}
	for _, c := range casos {
		a.Fila(c.rompe, []any{n})
		if ver(t, a, duena, n) != nil {
			t.Errorf("%s: se ve", c.nombre)
		}
		if c.arregla != "" {
			a.Fila(c.arregla, []any{n})
		}
	}
	// Cuenta vacía: otro negocio listo pero sin turnos.
	v := negocioListo(t, a, "estetica-vacia")
	a.Fila("DELETE FROM turnos WHERE negocio_id = $1", []any{v})
	if ver(t, a, duena, v) != nil {
		t.Error("cuenta vacía: se ve")
	}
}

// Con SUGERENCIAS_ACTIVAS apagado (el valor por defecto) la respuesta es siempre null.
func TestApagadasPorDefecto(t *testing.T) {
	a := armar(t, false)
	n := negocioListo(t, a, "spa-apagado")
	if ver(t, a, duena, n) != nil {
		t.Error("con SUGERENCIAS_ACTIVAS=false se ve")
	}
	if c := entero(t, a, "SELECT count(*) FROM sugerencias_metricas"); c != 0 {
		t.Errorf("contó %d", c)
	}
}

// El alta guarda el origen (utm) válido y descarta lo que no sirve.
func TestAltaConOrigen(t *testing.T) {
	a := armar(t, false)
	var n struct {
		ID string `json:"id"`
	}
	a.Esperar(201, P{Metodo: "POST", Ruta: "/v1/negocios", Quien: duena, Cuerpo: map[string]any{
		"slug": "con-origen", "nombre": "Con origen",
		"origen": map[string]string{"fuente": "tulocalweb", "medio": "sugerencia", "contenido": "tlw-tl", "campana": "<script>", "llego": "2026-12-01T10:00:00.000Z"},
	}}, &n)
	var fuente, medio, contenido string
	var campana *string
	a.Fila("SELECT origen->>'fuente', origen->>'medio', origen->>'contenido', origen->>'campana' FROM negocios WHERE id = $1", []any{n.ID}, &fuente, &medio, &contenido, &campana)
	if fuente != "tulocalweb" || medio != "sugerencia" || contenido != "tlw-tl" || campana != nil {
		t.Errorf("origen: %s %s %s %v", fuente, medio, contenido, campana)
	}
	a.Esperar(201, P{Metodo: "POST", Ruta: "/v1/negocios", Quien: duena, Cuerpo: map[string]any{"slug": "sin-origen", "nombre": "Sin origen"}}, &n)
	var nulo bool
	a.Fila("SELECT origen IS NULL FROM negocios WHERE id = $1", []any{n.ID}, &nulo)
	if !nulo {
		t.Error("sin origen guardó algo")
	}
}
