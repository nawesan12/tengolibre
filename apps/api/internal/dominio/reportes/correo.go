package reportes

import (
	"context"
	"errors"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/avisos"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/correo"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/db"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/trabajos"
)

// El reporte mensual por mail: el primer día de cada mes, a la mañana, a la dueña (y a los
// encargados que lo pidieron) de cada negocio que lo tiene prendido. El mail resume el mes anterior
// y lleva a la versión completa, imprimible, del panel (reportes.Mensual).

// HoraMensual: desde qué hora del día 1 (la del negocio) sale.
const HoraMensual = 8

// TrabajoMensual del worker: cada hora mira si es el día 1 de algún negocio.
func TrabajoMensual(pool *db.Pool) trabajos.Trabajo {
	return trabajos.Trabajo{Nombre: "reporte_mensual", Cada: time.Hour, Correr: func(ctx context.Context, ahora time.Time) (any, error) {
		return EncolarMensual(ctx, pool, ahora)
	}}
}

// Armadores para avisos.Enviador.
func Armadores() map[string]avisos.Armador {
	return map[string]avisos.Armador{avisos.Mensual: armarMensual}
}

// EncolarMensual encola el reporte del mes pasado. La clave (mes + persona) hace que salga una sola
// vez aunque el trabajo corra todo el día 1.
func EncolarMensual(ctx context.Context, pool *db.Pool, ahora time.Time) (any, error) {
	encolados := 0
	err := trabajos.PorNegocio(ctx, pool, func(_ string, tx pgx.Tx) error {
		zona, err := zonaDe(ctx, tx)
		if err != nil {
			return err
		}
		local := ahora.In(zona)
		if local.Day() != 1 || local.Hour() < HoraMensual {
			return nil
		}
		var prendido bool
		if err := tx.QueryRow(ctx, "SELECT reporte_mensual FROM ajustes WHERE negocio_id = app.negocio_actual()").Scan(&prendido); err != nil || !prendido {
			return err
		}
		mes := MesAnterior(ahora, zona)
		filas, err := tx.Query(ctx, `SELECT id::text, email FROM miembros
			WHERE activo AND email <> '' AND (rol = 'duenio' OR (rol = 'encargado' AND reporte_mensual))`)
		if err != nil {
			return err
		}
		type persona struct{ ID, Email string }
		ps, err := pgx.CollectRows(filas, pgx.RowToStructByPos[persona])
		if err != nil {
			return err
		}
		for _, p := range ps {
			ok, err := avisos.Encolar(ctx, tx, avisos.Aviso{Tipo: avisos.Mensual, Para: p.Email, Clave: "mensual:" + mes + ":" + p.ID,
				Datos: map[string]any{"mes": mes, "miembro": p.ID}}, ahora)
			if err != nil {
				return err
			}
			if ok {
				encolados++
			}
		}
		return nil
	})
	return map[string]int{"encolados": encolados}, err
}

// CartaMensual: el mail del reporte (lo usa también la vista previa de los tests).
func CartaMensual(sitio string, rm ReporteMensual) avisos.Carta {
	c := avisos.Carta{Encabezado: "Reporte mensual · " + rm.Negocio + " · " + rm.Nombre,
		Asunto: "Tu " + rm.Nombre + " en " + rm.Negocio, Titulo: rm.Titulo, Parrafos: []string{rm.Bajada}}
	for _, k := range rm.Indicadores {
		t := strings.ToUpper(k.Titulo[:1]) + strings.ToLower(k.Titulo[1:])
		v := k.ValorTexto
		if k.Texto != "" {
			v += " · " + k.Texto
		}
		c.Filas = append(c.Filas, avisos.Fila{Etiqueta: t, Valor: v})
	}
	if rm.Clientes.Nuevos+rm.Clientes.Recurrentes > 0 {
		v := rm.Clientes.Texto
		if p := rm.Clientes.Volvieron; p != nil {
			v = strings.TrimSuffix(v, ".") + " · " + numero(math.Round(*p)) + "% volvieron"
		}
		c.Filas = append(c.Filas, avisos.Fila{Etiqueta: "Clientes", Valor: v})
	}
	if rm.Agenda.HorasLibres > 0 {
		v := strconv.Itoa(rm.Agenda.HorasLibres) + " h quedaron libres"
		if f := rm.Agenda.FranjaLibre; f != nil {
			v += " · " + f.Texto
		}
		c.Filas = append(c.Filas, avisos.Fila{Etiqueta: "Huecos", Valor: v})
	}
	for _, d := range rm.Destacados {
		c.Lista = append(c.Lista, d.Titulo+": "+d.Texto)
	}
	for _, p := range rm.Siguiente {
		c.Lista = append(c.Lista, "Para "+rm.NombreSiguiente+" · "+p.Titulo+": "+p.Texto)
	}
	c.Boton, c.Link = "Ver el reporte completo", avisos.LinkMensual(sitio, rm.Mes)
	c.Notas = []string{rm.Periodo + ". El reporte completo se imprime o se guarda en PDF.",
		"Te llega el primer día de cada mes. Lo apagás en Reportes › Reporte mensual."}
	return c
}

func armarMensual(ctx context.Context, tx pgx.Tx, f avisos.EnCola, ahora time.Time) (correo.Mensaje, string, error) {
	sin := correo.Mensaje{Para: f.Para}
	var rol, email string
	var activo, pidio, prendido bool
	err := tx.QueryRow(ctx, `SELECT m.rol, m.email, m.activo, m.reporte_mensual, a.reporte_mensual FROM miembros m
		JOIN ajustes a ON a.negocio_id = m.negocio_id WHERE m.id::text = $1`, f.Datos["miembro"]).Scan(&rol, &email, &activo, &pidio, &prendido)
	if errors.Is(err, pgx.ErrNoRows) {
		return sin, "ya no es del equipo", nil
	}
	if err != nil {
		return sin, "", err
	}
	switch {
	case !prendido:
		return sin, "el negocio apagó el reporte mensual", nil
	case !activo || (rol != "duenio" && !(rol == "encargado" && pidio)):
		return sin, "ya no lo recibe", nil
	}
	if email != "" {
		sin.Para = email
	}
	rm, err := Mensual(ctx, tx, f.Datos["mes"], ahora)
	if err != nil {
		return sin, "", err
	}
	// Un mes sin turnos no manda nada (el negocio recién arranca, o está cerrado).
	if rm.Resumen.Turnos == 0 {
		return sin, "no hubo turnos ese mes", nil
	}
	return CartaMensual(f.Sitio, rm).Mensaje(sin.Para), "", nil
}
