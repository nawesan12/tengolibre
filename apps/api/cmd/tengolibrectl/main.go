// Comando tengolibrectl: tareas de operación de la base.
//
//	tengolibrectl bootstrap   crea roles y base (superusuario: TENGOLIBRE_BASE_DE_DATOS_ADMIN)
//	tengolibrectl migrar      aplica las migraciones (TENGOLIBRE_BASE_DE_DATOS_MIGRADOR)
//	tengolibrectl dominios-caddy
//	                          escribe en stdout los bloques de Caddy de los dominios propios
//	                          verificados (los instala infra/vps/dominios.sh, que valida y vuelve atrás)
//
// Las claves de los roles salen de TENGOLIBRE_CLAVE_<ROL> (vacías en desarrollo).
package main

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/config"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/db"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "uso: tengolibrectl bootstrap|migrar|dominios-caddy")
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
	default:
		return fmt.Errorf("comando desconocido: %s", cmd)
	}
	return nil
}
