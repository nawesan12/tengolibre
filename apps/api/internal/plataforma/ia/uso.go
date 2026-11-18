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
	// TopeDiario: respuestas con IA por negocio y por día, para frenar abusos.
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
	e := Estado{CupoMensual: m.Topes.CupoMensual, TopeDiario: m.Topes.TopeDiario}
	dia, mes := m.hoy()
	err := m.Pool.EnNegocio(ctx, negocio, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `SELECT coalesce(sum(respuestas), 0)::int, coalesce(sum(respuestas) FILTER (WHERE dia = $2), 0)::int
			FROM ia_uso WHERE dia >= $1`, mes, dia).Scan(&e.UsadasMes, &e.UsadasHoy)
	})
	return e, err
}

// Chequear antes de llamar al modelo. Devuelve *Agotado si no se puede.
func (m *Medidor) Chequear(ctx context.Context, negocio string) error {
	e, err := m.Leer(ctx, negocio)
	if err != nil {
		return err
	}
	if m.Topes.CupoMensual > 0 && e.UsadasMes >= m.Topes.CupoMensual {
		return &Agotado{Motivo: "mes", Mensaje: fmt.Sprintf("Ya usaste las %d respuestas con IA de este mes. Sigo contestando lo "+
			"básico (agenda, horarios libres, mover y cancelar turnos); el mes que viene vuelve todo.", m.Topes.CupoMensual)}
	}
	if m.Topes.TopeDiario > 0 && e.UsadasHoy >= m.Topes.TopeDiario {
		return &Agotado{Motivo: "dia", Mensaje: "Por hoy llegaste al tope de respuestas con IA. Sigo contestando lo básico " +
			"(agenda, horarios libres, mover y cancelar turnos); mañana vuelve todo."}
	}
	if m.Topes.PresupuestoMicro > 0 {
		_, mes := m.hoy()
		var gasto int64
		if err := m.Pool.EnTx(ctx, func(tx pgx.Tx) error {
			return tx.QueryRow(ctx, "SELECT app.ia_gasto_desde($1)", mes).Scan(&gasto)
		}); err != nil {
			return err
		}
		if gasto >= m.Topes.PresupuestoMicro {
			return &Agotado{Motivo: "plataforma", Mensaje: "La IA está en pausa por el resto del mes. Sigo contestando lo " +
				"básico (agenda, horarios libres, mover y cancelar turnos)."}
		}
	}
	return nil
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
