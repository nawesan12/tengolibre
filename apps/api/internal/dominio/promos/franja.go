package promos

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/disponibilidad"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/problema"
)

// Restricciones de un cupón: días de la semana, una franja horaria y la antigüedad del cliente. Las
// usa el cupón listo de «Horas muertas» (reportes), y cualquier cupón del panel puede llevarlas. Un
// cupón sin restricciones vale como siempre.

// DiasSemana en plural, como en «los martes» (0 = domingo).
var DiasSemana = [...]string{"domingos", "lunes", "martes", "miércoles", "jueves", "viernes", "sábados"}

// minutos de "HH:MM", o -1 si no es una hora.
func minutos(h string) int {
	t, err := time.Parse("15:04", h)
	if err != nil {
		return -1
	}
	return t.Hour()*60 + t.Minute()
}

// horaCorta: "14:00" → "14", "14:30" → "14:30".
func horaCorta(h string) string {
	if strings.HasSuffix(h, ":00") {
		return strings.TrimPrefix(strings.TrimSuffix(h, ":00"), "0")
	}
	return h
}

// Enumerar: "a", "a y b", "a, b y c".
func Enumerar(xs []string) string {
	switch len(xs) {
	case 0:
		return ""
	case 1:
		return xs[0]
	}
	return strings.Join(xs[:len(xs)-1], ", ") + " y " + xs[len(xs)-1]
}

// Franja en palabras: "los martes y miércoles de 14 a 16 h". Vacío si no hay días ni horas.
func Franja(dias []int, desde, hasta *string) string {
	var partes []string
	if len(dias) > 0 && len(dias) < 7 {
		ds := slices.Clone(dias)
		// De lunes a domingo, como se lee una semana.
		slices.SortFunc(ds, func(a, b int) int { return (a+6)%7 - (b+6)%7 })
		nombres := make([]string, len(ds))
		for i, d := range ds {
			nombres[i] = DiasSemana[d]
		}
		partes = append(partes, "los "+Enumerar(nombres))
	}
	if desde != nil && hasta != nil {
		partes = append(partes, "de "+horaCorta(*desde)+" a "+horaCorta(*hasta)+" h")
	}
	return strings.Join(partes, " ")
}

// Condiciones del cupón en una frase («Vale los martes de 14 a 16 h, para clientes que vienen hace
// más de 2 meses.»). Vacío sin restricciones.
func Condiciones(c Cupon) string {
	f := Franja(c.Dias, c.HoraDesde, c.HoraHasta)
	var a string
	if c.AntiguedadDias != nil {
		a = "para clientes que vienen hace más de " + Antiguedad(*c.AntiguedadDias)
	}
	switch {
	case f != "" && a != "":
		return "Vale " + f + ", " + a + "."
	case f != "":
		return "Vale " + f + "."
	case a != "":
		return "Vale " + a + "."
	}
	return ""
}

// Antiguedad en palabras: 60 → "2 meses", 30 → "1 mes", 45 → "45 días".
func Antiguedad(dias int) string {
	switch {
	case dias%30 == 0 && dias/30 == 1:
		return "1 mes"
	case dias%30 == 0:
		return strconv.Itoa(dias/30) + " meses"
	case dias == 1:
		return "1 día"
	}
	return strconv.Itoa(dias) + " días"
}

func validarRestricciones(c *Cupon) error {
	if c.Dias == nil {
		c.Dias = []int{}
	}
	vistos := map[int]bool{}
	for _, d := range c.Dias {
		if d < 0 || d > 6 || vistos[d] {
			return problema.Invalido("Los días van de 0 (domingo) a 6 (sábado), sin repetir.", problema.Campo{Ruta: "dias", Mensaje: "inválido"})
		}
		vistos[d] = true
	}
	slices.Sort(c.Dias)
	if c.HoraDesde != nil && *c.HoraDesde == "" {
		c.HoraDesde = nil
	}
	if c.HoraHasta != nil && *c.HoraHasta == "" {
		c.HoraHasta = nil
	}
	if (c.HoraDesde == nil) != (c.HoraHasta == nil) {
		return problema.Invalido("La franja lleva hora de inicio y de fin.", problema.Campo{Ruta: "horaDesde", Mensaje: "incompleto"})
	}
	if c.HoraDesde != nil {
		d, h := minutos(*c.HoraDesde), minutos(*c.HoraHasta)
		if d < 0 || h < 0 || h <= d {
			return problema.Invalido("La franja va como HH:MM y termina después de empezar.", problema.Campo{Ruta: "horaHasta", Mensaje: "fuera de rango"})
		}
	}
	if c.AntiguedadDias != nil && *c.AntiguedadDias == 0 {
		c.AntiguedadDias = nil
	}
	if c.AntiguedadDias != nil && (*c.AntiguedadDias < 1 || *c.AntiguedadDias > 3650) {
		return problema.Invalido("La antigüedad va de 1 a 3650 días.", problema.Campo{Ruta: "antiguedadDias", Mensaje: "fuera de rango"})
	}
	return nil
}

