// Package planes: qué incluye cada plan de Tengo Libre y cómo se aplica en el servidor. Es la única
// fuente de los precios, los topes y las funciones; las apps y la web muestran lo que dice acá
// (GET /v1/panel/suscripcion › catalogo).
//
// Reglas (19/11/2026): precios finales en pesos por mes, 0 % de comisión, prueba de 14 días con
// Equipo, anual = 10 meses que valen 12. Bajar de plan nunca borra nada: lo que supera el tope
// queda en sólo lectura (o fuera de la agenda, los profesionales que sobran) hasta subir o liberar
// lugar. El plan que rige lo calcula Postgres (app.plan_vigente, 00038), con la misma regla que
// suscripcion.LeerAcceso.
package planes

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/problema"
)

// Funcion que un plan incluye o no.
type Funcion string

const (
	Sena              Funcion = "sena"               // seña con Mercado Pago al reservar
	Espera            Funcion = "espera"             // lista de espera que ofrece el hueco sola
	Membresias        Funcion = "membresias"         // planes de visitas y socios
	Cupones           Funcion = "cupones"            // cupones de descuento
	Referidos         Funcion = "referidos"          // programa de referidos
	Penalidades       Funcion = "penalidades"        // penalidades por ausencias
	ReportesCompletos Funcion = "reportes_completos" // ausentes, horas muertas, informe mensual
	Sucursales        Funcion = "sucursales"         // varias sucursales en un grupo
	Dominio           Funcion = "dominio"            // dominio propio
	Plantillas        Funcion = "plantillas"         // plantillas de página (la web)
	CSV               Funcion = "csv"                // exportar reportes en CSV
)

// Todas las funciones, en orden.
var Todas = []Funcion{Sena, Espera, Membresias, Cupones, Referidos, Penalidades, ReportesCompletos, Sucursales, Dominio, Plantillas, CSV}

// Cada función, con su verbo, para los mensajes («Los cupones vienen con…»).
var nombres = map[Funcion]string{
	Sena: "La seña con Mercado Pago viene", Espera: "La lista de espera viene", Membresias: "Las membresías vienen",
	Cupones: "Los cupones vienen", Referidos: "Los referidos vienen", Penalidades: "Las penalidades por ausencias vienen",
	ReportesCompletos: "Los reportes completos vienen", Sucursales: "Las sucursales vienen", Dominio: "El dominio propio viene",
	Plantillas: "Las plantillas de página vienen", CSV: "Exportar en CSV viene",
}

// Plan que se vende.
type Plan struct {
	Clave  string `json:"clave"`
	Nombre string `json:"nombre"`
	// Precio por mes, final, en pesos. PrecioAnual: 10 meses (se usan 12).
	Precio      int64  `json:"precio"`
	PrecioAnual int64  `json:"precioAnual"`
	Destacado   bool   `json:"destacado"`
	Bajada      string `json:"bajada"`
	// Topes. Profesionales: activos que toman turnos (Centro suma extras pagos).
	Profesionales int `json:"profesionales"`
	Sucursales    int `json:"sucursales"`
	// WhatsAppIncluidos: recordatorios automáticos por WhatsApp por mes (fuera de eso, packs).
	WhatsAppIncluidos int `json:"whatsappIncluidos"`
	// CupoIA: respuestas del asistente con IA por mes; TopeDiarioIA, por día (CupoIA/20, para que el
	// cupo del mes se pueda usar en los días hábiles y un día no se coma todo).
	CupoIA       int       `json:"cupoIA"`
	TopeDiarioIA int       `json:"topeDiarioIA"`
	Funciones    []Funcion `json:"funciones"`
	// Incluye: lo que se le muestra a la persona, en orden. Nada que no exista. IncluyeTienda: lo mismo
	// para quien compra en la App Store o Google Play (sin los profesionales extra, que se pagan sólo con
	// la suscripción por Mercado Pago).
	Incluye       []string `json:"incluye"`
	IncluyeTienda []string `json:"incluyeTienda"`
}

