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
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/negocios"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/promos"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/auditoria"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/problema"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/sesion"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/web"
)

// Reporte mensual («Reporte mensual v2» del diseño): el resumen de un mes comparado con el anterior,
// en cuatro partes (portada con los números, agenda, plata y servicios, equipo y clientes). Lo
// muestra el front como página imprimible (GET /v1/panel/reportes/mensual) y lo resume el mail que
// sale el primer día de cada mes (avisos.Trabajos.Mensual).

// Indicador de la portada (FACTURADO, TURNOS, OCUPACIÓN, AUSENCIAS).
type Indicador struct {
	Clave    string   `json:"clave"`
	Titulo   string   `json:"titulo"`
	Valor    *float64 `json:"valor"`
	Anterior *float64 `json:"anterior"`
	// ValorTexto: "$ 2.184.000", "312", "78%". Texto: "12% más que septiembre".
	ValorTexto string `json:"valorTexto"`
	Texto      string `json:"texto"`
	// Tendencia: sube, baja o igual. Bueno: si el cambio es para bien (las ausencias que bajan, sí).
	Tendencia string `json:"tendencia"`
	Bueno     bool   `json:"bueno"`
}

// FranjaLibre del mes: donde se juntó lo libre.
type FranjaLibre struct {
	Dias      []int  `json:"dias"`
	HoraDesde int    `json:"horaDesde"`
	HoraHasta int    `json:"horaHasta"`
	Texto     string `json:"texto"`   // "Martes y miércoles, 14 a 16 h"
	Detalle   string `json:"detalle"` // "Ahí está la mitad de lo libre del mes."
	// Momento: "los martes a la tarde".
	Momento string `json:"momento"`
}

// Consejo con título y texto (la sugerencia de la agenda).
type Consejo struct {
	Titulo string `json:"titulo"`
	Texto  string `json:"texto"`
}

// AgendaMensual: la página 2.
type AgendaMensual struct {
	Ocupacion      *float64     `json:"ocupacion"`
	Titulo         string       `json:"titulo"`
	Mapa           Mapa         `json:"mapa"`
	HorasLibres    int          `json:"horasLibres"`
	FranjaLibre    *FranjaLibre `json:"franjaLibre"`
	HuecosLlenados int          `json:"huecosLlenados"`
	Sugerencia     *Consejo     `json:"sugerencia"`
}

// Semana de facturación ("1–7 oct").
type Semana struct {
	Etiqueta string `json:"etiqueta"`
	Ingresos int64  `json:"ingresos"`
}

// ServicioMes en la tabla de la página 3.
type ServicioMes struct {
	Nombre    string   `json:"nombre"`
	Turnos    int      `json:"turnos"`
	Facturado int64    `json:"facturado"`
	Parte     *float64 `json:"parte"`
}

// PlataMensual: la página 3.
type PlataMensual struct {
	Titulo   string `json:"titulo"`
	Ingresos int64  `json:"ingresos"`
	// Señas cobradas por Mercado Pago en el mes (lo devuelto ya descontado).
	Senas       int64 `json:"senas"`
	TurnosSena  int   `json:"turnosConSena"`
	Membresias  int64 `json:"membresias"`
	SociosActiv int   `json:"sociosActivos"`
	Ticket      int64 `json:"ticketPromedio"`
	TicketAntes int64 `json:"ticketAnterior"`
	// TicketTexto: "Subió $ 450".
	TicketTexto string        `json:"ticketTexto"`
	Semanas     []Semana      `json:"semanas"`
	Servicios   []ServicioMes `json:"servicios"`
	Nota        string        `json:"nota"`
}

// ProfesionalMes en el equipo.
type ProfesionalMes struct {
	ID        *string  `json:"id"`
	Nombre    string   `json:"nombre"`
	Ocupacion *float64 `json:"ocupacion"`
	Turnos    int      `json:"turnos"`
	Resenas   int      `json:"resenas"`
	Puntaje   *float64 `json:"puntaje"`
}

// ClientesMes: quiénes vinieron.
type ClientesMes struct {
	Nuevos      int      `json:"nuevos"`
	Recurrentes int      `json:"recurrentes"`
	Volvieron   *float64 `json:"volvieron"`
	Nuevas      *float64 `json:"nuevas"`
	PorReferido int      `json:"porReferido"`
	Texto       string   `json:"texto"`
}

// ResenaDestacada del mes.
type ResenaDestacada struct {
	Comentario  string `json:"comentario"`
	Nombre      string `json:"nombre"`
	Profesional string `json:"profesional"`
	Puntaje     int    `json:"puntaje"`
}

// ResenasMes del mes.
type ResenasMes struct {
	Cantidad  int              `json:"cantidad"`
	Promedio  *float64         `json:"promedio"`
	Destacada *ResenaDestacada `json:"destacada"`
}

