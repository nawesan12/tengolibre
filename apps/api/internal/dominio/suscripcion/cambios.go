package suscripcion

// Lo que se hace con una suscripción ya dada de alta: cambiar de plan (el precio nuevo va al
// preapproval y rige desde el próximo cobro), pagar la cuota que falló, cambiar la tarjeta y
// reactivar una cancelada. Mercado Pago no deja pagar una cuota suelta ni cambiar la tarjeta de un
// preapproval sin un formulario de tarjeta propio, así que esos tres hacen un alta nueva (el link
// de Mercado Pago donde la persona carga la tarjeta) que reemplaza a la vieja: cuando la nueva queda
// autorizada, la vieja se cancela (ver reemplazar).

import (
	"context"
	"errors"
	"math"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/planes"
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

// alta crea una suscripción pendiente de plan (y período, con los profesionales extra de Centro) con
// su preapproval y devuelve su id y el init_point. reemplaza: la que se cancela cuando esta quede
// autorizada. Las pendientes de antes (un checkout que se abandonó) se cancelan acá y en Mercado
// Pago: si no, ese link se podía seguir pagando y quedaban dos suscripciones cobrando.
func (s *Servicio) alta(ctx context.Context, plan Plan, periodo string, extras, diasGratis int, reemplaza *string) (string, string, error) {
	ses := sesion.De(ctx)
	ref := ses.NegocioID + "-" + mercadopago.Azar()
	if plan.Clave != "centro" {
		extras = 0
	}
	monto := plan.PrecioDe(periodo, extras)
	var id, email string
	var abandonadas []string
	err := s.Pool.EnNegocio(ctx, ses.NegocioID, func(tx pgx.Tx) error {
		filas, err := tx.Query(ctx, `UPDATE suscripciones SET estado = 'cancelada', cancelada_en = $1, motivo = 'Checkout abandonado',
			actualizado_en = now() WHERE estado = 'pendiente' RETURNING mp_id`, s.ahora())
		if err != nil {
			return err
		}
		ids, err := pgx.CollectRows(filas, pgx.RowTo[*string])
		if err != nil {
			return err
		}
		for _, x := range ids {
			if x != nil {
				abandonadas = append(abandonadas, *x)
			}
		}
		_ = tx.QueryRow(ctx, "SELECT email FROM miembros WHERE clerk_id = $1", ses.ClerkID).Scan(&email)
		return tx.QueryRow(ctx, `INSERT INTO suscripciones (negocio_id, referencia, plan, monto, email, reemplaza, periodo, profesionales_extra)
			VALUES (app.negocio_actual(), $1, $2, $3, $4, $5, $6, $7) RETURNING id::text`, ref, plan.Clave, monto, email, reemplaza, periodo, extras).Scan(&id)
	})
	if err != nil {
		return "", "", err
	}
	for _, mp := range abandonadas {
		if err := s.cliente().CambiarPreaprobacion(context.WithoutCancel(ctx), s.TokenPlataforma, mp, "cancelled"); err != nil {
			s.log().Warn("suscripcion: no se pudo cancelar en Mercado Pago un checkout abandonado", "preapproval", mp, "error", err)
		}
	}
	p, err := s.cliente().CrearPreaprobacion(ctx, s.TokenPlataforma, preaprobacion(plan, periodo, ref, email, monto, diasGratis, s.Sitio))
	if err != nil {
		return "", "", errorMP(err)
	}
	err = s.Pool.EnNegocio(context.WithoutCancel(ctx), ses.NegocioID, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, "UPDATE suscripciones SET mp_id = $2, link = $3, actualizado_en = now() WHERE id = $1", id, p.ID, p.InitPoint)
		return err
	})
	if err != nil {
		// El preapproval ya existe y se puede pagar: ConciliarPreaprobacion encuentra la fila por la
		// referencia. Que la persona siga.
		s.log().Error("suscripcion: no se guardó el preapproval de un alta", "negocio", ses.NegocioID, "preapproval", p.ID, "error", err)
	}
	return id, p.InitPoint, nil
}

// preaprobacion: el preapproval de un plan. El anual cobra 10 meses cada 12.
func preaprobacion(plan Plan, periodo, ref, email string, monto int64, diasGratis int, sitio string) mercadopago.NuevaPreaprobacion {
	motivo, meses := "Tengo Libre "+plan.Nombre+" — suscripción mensual", 1
	if periodo == "anual" {
		motivo, meses = "Tengo Libre "+plan.Nombre+" — suscripción anual (12 meses al precio de 10)", 12
	}
	return mercadopago.NuevaPreaprobacion{Motivo: motivo, Referencia: ref, Email: email, Monto: monto, Meses: meses,
		DiasGratis: diasGratis, Vuelta: sitio + "/panel/suscripcion?resultado=vuelta"}
}

