// Package reportes: los números del negocio para el panel (panel/reports y panel/stats de Jiku):
// ingresos, turnos por estado, por profesional y por servicio, ocupación, clientes nuevos y
// recurrentes, ausencias; el tablero de inicio, y los CSV de clientes y de turnos. Ingresos es lo
// facturado: el precio de los turnos atendidos. Cobrado es la plata que entró de verdad: las señas
// online aprobadas y lo cobrado en la caja del local, por el día en que entró.
package reportes

import (
	"context"
	"maps"
	"math"
	"net/http"
	"slices"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/caja"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/clientes"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/disponibilidad"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/negocios"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/db"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/problema"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/sesion"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/web"
)

// Servicio de reportes. Ahora es el reloj (tests); nil: time.Now.
type Servicio struct {
	Pool   *db.Pool
	Sesion *sesion.Servicio
	Ahora  func() time.Time
}

func (s *Servicio) ahora() time.Time {
	if s.Ahora != nil {
		return s.Ahora()
	}
	return time.Now()
}

// Rutas del dominio. Todo es de gestión: son los números del negocio y los datos de todos los clientes.
func (s *Servicio) Rutas(mux *http.ServeMux) {
	g := func(h web.Manejador) http.HandlerFunc { return s.Sesion.Panel(sesion.Gestion, web.Envolver(h)) }
	mux.HandleFunc("GET /v1/panel/reportes", g(s.reporte))
	mux.HandleFunc("GET /v1/panel/estadisticas", g(s.tablero))
	mux.HandleFunc("GET /v1/panel/reportes/clientes.csv", g(s.csvClientes))
	mux.HandleFunc("GET /v1/panel/reportes/turnos.csv", g(s.csvTurnos))
}

// MaxDias de un reporte.
const MaxDias = 366

// vivo: los turnos que cuentan. Un pendiente_pago vencido es un horario que nunca se tomó.
const vivo = "NOT (t.estado = 'pendiente_pago' AND t.vence_en <= now())"

// rango de fechas del negocio fijado: desde y hasta inclusive, en su zona.
type rango struct {
	zona         *time.Location
	desde, hasta string    // AAAA-MM-DD
	ini, fin     time.Time // [ini, fin)
}

func zonaDe(ctx context.Context, tx pgx.Tx) (*time.Location, error) {
	n, err := negocios.Leer(ctx, tx)
	if err != nil {
		return nil, err
	}
	z, err := time.LoadLocation(n.ZonaHoraria)
	if err != nil {
		z = disponibilidad.ZonaPorDefecto
	}
	return z, nil
}

func nuevoRango(zona *time.Location, desde, hasta time.Time) rango {
	return rango{zona: zona, desde: desde.Format(time.DateOnly), hasta: hasta.Format(time.DateOnly),
		ini: desde, fin: hasta.AddDate(0, 0, 1)}
}

// leerRango: ?desde=&hasta= (AAAA-MM-DD). Por defecto, los últimos 30 días hasta hoy.
func leerRango(r *http.Request, zona *time.Location, ahora time.Time) (rango, error) {
	q := r.URL.Query()
	hoy := ahora.In(zona)
	hasta := time.Date(hoy.Year(), hoy.Month(), hoy.Day(), 0, 0, 0, 0, zona)
	var err error
	if v := q.Get("hasta"); v != "" {
		if hasta, err = time.ParseInLocation(time.DateOnly, v, zona); err != nil {
			return rango{}, problema.Invalido("hasta va como AAAA-MM-DD.")
		}
	}
	desde := hasta.AddDate(0, 0, -29)
	if v := q.Get("desde"); v != "" {
		if desde, err = time.ParseInLocation(time.DateOnly, v, zona); err != nil {
			return rango{}, problema.Invalido("desde va como AAAA-MM-DD.")
		}
	}
	if hasta.Before(desde) || hasta.Sub(desde) > (MaxDias-1)*24*time.Hour+time.Hour {
		return rango{}, problema.Invalido("desde y hasta van en orden y con hasta 366 días.")
	}
	return nuevoRango(zona, desde, hasta), nil
}