// Paso para el mes que viene ("01", "Llená los martes a la tarde", …).
type Paso struct {
	Numero string `json:"numero"`
	Titulo string `json:"titulo"`
	Texto  string `json:"texto"`
}

// ReporteMensual completo.
type ReporteMensual struct {
	Negocio string `json:"negocio"`
	Slug    string `json:"slug"`
	// Mes y Anterior: AAAA-MM. Nombre: "octubre 2026". NombreAnterior y NombreSiguiente: "septiembre", "noviembre".
	Mes             string `json:"mes"`
	Anterior        string `json:"anterior"`
	Nombre          string `json:"nombre"`
	NombreAnterior  string `json:"nombreAnterior"`
	NombreSiguiente string `json:"nombreSiguiente"`
	Desde           string `json:"desde"`
	Hasta           string `json:"hasta"`
	// Periodo: "Del 1 al 31 de octubre de 2026 · comparado con septiembre".
	Periodo string `json:"periodo"`
	// Completo: false si el mes todavía no terminó.
	Completo        bool             `json:"completo"`
	Titulo          string           `json:"titulo"`
	Bajada          string           `json:"bajada"`
	Indicadores     []Indicador      `json:"indicadores"`
	Resumen         Resumen          `json:"resumen"`
	ResumenAnterior Resumen          `json:"resumenAnterior"`
	Agenda          AgendaMensual    `json:"agenda"`
	Plata           PlataMensual     `json:"plata"`
	Equipo          []ProfesionalMes `json:"equipo"`
	Clientes        ClientesMes      `json:"clientes"`
	Resenas         ResenasMes       `json:"resenas"`
	// Destacados: lo mejor del mes (mejor día, lo más pedido, del equipo, reseñas).
	Destacados []Consejo `json:"destacados"`
	Siguiente  []Paso    `json:"siguiente"`
	// Pie: "Hecho con Tengo Libre · tengolibre.com.ar/lumi".
	Pie string `json:"pie"`
}

var meses = [...]string{"enero", "febrero", "marzo", "abril", "mayo", "junio", "julio", "agosto", "septiembre", "octubre", "noviembre", "diciembre"}

var diasLargos = [...]string{"domingo", "lunes", "martes", "miércoles", "jueves", "viernes", "sábado"}

func nombreMes(t time.Time) string { return meses[t.Month()-1] }

// MesAnterior al de ahora en la zona: el que resume el mail del día 1.
func MesAnterior(ahora time.Time, zona *time.Location) string {
	h := ahora.In(zona)
	return time.Date(h.Year(), h.Month(), 1, 0, 0, 0, 0, zona).AddDate(0, -1, 0).Format("2006-01")
}

func ptr(v float64) *float64 { return &v }

func redondear(v float64) float64 { return math.Round(v*10) / 10 }