// Un cambio a la vez por suscripción (leer, Mercado Pago, escribir): dos pedidos simultáneos podían
// dejar la base en un plan y el preapproval cobrando otro. tomar marca la suscripción (dura hasta dos
// minutos, por si el proceso se cae a la mitad) y soltar la libera.
func (s *Servicio) tomar(ctx context.Context, negocio, id string) error {
	var ok bool
	err := s.Pool.EnNegocio(ctx, negocio, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `UPDATE suscripciones SET cambiando_hasta = $2 WHERE id = $1 AND (cambiando_hasta IS NULL OR cambiando_hasta < $3)`,
			id, s.ahora().Add(2*time.Minute), s.ahora())
		ok = tag.RowsAffected() == 1
		return err
	})
	if err == nil && !ok {
		return problema.Conflicto("cambio_en_curso", "Ya hay un cambio de tu suscripción en curso. Probá de nuevo en un minuto.")
	}
	return err
}

func (s *Servicio) soltar(ctx context.Context, negocio, id string) {
	err := s.Pool.EnNegocio(context.WithoutCancel(ctx), negocio, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, "UPDATE suscripciones SET cambiando_hasta = NULL WHERE id = $1", id)
		return err
	})
	if err != nil {
		s.log().Error("suscripcion: no se pudo soltar el cambio", "suscripcion", id, "error", err)
	}
}

// activaMP: la suscripción vigente de Mercado Pago (con su preapproval), para cambiarla. Las de las
// tiendas se cambian en la tienda.
func (s *Servicio) activaMP(ctx context.Context, negocio string) (*Suscripcion, string, error) {
	var x *Suscripcion
	var mpID *string
	err := s.Pool.EnNegocio(ctx, negocio, func(tx pgx.Tx) error {
		var err error
		if x, err = vigente(ctx, tx, false); err != nil || x == nil {
			return err
		}
		if p := ProblemaDeTienda(x.Origen); p != nil {
			return p
		}
		return tx.QueryRow(ctx, "SELECT mp_id FROM suscripciones WHERE id = $1", x.ID).Scan(&mpID)
	})
	if err != nil {
		return nil, "", err
	}
	if x == nil || x.Estado != "autorizada" || x.CanceladaEn != nil || mpID == nil {
		return nil, "", problema.Conflicto("sin_suscripcion", "Para cambiar de plan necesitás una suscripción activa.")
	}
	return x, *mpID, nil
}

// cambiar de plan (o de período) una suscripción autorizada de Mercado Pago ({"plan", "periodo"}).
//   - Otro período: Mercado Pago no deja cambiar la frecuencia, así que es un alta nueva que empieza a
//     cobrar cuando vence lo pagado y reemplaza a la vieja ({"url", "desde"}). El plan nuevo rige
//     desde ese primer cobro.
//   - Mismo período: ver aplicarCambio (subir se cobra ya, bajar rige desde el próximo cobro).
//
// Bajar de plan no borra nada: lo que sobra queda en sólo lectura (planes).
func (s *Servicio) cambiar(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	var in struct {
		Plan    string `json:"plan"`
		Periodo string `json:"periodo"`
	}
	if err := web.Leer(w, r, &in); err != nil {
		return err
	}
	plan, ok := planDe(in.Plan)
	if !ok {
		return problema.Invalido("Elegí Inicial, Equipo o Centro.", problema.Campo{Ruta: "plan", Mensaje: "inválido"})
	}
	if in.Periodo != "" {
		if _, ok := periodoDe(in.Periodo); !ok {
			return problema.Invalido("El período es mensual o anual.", problema.Campo{Ruta: "periodo", Mensaje: "inválido"})
		}
	}
	if err := s.sinSuscripciones(); err != nil {
		return err
	}
	negocio := sesion.De(ctx).NegocioID
	x, mpID, err := s.activaMP(ctx, negocio)
	if err != nil {
		return err
	}
	if err := s.tomar(ctx, negocio, x.ID); err != nil {
		return err
	}
	defer s.soltar(ctx, negocio, x.ID)
	periodo := x.Periodo
	if in.Periodo != "" {
		periodo, _ = periodoDe(in.Periodo)
	}
	extras := 0
	if plan.Clave == "centro" {
		extras = x.ProfesionalesExtra
		if x.ExtrasSiguiente != nil && x.PlanSiguiente != nil && *x.PlanSiguiente == "centro" {
			extras = *x.ExtrasSiguiente
		}
	}
	if periodo != x.Periodo {
		_, u, err := s.alta(ctx, plan, periodo, extras, diasHasta(finDelPeriodo(x), s.ahora()), &x.ID)
		if err != nil {
			return err
		}
		out := map[string]any{"url": u, "hoy": 0, "despues": plan.PrecioDe(periodo, extras)}
		if f := finDelPeriodo(x); f != nil {
			out["desde"] = f
		}
		web.JSON(w, http.StatusOK, out)
		return nil
	}
	return s.aplicarCambio(ctx, w, x, mpID, plan, extras)
}

