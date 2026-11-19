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

// comprarPack crea la orden del pack y devuelve el link de pago ({"url"}).
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
	err := s.Pool.EnNegocio(ctx, negocio, func(tx pgx.Tx) error {
		// Un checkout que se abandonó queda cancelado: no se acumulan pendientes.
		if _, err := tx.Exec(ctx, `UPDATE whatsapp_packs SET estado = 'cancelado' WHERE estado = 'pendiente' AND creado_en < $1`,
			s.ahora().Add(-24*time.Hour)); err != nil {
			return err
		}
		return tx.QueryRow(ctx, `INSERT INTO whatsapp_packs (negocio_id, referencia, cantidad, monto, creado_en)
			VALUES (app.negocio_actual(), $1, $2, $3, $4) RETURNING id::text`, ref, planes.PackWhatsAppCantidad, planes.PackWhatsAppPrecio, s.ahora()).Scan(&id)
	})
	if err != nil {
		return err
	}
	o := mercadopago.NuevaOrden(ref, "Tengo Libre — 100 recordatorios por WhatsApp", planes.PackWhatsAppPrecio)
	o.ExpirationTime = mercadopago.Duracion(24 * time.Hour)
	vuelta := s.Sitio + "/panel/suscripcion?resultado=whatsapp"
	o.Config = &mercadopago.ConfigOrden{Online: &mercadopago.ConfigOnline{SuccessURL: vuelta, FailureURL: vuelta, PendingURL: vuelta, AutoReturn: "approved"}}
	c, err := s.cliente().CrearOrden(ctx, s.TokenPlataforma, "pack:"+id, o)
	if err != nil {
		return errorMP(err)
	}
	err = s.Pool.EnNegocio(ctx, negocio, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, "UPDATE whatsapp_packs SET mp_orden = $2, link = $3 WHERE id = $1", id, c.ID, c.CheckoutURL)
		return err
	})
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusCreated, map[string]string{"packId": id, "url": c.CheckoutURL})
	return nil
}

// RecibirOrdenPlataforma es el receptor de una orden de la cuenta de la plataforma (el webhook de
// cobros la pasa acá cuando el vendedor no es un negocio conectado).
func (s *Servicio) RecibirOrdenPlataforma(ctx context.Context, a mercadopago.Aviso) error {
	return s.ConciliarPack(ctx, a.ID())
}

// ConciliarPack trae la orden con el token de la plataforma (el aviso no es la fuente de verdad) y,
// si es un pack pagado, lo acredita una sola vez.
func (s *Servicio) ConciliarPack(ctx context.Context, orden string) error {
	if s.TokenPlataforma == "" || orden == "" {
		return nil
	}
	o, err := s.cliente().LeerOrden(ctx, s.TokenPlataforma, orden)
	if err != nil {
		return noEncontradoEsNada(err)
	}
	negocio := negocioDeReferencia(o.ExternalReference)
	if negocio == "" || !strings.Contains(o.ExternalReference, marcaPack) {
		return nil
	}
	t, ok := o.Relevante()
	pagado := ok && t.Status == "processed" && mercadopago.Pesos(mercadopago.LeerImporte(o.TotalPaidAmount)) >= planes.PackWhatsAppPrecio
	return s.Pool.EnNegocio(ctx, negocio, func(tx pgx.Tx) error {
		var id, estado string
		var cantidad int
		err := tx.QueryRow(ctx, "SELECT id::text, estado, cantidad FROM whatsapp_packs WHERE referencia = $1 FOR UPDATE", o.ExternalReference).
			Scan(&id, &estado, &cantidad)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if err != nil || estado == "acreditado" || !pagado {
			return err
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
