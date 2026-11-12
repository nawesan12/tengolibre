// Comando tengolibrectl: tareas de operación de la base.
//
//	tengolibrectl bootstrap   crea roles y base (superusuario: TENGOLIBRE_BASE_DE_DATOS_ADMIN)
//	tengolibrectl migrar      aplica las migraciones (TENGOLIBRE_BASE_DE_DATOS_MIGRADOR)
//	tengolibrectl dominios-caddy
//	                          escribe en stdout los bloques de Caddy de los dominios propios
//	                          verificados (los instala infra/vps/dominios.sh, que valida y vuelve atrás)
//	tengolibrectl recuperar-cuenta <email o user_…>
//	                          deshace la baja de una cuenta borrada hace menos de 30 días (la persona
//	                          escribió a hola@): vuelven sus membresías y los negocios que la baja cerró
//
// Las claves de los roles salen de TENGOLIBRE_CLAVE_<ROL> (vacías en desarrollo).
package main

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/config"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/db"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "uso: tengolibrectl bootstrap|migrar|dominios-caddy|recuperar-cuenta <email o user_…>")
		os.Exit(2)
	}
	if err := correr(os.Args[1]); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func correr(cmd string) error {
	cfg, err := config.Leer()
	if err != nil {
		return err
	}
	ctx := context.Background()
	switch cmd {
	case "bootstrap":
		if err := cfg.Exigir("BaseAdmin", "BaseMigrador"); err != nil {
			return err
		}
		u, err := url.Parse(cfg.BaseMigrador)
		if err != nil {
			return err
		}
		claves := db.Claves{}
		for _, r := range db.Roles {
			claves[r] = os.Getenv("TENGOLIBRE_CLAVE_" + strings.ToUpper(strings.TrimPrefix(r, "tengolibre_")))
		}
		if err := db.Bootstrap(ctx, cfg.BaseAdmin, strings.TrimPrefix(u.Path, "/"), claves); err != nil {
			return err
		}
		fmt.Println("roles y base listos")
	case "migrar":
		if err := cfg.Exigir("BaseMigrador"); err != nil {
			return err
		}
		if err := db.Migrar(ctx, cfg.BaseMigrador); err != nil {
			return err
		}
		fmt.Println("migraciones aplicadas")
	case "dominios-caddy":
		if err := cfg.Exigir("BaseMigrador"); err != nil {
			return err
		}
		ds, err := dominiosVerificados(ctx, cfg.BaseMigrador)
		if err != nil {
			return err
		}
		fmt.Print(BloquesCaddy(ds))
	case "recuperar-cuenta":
		if err := cfg.Exigir("BaseMigrador"); err != nil {
			return err
		}
		if len(os.Args) < 3 || strings.TrimSpace(os.Args[2]) == "" {
			return fmt.Errorf("falta el email o el clerk_id de la cuenta")
		}
		id, err := recuperarCuenta(ctx, cfg.BaseMigrador, strings.TrimSpace(os.Args[2]))
		if err != nil {
			return err
		}
		if id == "" {
			return fmt.Errorf("no hay una cuenta borrada sin purgar con %q", os.Args[2])
		}
		fmt.Println("cuenta recuperada:", id)
	default:
		return fmt.Errorf("comando desconocido: %s", cmd)
	}
	return nil
}

// recuperarCuenta corre app.recuperar_cuenta como el migrador (la app no tiene permiso). Devuelve el
// clerk_id recuperado, o vacío si no había una baja que se pudiera deshacer.
func recuperarCuenta(ctx context.Context, url, quien string) (string, error) {
	conn, err := pgx.Connect(ctx, url)
	if err != nil {
		return "", err
	}
	defer conn.Close(ctx)
	var id *string
	if err := conn.QueryRow(ctx, "SELECT app.recuperar_cuenta($1)", quien).Scan(&id); err != nil {
		return "", err
	}
	if id == nil {
		return "", nil
	}
	return *id, nil
}
