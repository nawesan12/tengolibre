package reportes

import (
	"bytes"
	"context"
	"encoding/csv"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

// MaxFilasCSV de una exportación.
const MaxFilasCSV = 20000

// celda neutraliza lo que una planilla tomaría como fórmula (=, @, +, -; un teléfono +54… queda
// como está porque sigue un dígito).
func celda(v string) string {
	if v == "" {
		return v
	}
	switch v[0] {
	case '=', '@', '\t', '\r':
		return "'" + v
	case '+', '-':
		if len(v) == 1 || v[1] < '0' || v[1] > '9' {
			return "'" + v
		}
	}
	return v
}

// escribirCSV manda el archivo entero (se arma antes, así un error es un problem+json y no un CSV
// cortado). Con BOM: sin él, Excel abre los acentos rotos.
func escribirCSV(w http.ResponseWriter, nombre string, encabezado []string, filas [][]string) error {
	var b bytes.Buffer
	b.WriteString("\xEF\xBB\xBF")
	cw := csv.NewWriter(&b)
	if err := cw.Write(encabezado); err != nil {
		return err
	}
	for _, f := range filas {
		for i := range f {
			f[i] = celda(f[i])
		}
		if err := cw.Write(f); err != nil {
			return err
		}
	}
	cw.Flush()
	if err := cw.Error(); err != nil {
		return err
	}
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="`+nombre+`"`)
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
	_, err := w.Write(b.Bytes())
	return err
}

func fecha(t *time.Time, zona *time.Location, formato string) string {
	if t == nil {
		return ""
	}
	return t.In(zona).Format(formato)
}

const fechaHora = "2006-01-02 15:04"

func (s *Servicio) csvClientes(w http.ResponseWriter, r *http.Request) error {
	var filas [][]string
	var hoy string
	ahora := s.ahora()
	err := s.enNegocio(r, func(ctx context.Context, tx pgx.Tx) error {
		zona, err := zonaDe(ctx, tx)
		if err != nil {
			return err
		}
		hoy = ahora.In(zona).Format(time.DateOnly)
		fs, err := tx.Query(ctx, `SELECT c.nombre, c.telefono, c.email, c.creado_en,
			count(t.id) FILTER (WHERE t.estado = 'atendido'),
			max(t.inicio) FILTER (WHERE t.estado = 'atendido'),
			min(t.inicio) FILTER (WHERE t.estado IN ('pendiente', 'confirmado') AND t.inicio >= $1),
			coalesce(sum(t.precio) FILTER (WHERE t.estado = 'atendido'), 0),
			count(t.id) FILTER (WHERE t.estado = 'ausente'),
			coalesce((SELECT string_agg(e.nombre, ', ' ORDER BY lower(e.nombre)) FROM cliente_etiquetas ce
				JOIN etiquetas e ON e.id = ce.etiqueta_id WHERE ce.cliente_id = c.id), '')
			FROM clientes c LEFT JOIN turnos t ON t.cliente_id = c.id
			GROUP BY c.id ORDER BY lower(c.nombre), c.id LIMIT $2`, ahora, MaxFilasCSV)
		if err != nil {
			return err
		}
		defer fs.Close()
		for fs.Next() {
			var nombre, tel, email, etiquetas string
			var alta time.Time
			var ultima, proxima *time.Time
			var visitas, ausencias int
			var gastado int64
			if err := fs.Scan(&nombre, &tel, &email, &alta, &visitas, &ultima, &proxima, &gastado, &ausencias, &etiquetas); err != nil {
				return err
			}
			filas = append(filas, []string{nombre, tel, email, fecha(&alta, zona, time.DateOnly), strconv.Itoa(visitas),
				fecha(ultima, zona, fechaHora), fecha(proxima, zona, fechaHora), strconv.FormatInt(gastado, 10),
				strconv.Itoa(ausencias), etiquetas})
		}
		return fs.Err()
	})
	if err != nil {
		return err
	}
	return escribirCSV(w, "clientes_"+hoy+".csv",
		[]string{"Nombre", "Teléfono", "Email", "Alta", "Visitas", "Última visita", "Próximo turno", "Gastado", "Ausencias", "Etiquetas"}, filas)
}

// csvTurnos del rango (?desde=&hasta=, por defecto los últimos 30 días), en orden.
func (s *Servicio) csvTurnos(w http.ResponseWriter, r *http.Request) error {
	var filas [][]string
	var rg rango
	ahora := s.ahora()
	err := s.enNegocio(r, func(ctx context.Context, tx pgx.Tx) error {
		zona, err := zonaDe(ctx, tx)
		if err != nil {
			return err
		}
		if rg, err = leerRango(r, zona, ahora); err != nil {
			return err
		}
		fs, err := tx.Query(ctx, `SELECT t.inicio, c.nombre, c.telefono, c.email, t.servicio, t.profesional, t.estado, t.origen,
			t.precio, t.nota FROM turnos t JOIN clientes c ON c.id = t.cliente_id
			WHERE t.inicio >= $1 AND t.inicio < $2 ORDER BY t.inicio, t.id LIMIT $3`, rg.ini, rg.fin, MaxFilasCSV)
		if err != nil {
			return err
		}
		defer fs.Close()
		for fs.Next() {
			var inicio time.Time
			var nombre, tel, email, servicio, profesional, estado, origen, nota string
			var precio int64
			if err := fs.Scan(&inicio, &nombre, &tel, &email, &servicio, &profesional, &estado, &origen, &precio, &nota); err != nil {
				return err
			}
			en := inicio.In(zona)
			filas = append(filas, []string{en.Format(time.DateOnly), en.Format("15:04"), nombre, tel, email, servicio, profesional,
				strings.ReplaceAll(estado, "_", " "), origen, strconv.FormatInt(precio, 10), nota})
		}
		return fs.Err()
	})
	if err != nil {
		return err
	}
	return escribirCSV(w, "turnos_"+rg.desde+"_"+rg.hasta+".csv",
		[]string{"Fecha", "Hora", "Cliente", "Teléfono", "Email", "Servicio", "Profesional", "Estado", "Origen", "Precio", "Nota"}, filas)
}
