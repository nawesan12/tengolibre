package suscripcion

// Los packs de recordatorios por WhatsApp (100 por $5.900): un pago suelto con Checkout Pro desde la
// cuenta de la plataforma (una orden, como las señas pero con el token de Tengo Libre). Cuando
// Mercado Pago avisa que se pagó, se suman al saldo del negocio (no vencen). El aviso entra por el
// webhook de cobros: una orden cuyo vendedor no es un negocio conectado viene a ConciliarPack.

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/planes"
	"github.com/nawesan12/tengolibre/apps/api/internal/integraciones/mercadopago"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/auditoria"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/sesion"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/web"
)

// marcaPack va en la referencia de la orden: <negocio>-wa-<azar>.
const marcaPack = "-wa-"

// comprarPack crea la orden del pack y devuelve el link de pago ({"packId", "url"}). Si ya hay una
// pendiente de las últimas 24 horas con link, devuelve esa: un doble clic no arma dos órdenes (que se
// podían pagar las dos) y no se acumulan órdenes contra Mercado Pago.
func (s *Servicio) comprarPack(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	if r.ContentLength != 0 {
		var vacio struct{}
		if err := web.Leer(w, r, &vacio); err != nil {
			return err
		}
	}
	if err := s.sinSuscripciones(); err != nil {
		return err
	}
	negocio := sesion.De(ctx).NegocioID
	ref := negocio + marcaPack + mercadopago.Azar()
	var id string
	var link *string
	err := s.Pool.EnNegocio(ctx, negocio, func(tx pgx.Tx) error {
		// En fila por negocio: dos pedidos a la vez no arman dos órdenes.
		if _, err := tx.Exec(ctx, "SELECT pg_advisory_xact_lock(hashtext('pack:' || app.negocio_actual()::text))"); err != nil {
			return err
		}
		desde := s.ahora().Add(-24 * time.Hour)
		if _, err := tx.Exec(ctx, `UPDATE whatsapp_packs SET estado = 'cancelado' WHERE estado = 'pendiente' AND creado_en < $1`, desde); err != nil {
			return err
		}
		// Sin link todavía (el otro pedido sigue esperando a Mercado Pago, o falló): se arma la orden con su
		// misma referencia y clave de idempotencia; con una nueva, el pago no encontraba el pack.
		err := tx.QueryRow(ctx, `SELECT id::text, link, referencia FROM whatsapp_packs WHERE estado = 'pendiente' AND creado_en >= $1
			ORDER BY creado_en DESC LIMIT 1`, desde).Scan(&id, &link, &ref)
		if err == nil || !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		return tx.QueryRow(ctx, `INSERT INTO whatsapp_packs (negocio_id, referencia, cantidad, monto, creado_en)
			VALUES (app.negocio_actual(), $1, $2, $3, $4) RETURNING id::text`, ref, planes.PackWhatsAppCantidad, planes.PackWhatsAppPrecio, s.ahora()).Scan(&id)
	})
	if err != nil {
		return err
	}
	if link != nil {
		web.JSON(w, http.StatusCreated, map[string]string{"packId": id, "url": *link})
		return nil
	}
	o := mercadopago.NuevaOrden(ref, "Tengo Libre — 100 recordatorios por WhatsApp", planes.PackWhatsAppPrecio)
	o.ExpirationTime = mercadopago.Duracion(24 * time.Hour)
	vuelta := s.Sitio + "/panel/suscripcion?resultado=whatsapp"
	o.Config = &mercadopago.ConfigOrden{Online: &mercadopago.ConfigOnline{SuccessURL: vuelta, FailureURL: vuelta, PendingURL: vuelta, AutoReturn: "approved"}}
	c, err := s.cliente().CrearOrden(ctx, s.TokenPlataforma, "pack:"+id, o)
	if err != nil {
		return errorMP(err)
	}
	err = s.Pool.EnNegocio(context.WithoutCancel(ctx), negocio, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, "UPDATE whatsapp_packs SET mp_orden = $2, link = $3 WHERE id = $1", id, c.ID, c.CheckoutURL)
		return err
	})
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusCreated, map[string]string{"packId": id, "url": c.CheckoutURL})
	return nil
}

// conciliarPack: si la orden es un pack pagado, lo acredita una sola vez (aunque ya se hubiera dado
// por abandonado). Si Mercado Pago la devolvió o hubo un contracargo, el pack pasa a devuelto y se
// descuenta del saldo lo que quede (lo ya mandado no se puede sacar).
func (s *Servicio) conciliarPack(ctx context.Context, o mercadopago.OrdenLeida) error {
	negocio := negocioDeReferencia(o.ExternalReference)
	if negocio == "" || !strings.Contains(o.ExternalReference, marcaPack) {
		return nil
	}
	t, ok := o.Relevante()
	pagado := ok && t.Status == "processed" && mercadopago.Pesos(mercadopago.LeerImporte(o.TotalPaidAmount)) >= planes.PackWhatsAppPrecio
	devuelto := ok && (t.Status == "refunded" || t.Status == "charged_back")
	return s.Pool.EnNegocio(ctx, negocio, func(tx pgx.Tx) error {
		var id, estado string
		var cantidad int
		err := tx.QueryRow(ctx, "SELECT id::text, estado, cantidad FROM whatsapp_packs WHERE referencia = $1 FOR UPDATE", o.ExternalReference).
			Scan(&id, &estado, &cantidad)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		if devuelto && estado == "acreditado" {
			if _, err := tx.Exec(ctx, "UPDATE whatsapp_packs SET estado = 'devuelto' WHERE id = $1", id); err != nil {
				return err
			}
			if _, err := tx.Exec(ctx, "UPDATE negocios SET whatsapp_saldo = greatest(whatsapp_saldo - $1, 0) WHERE id = app.negocio_actual()", cantidad); err != nil {
				return err
			}
			return auditoria.Anotar(ctx, tx, auditoria.Registro{Accion: "suscripcion.pack_whatsapp_devuelto", Entidad: "whatsapp_pack", EntidadID: id,
				Detalle: map[string]any{"cantidad": cantidad, "orden": o.ID, "estado": t.Status}})
		}
		if estado == "acreditado" || estado == "devuelto" || !pagado {
			return nil
		}
		if _, err := tx.Exec(ctx, "UPDATE whatsapp_packs SET estado = 'acreditado', acreditado_en = $2, mp_orden = coalesce(mp_orden, $3) WHERE id = $1",
			id, s.ahora(), o.ID); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, "UPDATE negocios SET whatsapp_saldo = whatsapp_saldo + $1 WHERE id = app.negocio_actual()", cantidad); err != nil {
			return err
		}
		return auditoria.Anotar(ctx, tx, auditoria.Registro{Accion: "suscripcion.pack_whatsapp", Entidad: "whatsapp_pack", EntidadID: id,
			Detalle: map[string]any{"cantidad": cantidad, "orden": o.ID}})
	})
}
