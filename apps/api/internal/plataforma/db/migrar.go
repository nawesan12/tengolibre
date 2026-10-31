package db

import (
	"context"
	"database/sql"
	"io/fs"

	_ "github.com/jackc/pgx/v5/stdlib" // driver "pgx" para database/sql (lo pide goose)
	"github.com/pressly/goose/v3"

	esquema "github.com/nawesan12/tengolibre/apps/api/db"
)

// Migrar aplica las migraciones pendientes con el rol dueño del esquema.
func Migrar(ctx context.Context, migradorURL string) error {
	conn, err := sql.Open("pgx", migradorURL)
	if err != nil {
		return err
	}
	defer conn.Close()
	archivos, err := fs.Sub(esquema.Migraciones, "migraciones")
	if err != nil {
		return err
	}
	// Fuera de orden: varias ramas suman migraciones en paralelo y se mergean en cualquier orden.
	prov, err := goose.NewProvider(goose.DialectPostgres, conn, archivos, goose.WithAllowOutofOrder(true))
	if err != nil {
		return err
	}
	_, err = prov.Up(ctx)
	return err
}
