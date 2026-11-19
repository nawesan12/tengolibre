package turnos

import (
	"context"
	"strings"
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
		// Los datos de un cliente nuevo que trae un asistente de IA no son de confianza: no pisan el
		// email de un cliente que ya existe ni lo marcan email_confiable (ver app.cuidar_email).
		if in.DatosDeLaWeb {
			if _, err := tx.Exec(ctx, "SELECT set_config('app.origen', 'web', true)"); err != nil {
				return err
			}
		}
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

// SePuedeConfirmar: lo mismo que chequea CambiarEstado antes de confirmar un pendiente_pago cuya seña
// venció (que el horario siga libre), sin cambiar nada. Para la previa del asistente: así la propuesta
// no se arma para fallar al confirmarla.
func (s *Servicio) SePuedeConfirmar(ctx context.Context, negocioID string, t Turno) error {
	if t.Estado != "pendiente_pago" || t.VenceEn == nil || t.VenceEn.After(s.ahora()) {
		return nil
	}
	return s.Pool.EnNegocio(ctx, negocioID, func(tx pgx.Tx) error {
		libre, err := SigueLibre(ctx, tx, t.ID, s.ahora())
		if err != nil {
			return err
		}
		if !libre {
			return problema.Conflicto("horario_ocupado", "Se venció el plazo para pagar la seña y ese horario ya no está libre. Reprogramalo o cancelalo.")
		}
		return nil
	})
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
		if estado != antes {
			if err := antesDeEmpezar(t, estado, s.ahora()); err != nil {
				return err
			}
		}
		// Como en el panel: un pendiente_pago vencido ya no guardaba el horario; se confirma sólo si sigue libre.
		if t.Estado == "pendiente_pago" && (estado == "pendiente" || estado == "confirmado") && t.VenceEn != nil && !t.VenceEn.After(s.ahora()) {
			libre, err := SigueLibre(ctx, tx, t.ID, s.ahora())
			if err != nil {
				return err
			}
			if !libre {
				return problema.Conflicto("horario_ocupado", "Se venció el plazo para pagar y ese horario ya no está libre. Reprogramalo o cancelalo.")
			}
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
	f := Filtro{Desde: desde, Hasta: hasta, ClienteID: clienteID, Limite: limite, ConCancelados: true, ConVencidos: true}
	if profesionalID != "" {
		f.Profesionales = []string{profesionalID}
	}
	ts, _, err := AgendaFiltrada(ctx, tx, f)
	return ts, err
}

// Filtro de AgendaFiltrada. Todo se filtra en la base, antes del tope: con el tope primero, un salón
// con mucho movimiento dejaba afuera turnos que existían (los cancelados y los de otros profesionales
// ocupaban las filas).
type Filtro struct {
	Desde, Hasta time.Time
	// Profesionales: sólo los de estos (nil: todos). Una lista vacía no nil no trae nada.
	Profesionales []string
	ClienteID     string
	// Cliente: palabras del nombre del cliente (cada una, comienzo de una palabra del nombre; sin
	// tildes ni mayúsculas).
	Cliente []string
	// ConCancelados: también los cancelados. ConVencidos: también los pendiente_pago cuya seña venció
	// (VencidosAl): el horario ya no está guardado.
	ConCancelados, ConVencidos bool
	VencidosAl                 time.Time
	// DesdeAhora: sólo los que todavía no empezaron (cero: sin ese filtro).
	DesdeAhora time.Time
	Limite     int
}

// AgendaFiltrada: los turnos del filtro en orden y si quedaron más afuera del tope.
func AgendaFiltrada(ctx context.Context, tx pgx.Tx, f Filtro) ([]Turno, bool, error) {
	if f.Profesionales != nil && len(f.Profesionales) == 0 {
		return []Turno{}, false, nil
	}
	if f.Limite <= 0 {
		f.Limite = 300
	}
	vencidos := f.VencidosAl
	if vencidos.IsZero() {
		vencidos = time.Now()
	}
	palabras := []string{}
	for _, w := range f.Cliente {
		if w = strings.TrimSpace(w); w != "" {
			palabras = append(palabras, w)
		}
	}
	filas, err := tx.Query(ctx, "SELECT "+columnasTurno+desdeTurnos+` WHERE t.inicio >= $1 AND t.inicio < $2
		AND ($3::text[] IS NULL OR t.profesional_id::text = ANY ($3)) AND ($4 = '' OR t.cliente_id::text = $4)
		AND ($5 OR t.estado <> 'cancelado')
		AND ($6 OR NOT (t.estado = 'pendiente_pago' AND t.vence_en IS NOT NULL AND t.vence_en <= $7))
		AND ($8::timestamptz IS NULL OR t.inicio >= $8)
		AND NOT EXISTS (SELECT 1 FROM unnest($9::text[]) w WHERE position(' ' || w IN ' ' ||
			translate(lower(c.nombre), 'áàäâéèëêíìïîóòöôúùüûñ', 'aaaaeeeeiiiioooouuuun')) = 0)
		ORDER BY t.inicio LIMIT $10`, f.Desde, f.Hasta, f.Profesionales, f.ClienteID, f.ConCancelados, f.ConVencidos, vencidos,
		nilSiCero(f.DesdeAhora), palabras, f.Limite+1)
	if err != nil {
		return nil, false, err
	}
	defer filas.Close()
	out := []Turno{}
	for filas.Next() {
		var t Turno
		if err := escanearTurno(filas, &t); err != nil {
			return nil, false, err
		}
		out = append(out, t)
	}
	if err := filas.Err(); err != nil {
		return nil, false, err
	}
	if len(out) > f.Limite {
		return out[:f.Limite], true, nil
	}
	return out, false, nil
}

func nilSiCero(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	return &t
}
