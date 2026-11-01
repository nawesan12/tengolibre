package turnos

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/problema"
)

// Operaciones de la agenda para otros dominios (el asistente del negocio por MCP). Hacen lo mismo
// que las rutas del panel (las mismas reglas, extras y avisos) y corren con la sesión que traiga el
// contexto: quien llama la arma con el negocio y quién la hace.

// Reservar como desde el panel: el turno queda confirmado. Nunca es sobreturno.
func (s *Servicio) Reservar(ctx context.Context, negocioID string, in Pedido) (Turno, error) {
	in.Forzar, in.UsarMembresia = false, false
	var t Turno
	err := s.Pool.EnNegocio(ctx, negocioID, func(tx pgx.Tx) (err error) {
		if t, err = reservar(ctx, tx, in, "panel", s.ahora()); err != nil {
			return err
		}
		return s.alReservar(ctx, tx, &t, in, "panel")
	})
	if err != nil {
		return t, err
	}
	s.despuesDeReservar(ctx, negocioID, &t)
	s.avisar(ctx, negocioID, t.ID, "reservado")
	return t, nil
}

// OrigenIA: el turno lo sacó el asistente de IA de una persona (el servidor MCP público).
const OrigenIA = "ia"

// ReservarIA reserva para el asistente de IA de una persona (servidor MCP público): las mismas
// reglas que la página pública (penalidades, seña con link de Mercado Pago, lo cargado no es de
// confianza), sin cupones ni membresías, y con origen "ia" para contarlas en el panel.
func (s *Servicio) ReservarIA(ctx context.Context, negocioID string, in Pedido) (Turno, error) {
	in.Forzar, in.UsarMembresia, in.ClienteID, in.SinAnticipacion = false, false, "", false
	in.Cupon, in.Referido, in.SerieID, in.reglas = "", "", "", nil
	var t Turno
	err := s.Pool.EnNegocio(ctx, negocioID, func(tx pgx.Tx) (err error) {
		if _, err := tx.Exec(ctx, "SELECT set_config('app.origen', 'web', true)"); err != nil {
			return err
		}
		if t, err = reservar(ctx, tx, in, OrigenIA, s.ahora()); err != nil {
			return err
		}
		return s.alReservar(ctx, tx, &t, in, OrigenIA)
	})
	if err != nil {
		return t, err
	}
	s.despuesDeReservar(ctx, negocioID, &t)
	s.avisar(ctx, negocioID, t.ID, "reservado")
	return t, nil
}

// Ver un turno del negocio.
func (s *Servicio) Ver(ctx context.Context, negocioID, turnoID string) (Turno, error) {
	var t Turno
	err := s.Pool.EnNegocio(ctx, negocioID, func(tx pgx.Tx) (err error) {
		t, err = leerTurno(ctx, tx, "t.id = $1", turnoID, false)
		return err
	})
	return t, err
}

// CambiarEstado con los mismos pasos permitidos que el panel. Devuelve también el estado que tenía.
func (s *Servicio) CambiarEstado(ctx context.Context, negocioID, turnoID, estado string) (Turno, string, error) {
	var t Turno
	var antes string
	err := s.Pool.EnNegocio(ctx, negocioID, func(tx pgx.Tx) error {
		var err error
		if t, err = leerTurno(ctx, tx, "t.id = $1", turnoID, true); err != nil {
			return err
		}
		antes = t.Estado
		if !PuedePasar(t.Estado, estado) {
			return problema.Conflicto("estado", "No se puede pasar de "+t.Estado+" a "+estado+".")
		}
		if _, err := tx.Exec(ctx, `UPDATE turnos SET estado = $2,
			vence_en = CASE WHEN $2 = 'pendiente_pago' THEN vence_en END, actualizado_en = now() WHERE id = $1`, t.ID, estado); err != nil {
			return err
		}
		if estado == "cancelado" && antes != "cancelado" {
			if err := s.alCancelar(ctx, tx, t.ID); err != nil {
				return err
			}
		}
		t, err = leerTurno(ctx, tx, "t.id = $1", t.ID, false)
		return err
	})
	if err != nil {
		return t, antes, err
	}
	if t.Estado != antes {
		s.avisar(ctx, negocioID, t.ID, t.Estado)
	}
	return t, antes, nil
}

// Reprogramar con las reglas de una reserva nueva (sin sobreturno). profesionalID vacío: el mismo.
func (s *Servicio) Reprogramar(ctx context.Context, negocioID, turnoID string, inicio time.Time, profesionalID string) (Turno, error) {
	var t Turno
	err := s.Pool.EnNegocio(ctx, negocioID, func(tx pgx.Tx) error {
		actual, err := leerTurno(ctx, tx, "t.id = $1", turnoID, true)
		if err != nil {
			return err
		}
		t, err = reprogramar(ctx, tx, actual, inicio, profesionalID, s.ahora(), false)
		return err
	})
	if err != nil {
		return t, err
	}
	s.avisar(ctx, negocioID, t.ID, "reprogramado")
	return t, nil
}

// Libres de un servicio en una fecha (AAAA-MM-DD, en la zona del negocio), como la página pública.
func (s *Servicio) Libres(ctx context.Context, negocioID, servicioID, profesionalID, fecha string) ([]HorarioLibre, error) {
	var out []HorarioLibre
	err := s.Pool.EnNegocio(ctx, negocioID, func(tx pgx.Tx) error {
		c, err := cargarContexto(ctx, tx)
		if err != nil {
			return err
		}
		out, err = libres(ctx, tx, c, servicioID, profesionalID, fecha, s.ahora())
		return err
	})
	return out, err
}

// Agenda: los turnos que empiezan entre desde y hasta (sin incluir hasta), opcionalmente de un
// profesional o de un cliente, en orden. Tope: limite.
func Agenda(ctx context.Context, tx pgx.Tx, desde, hasta time.Time, profesionalID, clienteID string, limite int) ([]Turno, error) {
	filas, err := tx.Query(ctx, "SELECT "+columnasTurno+desdeTurnos+` WHERE t.inicio >= $1 AND t.inicio < $2
		AND ($3 = '' OR t.profesional_id::text = $3) AND ($4 = '' OR t.cliente_id::text = $4)
		ORDER BY t.inicio LIMIT $5`, desde, hasta, profesionalID, clienteID, limite)
	if err != nil {
		return nil, err
	}
	defer filas.Close()
	out := []Turno{}
	for filas.Next() {
		var t Turno
		if err := escanearTurno(filas, &t); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, filas.Err()
}
