// Comando worker: los trabajos en segundo plano de TengoLibre. Manda la cola de mails (con
// reintentos), encola recordatorios, pedidos de reseña, el resumen diario, los recordatorios automáticos
// (cumpleaños, «hace mucho que no venís») y el reporte mensual, marca ausentes en los negocios que lo
// pidieron, manda el push y los WhatsApp (confirmaciones), ofrece los huecos de la lista de espera y
// purga lo viejo. Corre aparte de la API, con el rol de la app.
package main

import (
	"context"
	"log/slog"
	"maps"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/avisos"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/confirmaciones"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/espera"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/push"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/recordatorios"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/reportes"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/resenas"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/apns"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/config"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/correo"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/db"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/fcm"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/trabajos"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/webpush"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/whatsapp"
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
	// Los recordatorios automáticos y el reporte mensual se arman en sus dominios.
	armadores := recordatorios.Armadores()
	maps.Copy(armadores, reportes.Armadores())
	env := &avisos.Enviador{Pool: app, Correo: mails, Sitio: cfg.Sitio, Armadores: armadores}
	rec := &recordatorios.Servicio{Pool: app, Sitio: cfg.Sitio}
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
	// Push web del panel (VAPID): las mismas notificaciones, al navegador.
	cliWeb, err := webpush.Desde(cfg.VAPIDPublica, cfg.VAPIDPrivada, cfg.VAPIDSujeto)
	if err != nil {
		slog.Error("push: las claves VAPID no sirven, no se mandan avisos al navegador", "error", err)
	}
	ps := push.Nuevo(app, cliAPNs, cliFCM, slog.Default()).ConWeb(cliWeb)
	// WhatsApp (confirmación del día antes, recordatorio, respuestas). Sin configurar, sin trabajos.
	wa := confirmaciones.Nuevos(app, whatsapp.Desde(cfg.WhatsAppToken, cfg.WhatsAppTelefonoID, cfg.WhatsAppVersion), cfg.Sitio, slog.Default())
	// Ofrecer el hueco solo: cuando vence el link de una persona sin respuesta, pasa a la siguiente.
	es := &espera.Servicio{Pool: app}
	lista := append(tj.Lista(env), rs.Trabajo(), ps.Trabajo(), es.Trabajo(), rec.Trabajo(), reportes.TrabajoMensual(app))
	p := &trabajos.Planificador{Pool: app, Trabajos: append(lista, wa.Lista()...)}
	slog.Info("worker andando", "entorno", cfg.Entorno, "smtp", cfg.SMTP != "")
	p.Correr(ctx, 5*time.Second)
	return nil
}
