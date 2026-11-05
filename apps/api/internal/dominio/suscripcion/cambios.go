package suscripcion

// Lo que se hace con una suscripción ya dada de alta: cambiar de plan (el precio nuevo va al
// preapproval y rige desde el próximo cobro), pagar la cuota que falló, cambiar la tarjeta y
// reactivar una cancelada. Mercado Pago no deja pagar una cuota suelta ni cambiar la tarjeta de un
// preapproval sin un formulario de tarjeta propio, así que esos tres hacen un alta nueva (el link
// de Mercado Pago donde la persona carga la tarjeta) que reemplaza a la vieja: cuando la nueva queda
// autorizada, la vieja se cancela (ver reemplazar).

import (
	"context"
	"math"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/nawesan12/tengolibre/apps/api/internal/integraciones/mercadopago"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/auditoria"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/problema"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/sesion"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/web"
)

func (s *Servicio) sinSuscripciones() error {
	if s.TokenPlataforma == "" {
		return problema.Conflicto("sin_suscripciones", "Las suscripciones todavía no están habilitadas.")
	}
	return nil
}

// ultima: la vigente o, si no hay, la última cancelada (para mostrar «cancelada» y reactivarla).
func ultima(ctx context.Context, tx pgx.Tx) (*Suscripcion, error) {
	x, err := vigente(ctx, tx, false)
	if err != nil || x != nil {
		return x, err
	}
	var c Suscripcion
	err = escanear(tx.QueryRow(ctx, "SELECT "+columnas+" FROM suscripciones WHERE estado = 'cancelada' ORDER BY creado_en DESC LIMIT 1"), &c)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return &c, err
}

// diasHasta: los días que faltan hasta t, redondeados para arriba (0 si ya pasó).
func diasHasta(t *time.Time, ahora time.Time) int {
	if t == nil || !t.After(ahora) {
		return 0
	}
	return int(math.Ceil(t.Sub(ahora).Hours() / 24))
}

// alta crea una suscripción pendiente de plan con su preapproval y devuelve el init_point. reemplaza:
// la que se cancela cuando esta quede autorizada.
func (s *Servicio) alta(ctx context.Context, plan Plan, diasGratis int, reemplaza *string) (string, error) {
	ses := sesion.De(ctx)
	ref := ses.NegocioID + "-" + mercadopago.Azar()
	var id, email string
	err := s.Pool.EnNegocio(ctx, ses.NegocioID, func(tx pgx.Tx) error {
		// Un checkout que se abandonó no puede trabar el próximo: queda en el historial como cancelado.
		if _, err := tx.Exec(ctx, `UPDATE suscripciones SET estado = 'cancelada', cancelada_en = $1, motivo = 'Checkout abandonado',
			actualizado_en = now() WHERE estado = 'pendiente'`, s.ahora()); err != nil {
			return err
		}
		_ = tx.QueryRow(ctx, "SELECT email FROM miembros WHERE clerk_id = $1", ses.ClerkID).Scan(&email)
		return tx.QueryRow(ctx, `INSERT INTO suscripciones (negocio_id, referencia, plan, monto, email, reemplaza)
			VALUES (app.negocio_actual(), $1, $2, $3, $4, $5) RETURNING id::text`, ref, plan.Clave, plan.Precio, email, reemplaza).Scan(&id)
	})
	if err != nil {
		return "", err
	}
	p, err := s.cliente().CrearPreaprobacion(ctx, s.TokenPlataforma, mercadopago.NuevaPreaprobacion{
		Motivo: "TengoLibre " + plan.Nombre + " — suscripción mensual", Referencia: ref, Email: email, Monto: plan.Precio,
		DiasGratis: diasGratis, Vuelta: s.Sitio + "/panel/suscripcion?resultado=vuelta",
	})
	if err != nil {
		return "", errorMP(err)
	}
	err = s.Pool.EnNegocio(ctx, ses.NegocioID, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, "UPDATE suscripciones SET mp_id = $2, link = $3, actualizado_en = now() WHERE id = $1", id, p.ID, p.InitPoint)
		return err
	})
	return p.InitPoint, err
}

