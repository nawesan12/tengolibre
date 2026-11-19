// Package promos: cupones de descuento y el programa de referidos. Un cupón baja el precio del turno
// (y por lo tanto la seña) al reservar, con el uso reclamado dentro de la misma transacción para que
// dos reservas a la vez no pasen el tope. Un referido es un código por cliente: quien reserva con él
// le genera al dueño del código un cupón de recompensa de un solo uso.
package promos

import (
	"context"
	"crypto/rand"
	"errors"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/planes"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/turnos"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/db"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/problema"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/sesion"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/tope"
)

// Servicio de promos.
type Servicio struct {
	Pool   *db.Pool
	Sesion *sesion.Servicio
	// Escritura: el chequeo de sólo lectura de la suscripción. nil: no se chequea.
	Escritura func(http.HandlerFunc) http.HandlerFunc
	Ahora     func() time.Time
	// TopeValidar: cuántas validaciones de cupón o código de referido por IP cada VentanaValidar (0:
	// ValidarPorIP). Sin tope, los códigos se adivinan probando.
	TopeValidar int
	limite      tope.Contador
}

// Validaciones públicas de cupones y referidos por IP (sumadas) en cada ventana.
const (
	ValidarPorIP   = 20
	VentanaValidar = 10 * time.Minute
)

// permitirValidar: el tope por IP de las validaciones públicas (429 al pasarse).
func (s *Servicio) permitirValidar(r *http.Request) error {
	max := ValidarPorIP
	if s.TopeValidar > 0 {
		max = s.TopeValidar
	}
	if !s.limite.Permitir("validar:"+tope.IP(r), max, VentanaValidar, s.ahora()) {
		return tope.Demasiados("demasiados_intentos", "Probaste muchos códigos seguidos. Esperá unos minutos y volvé a intentar.")
	}
	return nil
}

func (s *Servicio) ahora() time.Time {
	if s.Ahora != nil {
		return s.Ahora()
	}
	return time.Now()
}

func (s *Servicio) escritura(h http.HandlerFunc) http.HandlerFunc {
	if s.Escritura == nil {
		return h
	}
	return s.Escritura(h)
}

// Cupon de descuento.
type Cupon struct {
	ID     string `json:"id"`
	Codigo string `json:"codigo"`
	// Tipo: porcentaje (0-100) o fijo (pesos).
	Tipo       string     `json:"tipo"`
	Valor      int64      `json:"valor"`
	Minimo     *int64     `json:"minimo"`
	MaxUsos    *int       `json:"maxUsos"`
	Usos       int        `json:"usos"`
	Desde      time.Time  `json:"desde"`
	Hasta      *time.Time `json:"hasta"`
	Activo     bool       `json:"activo"`
	ReferidoID *string    `json:"referidoId,omitempty"`
	CreadoEn   time.Time  `json:"creadoEn"`
	// Restricciones (franja.go). Dias: días de la semana en que vale (0 = domingo; vacío: todos).
	// HoraDesde y HoraHasta ("14:00"): el turno tiene que empezar en esa franja. AntiguedadDias:
	// sólo para clientes cuyo primer turno atendido fue hace al menos esos días.
	Dias           []int   `json:"dias"`
	HoraDesde      *string `json:"horaDesde"`
	HoraHasta      *string `json:"horaHasta"`
	AntiguedadDias *int    `json:"antiguedadDias"`
	// Condiciones: las restricciones en palabras («Vale los martes de 14 a 16 h.»). Vacío: ninguna.
	Condiciones string `json:"condiciones"`
}

const columnasCupon = `id::text, codigo, tipo, valor, minimo, max_usos, usos, desde, hasta, activo, referido_id::text, creado_en,
	dias, to_char(hora_desde, 'HH24:MI'), to_char(hora_hasta, 'HH24:MI'), antiguedad_dias`

func escanearCupon(f pgx.Row, c *Cupon) error {
	if err := f.Scan(&c.ID, &c.Codigo, &c.Tipo, &c.Valor, &c.Minimo, &c.MaxUsos, &c.Usos, &c.Desde, &c.Hasta, &c.Activo, &c.ReferidoID, &c.CreadoEn,
		&c.Dias, &c.HoraDesde, &c.HoraHasta, &c.AntiguedadDias); err != nil {
		return err
	}
	if c.Dias == nil {
		c.Dias = []int{}
	}
	c.Condiciones = Condiciones(*c)
	return nil
}

// Normalizar un código: sin espacios y en mayúsculas (se busca sin distinguir).
func Normalizar(codigo string) string { return strings.ToUpper(strings.TrimSpace(codigo)) }