// Fila de un reporte por profesional o por servicio.
type Fila struct {
	ID        *string `json:"id"`
	Nombre    string  `json:"nombre"`
	Turnos    int     `json:"turnos"`
	Atendidos int     `json:"atendidos"`
	Ausentes  int     `json:"ausentes"`
	Ingresos  int64   `json:"ingresos"`
	// Minutos reservados (turnos no cancelados) y, por profesional, los que tenía en su horario.
	MinutosReservados  int      `json:"minutosReservados"`
	MinutosDisponibles *int     `json:"minutosDisponibles,omitempty"`
	Ocupacion          *float64 `json:"ocupacion,omitempty"`
}

// Dia del reporte.
type Dia struct {
	Fecha     string `json:"fecha"`
	Turnos    int    `json:"turnos"`
	Atendidos int    `json:"atendidos"`
	Ingresos  int64  `json:"ingresos"`
	// Cobrado: señas y caja que entraron ese día.
	Cobrado int64 `json:"cobrado" db:"-"`
}

// Resumen del período.
type Resumen struct {
	// Turnos: los no cancelados.
	Turnos         int   `json:"turnos"`
	Atendidos      int   `json:"atendidos"`
	Ingresos       int64 `json:"ingresos"`
	TicketPromedio int64 `json:"ticketPromedio"`
	// Clientes distintos con algún turno no cancelado.
	Clientes int `json:"clientes"`
	// TasaAusencias: ausentes sobre (atendidos + ausentes), en porcentaje. Null sin turnos cerrados.
	TasaAusencias *float64 `json:"tasaAusencias"`
	// Ocupacion: minutos reservados sobre minutos de horario, en porcentaje. Null sin horarios.
	Ocupacion *float64 `json:"ocupacion"`
	// La plata real del período (por el día en que entró): Cobrado = CobradoSenas + CobradoCaja.
	Cobrado      int64 `json:"cobrado"`
	CobradoSenas int64 `json:"cobradoSenas"`
	CobradoCaja  int64 `json:"cobradoCaja"`
	// CobradoPorMedio: la caja por medio de pago (efectivo, transferencia, debito, credito, mercadopago).
	CobradoPorMedio map[string]int64 `json:"cobradoPorMedio"`
	// Productos: cuánto de la caja fue de productos.
	Productos int64 `json:"productos"`
	// Debe: lo que quedó debiendo («Después») de los turnos del período.
	Debe int64 `json:"debe"`
}

// Reporte de un período.
type Reporte struct {
	Desde          string         `json:"desde"`
	Hasta          string         `json:"hasta"`
	Resumen        Resumen        `json:"resumen"`
	PorEstado      map[string]int `json:"porEstado"`
	PorProfesional []Fila         `json:"porProfesional"`
	PorServicio    []Fila         `json:"porServicio"`
	PorDia         []Dia          `json:"porDia"`
	// Clientes nuevos: su primer turno (no cancelado) cae en el período; recurrentes: ya venían.
	ClientesNuevos      int `json:"clientesNuevos"`
	ClientesRecurrentes int `json:"clientesRecurrentes"`
}

func porcentaje(a, b int64) *float64 {
	if b <= 0 {
		return nil
	}
	v := math.Round(float64(a)*1000/float64(b)) / 10
	return &v
}

