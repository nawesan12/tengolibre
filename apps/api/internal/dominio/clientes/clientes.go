// Package clientes: el CRM del panel. El listado de clientes con lo que hicieron en el negocio
// (visitas, gasto, ausencias), la ficha con su historial, las etiquetas, las notas internas, las
// penalidades por ausencias y la búsqueda global. Los clientes nacen en turnos (al reservar) o acá
// (alta manual); los turnos se leen con SQL directo para no depender del paquete turnos, que es el
// que llama a Penalizado.
package clientes

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"

	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/db"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/problema"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/sesion"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/web"
)

// Servicio de clientes. Ahora es el reloj (tests); nil: time.Now.
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

// Atencion: los que tratan con los clientes (alta, edición, etiquetas). El profesional los ve y
// les escribe notas, pero no los edita.
var Atencion = []string{sesion.Duenio, sesion.Encargado, sesion.Recepcion}

// Rutas del dominio.
func (s *Servicio) Rutas(mux *http.ServeMux) {
	a := func(h web.Manejador) http.HandlerFunc { return s.Sesion.Panel(sesion.Agenda, web.Envolver(h)) }
	at := func(h web.Manejador) http.HandlerFunc { return s.Sesion.Panel(Atencion, web.Envolver(h)) }
	g := func(h web.Manejador) http.HandlerFunc { return s.Sesion.Panel(sesion.Gestion, web.Envolver(h)) }

	mux.HandleFunc("GET /v1/panel/clientes", a(s.listar))
	mux.HandleFunc("POST /v1/panel/clientes", at(s.crear))
	mux.HandleFunc("GET /v1/panel/clientes/{id}", a(s.ficha))
	mux.HandleFunc("PATCH /v1/panel/clientes/{id}", at(s.editar))
	mux.HandleFunc("PUT /v1/panel/clientes/{id}/etiquetas/{etiqueta}", at(s.asignar))
	mux.HandleFunc("DELETE /v1/panel/clientes/{id}/etiquetas/{etiqueta}", at(s.quitar))
	mux.HandleFunc("GET /v1/panel/clientes/{id}/notas", a(s.notas))
	mux.HandleFunc("POST /v1/panel/clientes/{id}/notas", a(s.crearNota))
	mux.HandleFunc("DELETE /v1/panel/clientes/{id}/notas/{nota}", a(s.borrarNota))
	mux.HandleFunc("GET /v1/panel/clientes/{id}/penalidades", a(s.penalidadesDe))
	mux.HandleFunc("POST /v1/panel/clientes/{id}/penalidades", g(s.penalizar))

	mux.HandleFunc("GET /v1/panel/etiquetas", a(s.etiquetas))
	mux.HandleFunc("POST /v1/panel/etiquetas", g(s.crearEtiqueta))
	mux.HandleFunc("PATCH /v1/panel/etiquetas/{id}", g(s.editarEtiqueta))
	mux.HandleFunc("DELETE /v1/panel/etiquetas/{id}", g(s.borrarEtiqueta))

	mux.HandleFunc("GET /v1/panel/penalidades", a(s.penalidades))
	mux.HandleFunc("POST /v1/panel/penalidades/{id}/levantar", g(s.levantar))
	mux.HandleFunc("GET /v1/panel/ausencias", a(s.ausencias))
	mux.HandleFunc("GET /v1/panel/ausencias/ajustes", g(s.leerAjustes))
	mux.HandleFunc("PUT /v1/panel/ausencias/ajustes", g(s.guardarAjustes))
	mux.HandleFunc("POST /v1/panel/ausencias/marcar", g(s.marcar))

	mux.HandleFunc("GET /v1/panel/buscar", a(s.buscar))
}

func (s *Servicio) enNegocio(r *http.Request, fn func(context.Context, pgx.Tx) error) error {
	ctx := r.Context()
	return s.Pool.EnNegocio(ctx, sesion.De(ctx).NegocioID, func(tx pgx.Tx) error { return fn(ctx, tx) })
}

// Etiqueta de clientes.
type Etiqueta struct {
	ID     string `json:"id"`
	Nombre string `json:"nombre"`
	Color  string `json:"color"`
}

