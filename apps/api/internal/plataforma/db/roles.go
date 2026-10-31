package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// Roles del clúster. La app no es dueña de ninguna tabla ni puede saltear RLS.
var Roles = []string{"tengolibre_migrador", "tengolibre_app"}

// Claves por rol para el bootstrap. Vacías en desarrollo y CI (Postgres local con trust).
type Claves map[string]string

// Bootstrap crea (o actualiza) los roles y la base, como superusuario. Es idempotente.
func Bootstrap(ctx context.Context, adminURL, base string, claves Claves) error {
	if err := CrearRoles(ctx, adminURL, claves); err != nil {
		return err
	}
	return CrearBase(ctx, adminURL, base)
}

// CrearRoles crea o actualiza los roles del clúster (sin superusuario ni bypass de RLS).
func CrearRoles(ctx context.Context, adminURL string, claves Claves) error {
	conn, err := pgx.Connect(ctx, adminURL)
	if err != nil {
		return err
	}
	defer conn.Close(ctx)
	for _, r := range Roles {
		var existe bool
		if err := conn.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = $1)", r).Scan(&existe); err != nil {
			return err
		}
		verbo := "CREATE"
		if existe {
			verbo = "ALTER"
		}
		sql := fmt.Sprintf("%s ROLE %s LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOBYPASSRLS", verbo, pgx.Identifier{r}.Sanitize())
		if c := claves[r]; c != "" {
			sql += " PASSWORD " + quoteLiteral(c)
		}
		if _, err := conn.Exec(ctx, sql); err != nil {
			return fmt.Errorf("rol %s: %w", r, err)
		}
	}
	return nil
}

// CrearBase crea la base con el migrador como dueño. Si ya existe con otro dueño, falla: las
// migraciones no podrían crear tablas y el error aparecería lejos de acá.
func CrearBase(ctx context.Context, adminURL, base string) error {
	conn, err := pgx.Connect(ctx, adminURL)
	if err != nil {
		return err
	}
	defer conn.Close(ctx)
	var duenio string
	err = conn.QueryRow(ctx, "SELECT pg_get_userbyid(datdba) FROM pg_database WHERE datname = $1", base).Scan(&duenio)
	if err == nil && duenio != "tengolibre_migrador" {
		return fmt.Errorf("la base %s ya existe y es de %s: tiene que ser de tengolibre_migrador (o usá otra base)", base, duenio)
	}
	if errors.Is(err, pgx.ErrNoRows) {
		if _, err := conn.Exec(ctx, "CREATE DATABASE "+pgx.Identifier{base}.Sanitize()+" OWNER tengolibre_migrador"); err != nil {
			return err
		}
		return nil
	}
	return err
}

func quoteLiteral(s string) string {
	out := []byte{'\''}
	for i := 0; i < len(s); i++ {
		if s[i] == '\'' {
			out = append(out, '\'')
		}
		out = append(out, s[i])
	}
	return string(append(out, '\''))
}