// extras: los profesionales que Centro suma arriba de 25 ({"cantidad": n}, de 0 a 100), a $1.900
// por mes cada uno. Sumar es subir (se cobra ya la diferencia por lo que queda del período); sacar
// rige desde el próximo cobro. Sólo con Centro pagado por Mercado Pago (los productos de las tiendas
// tienen precio fijo).
func (s *Servicio) extras(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	var in struct {
		Cantidad int `json:"cantidad"`
	}
	if err := web.Leer(w, r, &in); err != nil {
		return err
	}
	if in.Cantidad < 0 || in.Cantidad > planes.MaxProfesionalesExtra {
		return problema.Invalido("Podés sumar de 0 a 100 profesionales extra.", problema.Campo{Ruta: "cantidad", Mensaje: "fuera de rango"})
	}
	if err := s.sinSuscripciones(); err != nil {
		return err
	}
	negocio := sesion.De(ctx).NegocioID
	var origen string
	if err := s.Pool.EnNegocio(ctx, negocio, func(tx pgx.Tx) error {
		x, err := vigente(ctx, tx, false)
		if x != nil {
			origen = x.Origen
		}
		return err
	}); err != nil {
		return err
	}
	if origen != "" && origen != "mercadopago" {
		return problema.Conflicto("extras_tienda", "Los profesionales extra se suman con la suscripción por Mercado Pago (desde la web). "+
			"Tu suscripción se paga en la tienda de tu teléfono.")
	}
	x, mpID, err := s.activaMP(ctx, negocio)
	if err != nil {
		var p *problema.Problema
		if errors.As(err, &p) && p.Codigo == "sin_suscripcion" {
			return problema.Conflicto("extras_sin_centro", "Los profesionales extra se suman al plan Centro. Pasá a Centro primero.")
		}
		return err
	}
	if x.Plan != "centro" || (x.PlanSiguiente != nil && *x.PlanSiguiente != "centro") {
		return problema.Conflicto("extras_sin_centro", "Los profesionales extra se suman al plan Centro. Pasá a Centro primero.")
	}
	if err := s.tomar(ctx, negocio, x.ID); err != nil {
		return err
	}
	defer s.soltar(ctx, negocio, x.ID)
	plan, _ := planes.De("centro")
	return s.aplicarCambio(ctx, w, x, mpID, plan, in.Cantidad)
}

// finDelPeriodo: hasta cuándo está pago lo que se cobró (el próximo cobro; si Mercado Pago no lo
// dijo, el último más un período). nil: todavía no se cobró nada.
func finDelPeriodo(x *Suscripcion) *time.Time {
	if x.ProximoCobro != nil {
		return x.ProximoCobro
	}
	if x.UltimoCobro == nil {
		return nil
	}
	f := x.UltimoCobro.AddDate(0, 1, 0)
	if x.Periodo == "anual" {
		f = x.UltimoCobro.AddDate(1, 0, 0)
	}
	return &f
}

// Prorrateo: lo que se cobra hoy al subir de montoActual a nuevo, por la parte del período pago que
// queda (redondeado para arriba, en pesos). 0 si no queda nada del período.
func Prorrateo(periodo string, fin time.Time, ahora time.Time, montoActual, nuevo int64) int64 {
	if nuevo <= montoActual || !fin.After(ahora) {
		return 0
	}
	inicio := fin.AddDate(0, -1, 0)
	if periodo == "anual" {
		inicio = fin.AddDate(-1, 0, 0)
	}
	total := fin.Sub(inicio).Seconds()
	queda := min(fin.Sub(ahora).Seconds(), total)
	return int64(math.Ceil(float64(nuevo-montoActual) * queda / total))
}

