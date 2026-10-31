package main

import (
	"context"
	"net/url"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"

	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/db/pruebadb"
)

func TestBloquesCaddy(t *testing.T) {
	out := BloquesCaddy([]string{"turnos.lacorte.com.ar", "www.solbarber.com", "x.tengolibre.com.ar", "tengolibre.com.ar",
		"malo.com {\n}", "evil.com\nimport /etc/passwd", "turnos.lacorte.com.ar"})
	for _, quiero := range []string{"turnos.lacorte.com.ar, www.turnos.lacorte.com.ar {", "\nwww.solbarber.com {",
		"reverse_proxy 127.0.0.1:3400", "import ip_real", "request_header -X-Forwarded-For"} {
		if !strings.Contains(out, quiero) {
			t.Errorf("falta %q en:\n%s", quiero, out)
		}
	}
	for _, no := range []string{"tengolibre.com.ar {", "tengolibre.com.ar,", "malo", "evil", "passwd", "on_demand"} {
		if strings.Contains(out, no) {
			t.Errorf("no debería estar %q en:\n%s", no, out)
		}
	}
	if strings.Count(out, "lacorte") != 2 {
		t.Errorf("un dominio repetido tiene que ir una sola vez:\n%s", out)
	}
}

// El migrador ve los dominios verificados de todos los negocios activos (y ninguno más) gracias a
// la política `plataforma`, aunque la RLS esté forzada.
func TestDominiosVerificados(t *testing.T) {
	b := pruebadb.Nueva(t)
	ctx := context.Background()
	uno := pruebadb.CrearNegocio(t, b.App, "uno", "user_aaaaaaaaaa")
	dos := pruebadb.CrearNegocio(t, b.App, "dos", "user_bbbbbbbbbb")
	tres := pruebadb.CrearNegocio(t, b.App, "tres", "user_cccccccccc")
	conn, err := pgx.Connect(ctx, b.AdminURL)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close(ctx)
	if _, err := conn.Exec(ctx, `INSERT INTO dominios (negocio_id, nombre, token, estado) VALUES
		($1, 'uno.com', 'aaaaaaaaaaaaaaaaaaaaaaaa', 'verificado'), ($2, 'dos.com', 'bbbbbbbbbbbbbbbbbbbbbbbb', 'pendiente'),
		($3, 'tres.com', 'cccccccccccccccccccccccc', 'verificado')`, uno, dos, tres); err != nil {
		t.Fatal(err)
	}
	if _, err := conn.Exec(ctx, "UPDATE negocios SET activo = false WHERE id = $1", tres); err != nil {
		t.Fatal(err)
	}
	u, _ := url.Parse(b.AdminURL)
	u.User = url.User("tengolibre_migrador")
	ds, err := dominiosVerificados(ctx, u.String())
	if err != nil {
		t.Fatal(err)
	}
	if len(ds) != 1 || ds[0] != "uno.com" {
		t.Fatalf("dominios: %v", ds)
	}
}
