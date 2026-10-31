package main

import (
	"context"
	"regexp"
	"strings"

	"github.com/jackc/pgx/v5"
)

var reDominio = regexp.MustCompile(`^([a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?\.)+[a-z]{2,63}$`)

// BloquesCaddy arma /etc/caddy/sitios/tengolibre-dominios.caddy (calcado de Corredor): un bloque
// explícito por dominio propio verificado (y su www) hacia la web de TengoLibre (127.0.0.1:3400),
// con el HTTPS automático de Caddy (Let's Encrypt). Un bloque con nombre le gana al catch-all
// `https://` de Tu Local Web, y no se usa on_demand ni el ask global, que son de Tu Local Web. Estos
// visitantes entran directo (sin Cloudflare): `ip_real` (snippet de 00-cloudflare.caddy) pisa
// Cf-Connecting-Ip con la IP de la conexión y X-Forwarded-For se borra; si no, cualquiera inventaría
// su IP. La web resuelve el negocio por el Host con GET /v1/host/{host}.
// Lo que no parezca un dominio se saltea: el archivo nunca lleva texto que Caddy interprete.
func BloquesCaddy(dominios []string) string {
	var b strings.Builder
	b.WriteString("# Generado por `tengolibrectl dominios-caddy`: no editar a mano.\n")
	b.WriteString("# Dominios propios de los negocios de TengoLibre → web de TengoLibre (127.0.0.1:3400).\n")
	vistos := map[string]bool{}
	for _, d := range dominios {
		d = strings.ToLower(strings.TrimSpace(d))
		if !reDominio.MatchString(d) || d == "tengolibre.com.ar" || strings.HasSuffix(d, ".tengolibre.com.ar") || vistos[d] {
			continue
		}
		vistos[d] = true
		hosts := d
		if !strings.HasPrefix(d, "www.") {
			hosts += ", www." + d
		}
		b.WriteString("\n" + hosts + " {\n\tencode zstd gzip\n\timport ip_real\n" +
			"\trequest_header -X-Forwarded-For\n\treverse_proxy 127.0.0.1:3400\n}\n")
	}
	return b.String()
}

// dominiosVerificados lee, como migrador, los dominios verificados de negocios activos. El migrador
// no ve nada de otros negocios salvo con app.plataforma puesto (la política `plataforma`).
func dominiosVerificados(ctx context.Context, url string) ([]string, error) {
	conn, err := pgx.Connect(ctx, url)
	if err != nil {
		return nil, err
	}
	defer conn.Close(ctx)
	var out []string
	err = pgx.BeginFunc(ctx, conn, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, "SELECT set_config('app.plataforma', 'si', true)"); err != nil {
			return err
		}
		filas, err := tx.Query(ctx, `SELECT d.nombre FROM dominios d JOIN negocios n ON n.id = d.negocio_id
			WHERE d.estado = 'verificado' AND n.activo ORDER BY d.nombre`)
		if err != nil {
			return err
		}
		out, err = pgx.CollectRows(filas, pgx.RowTo[string])
		return err
	})
	return out, err
}
