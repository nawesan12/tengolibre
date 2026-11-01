// Package suscripcion: lo que el negocio le paga a TengoLibre. Planes, la prueba gratis
// (negocios.prueba_hasta), el alta con Mercado Pago (un preapproval con la cuenta de la plataforma),
// sus avisos y la baja. Y la regla de acceso de Jiku (subscription/access.ts): vencida la prueba y
// sin una suscripción vigente, el panel queda de sólo lectura. Leer siempre se puede, y la página
// pública sigue tomando reservas; lo que se corta es escribir, que es lo que se paga.
package suscripcion

import (
	"context"
	"errors"
	"log/slog"
	"math"
	"net/http"
	"regexp"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/nawesan12/tengolibre/apps/api/internal/integraciones/mercadopago"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/db"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/problema"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/sesion"
)

// DiasDePrueba de un negocio nuevo (el default de negocios.prueba_hasta en 00004).
const DiasDePrueba = 7

// DiasDeGracia: una suscripción pausada (una tarjeta que venció el fin de semana) sigue valiendo.
const DiasDeGracia = 7

// Plan que se vende. El precio es el que se manda en el preapproval: sale de acá, nunca del navegador.
type Plan struct {
	Clave  string `json:"clave"`
	Nombre string `json:"nombre"`
	Precio int64  `json:"precio"`
}

// Planes (PLAN_PRICES y PLAN_NAMES de Jiku). "sin_plan" no se vende.
var Planes = []Plan{{"inicial", "Inicial", 7000}, {"completo", "Completo", 15000}}

func planDe(clave string) (Plan, bool) {
	for _, p := range Planes {
		if p.Clave == clave {
			return p, true
		}
	}
	return Plan{}, false
}

// Servicio de suscripciones.
type Servicio struct {
	Pool   *db.Pool
	Sesion *sesion.Servicio
	// Cliente de Mercado Pago (nil: el de verdad) y el token de la cuenta de la plataforma. Sin token,
	// no se puede suscribir (el resto anda).
	Cliente         *mercadopago.Cliente
	TokenPlataforma string
	// Sitio del panel, para la vuelta de Mercado Pago.
	Sitio string
	Ahora func() time.Time
	Log   *slog.Logger
}

func (s *Servicio) ahora() time.Time {
	if s.Ahora != nil {
		return s.Ahora()
	}
	return time.Now()
}

func (s *Servicio) log() *slog.Logger {
	if s.Log != nil {
		return s.Log
	}
	return slog.Default()
}

// Acceso del negocio: si puede escribir, y por qué.
type Acceso struct {
	// Bloqueado: sin prueba y sin una suscripción que la pague, el panel es de sólo lectura.
	Bloqueado bool `json:"bloqueado"`
	// PruebaHasta, mientras la prueba corre.
	PruebaHasta    *time.Time `json:"pruebaHasta"`
	DiasDePrueba   int        `json:"diasDePrueba"`
	ConSuscripcion bool       `json:"conSuscripcion"`
	// Plan efectivo: durante la prueba, el completo (para mostrar el producto entero).
	Plan string `json:"plan"`
	// ProximoCobro de la suscripción autorizada (null: no hay, o Mercado Pago no lo dijo todavía).
	// Lo completa LeerCobros (no el chequeo de escritura).
	ProximoCobro *Cobro `json:"proximoCobro"`
	// CobroFallidoHasta: el último cobro no pasó y la suscripción vale, en gracia, hasta acá.
	CobroFallidoHasta *time.Time `json:"cobroFallidoHasta"`
}

// Cobro que viene.
type Cobro struct {
	Fecha time.Time `json:"fecha"`
	Monto int64     `json:"monto"`
}

// LeerCobros completa ProximoCobro y CobroFallidoHasta del negocio fijado en tx. Una pausada sin
// cancelar es un cobro rechazado (la cancelada con el mes pago también queda pausada, pero con
// cancelada_en).
func LeerCobros(ctx context.Context, tx pgx.Tx, ahora time.Time, a *Acceso) error {
	var fecha *time.Time
	var monto int64
	err := tx.QueryRow(ctx, `SELECT proximo_cobro, monto FROM suscripciones WHERE estado = 'autorizada'
		ORDER BY creado_en DESC LIMIT 1`).Scan(&fecha, &monto)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return err
	}
	if fecha != nil {
		a.ProximoCobro = &Cobro{Fecha: *fecha, Monto: monto}
		return nil
	}
	if err == nil {
		return nil // autorizada sin fecha: no hay cobro fallido que mostrar
	}
	err = tx.QueryRow(ctx, `SELECT gracia_hasta FROM suscripciones WHERE estado = 'pausada' AND cancelada_en IS NULL
		AND gracia_hasta > $1 ORDER BY creado_en DESC LIMIT 1`, ahora).Scan(&a.CobroFallidoHasta)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	return err
}

