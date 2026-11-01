package suscripcion

import (
	"context"
	"errors"
	"math"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"

	"github.com/nawesan12/tengolibre/apps/api/internal/integraciones/mercadopago"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/problema"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/sesion"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/web"
)

// Rutas del panel. Ninguna pasa por Escritura: con el panel bloqueado, suscribirse es justamente lo
// que tiene que poder hacerse. Los avisos de Mercado Pago entran por el webhook de cobros (Receptores).
func (s *Servicio) Rutas(mux *http.ServeMux) {
	ses := s.Sesion
	mux.HandleFunc("GET /v1/panel/suscripcion", ses.Panel(sesion.Gestion, web.Envolver(s.leer)))
	mux.HandleFunc("GET /v1/panel/acceso", ses.Panel(sesion.Agenda, web.Envolver(s.acceso)))
	mux.HandleFunc("POST /v1/panel/suscripcion", ses.Panel([]string{sesion.Duenio}, web.Envolver(s.suscribir)))
	mux.HandleFunc("POST /v1/panel/suscripcion/cancelar", ses.Panel([]string{sesion.Duenio}, web.Envolver(s.cancelar)))
	// Cambiar de plan, pagar la cuota que falló, cambiar la tarjeta y reactivar (cambios.go).
	mux.HandleFunc("POST /v1/panel/suscripcion/cambiar", ses.Panel([]string{sesion.Duenio}, web.Envolver(s.cambiar)))
	mux.HandleFunc("POST /v1/panel/suscripcion/pagar", ses.Panel([]string{sesion.Duenio}, web.Envolver(s.reemplazo(pagar))))
	mux.HandleFunc("POST /v1/panel/suscripcion/tarjeta", ses.Panel([]string{sesion.Duenio}, web.Envolver(s.reemplazo(tarjeta))))
	mux.HandleFunc("POST /v1/panel/suscripcion/reactivar", ses.Panel([]string{sesion.Duenio}, web.Envolver(s.reemplazo(reactivar))))
}

func (s *Servicio) leer(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	var out struct {
		Acceso
		NombrePlan  string       `json:"nombrePlan"`
		Catalogo    []Plan       `json:"catalogo"`
		Suscripcion *Suscripcion `json:"suscripcion"`
	}
	err := s.Pool.EnNegocio(ctx, sesion.De(ctx).NegocioID, func(tx pgx.Tx) error {
		var err error
		if out.Acceso, err = LeerAcceso(ctx, tx, s.ahora()); err != nil {
			return err
		}
		if err := LeerCobros(ctx, tx, s.ahora(), &out.Acceso); err != nil {
			return err
		}
		out.Suscripcion, err = ultima(ctx, tx)
		return err
	})
	if err != nil {
		return err
	}
	out.NombrePlan = "Sin plan"
	if p, ok := planDe(out.Plan); ok {
		out.NombrePlan = p.Nombre
	}
	out.Catalogo = Planes
	web.JSON(w, http.StatusOK, out)
	return nil
}

// acceso es lo mínimo que necesita cualquier rol del panel para avisar que está en sólo lectura.
func (s *Servicio) acceso(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	var a Acceso
	err := s.Pool.EnNegocio(ctx, sesion.De(ctx).NegocioID, func(tx pgx.Tx) (err error) {
		if a, err = LeerAcceso(ctx, tx, s.ahora()); err != nil {
			return err
		}
		return LeerCobros(ctx, tx, s.ahora(), &a)
	})
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusOK, a)
	return nil
}

func (s *Servicio) cliente() *mercadopago.Cliente {
	if s.Cliente != nil {
		return s.Cliente
	}
	return &mercadopago.Cliente{}
}

func errorMP(err error) error {
	var emp *mercadopago.ErrorMP
	if errors.As(err, &emp) {
		return &problema.Problema{Estado: http.StatusBadGateway, Codigo: "mercadopago", Titulo: "Mercado Pago no respondió",
			Detalle: "Probá de nuevo en un rato."}
	}
	return err
}

