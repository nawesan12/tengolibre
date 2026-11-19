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

	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/planes"
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
	// Profesionales extra de Centro y packs de WhatsApp (extras.go).
	mux.HandleFunc("PUT /v1/panel/suscripcion/extras", ses.Panel([]string{sesion.Duenio}, web.Envolver(s.extras)))
	mux.HandleFunc("POST /v1/panel/suscripcion/whatsapp", ses.Panel([]string{sesion.Duenio}, web.Envolver(s.comprarPack)))
	// La App Store (apple.go): la compra desde el iPhone y sus notificaciones.
	s.RutasApple(mux)
	s.RutasGoogle(mux)
}

// Uso del plan que rige: lo que lleva el negocio contra cada tope (para los avisos del panel).
type Uso struct {
	ProfesionalesActivos int `json:"profesionalesActivos"`
	TopeProfesionales    int `json:"topeProfesionales"`
	ProfesionalesExtra   int `json:"profesionalesExtra"`
	// WhatsApp: los incluidos del mes (Centro), los usados y el saldo de packs.
	WhatsAppIncluidos int `json:"whatsappIncluidos"`
	WhatsAppUsadosMes int `json:"whatsappUsadosMes"`
	WhatsAppSaldo     int `json:"whatsappSaldo"`
	CupoIA            int `json:"cupoIA"`
}

// Extras que se venden aparte (para la pantalla de planes).
type Extras struct {
	PrecioProfesional int64 `json:"precioProfesional"`
	PackWhatsApp      struct {
		Cantidad int   `json:"cantidad"`
		Precio   int64 `json:"precio"`
	} `json:"packWhatsApp"`
	MesesAnual   int `json:"mesesAnual"`
	DiasDePrueba int `json:"diasDePrueba"`
}

func extrasDeVenta() Extras {
	var e Extras
	e.PrecioProfesional = planes.PrecioProfesionalExtra
	e.PackWhatsApp.Cantidad, e.PackWhatsApp.Precio = planes.PackWhatsAppCantidad, planes.PackWhatsAppPrecio
	e.MesesAnual, e.DiasDePrueba = planes.MesesAnual, planes.DiasDePrueba
	return e
}

// LeerUso del negocio fijado en tx.
func LeerUso(ctx context.Context, tx pgx.Tx, ahora time.Time) (Uso, error) {
	v, err := planes.Leer(ctx, tx, ahora)
	if err != nil {
		return Uso{}, err
	}
	u := Uso{TopeProfesionales: v.TopeProfesionales(), ProfesionalesExtra: v.Extras, WhatsAppIncluidos: v.WhatsAppIncluidos, CupoIA: v.CupoIA}
	if err := tx.QueryRow(ctx, "SELECT count(*) FROM profesionales WHERE activo").Scan(&u.ProfesionalesActivos); err != nil {
		return u, err
	}
	if err := tx.QueryRow(ctx, "SELECT whatsapp_saldo FROM negocios WHERE id = app.negocio_actual()").Scan(&u.WhatsAppSaldo); err != nil {
		return u, err
	}
	u.WhatsAppUsadosMes, err = planes.WhatsAppDelMes(ctx, tx, ahora)
	return u, err
}

func (s *Servicio) leer(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	var out struct {
		Acceso
		NombrePlan  string       `json:"nombrePlan"`
		Catalogo    []Plan       `json:"catalogo"`
		Extras      Extras       `json:"extras"`
		CupoIA      int          `json:"cupoIA"`
		Uso         Uso          `json:"uso"`
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
		if out.Uso, err = LeerUso(ctx, tx, s.ahora()); err != nil {
			return err
		}
		out.Suscripcion, err = ultima(ctx, tx)
		return err
	})
	if err != nil {
		return err
	}
	out.NombrePlan = "Sin plan"
	if p, ok := planes.De(out.Plan); ok {
		out.NombrePlan = p.Nombre
	}
	out.Catalogo = Planes
	out.Extras = extrasDeVenta()
	out.CupoIA = out.Uso.CupoIA
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

