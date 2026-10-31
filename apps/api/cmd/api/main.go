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

	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/admin"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/arranque"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/avisos"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/catalogo"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/clientes"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/cobros"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/directorio"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/dominios"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/espera"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/grupos"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/invitaciones"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/membresias"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/miweb"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/negocios"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/promos"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/reportes"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/resenas"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/suscripcion"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/turnos"
	"github.com/nawesan12/tengolibre/apps/api/internal/integraciones/mercadopago"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/archivos"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/auditoria"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/cifrado"
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
	// Avisos: cada cambio de un turno encola sus mails y campanitas, y un horario liberado avisa a la
	// lista de espera.
	av := &avisos.Servicio{Pool: app, Sesion: ses, AlLiberarse: espera.Liberado}

	// Plata. Vencida la prueba sin suscripción, el panel queda de sólo lectura salvo lo de abajo.
	sus := &suscripcion.Servicio{Pool: app, Sesion: ses, TokenPlataforma: cfg.MPTokenPlataforma, Sitio: cfg.Sitio}
	ses.Escritura = sus.Escritura
	ses.SinChequeoDeEscritura = suscripcion.SinChequeo
	// Sin clave de cifrado, los negocios no pueden conectar Mercado Pago (el resto anda).
	cifra, err := cifrado.Nuevo(cfg.ClaveCifrado)
	if err != nil {
		slog.Warn("cobros: sin clave de cifrado válida, conectar Mercado Pago queda apagado", "error", err)
		cifra = nil
	}
	cob := &cobros.Servicio{Pool: app, Sesion: ses, Cifra: cifra, Sitio: cfg.Sitio,
		Aplicacion:     mercadopago.Aplicacion{ID: cfg.MPAppID, Secreto: cfg.MPSecreto},
		SecretoWebhook: cfg.MPSecretoWebhook, Escritura: sus.Escritura, OtrosAvisos: sus.Receptores(), Avisar: av.Avisar}
	pro := &promos.Servicio{Pool: app, Sesion: ses, Escritura: sus.Escritura}
	mem := &membresias.Servicio{Pool: app, Sesion: ses, Escritura: sus.Escritura}
	go cob.RenovarCada(ctx, 6*time.Hour)
	srv := &http.Server{
		Addr: cfg.Direccion,
		Handler: servidor.Nuevo(servidor.Dependencias{App: app, Version: version, Origenes: cfg.Origenes,
			Dominios: []servidor.Dominio{
				&negocios.Servicio{Pool: app, Sesion: ses},
				&catalogo.Servicio{Pool: app, Sesion: ses},
				// El orden de Extras importa: membresía, después cupón, y el cobro último (sobre el precio final).
				&turnos.Servicio{Pool: app, Sesion: ses, Avisar: av.Avisar, Extras: []turnos.Extra{mem, pro, cob}},
				&clientes.Servicio{Pool: app, Sesion: ses},
				&reportes.Servicio{Pool: app, Sesion: ses},
				av,
				cob, pro, mem, sus,
				&resenas.Servicio{Pool: app, Sesion: ses},
				&espera.Servicio{Pool: app, Sesion: ses},
				&invitaciones.Servicio{Pool: app, Sesion: ses},
				&auditoria.Servicio{Pool: app, Sesion: ses},
				&archivos.Servicio{Pool: app, Sesion: ses, Carpeta: cfg.Archivos, URLBase: cfg.ArchivosURL},
				&miweb.Servicio{Pool: app, Sesion: ses},
				&dominios.Servicio{Pool: app, Sesion: ses, Destino: cfg.DominiosDestino, IP: cfg.VPSIP},
				&directorio.Servicio{Pool: app, Sesion: ses},
				&grupos.Servicio{Pool: app, Sesion: ses},
				&arranque.Servicio{Pool: app, Sesion: ses},
				&admin.Servicio{Pool: app, Sesion: ses, Admins: cfg.Admins, AdminsEmails: cfg.AdminsEmails},
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
