package reportes

import (
	"context"
	"math"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/clientes"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/promos"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/auditoria"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/problema"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/web"
)

// Horas muertas (propuesta 11 del diseño v2): la semana como mapa de calor, día × hora, con qué parte
// de la capacidad quedó sin usar en las últimas 8 semanas. Capacidad: los minutos de horario de cada
// profesional activo en esa hora (menos los bloqueos); ocupado: lo que tapan los turnos no cancelados
// dentro de esa capacidad. «Vacío el 72% de las veces» es (capacidad − ocupado) / capacidad.

// SemanasHorasMuertas que mira el mapa.
const SemanasHorasMuertas = 8

// AntiguedadSugerida del cupón: clientes que vienen hace más de 2 meses.
const AntiguedadSugerida = 60

// Celda del mapa: un día de la semana (0 = domingo) y una hora (14 = de 14 a 15).
type Celda struct {
	Dia       int      `json:"dia"`
	Hora      int      `json:"hora"`
	Capacidad int      `json:"capacidadMinutos"`
	Ocupado   int      `json:"ocupadoMinutos"`
	Ocupacion *float64 `json:"ocupacion"`
	Vacio     *float64 `json:"vacio"`
	// Estado: en Horas muertas, suele_quedar_vacio (vacío ≥ 50%), ocupado o cerrado (sin horario);
	// en el reporte mensual, lleno (≥ 85% ocupado), a_medias (≥ 50%), libre o cerrado.
	Estado string `json:"estado"`
}

// Mapa de calor de un rango. Dias va de lunes a domingo (sólo los que tienen horario) y Horas de la
// primera a la última con horario; Celdas trae una por día × hora, día por día.
type Mapa struct {
	Dias      []int   `json:"dias"`
	Horas     []int   `json:"horas"`
	Celdas    []Celda `json:"celdas"`
	Capacidad int     `json:"capacidadMinutos"`
	Ocupado   int     `json:"ocupadoMinutos"`
}

func (m Mapa) celda(dia, hora int) *Celda {
	for i := range m.Celdas {
		if m.Celdas[i].Dia == dia && m.Celdas[i].Hora == hora {
			return &m.Celdas[i]
		}
	}
	return nil
}

// Franja peor (la que más queda vacía).
type Franja struct {
	Dia       int     `json:"dia"`
	HoraDesde int     `json:"horaDesde"`
	HoraHasta int     `json:"horaHasta"`
	Vacio     float64 `json:"vacio"`
	// Minutos de capacidad que quedaron libres en esa franja, en todo el rango.
	LibreMinutos int `json:"libreMinutos"`
	// Texto: «Martes de 14 a 16: vacío el 72% de las veces».
	Texto string `json:"texto"`
}

// Sugerencia de cupón para la franja.
type Sugerencia struct {
	Porcentaje     int    `json:"porcentaje"`
	Dias           []int  `json:"dias"`
	HoraDesde      string `json:"horaDesde"`
	HoraHasta      string `json:"horaHasta"`
	AntiguedadDias int    `json:"antiguedadDias"`
	// Destinatarios: clientes con email que vienen hace más de AntiguedadDias (y no se dieron de baja).
	Destinatarios int    `json:"destinatarios"`
	Codigo        string `json:"codigo"`
	Texto         string `json:"texto"`
}

// HorasMuertas: lo que devuelve GET /v1/panel/reportes/horas-muertas.
type HorasMuertas struct {
	Desde      string      `json:"desde"`
	Hasta      string      `json:"hasta"`
	Semanas    int         `json:"semanas"`
	Mapa       Mapa        `json:"mapa"`
	Peor       *Franja     `json:"peor"`
	Sugerencia *Sugerencia `json:"sugerencia"`
}

// Nombres de los días en singular (0 = domingo).
var nombresDia = [...]string{"Domingo", "Lunes", "Martes", "Miércoles", "Jueves", "Viernes", "Sábado"}

// aMinutos: "14:30" → 870.
func aMinutos(h string) int {
	t, err := time.Parse("15:04", h)
	if err != nil {
		return 0
	}
	return t.Hour()*60 + t.Minute()
}