// Mensual arma el reporte del mes (AAAA-MM) en el negocio fijado.
func Mensual(ctx context.Context, tx pgx.Tx, mes string, ahora time.Time) (ReporteMensual, error) {
	var rm ReporteMensual
	n, err := negocios.Leer(ctx, tx)
	if err != nil {
		return rm, err
	}
	zona, err := zonaDe(ctx, tx)
	if err != nil {
		return rm, err
	}
	ini, err := time.ParseInLocation("2006-01", mes, zona)
	if err != nil {
		return rm, problema.Invalido("mes va como AAAA-MM.")
	}
	fin := ini.AddDate(0, 1, 0)
	h := ahora.In(zona)
	if ini.Year() < 2020 || ini.After(time.Date(h.Year(), h.Month(), 1, 0, 0, 0, 0, zona)) {
		return rm, problema.Invalido("Ese mes todavía no empezó.")
	}
	antIni := ini.AddDate(0, -1, 0)
	rm = ReporteMensual{Negocio: n.Nombre, Slug: n.Slug, Mes: mes, Anterior: antIni.Format("2006-01"),
		Nombre: nombreMes(ini) + " " + strconv.Itoa(ini.Year()), NombreAnterior: nombreMes(antIni), NombreSiguiente: nombreMes(fin),
		Desde: ini.Format(time.DateOnly), Hasta: fin.AddDate(0, 0, -1).Format(time.DateOnly), Completo: !fin.After(ahora),
		Pie: "Hecho con Tengo Libre · tengolibre.com.ar/" + n.Slug}
	ultimo := fin.AddDate(0, 0, -1)
	rm.Periodo = "Del 1 al " + strconv.Itoa(ultimo.Day()) + " de " + nombreMes(ini) + " de " + strconv.Itoa(ini.Year()) +
		" · comparado con " + rm.NombreAnterior

	rep, err := armar(ctx, tx, nuevoRango(zona, ini, ultimo))
	if err != nil {
		return rm, err
	}
	ant, err := armar(ctx, tx, nuevoRango(zona, antIni, ini.AddDate(0, 0, -1)))
	if err != nil {
		return rm, err
	}
	rm.Resumen, rm.ResumenAnterior = rep.Resumen, ant.Resumen
	r, a := rep.Resumen, ant.Resumen

	// Portada.
	rm.Indicadores = []Indicador{
		indicadorPlata(r.Ingresos, a.Ingresos, rm.NombreAnterior),
		indicadorTurnos(r.Turnos, a.Turnos, rm.NombreAnterior),
		indicadorPuntos("ocupacion", "OCUPACIÓN", r.Ocupacion, a.Ocupacion, rm.NombreAnterior, true),
		indicadorPuntos("ausencias", "AUSENCIAS", r.TasaAusencias, a.TasaAusencias, rm.NombreAnterior, false),
	}

	// Agenda.
	rm.Agenda.Ocupacion = r.Ocupacion
	if r.Ocupacion != nil {
		rm.Agenda.Titulo = "Ocupaste el " + numero(math.Round(*r.Ocupacion)) + "% de tus horas libres."
	} else {
		rm.Agenda.Titulo = "Todavía no hay horarios cargados para medir la ocupación."
	}
	if rm.Agenda.Mapa, err = calcularMapa(ctx, tx, zona, ini, fin, clasificarMensual); err != nil {
		return rm, err
	}
	rm.Agenda.HorasLibres = int(math.Round(float64(rm.Agenda.Mapa.Capacidad-rm.Agenda.Mapa.Ocupado) / 60))
	rm.Agenda.FranjaLibre = franjaLibre(rm.Agenda.Mapa)
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM espera e WHERE e.avisado_en >= $1 AND e.avisado_en < $2
		AND EXISTS (SELECT 1 FROM turnos t JOIN clientes c ON c.id = t.cliente_id WHERE t.servicio_id = e.servicio_id
			AND lower(c.email) = lower(e.email) AND t.creado_en >= e.avisado_en AND t.creado_en < e.avisado_en + interval '2 hours')`,
		ini, fin).Scan(&rm.Agenda.HuecosLlenados); err != nil {
		return rm, err
	}
	var ausentesHace int
	var minutosPromedio float64
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM clientes c WHERE c.email <> ''
		AND (SELECT max(t.inicio) FROM turnos t WHERE t.cliente_id = c.id AND t.estado = 'atendido') <= $1
		AND NOT EXISTS (SELECT 1 FROM turnos t WHERE t.cliente_id = c.id AND t.inicio >= $2 AND t.estado IN ('pendiente', 'confirmado'))`,
		ahora.AddDate(0, 0, -AntiguedadSugerida), ahora).Scan(&ausentesHace); err != nil {
		return rm, err
	}
	if err := tx.QueryRow(ctx, `SELECT coalesce(avg(extract(epoch FROM t.fin - t.inicio) / 60), 0) FROM turnos t
		WHERE t.inicio >= $1 AND t.inicio < $2 AND t.estado = 'atendido'`, ini, fin).Scan(&minutosPromedio); err != nil {
		return rm, err
	}
	if f := rm.Agenda.FranjaLibre; f != nil {
		pct := 15
		texto := "Probá un cupón de " + strconv.Itoa(pct) + "% sólo para " + strings.ToLower(f.Texto[:1]) + f.Texto[1:]
		if ausentesHace > 0 {
			texto += ", y mandáselo a " + cuantos(ausentesHace, "el cliente", "los "+strconv.Itoa(ausentesHace)+" clientes") +
				" que no " + plural(ausentesHace, "viene", "vienen") + " hace más de " + strconv.Itoa(AntiguedadSugerida) + " días."
			if horas := int(math.Round(float64(ausentesHace) / 2 * minutosPromedio / 60)); horas > 0 {
				texto += " Con la mitad de esos turnos llenarías unas " + cuantos(horas, "1 hora", strconv.Itoa(horas)+" horas") + " más por mes."
			}
		} else {
			texto += "."
		}
		rm.Agenda.Sugerencia = &Consejo{Titulo: "Para llenar " + f.Momento, Texto: texto}
	}

	// Plata.
	p := &rm.Plata
	p.Ingresos, p.Ticket, p.TicketAntes = r.Ingresos, r.TicketPromedio, a.TicketPromedio
	if err := tx.QueryRow(ctx, `SELECT coalesce(sum(monto - devuelto), 0), count(*) FROM pagos
		WHERE estado = 'aprobado' AND creado_en >= $1 AND creado_en < $2`, ini, fin).Scan(&p.Senas, &p.TurnosSena); err != nil {
		return rm, err
	}
	if err := tx.QueryRow(ctx, `SELECT coalesce(sum(pl.precio) FILTER (WHERE m.creado_en >= $1 AND m.creado_en < $2), 0),
			count(*) FILTER (WHERE m.desde < $2 AND m.hasta >= $2)
		FROM membresias m JOIN membresia_planes pl ON pl.id = m.plan_id WHERE m.estado = 'activa'`, ini, fin).
		Scan(&p.Membresias, &p.SociosActiv); err != nil {
		return rm, err
	}
	p.Titulo = "Entraron " + promos.Pesos(p.Ingresos) + "."
	if p.Senas > 0 {
		p.Titulo = "Entraron " + promos.Pesos(p.Ingresos) + ", y " + promos.Pesos(p.Senas) + " fueron señas cobradas."
	}
	switch d := p.Ticket - p.TicketAntes; {
	case p.TicketAntes == 0 || d == 0:
		p.TicketTexto = "Igual que " + rm.NombreAnterior
	case d > 0:
		p.TicketTexto = "Subió " + promos.Pesos(d)
	default:
		p.TicketTexto = "Bajó " + promos.Pesos(-d)
	}
	if p.TicketAntes == 0 && p.Ticket > 0 {
		p.TicketTexto = "Sin datos de " + rm.NombreAnterior
	}
	p.Nota = "Montos finales, con descuentos aplicados. Sin la comisión de Mercado Pago."
	abrev := nombreMes(ini)[:3]
	p.Semanas = []Semana{}
	for i, d := range rep.PorDia {
		if i%7 == 0 {
			hasta := min(i+7, len(rep.PorDia))
			et := strconv.Itoa(i+1) + "–" + strconv.Itoa(hasta)
			if i == 0 {
				et += " " + abrev
			}
			p.Semanas = append(p.Semanas, Semana{Etiqueta: et})
		}
		p.Semanas[len(p.Semanas)-1].Ingresos += d.Ingresos
	}
	p.Servicios = []ServicioMes{}
	servs := slices.Clone(rep.PorServicio)
	slices.SortStableFunc(servs, func(x, y Fila) int { return int(y.Ingresos - x.Ingresos) })
	var otros ServicioMes
	for i, f := range servs {
		if f.Turnos == 0 && f.Ingresos == 0 {
			continue
		}
		if i >= 4 && len(servs) > 5 {
			otros.Turnos += f.Turnos
			otros.Facturado += f.Ingresos
			continue
		}
		p.Servicios = append(p.Servicios, ServicioMes{Nombre: f.Nombre, Turnos: f.Turnos, Facturado: f.Ingresos, Parte: porcentaje(f.Ingresos, r.Ingresos)})
	}
	if otros.Turnos > 0 {
		otros.Nombre, otros.Parte = "Otros", porcentaje(otros.Facturado, r.Ingresos)
		p.Servicios = append(p.Servicios, otros)
	}

	// Equipo, con las reseñas del mes (las reseñas copian el nombre del profesional).
	type resProf struct {
		cantidad int
		promedio float64
	}
	porNombre := map[string]resProf{}
	filas, err := tx.Query(ctx, `SELECT profesional, count(*), avg(puntaje)::float8 FROM resenas WHERE creado_en >= $1 AND creado_en < $2
		GROUP BY profesional`, ini, fin)
	if err != nil {
		return rm, err
	}
	for filas.Next() {
		var nombre string
		var x resProf
		if err := filas.Scan(&nombre, &x.cantidad, &x.promedio); err != nil {
			filas.Close()
			return rm, err
		}
		porNombre[nombre] = x
	}
	filas.Close()
	if err := filas.Err(); err != nil {
		return rm, err
	}
	rm.Equipo = []ProfesionalMes{}
	for _, f := range rep.PorProfesional {
		x := ProfesionalMes{ID: f.ID, Nombre: f.Nombre, Ocupacion: f.Ocupacion, Turnos: f.Turnos}
		if rp, ok := porNombre[f.Nombre]; ok {
			x.Resenas, x.Puntaje = rp.cantidad, ptr(redondear(rp.promedio))
		}
		rm.Equipo = append(rm.Equipo, x)
	}
	slices.SortStableFunc(rm.Equipo, func(x, y ProfesionalMes) int { return y.Turnos - x.Turnos })

	// Clientes.
	c := &rm.Clientes
	c.Nuevos, c.Recurrentes = rep.ClientesNuevos, rep.ClientesRecurrentes
	c.Volvieron = porcentaje(int64(c.Recurrentes), int64(c.Nuevos+c.Recurrentes))
	c.Nuevas = porcentaje(int64(c.Nuevos), int64(c.Nuevos+c.Recurrentes))
	if err := tx.QueryRow(ctx, "SELECT count(*) FROM referido_usos WHERE creado_en >= $1 AND creado_en < $2", ini, fin).Scan(&c.PorReferido); err != nil {
		return rm, err
	}
	switch {
	case c.Nuevos == 0:
		c.Texto = "Este mes no llegaron clientes nuevos."
	case c.PorReferido > 0:
		c.Texto = cuantos(c.Nuevos, "1 cliente nuevo", strconv.Itoa(c.Nuevos)+" clientes nuevos") + ": " +
			cuantos(c.PorReferido, "1 llegó", strconv.Itoa(c.PorReferido)+" llegaron") + " por referidos."
	default:
		c.Texto = cuantos(c.Nuevos, "1 cliente nuevo", strconv.Itoa(c.Nuevos)+" clientes nuevos") + " este mes."
	}

	// Reseñas.
	var prom *float64
	if err := tx.QueryRow(ctx, "SELECT count(*), avg(puntaje)::float8 FROM resenas WHERE creado_en >= $1 AND creado_en < $2", ini, fin).
		Scan(&rm.Resenas.Cantidad, &prom); err != nil {
		return rm, err
	}
	if prom != nil {
		rm.Resenas.Promedio = ptr(redondear(*prom))
	}
	var d ResenaDestacada
	err = tx.QueryRow(ctx, `SELECT comentario, nombre, profesional, puntaje FROM resenas
		WHERE creado_en >= $1 AND creado_en < $2 AND visible AND comentario <> '' AND length(comentario) <= 200
		ORDER BY puntaje DESC, length(comentario) DESC, creado_en DESC LIMIT 1`, ini, fin).Scan(&d.Comentario, &d.Nombre, &d.Profesional, &d.Puntaje)
	if err == nil {
		d.Nombre = primerNombre(d.Nombre)
		rm.Resenas.Destacada = &d
	} else if err != pgx.ErrNoRows {
		return rm, err
	}

	// Titular, bajada, lo mejor del mes y lo que sigue.
	if err := titular(ctx, tx, &rm, ini, zona); err != nil {
		return rm, err
	}
	rm.Destacados = destacados(rm, rep, zona)
	if rm.Siguiente, err = siguiente(ctx, tx, rm, ahora); err != nil {
		return rm, err
	}
	return rm, nil
}

