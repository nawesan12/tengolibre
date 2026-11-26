// Package sugerencias: las sugerencias de la casa (planes/promos-cruzadas-produccion-corregido.md).
// En el Inicio del panel, a la dueña o la encargada, una recomendación discreta de otra app de Grupo
// Vimo que resuelve algo que Tengo Libre no hace. Hoy hay una sola: Tu Local Web («Vendé tus productos
// online») para los negocios con rubro cargado.
//
// Cada app usa sólo su propio dato (el rubro que cargó el negocio, en su base): no se consulta a otras
// apps ni a Clerk. Todo arranca apagado (SUGERENCIAS_ACTIVAS) y se prende el día del lanzamiento.
package sugerencias

import (
	"net/url"
	"slices"
	"strings"
)

// Fuente: la app que muestra (utm_source de cada destino).
const Fuente = "tengolibre"

// Marcas que se pueden sugerir. Lista cerrada: ninguna otra app del grupo entra (lo prueba un test).
var Marcas = []string{"tulocalweb", "tengolibre", "caigo"}

// Dominios a los que puede llevar una sugerencia (https, sin otra cosa que los utm).
var Dominios = []string{"tulocalweb.com.ar", "lugares.caigo.com.ar"}

// Sello: lo que dice arriba del título en las tres plataformas.
const Sello = "De Grupo Vimo"

// Sugerencia tal como la reciben la web y las apps. Imagen y Banner son rutas del sitio (la API les
// suma el dominio): la miniatura cuadrada y el banner 2:1 (sólo la web, desde 768 px).
type Sugerencia struct {
	ID     string `json:"id"`
	Marca  string `json:"marca"`
	Sello  string `json:"sello"`
	Titulo string `json:"titulo"`
	CTA    string `json:"cta"`
	URL    string `json:"url"`
	Imagen string `json:"imagen"`
	Banner string `json:"banner"`
	// Color de la marca sugerida: el fondo de la imagen mientras carga.
	Color string `json:"color"`
}

// Catalogo: la única fuente de los textos. Sólo se nombra lo que la app sugerida ya hace, sin plazos
// de prueba (no se pueden garantizar desde acá).
var Catalogo = []Sugerencia{
	{
		ID:     "tl-tlw",
		Marca:  "tulocalweb",
		Sello:  Sello,
		Titulo: "Vendé tus productos online",
		CTA:    "Conocé Tu Local Web",
		URL:    "https://tulocalweb.com.ar/empezar?utm_source=tengolibre&utm_medium=sugerencia&utm_content=tl-tlw",
		Imagen: "/sugerencias/tulocalweb-mini.jpg",
		Banner: "/sugerencias/tulocalweb-banner.jpg",
		Color:  "#D2582F",
	},
}

// DelCatalogo busca una sugerencia por id.
func DelCatalogo(id string) (Sugerencia, bool) {
	i := slices.IndexFunc(Catalogo, func(s Sugerencia) bool { return s.ID == id })
	if i < 0 {
		return Sugerencia{}, false
	}
	return Catalogo[i], true
}

// destinoValido: https, un dominio de la lista blanca, sin usuario ni fragmento, y sólo los tres utm
// (utm_source de esta app, utm_medium=sugerencia y utm_content con el id). Nunca ref: en Tu Local Web
// es el código de partner y suma comisión.
func destinoValido(s Sugerencia) bool {
	u, err := url.Parse(s.URL)
	if err != nil || u.Scheme != "https" || u.User != nil || u.Fragment != "" || !slices.Contains(Dominios, u.Host) {
		return false
	}
	q := u.Query()
	if len(q) != 3 || q.Get("utm_source") != Fuente || q.Get("utm_medium") != "sugerencia" || q.Get("utm_content") != s.ID {
		return false
	}
	for _, v := range q {
		if len(v) != 1 {
			return false
		}
	}
	return strings.HasPrefix(s.Imagen, "/sugerencias/") && strings.HasPrefix(s.Banner, "/sugerencias/")
}
