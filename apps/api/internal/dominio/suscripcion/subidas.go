package suscripcion

// Subir de plan (o sumar profesionales extra) a mitad de un período ya pagado: la diferencia por los
// días que faltan se cobra en el momento, con una orden suelta de la cuenta de la plataforma (como
// los packs de WhatsApp). El plan nuevo rige recién cuando Mercado Pago acredita esa orden: ahí
// cambia el monto del preapproval (los cobros siguientes ya son del plan nuevo) y el del negocio.
// Así nadie usa Centro pagando Inicial (subir, usar y volver a bajar antes del cobro).

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/nawesan12/tengolibre/apps/api/internal/integraciones/mercadopago"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/auditoria"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/sesion"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/trabajos"
)

// marcaSubida va en la referencia de la orden: <negocio>-up-<azar>.
const marcaSubida = "-up-"

// La orden de una subida vence en una hora: el monto depende de los días que faltan.
const venceSubida = time.Hour

// ordenSubida crea la orden de la diferencia y devuelve su link de pago. Las subidas pendientes de
// antes se cancelan acá (si igual se pagan, ConciliarSubida las aplica si siguen teniendo sentido).
func (s *Servicio) ordenSubida(ctx context.Context, x *Suscripcion, plan Plan, extras int, nuevo, hoy int64) (string, error) {
	negocio := sesion.De(ctx).NegocioID
	ref := negocio + marcaSubida + mercadopago.Azar()
	var id string
	err := s.Pool.EnNegocio(ctx, negocio, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, "UPDATE suscripcion_subidas SET estado = 'cancelado' WHERE estado = 'pendiente'"); err != nil {
			return err
		}
		return tx.QueryRow(ctx, `INSERT INTO suscripcion_subidas (negocio_id, suscripcion_id, referencia, plan, profesionales_extra, monto_nuevo,
			diferencia, creado_en) VALUES (app.negocio_actual(), $1, $2, $3, $4, $5, $6, $7) RETURNING id::text`,
			x.ID, ref, plan.Clave, extras, nuevo, hoy, s.ahora()).Scan(&id)
	})
	if err != nil {
		return "", err
	}
	titulo := "Tengo Libre — pasar a " + plan.Nombre + " (lo que queda del período)"
	if plan.Clave == x.Plan {
		titulo = "Tengo Libre — profesionales extra (lo que queda del período)"
	}
	o := mercadopago.NuevaOrden(ref, titulo, hoy)
	o.ExpirationTime = mercadopago.Duracion(venceSubida)
	vuelta := s.Sitio + "/panel/suscripcion?resultado=subida"
	o.Config = &mercadopago.ConfigOrden{Online: &mercadopago.ConfigOnline{SuccessURL: vuelta, FailureURL: vuelta, PendingURL: vuelta, AutoReturn: "approved"}}
	c, err := s.cliente().CrearOrden(ctx, s.TokenPlataforma, "subida:"+id, o)
	if err != nil {
		return "", errorMP(err)
	}
	err = s.Pool.EnNegocio(context.WithoutCancel(ctx), negocio, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, "UPDATE suscripcion_subidas SET mp_orden = $2, link = $3 WHERE id = $1", id, c.ID, c.CheckoutURL)
		return err
	})
	return c.CheckoutURL, err
}

