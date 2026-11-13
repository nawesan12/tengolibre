package suscripcion

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/auditoria"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/googleplay"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/problema"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/sesion"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/trabajos"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/web"
)

// La suscripción por Google Play: quien se suscribe desde Android paga con Google (Play Billing),
// como desde el iPhone con la App Store. Cada una se maneja donde se contrató.
//
// La app manda el token de compra después de comprar o restaurar (POST /v1/panel/suscripcion/google);
// acá se consulta a Google (purchases.subscriptionsv2) y se concilia. Google avisa cada renovación,
// cancelación o falla de cobro por Pub/Sub (RTDN, POST /v1/webhooks/google-play?clave=…): se vuelve a
// consultar el token. El negocio sale del obfuscatedAccountId, que la app pone al comprar.

// PaqueteAndroid de la app de Android (los productos son los mismos que en la App Store).
const PaqueteAndroid = "ar.com.tengolibre.app"

// Consultor es lo que hace falta de Google Play (*googleplay.Cliente lo cumple; en los tests, uno falso).
type Consultor interface {
	Consultar(ctx context.Context, paquete, tokenCompra string) (*googleplay.Suscripcion, error)
}

// ProblemaDeGoogle: lo que responde Mercado Pago (suscribir, cancelar, cambiar) si la suscripción
// vigente es de Google Play.
func ProblemaDeGoogle() *problema.Problema {
	return problema.Conflicto("suscripcion_google",
		"Tu suscripción se paga con Google Play: se cambia o se cancela desde Android (Play Store › Pagos y suscripciones).")
}

// ProblemaDeTienda: el de Apple o el de Google si la suscripción vigente se paga en una tienda.
func ProblemaDeTienda(origen string) *problema.Problema {
	switch origen {
	case "apple":
		return ProblemaDeApple()
	case "google":
		return ProblemaDeGoogle()
	}
	return nil
}

// RutasGoogle: la compra desde la app y las notificaciones de Google Play.
func (s *Servicio) RutasGoogle(mux *http.ServeMux) {
	mux.HandleFunc("POST /v1/panel/suscripcion/google", s.Sesion.Panel([]string{sesion.Duenio}, web.Envolver(s.compraGoogle)))
	mux.HandleFunc("POST /v1/webhooks/google-play", web.Envolver(s.notificacionGoogle))
}

func (s *Servicio) consultor() (Consultor, bool) {
	if s.Google == nil {
		return nil, false
	}
	return s.Google, true
}