// Cliente como lo ve el listado: sus datos y lo que hizo en el negocio.
type Cliente struct {
	ID       string    `json:"id"`
	Nombre   string    `json:"nombre"`
	Telefono string    `json:"telefono"`
	Email    string    `json:"email"`
	CreadoEn time.Time `json:"creadoEn"`
	// Nacimiento: AAAA-MM-DD, opcional (para el recordatorio de cumpleaños).
	Nacimiento *string `json:"nacimiento"`
	// Visitas: turnos atendidos. Gastado: la suma de sus precios (no hay cobros todavía).
	Visitas      int        `json:"visitas"`
	UltimaVisita *time.Time `json:"ultimaVisita"`
	ProximoTurno *time.Time `json:"proximoTurno"`
	Gastado      int64      `json:"gastado"`
	Ausencias    int        `json:"ausencias"`
	Penalizado   bool       `json:"penalizado"`
	Etiquetas    []Etiqueta `json:"etiquetas"`
}

// columnasCliente(n): las columnas del resumen, con $n = ahora (para el próximo turno). Un
// pendiente_pago vencido ya no cuenta como próximo: el horario quedó libre.
func columnasCliente(n int) string {
	return strings.ReplaceAll(columnasConAhora, "$ahora", "$"+strconv.Itoa(n))
}

const columnasConAhora = `c.id::text, c.nombre, c.telefono, c.email, c.creado_en, to_char(c.nacimiento, 'YYYY-MM-DD'),
	count(t.id) FILTER (WHERE t.estado = 'atendido') AS visitas,
	max(t.inicio) FILTER (WHERE t.estado = 'atendido') AS ultima,
	min(t.inicio) FILTER (WHERE t.estado IN ('pendiente_pago', 'pendiente', 'confirmado') AND t.inicio >= $ahora
		AND (t.estado <> 'pendiente_pago' OR t.vence_en > $ahora)) AS proxima,
	coalesce(sum(t.precio) FILTER (WHERE t.estado = 'atendido'), 0) AS gastado,
	count(t.id) FILTER (WHERE t.estado = 'ausente') AS ausencias,
	EXISTS (SELECT 1 FROM penalidades p WHERE p.cliente_id = c.id AND p.levantada_en IS NULL AND p.hasta > now()) AS penalizado`

const desdeClientes = " FROM clientes c LEFT JOIN turnos t ON t.cliente_id = c.id"

func escanearCliente(f pgx.Row, c *Cliente) error {
	c.Etiquetas = []Etiqueta{}
	return f.Scan(&c.ID, &c.Nombre, &c.Telefono, &c.Email, &c.CreadoEn, &c.Nacimiento,
		&c.Visitas, &c.UltimaVisita, &c.ProximoTurno, &c.Gastado, &c.Ausencias, &c.Penalizado)
}

// LeerCliente con su resumen, en el negocio fijado.
func LeerCliente(ctx context.Context, tx pgx.Tx, id string, ahora time.Time) (Cliente, error) {
	var c Cliente
	if err := escanearCliente(tx.QueryRow(ctx, "SELECT "+columnasCliente(1)+desdeClientes+" WHERE c.id = $2 GROUP BY c.id", ahora, id), &c); err != nil {
		return c, err
	}
	err := ponerEtiquetas(ctx, tx, []*Cliente{&c})
	return c, err
}

// ponerEtiquetas carga las etiquetas de varios clientes en una sola consulta.
func ponerEtiquetas(ctx context.Context, tx pgx.Tx, cs []*Cliente) error {
	if len(cs) == 0 {
		return nil
	}
	ids := make([]string, len(cs))
	por := map[string]*Cliente{}
	for i, c := range cs {
		ids[i], por[c.ID] = c.ID, c
	}
	filas, err := tx.Query(ctx, `SELECT ce.cliente_id::text, e.id::text, e.nombre, e.color
		FROM cliente_etiquetas ce JOIN etiquetas e ON e.id = ce.etiqueta_id
		WHERE ce.cliente_id = ANY($1::uuid[]) ORDER BY lower(e.nombre)`, ids)
	if err != nil {
		return err
	}
	defer filas.Close()
	for filas.Next() {
		var cid string
		var e Etiqueta
		if err := filas.Scan(&cid, &e.ID, &e.Nombre, &e.Color); err != nil {
			return err
		}
		por[cid].Etiquetas = append(por[cid].Etiquetas, e)
	}
	return filas.Err()
}

// existe devuelve 404 si el cliente no es del negocio fijado.
func existe(ctx context.Context, tx pgx.Tx, id string) error {
	var ok bool
	if err := tx.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM clientes WHERE id = $1)", id).Scan(&ok); err != nil {
		return err
	}
	if !ok {
		return problema.NoEncontrado("Ese cliente no existe o no es de este negocio.")
	}
	return nil
}

