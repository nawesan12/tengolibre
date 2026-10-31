package reportes

import (
	"context"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/clientes"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/web"
)

// Tablero de inicio del panel (getDashboardStats de Jiku).
type Tablero struct {
	TurnosHoy  int `json:"turnosHoy"`
	TurnosAyer int `json:"turnosAyer"`
	// Ingresos del mes en curso (hasta hoy) y del mes anterior entero.
	IngresosMes         int64 `json:"ingresosMes"`
	IngresosMesAnterior int64 `json:"ingresosMesAnterior"`
	// Clientes con algún turno no cancelado en los últimos 30 días, y en los 30 anteriores.
	ClientesActivos         int          `json:"clientesActivos"`
	ClientesActivosAnterior int          `json:"clientesActivosAnterior"`
	OcupacionHoy            *float64     `json:"ocupacionHoy"`
	Proximos                []Proximo    `json:"proximos"`
	Ultimos7Dias            []Dia        `json:"ultimos7Dias"`
	Ultimos6Meses           []IngresoMes `json:"ultimos6Meses"`
}

// Proximo turno (los cinco que siguen).
type Proximo struct {
	ID          string    `json:"id"`
	Inicio      time.Time `json:"inicio"`
	Estado      string    `json:"estado"`
	Servicio    string    `json:"servicio"`
	Profesional string    `json:"profesional"`
	ClienteID   string    `json:"clienteId"`
	Cliente     string    `json:"cliente"`
}

// IngresoMes: lo facturado en un mes (AAAA-MM).
type IngresoMes struct {
	Mes      string `json:"mes"`
	Ingresos int64  `json:"ingresos"`
}

func contar(ctx context.Context, tx pgx.Tx, ini, fin time.Time) (turnos int, ingresos int64, clientes int, err error) {
	err = tx.QueryRow(ctx, `SELECT count(*) FILTER (WHERE t.estado <> 'cancelado'),
		coalesce(sum(t.precio) FILTER (WHERE t.estado = 'atendido'), 0),
		count(DISTINCT t.cliente_id) FILTER (WHERE t.estado <> 'cancelado')
		FROM turnos t WHERE t.inicio >= $1 AND t.inicio < $2 AND `+vivo, ini, fin).Scan(&turnos, &ingresos, &clientes)
	return
}

func (s *Servicio) tablero(w http.ResponseWriter, r *http.Request) error {
	ahora := s.ahora()
	var tb Tablero
	err := s.enNegocio(r, func(ctx context.Context, tx pgx.Tx) error {
		if _, err := clientes.MarcarAusentes(ctx, tx, ahora); err != nil {
			return err
		}
		zona, err := zonaDe(ctx, tx)
		if err != nil {
			return err
		}
		h := ahora.In(zona)
		hoy := time.Date(h.Year(), h.Month(), h.Day(), 0, 0, 0, 0, zona)
		manana := hoy.AddDate(0, 0, 1)
		if tb.TurnosHoy, _, _, err = contar(ctx, tx, hoy, manana); err != nil {
			return err
		}
		if tb.TurnosAyer, _, _, err = contar(ctx, tx, hoy.AddDate(0, 0, -1), hoy); err != nil {
			return err
		}
		mes := time.Date(h.Year(), h.Month(), 1, 0, 0, 0, 0, zona)
		if _, tb.IngresosMes, _, err = contar(ctx, tx, mes, manana); err != nil {
			return err
		}
		if _, tb.IngresosMesAnterior, _, err = contar(ctx, tx, mes.AddDate(0, -1, 0), mes); err != nil {
			return err
		}
		if _, _, tb.ClientesActivos, err = contar(ctx, tx, hoy.AddDate(0, 0, -29), manana); err != nil {
			return err
		}
		if _, _, tb.ClientesActivosAnterior, err = contar(ctx, tx, hoy.AddDate(0, 0, -59), hoy.AddDate(0, 0, -29)); err != nil {
			return err
		}
		// La ocupación de hoy y la semana salen del mismo armado que el reporte.
		dia, err := armar(ctx, tx, nuevoRango(zona, hoy, hoy))
		if err != nil {
			return err
		}
		tb.OcupacionHoy = dia.Resumen.Ocupacion
		semana, err := armar(ctx, tx, nuevoRango(zona, hoy.AddDate(0, 0, -6), hoy))
		if err != nil {
			return err
		}
		tb.Ultimos7Dias = semana.PorDia
		for i := 5; i >= 0; i-- {
			ini := mes.AddDate(0, -i, 0)
			fin := ini.AddDate(0, 1, 0)
			_, ing, _, err := contar(ctx, tx, ini, fin)
			if err != nil {
				return err
			}
			tb.Ultimos6Meses = append(tb.Ultimos6Meses, IngresoMes{Mes: ini.Format("2006-01"), Ingresos: ing})
		}
		filas, err := tx.Query(ctx, `SELECT t.id::text, t.inicio, t.estado, t.servicio, t.profesional, c.id::text, c.nombre
			FROM turnos t JOIN clientes c ON c.id = t.cliente_id
			WHERE t.inicio >= $1 AND t.estado IN ('pendiente_pago', 'pendiente', 'confirmado') AND `+vivo+`
			ORDER BY t.inicio LIMIT 5`, ahora)
		if err != nil {
			return err
		}
		tb.Proximos, err = pgx.CollectRows(filas, pgx.RowToStructByPos[Proximo])
		return err
	})
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusOK, tb)
	return nil
}
