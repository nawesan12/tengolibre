// Package pruebadb arma una base de Postgres nueva por test: roles, base propia y migraciones.
// Los tests de datos corren contra Postgres de verdad, nunca contra un doble.
//
// Usa TENGOLIBRE_BASE_DE_DATOS_TEST (superusuario). Sin Postgres, el test se saltea en local y
// falla en CI.
package pruebadb

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/url"
	"os"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5"

	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/db"
)

const porDefecto = "postgres://tengolibre@127.0.0.1:5446/tengolibre?sslmode=disable"

// Bases son las conexiones de cada rol a la base del test.
type Bases struct {
	App *db.Pool
	// AdminURL es la URL de superusuario a la base del test (para inspeccionar el catálogo).
	AdminURL string
}

var rolesListos sync.Once

// Nueva crea la base del test y la borra al terminar.
func Nueva(t testing.TB) *Bases {
	t.Helper()
	ctx := context.Background()
	admin := os.Getenv("TENGOLIBRE_BASE_DE_DATOS_TEST")
	if admin == "" {
		admin = porDefecto
	}
	conn, err := pgx.Connect(ctx, admin)
	if err != nil {
		if os.Getenv("CI") != "" {
			t.Fatalf("sin Postgres en CI: %v", err)
		}
		t.Skipf("sin Postgres local (docker compose -f infra/compose.dev.yml up -d): %v", err)
	}
	defer conn.Close(ctx)

	// sync.Once cubre un paquete, pero `go test ./...` corre los paquetes en procesos paralelos que
	// crean los mismos roles a la vez ("tuple concurrently updated"): un advisory lock los pone en fila.
	var errRoles error
	rolesListos.Do(func() {
		if _, errRoles = conn.Exec(ctx, "SELECT pg_advisory_lock(735200)"); errRoles != nil {
			return
		}
		defer func() { _, _ = conn.Exec(ctx, "SELECT pg_advisory_unlock(735200)") }()
		errRoles = db.CrearRoles(ctx, admin, nil)
	})
	if errRoles != nil {
		t.Fatal(errRoles)
	}

	b := make([]byte, 6)
	_, _ = rand.Read(b)
	nombre := "prueba_" + hex.EncodeToString(b)
	if err := db.CrearBase(ctx, admin, nombre); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		c, err := pgx.Connect(context.Background(), admin)
		if err != nil {
			return
		}
		defer c.Close(context.Background())
		_, _ = c.Exec(context.Background(), "DROP DATABASE "+pgx.Identifier{nombre}.Sanitize()+" WITH (FORCE)")
	})

	if err := db.Migrar(ctx, conRol(t, admin, "tengolibre_migrador", nombre)); err != nil {
		t.Fatalf("migraciones: %v", err)
	}
	abrir := func(rol string) *db.Pool {
		p, err := db.Abrir(ctx, conRol(t, admin, rol, nombre))
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(p.Close)
		return p
	}
	u, _ := url.Parse(admin)
	u.Path = "/" + nombre
	return &Bases{
		App:      abrir("tengolibre_app"),
		AdminURL: u.String(),
	}
}

func conRol(t testing.TB, admin, rol, base string) string {
	t.Helper()
	u, err := url.Parse(admin)
	if err != nil {
		t.Fatal(err)
	}
	u.User = url.User(rol)
	u.Path = "/" + base
	return u.String()
}

// CrearNegocio da de alta un negocio con el rol de la app (dueña: clerkID) y devuelve su id.
func CrearNegocio(t testing.TB, app *db.Pool, slug, clerkID string) string {
	t.Helper()
	var id string
	err := app.EnTx(context.Background(), func(tx pgx.Tx) error {
		return tx.QueryRow(context.Background(), "SELECT app.crear_negocio($1, $2, $3, $4, $5)",
			slug, "Negocio "+slug, clerkID, slug+"@prueba.com", "Dueña "+slug).Scan(&id)
	})
	if err != nil {
		t.Fatal(err)
	}
	return id
}
