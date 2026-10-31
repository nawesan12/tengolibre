package confirmaciones

import (
	"context"
	"crypto/subtle"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/avisos"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/auditoria"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/db"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/problema"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/web"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/whatsapp"
)

// verificarWebhook: al suscribir el webhook, Meta manda hub.mode=subscribe, el token que se cargó y
// un desafío que hay que devolver tal cual.
func (s *Servicio) verificarWebhook(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	token := q.Get("hub.verify_token")
	if s.Verificacion == "" || q.Get("hub.mode") != "subscribe" ||
		subtle.ConstantTimeCompare([]byte(token), []byte(s.Verificacion)) != 1 {
		problema.Escribir(w, problema.SinPermiso("El token de verificación no coincide."))
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = io.WriteString(w, q.Get("hub.challenge"))
}

// webhook: los botones que tocaron los clientes y los estados de lo que mandamos. Responde 200 en
// cuanto lo procesó; un error de la base es 500 y Meta lo vuelve a mandar (todo es idempotente).
func (s *Servicio) webhook(w http.ResponseWriter, r *http.Request) error {
	if s.SecretoApp == "" {
		return problema.NoEncontrado("El webhook de WhatsApp no está configurado.")
	}
	cuerpo, err := io.ReadAll(http.MaxBytesReader(w, r.Body, web.MaxCuerpo))
	if err != nil {
		return problema.Invalido("El cuerpo es demasiado grande.")
	}
	if !whatsapp.FirmaValida(s.SecretoApp, cuerpo, r.Header.Get("X-Hub-Signature-256")) {
		return &problema.Problema{Estado: http.StatusUnauthorized, Codigo: "firma_invalida", Titulo: "Firma inválida",
			Detalle: "X-Hub-Signature-256 no corresponde al cuerpo."}
	}
	rs, es, err := whatsapp.Interpretar(cuerpo)
	if err != nil {
		return problema.Invalido("El cuerpo no es un webhook de WhatsApp.")
	}
	ctx := r.Context()
	var errs []error
	for _, x := range rs {
		if s.TelefonoID != "" && x.TelefonoID != "" && x.TelefonoID != s.TelefonoID {
			continue
		}
		if err := s.Responder(ctx, x); err != nil {
			errs = append(errs, err)
		}
	}
	for _, x := range es {
		if s.TelefonoID != "" && x.TelefonoID != "" && x.TelefonoID != s.TelefonoID {
			continue
		}
		if err := s.CambioDeEstado(ctx, x); err != nil {
			errs = append(errs, err)
		}
	}
	if err := errors.Join(errs...); err != nil {
		return err
	}
	web.JSON(w, http.StatusOK, map[string]bool{"ok": true})
	return nil
}

// accion saca qué tocó y el token del mensaje. Sin payload (un botón viejo, o un texto con el mismo
// título), vale el texto del botón y el mensaje respondido.
func accion(x whatsapp.Respuesta) (string, string) {
	if a, token, ok := strings.Cut(x.Payload, ":"); ok && (a == Confirmo || a == Cambiar) {
		return a, token
	}
	switch strings.ToLower(strings.TrimSpace(x.Texto)) {
	case "confirmo":
		return Confirmo, ""
	case "necesito cambiarlo":
		return Cambiar, ""
	}
	return "", ""
}

// negocioDe: del token del botón o del wamid al negocio (nil: no es nuestro).
func negocioDe(ctx context.Context, p *db.Pool, token, wamid string) (*string, error) {
	var n *string
	err := p.EnTx(ctx, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, "SELECT app.negocio_de_whatsapp($1, $2)::text", token, wamid).Scan(&n)
	})
	return n, err
}

