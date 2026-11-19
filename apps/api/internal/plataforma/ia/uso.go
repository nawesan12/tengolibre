package ia

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/db"
)

// Topes de uso. Cero: sin ese tope.
type Topes struct {
	// CupoMensual: respuestas con IA por negocio y por mes (lo que promete el plan; si no promete
	// nada, un valor conservador).
	CupoMensual int
	// TopeDiario: respuestas con IA por negocio y por día, para frenar abusos. Con CupoDe es el techo
	// de seguridad: el tope diario de cada plan sale del plan y no lo pasa.
	TopeDiario int
	// PresupuestoMicro: lo que puede gastar toda la plataforma en el mes, en micro-dólares. Pasado
	// eso la IA se apaga y queda sólo el motor propio.
	PresupuestoMicro int64
}

// Agotado: no hay más IA por ahora. Mensaje es para mostrarle a la persona tal cual.
type Agotado struct {
	Motivo  string // mes | dia | plataforma
	Mensaje string
}

func (a *Agotado) Error() string { return "ia: tope " + a.Motivo }

// Medidor lleva la cuenta en ia_uso y aplica los topes. Los días y meses son los de Argentina.
type Medidor struct {
	Pool  *db.Pool
	Topes Topes
	Ahora func() time.Time
	// CupoDe: el cupo mensual y el tope diario del plan del negocio fijado en tx (nil: los de Topes
	// para todos). Así el cupo de un plan grande se puede usar de verdad en el mes.
	CupoDe func(ctx context.Context, tx pgx.Tx, ahora time.Time) (mensual, diario int, err error)
}

func (m *Medidor) ahora() time.Time {
	if m.Ahora != nil {
		return m.Ahora()
	}
	return time.Now()
}

// cupo del negocio fijado en tx: el mensual y el diario (el del plan, sin pasar el techo de Topes).
func (m *Medidor) cupo(ctx context.Context, tx pgx.Tx) (int, int, error) {
	if m.CupoDe == nil {
		return m.Topes.CupoMensual, m.Topes.TopeDiario, nil
	}
	mensual, diario, err := m.CupoDe(ctx, tx, m.ahora())
	if m.Topes.TopeDiario > 0 && (diario == 0 || diario > m.Topes.TopeDiario) {
		diario = m.Topes.TopeDiario
	}
	return mensual, diario, err
}

var zonaAR = func() *time.Location {
	z, err := time.LoadLocation("America/Argentina/Buenos_Aires")
	if err != nil {
		return time.FixedZone("UTC-3", -3*3600)
	}
	return z
}()

func (m *Medidor) hoy() (dia, mes time.Time) {
	a := time.Now()
	if m.Ahora != nil {
		a = m.Ahora()
	}
	a = a.In(zonaAR)
	dia = time.Date(a.Year(), a.Month(), a.Day(), 0, 0, 0, 0, time.UTC)
	return dia, time.Date(a.Year(), a.Month(), 1, 0, 0, 0, 0, time.UTC)
}

// Estado del cupo de un negocio.
type Estado struct {
	CupoMensual int `json:"cupoMensual"`
	UsadasMes   int `json:"usadasMes"`
	UsadasHoy   int `json:"usadasHoy"`
	TopeDiario  int `json:"topeDiario"`
}

// Leer cuánto lleva usado el negocio este mes y hoy.
func (m *Medidor) Leer(ctx context.Context, negocio string) (Estado, error) {
	var e Estado
	dia, mes := m.hoy()
	err := m.Pool.EnNegocio(ctx, negocio, func(tx pgx.Tx) (err error) {
		if e.CupoMensual, e.TopeDiario, err = m.cupo(ctx, tx); err != nil {
			return err
		}
		return tx.QueryRow(ctx, `SELECT coalesce(sum(respuestas), 0)::int, coalesce(sum(respuestas) FILTER (WHERE dia = $2), 0)::int
			FROM ia_uso WHERE dia >= $1`, mes, dia).Scan(&e.UsadasMes, &e.UsadasHoy)
	})
	return e, err
}

// EstimadoMicro: lo que se reserva del presupuesto de la plataforma por respuesta, antes de llamar
// al modelo (unas 4 llamadas de Haiku con el caché frío). Al anotar se ajusta a lo real.
const EstimadoMicro int64 = 6000

// Reserva: una respuesta ya contada en ia_uso antes de llamar al modelo. Se cierra con Anotar (lo
// real) o con Devolver (no llegó al modelo).
type Reserva struct {
	negocio, ruta string
	dia           time.Time
	estimado      int64
}

func (m *Medidor) agotado(e Estado, gasto int64) error {
	if e.CupoMensual > 0 && e.UsadasMes >= e.CupoMensual {
		return &Agotado{Motivo: "mes", Mensaje: fmt.Sprintf("Ya usaste las %d respuestas con IA de este mes. Sigo contestando lo "+
			"básico (agenda, horarios libres, mover y cancelar turnos); el mes que viene vuelve todo, o antes si pasás a un plan más grande.", e.CupoMensual)}
	}
	if e.TopeDiario > 0 && e.UsadasHoy >= e.TopeDiario {
		return &Agotado{Motivo: "dia", Mensaje: "Por hoy llegaste al tope de respuestas con IA. Sigo contestando lo básico " +
			"(agenda, horarios libres, mover y cancelar turnos); mañana vuelve todo."}
	}
	if m.Topes.PresupuestoMicro > 0 && gasto >= m.Topes.PresupuestoMicro {
		return &Agotado{Motivo: "plataforma", Mensaje: "La IA está en pausa por el resto del mes. Sigo contestando lo " +
			"básico (agenda, horarios libres, mover y cancelar turnos)."}
	}
	return nil
}