func clasificarMensual(c Celda) string {
	switch o := *c.Ocupacion; {
	case o >= 85:
		return "lleno"
	case o >= 50:
		return "a_medias"
	}
	return "libre"
}

func primerNombre(n string) string {
	if f := strings.Fields(n); len(f) > 0 {
		return f[0]
	}
	return n
}

func plural(n int, uno, varios string) string {
	if n == 1 {
		return uno
	}
	return varios
}

func cuantos(n int, uno, varios string) string { return plural(n, uno, varios) }

func momento(hora int) string {
	switch {
	case hora < 12:
		return "a la mañana"
	case hora < 19:
		return "a la tarde"
	}
	return "a la noche"
}

// franjaLibre: la peor franja del mes, con los otros días en que esas mismas horas también quedaron
// más vacías que llenas. Nil si nada quedó libre de verdad.
func franjaLibre(m Mapa) *FranjaLibre {
	peor := peorFranja(m)
	if peor == nil || peor.Vacio < 50 {
		return nil
	}
	f := &FranjaLibre{HoraDesde: peor.HoraDesde, HoraHasta: peor.HoraHasta}
	libreFranja := 0
	for _, dia := range m.Dias {
		capT, libre := 0, 0
		for h := peor.HoraDesde; h < peor.HoraHasta; h++ {
			if c := m.celda(dia, h); c != nil {
				capT += c.Capacidad
				libre += c.Capacidad - c.Ocupado
			}
		}
		if capT > 0 && (dia == peor.Dia || float64(libre)/float64(capT) >= .5) {
			f.Dias = append(f.Dias, dia)
			libreFranja += libre
		}
	}
	nombres := make([]string, len(f.Dias))
	plurales := make([]string, len(f.Dias))
	for i, d := range f.Dias {
		nombres[i] = diasLargos[d]
		plurales[i] = promos.DiasSemana[d]
	}
	t := promos.Enumerar(nombres)
	f.Texto = strings.ToUpper(t[:1]) + t[1:]
	f.Texto += ", " + strconv.Itoa(f.HoraDesde) + " a " + strconv.Itoa(f.HoraHasta) + " h"
	f.Momento = "los " + promos.Enumerar(plurales) + " " + momento(f.HoraDesde)
	if total := m.Capacidad - m.Ocupado; total > 0 {
		parte := float64(libreFranja) * 100 / float64(total)
		switch {
		case parte >= 45 && parte <= 55:
			f.Detalle = "Ahí está la mitad de lo libre del mes."
		default:
			f.Detalle = "Ahí está el " + strconv.Itoa(int(math.Round(parte))) + "% de lo libre del mes."
		}
	}
	return f
}

