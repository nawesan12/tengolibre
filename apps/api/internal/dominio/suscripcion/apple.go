package suscripcion

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/planes"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/appstore"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/auditoria"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/problema"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/sesion"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/web"
)

// La suscripción por la App Store: quien se suscribe desde el iPhone paga con Apple (compra dentro
// de la app, StoreKit 2); desde la web, con Mercado Pago. Cada una se maneja donde se contrató.
//
// La app manda la transacción firmada después de comprar (POST /v1/panel/suscripcion/apple) y la App
// Store avisa cada renovación, cancelación, falla de cobro o reembolso (POST /v1/webhooks/appstore,
// App Store Server Notifications V2). El negocio sale del appAccountToken, que la app pone al comprar
// con el id del negocio.

// BundleApple de la app de iPhone.
const BundleApple = "ar.com.tengolibre.app"

// ProductosApple: el producto de cada plan y período, el mismo en la App Store y en Google Play
// (grupo de suscripciones «Tengo Libre»). completo.mensual es el plan de antes: se lee como Equipo
// mensual para las suscripciones que ya existían, pero no se vende más.
var ProductosApple = map[string]string{
	"ar.com.tengolibre.inicial.mensual":  "inicial",
	"ar.com.tengolibre.inicial.anual":    "inicial",
	"ar.com.tengolibre.equipo.mensual":   "equipo",
	"ar.com.tengolibre.equipo.anual":     "equipo",
	"ar.com.tengolibre.centro.mensual":   "centro",
	"ar.com.tengolibre.centro.anual":     "centro",
	"ar.com.tengolibre.completo.mensual": "equipo",
}

// ProductoTienda: el plan y el período de un producto de la App Store o de Google Play.
func ProductoTienda(id string) (Plan, string, bool) {
	clave, ok := ProductosApple[id]
	if !ok {
		return Plan{}, "", false
	}
	p, ok := planes.De(clave)
	periodo := "mensual"
	if strings.HasSuffix(id, ".anual") {
		periodo = "anual"
	}
	return p, periodo, ok
}

// ProblemaDeApple: lo que responde Mercado Pago (suscribir, cancelar, cambiar) si la suscripción
// vigente es de la App Store.
func ProblemaDeApple() *problema.Problema {
	return problema.Conflicto("suscripcion_apple",
		"Tu suscripción se paga con la App Store: se cambia o se cancela desde el iPhone (Ajustes › tu nombre › Suscripciones).")
}

func (s *Servicio) verificador() *appstore.Verificador {
	if s.Apple != nil {
		return s.Apple
	}
	return &appstore.Verificador{}
}

// RutasApple: la compra desde la app y las notificaciones de la App Store.
func (s *Servicio) RutasApple(mux *http.ServeMux) {
	mux.HandleFunc("POST /v1/panel/suscripcion/apple", s.Sesion.Panel([]string{sesion.Duenio}, web.Envolver(s.compraApple)))
	mux.HandleFunc("POST /v1/webhooks/appstore", web.Envolver(s.notificacionApple))
}

// compraApple: {"transaccion": "<JWS de StoreKit 2>"} después de comprar o restaurar en el iPhone.
func (s *Servicio) compraApple(w http.ResponseWriter, r *http.Request) error {
	var in struct {
		Transaccion string `json:"transaccion"`
	}
	if err := web.Leer(w, r, &in); err != nil {
		return err
	}
	t, err := s.verificador().LeerTransaccion(in.Transaccion)
	if err != nil {
		return problema.Invalido("La compra no vino firmada por la App Store.", problema.Campo{Ruta: "transaccion", Mensaje: "firma"})
	}
	ctx := r.Context()
	negocio := sesion.De(ctx).NegocioID
	if t.BundleID != BundleApple {
		return problema.Invalido("La compra es de otra app.")
	}
	if !s.aceptaApple(t.Environment) {
		return problema.Invalido("Esa compra es de prueba (sandbox) y no se cobró.", problema.Campo{Ruta: "transaccion", Mensaje: "prueba"})
	}
	if !strings.EqualFold(t.AppAccountToken, negocio) {
		return problema.Conflicto("otro_negocio", "Esa suscripción es de otro negocio. Entrá a ese negocio para verla.")
	}
	var x *Suscripcion
	err = s.Pool.EnNegocio(ctx, negocio, func(tx pgx.Tx) (err error) {
		if err = conciliarApple(ctx, tx, negocio, t, nil, "", s.ahora()); err != nil {
			return err
		}
		x, err = ultima(ctx, tx)
		return err
	})
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusOK, x)
	return nil
}

