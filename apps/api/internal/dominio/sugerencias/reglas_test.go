package sugerencias

import (
	"net/url"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/miweb"
)

var ahora = time.Date(2026, 12, 1, 12, 0, 0, 0, time.UTC)

// base: una dueña de una barbería con 20 días, el alta terminada y turnos. Le toca la sugerencia.
func base() Datos {
	return Datos{Rol: "duenio", Rubro: "barberia", CreadoEn: ahora.AddDate(0, 0, -20), AltaCompleta: true, Turnos: 1, Cerradas: map[string]time.Time{}}
}

// Una fila por regla: qué cambia del caso base y si se muestra.
func TestReglas(t *testing.T) {
	casos := []struct {
		nombre  string
		cambiar func(d *Datos)
		ve      bool
	}{
		{"dueña con rubro", func(d *Datos) {}, true},
		{"encargada", func(d *Datos) { d.Rol = "encargado" }, true},
		{"rubro otro", func(d *Datos) { d.Rubro = "otro" }, true},
		{"sin rubro", func(d *Datos) { d.Rubro = "" }, false},
		{"rubro desconocido", func(d *Datos) { d.Rubro = "gastronomia" }, false},
		{"profesional (empleada)", func(d *Datos) { d.Rol = "profesional" }, false},
		{"recepción", func(d *Datos) { d.Rol = "recepcion" }, false},
		{"cuenta de 13 días", func(d *Datos) { d.CreadoEn = ahora.AddDate(0, 0, -13) }, false},
		{"cuenta de 14 días justos", func(d *Datos) { d.CreadoEn = ahora.AddDate(0, 0, -14) }, true},
		{"alta sin terminar (falta un paso obligatorio)", func(d *Datos) { d.AltaCompleta = false }, false},
		{"cuenta vacía (sin turnos)", func(d *Datos) { d.Turnos = 0 }, false},
		{"prueba vencida sin plan", func(d *Datos) { d.Bloqueado = true }, false},
		{"interruptor apagado", func(d *Datos) { d.Apagadas = true }, false},
		{"cerrada por 30 días, día 29", func(d *Datos) { d.Cerradas["tulocalweb"] = ahora.AddDate(0, 0, 1) }, false},
		{"cerrada por 30 días, ya pasó", func(d *Datos) { d.Cerradas["tulocalweb"] = ahora.AddDate(0, 0, -1) }, true},
		{"ya la usa", func(d *Datos) { d.Cerradas["tulocalweb"] = ParaSiempre }, false},
		{"cerrada otra marca", func(d *Datos) { d.Cerradas["caigo"] = ParaSiempre }, true},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			d := base()
			c.cambiar(&d)
			s := Elegir(d, ahora)
			if (s != nil) != c.ve {
				t.Fatalf("ve=%v, quería %v", s != nil, c.ve)
			}
			if s != nil && s.ID != "tl-tlw" {
				t.Fatalf("sugerencia %q", s.ID)
			}
		})
	}
}

// Todos los rubros del directorio la activan.
func TestTodosLosRubros(t *testing.T) {
	for _, r := range miweb.Rubros {
		d := base()
		d.Rubro = r
		if Elegir(d, ahora) == nil {
			t.Errorf("el rubro %q no ve la sugerencia", r)
		}
	}
}