func indicadorPlata(v, a int64, anterior string) Indicador {
	x := Indicador{Clave: "facturado", Titulo: "FACTURADO", Valor: ptr(float64(v)), Anterior: ptr(float64(a)), ValorTexto: promos.Pesos(v)}
	switch {
	case a == 0 && v == 0:
		x.Texto, x.Tendencia = "Sin facturación", "igual"
	case a == 0:
		x.Texto, x.Tendencia, x.Bueno = "Primer mes con facturación", "sube", true
	case v == a:
		x.Texto, x.Tendencia = "Igual que "+anterior, "igual"
	case v > a:
		x.Texto, x.Tendencia, x.Bueno = strconv.Itoa(int(math.Round(float64(v-a)*100/float64(a))))+"% más que "+anterior, "sube", true
	default:
		x.Texto, x.Tendencia = strconv.Itoa(int(math.Round(float64(a-v)*100/float64(a))))+"% menos que "+anterior, "baja"
	}
	return x
}

func indicadorTurnos(v, a int, anterior string) Indicador {
	x := Indicador{Clave: "turnos", Titulo: "TURNOS", Valor: ptr(float64(v)), Anterior: ptr(float64(a)), ValorTexto: strconv.Itoa(v)}
	switch {
	case v == a:
		x.Texto, x.Tendencia = "Igual que "+anterior, "igual"
	case v > a:
		x.Texto, x.Tendencia, x.Bueno = strconv.Itoa(v-a)+" más que "+anterior, "sube", true
	default:
		x.Texto, x.Tendencia = strconv.Itoa(a-v)+" menos que "+anterior, "baja"
	}
	return x
}