// LeerAcceso del negocio fijado en tx. Una suscripción pausada cuenta mientras dure la gracia.
func LeerAcceso(ctx context.Context, tx pgx.Tx, ahora time.Time) (Acceso, error) {
	var a Acceso
	var hasta *time.Time
	var plan string
	err := tx.QueryRow(ctx, `SELECT n.prueba_hasta, n.plan, EXISTS (SELECT 1 FROM suscripciones s WHERE s.estado = 'autorizada'
			OR (s.estado = 'pausada' AND (s.gracia_hasta IS NULL OR s.gracia_hasta > $1)))
		FROM negocios n WHERE n.id = app.negocio_actual()`, ahora).Scan(&hasta, &plan, &a.ConSuscripcion)
	if err != nil {
		return a, err
	}
	enPrueba := hasta != nil && hasta.After(ahora)
	if enPrueba {
		a.PruebaHasta = hasta
		a.DiasDePrueba = int(math.Ceil(hasta.Sub(ahora).Hours() / 24))
	}
	a.Bloqueado = !enPrueba && !a.ConSuscripcion
	switch {
	case enPrueba:
		a.Plan = "completo"
	case a.ConSuscripcion:
		a.Plan = plan
	default:
		a.Plan = "sin_plan"
	}
	return a, nil
}

// ProblemaSinSuscripcion es lo que responde una escritura con el panel bloqueado (402, como Jiku).
func ProblemaSinSuscripcion() *problema.Problema {
	return &problema.Problema{Estado: http.StatusPaymentRequired, Codigo: "suscripcion_requerida",
		Titulo: "Tu prueba gratis terminó", Detalle: "Suscribite para seguir operando. Podés seguir viendo todo."}
}

// SinChequeo son las rutas del panel que escriben igual con el panel bloqueado: suscribirse (es lo
// que destraba), desconectar Mercado Pago y devolver plata a un cliente. Van a
// sesion.Servicio.SinChequeoDeEscritura.
var SinChequeo = []string{"/v1/panel/suscripcion", "/v1/panel/mercadopago", "/v1/panel/pagos/"}

// Escritura es el chequeo reutilizable de sólo lectura: va adentro de sesion.Panel (necesita el
// negocio de la sesión) y antes del handler de una ruta que escribe.
//
//	mux.HandleFunc("POST /v1/panel/x", ses.Panel(roles, sus.Escritura(web.Envolver(h))))
func (s *Servicio) Escritura(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		negocio := sesion.De(r.Context()).NegocioID
		if negocio == "" {
			problema.Escribir(w, problema.SinSesion())
			return
		}
		var a Acceso
		err := s.Pool.EnNegocio(r.Context(), negocio, func(tx pgx.Tx) (err error) {
			a, err = LeerAcceso(r.Context(), tx, s.ahora())
			return err
		})
		if err != nil {
			s.log().Error("suscripcion: no se pudo leer el acceso", "negocio", negocio, "error", err)
			problema.Escribir(w, problema.Interno())
			return
		}
		if a.Bloqueado {
			problema.Escribir(w, ProblemaSinSuscripcion())
			return
		}
		next(w, r)
	}
}

// Suscripcion como la ve el panel.
type Suscripcion struct {
	ID           string     `json:"id"`
	Plan         string     `json:"plan"`
	Estado       string     `json:"estado"`
	Monto        int64      `json:"monto"`
	Link         *string    `json:"link"`
	ProximoCobro *time.Time `json:"proximoCobro"`
	UltimoCobro  *time.Time `json:"ultimoCobro"`
	GraciaHasta  *time.Time `json:"graciaHasta"`
	CanceladaEn  *time.Time `json:"canceladaEn"`
	CreadoEn     time.Time  `json:"creadoEn"`
}

const columnas = `id::text, plan, estado, monto, link, proximo_cobro, ultimo_cobro, gracia_hasta, cancelada_en, creado_en`

func escanear(f pgx.Row, x *Suscripcion) error {
	return f.Scan(&x.ID, &x.Plan, &x.Estado, &x.Monto, &x.Link, &x.ProximoCobro, &x.UltimoCobro, &x.GraciaHasta, &x.CanceladaEn, &x.CreadoEn)
}

// vigente: la última autorizada, pausada o pendiente, en ese orden (un alta que reemplaza a otra
// queda pendiente hasta que Mercado Pago la autorice: hasta ahí se ve la de antes).
func vigente(ctx context.Context, tx pgx.Tx, bloquear bool) (*Suscripcion, error) {
	q := "SELECT " + columnas + ` FROM suscripciones WHERE estado IN ('autorizada', 'pausada', 'pendiente')
		ORDER BY CASE estado WHEN 'autorizada' THEN 0 WHEN 'pausada' THEN 1 ELSE 2 END, creado_en DESC LIMIT 1`
	if bloquear {
		q += " FOR UPDATE"
	}
	var x Suscripcion
	err := escanear(tx.QueryRow(ctx, q), &x)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return &x, err
}

var esUUID = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

// negocioDeReferencia: la referencia empieza con el id del negocio. Lo que no la tenga no es nuestro.
func negocioDeReferencia(ref string) string {
	if len(ref) < 36 || !esUUID.MatchString(ref[:36]) {
		return ""
	}
	return ref[:36]
}
