package turnos

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
)

// Extra engancha a la reserva lo que tiene que ver con la plata (membresías, cupones y referidos,
// cobros), sin que turnos los conozca. Corren en el orden de Servicio.Extras: el cobro va último,
// porque cobra sobre el precio que dejaron los anteriores.
type Extra interface {
	// AlReservar corre dentro de la transacción de la reserva, con el turno ya insertado. Puede
	// cambiar el precio, el estado o el vencimiento (en la base y en r.Turno); si devuelve error, la
	// reserva entera se revierte.
	AlReservar(ctx context.Context, tx pgx.Tx, r *Reserva) error
	// DespuesDeReservar corre ya confirmada la reserva, afuera de la transacción (p. ej. pedirle el
	// link de pago a Mercado Pago). No puede fallar la reserva.
	DespuesDeReservar(ctx context.Context, negocioID string, t *Turno)
	// AlCancelar corre en la transacción que cancela un turno (devolver la visita, anular el cobro).
	AlCancelar(ctx context.Context, tx pgx.Tx, turnoID string) error
}

// Estimador: un Extra que sabe cuánto se cobraría por adelantado sobre un precio, para mostrarlo
// antes de reservar (volver a reservar, en repetir.go). Lo implementa cobros. nil: no se cobra.
type Estimador interface {
	Estimar(ctx context.Context, tx pgx.Tx, precio int64) (*Pago, error)
}

// Reserva que se está haciendo: el turno recién creado y lo que pidió quien reserva.
type Reserva struct {
	Turno  *Turno
	Pedido Pedido
	// Origen: web o panel.
	Origen string
	// Identificado: la reserva web la hizo el socio con sesión y pidió usar su membresía.
	Identificado bool
	Ahora        time.Time
}

// Pago que pide la reserva, cuando el negocio cobra seña o total.
type Pago struct {
	Monto int64  `json:"monto"`
	Modo  string `json:"modo"`
	// URL de Checkout Pro. Vacía si Mercado Pago no respondió: se pide de nuevo desde el link del turno.
	URL string `json:"url,omitempty"`
}

func (s *Servicio) alReservar(ctx context.Context, tx pgx.Tx, t *Turno, in Pedido, origen string) error {
	r := &Reserva{Turno: t, Pedido: in, Origen: origen, Identificado: in.UsarMembresia && in.ClienteID != "" && origen == "web", Ahora: s.ahora()}
	for _, e := range s.Extras {
		if err := e.AlReservar(ctx, tx, r); err != nil {
			return err
		}
	}
	return nil
}

func (s *Servicio) despuesDeReservar(ctx context.Context, negocioID string, t *Turno) {
	for _, e := range s.Extras {
		e.DespuesDeReservar(ctx, negocioID, t)
	}
}

func (s *Servicio) alCancelar(ctx context.Context, tx pgx.Tx, turnoID string) error {
	for _, e := range s.Extras {
		if err := e.AlCancelar(ctx, tx, turnoID); err != nil {
			return err
		}
	}
	return nil
}

func (s *Servicio) estimar(ctx context.Context, tx pgx.Tx, precio int64) (*Pago, error) {
	for _, e := range s.Extras {
		if x, ok := e.(Estimador); ok {
			p, err := x.Estimar(ctx, tx, precio)
			if err != nil || p != nil {
				return p, err
			}
		}
	}
	return nil, nil
}