// indicadorPuntos: un porcentaje que se compara en puntos (ocupación) o contra el anterior
// (ausencias: «Bajó del 9%»). subirEsBueno dice para qué lado es buena noticia.
func indicadorPuntos(clave, titulo string, v, a *float64, anterior string, subirEsBueno bool) Indicador {
	x := Indicador{Clave: clave, Titulo: titulo, Valor: v, Anterior: a, ValorTexto: "—", Tendencia: "igual"}
	if v == nil {
		x.Texto = "Sin datos"
		return x
	}
	x.ValorTexto = numero(math.Round(*v)) + "%"
	if a == nil {
		x.Texto = "Sin datos de " + anterior
		return x
	}
	d := math.Round(*v - *a)
	switch {
	case d == 0:
		x.Texto = "Igual que " + anterior
	case d > 0:
		x.Tendencia, x.Bueno = "sube", subirEsBueno
	default:
		x.Tendencia, x.Bueno = "baja", !subirEsBueno
	}
	if d != 0 {
		if clave == "ausencias" {
			verbo := "Subió"
			if d < 0 {
				verbo = "Bajó"
			}
			x.Texto = verbo + " del " + numero(math.Round(*a)) + "%"
		} else {
			p := int(math.Abs(d))
			x.Texto = cuantos(p, "1 punto", strconv.Itoa(p)+" puntos")
			if d > 0 {
				x.Texto += " más"
			} else {
				x.Texto += " menos"
			}
		}
	}
	return x
}

// titular y bajada de la portada.
func titular(ctx context.Context, tx pgx.Tx, rm *ReporteMensual, ini time.Time, zona *time.Location) error {
	r, a := rm.Resumen, rm.ResumenAnterior
	if r.Turnos == 0 {
		rm.Titulo = "Un mes sin turnos."
		rm.Bajada = "No hubo turnos en " + nombreMes(ini) + ". Cuando empiecen a reservar, acá vas a ver cómo te fue."
		return nil
	}
	// El mejor mes del año: más facturado que cada uno de los 11 anteriores (y alguno tuvo algo).
	mejor, alguno := r.Ingresos > 0, false
	for i := 1; i <= 11; i++ {
		m := ini.AddDate(0, -i, 0)
		_, ing, _, err := contar(ctx, tx, m, m.AddDate(0, 1, 0))
		if err != nil {
			return err
		}
		alguno = alguno || ing > 0
		if ing >= r.Ingresos {
			mejor = false
		}
	}
	switch {
	case mejor && alguno:
		rm.Titulo = "Tu mejor mes del año."
	case r.Ingresos > a.Ingresos:
		rm.Titulo = "Un mes mejor que " + rm.NombreAnterior + "."
	case r.Ingresos < a.Ingresos:
		rm.Titulo = "Un mes más tranquilo que " + rm.NombreAnterior + "."
	default:
		rm.Titulo = "Un mes parejo con " + rm.NombreAnterior + "."
	}
	var b string
	switch {
	case a.Ingresos == 0:
		b = "Facturaste " + promos.Pesos(r.Ingresos)
	case r.Ingresos == a.Ingresos:
		b = "Facturaste lo mismo que en " + rm.NombreAnterior
	case r.Ingresos > a.Ingresos:
		b = "Facturaste un " + strconv.Itoa(int(math.Round(float64(r.Ingresos-a.Ingresos)*100/float64(a.Ingresos)))) + "% más que en " + rm.NombreAnterior
	default:
		b = "Facturaste un " + strconv.Itoa(int(math.Round(float64(a.Ingresos-r.Ingresos)*100/float64(a.Ingresos)))) + "% menos que en " + rm.NombreAnterior
	}
	switch t := r.TasaAusencias; {
	case t != nil && *t < 5:
		b += " y casi nadie faltó."
	case t != nil && *t >= 10:
		b += ", pero faltó el " + numero(math.Round(*t)) + "% de los turnos."
	default:
		b += "."
	}
	if f := rm.Agenda.FranjaLibre; f != nil {
		b += " Quedan lugares libres " + f.Momento + ": más abajo te contamos cómo llenarlos."
	}
	rm.Bajada = b
	return nil
}