// Descuento en pesos que un cupón le hace a un precio (calculateCouponDiscount): siempre entre 0 y
// el precio, nunca un total negativo.
func Descuento(precio int64, tipo string, valor int64) int64 {
	if precio <= 0 || valor <= 0 {
		return 0
	}
	d := valor
	if tipo == "porcentaje" {
		d = (precio*valor + 50) / 100
	}
	return max(0, min(d, precio))
}

// vigente busca un cupón activo por código y dice por qué no sirve (motivo vacío: sirve).
func vigente(ctx context.Context, tx pgx.Tx, codigo string, precio *int64, ahora time.Time) (Cupon, string, error) {
	var c Cupon
	err := escanearCupon(tx.QueryRow(ctx, "SELECT "+columnasCupon+" FROM cupones WHERE codigo = $1 AND activo", Normalizar(codigo)), &c)
	if errors.Is(err, pgx.ErrNoRows) {
		return c, "Cupón no encontrado", nil
	}
	if err != nil {
		return c, "", err
	}
	switch {
	case c.Desde.After(ahora) || (c.Hasta != nil && !c.Hasta.After(ahora)):
		return c, "Cupón vencido", nil
	case c.MaxUsos != nil && c.Usos >= *c.MaxUsos:
		return c, "Cupón agotado", nil
	case precio != nil && c.Minimo != nil && *precio < *c.Minimo:
		return c, "No aplica para este servicio", nil
	}
	return c, "", nil
}

// AjustesReferidos: el programa de referidos del negocio.
type AjustesReferidos struct {
	Activo bool `json:"activo"`
	// Recompensa para quien refirió: un cupón de un uso de este tipo y valor. Sin tipo: no hay premio.
	Tipo  *string `json:"tipo"`
	Valor *int64  `json:"valor"`
	// Tope: cuántas veces sirve el código de una persona. nil: sin tope.
	Tope *int `json:"tope"`
}

func leerReferidos(ctx context.Context, tx pgx.Tx) (AjustesReferidos, error) {
	var a AjustesReferidos
	err := tx.QueryRow(ctx, `SELECT referidos_activo, referidos_tipo, referidos_valor, referidos_tope FROM ajustes
		WHERE negocio_id = app.negocio_actual()`).Scan(&a.Activo, &a.Tipo, &a.Valor, &a.Tope)
	return a, err
}

// alfabeto de los códigos de referido: sin 0/O ni 1/I, que se confunden al dictarlos.
const alfabeto = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"

// LargoReferido: 32^8 (un billón) de códigos posibles, con el límite de intentos de validar no se
// adivina uno ajeno.
const LargoReferido = 8

// alAzar: n caracteres de alfabeto (32 símbolos: un byte módulo 32 no tiene sesgo).
func alAzar(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	for i := range b {
		b[i] = alfabeto[int(b[i])%len(alfabeto)]
	}
	return string(b)
}

func codigoReferido() string { return alAzar(LargoReferido) }

// liberarVencidos devuelve los usos del cupón (por código) que tomaron reservas pendiente_pago
// vencidas: ese horario nunca se pagó, y el turno no cambia de estado al vencer (el trigger
// promos_del_turno no se entera). Lo mismo con los usos del código de referido sin premiar.
func liberarVencidos(ctx context.Context, tx pgx.Tx, cupon, referido string, ahora time.Time) error {
	if cupon != "" {
		if _, err := tx.Exec(ctx, `WITH usados AS (
				DELETE FROM cupon_usos u USING turnos t, cupones c
				WHERE c.codigo = $1 AND u.cupon_id = c.id AND t.id = u.turno_id AND t.estado = 'pendiente_pago' AND t.vence_en <= $2
				RETURNING u.cupon_id)
			UPDATE cupones c SET usos = greatest(c.usos - (SELECT count(*) FROM usados), 0)
			WHERE c.id IN (SELECT cupon_id FROM usados)`, Normalizar(cupon), ahora); err != nil {
			return err
		}
	}
	if referido != "" {
		if _, err := tx.Exec(ctx, `DELETE FROM referido_usos u USING turnos t, referidos r
			WHERE r.codigo = $1 AND u.referido_id = r.id AND t.id = u.turno_id AND u.premiado_en IS NULL
				AND t.estado = 'pendiente_pago' AND t.vence_en <= $2`, Normalizar(referido), ahora); err != nil {
			return err
		}
	}
	return nil
}

