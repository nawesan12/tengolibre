package directorio

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/problema"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/sesion"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/web"
)

// Pasado es un turno anterior del cliente (atendido o confirmado).
type Pasado struct {
	Inicio        time.Time
	ProfesionalID string
	Profesional   string
	ServicioID    string
}

// Sugerencia de horario: "suele reservar los martes a las 18:00 con Ana".
type Sugerencia struct {
	Dia           int     `json:"dia"` // 0 = domingo
	Hora          string  `json:"hora"`
	ProfesionalID string  `json:"profesionalId"`
	Profesional   string  `json:"profesional"`
	Confianza     float64 `json:"confianza"` // 0 a 1
	Motivo        string  `json:"motivo"`
	// Proxima: la próxima fecha (YYYY-MM-DD) con ese día y esa hora todavía por delante.
	Proxima string `json:"proxima"`
}

var dias = []string{"domingos", "lunes", "martes", "miércoles", "jueves", "viernes", "sábados"}

// Sugerir es smart-scheduling.ts de Jiku como función pura: agrupa los turnos por día de la semana,
// hora (redondeada a la grilla del negocio) y profesional, y propone los patrones que pesan al
// menos 15 %, los más fuertes primero (hasta 5). Con menos de dos turnos no hay patrón. Si se pide un
// servicio y hay al menos dos turnos de ese servicio, se miran sólo esos.
// A diferencia de Jiku, el redondeo arrastra la hora (10:50 con grilla de 30 es 11:00, no "10:60").
func Sugerir(pasados []Pasado, servicioID string, intervalo int, ahora time.Time, zona *time.Location) []Sugerencia {
	if intervalo <= 0 {
		intervalo = 30
	}
	if servicioID != "" {
		var del []Pasado
		for _, p := range pasados {
			if p.ServicioID == servicioID {
				del = append(del, p)
			}
		}
		if len(del) >= 2 {
			pasados = del
		}
	}
	out := []Sugerencia{}
	if len(pasados) < 2 {
		return out
	}
	type clave struct {
		dia, minutos int
		profesional  string
	}
	cuenta := map[clave]int{}
	nombres := map[string]string{}
	for _, p := range pasados {
		t := p.Inicio.In(zona)
		m := t.Hour()*60 + t.Minute()
		m = (m + intervalo/2) / intervalo * intervalo
		dia := int(t.Weekday())
		if m >= 24*60 { // redondeó a la medianoche: es el día siguiente
			m -= 24 * 60
			dia = (dia + 1) % 7
		}
		cuenta[clave{dia, m, p.ProfesionalID}]++
		nombres[p.ProfesionalID] = p.Profesional
	}
	for k, n := range cuenta {
		conf := float64(n) / float64(len(pasados))
		if conf < 0.15 {
			continue
		}
		hora := fmt.Sprintf("%02d:%02d", k.minutos/60, k.minutos%60)
		motivo := fmt.Sprintf("Suele reservar los %s a las %s", dias[k.dia], hora)
		if nombres[k.profesional] != "" {
			motivo += " con " + nombres[k.profesional]
		}
		out = append(out, Sugerencia{Dia: k.dia, Hora: hora, ProfesionalID: k.profesional, Profesional: nombres[k.profesional],
			Confianza: conf, Motivo: motivo, Proxima: proxima(ahora.In(zona), k.dia, k.minutos)})
	}
	sort.Slice(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if a.Confianza != b.Confianza {
			return a.Confianza > b.Confianza
		}
		if a.Proxima != b.Proxima {
			return a.Proxima < b.Proxima
		}
		if a.Hora != b.Hora {
			return a.Hora < b.Hora
		}
		return a.ProfesionalID < b.ProfesionalID
	})
	if len(out) > 5 {
		out = out[:5]
	}
	return out
}

// proxima fecha con ese día de la semana cuya hora todavía no pasó.
func proxima(ahora time.Time, dia, minutos int) string {
	d := ahora
	for i := 0; i < 8; i++ {
		inicio := time.Date(d.Year(), d.Month(), d.Day(), minutos/60, minutos%60, 0, 0, ahora.Location())
		if int(d.Weekday()) == dia && inicio.After(ahora) {
			return d.Format("2006-01-02")
		}
		d = d.AddDate(0, 0, 1)
	}
	return ""
}

// sugerenciasDe calcula las sugerencias de un cliente con el negocio fijado en tx.
func sugerenciasDe(ctx context.Context, tx pgx.Tx, clienteID, servicioID string, ahora time.Time) ([]Sugerencia, error) {
	var zona string
	var intervalo int
	if err := tx.QueryRow(ctx, `SELECT n.zona_horaria, a.intervalo FROM negocios n JOIN ajustes a ON a.negocio_id = n.id
		WHERE n.id = app.negocio_actual()`).Scan(&zona, &intervalo); err != nil {
		return nil, err
	}
	loc, err := time.LoadLocation(zona)
	if err != nil {
		return nil, err
	}
	filas, err := tx.Query(ctx, `SELECT inicio, coalesce(profesional_id::text, ''), profesional, coalesce(servicio_id::text, '')
		FROM turnos WHERE cliente_id = $1 AND estado IN ('atendido', 'confirmado') ORDER BY inicio DESC LIMIT 50`, clienteID)
	if err != nil {
		return nil, err
	}
	ps, err := pgx.CollectRows(filas, pgx.RowToStructByPos[Pasado])
	if err != nil {
		return nil, err
	}
	return Sugerir(ps, servicioID, intervalo, ahora, loc), nil
}

func (s *Servicio) sugerenciasCliente(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	codigo := r.PathValue("codigo")
	var negocio *string
	err := s.Pool.EnTx(ctx, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, "SELECT app.negocio_del_turno($1)::text", codigo).Scan(&negocio)
	})
	if err != nil {
		return err
	}
	if negocio == nil {
		return problema.NoEncontrado("No encontramos ese turno.")
	}
	var out []Sugerencia
	err = s.Pool.EnNegocio(ctx, *negocio, func(tx pgx.Tx) error {
		var cliente string
		if err := tx.QueryRow(ctx, "SELECT cliente_id::text FROM turnos WHERE codigo = $1", codigo).Scan(&cliente); err != nil {
			return err
		}
		var err error
		out, err = sugerenciasDe(ctx, tx, cliente, r.URL.Query().Get("servicio"), s.ahora())
		return err
	})
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusOK, map[string]any{"sugerencias": out})
	return nil
}

func (s *Servicio) sugerenciasPanel(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	var out []Sugerencia
	err := s.Pool.EnNegocio(ctx, sesion.De(ctx).NegocioID, func(tx pgx.Tx) error {
		var existe bool
		if err := tx.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM clientes WHERE id = $1)", r.PathValue("id")).Scan(&existe); err != nil {
			return err
		}
		if !existe {
			return pgx.ErrNoRows
		}
		var err error
		out, err = sugerenciasDe(ctx, tx, r.PathValue("id"), r.URL.Query().Get("servicio"), s.ahora())
		return err
	})
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusOK, map[string]any{"sugerencias": out})
	return nil
}