// Lo que se cobra aparte.
const (
	// PrecioProfesionalExtra: cada profesional arriba de 25 en Centro, por mes.
	PrecioProfesionalExtra int64 = 1900
	// MaxProfesionalesExtra que se pueden sumar desde el panel.
	MaxProfesionalesExtra = 100
	// Pack de recordatorios por WhatsApp (fuera de lo que incluye Centro).
	PackWhatsAppCantidad       = 100
	PackWhatsAppPrecio   int64 = 5900
	// DiasDePrueba con Equipo.
	DiasDePrueba = 14
	// MesesAnual que se cobran en el pago anual (se usan 12).
	MesesAnual = 10
)

var equipo = []Funcion{Sena, Espera, Membresias, Cupones, Referidos, Penalidades, ReportesCompletos}

// Planes, de menor a mayor.
var Planes = []Plan{
	{Clave: "inicial", Nombre: "Inicial", Precio: 6900, PrecioAnual: 6900 * MesesAnual, Bajada: "Para un negocio chico.",
		Profesionales: 5, Sucursales: 1, CupoIA: 100, TopeDiarioIA: 5, Funciones: []Funcion{},
		Incluye: []string{"Hasta 5 profesionales", "Link propio y reservas por la web", "Recordatorios por mail y en el celular, sin límite",
			"Ficha de cada cliente y caja", "Asistente: 100 respuestas al mes", "1 sucursal"}},
	{Clave: "equipo", Nombre: "Equipo", Precio: 14900, PrecioAnual: 14900 * MesesAnual, Destacado: true,
		Bajada: "Cuando suma gente y le empiezan a faltar a los turnos.", Profesionales: 12, Sucursales: 1, CupoIA: 400, TopeDiarioIA: 20, Funciones: equipo,
		Incluye: []string{"Hasta 12 profesionales", "Todo lo de Inicial", "Seña con Mercado Pago al reservar",
			"Lista de espera que ofrece el hueco sola", "Membresías, cupones y referidos", "Penalidades por ausencias",
			"Reportes completos", "Asistente: 400 respuestas al mes"}},
	{Clave: "centro", Nombre: "Centro", Precio: 29900, PrecioAnual: 29900 * MesesAnual,
		Bajada: "Para equipos grandes o con más de un local.", Profesionales: 25, Sucursales: 3, WhatsAppIncluidos: 150, CupoIA: 1500, TopeDiarioIA: 75,
		Funciones: append(append([]Funcion{}, equipo...), Sucursales, Dominio, Plantillas, CSV),
		Incluye: []string{"Hasta 25 profesionales (cada uno extra, $1.900 por mes)", "Todo lo de Equipo", "Hasta 3 sucursales juntas, cada una con su plan",
			"Dominio propio y plantillas de página", "150 recordatorios por WhatsApp incluidos por mes", "Reportes con exportación CSV",
			"Asistente: 1.500 respuestas al mes"}},
}

func init() {
	for i := range Planes {
		p := &Planes[i]
		p.IncluyeTienda = append([]string{}, p.Incluye...)
		if p.Clave == "centro" {
			p.IncluyeTienda[0] = "Hasta 25 profesionales"
		}
	}
}

// ProblemaFueraDelPlan: 402 tope_plan de un profesional activo que quedó fuera del plan (bajó de plan
// y sobra): no toma turnos hasta subir de plan o pausar a otra persona.
func ProblemaFueraDelPlan(ctx context.Context, tx pgx.Tx, ahora time.Time, nombre string) error {
	v, err := Leer(ctx, tx, ahora)
	if err != nil {
		return err
	}
	var activos int
	if err := tx.QueryRow(ctx, "SELECT count(*) FROM profesionales WHERE activo").Scan(&activos); err != nil {
		return err
	}
	sig := SiguientePorProfesionales(activos)
	d := fmt.Sprintf("%s quedó fuera de tu plan %s (llega hasta %d profesionales): no toma turnos.", nombre, nombreDe(v.Clave), v.TopeProfesionales())
	if sig.Clave != "" && Rango(sig.Clave) > Rango(v.Clave) {
		d += " Pasá a " + sig.Nombre + " o pausá a otra persona."
	} else {
		sig = Plan{}
		d += " Pausá a otra persona para hacerle lugar."
	}
	return &problema.Problema{Estado: http.StatusPaymentRequired, Codigo: "tope_plan", Titulo: "Fuera de tu plan", Detalle: d,
		Plan: sig.Clave, Limite: v.TopeProfesionales()}
}