// aplicarCambio de plan o de extras dentro del mismo período (x ya está tomada):
//   - Sin nada cobrado todavía (la prueba no se cobra): cambia el monto y el plan ya; el primer cobro
//     ya es el nuevo. En la prueba rige Equipo igual (app.plan_vigente).
//   - Subir: se cobra ya la diferencia por lo que queda del período, con una orden de la cuenta de
//     la plataforma ({"url", "hoy", "despues"}); el plan nuevo rige cuando se acredita (ConciliarSubida).
//   - Bajar: el monto del preapproval baja ya (el próximo cobro es el nuevo) y el plan queda
//     programado (plan_siguiente): hasta ese cobro se usa lo que ya se pagó.
//   - Volver al plan pagado: deshace la baja programada.
func (s *Servicio) aplicarCambio(ctx context.Context, w http.ResponseWriter, x *Suscripcion, mpID string, plan Plan, extras int) error {
	negocio := sesion.De(ctx).NegocioID
	if plan.Clave != "centro" {
		extras = 0
	}
	actual, ok := planes.De(x.Plan)
	if !ok {
		actual = planes.Planes[0]
	}
	montoActual := actual.PrecioDe(x.Periodo, x.ProfesionalesExtra)
	nuevo := plan.PrecioDe(x.Periodo, extras)
	mismo := plan.Clave == actual.Clave && extras == x.ProfesionalesExtra
	if mismo && x.PlanSiguiente == nil && x.ExtrasSiguiente == nil {
		return problema.Conflicto("mismo_plan", "Ya estás en el plan "+plan.Nombre+".")
	}
	sinCobrar := x.UltimoCobro == nil
	fin := finDelPeriodo(x)
	var hoy int64
	if !sinCobrar && !mismo && fin != nil {
		hoy = Prorrateo(x.Periodo, *fin, s.ahora(), montoActual, nuevo)
	}
	if hoy > 0 {
		u, err := s.ordenSubida(ctx, x, plan, extras, nuevo, hoy)
		if err != nil {
			return err
		}
		web.JSON(w, http.StatusOK, map[string]any{"url": u, "hoy": hoy, "despues": nuevo, "desde": fin})
		return nil
	}
	if err := s.cliente().CambiarMontoPreaprobacion(ctx, s.TokenPlataforma, mpID, nuevo); err != nil {
		return errorMP(err)
	}
	accion := "suscripcion.plan_programado"
	err := s.Pool.EnNegocio(context.WithoutCancel(ctx), negocio, func(tx pgx.Tx) error {
		switch {
		case mismo:
			accion = "suscripcion.baja_deshecha"
			if _, err := tx.Exec(ctx, `UPDATE suscripciones SET monto = $2, plan_siguiente = NULL, extras_siguiente = NULL, actualizado_en = now()
				WHERE id = $1`, x.ID, nuevo); err != nil {
				return err
			}
		case sinCobrar || nuevo >= montoActual:
			// Sin nada cobrado, o una subida sin días que prorratear (el cobro es ahora): rige ya.
			accion = "suscripcion.plan_cambiado"
			if _, err := tx.Exec(ctx, `UPDATE suscripciones SET plan = $2, monto = $3, profesionales_extra = $4, plan_siguiente = NULL,
				extras_siguiente = NULL, actualizado_en = now() WHERE id = $1`, x.ID, plan.Clave, nuevo, extras); err != nil {
				return err
			}
			if x.Reemplaza == nil || x.UltimoCobro != nil {
				if err := ponerPlan(ctx, tx, x.ID); err != nil {
					return err
				}
			}
		default:
			if _, err := tx.Exec(ctx, `UPDATE suscripciones SET monto = $2, plan_siguiente = $3, extras_siguiente = $4, actualizado_en = now()
				WHERE id = $1`, x.ID, nuevo, plan.Clave, extras); err != nil {
				return err
			}
		}
		if err := auditoria.Anotar(ctx, tx, auditoria.Registro{Accion: accion, Entidad: "suscripcion", EntidadID: x.ID,
			Detalle: map[string]any{"antes": x.Plan, "ahora": plan.Clave, "extras": extras, "monto": nuevo, "periodo": x.Periodo}}); err != nil {
			return err
		}
		var err error
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
		if p := ProblemaDeTienda(x.Origen); p != nil {
			return p
		}
		dias, reemplaza, err := elegir(x, s.ahora())
		if err != nil {
			return err
		}
		// Una vieja de «completo» se reactiva como Equipo. Si había una baja programada, el alta nueva ya es
		// del plan de abajo (empieza a cobrar cuando vence lo pagado).
		clave, extras := x.Plan, x.ProfesionalesExtra
		if x.PlanSiguiente != nil {
			clave = *x.PlanSiguiente
		}
		if x.ExtrasSiguiente != nil {
			extras = *x.ExtrasSiguiente
		}
		plan, _ := planes.De(clave)
		var de *string
		if reemplaza {
			de = &x.ID
		}
		_, u, err := s.alta(ctx, plan, x.Periodo, extras, dias, de)
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