// calcularMapa arma el mapa de [ini, fin) (medianoches locales) en el negocio fijado. clasificar
// pone el estado de cada celda con horario.
func calcularMapa(ctx context.Context, tx pgx.Tx, zona *time.Location, ini, fin time.Time, clasificar func(c Celda) string) (Mapa, error) {
	type franja struct{ desde, hasta int }
	horarios := map[string]map[int][]franja{} // profesional → día → franjas
	filas, err := tx.Query(ctx, `SELECT h.profesional_id::text, h.dia, to_char(h.desde, 'HH24:MI'), to_char(h.hasta, 'HH24:MI')
		FROM horarios h JOIN profesionales p ON p.id = h.profesional_id AND p.activo`)
	if err != nil {
		return Mapa{}, err
	}
	for filas.Next() {
		var p, d, h string
		var dia int
		if err := filas.Scan(&p, &dia, &d, &h); err != nil {
			filas.Close()
			return Mapa{}, err
		}
		if horarios[p] == nil {
			horarios[p] = map[int][]franja{}
		}
		hasta := aMinutos(h)
		if h == "23:59" {
			hasta = 24 * 60
		}
		horarios[p][dia] = append(horarios[p][dia], franja{aMinutos(d), hasta})
	}
	filas.Close()
	if err := filas.Err(); err != nil {
		return Mapa{}, err
	}

	type bloqueo struct {
		entero       bool
		desde, hasta int
	}
	bloqueos := map[string][]bloqueo{} // profesional + fecha
	filas, err = tx.Query(ctx, `SELECT profesional_id::text, fecha::text, coalesce(to_char(desde, 'HH24:MI'), ''), coalesce(to_char(hasta, 'HH24:MI'), '')
		FROM bloqueos WHERE fecha >= $1::date AND fecha < $2::date`, ini.Format(time.DateOnly), fin.Format(time.DateOnly))
	if err != nil {
		return Mapa{}, err
	}
	for filas.Next() {
		var p, f, d, h string
		if err := filas.Scan(&p, &f, &d, &h); err != nil {
			filas.Close()
			return Mapa{}, err
		}
		bloqueos[p+f] = append(bloqueos[p+f], bloqueo{entero: d == "", desde: aMinutos(d), hasta: aMinutos(h)})
	}
	filas.Close()
	if err := filas.Err(); err != nil {
		return Mapa{}, err
	}

	type turno struct{ inicio, fin time.Time }
	turnos := map[string][]turno{} // profesional + fecha
	filas, err = tx.Query(ctx, `SELECT t.profesional_id::text, t.inicio, t.fin FROM turnos t
		WHERE t.inicio >= $1 AND t.inicio < $2 AND t.estado <> 'cancelado' AND t.profesional_id IS NOT NULL AND `+vivo, ini, fin)
	if err != nil {
		return Mapa{}, err
	}
	for filas.Next() {
		var p string
		var t turno
		if err := filas.Scan(&p, &t.inicio, &t.fin); err != nil {
			filas.Close()
			return Mapa{}, err
		}
		t.inicio, t.fin = t.inicio.In(zona), t.fin.In(zona)
		k := p + t.inicio.Format(time.DateOnly)
		turnos[k] = append(turnos[k], t)
	}
	filas.Close()
	if err := filas.Err(); err != nil {
		return Mapa{}, err
	}

	var capacidad, ocupado [7][24]int
	var cap, ocu [24 * 60]bool
	for d := ini; d.Before(fin); d = d.AddDate(0, 0, 1) {
		fecha, dia := d.Format(time.DateOnly), int(d.Weekday())
		for p, porDia := range horarios {
			fs := porDia[dia]
			if len(fs) == 0 {
				continue
			}
			bs := bloqueos[p+fecha]
			if slices.ContainsFunc(bs, func(b bloqueo) bool { return b.entero }) {
				continue
			}
			clear(cap[:])
			clear(ocu[:])
			for _, f := range fs {
				for m := f.desde; m < f.hasta && m < len(cap); m++ {
					cap[m] = true
				}
			}
			for _, b := range bs {
				for m := b.desde; m < b.hasta && m < len(cap); m++ {
					cap[m] = false
				}
			}
			for _, t := range turnos[p+fecha] {
				desde := t.inicio.Hour()*60 + t.inicio.Minute()
				hasta := t.fin.Hour()*60 + t.fin.Minute()
				if t.fin.Format(time.DateOnly) != fecha {
					hasta = 24 * 60
				}
				for m := desde; m < hasta && m < len(ocu); m++ {
					ocu[m] = true
				}
			}
			for m := range cap {
				if cap[m] {
					capacidad[dia][m/60]++
					if ocu[m] {
						ocupado[dia][m/60]++
					}
				}
			}
		}
	}

	var mapa Mapa
	primera, ultima := 24, -1
	for dia := range 7 {
		for h := range 24 {
			if capacidad[dia][h] > 0 {
				primera, ultima = min(primera, h), max(ultima, h)
			}
		}
	}
	mapa.Dias, mapa.Horas, mapa.Celdas = []int{}, []int{}, []Celda{}
	if ultima < 0 {
		return mapa, nil
	}
	for i := range 7 {
		dia := (i + 1) % 7 // de lunes a domingo
		hay := false
		for h := range 24 {
			hay = hay || capacidad[dia][h] > 0
		}
		if hay {
			mapa.Dias = append(mapa.Dias, dia)
		}
	}
	for h := primera; h <= ultima; h++ {
		mapa.Horas = append(mapa.Horas, h)
	}
	for _, dia := range mapa.Dias {
		for _, h := range mapa.Horas {
			c := Celda{Dia: dia, Hora: h, Capacidad: capacidad[dia][h], Ocupado: ocupado[dia][h], Estado: "cerrado"}
			if c.Capacidad > 0 {
				c.Ocupacion = porcentaje(int64(c.Ocupado), int64(c.Capacidad))
				c.Vacio = porcentaje(int64(c.Capacidad-c.Ocupado), int64(c.Capacidad))
				c.Estado = clasificar(c)
			}
			mapa.Capacidad += c.Capacidad
			mapa.Ocupado += c.Ocupado
			mapa.Celdas = append(mapa.Celdas, c)
		}
	}
	return mapa, nil
}