// destacados: lo mejor del mes.
func destacados(rm ReporteMensual, rep Reporte, zona *time.Location) []Consejo {
	out := []Consejo{}
	var mejor *Dia
	for i := range rep.PorDia {
		if d := &rep.PorDia[i]; d.Ingresos > 0 && (mejor == nil || d.Ingresos > mejor.Ingresos) {
			mejor = d
		}
	}
	if mejor != nil {
		f, _ := time.ParseInLocation(time.DateOnly, mejor.Fecha, zona)
		out = append(out, Consejo{Titulo: "Mejor día", Texto: "El " + diasLargos[f.Weekday()] + " " + strconv.Itoa(f.Day()) + " fue tu mejor día: " +
			promos.Pesos(mejor.Ingresos) + " en " + cuantos(mejor.Atendidos, "1 turno", strconv.Itoa(mejor.Atendidos)+" turnos") + "."})
	}
	var top *Fila
	for i := range rep.PorServicio {
		if f := &rep.PorServicio[i]; f.Turnos > 0 && (top == nil || f.Turnos > top.Turnos) {
			top = f
		}
	}
	if top != nil {
		out = append(out, Consejo{Titulo: "Lo más pedido", Texto: top.Nombre + ": " + cuantos(top.Turnos, "1 turno", strconv.Itoa(top.Turnos)+" turnos") + "."})
	}
	if len(rm.Equipo) > 1 && rm.Equipo[0].Turnos > 0 {
		e := rm.Equipo[0]
		t := e.Nombre + " atendió " + cuantos(e.Turnos, "1 turno", strconv.Itoa(e.Turnos)+" turnos")
		if e.Ocupacion != nil {
			t += ", con " + numero(math.Round(*e.Ocupacion)) + "% de ocupación"
		}
		out = append(out, Consejo{Titulo: "Del equipo", Texto: t + "."})
	}
	if rm.Resenas.Promedio != nil {
		out = append(out, Consejo{Titulo: "Reseñas", Texto: numero(*rm.Resenas.Promedio) + " ★ en " +
			cuantos(rm.Resenas.Cantidad, "1 reseña nueva", strconv.Itoa(rm.Resenas.Cantidad)+" reseñas nuevas") + "."})
	}
	return out
}

// siguiente: hasta tres pasos para el mes que viene, de lo más concreto a lo más general.
func siguiente(ctx context.Context, tx pgx.Tx, rm ReporteMensual, ahora time.Time) ([]Paso, error) {
	var ps []Paso
	agregar := func(titulo, texto string) {
		if len(ps) < 3 {
			ps = append(ps, Paso{Numero: "0" + strconv.Itoa(len(ps)+1), Titulo: titulo, Texto: texto})
		}
	}
	if f := rm.Agenda.FranjaLibre; f != nil {
		agregar("Llená "+f.Momento, "Cupón de 15% para "+strconv.Itoa(f.HoraDesde)+" a "+strconv.Itoa(f.HoraHasta)+
			" h y aviso a los clientes que no vienen hace "+strconv.Itoa(AntiguedadSugerida)+" días.")
	}
	var esperando int
	if err := tx.QueryRow(ctx, "SELECT count(*) FROM espera WHERE avisado_en IS NULL AND hasta >= $1::date", ahora.Format(time.DateOnly)).
		Scan(&esperando); err != nil {
		return nil, err
	}
	if esperando > 0 {
		agregar("Sumá horarios donde hay espera", cuantos(esperando, "Queda 1 persona", "Quedan "+strconv.Itoa(esperando)+" personas")+
			" en lista de espera: un día más de atención los convierte en turnos.")
	}
	if t := rm.Resumen.TasaAusencias; t != nil && *t >= 8 {
		var modo string
		if err := tx.QueryRow(ctx, "SELECT cobro_modo FROM ajustes WHERE negocio_id = app.negocio_actual()").Scan(&modo); err != nil {
			return nil, err
		}
		if modo == "sin_cobro" {
			agregar("Pedí seña al reservar", "Faltó el "+numero(math.Round(*t))+"% de los turnos: con seña, casi nadie falta.")
		} else {
			agregar("Recordales el turno", "Faltó el "+numero(math.Round(*t))+"% de los turnos: el recordatorio del día antes ayuda.")
		}
	}
	if rm.Resumen.Atendidos > 0 && rm.Resenas.Cantidad*10 < rm.Resumen.Atendidos && len(rm.Plata.Servicios) > 0 {
		agregar("Pedí reseñas después de "+strings.ToLower(rm.Plata.Servicios[0].Nombre),
			"Es lo que más facturás: cada reseña nueva ayuda a que te encuentren.")
	}
	agregar("Compartí tu link", "tengolibre.com.ar/"+rm.Slug+" en tu Instagram y tu WhatsApp: es lo que más reservas trae.")
	return ps, nil
}