// suscribir crea la suscripción pendiente y devuelve el init_point. El plan cambia recién cuando
// Mercado Pago avisa que quedó autorizada. Lo que queda de la prueba no se cobra.
func (s *Servicio) suscribir(w http.ResponseWriter, r *http.Request) error {
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
	if s.TokenPlataforma == "" {
		return problema.Conflicto("sin_suscripciones", "Las suscripciones todavía no están habilitadas.")
	}
	ses := sesion.De(ctx)
	ref := ses.NegocioID + "-" + mercadopago.Azar()
	var id, email string
	var diasGratis int
	err := s.Pool.EnNegocio(ctx, ses.NegocioID, func(tx pgx.Tx) error {
		a, err := LeerAcceso(ctx, tx, s.ahora())
		if err != nil {
			return err
		}
		var activa bool
		if err := tx.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM suscripciones WHERE estado = 'autorizada')").Scan(&activa); err != nil {
			return err
		}
		if activa {
			return problema.Conflicto("ya_suscripto", "Ya tenés una suscripción activa. Cancelala primero para cambiar de plan.")
		}
		// Un checkout que se abandonó no puede trabar el próximo: queda en el historial como cancelado.
		if _, err := tx.Exec(ctx, `UPDATE suscripciones SET estado = 'cancelada', cancelada_en = $1, motivo = 'Checkout abandonado',
			actualizado_en = now() WHERE estado = 'pendiente'`, s.ahora()); err != nil {
			return err
		}
		if a.PruebaHasta != nil {
			diasGratis = int(math.Ceil(a.PruebaHasta.Sub(s.ahora()).Hours() / 24))
		}
		_ = tx.QueryRow(ctx, "SELECT email FROM miembros WHERE clerk_id = $1", ses.ClerkID).Scan(&email)
		return tx.QueryRow(ctx, `INSERT INTO suscripciones (negocio_id, referencia, plan, monto, email)
			VALUES (app.negocio_actual(), $1, $2, $3, $4) RETURNING id::text`, ref, plan.Clave, plan.Precio, email).Scan(&id)
	})
	if err != nil {
		return err
	}
	p, err := s.cliente().CrearPreaprobacion(ctx, s.TokenPlataforma, mercadopago.NuevaPreaprobacion{
		Motivo: "TengoLibre " + plan.Nombre + " — suscripción mensual", Referencia: ref, Email: email, Monto: plan.Precio,
		DiasGratis: diasGratis, Vuelta: s.Sitio + "/panel/suscripcion?resultado=vuelta",
	})
	if err != nil {
		return errorMP(err)
	}
	err = s.Pool.EnNegocio(ctx, ses.NegocioID, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, "UPDATE suscripciones SET mp_id = $2, link = $3, actualizado_en = now() WHERE id = $1", id, p.ID, p.InitPoint)
		return err
	})
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusCreated, map[string]string{"suscripcionId": id, "url": p.InitPoint})
	return nil
}

