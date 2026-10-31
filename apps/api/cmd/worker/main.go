// Comando worker: los trabajos en segundo plano de TengoLibre. Manda la cola de mails (con
// reintentos), encola recordatorios, pedidos de reseña y el resumen diario, marca ausentes en los
// negocios que lo pidieron y purga lo viejo. Corre aparte de la API, con el rol de la app.
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/avisos"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/espera"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/push"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/resenas"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/apns"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/config"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/correo"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/db"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/fcm"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/trabajos"
)

func main() {
	if err := correr(); err != nil {
		slog.Error("el worker no arrancó", "error", err)
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
	// Fuera de desarrollo, sin un SMTP de verdad no arranca (no se mandan mails al log en producción).
	if !cfg.Desarrollo() {
		if err := cfg.ExigirCorreo(); err != nil {
			return err
		}
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	app, err := db.Abrir(ctx, cfg.BaseApp)
	if err != nil {
		return err
	}
	defer app.Close()

	mails := correo.Elegir(cfg.SMTP, cfg.SMTPUsuario, cfg.SMTPClave, cfg.CorreoDe)
	// El mismo Avisar que la API le pasa a turnos: el trabajo de ausentes lo llama por cada turno que
	// marca (ahí se enganchan las penalidades).
	av := &avisos.Servicio{Pool: app, AlLiberarse: espera.Liberado}
	tj := &avisos.Trabajos{Pool: app, DespuesDeMarcar: av.Avisar}
	rs := &resenas.Servicio{Pool: app}
	env := &avisos.Enviador{Pool: app, Correo: mails, Sitio: cfg.Sitio}
	// Push al teléfono: APNs (iPhone) y FCM (Android). Sin configurar, apagado (lo dice el log); una
	// clave que no sirve se anota y deja apagada esa plataforma, sin tirar el worker.
	cliAPNs, err := apns.Desde(cfg.APNsKeyID, cfg.APNsTeamID, cfg.APNsClave, cfg.APNsTopic)
	if err != nil {
		slog.Error("push: la clave de APNs no sirve, no se mandan avisos al iPhone", "error", err)
	}
	cliFCM, err := fcm.Desde(cfg.FCMCuenta)
	if err != nil {
		slog.Error("push: la cuenta de FCM no sirve, no se mandan avisos a Android", "error", err)
	}
	ps := push.Nuevo(app, cliAPNs, cliFCM, slog.Default())
	p := &trabajos.Planificador{Pool: app, Trabajos: append(tj.Lista(env), rs.Trabajo(), ps.Trabajo())}
	slog.Info("worker andando", "entorno", cfg.Entorno, "smtp", cfg.SMTP != "")
	p.Correr(ctx, 5*time.Second)
	return nil
}
