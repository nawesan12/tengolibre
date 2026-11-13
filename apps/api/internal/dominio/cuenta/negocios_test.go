package cuenta_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/cuenta"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/db/pruebadb"
)

// conDatos le carga a un negocio un cliente con un turno atendido, la seña cobrada por Mercado
// Pago, el cobro en la caja y una suscripción paga a TengoLibre (en el estado pedido).
func conDatos(t *testing.T, c *pgx.Conn, negocio, suscripcion string) {
	t.Helper()
	cli := uno[string](t, c, "INSERT INTO clientes (negocio_id, nombre, telefono, email) VALUES ($1, 'Juana', '1111111111', 'juana@x.com') RETURNING id::text", negocio)
	turno := uno[string](t, c, `INSERT INTO turnos (negocio_id, codigo, servicio, profesional, cliente_id, inicio, fin, precio, estado, origen, nota)
		VALUES ($1, replace(gen_random_uuid()::text, '-', ''), 'Corte', 'Ana', $2, now() - interval '200 days', now() - interval '200 days' + interval '30 minutes',
			8000, 'atendido', 'web', 'Es alérgica') RETURNING id::text`, negocio, cli)
	exec(t, c, `INSERT INTO pagos (negocio_id, turno_id, referencia, monto, modo, estado, mp_pago)
		VALUES ($1, $2, replace(gen_random_uuid()::text, '-', ''), 2000, 'fija', 'aprobado', 123456)`, negocio, turno)
	exec(t, c, `INSERT INTO caja_cobros (negocio_id, turno_id, servicio, sena, total, medio, estado, cobrado_en)
		VALUES ($1, $2, 8000, 2000, 6000, 'efectivo', 'cobrado', now() - interval '200 days')`, negocio, turno)
	exec(t, c, `INSERT INTO suscripciones (negocio_id, referencia, mp_id, plan, estado, email, monto, ultimo_cobro)
		VALUES ($1, replace(gen_random_uuid()::text, '-', ''), gen_random_uuid()::text, 'inicial', $2, 'duenia@x.com', 15000, now() - interval '120 days')`,
		negocio, suscripcion)
}

