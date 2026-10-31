// Package config lee la configuración de la API del entorno. Fuera de desarrollo, lo que falte
// hace que el proceso no arranque, en vez de arrancar a medias.
package config

import (
	"fmt"
	"os"
	"strings"
)

// Config de la API.
type Config struct {
	Entorno   string // desarrollo | staging | produccion
	Direccion string
	// URLs de Postgres por rol.
	BaseApp, BaseMigrador, BaseAdmin string
	// Sitio público (links de los mails, CORS).
	Sitio string
	// Clerk (ingreso de Grupo Vimo): clave secreta y publicable. Sin clave secreta, apagado.
	ClerkSecreto, ClerkPublicable string
	// Orígenes del front (CORS y `azp` de Clerk). Vacío: el sitio.
	Origenes []string
}

// Desarrollo dice si se está corriendo en local.
func (c Config) Desarrollo() bool { return c.Entorno == "desarrollo" }

// Leer arma la configuración desde las variables TENGOLIBRE_*.
func Leer() (Config, error) {
	c := Config{
		Entorno:         valor("TENGOLIBRE_ENTORNO", "desarrollo"),
		Direccion:       valor("TENGOLIBRE_DIRECCION", ":4200"),
		BaseApp:         os.Getenv("TENGOLIBRE_BASE_DE_DATOS"),
		BaseMigrador:    os.Getenv("TENGOLIBRE_BASE_DE_DATOS_MIGRADOR"),
		BaseAdmin:       os.Getenv("TENGOLIBRE_BASE_DE_DATOS_ADMIN"),
		Sitio:           valor("TENGOLIBRE_SITIO", "https://tengolibre.com.ar"),
		ClerkSecreto:    os.Getenv("TENGOLIBRE_CLERK_SECRETO"),
		ClerkPublicable: os.Getenv("TENGOLIBRE_CLERK_PUBLICABLE"),
		Origenes:        lista(os.Getenv("TENGOLIBRE_ORIGENES")),
	}
	switch c.Entorno {
	case "desarrollo", "staging", "produccion":
	default:
		return c, fmt.Errorf("TENGOLIBRE_ENTORNO=%q: tiene que ser desarrollo, staging o produccion", c.Entorno)
	}
	if c.Desarrollo() {
		// Postgres de infra/compose.dev.yml. La base de la app es tengolibre_dev (la crea el bootstrap
		// con el migrador como dueño); el admin entra a la base por defecto del contenedor.
		local := "postgres://%s@127.0.0.1:5446/%s?sslmode=disable"
		c.BaseApp = conDefecto(c.BaseApp, fmt.Sprintf(local, "tengolibre_app", "tengolibre_dev"))
		c.BaseMigrador = conDefecto(c.BaseMigrador, fmt.Sprintf(local, "tengolibre_migrador", "tengolibre_dev"))
		c.BaseAdmin = conDefecto(c.BaseAdmin, fmt.Sprintf(local, "tengolibre", "tengolibre"))
		c.Sitio = conDefecto(os.Getenv("TENGOLIBRE_SITIO"), "http://localhost:5173")
	}
	if len(c.Origenes) == 0 {
		c.Origenes = []string{c.Sitio}
	}
	return c, nil
}

// lista separa por comas y saca los vacíos.
func lista(v string) []string {
	var out []string
	for _, x := range strings.Split(v, ",") {
		if x = strings.TrimSpace(x); x != "" {
			out = append(out, x)
		}
	}
	return out
}

// Exigir falla si falta alguna de las variables pedidas (por nombre de campo).
func (c Config) Exigir(campos ...string) error {
	vals := map[string]string{"BaseApp": c.BaseApp, "BaseMigrador": c.BaseMigrador, "BaseAdmin": c.BaseAdmin}
	var faltan []string
	for _, f := range campos {
		if vals[f] == "" {
			faltan = append(faltan, f)
		}
	}
	if len(faltan) > 0 {
		return fmt.Errorf("falta configurar: %s", strings.Join(faltan, ", "))
	}
	return nil
}

func valor(k, def string) string { return conDefecto(os.Getenv(k), def) }

func conDefecto(v, def string) string {
	if v == "" {
		return def
	}
	return v
}