// compraGoogle: {"producto": "ar.com.tengolibre.completo.mensual", "token": "<purchaseToken>"}.
func (s *Servicio) compraGoogle(w http.ResponseWriter, r *http.Request) error {
	var in struct {
		Producto string `json:"producto"`
		Token    string `json:"token"`
	}
	if err := web.Leer(w, r, &in); err != nil {
		return err
	}
	in.Token = strings.TrimSpace(in.Token)
	if in.Token == "" || len(in.Token) > 4096 {
		return problema.Invalido("Falta el token de la compra.", problema.Campo{Ruta: "token", Mensaje: "obligatorio"})
	}
	g, ok := s.consultor()
	if !ok {
		return problema.Conflicto("google_apagado", "Todavía no podemos confirmar compras de Google Play. Probá más tarde.")
	}
	ctx := r.Context()
	negocio := sesion.De(ctx).NegocioID
	sus, err := g.Consultar(ctx, PaqueteAndroid, in.Token)
	if errors.Is(err, googleplay.ErrNoEncontrada) {
		return problema.Invalido("Google Play no reconoce esa compra.", problema.Campo{Ruta: "token", Mensaje: "desconocido"})
	}
	if err != nil {
		return err
	}
	if !strings.EqualFold(sus.Identificadores.Cuenta, negocio) {
		return problema.Conflicto("otro_negocio", "Esa suscripción es de otro negocio. Entrá a ese negocio para verla.")
	}
	var x *Suscripcion
	err = s.Pool.EnNegocio(ctx, negocio, func(tx pgx.Tx) (err error) {
		if err = conciliarGoogle(ctx, tx, negocio, in.Token, *sus, "", s.ahora()); err != nil {
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

// notificacionGoogle: el push de Pub/Sub {"message": {"data": base64(DeveloperNotification)}}. Lo que
// no es nuestro o no se reconoce responde 200 igual (si no, Pub/Sub reintenta).
func (s *Servicio) notificacionGoogle(w http.ResponseWriter, r *http.Request) error {
	if s.ClaveGoogle == "" || subtle.ConstantTimeCompare([]byte(r.URL.Query().Get("clave")), []byte(s.ClaveGoogle)) != 1 {
		return problema.NoEncontrado("No encontramos esa página.")
	}
	var in struct {
		Message struct {
			Data string `json:"data"`
		} `json:"message"`
	}
	if err := web.Leer(w, r, &in); err != nil {
		return err
	}
	var n struct {
		Paquete string `json:"packageName"`
		Sus     *struct {
			Tipo  int    `json:"notificationType"`
			Token string `json:"purchaseToken"`
		} `json:"subscriptionNotification"`
	}
	crudo, err := base64.StdEncoding.DecodeString(in.Message.Data)
	if err != nil || json.Unmarshal(crudo, &n) != nil || n.Sus == nil || n.Paquete != PaqueteAndroid || n.Sus.Token == "" {
		s.log().Info("googleplay: notificación sin suscripción nuestra", "paquete", n.Paquete)
		w.WriteHeader(http.StatusOK)
		return nil
	}
	g, ok := s.consultor()
	if !ok {
		w.WriteHeader(http.StatusOK)
		return nil
	}
	ctx := r.Context()
	sus, err := g.Consultar(ctx, PaqueteAndroid, n.Sus.Token)
	if errors.Is(err, googleplay.ErrNoEncontrada) {
		w.WriteHeader(http.StatusOK)
		return nil
	}
	if err != nil {
		return err // que Pub/Sub reintente
	}
	negocio := strings.ToLower(sus.Identificadores.Cuenta)
	if !esUUID.MatchString(negocio) {
		s.log().Warn("googleplay: suscripción sin negocio", "tipo", n.Sus.Tipo)
		w.WriteHeader(http.StatusOK)
		return nil
	}
	evento := "rtdn/" + itoa(n.Sus.Tipo)
	err = s.Pool.EnNegocio(ctx, negocio, func(tx pgx.Tx) error {
		return conciliarGoogle(ctx, tx, negocio, n.Sus.Token, *sus, evento, s.ahora())
	})
	if err != nil {
		var p *problema.Problema
		if errors.As(err, &p) {
			s.log().Warn("googleplay: no se pudo conciliar", "tipo", n.Sus.Tipo, "error", err)
			w.WriteHeader(http.StatusOK)
			return nil
		}
		return err
	}
	s.log().Info("googleplay: notificación conciliada", "tipo", n.Sus.Tipo, "negocio", negocio)
	w.WriteHeader(http.StatusOK)
	return nil
}

func itoa(n int) string {
	b, _ := json.Marshal(n)
	return string(b)
}

// conciliarGoogle deja la suscripción de Google Play de este negocio como dice Google: autorizada
// mientras está activa (o cancelada pero sin vencer, sin renovación), pausada en el período de
// gracia, cancelada si venció, quedó suspendida o en pausa. evento: "rtdn/<tipo>" ("" desde la app).
func conciliarGoogle(ctx context.Context, tx pgx.Tx, negocio, token string, sus googleplay.Suscripcion, evento string, ahora time.Time) error {
	plan, ok := planDe(ProductosApple[sus.Producto()])
	if !ok {
		return problema.Invalido("Ese producto de Google Play no es un plan de Tengo Libre.")
	}
	vence := sus.Vence()
	estado := "autorizada"
	var gracia, cancelada *time.Time
	switch sus.Estado {
	case "SUBSCRIPTION_STATE_PENDING":
		return nil // todavía no se pagó: se concilia cuando Google avise
	case "SUBSCRIPTION_STATE_IN_GRACE_PERIOD":
		g := vence
		estado, gracia = "pausada", &g
	case "SUBSCRIPTION_STATE_ACTIVE", "SUBSCRIPTION_STATE_CANCELED":
		if vence.IsZero() || vence.Before(ahora.Add(-time.Hour)) {
			estado = "cancelada"
		}
	default: // ON_HOLD, PAUSED, EXPIRED: sin acceso
		estado = "cancelada"
	}
	if !sus.Renueva() {
		cancelada = &ahora
	}
	inicio, _ := time.Parse(time.RFC3339Nano, sus.Inicio)
	if inicio.IsZero() {
		inicio = ahora
	}
	// Si cambió de plan, Google da un token nuevo y el anterior queda en linkedPurchaseToken.
	var id string
	err := tx.QueryRow(ctx, "SELECT id::text FROM suscripciones WHERE google_token = $1 OR (google_token = $2 AND $2 <> '') ORDER BY google_token = $1 DESC LIMIT 1 FOR UPDATE",
		token, sus.TokenAnterior).Scan(&id)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		if estado == "cancelada" {
			return nil // nunca la tuvimos activa: nada que hacer
		}
		h := sha256.Sum256([]byte(token))
		err = tx.QueryRow(ctx, `INSERT INTO suscripciones (negocio_id, referencia, plan, estado, monto, proximo_cobro, ultimo_cobro,
				gracia_hasta, cancelada_en, origen, google_token, google_producto, motivo)
			VALUES (app.negocio_actual(), $1, $2, $3, $4, $5, $6, $7, $8, 'google', $9, $10, 'Google Play') RETURNING id::text`,
			negocio+"-google-"+hex.EncodeToString(h[:8]), plan.Clave, estado, plan.Precio, vence, inicio, gracia, cancelada, token, sus.Producto()).Scan(&id)
		if err != nil {
			return err
		}
	case err != nil:
		return err
	default:
		_, err = tx.Exec(ctx, `UPDATE suscripciones SET plan = $2, monto = $3, estado = $4, proximo_cobro = $5, ultimo_cobro = $6,
				gracia_hasta = $7, cancelada_en = CASE WHEN $8::timestamptz IS NULL THEN NULL ELSE coalesce(cancelada_en, $8) END,
				google_token = $9, google_producto = $10, actualizado_en = now() WHERE id = $1`,
			id, plan.Clave, plan.Precio, estado, vence, inicio, gracia, cancelada, token, sus.Producto())
		if err != nil {
			return err
		}
	}
	nuevoPlan := plan.Clave
	if estado == "cancelada" {
		nuevoPlan = "sin_plan"
	}
	if _, err := tx.Exec(ctx, "UPDATE negocios SET plan = $1 WHERE id = app.negocio_actual()", nuevoPlan); err != nil {
		return err
	}
	if evento == "" {
		evento = "compra desde Android"
	}
	return auditoria.Anotar(ctx, tx, auditoria.Registro{Accion: "suscripcion.google", Entidad: "suscripcion", EntidadID: id,
		Detalle: map[string]any{"evento": evento, "estado": estado, "plan": plan.Clave, "vence": vence, "prueba": sus.Prueba != nil}})
}

// TrabajoGoogle vuelve a consultar a Google, cada 6 horas, las suscripciones de Play que vencen en los
// próximos 2 días o vencieron hace menos de 7. RTDN necesita Pub/Sub, y Pub/Sub pide facturación en
// Google Cloud: sin esto, una renovación cobrada por Google no llegaría nunca y el negocio quedaría
// vencido. Sin cuenta de Play (Google nil), no hace nada.
func (s *Servicio) TrabajoGoogle() trabajos.Trabajo {
	return trabajos.Trabajo{Nombre: "sondeo_google_play", Cada: 6 * time.Hour, Correr: func(ctx context.Context, ahora time.Time) (any, error) {
		n, err := s.SondearGoogle(ctx, ahora)
		return map[string]int{"consultadas": n}, err
	}}
}

// SondearGoogle concilia, negocio por negocio, las suscripciones de Google Play cerca del vencimiento.
// Devuelve cuántas consultó. Un token que Google ya no conoce se saltea.
func (s *Servicio) SondearGoogle(ctx context.Context, ahora time.Time) (int, error) {
	g, ok := s.consultor()
	if !ok {
		return 0, nil
	}
	type pendiente struct{ negocio, token string }
	var lista []pendiente
	err := trabajos.PorNegocio(ctx, s.Pool, func(negocio string, tx pgx.Tx) error {
		filas, err := tx.Query(ctx, `SELECT google_token FROM suscripciones
			WHERE origen = 'google' AND google_token <> '' AND estado IN ('autorizada', 'pausada')
				AND proximo_cobro BETWEEN $1 AND $2`, ahora.AddDate(0, 0, -7), ahora.AddDate(0, 0, 2))
		if err != nil {
			return err
		}
		tokens, err := pgx.CollectRows(filas, pgx.RowTo[string])
		for _, t := range tokens {
			lista = append(lista, pendiente{negocio, t})
		}
		return err
	})
	if err != nil {
		return 0, err
	}
	var errs []error
	for _, p := range lista {
		sus, err := g.Consultar(ctx, PaqueteAndroid, p.token)
		if errors.Is(err, googleplay.ErrNoEncontrada) {
			continue
		}
		if err != nil {
			errs = append(errs, err)
			continue
		}
		err = s.Pool.EnNegocio(ctx, p.negocio, func(tx pgx.Tx) error {
			return conciliarGoogle(ctx, tx, p.negocio, p.token, *sus, "sondeo", ahora)
		})
		if err != nil {
			errs = append(errs, err)
		}
	}
	return len(lista), errors.Join(errs...)
}
