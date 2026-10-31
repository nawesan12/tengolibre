package db_test

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"

	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/db"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/db/pruebadb"
)

// Toda tabla del esquema public tiene RLS habilitada y forzada, y si tiene negocio_id, la política
// por negocio. Una tabla nueva sin aislamiento rompe este test.
func TestTodaTablaTieneAislamiento(t *testing.T) {
	b := pruebadb.Nueva(t)
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, b.AdminURL)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close(ctx)
	filas, err := conn.Query(ctx, `
		SELECT c.relname, c.relrowsecurity, c.relforcerowsecurity,
		       EXISTS (SELECT 1 FROM pg_attribute a WHERE a.attrelid = c.oid AND a.attname = 'negocio_id' AND NOT a.attisdropped),
		       EXISTS (SELECT 1 FROM pg_policy p WHERE p.polrelid = c.oid AND p.polname = 'negocio')
		FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE n.nspname = 'public' AND c.relkind IN ('r', 'p') AND c.relname <> 'goose_db_version'`)
	if err != nil {
		t.Fatal(err)
	}
	defer filas.Close()
	vistas := 0
	for filas.Next() {
		var tabla string
		var rls, forzada, conNegocio, politica bool
		if err := filas.Scan(&tabla, &rls, &forzada, &conNegocio, &politica); err != nil {
			t.Fatal(err)
		}
		vistas++
		if !rls || !forzada {
			t.Errorf("%s: RLS habilitada=%v forzada=%v; usá SELECT app.aplicar_rls('%s')", tabla, rls, forzada, tabla)
		}
		if conNegocio && !politica {
			t.Errorf("%s tiene negocio_id y no tiene la política 'negocio'", tabla)
		}
	}
	if vistas < 10 {
		t.Fatalf("el catálogo devolvió %d tablas: la consulta está mal", vistas)
	}
}

// La app no puede saltear RLS ni ser dueña de nada.
func TestElRolDeLaAppNoTienePrivilegios(t *testing.T) {
	b := pruebadb.Nueva(t)
	ctx := context.Background()
	var super, bypass bool
	if err := b.App.QueryRow(ctx, "SELECT rolsuper, rolbypassrls FROM pg_roles WHERE rolname = current_user").Scan(&super, &bypass); err != nil {
		t.Fatal(err)
	}
	if super || bypass {
		t.Fatalf("tengolibre_app: superusuario=%v bypassrls=%v", super, bypass)
	}
}

// Un negocio no ve los datos de otro, y sin negocio fijado no se ve nada.
func TestUnNegocioNoVeAOtro(t *testing.T) {
	b := pruebadb.Nueva(t)
	ctx := context.Background()
	a := pruebadb.CrearNegocio(t, b.App, "uno", "user_aaaaaaaaaa")
	o := pruebadb.CrearNegocio(t, b.App, "dos", "user_bbbbbbbbbb")
	for _, id := range []string{a, o} {
		err := b.App.EnNegocio(ctx, id, func(tx pgx.Tx) error {
			_, err := tx.Exec(ctx, "INSERT INTO categorias (negocio_id, nombre) VALUES ($1, 'Cortes')", id)
			return err
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	contar := func(id string) (n int) {
		err := b.App.EnNegocio(ctx, id, func(tx pgx.Tx) error {
			return tx.QueryRow(ctx, "SELECT count(*) FROM categorias").Scan(&n)
		})
		if err != nil {
			t.Fatal(err)
		}
		return n
	}
	if contar(a) != 1 || contar(o) != 1 {
		t.Fatal("cada negocio tiene que ver sólo su categoría")
	}
	var n int
	_ = b.App.EnTx(ctx, func(tx pgx.Tx) error { return tx.QueryRow(ctx, "SELECT count(*) FROM categorias").Scan(&n) })
	if n != 0 {
		t.Fatalf("sin negocio se ven %d categorías", n)
	}
	// Escribir en otro negocio con el propio fijado falla.
	err := b.App.EnNegocio(ctx, a, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, "INSERT INTO categorias (negocio_id, nombre) VALUES ($1, 'Intruso')", o)
		return err
	})
	if err == nil {
		t.Fatal("pudo escribir en otro negocio")
	}
	if err := b.App.EnNegocio(ctx, "", func(pgx.Tx) error { return nil }); err != db.ErrSinNegocio {
		t.Fatal("EnNegocio sin negocio tiene que fallar")
	}
}