// agrupar turnos del rango por una columna (profesional o servicio). Si el profesional o el servicio
// se borró, el turno conserva el nombre copiado y se agrupa por ese nombre.
func agrupar(ctx context.Context, tx pgx.Tx, rg rango, id, copia, tabla string) ([]Fila, error) {
	filas, err := tx.Query(ctx, `SELECT t.`+id+`::text, coalesce(max(x.nombre), max(t.`+copia+`)),
		count(*) FILTER (WHERE t.estado <> 'cancelado'),
		count(*) FILTER (WHERE t.estado = 'atendido'),
		count(*) FILTER (WHERE t.estado = 'ausente'),
		coalesce(sum(t.precio) FILTER (WHERE t.estado = 'atendido'), 0),
		coalesce(sum(extract(epoch FROM t.fin - t.inicio) / 60) FILTER (WHERE t.estado <> 'cancelado'), 0)::int
		FROM turnos t LEFT JOIN `+tabla+` x ON x.id = t.`+id+`
		WHERE t.inicio >= $1 AND t.inicio < $2 AND `+vivo+`
		GROUP BY t.`+id+`, CASE WHEN t.`+id+` IS NULL THEN t.`+copia+` END
		ORDER BY 6 DESC, 3 DESC, 2`, rg.ini, rg.fin)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(filas, func(f pgx.CollectableRow) (Fila, error) {
		var x Fila
		err := f.Scan(&x.ID, &x.Nombre, &x.Turnos, &x.Atendidos, &x.Ausentes, &x.Ingresos, &x.MinutosReservados)
		return x, err
	})
}

// disponibles: los minutos de horario de cada profesional activo en el rango, menos los días
// bloqueados enteros y lo que tapan los bloqueos parciales. Es una aproximación: no descuenta los
// bloqueos semanales ni las excepciones (que son de a un horario).
func disponibles(ctx context.Context, tx pgx.Tx, rg rango) (map[string]int, map[string]string, error) {
	filas, err := tx.Query(ctx, `WITH dias AS (
			SELECT d::date AS fecha FROM generate_series($1::date, $2::date, interval '1 day') d
		), franjas AS (
			SELECT h.profesional_id, d.fecha, h.desde, h.hasta FROM dias d
			JOIN horarios h ON h.dia = extract(dow FROM d.fecha)::int
			JOIN profesionales p ON p.id = h.profesional_id AND p.activo
			WHERE NOT EXISTS (SELECT 1 FROM bloqueos b WHERE b.profesional_id = h.profesional_id AND b.fecha = d.fecha AND b.desde IS NULL)
		)
		SELECT f.profesional_id::text, max(p.nombre), sum(greatest(0, extract(epoch FROM f.hasta - f.desde) / 60
			- coalesce((SELECT sum(extract(epoch FROM least(f.hasta, b.hasta) - greatest(f.desde, b.desde)) / 60) FROM bloqueos b
				WHERE b.profesional_id = f.profesional_id AND b.fecha = f.fecha AND b.desde < f.hasta AND b.hasta > f.desde), 0)))::int
		FROM franjas f JOIN profesionales p ON p.id = f.profesional_id
		GROUP BY f.profesional_id`, rg.desde, rg.hasta)
	if err != nil {
		return nil, nil, err
	}
	defer filas.Close()
	minutos, nombres := map[string]int{}, map[string]string{}
	for filas.Next() {
		var id, nombre string
		var m int
		if err := filas.Scan(&id, &nombre, &m); err != nil {
			return nil, nil, err
		}
		minutos[id], nombres[id] = m, nombre
	}
	return minutos, nombres, filas.Err()
}

