package turnos

// Volver a reservar en un toque (propuesta 04 del diseño v2): en Mi turno ya atendido, el link del
// cliente trae el próximo turno armado (mismo servicio, mismo profesional, a las N semanas, el horario
// más parecido) y tres fechas cercanas más. N es la frecuencia habitual del cliente (la mediana entre
// sus turnos atendidos) o 4. Se reserva con los datos que ya tiene el negocio y, si cobra seña, igual
// que la reserva pública: queda pendiente_pago con el link de Mercado Pago.

import (
	"context"
	"math"
	"net/http"
	"sort"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/catalogo"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/disponibilidad"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/problema"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/web"
)

// SemanasPorDefecto: a cuántas semanas se sugiere repetir sin historia del cliente.
const SemanasPorDefecto = 4

// Opcion de horario para volver a reservar.
type Opcion struct {
	Inicio time.Time `json:"inicio"`
	Fecha  string    `json:"fecha"`
	Hora   string    `json:"hora"`
	// MismoHorario: misma hora que el turno atendido.
	MismoHorario bool `json:"mismoHorario"`
}

// Repetir: lo que ve Mi turno atendido.
type Repetir struct {
	Semanas int `json:"semanas"`
	// Habitual: Semanas sale de la historia del cliente (si no, es la de por defecto).
	Habitual bool `json:"habitual"`
	Servicio struct {
		ID       string `json:"id"`
		Nombre   string `json:"nombre"`
		Precio   int64  `json:"precio"`
		Duracion int    `json:"duracion"`
	} `json:"servicio"`
	Profesional struct {
		ID     string `json:"id"`
		Nombre string `json:"nombre"`
	} `json:"profesional"`
	// Sugerencia: null si no hay nada libre cerca.
	Sugerencia   *Opcion  `json:"sugerencia"`
	Alternativas []Opcion `json:"alternativas"`
	// Pago: la seña (o el total) que se cobraría. null: el negocio no cobra por adelantado.
	Pago *Pago `json:"pago"`
}

// frecuencia del cliente en semanas: la mediana entre sus turnos atendidos (dos en la misma semana
// no cuentan como frecuencia).
func frecuencia(ctx context.Context, tx pgx.Tx, clienteID string) (int, bool, error) {
	var dias *float64
	err := tx.QueryRow(ctx, `SELECT percentile_cont(0.5) WITHIN GROUP (ORDER BY d) FROM (
			SELECT extract(epoch FROM inicio - lag(inicio) OVER (ORDER BY inicio)) / 86400 AS d
			FROM turnos WHERE cliente_id = $1 AND estado = 'atendido') x
		WHERE d >= 4`, clienteID).Scan(&dias)
	if err != nil || dias == nil {
		return SemanasPorDefecto, false, err
	}
	return min(max(int(math.Round(*dias/7)), 1), 12), true, nil
}

// paraRepetir: el turno del link, atendido y con su servicio y profesional todavía disponibles.
func paraRepetir(ctx context.Context, tx pgx.Tx, cod string) (Turno, catalogo.Prestacion, catalogo.Profesional, error) {
	t, err := leerTurno(ctx, tx, "t.codigo = $1", cod, false)
	if err != nil {
		return t, catalogo.Prestacion{}, catalogo.Profesional{}, err
	}
	if t.Estado != "atendido" {
		return t, catalogo.Prestacion{}, catalogo.Profesional{}, problema.Conflicto("no_atendido", "Se vuelve a reservar desde un turno ya atendido.")
	}
	if t.ServicioID == nil || t.ProfesionalID == nil {
		return t, catalogo.Prestacion{}, catalogo.Profesional{}, problema.Conflicto("no_disponible",
			"Ese servicio o profesional ya no está: sacá un turno nuevo desde la página.")
	}
	srv, p, err := servicioYProfesional(ctx, tx, *t.ServicioID, *t.ProfesionalID)
	if err != nil {
		return t, srv, p, problema.Conflicto("no_disponible", "Ese servicio o profesional ya no está: sacá un turno nuevo desde la página.")
	}
	return t, srv, p, nil
}