// notificacionApple: {"signedPayload": "<JWS>"} (App Store Server Notifications V2). Lo que no es
// nuestro o no se reconoce responde 200 igual (si no, la App Store reintenta días).
func (s *Servicio) notificacionApple(w http.ResponseWriter, r *http.Request) error {
	var in struct {
		SignedPayload string `json:"signedPayload"`
	}
	if err := web.Leer(w, r, &in); err != nil {
		return err
	}
	n, err := s.verificador().LeerNotificacion(in.SignedPayload)
	if err != nil {
		return problema.Invalido("La notificación no vino firmada por la App Store.")
	}
	ctx := r.Context()
	if n.Tipo == "TEST" || n.Transaccion == nil || n.BundleID != BundleApple {
		s.log().Info("appstore: notificación sin transacción nuestra", "tipo", n.Tipo, "bundle", n.BundleID)
		w.WriteHeader(http.StatusOK)
		return nil
	}
	if !s.aceptaApple(n.Entorno) || !s.aceptaApple(n.Transaccion.Environment) {
		s.log().Info("appstore: notificación de prueba ignorada", "tipo", n.Tipo, "entorno", n.Entorno)
		w.WriteHeader(http.StatusOK)
		return nil
	}
	negocio := strings.ToLower(n.Transaccion.AppAccountToken)
	if !esUUID.MatchString(negocio) {
		s.log().Warn("appstore: transacción sin negocio", "tipo", n.Tipo, "original", n.Transaccion.OriginalTransactionID)
		w.WriteHeader(http.StatusOK)
		return nil
	}
	err = s.Pool.EnNegocio(ctx, negocio, func(tx pgx.Tx) error {
		return conciliarApple(ctx, tx, negocio, *n.Transaccion, n.Renovacion, n.Tipo+"/"+n.Subtipo, s.ahora())
	})
	if err != nil {
		var p *problema.Problema
		if errors.As(err, &p) {
			s.log().Warn("appstore: no se pudo conciliar", "tipo", n.Tipo, "error", err)
			w.WriteHeader(http.StatusOK)
			return nil
		}
		return err
	}
	s.log().Info("appstore: notificación conciliada", "tipo", n.Tipo, "subtipo", n.Subtipo, "negocio", negocio)
	w.WriteHeader(http.StatusOK)
	return nil
}