// cambiar de plan una suscripción autorizada: el precio nuevo va al preapproval (rige desde el
// próximo cobro) y el plan cambia ya.
func (s *Servicio) cambiar(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	var in struct {
		Plan string `json:"plan"`
	}
	if err := web.Leer(w, r, &in); err != nil {
		return err
	}
	plan, ok := planDe(in.Plan)
	if !ok {
		return problema.Invalido("Elegí el plan inicial o el completo.", problema.Campo{Ruta: "plan", Mensaje: "inválido"})
	}
	if err := s.sinSuscripciones(); err != nil {
		return err
	}
	negocio := sesion.De(ctx).NegocioID
	var x *Suscripcion
	var mpID *string
	err := s.Pool.EnNegocio(ctx, negocio, func(tx pgx.Tx) error {
		var err error
		if x, err = vigente(ctx, tx, false); err != nil || x == nil {
			return err
		}
		if x.Origen == "apple" {
			return ProblemaDeApple()
		}
		return tx.QueryRow(ctx, "SELECT mp_id FROM suscripciones WHERE id = $1", x.ID).Scan(&mpID)
	})
	if err != nil {
		return err
	}
	switch {
	case x == nil || x.Estado != "autorizada" || x.CanceladaEn != nil || mpID == nil:
		return problema.Conflicto("sin_suscripcion", "Para cambiar de plan necesitás una suscripción activa.")
	case x.Plan == plan.Clave:
		return problema.Conflicto("mismo_plan", "Ya estás en el plan "+plan.Nombre+".")
	}
	if err := s.cliente().CambiarMontoPreaprobacion(ctx, s.TokenPlataforma, *mpID, plan.Precio); err != nil {
		return errorMP(err)
	}
	err = s.Pool.EnNegocio(context.WithoutCancel(ctx), negocio, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, "UPDATE suscripciones SET plan = $2, monto = $3, actualizado_en = now() WHERE id = $1",
			x.ID, plan.Clave, plan.Precio); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, "UPDATE negocios SET plan = $1 WHERE id = app.negocio_actual()", plan.Clave); err != nil {
			return err
		}
		if err := auditoria.Anotar(ctx, tx, auditoria.Registro{Accion: "suscripcion.plan_cambiado", Entidad: "suscripcion", EntidadID: x.ID,
			Detalle: map[string]any{"antes": x.Plan, "ahora": plan.Clave, "monto": plan.Precio}}); err != nil {
			return err
		}
		x, err = vigente(ctx, tx, false)
		return err
	})
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusOK, x)
	return nil
}

// reemplazo: pagar, tarjeta y reactivar. elegir dice, con la última suscripción, si se puede, cuántos
// días no se cobran y si reemplaza a esa (si no, es un alta sin nada que cancelar).
func (s *Servicio) reemplazo(elegir func(x *Suscripcion, ahora time.Time) (dias int, reemplaza bool, err error)) web.Manejador {
	return func(w http.ResponseWriter, r *http.Request) error {
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
		var x *Suscripcion
		err := s.Pool.EnNegocio(ctx, sesion.De(ctx).NegocioID, func(tx pgx.Tx) (err error) {
			x, err = ultima(ctx, tx)
			return err
		})
		if err != nil {
			return err
		}
		if x == nil {
			return problema.NoEncontrado("No hay una suscripción. Elegí un plan para suscribirte.")
		}
		if x.Origen == "apple" {
			return ProblemaDeApple()
		}
		dias, reemplaza, err := elegir(x, s.ahora())
		if err != nil {
			return err
		}
		plan, _ := planDe(x.Plan)
		var de *string
		if reemplaza {
			de = &x.ID
		}
		u, err := s.alta(ctx, plan, dias, de)
		if err != nil {
			return err
		}
		web.JSON(w, http.StatusOK, map[string]string{"url": u})
		return nil
	}
}

// fallida: un cobro que no pasó (pausada sin que nadie la cancele).
func fallida(x *Suscripcion) bool { return x.Estado == "pausada" && x.CanceladaEn == nil }

// pagar la cuota que falló: un alta nueva que cobra ya (sin días gratis) y reemplaza a la pausada.
func pagar(x *Suscripcion, _ time.Time) (int, bool, error) {
	if !fallida(x) {
		return 0, false, problema.Conflicto("sin_deuda", "No hay una cuota sin pagar.")
	}
	return 0, true, nil
}

// tarjeta: otro medio de pago. Con el mes pago, el alta nueva empieza a cobrar cuando vence.
func tarjeta(x *Suscripcion, ahora time.Time) (int, bool, error) {
	switch {
	case x.Estado == "autorizada" && x.CanceladaEn == nil:
		return diasHasta(x.ProximoCobro, ahora), true, nil
	case fallida(x):
		return 0, true, nil
	}
	return 0, false, problema.Conflicto("sin_suscripcion", "No hay una suscripción activa a la que cambiarle la tarjeta.")
}

// reactivar una cancelada, con el mismo plan. Si todavía tiene el mes pago, empieza a cobrar al
// terminar.
func reactivar(x *Suscripcion, ahora time.Time) (int, bool, error) {
	switch {
	case x.Estado == "cancelada":
		return 0, false, nil
	case x.Estado == "pausada" && x.CanceladaEn != nil:
		return diasHasta(x.GraciaHasta, ahora), true, nil
	}
	return 0, false, problema.Conflicto("no_cancelada", "Tu suscripción no está cancelada.")
}

// reemplazar: la suscripción id quedó autorizada; si reemplaza a otra, esa se cancela acá (sin
// tocar el plan del negocio) y devuelve su preapproval para cancelarlo en Mercado Pago.
func reemplazar(ctx context.Context, tx pgx.Tx, id string, ahora time.Time) (*string, error) {
	var vieja, mpID *string
	err := tx.QueryRow(ctx, `SELECT v.id::text, v.mp_id FROM suscripciones s JOIN suscripciones v ON v.id = s.reemplaza
		WHERE s.id = $1 AND v.estado IN ('autorizada', 'pausada', 'pendiente')`, id).Scan(&vieja, &mpID)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	_, err = tx.Exec(ctx, `UPDATE suscripciones SET estado = 'cancelada', cancelada_en = coalesce(cancelada_en, $2), gracia_hasta = NULL,
		motivo = 'Reemplazada por un alta nueva', actualizado_en = now() WHERE id = $1`, *vieja, ahora)
	return mpID, err
}