// repetir arma la sugerencia en el negocio fijado.
func (s *Servicio) repetir(ctx context.Context, tx pgx.Tx, cod string) (Repetir, error) {
	var out Repetir
	t, srv, p, err := paraRepetir(ctx, tx, cod)
	if err != nil {
		return out, err
	}
	c, err := cargarContexto(ctx, tx)
	if err != nil {
		return out, err
	}
	ahora := s.ahora()
	if out.Semanas, out.Habitual, err = frecuencia(ctx, tx, t.Cliente.ID); err != nil {
		return out, err
	}
	out.Servicio.ID, out.Servicio.Nombre, out.Servicio.Precio, out.Servicio.Duracion = srv.ID, srv.Nombre, srv.Precio, srv.Duracion
	out.Profesional.ID, out.Profesional.Nombre = p.ID, p.Nombre
	out.Alternativas = []Opcion{}

	// El objetivo: a las N semanas, a la misma hora. Si ya pasó (el link se abrió tarde), la semana
	// siguiente que todavía no pasó.
	objetivo := t.Inicio.In(c.zona).AddDate(0, 0, 7*out.Semanas)
	a := ahora.In(c.zona)
	hoy := time.Date(a.Year(), a.Month(), a.Day(), 0, 0, 0, 0, c.zona)
	for objetivo.Before(hoy) {
		objetivo = objetivo.AddDate(0, 0, 7)
	}
	// Días a mirar, del más parecido al menos: el mismo día de la semana una y dos semanas antes y
	// después, y después los días de alrededor.
	desplazamientos := []int{0, -7, 7, -14, 14}
	for d := 1; d <= 6; d++ {
		desplazamientos = append(desplazamientos, -d, d)
	}
	var opciones []Opcion
	for _, d := range desplazamientos {
		dia := objetivo.AddDate(0, 0, d)
		if dia.Before(hoy) {
			continue
		}
		fecha := dia.Format(time.DateOnly)
		datos, err := datosDelDia(ctx, tx, p, fecha, c.zona, "")
		if err != nil {
			return out, err
		}
		m := masCercano(disponibilidad.Dia(disponibilidad.Consulta{Profesional: datos, Fecha: fecha, Duracion: srv.Duracion,
			Reglas: c.reglas(), Ahora: ahora, Zona: c.zona}), dia)
		if m == nil {
			continue
		}
		opciones = append(opciones, Opcion{Inicio: m.Inicio, Fecha: fecha, Hora: m.Hora, MismoHorario: m.Hora == t.Inicio.In(c.zona).Format("15:04")})
		if len(opciones) == 4 {
			break
		}
	}
	if len(opciones) > 0 {
		out.Sugerencia = &opciones[0]
		out.Alternativas = append(out.Alternativas, opciones[1:]...)
		sort.Slice(out.Alternativas, func(i, j int) bool { return out.Alternativas[i].Inicio.Before(out.Alternativas[j].Inicio) })
	}
	out.Pago, err = s.estimar(ctx, tx, srv.Precio)
	return out, err
}

func (s *Servicio) verRepetir(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	cod := r.PathValue("codigo")
	negocio, err := s.negocioDelTurno(ctx, cod)
	if err != nil {
		return err
	}
	var out Repetir
	err = s.Pool.EnNegocio(ctx, negocio, func(tx pgx.Tx) (err error) {
		out, err = s.repetir(ctx, tx, cod)
		return err
	})
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusOK, out)
	return nil
}

// reservarRepetir: el turno nuevo, a nombre del mismo cliente, con el mismo servicio y profesional.
// Pasa por las mismas reglas que la reserva pública (penalidades, disponibilidad, seña).
func (s *Servicio) reservarRepetir(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	var in struct {
		Inicio time.Time `json:"inicio"`
	}
	if err := web.Leer(w, r, &in); err != nil {
		return err
	}
	if in.Inicio.IsZero() {
		return problema.Invalido("Falta el horario.", problema.Campo{Ruta: "inicio", Mensaje: "requerido"})
	}
	cod := r.PathValue("codigo")
	negocio, err := s.negocioDelTurno(ctx, cod)
	if err != nil {
		return err
	}
	var t Turno
	err = s.Pool.EnNegocio(ctx, negocio, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, "SELECT set_config('app.origen', 'web', true)"); err != nil {
			return err
		}
		viejo, srv, p, err := paraRepetir(ctx, tx, cod)
		if err != nil {
			return err
		}
		pedido := Pedido{ServicioID: srv.ID, ProfesionalID: p.ID, Inicio: in.Inicio, ClienteID: viejo.Cliente.ID}
		if t, err = reservar(ctx, tx, pedido, "web", s.ahora()); err != nil {
			return err
		}
		return s.alReservar(ctx, tx, &t, pedido, "web")
	})
	if err != nil {
		return err
	}
	s.despuesDeReservar(ctx, negocio, &t)
	s.avisar(ctx, negocio, t.ID, "reservado")
	t.Cliente.ID = ""
	web.JSON(w, http.StatusCreated, t)
	return nil
}