// armar el reporte del rango en el negocio fijado.
func armar(ctx context.Context, tx pgx.Tx, rg rango) (Reporte, error) {
	rep := Reporte{Desde: rg.desde, Hasta: rg.hasta, PorEstado: map[string]int{}, PorDia: []Dia{}}
	filas, err := tx.Query(ctx, "SELECT t.estado, count(*) FROM turnos t WHERE t.inicio >= $1 AND t.inicio < $2 AND "+vivo+" GROUP BY 1",
		rg.ini, rg.fin)
	if err != nil {
		return rep, err
	}
	for filas.Next() {
		var e string
		var n int
		if err := filas.Scan(&e, &n); err != nil {
			filas.Close()
			return rep, err
		}
		rep.PorEstado[e] = n
	}
	filas.Close()
	if err := filas.Err(); err != nil {
		return rep, err
	}

	if rep.PorServicio, err = agrupar(ctx, tx, rg, "servicio_id", "servicio", "servicios"); err != nil {
		return rep, err
	}
	if rep.PorProfesional, err = agrupar(ctx, tx, rg, "profesional_id", "profesional", "profesionales"); err != nil {
		return rep, err
	}
	minutos, nombres, err := disponibles(ctx, tx, rg)
	if err != nil {
		return rep, err
	}
	var reservados, totales int64
	vistos := map[string]bool{}
	for i := range rep.PorProfesional {
		f := &rep.PorProfesional[i]
		if f.ID == nil {
			continue
		}
		if m, ok := minutos[*f.ID]; ok {
			vistos[*f.ID] = true
			f.MinutosDisponibles, f.Ocupacion = &m, porcentaje(int64(f.MinutosReservados), int64(m))
			reservados, totales = reservados+int64(f.MinutosReservados), totales+int64(m)
		}
	}
	// Los que tenían horario y no tuvieron turnos también cuentan (en cero).
	for _, id := range slices.Sorted(maps.Keys(minutos)) {
		if m := minutos[id]; !vistos[id] {
			rep.PorProfesional = append(rep.PorProfesional, Fila{ID: &id, Nombre: nombres[id], MinutosDisponibles: &m,
				Ocupacion: porcentaje(0, int64(m))})
			totales += int64(m)
		}
	}

	filas, err = tx.Query(ctx, `SELECT to_char(t.inicio AT TIME ZONE $3, 'YYYY-MM-DD'),
		count(*) FILTER (WHERE t.estado <> 'cancelado'), count(*) FILTER (WHERE t.estado = 'atendido'),
		coalesce(sum(t.precio) FILTER (WHERE t.estado = 'atendido'), 0)
		FROM turnos t WHERE t.inicio >= $1 AND t.inicio < $2 AND `+vivo+` GROUP BY 1`, rg.ini, rg.fin, rg.zona.String())
	if err != nil {
		return rep, err
	}
	dias, err := pgx.CollectRows(filas, pgx.RowToStructByPos[Dia])
	if err != nil {
		return rep, err
	}
	porFecha := map[string]Dia{}
	for _, d := range dias {
		porFecha[d.Fecha] = d
	}
	for d := rg.ini; d.Before(rg.fin); d = d.AddDate(0, 0, 1) {
		f := d.Format(time.DateOnly)
		x, ok := porFecha[f]
		if !ok {
			x = Dia{Fecha: f}
		}
		rep.PorDia = append(rep.PorDia, x)
		rep.Resumen.Turnos += x.Turnos
		rep.Resumen.Atendidos += x.Atendidos
		rep.Resumen.Ingresos += x.Ingresos
	}

	if err := tx.QueryRow(ctx, `WITH en_rango AS (
			SELECT DISTINCT t.cliente_id FROM turnos t
			WHERE t.inicio >= $1 AND t.inicio < $2 AND t.estado <> 'cancelado' AND `+vivo+`
		), primera AS (
			SELECT t.cliente_id, min(t.inicio) AS primera FROM turnos t
			WHERE t.cliente_id IN (SELECT cliente_id FROM en_rango) AND t.estado <> 'cancelado' AND `+vivo+`
			GROUP BY t.cliente_id
		)
		SELECT count(*) FILTER (WHERE primera >= $1), count(*) FILTER (WHERE primera < $1) FROM primera`,
		rg.ini, rg.fin).Scan(&rep.ClientesNuevos, &rep.ClientesRecurrentes); err != nil {
		return rep, err
	}

	if err := cobrado(ctx, tx, rg, &rep); err != nil {
		return rep, err
	}

	rep.Resumen.Clientes = rep.ClientesNuevos + rep.ClientesRecurrentes
	if rep.Resumen.Atendidos > 0 {
		rep.Resumen.TicketPromedio = rep.Resumen.Ingresos / int64(rep.Resumen.Atendidos)
	}
	rep.Resumen.TasaAusencias = porcentaje(int64(rep.PorEstado["ausente"]), int64(rep.Resumen.Atendidos+rep.PorEstado["ausente"]))
	rep.Resumen.Ocupacion = porcentaje(reservados, totales)
	return rep, nil
}