// AlReservar (turnos.Extra): aplica el cupón y registra el referido. Un cupón que no sirve frena la
// reserva (mejor que cobrarle a alguien el precio entero sin avisar); un referido que no sirve se
// ignora, como en Jiku: el premio es para otro.
func (s *Servicio) AlReservar(ctx context.Context, tx pgx.Tx, r *turnos.Reserva) error {
	t := r.Turno
	if err := liberarVencidos(ctx, tx, r.Pedido.Cupon, r.Pedido.Referido, r.Ahora); err != nil {
		return err
	}
	if r.Pedido.Cupon != "" {
		// Cupones y referidos vienen con Equipo: con un plan más chico no se aplican.
		if ok, err := planes.Tiene(ctx, tx, r.Ahora, planes.Cupones); err != nil || !ok {
			if err == nil {
				err = problema.NoProcesable("cupon_invalido", "Este negocio no tiene cupones activos.")
			}
			return err
		}
		c, motivo, err := vigente(ctx, tx, r.Pedido.Cupon, &t.Precio, r.Ahora)
		if err != nil {
			return err
		}
		if motivo == "" {
			if motivo, err = restriccion(ctx, tx, c, t.Inicio, t.Cliente.ID, r.Ahora); err != nil {
				return err
			}
		}
		if motivo != "" {
			return problema.NoProcesable("cupon_invalido", motivo+".")
		}
		// El tope viaja con el UPDATE: dos reservas a la vez no pasan max_usos.
		tag, err := tx.Exec(ctx, "UPDATE cupones SET usos = usos + 1 WHERE id = $1 AND (max_usos IS NULL OR usos < max_usos)", c.ID)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return problema.NoProcesable("cupon_invalido", "Cupón agotado.")
		}
		d := Descuento(t.Precio, c.Tipo, c.Valor)
		if _, err := tx.Exec(ctx, "UPDATE turnos SET precio = precio - $2 WHERE id = $1", t.ID, d); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `INSERT INTO cupon_usos (negocio_id, cupon_id, turno_id, cliente_id, descuento)
			VALUES (app.negocio_actual(), $1, $2, $3, $4)`, c.ID, t.ID, t.Cliente.ID, d); err != nil {
			return err
		}
		t.Precio -= d
		t.Descuento += d
	}
	if r.Pedido.Referido != "" {
		if ok, err := planes.Tiene(ctx, tx, r.Ahora, planes.Referidos); err != nil || !ok {
			return err
		}
		return s.referir(ctx, tx, r)
	}
	return nil
}

func (s *Servicio) referir(ctx context.Context, tx pgx.Tx, r *turnos.Reserva) error {
	a, err := leerReferidos(ctx, tx)
	if err != nil || !a.Activo {
		return err
	}
	var id, codigo, duenio string
	var usos int
	var repetido bool
	err = tx.QueryRow(ctx, `SELECT r.id::text, r.codigo, r.cliente_id::text, (SELECT count(*) FROM referido_usos u WHERE u.referido_id = r.id),
			EXISTS (SELECT 1 FROM referido_usos u WHERE u.referido_id = r.id AND u.cliente_id = $2)
		FROM referidos r WHERE r.codigo = $1`, Normalizar(r.Pedido.Referido), r.Turno.Cliente.ID).
		Scan(&id, &codigo, &duenio, &usos, &repetido)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	// Ni uno mismo, ni la misma persona dos veces, ni más allá del tope.
	if duenio == r.Turno.Cliente.ID || repetido || (a.Tope != nil && usos >= *a.Tope) {
		return nil
	}
	// El premio no se da acá: lo da app.premiar_referido cuando el turno queda confirmado o atendido
	// (trigger promos_del_turno), así un pendiente_pago que nunca se paga no premia. Un turno que ya
	// nace confirmado (el panel, que no cobra seña) se premia en el momento.
	if _, err := tx.Exec(ctx, `INSERT INTO referido_usos (negocio_id, referido_id, turno_id, cliente_id, premio_desde)
		VALUES (app.negocio_actual(), $1, $2, $3, $4)`, id, r.Turno.ID, r.Turno.Cliente.ID, r.Ahora); err != nil {
		return err
	}
	if r.Origen == "panel" && (r.Turno.Estado == "confirmado" || r.Turno.Estado == "atendido") {
		_, err = tx.Exec(ctx, "SELECT app.premiar_referido($1)", r.Turno.ID)
	}
	return err
}

// DespuesDeReservar (turnos.Extra): nada.
func (s *Servicio) DespuesDeReservar(context.Context, string, *turnos.Turno) {}

// AlCancelar (turnos.Extra): nada acá. Al cancelar, el trigger promos_del_turno (00029) devuelve el
// uso del cupón y el lugar del referido, por cualquier camino que se cancele.
func (s *Servicio) AlCancelar(context.Context, pgx.Tx, string) error { return nil }
