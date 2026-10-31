// Package membresias: un negocio que vende "cuatro cortes al mes" en vez de cobrar cada visita.
// Planes, socios, las visitas de cada período y el consumo al reservar.
//
// El saldo de una membresía es la suma de sus movimientos (membresia_creditos), nunca una columna:
// un contador y un historial son dos registros del mismo hecho y se desfasan.
package membresias

import (
	"context"
	"errors"
	"net/http"
	"slices"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/turnos"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/db"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/sesion"
)

// Servicio de membresías.
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

// Plan de membresía.
type Plan struct {
	ID            string   `json:"id"`
	Nombre        string   `json:"nombre"`
	Descripcion   string   `json:"descripcion"`
	Precio        int64    `json:"precio"`
	Visitas       int      `json:"visitas"`
	DuracionDias  int      `json:"duracionDias"`
	PrioridadDias int      `json:"prioridadDias"`
	Beneficios    []string `json:"beneficios"`
	TopeSocios    *int     `json:"topeSocios"`
	// Servicios que cubre la visita. Vacío: todos.
	Servicios []string `json:"servicios"`
	Activo    bool     `json:"activo"`
	Orden     int      `json:"orden"`
}

const columnasPlan = `id::text, nombre, descripcion, precio, visitas, duracion_dias, prioridad_dias, beneficios, tope_socios,
	servicios::text[], activo, orden`

func escanearPlan(f pgx.Row, p *Plan) error {
	return f.Scan(&p.ID, &p.Nombre, &p.Descripcion, &p.Precio, &p.Visitas, &p.DuracionDias, &p.PrioridadDias, &p.Beneficios,
		&p.TopeSocios, &p.Servicios, &p.Activo, &p.Orden)
}

// Cubre dice si el plan cubre un servicio.
func (p Plan) Cubre(servicioID string) bool {
	return len(p.Servicios) == 0 || slices.Contains(p.Servicios, servicioID)
}

// Saldo de visitas de una membresía.
func Saldo(ctx context.Context, tx pgx.Tx, membresiaID string) (int, error) {
	var n int
	err := tx.QueryRow(ctx, "SELECT coalesce(sum(cantidad), 0) FROM membresia_creditos WHERE membresia_id = $1", membresiaID).Scan(&n)
	return n, err
}

// AlReservar (turnos.Extra): si el cliente tiene una membresía vigente que cubre el servicio y le
// quedan visitas, gasta una y el turno queda cubierto. Como en Jiku, sólo en las reservas del panel:
// la página pública todavía no identifica al cliente (cualquiera puede escribir un teléfono), y
// gastarle una visita a otro sería peor que cobrarle la seña.
//
// La membresía se bloquea y el saldo se relee acá adentro: dos reservas a la vez no se llevan la
// misma última visita. Sin visitas, la reserva sigue normal (no falla).
func (s *Servicio) AlReservar(ctx context.Context, tx pgx.Tx, r *turnos.Reserva) error {
	t := r.Turno
	if r.Origen != "panel" || t.ServicioID == nil {
		return nil
	}
	var id string
	var p Plan
	err := tx.QueryRow(ctx, `SELECT m.id::text, p.servicios::text[] FROM membresias m JOIN membresia_planes p ON p.id = m.plan_id
		WHERE m.cliente_id = $1 AND m.estado = 'activa' AND m.desde <= $2 AND m.hasta > $2
		ORDER BY m.hasta DESC LIMIT 1 FOR UPDATE OF m`, t.Cliente.ID, r.Ahora).Scan(&id, &p.Servicios)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	if !p.Cubre(*t.ServicioID) {
		return nil
	}
	saldo, err := Saldo(ctx, tx, id)
	if err != nil || saldo < 1 {
		return err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO membresia_creditos (negocio_id, membresia_id, cantidad, motivo, turno_id)
		VALUES (app.negocio_actual(), $1, -1, 'Turno reservado', $2)`, id, t.ID); err != nil {
		return err
	}
	_, err = tx.Exec(ctx, "UPDATE turnos SET membresia_id = $2 WHERE id = $1", t.ID, id)
	return err
}

// DespuesDeReservar (turnos.Extra): nada.
func (s *Servicio) DespuesDeReservar(context.Context, string, *turnos.Turno) {}

// AlCancelar (turnos.Extra): devuelve la visita de un turno cubierto (si no, el socio pierde el
// horario y la visita). Idempotente: sólo si gastó y todavía no se devolvió.
func (s *Servicio) AlCancelar(ctx context.Context, tx pgx.Tx, turnoID string) error {
	_, err := tx.Exec(ctx, `INSERT INTO membresia_creditos (negocio_id, membresia_id, cantidad, motivo, turno_id)
		SELECT negocio_id, membresia_id, 1, 'Turno cancelado', turno_id FROM membresia_creditos g
		WHERE g.turno_id = $1 AND g.cantidad < 0
			AND NOT EXISTS (SELECT 1 FROM membresia_creditos d WHERE d.turno_id = $1 AND d.cantidad > 0)
		LIMIT 1`, turnoID)
	return err
}