func clasificarVacio(c Celda) string {
	if *c.Vacio >= 50 {
		return "suele_quedar_vacio"
	}
	return "ocupado"
}

// capacidadMinima de una franja para tomarla en serio: con menos, un solo turno cambia todo.
const capacidadMinima = 120

// peorFranja: las dos horas seguidas del mismo día que más quedan vacías (o una sola, si ningún día
// tiene dos seguidas). Nil sin horarios.
func peorFranja(m Mapa) *Franja {
	var mejor *Franja
	for _, largo := range []int{2, 1} {
		for _, dia := range m.Dias {
			for i := 0; i+largo <= len(m.Horas); i++ {
				capT, libre := 0, 0
				ok := true
				for j := range largo {
					c := m.celda(dia, m.Horas[i+j])
					if c == nil || c.Capacidad == 0 {
						ok = false
						break
					}
					capT += c.Capacidad
					libre += c.Capacidad - c.Ocupado
				}
				if !ok || capT < capacidadMinima {
					continue
				}
				v := *porcentaje(int64(libre), int64(capT))
				if mejor == nil || v > mejor.Vacio || (v == mejor.Vacio && libre > mejor.LibreMinutos) {
					mejor = &Franja{Dia: dia, HoraDesde: m.Horas[i], HoraHasta: m.Horas[i] + largo, Vacio: v, LibreMinutos: libre}
				}
			}
		}
		if mejor != nil {
			break
		}
	}
	if mejor != nil {
		mejor.Texto = nombresDia[mejor.Dia] + " de " + strconv.Itoa(mejor.HoraDesde) + " a " + strconv.Itoa(mejor.HoraHasta) +
			": vacío el " + numero(math.Round(mejor.Vacio)) + "% de las veces"
	}
	return mejor
}