// cobrado suma al reporte la plata que entró: señas online (caja.SenasEntre) y la caja del local.
func cobrado(ctx context.Context, tx pgx.Tx, rg rango, rep *Reporte) error {
	r := &rep.Resumen
	r.CobradoPorMedio = map[string]int64{}
	for _, m := range caja.Medios {
		if m != "despues" {
			r.CobradoPorMedio[m] = 0
		}
	}
	if err := tx.QueryRow(ctx, caja.SenasEntre, rg.ini, rg.fin).Scan(&r.CobradoSenas); err != nil {
		return err
	}
	filas, err := tx.Query(ctx, `SELECT medio, sum(total)::bigint, sum(productos)::bigint FROM caja_cobros
		WHERE estado = 'cobrado' AND cobrado_en >= $1 AND cobrado_en < $2 GROUP BY 1`, rg.ini, rg.fin)
	if err != nil {
		return err
	}
	for filas.Next() {
		var m string
		var total, productos int64
		if err := filas.Scan(&m, &total, &productos); err != nil {
			filas.Close()
			return err
		}
		r.CobradoPorMedio[m] = total
		r.CobradoCaja += total
		r.Productos += productos
	}
	filas.Close()
	if err := filas.Err(); err != nil {
		return err
	}
	r.Cobrado = r.CobradoSenas + r.CobradoCaja
	if err := tx.QueryRow(ctx, `SELECT coalesce(sum(k.total), 0)::bigint FROM caja_cobros k JOIN turnos t ON t.id = k.turno_id
		WHERE k.estado = 'debe' AND t.inicio >= $1 AND t.inicio < $2`, rg.ini, rg.fin).Scan(&r.Debe); err != nil {
		return err
	}
	// Por día, en la zona del negocio.
	filas, err = tx.Query(ctx, `SELECT dia, sum(monto)::bigint FROM (
			SELECT to_char(creado_en AT TIME ZONE $3, 'YYYY-MM-DD') AS dia, monto - devuelto AS monto FROM pagos
			WHERE estado = 'aprobado' AND creado_en >= $1 AND creado_en < $2
			UNION ALL
			SELECT to_char(cobrado_en AT TIME ZONE $3, 'YYYY-MM-DD'), total FROM caja_cobros
			WHERE estado = 'cobrado' AND cobrado_en >= $1 AND cobrado_en < $2) x GROUP BY 1`, rg.ini, rg.fin, rg.zona.String())
	if err != nil {
		return err
	}
	defer filas.Close()
	porFecha := map[string]int64{}
	for filas.Next() {
		var d string
		var m int64
		if err := filas.Scan(&d, &m); err != nil {
			return err
		}
		porFecha[d] = m
	}
	for i := range rep.PorDia {
		rep.PorDia[i].Cobrado = porFecha[rep.PorDia[i].Fecha]
	}
	return filas.Err()
}

func (s *Servicio) enNegocio(r *http.Request, fn func(context.Context, pgx.Tx) error) error {
	ctx := r.Context()
	return s.Pool.EnNegocio(ctx, sesion.De(ctx).NegocioID, func(tx pgx.Tx) error { return fn(ctx, tx) })
}

func (s *Servicio) reporte(w http.ResponseWriter, r *http.Request) error {
	var rep Reporte
	ahora := s.ahora()
	err := s.enNegocio(r, func(ctx context.Context, tx pgx.Tx) error {
		if _, err := clientes.MarcarAusentes(ctx, tx, ahora); err != nil {
			return err
		}
		zona, err := zonaDe(ctx, tx)
		if err != nil {
			return err
		}
		rg, err := leerRango(r, zona, ahora)
		if err != nil {
			return err
		}
		rep, err = armar(ctx, tx, rg)
		return err
	})
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusOK, rep)
	return nil
}