// SinPlan: los topes con la prueba vencida y sin suscripción (el panel está en sólo lectura; la
// página pública sigue tomando reservas con los topes de Inicial).
var SinPlan = Plan{Clave: "sin_plan", Nombre: "Sin plan", Profesionales: 5, Sucursales: 1, Funciones: []Funcion{}, Incluye: []string{}}

// De una clave ("completo" es el Equipo de antes). ok es false si no es un plan que se vende.
func De(clave string) (Plan, bool) {
	if clave == "completo" {
		clave = "equipo"
	}
	for _, p := range Planes {
		if p.Clave == clave {
			return p, true
		}
	}
	return Plan{}, false
}

// Rango: 0 sin plan, 1 Inicial, 2 Equipo, 3 Centro.
func Rango(clave string) int {
	for i, p := range Planes {
		if p.Clave == clave || (clave == "completo" && p.Clave == "equipo") {
			return i + 1
		}
	}
	return 0
}

// Tiene la función.
func (p Plan) Tiene(f Funcion) bool {
	for _, x := range p.Funciones {
		if x == f {
			return true
		}
	}
	return false
}

// Precio de un período (mensual o anual), con los profesionales extra de Centro.
func (p Plan) PrecioDe(periodo string, extras int) int64 {
	mes := p.Precio
	if p.Clave == "centro" {
		mes += int64(extras) * PrecioProfesionalExtra
	}
	if periodo == "anual" {
		return mes * MesesAnual
	}
	return mes
}

// Minimo: el primer plan que incluye f.
func Minimo(f Funcion) Plan {
	for _, p := range Planes {
		if p.Tiene(f) {
			return p
		}
	}
	return Planes[len(Planes)-1]
}

// Vigente: el plan que rige para el negocio fijado en tx, y los profesionales extra que pagó.
type Vigente struct {
	Plan
	Extras int
}

// TopeProfesionales activos que toman turnos.
func (v Vigente) TopeProfesionales() int {
	if v.Clave == "centro" {
		return v.Profesionales + v.Extras
	}
	return v.Profesionales
}

// Leer el plan que rige en el negocio fijado en tx (app.plan_vigente).
func Leer(ctx context.Context, tx pgx.Tx, ahora time.Time) (Vigente, error) {
	var clave string
	var v Vigente
	err := tx.QueryRow(ctx, "SELECT app.plan_vigente($1), profesionales_extra FROM negocios WHERE id = app.negocio_actual()", ahora).
		Scan(&clave, &v.Extras)
	if err != nil {
		return v, err
	}
	p, ok := De(clave)
	if !ok {
		p = SinPlan
	}
	v.Plan = p
	return v, nil
}

// Tiene: si el plan que rige en el negocio fijado incluye f.
func Tiene(ctx context.Context, tx pgx.Tx, ahora time.Time, f Funcion) (bool, error) {
	v, err := Leer(ctx, tx, ahora)
	return err == nil && v.Tiene(f), err
}

// Exigir f: si el plan que rige no la incluye, el problema tope_plan con el plan que la trae.
func Exigir(ctx context.Context, tx pgx.Tx, ahora time.Time, f Funcion) error {
	v, err := Leer(ctx, tx, ahora)
	if err != nil {
		return err
	}
	if v.Tiene(f) {
		return nil
	}
	return ProblemaFuncion(f)
}

// ProblemaFuncion: 402 tope_plan, «X viene con el plan Y».
func ProblemaFuncion(f Funcion) *problema.Problema {
	m := Minimo(f)
	return &problema.Problema{Estado: http.StatusPaymentRequired, Codigo: "tope_plan", Titulo: "No está en tu plan",
		Detalle: nombres[f] + " con el plan " + m.Nombre + ". Pasá a " + m.Nombre + " para usarlo.", Plan: m.Clave}
}