// numero: 72 → "72", 12.5 → "12,5".
func numero(v float64) string {
	s := strconv.FormatFloat(v, 'f', -1, 64)
	if i := strings.IndexByte(s, '.'); i >= 0 {
		s = s[:i] + "," + s[i+1:i+2]
	}
	return s
}

// sinAcentos para armar códigos de cupón: "MIÉRCOLES" → "MIERCOLES".
var sinAcentos = strings.NewReplacer("Á", "A", "É", "E", "Í", "I", "Ó", "O", "Ú", "U")

// clientesAntiguos cuenta los clientes con email, sin baja de mails, cuyo primer turno atendido fue
// hace más de dias.
func clientesAntiguos(ctx context.Context, tx pgx.Tx, ahora time.Time, dias int) (int, error) {
	var n int
	err := tx.QueryRow(ctx, `SELECT count(*) FROM clientes c LEFT JOIN preferencias p ON p.cliente_id = c.id
		WHERE c.email <> '' AND coalesce(p.email, true)
		AND (SELECT min(t.inicio) FROM turnos t WHERE t.cliente_id = c.id AND t.estado = 'atendido') <= $1`,
		ahora.AddDate(0, 0, -dias)).Scan(&n)
	return n, err
}

// sugerir el cupón para la franja: más descuento cuanto más vacía. Nil si la franja no está tan mal.
func sugerir(ctx context.Context, tx pgx.Tx, f *Franja, ahora time.Time) (*Sugerencia, error) {
	if f == nil || f.Vacio < 40 {
		return nil, nil
	}
	pct := 10
	switch {
	case f.Vacio >= 80:
		pct = 20
	case f.Vacio >= 50:
		pct = 15
	}
	n, err := clientesAntiguos(ctx, tx, ahora, AntiguedadSugerida)
	if err != nil {
		return nil, err
	}
	return &Sugerencia{Porcentaje: pct, Dias: []int{f.Dia}, HoraDesde: hora(f.HoraDesde), HoraHasta: hora(f.HoraHasta),
		AntiguedadDias: AntiguedadSugerida, Destinatarios: n,
		Codigo: sinAcentos.Replace(strings.ToUpper(nombresDia[f.Dia])) + strconv.Itoa(f.HoraDesde),
		Texto: "Probá un " + strconv.Itoa(pct) + "% sólo en esa franja, para clientes que vienen hace más de " +
			promos.Antiguedad(AntiguedadSugerida) + "."}, nil
}

func hora(h int) string {
	if h >= 24 {
		return "23:59"
	}
	return time.Date(2000, 1, 1, h, 0, 0, 0, time.UTC).Format("15:04")
}

// horasMuertas de las últimas 8 semanas enteras (sin hoy), en el negocio fijado.
func horasMuertas(ctx context.Context, tx pgx.Tx, ahora time.Time) (HorasMuertas, error) {
	zona, err := zonaDe(ctx, tx)
	if err != nil {
		return HorasMuertas{}, err
	}
	h := ahora.In(zona)
	fin := time.Date(h.Year(), h.Month(), h.Day(), 0, 0, 0, 0, zona)
	ini := fin.AddDate(0, 0, -7*SemanasHorasMuertas)
	out := HorasMuertas{Desde: ini.Format(time.DateOnly), Hasta: fin.AddDate(0, 0, -1).Format(time.DateOnly), Semanas: SemanasHorasMuertas}
	if out.Mapa, err = calcularMapa(ctx, tx, zona, ini, fin, clasificarVacio); err != nil {
		return out, err
	}
	out.Peor = peorFranja(out.Mapa)
	out.Sugerencia, err = sugerir(ctx, tx, out.Peor, ahora)
	return out, err
}

func (s *Servicio) horasMuertas(w http.ResponseWriter, r *http.Request) error {
	var out HorasMuertas
	ahora := s.ahora()
	err := s.enNegocio(r, func(ctx context.Context, tx pgx.Tx) (err error) {
		if _, err := clientes.MarcarAusentes(ctx, tx, ahora); err != nil {
			return err
		}
		out, err = horasMuertas(ctx, tx, ahora)
		return err
	})
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusOK, out)
	return nil
}

