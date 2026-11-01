package cobros

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/turnos"
	"github.com/nawesan12/tengolibre/apps/api/internal/integraciones/mercadopago"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/problema"
)

// Config de cobro del negocio (BusinessSettings.paymentMode y deposit*).
type Config struct {
	// Modo: sin_cobro, porcentaje (seña), fija (seña) o total.
	Modo           string `json:"modo"`
	SenaPorcentaje int    `json:"senaPorcentaje"`
	SenaFija       int64  `json:"senaFija"`
}

func leerConfig(ctx context.Context, tx pgx.Tx) (Config, error) {
	var c Config
	err := tx.QueryRow(ctx, "SELECT cobro_modo, sena_porcentaje, sena_fija FROM ajustes WHERE negocio_id = app.negocio_actual()").
		Scan(&c.Modo, &c.SenaPorcentaje, &c.SenaFija)
	return c, err
}

// Monto que se cobra por adelantado sobre un precio (ya con el descuento): calculatePaymentAmount.
func Monto(precio int64, c Config) int64 {
	if precio <= 0 {
		return 0
	}
	switch c.Modo {
	case "total":
		return precio
	case "porcentaje":
		return (precio*int64(c.SenaPorcentaje) + 50) / 100
	case "fija":
		return min(c.SenaFija, precio)
	}
	return 0
}

// puedeCobrar: el negocio tiene su cuenta conectada y usable. Sin eso, la reserva se toma sin cobrar
// (como Jiku): una reserva sin seña es peor producto, pero plata cobrada en otra cuenta es una deuda.
func (s *Servicio) puedeCobrar(ctx context.Context, tx pgx.Tx) (bool, error) {
	if !s.Activo() {
		return false, nil
	}
	c, err := leerConexion(ctx, tx)
	return c.Conectada && !c.Reautorizar, err
}

func nuevaReferencia() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return "TL" + strings.ToUpper(hex.EncodeToString(b))
}