// Penalizado dice si el cliente tiene una penalidad vigente en el negocio fijado. Lo usa la reserva
// desde la página pública (turnos.reservar).
func Penalizado(ctx context.Context, tx pgx.Tx, clienteID string) (bool, error) {
	var p bool
	err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM penalidades
		WHERE cliente_id = $1 AND levantada_en IS NULL AND hasta > now())`, clienteID).Scan(&p)
	return p, err
}

// GraciaAusente: cuánto después del fin un turno sin cerrar pasa a ausente (si el negocio lo pidió).
// Es para que quien cierra el turno un rato tarde no le cueste una penalidad al cliente.
const GraciaAusente = 2 * time.Hour

// MarcarAusentes pasa a ausente los turnos pendientes o confirmados que terminaron hace más de
// GraciaAusente, si el negocio fijado tiene marcar_ausentes. Sin cron: lo llaman el listado y la
// ficha de clientes, los reportes y POST /v1/panel/ausencias/marcar. Las penalidades que eso
// dispare las crea el trigger de la base.
func MarcarAusentes(ctx context.Context, tx pgx.Tx, ahora time.Time) (int64, error) {
	tag, err := tx.Exec(ctx, `UPDATE turnos SET estado = 'ausente', actualizado_en = now()
		WHERE estado IN ('pendiente', 'confirmado') AND fin < $1
		AND (SELECT marcar_ausentes FROM ajustes WHERE negocio_id = app.negocio_actual())`, ahora.Add(-GraciaAusente))
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

// ---- Listado

// patron para ILIKE con lo que escribió la persona, sin que % o _ hagan de comodín.
func patron(q string) string {
	r := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return "%" + r.Replace(q) + "%"
}

func soloDigitos(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// paginado lee pagina (desde 1) y porPagina (hasta max).
func paginado(r *http.Request, porDefecto, max int) (pagina, porPagina int) {
	q := r.URL.Query()
	pagina, porPagina = 1, porDefecto
	if v, err := strconv.Atoi(q.Get("pagina")); err == nil && v > 0 {
		pagina = v
	}
	if v, err := strconv.Atoi(q.Get("porPagina")); err == nil && v > 0 {
		porPagina = min(v, max)
	}
	return pagina, porPagina
}

// Órdenes del listado.
var ordenes = map[string]string{
	"nombre":   "lower(c.nombre), c.id",
	"reciente": "c.creado_en DESC, c.id",
	"ultima":   "ultima DESC NULLS LAST, lower(c.nombre), c.id",
	"gastado":  "gastado DESC, lower(c.nombre), c.id",
	"visitas":  "visitas DESC, lower(c.nombre), c.id",
}

// listar: q busca por nombre, email o teléfono; etiqueta (una o varias, separadas por coma) deja
// los que tengan alguna; penalizados=1, sólo los penalizados.
func (s *Servicio) listar(w http.ResponseWriter, r *http.Request) error {
	q := r.URL.Query()
	pagina, porPagina := paginado(r, 20, 100)
	orden, ok := ordenes[q.Get("orden")]
	if q.Get("orden") == "" {
		orden, ok = ordenes["nombre"], true
	}
	if !ok {
		return problema.Invalido("orden va como nombre, reciente, ultima, gastado o visitas.")
	}
	ahora := s.ahora()
	var args []any
	cond := []string{"true"}
	arg := func(v any) string {
		args = append(args, v)
		return "$" + strconv.Itoa(len(args))
	}
	if b := strings.TrimSpace(q.Get("q")); b != "" {
		p := arg(patron(b))
		c := "c.nombre ILIKE " + p + " OR c.email ILIKE " + p
		if d := soloDigitos(b); len(d) >= 3 {
			c += " OR c.telefono LIKE " + arg(patron(d))
		}
		cond = append(cond, "("+c+")")
	}
	if v := q.Get("etiqueta"); v != "" {
		cond = append(cond, "EXISTS (SELECT 1 FROM cliente_etiquetas ce WHERE ce.cliente_id = c.id AND ce.etiqueta_id::text = ANY("+
			arg(strings.Split(v, ","))+"))")
	}
	if q.Get("penalizados") == "1" {
		cond = append(cond, "EXISTS (SELECT 1 FROM penalidades p WHERE p.cliente_id = c.id AND p.levantada_en IS NULL AND p.hasta > now())")
	}
	donde := " WHERE " + strings.Join(cond, " AND ")
	out := struct {
		Clientes  []*Cliente `json:"clientes"`
		Total     int        `json:"total"`
		Pagina    int        `json:"pagina"`
		PorPagina int        `json:"porPagina"`
	}{Clientes: []*Cliente{}, Pagina: pagina, PorPagina: porPagina}
	err := s.enNegocio(r, func(ctx context.Context, tx pgx.Tx) error {
		if _, err := MarcarAusentes(ctx, tx, ahora); err != nil {
			return err
		}
		if err := tx.QueryRow(ctx, "SELECT count(*) FROM clientes c"+donde, args...).Scan(&out.Total); err != nil {
			return err
		}
		// ahora va al final, después de los argumentos de los filtros.
		filas, err := tx.Query(ctx, "SELECT "+columnasCliente(len(args)+1)+desdeClientes+donde+" GROUP BY c.id ORDER BY "+orden+
			" LIMIT "+strconv.Itoa(porPagina)+" OFFSET "+strconv.Itoa((pagina-1)*porPagina), append(args, ahora)...)
		if err != nil {
			return err
		}
		defer filas.Close()
		for filas.Next() {
			c := &Cliente{}
			if err := escanearCliente(filas, c); err != nil {
				return err
			}
			out.Clientes = append(out.Clientes, c)
		}
		if err := filas.Err(); err != nil {
			return err
		}
		return ponerEtiquetas(ctx, tx, out.Clientes)
	})
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusOK, out)
	return nil
}

// ---- Alta, ficha y edición

// limpiarTelefono deja dígitos y un + inicial, igual que la reserva: así el cliente que se da de
// alta a mano es el mismo que después reserva con el teléfono escrito de otra forma.
func limpiarTelefono(t string) string {
	var b strings.Builder
	for i, r := range strings.TrimSpace(t) {
		if (r >= '0' && r <= '9') || (r == '+' && i == 0) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func validarNombre(n string) error {
	if l := utf8.RuneCountInString(n); l < 1 || l > 80 {
		return problema.Invalido("Falta el nombre (hasta 80).", problema.Campo{Ruta: "nombre", Mensaje: "largo inválido"})
	}
	return nil
}

func validarEmail(e string) error {
	if e != "" && (!strings.Contains(e, "@") || len(e) > 120) {
		return problema.Invalido("Revisá el email.", problema.Campo{Ruta: "email", Mensaje: "inválido"})
	}
	return nil
}

// ValidarNacimiento: AAAA-MM-DD, desde 1900 y no en el futuro. "" queda en nil (sin fecha).
func ValidarNacimiento(v *string, ahora time.Time, ruta string) (*string, error) {
	if v == nil || strings.TrimSpace(*v) == "" {
		return nil, nil
	}
	f, err := time.Parse(time.DateOnly, strings.TrimSpace(*v))
	if err != nil || f.Year() < 1900 || f.After(ahora) {
		return nil, problema.Invalido("Revisá la fecha de nacimiento (AAAA-MM-DD).", problema.Campo{Ruta: ruta, Mensaje: "inválida"})
	}
	x := f.Format(time.DateOnly)
	return &x, nil
}

func (s *Servicio) crear(w http.ResponseWriter, r *http.Request) error {
	var in struct {
		Nombre     string  `json:"nombre"`
		Telefono   string  `json:"telefono"`
		Email      string  `json:"email"`
		Nacimiento *string `json:"nacimiento"`
	}
	if err := web.Leer(w, r, &in); err != nil {
		return err
	}
	in.Nombre, in.Telefono, in.Email = strings.TrimSpace(in.Nombre), limpiarTelefono(in.Telefono), strings.ToLower(strings.TrimSpace(in.Email))
	if err := validarNombre(in.Nombre); err != nil {
		return err
	}
	if err := validarEmail(in.Email); err != nil {
		return err
	}
	if len(in.Telefono) > 30 {
		return problema.Invalido("Revisá el teléfono.", problema.Campo{Ruta: "telefono", Mensaje: "inválido"})
	}
	nacimiento, err := ValidarNacimiento(in.Nacimiento, s.ahora(), "nacimiento")
	if err != nil {
		return err
	}
	var c Cliente
	err = s.enNegocio(r, func(ctx context.Context, tx pgx.Tx) error {
		if in.Telefono != "" {
			var otro string
			err := tx.QueryRow(ctx, "SELECT nombre FROM clientes WHERE telefono = $1", in.Telefono).Scan(&otro)
			if err == nil {
				return problema.Conflicto("cliente_existente", "Ya hay un cliente con ese teléfono: "+otro+".")
			}
			if err != pgx.ErrNoRows {
				return err
			}
		}
		var id string
		if err := tx.QueryRow(ctx, `INSERT INTO clientes (negocio_id, nombre, telefono, email, nacimiento)
			VALUES (app.negocio_actual(), $1, $2, $3, $4::date) RETURNING id::text`, in.Nombre, in.Telefono, in.Email, nacimiento).Scan(&id); err != nil {
			return err
		}
		var err error
		c, err = LeerCliente(ctx, tx, id, s.ahora())
		return err
	})
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusCreated, c)
	return nil
}

// TurnoDeCliente: una línea del historial de la ficha.
type TurnoDeCliente struct {
	ID            string    `json:"id"`
	Codigo        string    `json:"codigo"`
	ServicioID    *string   `json:"servicioId"`
	Servicio      string    `json:"servicio"`
	ProfesionalID *string   `json:"profesionalId"`
	Profesional   string    `json:"profesional"`
	Inicio        time.Time `json:"inicio"`
	Fin           time.Time `json:"fin"`
	Precio        int64     `json:"precio"`
	Estado        string    `json:"estado"`
	Origen        string    `json:"origen"`
	Nota          string    `json:"nota"`
}

// Ficha del cliente: el resumen, el historial (lo más nuevo primero, hasta 500) y la penalidad
// vigente si la hay.
type Ficha struct {
	Cliente
	Turnos    []TurnoDeCliente `json:"turnos"`
	Penalidad *Penalidad       `json:"penalidad"`
}

func (s *Servicio) ficha(w http.ResponseWriter, r *http.Request) error {
	var f Ficha
	ahora := s.ahora()
	err := s.enNegocio(r, func(ctx context.Context, tx pgx.Tx) error {
		if _, err := MarcarAusentes(ctx, tx, ahora); err != nil {
			return err
		}
		var err error
		if f.Cliente, err = LeerCliente(ctx, tx, r.PathValue("id"), ahora); err != nil {
			return err
		}
		filas, err := tx.Query(ctx, `SELECT id::text, codigo, servicio_id::text, servicio, profesional_id::text, profesional,
			inicio, fin, precio, estado, origen, nota FROM turnos WHERE cliente_id = $1 ORDER BY inicio DESC LIMIT 500`, f.ID)
		if err != nil {
			return err
		}
		if f.Turnos, err = pgx.CollectRows(filas, pgx.RowToStructByPos[TurnoDeCliente]); err != nil {
			return err
		}
		ps, err := leerPenalidades(ctx, tx, "p.cliente_id = $1 AND p.levantada_en IS NULL AND p.hasta > now() ORDER BY p.hasta DESC LIMIT 1", f.ID)
		if err != nil {
			return err
		}
		if len(ps) > 0 {
			f.Penalidad = &ps[0]
		}
		return nil
	})
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusOK, f)
	return nil
}

// editar el nombre, el email o la fecha de nacimiento ("" la borra). El teléfono no: es lo que
// identifica al cliente cuando reserva.
func (s *Servicio) editar(w http.ResponseWriter, r *http.Request) error {
	var in struct {
		Nombre     *string `json:"nombre"`
		Email      *string `json:"email"`
		Nacimiento *string `json:"nacimiento"`
	}
	if err := web.Leer(w, r, &in); err != nil {
		return err
	}
	if in.Nombre != nil {
		*in.Nombre = strings.TrimSpace(*in.Nombre)
		if err := validarNombre(*in.Nombre); err != nil {
			return err
		}
	}
	if in.Email != nil {
		*in.Email = strings.ToLower(strings.TrimSpace(*in.Email))
		if err := validarEmail(*in.Email); err != nil {
			return err
		}
	}
	nacimiento, err := ValidarNacimiento(in.Nacimiento, s.ahora(), "nacimiento")
	if err != nil {
		return err
	}
	var c Cliente
	err = s.enNegocio(r, func(ctx context.Context, tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `UPDATE clientes SET nombre = coalesce($2, nombre), email = coalesce($3, email),
			nacimiento = CASE WHEN $4 THEN $5::date ELSE nacimiento END WHERE id = $1`,
			r.PathValue("id"), in.Nombre, in.Email, in.Nacimiento != nil, nacimiento)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return problema.NoEncontrado("Ese cliente no existe o no es de este negocio.")
		}
		c, err = LeerCliente(ctx, tx, r.PathValue("id"), s.ahora())
		return err
	})
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusOK, c)
	return nil
}