func TestPurgaDeNegociosCerrados(t *testing.T) {
	a, _ := armar(t)
	ctx := context.Background()
	cerrado := pruebadb.CrearNegocio(t, a.Bases.App, "cerrado", ana)
	retenido := pruebadb.CrearNegocio(t, a.Bases.App, "retenido", berta)
	abierto := pruebadb.CrearNegocio(t, a.Bases.App, "abierto", carla)
	reabierto := pruebadb.CrearNegocio(t, a.Bases.App, "reabierto", carla)
	admin(t, a, func(c *pgx.Conn) {
		conDatos(t, c, cerrado, "cancelada")
		conDatos(t, c, retenido, "autorizada")
		conDatos(t, c, abierto, "cancelada")
		exec(t, c, "UPDATE negocios SET activo = false WHERE id = ANY ($1::uuid[])", []string{cerrado, retenido, reabierto})
		// Reabrir vacía la fecha del cierre.
		exec(t, c, "UPDATE negocios SET activo = true WHERE id = $1", reabierto)
		if uno[bool](t, c, "SELECT cerrado_en IS NOT NULL FROM negocios WHERE id = $1", reabierto) {
			t.Error("el negocio reabierto sigue con fecha de cierre")
		}
		if !uno[bool](t, c, "SELECT cerrado_en IS NOT NULL FROM negocios WHERE id = $1", cerrado) {
			t.Error("el cierre no anotó la fecha")
		}
	})
	carpeta := t.TempDir()
	for _, n := range []string{cerrado, abierto} {
		if err := os.MkdirAll(filepath.Join(carpeta, n), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(carpeta, n, "logo.png"), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	// Antes de los 90 días, nada.
	listar := func(ahora time.Time) []cuenta.NegocioParaPurgar {
		t.Helper()
		var ns []cuenta.NegocioParaPurgar
		if err := a.Bases.App.EnTx(ctx, func(tx pgx.Tx) error {
			var err error
			ns, err = cuenta.ParaPurgar(ctx, tx, ahora)
			return err
		}); err != nil {
			t.Fatal(err)
		}
		return ns
	}
	if ns := listar(time.Now().AddDate(0, 0, 89)); len(ns) != 0 {
		t.Fatalf("antes de los 90 días: %+v", ns)
	}
	if ids, err := cuenta.PurgarNegocios(ctx, a.Bases.App, time.Now().AddDate(0, 0, 89), carpeta); err != nil || len(ids) != 0 {
		t.Fatalf("purga antes de tiempo: %v %v", ids, err)
	}

	// Listar muestra los dos cerrados (uno retenido por la suscripción que cobra) y no borra nada.
	despues := time.Now().AddDate(0, 0, 91)
	ns := listar(despues)
	if len(ns) != 2 {
		t.Fatalf("para purgar: %+v", ns)
	}
	for _, n := range ns {
		if n.Clientes != 1 || n.Turnos != 1 || n.Cobros != 3 || n.Retenido != (n.ID == retenido) {
			t.Errorf("para purgar: %+v", n)
		}
	}
	admin(t, a, func(c *pgx.Conn) {
		if n := uno[int](t, c, "SELECT count(*) FROM negocios"); n != 4 {
			t.Fatalf("listar borró algo: quedan %d negocios", n)
		}
	})

	ids, err := cuenta.PurgarNegocios(ctx, a.Bases.App, despues, carpeta)
	if err != nil || len(ids) != 1 || ids[0] != cerrado {
		t.Fatalf("purga: %v %v", ids, err)
	}
	admin(t, a, func(c *pgx.Conn) {
		if n := uno[int](t, c, "SELECT count(*) FROM negocios WHERE id = $1", cerrado); n != 0 {
			t.Error("el negocio cerrado sigue")
		}
		for _, tabla := range []string{"clientes", "turnos", "pagos", "caja_cobros", "suscripciones", "miembros"} {
			if n := uno[int](t, c, "SELECT count(*) FROM "+tabla+" WHERE negocio_id = $1", cerrado); n != 0 {
				t.Errorf("%s del negocio purgado: %d", tabla, n)
			}
		}
		for _, n := range []string{retenido, abierto} {
			if uno[int](t, c, "SELECT count(*) FROM clientes WHERE negocio_id = $1", n) != 1 {
				t.Errorf("se tocó un negocio que no tocaba: %s", n)
			}
		}
		// Los cobros quedan archivados, sin datos de nadie.
		var origenes string
		if err := c.QueryRow(ctx, `SELECT string_agg(origen || ':' || monto, ',' ORDER BY origen) FROM auth.cobros_archivados WHERE negocio_id = $1`,
			cerrado).Scan(&origenes); err != nil {
			t.Fatal(err)
		}
		if origenes != "caja:6000,sena:2000,suscripcion:15000" {
			t.Errorf("cobros archivados: %q", origenes)
		}
		if n := uno[int](t, c, `SELECT count(*) FROM auth.cobros_archivados a WHERE a::text ILIKE '%juana%' OR a::text ILIKE '%duenia%'`); n != 0 {
			t.Errorf("hay datos personales en el archivo: %d", n)
		}
	})
	if _, err := os.Stat(filepath.Join(carpeta, cerrado)); !os.IsNotExist(err) {
		t.Errorf("las imágenes del negocio purgado siguen: %v", err)
	}
	if _, err := os.Stat(filepath.Join(carpeta, abierto, "logo.png")); err != nil {
		t.Errorf("se borraron imágenes de un negocio abierto: %v", err)
	}
	// La app no lee el archivo de cobros.
	err = a.Bases.App.EnTx(ctx, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, "SELECT count(*) FROM auth.cobros_archivados")
		return err
	})
	if err == nil {
		t.Error("la app no tendría que poder leer auth.cobros_archivados")
	}
}
