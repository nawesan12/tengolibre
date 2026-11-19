package db

import (
	"context"
	"database/sql"
	"errors"
	"io/fs"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"

	_ "github.com/jackc/pgx/v5/stdlib" // driver "pgx" para database/sql (lo pide goose)
	"github.com/pressly/goose/v3"

	esquema "github.com/nawesan12/tengolibre/apps/api/db"
)

// Migrar aplica las migraciones pendientes con el rol dueño del esquema. Con lock_timeout: se migra con
// la API y el worker andando, y un ALTER/CREATE POLICY que espera un lock detrás de una transacción
// larga encolaba detrás todas las consultas a esa tabla (el sitio caído). Así falla a los 5 s y se
// reintenta.
func Migrar(ctx context.Context, migradorURL string) error {
	var err error
	for intento := 1; intento <= 3; intento++ {
		if err = migrar(ctx, migradorURL); err == nil || !esperaLock(err) {
			return err
		}
		time.Sleep(time.Duration(intento) * 3 * time.Second)
	}
	return err
}

func esperaLock(err error) bool {
	var pg *pgconn.PgError
	return errors.As(err, &pg) && pg.Code == "55P03"
}

func migrar(ctx context.Context, migradorURL string) error {
	// pgx pasa los parámetros que no conoce de la URL a la sesión: cada conexión sale con su
	// lock_timeout (salvo que la URL ya traiga uno).
	if !strings.Contains(migradorURL, "lock_timeout") {
		if strings.Contains(migradorURL, "?") {
			migradorURL += "&lock_timeout=5s"
		} else {
			migradorURL += "?lock_timeout=5s"
		}
	}
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
