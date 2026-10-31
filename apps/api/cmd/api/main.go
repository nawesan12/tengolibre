// Comando api: el servidor HTTP de TengoLibre para el VPS.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/catalogo"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/negocios"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/turnos"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/clerk"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/config"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/db"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/servidor"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/sesion"
)

// version la pone el build: -ldflags "-X main.version=<commit>".
var version = "dev"

func main() {
	if err := correr(); err != nil {
		slog.Error("la api no arrancó", "error", err)
		os.Exit(1)
	}
}

func correr() error {
	cfg, err := config.Leer()
	if err != nil {
		return err
	}
	if err := cfg.Exigir("BaseApp"); err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	app, err := db.Abrir(ctx, cfg.BaseApp)
	if err != nil {
		return err
	}
	defer app.Close()

	ses := &sesion.Servicio{Pool: app, Verificador: verificador(cfg)}
	srv := &http.Server{
		Addr: cfg.Direccion,
		Handler: servidor.Nuevo(servidor.Dependencias{App: app, Version: version, Origenes: cfg.Origenes,
			Dominios: []servidor.Dominio{
				&negocios.Servicio{Pool: app, Sesion: ses},
				&catalogo.Servicio{Pool: app, Sesion: ses},
				&turnos.Servicio{Pool: app, Sesion: ses},
			}}),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       90 * time.Second,
	}
	errs := make(chan error, 1)
	go func() {
		slog.Info("api escuchando", "direccion", cfg.Direccion, "entorno", cfg.Entorno, "version", version)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errs <- err
		}
	}()
	select {
	case err := <-errs:
		return err
	case <-ctx.Done():
	}
	apagar, cancelar := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancelar()
	return srv.Shutdown(apagar)
}

// verificador de sesiones: Clerk si está configurado; en desarrollo sin Clerk, el falso ("dev:user_…");
// si no, ninguno (todo lo que pide sesión responde 401).
func verificador(cfg config.Config) sesion.Verificador {
	if cfg.ClerkSecreto != "" {
		c, err := clerk.Nuevo(cfg.ClerkSecreto, cfg.ClerkPublicable, cfg.Origenes)
		if err == nil {
			return c
		}
		slog.Error("clerk: configuración inválida, ingreso apagado", "error", err)
		return nil
	}
	if cfg.Desarrollo() {
		slog.Warn("sin Clerk: en desarrollo vale el token dev:user_…")
		return sesion.Falso{}
	}
	return nil
}