// VigenciaCupon del cupón de horas muertas, por defecto.
const VigenciaCupon = 30

// crearCupon: el cupón de la sugerencia, listo. Todo lo del cuerpo es opcional y pisa la sugerencia.
func (s *Servicio) crearCupon(w http.ResponseWriter, r *http.Request) error {
	var in struct {
		Codigo         *string `json:"codigo"`
		Porcentaje     *int    `json:"porcentaje"`
		Dias           *[]int  `json:"dias"`
		HoraDesde      *string `json:"horaDesde"`
		HoraHasta      *string `json:"horaHasta"`
		AntiguedadDias *int    `json:"antiguedadDias"`
		VigenciaDias   *int    `json:"vigenciaDias"`
	}
	if r.ContentLength != 0 {
		if err := web.Leer(w, r, &in); err != nil {
			return err
		}
	}
	vigencia := VigenciaCupon
	if in.VigenciaDias != nil {
		if *in.VigenciaDias < 1 || *in.VigenciaDias > 365 {
			return problema.Invalido("La vigencia va de 1 a 365 días.", problema.Campo{Ruta: "vigenciaDias", Mensaje: "fuera de rango"})
		}
		vigencia = *in.VigenciaDias
	}
	ahora := s.ahora()
	var c promos.Cupon
	err := s.enNegocio(r, func(ctx context.Context, tx pgx.Tx) error {
		hm, err := horasMuertas(ctx, tx, ahora)
		if err != nil {
			return err
		}
		sug := hm.Sugerencia
		if sug == nil {
			if in.Dias == nil || in.HoraDesde == nil || in.HoraHasta == nil {
				return problema.Conflicto("sin_franja", "No hay una franja que quede vacía seguido: elegí días y horas para el cupón.")
			}
			sug = &Sugerencia{Porcentaje: 15, AntiguedadDias: AntiguedadSugerida, Codigo: "HORAS-LIBRES"}
		}
		hasta := ahora.AddDate(0, 0, vigencia)
		c = promos.Cupon{Codigo: sug.Codigo, Tipo: "porcentaje", Valor: int64(sug.Porcentaje), Desde: ahora, Hasta: &hasta, Activo: true,
			Dias: sug.Dias, HoraDesde: &sug.HoraDesde, HoraHasta: &sug.HoraHasta}
		if sug.AntiguedadDias > 0 {
			a := sug.AntiguedadDias
			c.AntiguedadDias = &a
		}
		if in.Porcentaje != nil {
			c.Valor = int64(*in.Porcentaje)
		}
		if in.Dias != nil {
			c.Dias = *in.Dias
		}
		if in.HoraDesde != nil {
			c.HoraDesde = in.HoraDesde
		}
		if in.HoraHasta != nil {
			c.HoraHasta = in.HoraHasta
		}
		if in.AntiguedadDias != nil {
			c.AntiguedadDias = in.AntiguedadDias
		}
		if in.Codigo != nil {
			c.Codigo = *in.Codigo
			if err := promos.Crear(ctx, tx, &c); err != nil {
				return err
			}
		} else {
			// El código sugerido puede estar tomado (un cupón de otro mes): se prueba con -2, -3…
			base := c.Codigo
			for i := 1; ; i++ {
				if i > 1 {
					c.Codigo = base + "-" + strconv.Itoa(i)
				}
				err := promos.Crear(ctx, tx, &c)
				if p, ok := err.(*problema.Problema); ok && p.Codigo == "duplicado" && i < 20 {
					continue
				}
				if err != nil {
					return err
				}
				break
			}
		}
		return auditoria.Anotar(ctx, tx, auditoria.Registro{Accion: "cupon.horas_muertas", Entidad: "cupon", EntidadID: c.ID,
			Detalle: map[string]any{"codigo": c.Codigo, "valor": c.Valor, "dias": c.Dias, "horaDesde": c.HoraDesde, "horaHasta": c.HoraHasta}})
	})
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusCreated, c)
	return nil
}