// zona del negocio fijado.
func zona(ctx context.Context, tx pgx.Tx) (*time.Location, error) {
	var z string
	if err := tx.QueryRow(ctx, "SELECT zona_horaria FROM negocios WHERE id = app.negocio_actual()").Scan(&z); err != nil {
		return nil, err
	}
	l, err := time.LoadLocation(z)
	if err != nil {
		return disponibilidad.ZonaPorDefecto, nil
	}
	return l, nil
}

// EnFranja dice si un turno que empieza en inicio (hora local) cae en los días y la franja del cupón.
func EnFranja(c Cupon, inicio time.Time) bool {
	if len(c.Dias) > 0 && !slices.Contains(c.Dias, int(inicio.Weekday())) {
		return false
	}
	if c.HoraDesde != nil && c.HoraHasta != nil {
		m := inicio.Hour()*60 + inicio.Minute()
		if m < minutos(*c.HoraDesde) || m >= minutos(*c.HoraHasta) {
			return false
		}
	}
	return true
}

// restriccion: por qué el cupón no vale para ese turno y ese cliente (vacío: vale).
func restriccion(ctx context.Context, tx pgx.Tx, c Cupon, inicio time.Time, clienteID string, ahora time.Time) (string, error) {
	if len(c.Dias) > 0 || c.HoraDesde != nil {
		z, err := zona(ctx, tx)
		if err != nil {
			return "", err
		}
		if !EnFranja(c, inicio.In(z)) {
			return "Este cupón vale sólo " + Franja(c.Dias, c.HoraDesde, c.HoraHasta), nil
		}
	}
	if c.AntiguedadDias != nil {
		var primera *time.Time
		if err := tx.QueryRow(ctx, "SELECT min(inicio) FROM turnos WHERE cliente_id = $1 AND estado = 'atendido'", clienteID).
			Scan(&primera); err != nil {
			return "", err
		}
		if primera == nil || primera.After(ahora.AddDate(0, 0, -*c.AntiguedadDias)) {
			return "Este cupón es para clientes que vienen hace más de " + Antiguedad(*c.AntiguedadDias), nil
		}
	}
	return "", nil
}

// Crear valida y guarda un cupón en el negocio fijado (con sus restricciones). Un código repetido
// es 409 duplicado.
func Crear(ctx context.Context, tx pgx.Tx, c *Cupon) error {
	c.Codigo = Normalizar(c.Codigo)
	if !reCodigo.MatchString(c.Codigo) {
		return problema.Invalido("El código va con letras, números, - o _ (de 3 a 40).", problema.Campo{Ruta: "codigo", Mensaje: "formato inválido"})
	}
	if err := validarCupon(c); err != nil {
		return err
	}
	// El savepoint deja la transacción usable si el código choca (quien llama puede probar otro).
	sp, err := tx.Begin(ctx)
	if err != nil {
		return err
	}
	err = escanearCupon(sp.QueryRow(ctx, `INSERT INTO cupones (negocio_id, codigo, tipo, valor, minimo, max_usos, desde, hasta, activo,
			dias, hora_desde, hora_hasta, antiguedad_dias)
		VALUES (app.negocio_actual(), $1, $2, $3, $4, $5, $6, $7, $8, $9, $10::time, $11::time, $12) RETURNING `+columnasCupon,
		c.Codigo, c.Tipo, c.Valor, c.Minimo, c.MaxUsos, c.Desde, c.Hasta, c.Activo, c.Dias, c.HoraDesde, c.HoraHasta, c.AntiguedadDias), c)
	var pg *pgconn.PgError
	if errors.As(err, &pg) && pg.Code == "23505" {
		_ = sp.Rollback(ctx)
		return problema.Conflicto("duplicado", "Ya hay un cupón con ese código.")
	}
	if err != nil {
		_ = sp.Rollback(ctx)
		return err
	}
	return sp.Commit(ctx)
}

// Leer un cupón del negocio fijado.
func Leer(ctx context.Context, tx pgx.Tx, id string) (Cupon, error) {
	var c Cupon
	err := escanearCupon(tx.QueryRow(ctx, "SELECT "+columnasCupon+" FROM cupones WHERE id = $1", id), &c)
	return c, err
}

// Vigente dice si el cupón sirve a esa hora (activo, en fechas y con usos).
func Vigente(c Cupon, ahora time.Time) bool {
	return c.Activo && !c.Desde.After(ahora) && (c.Hasta == nil || c.Hasta.After(ahora)) && (c.MaxUsos == nil || c.Usos < *c.MaxUsos)
}

// DescuentoEnPalabras: "15%" o "$ 2.000".
func DescuentoEnPalabras(c Cupon) string {
	if c.Tipo == "porcentaje" {
		return fmt.Sprintf("%d%%", c.Valor)
	}
	return Pesos(c.Valor)
}

// Pesos con separador de miles: 2184000 → "$ 2.184.000".
func Pesos(v int64) string {
	signo := ""
	if v < 0 {
		signo, v = "−", -v
	}
	s := strconv.FormatInt(v, 10)
	var b strings.Builder
	for i, r := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			b.WriteByte('.')
		}
		b.WriteRune(r)
	}
	return signo + "$ " + b.String()
}