// Reservar una respuesta antes de llamar al modelo: chequea los topes y la cuenta en la misma
// transacción, en fila (un advisory lock por negocio y otro para el presupuesto común). Antes se
// chequeaba y se anotaba al terminar: N pedidos en paralelo pasaban todos el tope. Devuelve *Agotado
// si no se puede.
func (m *Medidor) Reservar(ctx context.Context, negocio, ruta string) (*Reserva, error) {
	dia, mes := m.hoy()
	r := &Reserva{negocio: negocio, ruta: ruta, dia: dia}
	err := m.Pool.EnNegocio(ctx, negocio, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, "SELECT pg_advisory_xact_lock(hashtext('ia:' || $1))", negocio); err != nil {
			return err
		}
		cupo, diario, err := m.cupo(ctx, tx)
		if err != nil {
			return err
		}
		e := Estado{CupoMensual: cupo, TopeDiario: diario}
		if err := tx.QueryRow(ctx, `SELECT coalesce(sum(respuestas), 0)::int, coalesce(sum(respuestas) FILTER (WHERE dia = $2), 0)::int
			FROM ia_uso WHERE dia >= $1`, mes, dia).Scan(&e.UsadasMes, &e.UsadasHoy); err != nil {
			return err
		}
		var gasto int64
		if m.Topes.PresupuestoMicro > 0 {
			r.estimado = EstimadoMicro
			if _, err := tx.Exec(ctx, "SELECT pg_advisory_xact_lock(hashtext('ia:plataforma'))"); err != nil {
				return err
			}
			if err := tx.QueryRow(ctx, "SELECT app.ia_gasto_desde($1)", mes).Scan(&gasto); err != nil {
				return err
			}
		}
		if err := m.agotado(e, gasto); err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `INSERT INTO ia_uso AS u (negocio_id, dia, ruta, respuestas, costo_micro)
			VALUES (app.negocio_actual(), $1, $2, 1, $3)
			ON CONFLICT (negocio_id, dia, ruta) DO UPDATE SET respuestas = u.respuestas + 1, costo_micro = u.costo_micro + EXCLUDED.costo_micro`,
			dia, ruta, r.estimado)
		return err
	})
	if err != nil {
		return nil, err
	}
	return r, nil
}

// Cerrar una reserva con lo que usó de verdad. Sin pedidos al modelo, la respuesta no cuenta (se
// devuelve la reserva).
func (m *Medidor) Cerrar(ctx context.Context, r *Reserva, u Uso) error {
	if r == nil {
		return nil
	}
	respuestas := 0
	if u.Pedidos == 0 {
		respuestas = -1
	}
	costo := u.CostoMicro() - r.estimado
	return m.Pool.EnNegocio(context.WithoutCancel(ctx), r.negocio, func(tx pgx.Tx) error {
		_, err := tx.Exec(context.WithoutCancel(ctx), `UPDATE ia_uso SET respuestas = greatest(respuestas + $3, 0), pedidos = pedidos + $4,
				tokens_entrada = tokens_entrada + $5, cache_leido = cache_leido + $6, cache_escrito = cache_escrito + $7,
				tokens_salida = tokens_salida + $8, costo_micro = greatest(costo_micro + $9, 0)
			WHERE negocio_id = app.negocio_actual() AND dia = $1 AND ruta = $2`,
			r.dia, r.ruta, respuestas, u.Pedidos, u.Entrada, u.CacheLeido, u.CacheEscrito, u.Salida, costo)
		return err
	})
}

// Chequear sin reservar (para mostrar el estado). Para llamar al modelo, Reservar.
func (m *Medidor) Chequear(ctx context.Context, negocio string) error {
	e, err := m.Leer(ctx, negocio)
	if err != nil {
		return err
	}
	var gasto int64
	if m.Topes.PresupuestoMicro > 0 {
		_, mes := m.hoy()
		if err := m.Pool.EnTx(ctx, func(tx pgx.Tx) error {
			return tx.QueryRow(ctx, "SELECT app.ia_gasto_desde($1)", mes).Scan(&gasto)
		}); err != nil {
			return err
		}
	}
	return m.agotado(e, gasto)
}

// Anotar lo que usó una respuesta (respuestas: 1 si llegó al modelo, aunque haya fallado a medias).
func (m *Medidor) Anotar(ctx context.Context, negocio, ruta string, u Uso, respuestas int) error {
	dia, _ := m.hoy()
	return m.Pool.EnNegocio(context.WithoutCancel(ctx), negocio, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `INSERT INTO ia_uso AS u (negocio_id, dia, ruta, respuestas, pedidos, tokens_entrada, cache_leido,
				cache_escrito, tokens_salida, costo_micro)
			VALUES (app.negocio_actual(), $1, $2, $3, $4, $5, $6, $7, $8, $9)
			ON CONFLICT (negocio_id, dia, ruta) DO UPDATE SET respuestas = u.respuestas + EXCLUDED.respuestas,
				pedidos = u.pedidos + EXCLUDED.pedidos, tokens_entrada = u.tokens_entrada + EXCLUDED.tokens_entrada,
				cache_leido = u.cache_leido + EXCLUDED.cache_leido, cache_escrito = u.cache_escrito + EXCLUDED.cache_escrito,
				tokens_salida = u.tokens_salida + EXCLUDED.tokens_salida, costo_micro = u.costo_micro + EXCLUDED.costo_micro`,
			dia, ruta, respuestas, u.Pedidos, u.Entrada, u.CacheLeido, u.CacheEscrito, u.Salida, u.CostoMicro())
		return err
	})
}
