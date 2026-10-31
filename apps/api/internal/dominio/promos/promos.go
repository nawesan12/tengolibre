// Package promos: cupones de descuento y el programa de referidos. Un cupón baja el precio del turno
// (y por lo tanto la seña) al reservar, con el uso reclamado dentro de la misma transacción para que
// dos reservas a la vez no pasen el tope. Un referido es un código por cliente: quien reserva con él
// le genera al dueño del código un cupón de recompensa de un solo uso.
package promos

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/turnos"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/db"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/problema"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/sesion"
)

// Servicio de promos.
type Servicio struct {
	Pool   *db.Pool
	Sesion *sesion.Servicio
	// Escritura: el chequeo de sólo lectura de la suscripción. nil: no se chequea.
	Escritura func(http.HandlerFunc) http.HandlerFunc
	Ahora     func() time.Time
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
}

const columnasCupon = `id::text, codigo, tipo, valor, minimo, max_usos, usos, desde, hasta, activo, referido_id::text, creado_en`

func escanearCupon(f pgx.Row, c *Cupon) error {
	return f.Scan(&c.ID, &c.Codigo, &c.Tipo, &c.Valor, &c.Minimo, &c.MaxUsos, &c.Usos, &c.Desde, &c.Hasta, &c.Activo, &c.ReferidoID, &c.CreadoEn)
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

func azar(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return strings.ToUpper(hex.EncodeToString(b))
}

// alfabeto de los códigos de referido: sin 0/O ni 1/I, que se confunden al dictarlos.
const alfabeto = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"

func codigoReferido() string {
	b := make([]byte, 6)
	_, _ = rand.Read(b)
	for i := range b {
		b[i] = alfabeto[int(b[i])%len(alfabeto)]
	}
	return string(b)
}

// AlReservar (turnos.Extra): aplica el cupón y registra el referido. Un cupón que no sirve frena la
// reserva (mejor que cobrarle a alguien el precio entero sin avisar); un referido que no sirve se
// ignora, como en Jiku: el premio es para otro.
func (s *Servicio) AlReservar(ctx context.Context, tx pgx.Tx, r *turnos.Reserva) error {
	t := r.Turno
	if r.Pedido.Cupon != "" {
		c, motivo, err := vigente(ctx, tx, r.Pedido.Cupon, &t.Precio, r.Ahora)
		if err != nil {
			return err
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
	if _, err := tx.Exec(ctx, `INSERT INTO referido_usos (negocio_id, referido_id, turno_id, cliente_id) VALUES (app.negocio_actual(), $1, $2, $3)`,
		id, r.Turno.ID, r.Turno.Cliente.ID); err != nil {
		return err
	}
	if a.Tipo == nil || a.Valor == nil {
		return nil
	}
	_, err = tx.Exec(ctx, `INSERT INTO cupones (negocio_id, codigo, tipo, valor, max_usos, desde, hasta, referido_id)
		VALUES (app.negocio_actual(), $1, $2, $3, 1, $4, $5, $6)`,
		"REF-"+codigo+"-"+azar(4), *a.Tipo, *a.Valor, r.Ahora, r.Ahora.AddDate(0, 0, 90), id)
	return err
}

// DespuesDeReservar (turnos.Extra): nada.
func (s *Servicio) DespuesDeReservar(context.Context, string, *turnos.Turno) {}

// AlCancelar (turnos.Extra): el uso del cupón no se devuelve (como Jiku).
func (s *Servicio) AlCancelar(context.Context, pgx.Tx, string) error { return nil }