// ConciliarSubida trae la orden con el token de la plataforma y, si es una subida pagada, la aplica
// una sola vez: el plan (y los extras) del negocio, la suscripción y el monto del preapproval. Si la
// suscripción ya no está (se canceló o se reemplazó), queda sin_aplicar y se avisa para devolverla.
func (s *Servicio) ConciliarSubida(ctx context.Context, o mercadopago.OrdenLeida) error {
	negocio := negocioDeReferencia(o.ExternalReference)
	if negocio == "" || !strings.Contains(o.ExternalReference, marcaSubida) {
		return nil
	}
	t, ok := o.Relevante()
	var sub, mpID string
	var diferencia int64
	err := s.Pool.EnNegocio(ctx, negocio, func(tx pgx.Tx) error {
		var id, estado string
		err := tx.QueryRow(ctx, "SELECT id::text, estado, suscripcion_id::text, diferencia FROM suscripcion_subidas WHERE referencia = $1",
			o.ExternalReference).Scan(&id, &estado, &sub, &diferencia)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		return err
	})
	if err != nil || sub == "" {
		return err
	}
	pagado := ok && t.Status == "processed" && mercadopago.Pesos(mercadopago.LeerImporte(o.TotalPaidAmount)) >= diferencia
	devuelto := ok && (t.Status == "refunded" || t.Status == "charged_back")
	if !pagado && !devuelto {
		return nil
	}
	// En fila con cambiar y extras: el PUT a Mercado Pago y la base cambian juntos.
	if err := s.tomar(ctx, negocio, sub); err != nil {
		return err
	}
	defer s.soltar(ctx, negocio, sub)
	var monto int64
	err = s.Pool.EnNegocio(ctx, negocio, func(tx pgx.Tx) error {
		var id, estado, plan string
		var extras int
		var nuevo int64
		if err := tx.QueryRow(ctx, `SELECT id::text, estado, plan, profesionales_extra, monto_nuevo FROM suscripcion_subidas
			WHERE referencia = $1 FOR UPDATE`, o.ExternalReference).Scan(&id, &estado, &plan, &extras, &nuevo); err != nil {
			return err
		}
		if devuelto {
			if estado != "acreditado" && estado != "sin_aplicar" {
				return nil
			}
			// Una diferencia devuelta (o un contracargo): el plan de arriba no se pagó. Lo revisa una persona
			// (bajar el plan sin avisar no es la respuesta), y la subida queda marcada.
			s.log().Error("suscripcion: devolvieron (o desconocieron) el pago de una subida de plan; revisar el plan del negocio",
				"negocio", negocio, "orden", o.ID, "estado", t.Status)
			if _, err := tx.Exec(ctx, "UPDATE suscripcion_subidas SET estado = 'devuelto' WHERE id = $1", id); err != nil {
				return err
			}
			return auditoria.Anotar(ctx, tx, auditoria.Registro{Accion: "suscripcion.subida_devuelta", Entidad: "suscripcion", EntidadID: sub,
				Detalle: map[string]any{"orden": o.ID, "estado": t.Status}})
		}
		if estado == "acreditado" || estado == "sin_aplicar" || estado == "devuelto" {
			return nil
		}
		var vale bool
		if err := tx.QueryRow(ctx, `SELECT estado = 'autorizada' AND cancelada_en IS NULL, coalesce(mp_id, '') FROM suscripciones
			WHERE id = $1 FOR UPDATE`, sub).Scan(&vale, &mpID); err != nil {
			return err
		}
		if !vale || mpID == "" {
			s.log().Error("suscripcion: se pagó una subida de plan de una suscripción que ya no está; devolver", "negocio", negocio, "orden", o.ID)
			_, err := tx.Exec(ctx, "UPDATE suscripcion_subidas SET estado = 'sin_aplicar', acreditado_en = $2, mp_orden = coalesce(mp_orden, $3) WHERE id = $1",
				id, s.ahora(), o.ID)
			mpID = ""
			return err
		}
		if _, err := tx.Exec(ctx, "UPDATE suscripcion_subidas SET estado = 'acreditado', acreditado_en = $2, mp_orden = coalesce(mp_orden, $3) WHERE id = $1",
			id, s.ahora(), o.ID); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `UPDATE suscripciones SET plan = $2, profesionales_extra = $3, monto = $4, plan_siguiente = NULL,
			extras_siguiente = NULL, actualizado_en = now() WHERE id = $1`, sub, plan, extras, nuevo); err != nil {
			return err
		}
		if err := ponerPlan(ctx, tx, sub); err != nil {
			return err
		}
		monto = nuevo
		return auditoria.Anotar(ctx, tx, auditoria.Registro{Accion: "suscripcion.plan_cambiado", Entidad: "suscripcion", EntidadID: sub,
			Detalle: map[string]any{"ahora": plan, "extras": extras, "monto": nuevo, "diferencia": diferencia, "orden": o.ID}})
	})
	if err != nil || monto == 0 || mpID == "" {
		return err
	}
	// Los cobros que siguen ya son del plan nuevo. Si Mercado Pago no responde, el webhook reintenta y
	// ConciliarCobro coteja lo que se cobró.
	if err := s.cliente().CambiarMontoPreaprobacion(context.WithoutCancel(ctx), s.TokenPlataforma, mpID, monto); err != nil {
		s.log().Error("suscripcion: subida acreditada, pero no se pudo cambiar el monto en Mercado Pago", "negocio", negocio, "error", err)
		return err
	}
	return nil
}

// RecibirOrdenPlataforma es el receptor de una orden de la cuenta de la plataforma (el webhook de
// cobros la pasa acá): un pack de WhatsApp o la diferencia de una subida de plan.
func (s *Servicio) RecibirOrdenPlataforma(ctx context.Context, a mercadopago.Aviso) error {
	return s.ConciliarOrdenPlataforma(ctx, a.ID())
}