// suscribir crea la suscripción pendiente ({"plan", "periodo": mensual|anual}) y devuelve el
// init_point. El plan cambia recién cuando Mercado Pago avisa que quedó autorizada. Lo que queda de
// la prueba no se cobra.
func (s *Servicio) suscribir(w http.ResponseWriter, r *http.Request) error {
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
	periodo, ok := periodoDe(in.Periodo)
	if !ok {
		return problema.Invalido("El período es mensual o anual.", problema.Campo{Ruta: "periodo", Mensaje: "inválido"})
	}
	if s.TokenPlataforma == "" {
		return problema.Conflicto("sin_suscripciones", "Las suscripciones todavía no están habilitadas.")
	}
	ses := sesion.De(ctx)
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
		if x, err := vigente(ctx, tx, false); err != nil {
			return err
		} else if x != nil && ProblemaDeTienda(x.Origen) != nil {
			return ProblemaDeTienda(x.Origen)
		}
		if activa {
			return problema.Conflicto("ya_suscripto", "Ya tenés una suscripción activa. Cambiá de plan desde ahí.")
		}
		if a.PruebaHasta != nil {
			diasGratis = int(math.Ceil(a.PruebaHasta.Sub(s.ahora()).Hours() / 24))
		}
		return nil
	})
	if err != nil {
		return err
	}
	u, err := s.alta(ctx, plan, periodo, 0, diasGratis, nil)
	if err != nil {
		return err
	}
	var id string
	err = s.Pool.EnNegocio(ctx, ses.NegocioID, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, "SELECT id::text FROM suscripciones WHERE link = $1 ORDER BY creado_en DESC LIMIT 1", u).Scan(&id)
	})
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusCreated, map[string]string{"suscripcionId": id, "url": u})
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
	hubo, err := s.bajaEn(ctx, sesion.De(ctx).NegocioID, in.Motivo, true)
	if err != nil {
		return err
	}
	if !hubo {
		return problema.NoEncontrado("No hay una suscripción para cancelar.")
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}

// CancelarPorBaja da de baja la suscripción del negocio que se cierra porque su única dueña borró
// la cuenta, para que Mercado Pago no le siga cobrando. Las de Apple y Google no se pueden cancelar
// desde acá (las maneja la tienda): esas se saltean y la app le avisa a la persona.
func (s *Servicio) CancelarPorBaja(ctx context.Context, negocio string) error {
	_, err := s.bajaEn(ctx, negocio, "Cuenta borrada", false)
	return err
}

// bajaEn cancela la suscripción vigente del negocio, en Mercado Pago (si llegó a existir allá) y
// acá. Devuelve false si no había ninguna. tiendaEsError: si es de Apple o Google, responder con el
// problema de la tienda (el panel) o saltearla (la baja de la cuenta).
func (s *Servicio) bajaEn(ctx context.Context, negocio, motivo string, tiendaEsError bool) (bool, error) {
	var x *Suscripcion
	var mpID *string
	err := s.Pool.EnNegocio(ctx, negocio, func(tx pgx.Tx) error {
		var err error
		if x, err = vigente(ctx, tx, false); err != nil || x == nil {
			return err
		}
		return tx.QueryRow(ctx, "SELECT mp_id FROM suscripciones WHERE id = $1", x.ID).Scan(&mpID)
	})
	if err != nil || x == nil {
		return false, err
	}
	if p := ProblemaDeTienda(x.Origen); p != nil {
		if tiendaEsError {
			return true, p
		}
		return true, nil
	}
	if mpID != nil && s.TokenPlataforma != "" {
		if err := s.cliente().CambiarPreaprobacion(ctx, s.TokenPlataforma, *mpID, "cancelled"); err != nil {
			return true, errorMP(err)
		}
	}
	err = s.Pool.EnNegocio(ctx, negocio, func(tx pgx.Tx) error {
		return darDeBaja(ctx, tx, x.ID, s.ahora(), motivo)
	})
	return true, err
}

// darDeBaja: con el mes pago, pausada hasta el próximo cobro; si no, cancelada y el plan del negocio
// sale de las que sigan valiendo (sin ninguna, sin plan).
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
	return recalcularPlan(ctx, tx, ahora)
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
			return ponerPlan(ctx, tx, subID)
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
			return recalcularPlan(ctx, tx, ahora)
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
			// Una cancelada o vencida no revive con un cobro suyo que llegó tarde (por ejemplo, la vieja
			// después de cambiar de plan, si su baja en Mercado Pago falló): pisaba el plan nuevo.
			tag, err := tx.Exec(ctx, `UPDATE suscripciones SET estado = 'autorizada', ultimo_cobro = $2, proximo_cobro = coalesce($3, proximo_cobro),
				gracia_hasta = NULL, actualizado_en = now() WHERE id = $1 AND estado IN ('pendiente', 'autorizada', 'pausada')`, subID, ahora, p.NextPaymentDate)
			if err != nil {
				return err
			}
			if tag.RowsAffected() == 0 {
				s.log().Error("suscripcion: cobro aprobado de una suscripción cancelada o vencida; revisar y devolver", "negocio", negocio, "cobro", id)
				return nil
			}
			return ponerPlan(ctx, tx, subID)
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

// ponerPlan: el negocio pasa al plan de la suscripción id (y a sus profesionales extra).
func ponerPlan(ctx context.Context, tx pgx.Tx, id string) error {
	_, err := tx.Exec(ctx, `UPDATE negocios n SET plan = s.plan, profesionales_extra = CASE WHEN s.plan = 'centro' THEN s.profesionales_extra ELSE 0 END
		FROM suscripciones s WHERE s.id = $1 AND n.id = app.negocio_actual()`, id)
	return err
}