// AlReservar (turnos.Extra): si el negocio cobra, la reserva web queda pendiente_pago por VencePago
// y se registra el pago pendiente. El monto sale del precio del turno (ya con el cupón), nunca del
// navegador. Un turno cubierto por una membresía no se cobra; las reservas del panel tampoco. Las
// que hace el asistente de IA de una persona (origen ia) se cobran como las de la web.
func (s *Servicio) AlReservar(ctx context.Context, tx pgx.Tx, r *turnos.Reserva) error {
	if r.Origen != "web" && r.Origen != turnos.OrigenIA {
		return nil
	}
	c, err := leerConfig(ctx, tx)
	if err != nil || c.Modo == "sin_cobro" {
		return err
	}
	var cubierto bool
	if err := tx.QueryRow(ctx, "SELECT membresia_id IS NOT NULL FROM turnos WHERE id = $1", r.Turno.ID).Scan(&cubierto); err != nil {
		return err
	}
	monto := Monto(r.Turno.Precio, c)
	if cubierto || monto <= 0 {
		return nil
	}
	ok, err := s.puedeCobrar(ctx, tx)
	if err != nil {
		return err
	}
	if !ok {
		s.log().Warn("cobros: reserva tomada sin cobrar: Mercado Pago no está usable", "turno", r.Turno.ID)
		return nil
	}
	vence := r.Ahora.Add(VencePago)
	if _, err := tx.Exec(ctx, `UPDATE turnos SET estado = 'pendiente_pago', vence_en = $2, actualizado_en = now() WHERE id = $1`,
		r.Turno.ID, vence); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO pagos (negocio_id, turno_id, referencia, monto, modo) VALUES (app.negocio_actual(), $1, $2, $3, $4)`,
		r.Turno.ID, nuevaReferencia(), monto, c.Modo); err != nil {
		return err
	}
	r.Turno.Estado, r.Turno.VenceEn = "pendiente_pago", &vence
	r.Turno.Pago = &turnos.Pago{Monto: monto, Modo: c.Modo}
	return nil
}

// Estimar (turnos.Estimador): lo que se cobraría por adelantado sobre un precio, con las mismas
// condiciones que AlReservar en la web (modo de cobro y cuenta usable). nil: no se cobra.
func (s *Servicio) Estimar(ctx context.Context, tx pgx.Tx, precio int64) (*turnos.Pago, error) {
	c, err := leerConfig(ctx, tx)
	if err != nil || c.Modo == "sin_cobro" {
		return nil, err
	}
	monto := Monto(precio, c)
	if monto <= 0 {
		return nil, nil
	}
	if ok, err := s.puedeCobrar(ctx, tx); err != nil || !ok {
		return nil, err
	}
	return &turnos.Pago{Monto: monto, Modo: c.Modo}, nil
}

// DespuesDeReservar (turnos.Extra): pide el link de pago (el checkout_url de una orden de Checkout Pro). Si Mercado Pago no responde, la
// reserva queda igual y el link se pide de nuevo desde /v1/t/{codigo}/pagar.
func (s *Servicio) DespuesDeReservar(ctx context.Context, negocio string, t *turnos.Turno) {
	if t.Pago == nil {
		return
	}
	u, err := s.linkDePago(ctx, negocio, t.ID)
	if err != nil {
		s.log().Error("cobros: no se pudo crear la orden", "turno", t.ID, "error", err)
		return
	}
	t.Pago.URL = u
}

// AlCancelar (turnos.Extra): el pago que todavía no se hizo se anula (no queda como deuda). Uno
// aprobado queda aprobado: se devuelve desde el panel.
func (s *Servicio) AlCancelar(ctx context.Context, tx pgx.Tx, turnoID string) error {
	_, err := tx.Exec(ctx, `UPDATE pagos SET estado = 'cancelado', actualizado_en = now()
		WHERE turno_id = $1 AND estado IN ('pendiente', 'en_proceso', 'rechazado')`, turnoID)
	return err
}

var (
	errYaPagado = problema.Conflicto("pagado", "Este turno ya está pagado.")
	errVencido  = problema.Conflicto("vencido", "Se venció el plazo para pagar: sacá el turno de nuevo.")
	errSinPago  = problema.NoEncontrado("Este turno no tiene un pago pendiente.")
)

// linkDePago crea (o reusa) la orden de Checkout Pro del pago de un turno, con el token del negocio, y
// devuelve su checkout_url. La orden vence con el turno: pasado vence_en, el horario ya se liberó.
// Una orden cuyo intento fue rechazado no admite otro intento: se crea una nueva (con otra clave de
// idempotencia, atada a la anterior). Un link viejo, de una preferencia, se reusa como antes.
func (s *Servicio) linkDePago(ctx context.Context, negocio, turnoID string) (string, error) {
	var ref, modo, servicio, nombre, codigo, estadoPago, estadoTurno string
	var monto int64
	var link, ordenAnterior *string
	var vence *time.Time
	err := s.Pool.EnNegocio(ctx, negocio, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `SELECT p.referencia, p.modo, p.monto, p.link, p.mp_orden, p.estado, t.estado, t.servicio, t.codigo, t.vence_en, n.nombre
			FROM pagos p JOIN turnos t ON t.id = p.turno_id JOIN negocios n ON n.id = p.negocio_id WHERE p.turno_id = $1`, turnoID).
			Scan(&ref, &modo, &monto, &link, &ordenAnterior, &estadoPago, &estadoTurno, &servicio, &codigo, &vence, &nombre)
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return "", errSinPago
	}
	if err != nil {
		return "", err
	}
	reintento := estadoPago == "rechazado" && ordenAnterior != nil
	switch {
	case estadoPago == "aprobado" || estadoPago == "devuelto":
		return "", errYaPagado
	case estadoTurno != "pendiente_pago" || estadoPago == "cancelado" || (vence != nil && !vence.After(s.ahora())):
		return "", errVencido
	case link != nil && !reintento:
		return *link, nil
	}
	token, err := s.tokenDe(ctx, negocio)
	if err != nil {
		return "", err
	}
	concepto := "Seña"
	if modo == "total" {
		concepto = "Turno"
	}
	vuelta := s.Sitio + "/turno/" + codigo
	o := mercadopago.NuevaOrden(ref, mercadopago.Recortar(concepto+": "+servicio+" en "+nombre, 250), monto)
	o.Config = &mercadopago.ConfigOrden{
		Online: &mercadopago.ConfigOnline{SuccessURL: vuelta + "?pago=ok", PendingURL: vuelta + "?pago=pendiente", FailureURL: vuelta + "?pago=error"},
		// Una seña no se paga en cuotas.
		PaymentMethod: &mercadopago.MediosOrden{MaxInstallments: 1},
		StatementDesc: mercadopago.Recortar(nombre, 13),
	}
	// auto_return sólo con un sitio https que no sea local.
	if strings.HasPrefix(s.Sitio, "https://") && !strings.Contains(s.Sitio, "localhost") {
		o.Config.Online.AutoReturn = "approved"
	}
	clave := "orden:" + ref
	if vence != nil {
		o.ExpirationTime = mercadopago.Duracion(vence.Sub(s.ahora()))
		// La duración cambia con el reloj: va en la clave para que un reintento con otro cuerpo no choque.
		clave += ":" + o.ExpirationTime
	}
	if reintento {
		clave += ":" + *ordenAnterior
	}
	creada, err := s.cliente().CrearOrden(ctx, token, clave, o)
	if err != nil {
		return "", err
	}
	err = s.Pool.EnNegocio(ctx, negocio, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `UPDATE pagos SET link = $2, mp_orden = $3, mp_transaccion = NULL,
			estado = CASE WHEN estado = 'rechazado' THEN 'pendiente' ELSE estado END, actualizado_en = now() WHERE referencia = $1`,
			ref, creada.CheckoutURL, creada.ID)
		return err
	})
	return creada.CheckoutURL, err
}

// negocioDe: el aviso trae el vendedor; de ahí sale el negocio y su token. nil: un aviso de la cuenta
// de la plataforma o de una cuenta que se desconectó.
func (s *Servicio) negocioDe(ctx context.Context, a mercadopago.Aviso) (*string, error) {
	if a.Usuario() == 0 {
		return nil, nil
	}
	var negocio *string
	err := s.Pool.EnTx(ctx, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, "SELECT app.negocio_de_mp($1)::text", a.Usuario()).Scan(&negocio)
	})
	return negocio, err
}

// recibirPago: aviso del tópico payment (pagos creados con preferencias, mientras existan).
func (s *Servicio) recibirPago(ctx context.Context, a mercadopago.Aviso) error {
	id, err := strconv.ParseInt(a.ID(), 10, 64)
	if err != nil {
		return nil
	}
	negocio, err := s.negocioDe(ctx, a)
	if err != nil || negocio == nil {
		return err
	}
	return s.ConciliarPago(ctx, *negocio, id)
}

// recibirOrden: aviso del tópico order (evento «Order (Mercado Pago)»).
func (s *Servicio) recibirOrden(ctx context.Context, a mercadopago.Aviso) error {
	negocio, err := s.negocioDe(ctx, a)
	if err != nil || negocio == nil {
		return err
	}
	return s.ConciliarOrden(ctx, *negocio, a.ID())
}

// noExiste: Mercado Pago no conoce el recurso, y reintentar no lo arregla.
func noExiste(err error) bool {
	var emp *mercadopago.ErrorMP
	return errors.As(err, &emp) && (emp.Estado == http.StatusNotFound || emp.Estado == http.StatusBadRequest)
}

// ConciliarPago trae el pago de Mercado Pago con el token del negocio (el aviso no es la fuente de
// verdad) y lo aplica al pago y al turno. Es idempotente.
func (s *Servicio) ConciliarPago(ctx context.Context, negocio string, id int64) error {
	token, err := s.tokenDe(ctx, negocio)
	if errors.Is(err, ErrSinConexion) || errors.Is(err, ErrReautorizar) {
		s.log().Warn("cobros: aviso de pago sin un token usable", "negocio", negocio, "pago", id)
		return nil
	}
	if err != nil {
		return err
	}
	p, err := s.cliente().LeerPago(ctx, token, id)
	if noExiste(err) {
		return nil
	}
	if err != nil {
		return err
	}
	mp := id
	return s.conciliar(ctx, negocio, observado{Referencia: p.ExternalReference, Estado: estadoDe(p.Status), MPEstado: p.Status, MPPago: &mp,
		Monto: mercadopago.Pesos(p.TransactionAmount), Devuelto: mercadopago.Pesos(p.TransactionRefunds)})
}

// ConciliarOrden trae la orden con el token del negocio y aplica su transacción relevante al pago y
// al turno. Es idempotente.
func (s *Servicio) ConciliarOrden(ctx context.Context, negocio, orden string) error {
	token, err := s.tokenDe(ctx, negocio)
	if errors.Is(err, ErrSinConexion) || errors.Is(err, ErrReautorizar) {
		s.log().Warn("cobros: aviso de orden sin un token usable", "negocio", negocio, "orden", orden)
		return nil
	}
	if err != nil {
		return err
	}
	o, err := s.cliente().LeerOrden(ctx, token, orden)
	if noExiste(err) {
		return nil
	}
	if err != nil {
		return err
	}
	return s.conciliar(ctx, negocio, deOrden(o))
}

// observado es lo que Mercado Pago dice de un cobro, venga de un pago o de una orden.
type observado struct {
	Referencia string
	// Estado en criollo (estadoDe / estadoDeTransaccion) y el crudo, para el panel.
	Estado, MPEstado   string
	MPPago             *int64
	Orden, Transaccion string
	Monto, Devuelto    int64
}

func deOrden(o mercadopago.OrdenLeida) observado {
	ob := observado{Referencia: o.ExternalReference, Orden: o.ID, MPEstado: o.Status}
	t, ok := o.Relevante()
	if !ok {
		// Sin intentos de pago: creada y esperando, o vencida/cancelada.
		ob.Estado = "pendiente"
		if o.Status == "canceled" || o.Status == "expired" {
			ob.Estado = "cancelado"
		}
		return ob
	}
	ob.Estado, ob.MPEstado, ob.Transaccion = estadoDeTransaccion(t.Status), t.Status, t.ID
	if n := t.IDPago(); n != 0 {
		ob.MPPago = &n
	}
	ob.Monto = mercadopago.Pesos(mercadopago.LeerImporte(t.Amount))
	if t.StatusDetail == "partially_refunded" {
		if pagado := mercadopago.LeerImporte(t.PaidAmount); pagado > 0 {
			ob.Devuelto = max(ob.Monto-mercadopago.Pesos(pagado), 0)
		}
	}
	return ob
}

// estadoDeTransaccion: el status de una transacción de una orden, en criollo.
func estadoDeTransaccion(mp string) string {
	switch mp {
	case "processed":
		return "aprobado"
	case "failed":
		return "rechazado"
	case "canceled", "expired":
		return "cancelado"
	case "refunded", "charged_back":
		return "devuelto"
	case "created":
		return "pendiente"
	}
	// processing, action_required, in_review…
	return "en_proceso"
}

// conciliar aplica lo observado y avisa si el turno cambió.
func (s *Servicio) conciliar(ctx context.Context, negocio string, ob observado) error {
	turno, evento, err := s.aplicar(ctx, negocio, ob)
	if err == nil && evento != "" {
		s.avisar(ctx, negocio, turno, evento)
	}
	return err
}

// estadoDe un pago de Mercado Pago, en criollo.
func estadoDe(mp string) string {
	switch mp {
	case "approved":
		return "aprobado"
	case "rejected":
		return "rechazado"
	case "cancelled":
		return "cancelado"
	case "refunded", "charged_back":
		return "devuelto"
	case "in_process", "in_mediation", "authorized":
		return "en_proceso"
	}
	return "pendiente"
}

// aplicar el pago. Aprobado confirma el turno si sigue esperando el pago; si la reserva ya venció y
// otra persona tomó el horario, el turno se cancela y el pago queda aprobado para devolverlo (así
// aparece en el panel, donde está el botón). Rechazado no cambia el turno: el comprador puede
// reintentar hasta que venza, y vencido ya no ocupa el horario.
func (s *Servicio) aplicar(ctx context.Context, negocio string, p observado) (turnoID, evento string, err error) {
	ahora := s.ahora()
	err = s.Pool.EnNegocio(ctx, negocio, func(tx pgx.Tx) error {
		var id, previo string
		var monto int64
		var ordenActual *string
		err := tx.QueryRow(ctx, "SELECT id::text, turno_id::text, monto, estado, mp_orden FROM pagos WHERE referencia = $1 FOR UPDATE",
			p.Referencia).Scan(&id, &turnoID, &monto, &previo, &ordenActual)
		if errors.Is(err, pgx.ErrNoRows) {
			// Un pago de la cuenta que no salió de TengoLibre: no es nuestro.
			return nil
		}
		if err != nil {
			return err
		}
		nuevo := p.Estado
		// Un aviso de una orden anterior (ya reemplazada por otra) no pisa a la actual, salvo que se haya pagado.
		if p.Orden != "" && ordenActual != nil && *ordenActual != p.Orden && nuevo != "aprobado" && nuevo != "devuelto" {
			return nil
		}
		// Un aprobado sólo se mueve a devuelto: otro intento rechazado del mismo checkout no lo pisa.
		if previo == "aprobado" && nuevo != "devuelto" && nuevo != "aprobado" {
			return nil
		}
		if previo == "devuelto" {
			nuevo = "devuelto"
		}
		devuelto := min(p.Devuelto, monto)
		if nuevo == "devuelto" && devuelto == 0 {
			devuelto = monto
		}
		if _, err := tx.Exec(ctx, `UPDATE pagos SET estado = $2, mp_pago = COALESCE($3, mp_pago), mp_estado = $4, devuelto = greatest(devuelto, $5),
			mp_orden = COALESCE($6, mp_orden), mp_transaccion = COALESCE($7, mp_transaccion), actualizado_en = now() WHERE id = $1`,
			id, nuevo, p.MPPago, p.MPEstado, devuelto, nulo(p.Orden), nulo(p.Transaccion)); err != nil {
			return err
		}
		if nuevo != "aprobado" || previo == "aprobado" {
			return nil
		}
		if p.Monto < monto {
			s.log().Error("cobros: pago aprobado por menos de lo pedido", "referencia", p.Referencia, "monto", monto)
			return nil
		}
		var estado string
		var vence *time.Time
		var profesional *string
		var inicio, fin time.Time
		if err := tx.QueryRow(ctx, "SELECT estado, vence_en, profesional_id::text, inicio, fin FROM turnos WHERE id = $1 FOR UPDATE", turnoID).
			Scan(&estado, &vence, &profesional, &inicio, &fin); err != nil {
			return err
		}
		if estado != "pendiente_pago" {
			// Cancelado mientras pagaba: el pago queda para devolver.
			return nil
		}
		ocupado := false
		if vence != nil && !vence.After(ahora) && profesional != nil {
			if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM turnos WHERE profesional_id = $1 AND id <> $2 AND inicio < $4 AND fin > $3
				AND (estado IN ('pendiente', 'confirmado') OR (estado = 'pendiente_pago' AND (vence_en IS NULL OR vence_en > $5))))`,
				*profesional, turnoID, inicio, fin, ahora).Scan(&ocupado); err != nil {
				return err
			}
		}
		if ocupado {
			s.log().Error("cobros: pago aprobado pero el horario ya se lo llevó otro", "turno", turnoID, "referencia", p.Referencia)
			evento = "pago_sin_lugar"
			_, err = tx.Exec(ctx, "UPDATE turnos SET estado = 'cancelado', vence_en = NULL, actualizado_en = now() WHERE id = $1", turnoID)
			return err
		}
		evento = "confirmado"
		// Pagó la seña desde la web: la confirmación es del cliente (el talón lo muestra así).
		_, err = tx.Exec(ctx, `UPDATE turnos SET estado = 'confirmado', confirmado_por = 'web', vence_en = NULL, actualizado_en = now()
			WHERE id = $1`, turnoID)
		return err
	})
	return turnoID, evento, err
}

func nulo(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