// ProblemaTope: 402 tope_plan por cantidad (profesionales, sucursales), con el plan que alcanza.
// que: «profesionales activos», «sucursales». sugerido vacío: no hay plan más grande (Centro).
func ProblemaTope(que string, limite int, actual string, sugerido Plan, extra string) *problema.Problema {
	d := fmt.Sprintf("Tu plan %s llega hasta %d %s.", nombreDe(actual), limite, que)
	if sugerido.Clave != "" {
		d += " Pasá a " + sugerido.Nombre + " para sumar más."
	}
	if extra != "" {
		d += " " + extra
	}
	return &problema.Problema{Estado: http.StatusPaymentRequired, Codigo: "tope_plan", Titulo: "Llegaste al tope de tu plan",
		Detalle: d, Plan: sugerido.Clave, Limite: limite}
}

func nombreDe(clave string) string {
	if p, ok := De(clave); ok {
		return p.Nombre
	}
	return "actual"
}

// SiguientePorProfesionales: el primer plan que permite n profesionales activos (vacío si ninguno).
func SiguientePorProfesionales(n int) Plan {
	for _, p := range Planes {
		if p.Profesionales >= n {
			return p
		}
	}
	return Plan{}
}

// Rutas del panel que escriben con una función de plan: suscripcion.Escritura las corta con
// tope_plan. Borrar, frenar o levantar no está: liberar lugar siempre se puede. La clave es el
// patrón del mux (r.Pattern).
var Rutas = map[string]Funcion{
	"POST /v1/panel/cupones":                      Cupones,
	"PATCH /v1/panel/cupones/{id}":                Cupones,
	"PUT /v1/panel/referidos/config":              Referidos,
	"POST /v1/panel/membresias/planes":            Membresias,
	"PATCH /v1/panel/membresias/planes/{id}":      Membresias,
	"POST /v1/panel/membresias/socios":            Membresias,
	"PATCH /v1/panel/membresias/socios/{id}":      Membresias,
	"POST /v1/panel/espera":                       Espera,
	"POST /v1/panel/espera/{id}/avisar":           Espera,
	"POST /v1/panel/espera/ofertas":               Espera,
	"PUT /v1/panel/ajustes/espera":                Espera,
	"POST /v1/panel/turnos/{id}/sena":             Sena,
	"POST /v1/panel/clientes/{id}/penalidades":    Penalidades,
	"PUT /v1/panel/ausencias/ajustes":             Penalidades,
	"POST /v1/panel/reportes/horas-muertas/cupon": ReportesCompletos,
	"PUT /v1/panel/reportes/mensual/ajustes":      ReportesCompletos,
	"PUT /v1/panel/dominio":                       Dominio,
	"POST /v1/panel/dominio/verificar":            Dominio,
}

// DeRuta: la función que pide el patrón (ok false: ninguna).
func DeRuta(patron string) (Funcion, bool) {
	f, ok := Rutas[strings.TrimSpace(patron)]
	return f, ok
}

// Precio en texto ("$14.900").
func Pesos(n int64) string {
	s := strconv.FormatInt(n, 10)
	var b strings.Builder
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			b.WriteByte('.')
		}
		b.WriteRune(c)
	}
	return "$" + b.String()
}

// CupoIA: las respuestas con IA por mes y por día del plan que rige en el negocio fijado (sin plan:
// las de Inicial, para que el asistente no se apague del todo en sólo lectura). Va a
// ia.Medidor.CupoDe.
func CupoIA(ctx context.Context, tx pgx.Tx, ahora time.Time) (mensual, diario int, err error) {
	v, err := Leer(ctx, tx, ahora)
	if err != nil {
		return 0, 0, err
	}
	if v.CupoIA == 0 {
		return Planes[0].CupoIA, Planes[0].TopeDiarioIA, nil
	}
	return v.CupoIA, v.TopeDiarioIA, nil
}