// El catálogo sólo tiene marcas permitidas: nunca Otra Más, De a Poco, La Columna, Bancamos, Corredor
// ni una app nueva; y Tengo Libre no se sugiere a sí misma.
func TestCatalogoSoloMarcasPermitidas(t *testing.T) {
	if !slices.Equal(Marcas, []string{"tulocalweb", "tengolibre", "caigo"}) {
		t.Fatalf("Marcas cambió: %v (sólo tulocalweb, tengolibre y caigo)", Marcas)
	}
	ids := map[string]bool{}
	for _, s := range Catalogo {
		if !slices.Contains(Marcas, s.Marca) {
			t.Errorf("%s: marca %q fuera de la lista", s.ID, s.Marca)
		}
		if s.Marca == Fuente {
			t.Errorf("%s: Tengo Libre no se sugiere a sí misma", s.ID)
		}
		if ids[s.ID] {
			t.Errorf("%s repetida", s.ID)
		}
		ids[s.ID] = true
		if s.Sello != "De Grupo Vimo" || s.Titulo == "" || s.CTA == "" || len([]rune(s.Titulo)) > 32 {
			t.Errorf("%s: textos %+v", s.ID, s)
		}
	}
	for _, r := range Reglas {
		if _, ok := DelCatalogo(r.Sugerencia); !ok {
			t.Errorf("la regla apunta a %q, que no está en el catálogo", r.Sugerencia)
		}
	}
}

// Cada destino está en la lista blanca de dominios y no lleva más que los utm.
func TestDestinosEnListaBlanca(t *testing.T) {
	for _, s := range Catalogo {
		if !destinoValido(s) {
			t.Errorf("%s: destino inválido %q", s.ID, s.URL)
		}
	}
	u, _ := url.Parse(Catalogo[0].URL)
	if u.Host != "tulocalweb.com.ar" || u.Path != "/empezar" {
		t.Errorf("tl-tlw va a %s", u)
	}
}

// El chequeo de la lista blanca rechaza lo que no corresponde.
func TestListaBlancaRechaza(t *testing.T) {
	ok := Catalogo[0]
	malos := map[string]string{
		"otra marca":     "https://otramas.com.ar/?utm_source=tengolibre&utm_medium=sugerencia&utm_content=tl-tlw",
		"otro dominio":   "https://tulocalweb.com.ar.evil.com/empezar?utm_source=tengolibre&utm_medium=sugerencia&utm_content=tl-tlw",
		"http":           "http://tulocalweb.com.ar/empezar?utm_source=tengolibre&utm_medium=sugerencia&utm_content=tl-tlw",
		"con ref":        "https://tulocalweb.com.ar/empezar?utm_source=tengolibre&utm_medium=sugerencia&utm_content=tl-tlw&ref=x",
		"fuente ajena":   "https://tulocalweb.com.ar/empezar?utm_source=caigo&utm_medium=sugerencia&utm_content=tl-tlw",
		"sin contenido":  "https://tulocalweb.com.ar/empezar?utm_source=tengolibre&utm_medium=sugerencia",
		"otro contenido": "https://tulocalweb.com.ar/empezar?utm_source=tengolibre&utm_medium=sugerencia&utm_content=otra",
		"corredor":       "https://corredor.com.ar/?utm_source=tengolibre&utm_medium=sugerencia&utm_content=tl-tlw",
	}
	for nombre, u := range malos {
		s := ok
		s.URL = u
		if destinoValido(s) {
			t.Errorf("%s: aceptó %s", nombre, u)
		}
	}
}

// El CTA dice lo que pasa al tocar (el destino es el alta de la otra app), sin prometer «gratis», y
// las marcas van con espacios duros para que ningún renglón las corte.
func TestTextosDelCatalogo(t *testing.T) {
	for _, s := range Catalogo {
		for _, x := range []string{s.Titulo, s.CTA} {
			if strings.Contains(strings.ToLower(x), "gratis") {
				t.Errorf("%s promete «gratis»: %q", s.ID, x)
			}
			for _, marca := range []string{"Tu Local Web", "Tengo Libre"} {
				if strings.Contains(x, marca) {
					t.Errorf("%s: %q tiene que ir con espacios duros", s.ID, marca)
				}
			}
		}
		if strings.HasSuffix(strings.Split(s.URL, "?")[0], "/empezar") && !strings.HasPrefix(s.CTA, "Creá") {
			t.Errorf("%s lleva al alta pero el CTA no lo dice: %q", s.ID, s.CTA)
		}
	}
}