// conciliarApple deja la suscripción de Apple de este negocio como dice la transacción (y la
// renovación, si vino): autorizada hasta que vence, pausada en el período de gracia, cancelada si
// venció, se revocó o se reembolsó. evento: "TIPO/SUBTIPO" de la notificación ("" desde la app).
func conciliarApple(ctx context.Context, tx pgx.Tx, negocio string, t appstore.Transaccion, ren *appstore.Renovacion, evento string, ahora time.Time) error {
	plan, periodo, ok := ProductoTienda(t.ProductID)
	if !ok {
		return problema.Invalido("Ese producto de la App Store no es un plan de Tengo Libre.")
	}
	vence := t.Vence()
	estado := "autorizada"
	var gracia *time.Time
	switch {
	case t.RevocationDate != 0 || strings.HasPrefix(evento, "REFUND") || strings.HasPrefix(evento, "REVOKE"):
		estado = "cancelada"
	case ren != nil && ren.GracePeriodExpiresDate != 0 && time.UnixMilli(ren.GracePeriodExpiresDate).After(ahora):
		g := time.UnixMilli(ren.GracePeriodExpiresDate)
		estado, gracia = "pausada", &g
	case vence.IsZero() || vence.Before(ahora.Add(-time.Hour)):
		estado = "cancelada"
	}
	// La renovación automática apagada: sigue valiendo hasta que vence, pero ya no renueva.
	var cancelada *time.Time
	if ren != nil && ren.AutoRenewStatus == 0 {
		cancelada = &ahora
	}
	// Al cambiar de plan en la App Store, el nuevo vale desde la próxima renovación.
	if ren != nil && ren.AutoRenewProductID != "" && ren.AutoRenewProductID != t.ProductID && estado == "autorizada" {
		evento += " (próximo: " + ren.AutoRenewProductID + ")"
	}
	var id string
	var guardadoVence *time.Time
	err := tx.QueryRow(ctx, "SELECT id::text, proximo_cobro FROM suscripciones WHERE apple_original = $1 FOR UPDATE", t.OriginalTransactionID).Scan(&id, &guardadoVence)
	// Las notificaciones pueden llegar desordenadas (la App Store reintenta): una con una transacción
	// que vence antes de lo que ya sabemos es vieja y no pisa (un DID_FAIL_TO_RENEW reintentado después
	// de un DID_RENEW pausaba una suscripción vigente). Lo que termina la suscripción (vencida,
	// reembolsada, revocada) y lo que viene de la app ("") sí valen siempre.
	final := t.RevocationDate != 0 || evento == "" || strings.HasPrefix(evento, "REFUND") || strings.HasPrefix(evento, "REVOKE") ||
		strings.HasPrefix(evento, "EXPIRED") || strings.HasPrefix(evento, "GRACE_PERIOD_EXPIRED")
	if err == nil && !final && guardadoVence != nil && !vence.IsZero() && vence.Before(*guardadoVence) {
		return nil
	}
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		if estado == "cancelada" {
			return nil // nunca la tuvimos activa: nada que hacer
		}
		err = tx.QueryRow(ctx, `INSERT INTO suscripciones (negocio_id, referencia, plan, estado, monto, proximo_cobro, ultimo_cobro,
				gracia_hasta, cancelada_en, origen, apple_original, apple_producto, apple_entorno, motivo)
			VALUES (app.negocio_actual(), $1, $2, $3, $4, $5, $6, $7, $8, 'apple', $9, $10, $11, 'App Store') RETURNING id::text`,
			negocio+"-apple-"+t.OriginalTransactionID, plan.Clave, estado, plan.PrecioDe(periodo, 0), vence, time.UnixMilli(t.PurchaseDate),
			gracia, cancelada, t.OriginalTransactionID, t.ProductID, entornoApple(t.Environment)).Scan(&id)
		if err != nil {
			return err
		}
	case err != nil:
		return err
	default:
		_, err = tx.Exec(ctx, `UPDATE suscripciones SET plan = $2, monto = $3, estado = $4, proximo_cobro = $5, ultimo_cobro = $6,
				gracia_hasta = $7, cancelada_en = CASE WHEN $8::timestamptz IS NULL THEN NULL ELSE coalesce(cancelada_en, $8) END,
				apple_producto = $9, apple_entorno = $10, actualizado_en = now() WHERE id = $1`,
			id, plan.Clave, plan.PrecioDe(periodo, 0), estado, vence, time.UnixMilli(t.PurchaseDate), gracia, cancelada, t.ProductID, entornoApple(t.Environment))
		if err != nil {
			return err
		}
	}
	if _, err := tx.Exec(ctx, "UPDATE suscripciones SET periodo = $2, profesionales_extra = 0 WHERE id = $1", id, periodo); err != nil {
		return err
	}
	// Dada de baja, el plan sale de las que sigan valiendo (puede haber otra, de otro origen).
	if estado == "cancelada" {
		if err := recalcularPlan(ctx, tx, ahora); err != nil {
			return err
		}
	} else if err := ponerPlan(ctx, tx, id); err != nil {
		return err
	}
	accion := "suscripcion.apple"
	if evento == "" {
		evento = "compra desde el iPhone"
	}
	return auditoria.Anotar(ctx, tx, auditoria.Registro{Accion: accion, Entidad: "suscripcion", EntidadID: id,
		Detalle: map[string]any{"evento": evento, "estado": estado, "plan": plan.Clave, "vence": vence}})
}

// aceptaApple: sólo las compras de producción, salvo con SandboxApple (desarrollo, tests, o un rato
// para la revisión de Apple, que compra en sandbox). Lo que no dice "Production" (Sandbox, Xcode, o
// nada) es de prueba.
func (s *Servicio) aceptaApple(entorno string) bool {
	return s.SandboxApple || entornoApple(entorno) == "Production"
}

func entornoApple(e string) string {
	if e == "Production" {
		return "Production"
	}
	return "Sandbox"
}