// ConciliarOrdenPlataforma lee la orden con el token de la plataforma (el aviso no es la fuente de
// verdad) y la aplica según su referencia. Es idempotente.
func (s *Servicio) ConciliarOrdenPlataforma(ctx context.Context, orden string) error {
	if s.TokenPlataforma == "" || orden == "" {
		return nil
	}
	o, err := s.cliente().LeerOrden(ctx, s.TokenPlataforma, orden)
	if err != nil {
		return noEncontradoEsNada(err)
	}
	switch {
	case strings.Contains(o.ExternalReference, marcaPack):
		return s.conciliarPack(ctx, o)
	case strings.Contains(o.ExternalReference, marcaSubida):
		return s.ConciliarSubida(ctx, o)
	}
	return nil
}

// ConciliarPendientes repasa las órdenes de la plataforma del negocio fijado que siguen pendientes
// (de las últimas 48 horas): si el aviso de Mercado Pago no llegó, el pago igual se acredita.
func (s *Servicio) ConciliarPendientes(ctx context.Context, negocio string) error {
	var ordenes []string
	err := s.Pool.EnNegocio(ctx, negocio, func(tx pgx.Tx) error {
		filas, err := tx.Query(ctx, `SELECT mp_orden FROM whatsapp_packs WHERE estado IN ('pendiente', 'cancelado') AND mp_orden IS NOT NULL
				AND creado_en > $1
			UNION ALL SELECT mp_orden FROM suscripcion_subidas WHERE estado IN ('pendiente', 'cancelado') AND mp_orden IS NOT NULL
				AND creado_en > $1`, s.ahora().Add(-48*time.Hour))
		if err != nil {
			return err
		}
		ordenes, err = pgx.CollectRows(filas, pgx.RowTo[string])
		return err
	})
	if err != nil {
		return err
	}
	var errs []error
	for _, o := range ordenes {
		if err := s.ConciliarOrdenPlataforma(ctx, o); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// conciliar: POST /v1/panel/suscripcion/conciliar, lo llama la vuelta de Mercado Pago (packs y
// subidas): no depende de que el aviso llegue.
func (s *Servicio) conciliar(ctx context.Context, negocio string) error {
	if s.TokenPlataforma == "" {
		return nil
	}
	return s.ConciliarPendientes(ctx, negocio)
}

// planDeSubida: el plan que lleva una subida pendiente con link (para mostrar «falta pagar»).
type SubidaPendiente struct {
	Plan       string    `json:"plan"`
	Extras     int       `json:"profesionalesExtra"`
	Diferencia int64     `json:"hoy"`
	Despues    int64     `json:"despues"`
	Link       string    `json:"link"`
	Vence      time.Time `json:"vence"`
}

func leerSubidaPendiente(ctx context.Context, tx pgx.Tx, ahora time.Time) (*SubidaPendiente, error) {
	var x SubidaPendiente
	var creado time.Time
	err := tx.QueryRow(ctx, `SELECT plan, profesionales_extra, diferencia, monto_nuevo, link, creado_en FROM suscripcion_subidas
		WHERE estado = 'pendiente' AND link IS NOT NULL AND creado_en > $1 ORDER BY creado_en DESC LIMIT 1`, ahora.Add(-venceSubida)).
		Scan(&x.Plan, &x.Extras, &x.Diferencia, &x.Despues, &x.Link, &creado)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	x.Vence = creado.Add(venceSubida)
	return &x, err
}



// TrabajoOrdenes: cada 10 minutos, las órdenes de la plataforma pendientes de las últimas 48 horas
// (packs y subidas) se vuelven a leer de Mercado Pago. Sin token de la plataforma, no hace nada.
func (s *Servicio) TrabajoOrdenes() trabajos.Trabajo {
	return trabajos.Trabajo{Nombre: "ordenes_plataforma", Cada: 10 * time.Minute, Correr: func(ctx context.Context, ahora time.Time) (any, error) {
		if s.TokenPlataforma == "" {
			return nil, nil
		}
		var ids []string
		err := s.Pool.EnTx(ctx, func(tx pgx.Tx) error {
			filas, err := tx.Query(ctx, "SELECT id::text FROM app.negocios_con_ordenes_pendientes($1) id", ahora.Add(-48*time.Hour))
			if err != nil {
				return err
			}
			ids, err = pgx.CollectRows(filas, pgx.RowTo[string])
			return err
		})
		if err != nil {
			return nil, err
		}
		errores := 0
		for _, id := range ids {
			if err := s.ConciliarPendientes(ctx, id); err != nil {
				errores++
				s.log().Warn("suscripcion: no se pudieron conciliar las órdenes pendientes", "negocio", id, "error", err)
			}
		}
		return map[string]int{"negocios": len(ids), "errores": errores}, nil
	}}
}