// Responder procesa un botón tocado: confirma el turno o lo marca para llamar, avisa a la campanita
// y deja en la cola la respuesta para el cliente.
func (s *Servicio) Responder(ctx context.Context, x whatsapp.Respuesta) error {
	acc, token := accion(x)
	if acc == "" {
		return nil
	}
	negocio, err := negocioDe(ctx, s.Pool, token, x.Contexto)
	if err != nil || negocio == nil {
		if negocio == nil && err == nil {
			slog.Info("whatsapp: respuesta a un mensaje que no conocemos", "contexto", x.Contexto)
		}
		return err
	}
	ahora := s.ahora()
	return s.Pool.EnNegocio(ctx, *negocio, func(tx pgx.Tx) error {
		var mid, tipo, para string
		var turnoID, respuesta, respuestaID *string
		err := tx.QueryRow(ctx, `SELECT id::text, tipo, para, turno_id::text, respuesta, respuesta_id FROM whatsapp_mensajes
			WHERE ($1 <> '' AND token = $1) OR ($2 <> '' AND wamid = $2) ORDER BY ($1 <> '' AND token = $1) DESC LIMIT 1 FOR UPDATE`,
			token, x.Contexto).Scan(&mid, &tipo, &para, &turnoID, &respuesta, &respuestaID)
		if err != nil {
			return err
		}
		// Meta repite webhooks, y el cliente puede tocar dos veces el mismo botón.
		if tipo != "confirmacion" || turnoID == nil || (respuestaID != nil && *respuestaID == x.ID) ||
			(respuesta != nil && *respuesta == acc) {
			return nil
		}
		if _, err := tx.Exec(ctx, `UPDATE whatsapp_mensajes SET respuesta = $2, respuesta_id = nullif($3, ''), respondido_en = $4
			WHERE id = $1`, mid, acc, x.ID, ahora); err != nil {
			return err
		}
		var id string
		d, err := leerTurno(ctx, tx, "t.id = $1", *turnoID, &id)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		if (d.Estado != "pendiente" && d.Estado != "confirmado") || !d.Inicio.After(ahora) {
			return encolarRespuesta(ctx, tx, id, para, textoNoVigente(s.Sitio, d.Slug), ahora)
		}
		if acc == Confirmo {
			if _, err := tx.Exec(ctx, `UPDATE turnos SET estado = 'confirmado', confirmado_por = 'whatsapp', confirmado_en = $2,
				cambio_pedido_en = NULL, llamar_desde = NULL, llamar_motivo = NULL, actualizado_en = now() WHERE id = $1`, id, ahora); err != nil {
				return err
			}
			if err := avisos.Notificar(ctx, tx, avisos.Notificacion{Tipo: NotiConfirmacion, TurnoID: id,
				Titulo: d.Cliente + " confirmó por WhatsApp", Texto: conQuien(d) + ", el " + cuando(d) + "."}); err != nil {
				return err
			}
			if err := auditoria.Anotar(ctx, tx, auditoria.Registro{Accion: "turno.confirmado_whatsapp", Entidad: "turno", EntidadID: id}); err != nil {
				return err
			}
			return encolarRespuesta(ctx, tx, id, para, textoConfirmado(s.Sitio, d.Codigo), ahora)
		}
		if _, err := tx.Exec(ctx, `UPDATE turnos SET cambio_pedido_en = $2, llamar_desde = $2, llamar_motivo = 'cambio',
			actualizado_en = now() WHERE id = $1`, id, ahora); err != nil {
			return err
		}
		texto := conQuien(d) + ", el " + cuando(d) + "."
		if d.Telefono != "" {
			texto = "Llamalo al " + d.Telefono + ": " + texto
		}
		if err := avisos.Notificar(ctx, tx, avisos.Notificacion{Tipo: NotiCambioPedido, TurnoID: id,
			Titulo: d.Cliente + " pidió cambiar su turno", Texto: texto}); err != nil {
			return err
		}
		if err := auditoria.Anotar(ctx, tx, auditoria.Registro{Accion: "turno.cambio_pedido", Entidad: "turno", EntidadID: id}); err != nil {
			return err
		}
		return encolarRespuesta(ctx, tx, id, para, textoCambio(s.Sitio, d.Negocio, d.Codigo), ahora)
	})
}

// CambioDeEstado anota entregado o leído y, si falló, lo da por fallido (como un envío que falla).
func (s *Servicio) CambioDeEstado(ctx context.Context, x whatsapp.Estado) error {
	if x.ID == "" {
		return nil
	}
	negocio, err := negocioDe(ctx, s.Pool, "", x.ID)
	if err != nil || negocio == nil {
		return err
	}
	cuando := x.Cuando
	if cuando.IsZero() {
		cuando = s.ahora()
	}
	return s.Pool.EnNegocio(ctx, *negocio, func(tx pgx.Tx) error {
		switch x.Estado {
		case "delivered":
			_, err := tx.Exec(ctx, "UPDATE whatsapp_mensajes SET entregado_en = coalesce(entregado_en, $2) WHERE wamid = $1", x.ID, cuando)
			return err
		case "read":
			_, err := tx.Exec(ctx, `UPDATE whatsapp_mensajes SET entregado_en = coalesce(entregado_en, $2), leido_en = coalesce(leido_en, $2)
				WHERE wamid = $1`, x.ID, cuando)
			return err
		case "failed":
			var id string
			if err := tx.QueryRow(ctx, "SELECT id::text FROM whatsapp_mensajes WHERE wamid = $1 AND estado = 'enviado' FOR UPDATE", x.ID).
				Scan(&id); errors.Is(err, pgx.ErrNoRows) {
				return nil
			} else if err != nil {
				return err
			}
			return fallido(ctx, tx, id, x.Error, s.ahora())
		}
		return nil
	})
}

// fallido da el mensaje por perdido. Si era la confirmación, el turno queda para llamar; si era la
// confirmación o el recordatorio, avisa a la campanita.
func fallido(ctx context.Context, tx pgx.Tx, id string, causa error, ahora time.Time) error {
	msj := "falló sin detalle"
	if causa != nil {
		msj = causa.Error()
	}
	if len(msj) > 1000 {
		msj = msj[:1000]
	}
	var tipo, para string
	var turnoID *string
	if err := tx.QueryRow(ctx, `UPDATE whatsapp_mensajes SET estado = 'fallido', ultimo_error = $2 WHERE id = $1
		RETURNING tipo, para, turno_id::text`, id, msj).Scan(&tipo, &para, &turnoID); err != nil {
		return err
	}
	if tipo == "respuesta" || turnoID == nil {
		return nil
	}
	if tipo == "confirmacion" {
		if _, err := tx.Exec(ctx, `UPDATE turnos SET llamar_desde = $2, llamar_motivo = 'no_llego', actualizado_en = now()
			WHERE id = $1 AND llamar_desde IS NULL AND estado IN ('pendiente', 'confirmado')`, *turnoID, ahora); err != nil {
			return err
		}
	}
	var cliente string
	if err := tx.QueryRow(ctx, "SELECT c.nombre FROM turnos t JOIN clientes c ON c.id = t.cliente_id WHERE t.id = $1", *turnoID).
		Scan(&cliente); err != nil {
		return err
	}
	texto := "Revisá su teléfono (" + para + ")."
	if errors.Is(causa, whatsapp.ErrDestinoInvalido) {
		texto = "Ese número (" + para + ") no recibe WhatsApp."
	}
	if tipo == "confirmacion" {
		texto += " Llamalo para confirmar."
	}
	return avisos.Notificar(ctx, tx, avisos.Notificacion{Tipo: avisos.NotiAvisoFallido, TurnoID: *turnoID,
		Titulo: "No le llegó el WhatsApp a " + cliente, Texto: texto})
}
