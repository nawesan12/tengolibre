// Package auditoria anota quién hizo qué en el panel, en la misma transacción que el cambio, y lo
// muestra (listado y CSV) a quien administra el negocio.
package auditoria

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/db"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/problema"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/sesion"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/web"
)

// Registro es una entrada. Quien vacío: la persona de la sesión del contexto; si tampoco hay, el
// sistema. Accion va como "entidad.hecho" (p. ej. "resena.ocultada").
type Registro struct {
	Quien, Accion, Entidad, EntidadID string
	Detalle                           any
}

// Anotar inserta el registro en la transacción (que ya tiene el negocio fijado).
func Anotar(ctx context.Context, tx pgx.Tx, r Registro) error {
	if r.Quien == "" {
		r.Quien = sesion.De(ctx).ClerkID
	}
	var detalle []byte
	if r.Detalle != nil {
		var err error
		if detalle, err = json.Marshal(r.Detalle); err != nil {
			return err
		}
	}
	_, err := tx.Exec(ctx, `INSERT INTO auditoria (negocio_id, clerk_id, accion, entidad, entidad_id, detalle)
		VALUES (app.negocio_actual(), $1, $2, $3, $4, $5)`, r.Quien, r.Accion, r.Entidad, r.EntidadID, detalle)
	return err
}

// Entrada de la auditoría como la ve el panel. Nombre: el del miembro (vacío si fue el sistema o si
// ya no está en el equipo).
type Entrada struct {
	ID        string          `json:"id"`
	ClerkID   string          `json:"clerkId"`
	Nombre    string          `json:"nombre"`
	Accion    string          `json:"accion"`
	Entidad   string          `json:"entidad"`
	EntidadID string          `json:"entidadId"`
	Detalle   json.RawMessage `json:"detalle"`
	CreadoEn  time.Time       `json:"creadoEn"`
}

// DePlata: lo que filtra «Sólo cambios de plata» (?plata=1), por entidad o por el prefijo de la
// acción: cobros y señas, devoluciones, caja, precios, cupones, membresías, productos y la
// suscripción. Quien anote algo de plata nuevo lo suma acá.
var DePlata = []string{"pago", "pagos", "cobro", "cobros", "caja", "precio", "precios", "servicio_precio", "cupon",
	"membresia", "membresias", "producto", "suscripcion", "devolucion", "sena"}

// Servicio de la pantalla de auditoría.
type Servicio struct {
	Pool   *db.Pool
	Sesion *sesion.Servicio
}

// Rutas: sólo quien administra.
func (s *Servicio) Rutas(mux *http.ServeMux) {
	mux.HandleFunc("GET /v1/panel/auditoria", s.Sesion.Panel(sesion.Gestion, web.Envolver(s.listar)))
	mux.HandleFunc("GET /v1/panel/auditoria/exportar", s.Sesion.Panel(sesion.Gestion, web.Envolver(s.exportar)))
}

// filtro: desde y hasta (AAAA-MM-DD, en UTC-3 da igual: es un rango ancho), entidad, quien y antes
// (cursor: el creadoEn del último de la página anterior).
func (s *Servicio) buscar(ctx context.Context, r *http.Request, limite int) ([]Entrada, error) {
	q := r.URL.Query()
	cond := []string{"true"}
	var args []any
	agregar := func(sql string, v any) {
		args = append(args, v)
		cond = append(cond, strings.ReplaceAll(sql, "?", "$"+strconv.Itoa(len(args))))
	}
	fecha := func(nombre string, suma int) error {
		v := q.Get(nombre)
		if v == "" {
			return nil
		}
		f, err := time.Parse(time.DateOnly, v)
		if err != nil {
			return problema.Invalido(nombre + " va como AAAA-MM-DD.")
		}
		op := ">="
		if suma > 0 {
			op = "<"
		}
		agregar("a.creado_en "+op+" ?", f.AddDate(0, 0, suma))
		return nil
	}
	if err := fecha("desde", 0); err != nil {
		return nil, err
	}
	if err := fecha("hasta", 1); err != nil {
		return nil, err
	}
	if v := q.Get("entidad"); v != "" {
		agregar("a.entidad = ?", v)
	}
	if v := q.Get("quien"); v != "" {
		agregar("a.clerk_id = ?", v)
	}
	if q.Get("plata") == "1" {
		agregar("(a.entidad = ANY(?) OR split_part(a.accion, '.', 1) = ANY(?))", DePlata)
	}
	if v := q.Get("antes"); v != "" {
		t, err := time.Parse(time.RFC3339Nano, v)
		if err != nil {
			return nil, problema.Invalido("antes va como fecha RFC 3339.")
		}
		agregar("a.creado_en < ?", t)
	}
	args = append(args, limite)
	out := []Entrada{}
	err := s.Pool.EnNegocio(ctx, sesion.De(ctx).NegocioID, func(tx pgx.Tx) error {
		filas, err := tx.Query(ctx, `SELECT a.id::text, a.clerk_id, coalesce(m.nombre, ''), a.accion, a.entidad, a.entidad_id,
			coalesce(a.detalle, 'null'::jsonb), a.creado_en
			FROM auditoria a LEFT JOIN miembros m ON m.clerk_id = a.clerk_id AND a.clerk_id <> ''
			WHERE `+strings.Join(cond, " AND ")+` ORDER BY a.creado_en DESC, a.id DESC LIMIT $`+strconv.Itoa(len(args)), args...)
		if err != nil {
			return err
		}
		out, err = pgx.CollectRows(filas, func(f pgx.CollectableRow) (Entrada, error) {
			var e Entrada
			var detalle []byte
			err := f.Scan(&e.ID, &e.ClerkID, &e.Nombre, &e.Accion, &e.Entidad, &e.EntidadID, &detalle, &e.CreadoEn)
			e.Detalle = detalle
			return e, err
		})
		return err
	})
	return out, err
}

func (s *Servicio) listar(w http.ResponseWriter, r *http.Request) error {
	limite := 50
	if v, err := strconv.Atoi(r.URL.Query().Get("limite")); err == nil && v > 0 {
		limite = min(v, 200)
	}
	es, err := s.buscar(r.Context(), r, limite)
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusOK, map[string]any{"entradas": es})
	return nil
}

// exportar: los mismos filtros, hasta 10.000 filas, en CSV (para abrir en una planilla).
func (s *Servicio) exportar(w http.ResponseWriter, r *http.Request) error {
	es, err := s.buscar(r.Context(), r, 10000)
	if err != nil {
		return err
	}
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="auditoria.csv"`)
	w.Header().Set("Cache-Control", "no-store")
	// BOM: sin él, Excel abre las tildes rotas.
	_, _ = w.Write([]byte{0xEF, 0xBB, 0xBF})
	c := csv.NewWriter(w)
	_ = c.Write([]string{"fecha", "quien", "persona", "accion", "entidad", "id", "detalle"})
	for _, e := range es {
		detalle := string(e.Detalle)
		if detalle == "null" {
			detalle = ""
		}
		_ = c.Write([]string{e.CreadoEn.UTC().Format(time.RFC3339), e.ClerkID, sinFormula(e.Nombre), e.Accion, e.Entidad,
			e.EntidadID, sinFormula(detalle)})
	}
	c.Flush()
	return c.Error()
}

// sinFormula: un texto que empieza con = + - @ lo toma la planilla como fórmula (inyección en CSV).
func sinFormula(v string) string {
	if v != "" && strings.ContainsRune("=+-@\t\r", rune(v[0])) {
		return "'" + v
	}
	return v
}