// cancelar la suscripción: en Mercado Pago (si llegó a existir allá) y acá. Si el mes ya estaba
// pago, sigue valiendo hasta el próximo cobro.
func (s *Servicio) cancelar(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	var in struct {
		Motivo string `json:"motivo"`
	}
	if r.ContentLength != 0 {
		if err := web.Leer(w, r, &in); err != nil {
			return err
		}
	}
	in.Motivo = strings.TrimSpace(in.Motivo)
	if utf8.RuneCountInString(in.Motivo) > 200 {
		return problema.Invalido("El motivo es muy largo (hasta 200).")
	}
	if in.Motivo == "" {
		in.Motivo = "Cancelada desde el panel"
	}
	negocio := sesion.De(ctx).NegocioID
	var x *Suscripcion
	var mpID *string
	err := s.Pool.EnNegocio(ctx, negocio, func(tx pgx.Tx) error {
		var err error
		if x, err = vigente(ctx, tx, false); err != nil || x == nil {
			return err
		}
		return tx.QueryRow(ctx, "SELECT mp_id FROM suscripciones WHERE id = $1", x.ID).Scan(&mpID)
	})
	if err != nil {
		return err
	}
	if x == nil {
		return problema.NoEncontrado("No hay una suscripción para cancelar.")
	}
	if mpID != nil && s.TokenPlataforma != "" {
		if err := s.cliente().CambiarPreaprobacion(ctx, s.TokenPlataforma, *mpID, "cancelled"); err != nil {
			return errorMP(err)
		}
	}
	err = s.Pool.EnNegocio(ctx, negocio, func(tx pgx.Tx) error {
		return darDeBaja(ctx, tx, x.ID, s.ahora(), in.Motivo)
	})
	if err != nil {
		return err
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}

// darDeBaja: con el mes pago, pausada hasta el próximo cobro; si no, cancelada y el negocio sin plan.
func darDeBaja(ctx context.Context, tx pgx.Tx, id string, ahora time.Time, motivo string) error {
	var proximo *time.Time
	var estado string
	if err := tx.QueryRow(ctx, "SELECT estado, proximo_cobro FROM suscripciones WHERE id = $1 FOR UPDATE", id).Scan(&estado, &proximo); err != nil {
		return err
	}
	if estado == "autorizada" && proximo != nil && proximo.After(ahora) {
		_, err := tx.Exec(ctx, `UPDATE suscripciones SET estado = 'pausada', gracia_hasta = $2, cancelada_en = coalesce(cancelada_en, $3),
			motivo = $4, actualizado_en = now() WHERE id = $1`, id, *proximo, ahora, motivo)
		return err
	}
	if _, err := tx.Exec(ctx, `UPDATE suscripciones SET estado = 'cancelada', cancelada_en = coalesce(cancelada_en, $2), motivo = $3,
		actualizado_en = now() WHERE id = $1`, id, ahora, motivo); err != nil {
		return err
	}
	_, err := tx.Exec(ctx, "UPDATE negocios SET plan = 'sin_plan' WHERE id = app.negocio_actual()")
	return err
}

// Receptores de los avisos de suscripciones, para el webhook de Mercado Pago (lo monta cobros).
func (s *Servicio) Receptores() map[string]mercadopago.Receptor {
	return map[string]mercadopago.Receptor{
		"subscription_preapproval":        func(ctx context.Context, a mercadopago.Aviso) error { return s.ConciliarPreaprobacion(ctx, a.ID()) },
		"subscription_authorized_payment": func(ctx context.Context, a mercadopago.Aviso) error { return s.ConciliarCobro(ctx, a.ID()) },
	}
}

// ConciliarPreaprobacion trae la suscripción de Mercado Pago (la fuente de verdad, con el token de
// la plataforma) y la aplica a su negocio.
func (s *Servicio) ConciliarPreaprobacion(ctx context.Context, id string) error {
	if s.TokenPlataforma == "" {
		return nil
	}
	p, err := s.cliente().LeerPreaprobacion(ctx, s.TokenPlataforma, id)
	if err != nil {
		return noEncontradoEsNada(err)
	}
	negocio := negocioDeReferencia(p.ExternalReference)
	if negocio == "" {
		return nil // no la creó TengoLibre
	}
	ahora := s.ahora()
	var viejaMP *string
	defer func() {
		// La que esta reemplazó ya se canceló acá; en Mercado Pago, lo mejor que se pueda.
		if viejaMP != nil {
			if err := s.cliente().CambiarPreaprobacion(context.WithoutCancel(ctx), s.TokenPlataforma, *viejaMP, "cancelled"); err != nil {
				s.log().Warn("suscripcion: no se pudo cancelar la reemplazada en Mercado Pago", "preapproval", *viejaMP, "error", err)
			}
		}
	}()
	return s.Pool.EnNegocio(ctx, negocio, func(tx pgx.Tx) error {
		var subID, plan string
		var cancelada, proximo *time.Time
		err := tx.QueryRow(ctx, `SELECT id::text, plan, cancelada_en, proximo_cobro FROM suscripciones
			WHERE mp_id = $1 OR referencia = $2 FOR UPDATE`, p.ID, p.ExternalReference).Scan(&subID, &plan, &cancelada, &proximo)
		if errors.Is(err, pgx.ErrNoRows) {
			s.log().Warn("suscripcion: aviso de una que no conocemos", "preapproval", p.ID)
			return nil
		}
		if err != nil {
			return err
		}
		switch p.Status {
		case "authorized":
			if _, err := tx.Exec(ctx, `UPDATE suscripciones SET estado = 'autorizada', mp_id = $2, email = coalesce(nullif($3, ''), email),
				proximo_cobro = coalesce($4, proximo_cobro), gracia_hasta = NULL, actualizado_en = now() WHERE id = $1`,
				subID, p.ID, p.PayerEmail, p.NextPaymentDate); err != nil {
				return err
			}
			if viejaMP, err = reemplazar(ctx, tx, subID, ahora); err != nil {
				return err
			}
			_, err = tx.Exec(ctx, "UPDATE negocios SET plan = $1 WHERE id = app.negocio_actual()", plan)
			return err
		case "paused":
			_, err = tx.Exec(ctx, `UPDATE suscripciones SET estado = 'pausada', gracia_hasta = coalesce(gracia_hasta, $2),
				actualizado_en = now() WHERE id = $1`, subID, ahora.AddDate(0, 0, DiasDeGracia))
			return err
		case "cancelled":
			if p.NextPaymentDate != nil {
				proximo = p.NextPaymentDate
			}
			// Cancelada por la persona con el mes pago: vale hasta el próximo cobro. Si no, se corta.
			if cancelada != nil && proximo != nil && proximo.After(ahora) {
				_, err = tx.Exec(ctx, `UPDATE suscripciones SET estado = 'pausada', gracia_hasta = $2, actualizado_en = now() WHERE id = $1`,
					subID, *proximo)
				return err
			}
			if _, err := tx.Exec(ctx, `UPDATE suscripciones SET estado = 'cancelada', cancelada_en = coalesce(cancelada_en, $2),
				motivo = CASE WHEN motivo = '' THEN 'Cancelada desde Mercado Pago' ELSE motivo END, actualizado_en = now() WHERE id = $1`,
				subID, ahora); err != nil {
				return err
			}
			_, err = tx.Exec(ctx, "UPDATE negocios SET plan = 'sin_plan' WHERE id = app.negocio_actual()")
			return err
		}
		return nil
	})
}

// ConciliarCobro aplica el cobro mensual de una suscripción: aprobado la deja autorizada; rechazado
// la pausa con gracia (el preapproval puede seguir "authorized" mientras Mercado Pago reintenta).
func (s *Servicio) ConciliarCobro(ctx context.Context, id string) error {
	if s.TokenPlataforma == "" {
		return nil
	}
	c, err := s.cliente().LeerCobroAutorizado(ctx, s.TokenPlataforma, id)
	if err != nil {
		return noEncontradoEsNada(err)
	}
	if c.PreapprovalID == "" {
		return nil
	}
	p, err := s.cliente().LeerPreaprobacion(ctx, s.TokenPlataforma, c.PreapprovalID)
	if err != nil {
		return noEncontradoEsNada(err)
	}
	negocio := negocioDeReferencia(p.ExternalReference)
	if negocio == "" {
		return nil
	}
	ahora := s.ahora()
	return s.Pool.EnNegocio(ctx, negocio, func(tx pgx.Tx) error {
		var subID, plan string
		err := tx.QueryRow(ctx, "SELECT id::text, plan FROM suscripciones WHERE mp_id = $1 FOR UPDATE", c.PreapprovalID).Scan(&subID, &plan)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		switch c.Payment.Status {
		case "approved":
			if _, err := tx.Exec(ctx, `UPDATE suscripciones SET estado = 'autorizada', ultimo_cobro = $2, proximo_cobro = coalesce($3, proximo_cobro),
				gracia_hasta = NULL, actualizado_en = now() WHERE id = $1`, subID, ahora, p.NextPaymentDate); err != nil {
				return err
			}
			_, err = tx.Exec(ctx, "UPDATE negocios SET plan = $1 WHERE id = app.negocio_actual()", plan)
			return err
		case "rejected", "cancelled":
			s.log().Warn("suscripcion: el cobro mensual no pasó", "negocio", negocio, "cobro", id)
			_, err = tx.Exec(ctx, `UPDATE suscripciones SET estado = 'pausada', gracia_hasta = coalesce(gracia_hasta, $2),
				actualizado_en = now() WHERE id = $1 AND estado IN ('autorizada', 'pausada')`, subID, ahora.AddDate(0, 0, DiasDeGracia))
			return err
		}
		return nil
	})
}

// noEncontradoEsNada: algo que Mercado Pago no conoce no se va a arreglar reintentando (un 500 haría
// que reintente para siempre). Sólo una caída real pide que vuelva.
func noEncontradoEsNada(err error) error {
	var emp *mercadopago.ErrorMP
	if errors.As(err, &emp) && (emp.Estado == http.StatusNotFound || emp.Estado == http.StatusBadRequest) {
		return nil
	}
	return err
}