// leerMes: ?mes=AAAA-MM; por defecto, el mes pasado.
func (s *Servicio) mensual(w http.ResponseWriter, r *http.Request) error {
	var rm ReporteMensual
	ahora := s.ahora()
	err := s.enNegocio(r, func(ctx context.Context, tx pgx.Tx) error {
		if _, err := clientes.MarcarAusentes(ctx, tx, ahora); err != nil {
			return err
		}
		mes := r.URL.Query().Get("mes")
		if mes == "" {
			zona, err := zonaDe(ctx, tx)
			if err != nil {
				return err
			}
			mes = MesAnterior(ahora, zona)
		}
		var err error
		rm, err = Mensual(ctx, tx, mes, ahora)
		return err
	})
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusOK, rm)
	return nil
}

// EncargadoReporte: quién del equipo (encargados) recibe el reporte.
type EncargadoReporte struct {
	ID     string `json:"id"`
	Nombre string `json:"nombre"`
	Email  string `json:"email"`
	Recibe bool   `json:"recibe"`
}

// AjustesMensual: el reporte prendido para el negocio (le llega a la dueña) y qué encargados lo piden.
type AjustesMensual struct {
	Activo     bool               `json:"activo"`
	Encargados []EncargadoReporte `json:"encargados"`
}

func leerAjustesMensual(ctx context.Context, tx pgx.Tx) (AjustesMensual, error) {
	var a AjustesMensual
	if err := tx.QueryRow(ctx, "SELECT reporte_mensual FROM ajustes WHERE negocio_id = app.negocio_actual()").Scan(&a.Activo); err != nil {
		return a, err
	}
	filas, err := tx.Query(ctx, `SELECT id::text, nombre, email, reporte_mensual FROM miembros
		WHERE rol = 'encargado' AND activo ORDER BY lower(nombre), id`)
	if err != nil {
		return a, err
	}
	a.Encargados, err = pgx.CollectRows(filas, pgx.RowToStructByPos[EncargadoReporte])
	return a, err
}

func (s *Servicio) leerAjustesMensual(w http.ResponseWriter, r *http.Request) error {
	var a AjustesMensual
	err := s.enNegocio(r, func(ctx context.Context, tx pgx.Tx) (err error) {
		a, err = leerAjustesMensual(ctx, tx)
		return err
	})
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusOK, a)
	return nil
}

// guardarAjustesMensual: activo, y la lista de encargados que cambian (los que no vienen quedan igual).
// Un encargado sólo se cambia a sí mismo; la dueña cambia a cualquiera.
func (s *Servicio) guardarAjustesMensual(w http.ResponseWriter, r *http.Request) error {
	var in struct {
		Activo     *bool `json:"activo"`
		Encargados []struct {
			ID     string `json:"id"`
			Recibe bool   `json:"recibe"`
		} `json:"encargados"`
	}
	if err := web.Leer(w, r, &in); err != nil {
		return err
	}
	ses := sesion.De(r.Context())
	var a AjustesMensual
	err := s.enNegocio(r, func(ctx context.Context, tx pgx.Tx) error {
		antes, err := leerAjustesMensual(ctx, tx)
		if err != nil {
			return err
		}
		if in.Activo != nil {
			if _, err := tx.Exec(ctx, "UPDATE ajustes SET reporte_mensual = $1, actualizado_en = now() WHERE negocio_id = app.negocio_actual()",
				*in.Activo); err != nil {
				return err
			}
		}
		for _, e := range in.Encargados {
			q := "UPDATE miembros SET reporte_mensual = $2 WHERE id::text = $1 AND rol = 'encargado'"
			args := []any{e.ID, e.Recibe}
			if ses.Rol != sesion.Duenio {
				q += " AND clerk_id = $3"
				args = append(args, ses.ClerkID)
			}
			tag, err := tx.Exec(ctx, q, args...)
			if err != nil {
				return err
			}
			if tag.RowsAffected() == 0 {
				return problema.NoEncontrado("Ese encargado no existe, o no podés cambiarlo.")
			}
		}
		if a, err = leerAjustesMensual(ctx, tx); err != nil {
			return err
		}
		return auditoria.Anotar(ctx, tx, auditoria.Registro{Accion: "ajustes.reporte_mensual", Entidad: "ajustes",
			Detalle: map[string]any{"antes": antes, "despues": a}})
	})
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusOK, a)
	return nil
}
