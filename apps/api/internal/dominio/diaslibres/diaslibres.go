// Package diaslibres: pedir días libres (propuesta 06 del diseño v2). Un profesional con usuario
// pide desde/hasta y un motivo desde su panel, y ve cuántos turnos tiene esos días. La dueña o
// encargada lo ve en la campanita y en la lista de pedidos: si lo aprueba, el profesional queda
// bloqueado esos días y sus turnos pasan a la lista «para reprogramar», que recepción va resolviendo
// (un turno que se cancela o se mueve sale solo de la lista: app.turno_ocupa).
package diaslibres

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"

	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/avisos"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/espera"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/auditoria"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/db"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/problema"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/sesion"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/web"
)

// Servicio de días libres. Ahora es el reloj (tests); nil: time.Now.
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

// Hasta cuántos días se piden de una vez (como el CHECK de la tabla: hasta - desde <= 60).
const maxDias = 61

var motivos = map[string]string{"vacaciones": "vacaciones", "tramite": "trámite", "salud": "salud", "otro": "otro"}

// Estados de un turno que todavía hay que atender (y por eso mover).
const activos = "('pendiente_pago', 'pendiente', 'confirmado')"

// Rutas del dominio.
func (s *Servicio) Rutas(mux *http.ServeMux) {
	a := func(h web.Manejador) http.HandlerFunc { return s.Sesion.Panel(sesion.Agenda, web.Envolver(h)) }
	g := func(h web.Manejador) http.HandlerFunc { return s.Sesion.Panel(sesion.Gestion, web.Envolver(h)) }
	m := func(h web.Manejador) http.HandlerFunc { return s.Sesion.Panel(avisos.Mostrador, web.Envolver(h)) }
	mux.HandleFunc("GET /v1/panel/dias-libres", a(s.listar))
	mux.HandleFunc("GET /v1/panel/dias-libres/turnos", a(s.turnosEnRango))
	mux.HandleFunc("POST /v1/panel/dias-libres", a(s.pedir))
	mux.HandleFunc("GET /v1/panel/dias-libres/{id}", a(s.ver))
	mux.HandleFunc("POST /v1/panel/dias-libres/{id}/cancelar", a(s.cancelar))
	mux.HandleFunc("POST /v1/panel/dias-libres/{id}/aprobar", g(s.aprobar))
	mux.HandleFunc("POST /v1/panel/dias-libres/{id}/rechazar", g(s.rechazar))
	mux.HandleFunc("GET /v1/panel/reprogramar", m(s.listarReprogramar))
	mux.HandleFunc("POST /v1/panel/reprogramar/{id}/resolver", m(s.resolver))
}

// Pedido de días libres.
type Pedido struct {
	ID            string `json:"id"`
	ProfesionalID string `json:"profesionalId"`
	Profesional   string `json:"profesional"`
	Desde         string `json:"desde"`
	Hasta         string `json:"hasta"`
	Motivo        string `json:"motivo"`
	Nota          string `json:"nota"`
	Estado        string `json:"estado"`
	// Turnos: cuántos tenía esos días al pedirlo; Afectados: cuántos siguen ahí ahora (activos).
	Turnos       int        `json:"turnos"`
	Afectados    int        `json:"afectados"`
	Respuesta    string     `json:"respuesta"`
	RespondidoEn *time.Time `json:"respondidoEn"`
	CreadoEn     time.Time  `json:"creadoEn"`
}

// Rango de un pedido en horas del negocio: [desde 00:00, hasta+1 00:00).
const rangoSQL = `t.inicio >= d.desde::timestamp AT TIME ZONE n.zona_horaria AND t.inicio < (d.hasta + 1)::timestamp AT TIME ZONE n.zona_horaria`

const columnas = `d.id::text, d.profesional_id::text, p.nombre, to_char(d.desde, 'YYYY-MM-DD'), to_char(d.hasta, 'YYYY-MM-DD'),
	d.motivo, d.nota, d.estado, d.turnos,
	(SELECT count(*) FROM turnos t WHERE t.profesional_id = d.profesional_id AND t.estado IN ` + activos + ` AND ` + rangoSQL + `),
	d.respuesta, d.respondido_en, d.creado_en`

const desdeSQL = ` FROM dias_libres d JOIN profesionales p ON p.id = d.profesional_id JOIN negocios n ON n.id = d.negocio_id`

func escanear(f pgx.Row, p *Pedido) error {
	return f.Scan(&p.ID, &p.ProfesionalID, &p.Profesional, &p.Desde, &p.Hasta, &p.Motivo, &p.Nota, &p.Estado, &p.Turnos, &p.Afectados,
		&p.Respuesta, &p.RespondidoEn, &p.CreadoEn)
}

func leer(ctx context.Context, tx pgx.Tx, id string, bloquear bool) (Pedido, error) {
	var p Pedido
	q := "SELECT " + columnas + desdeSQL + " WHERE d.id::text = $1"
	if bloquear {
		q += " FOR UPDATE OF d"
	}
	err := escanear(tx.QueryRow(ctx, q, id), &p)
	return p, err
}

func zona(ctx context.Context, tx pgx.Tx) *time.Location {
	var z string
	_ = tx.QueryRow(ctx, "SELECT zona_horaria FROM negocios WHERE id = app.negocio_actual()").Scan(&z)
	if l, err := time.LoadLocation(z); err == nil && z != "" {
		return l
	}
	l, _ := time.LoadLocation("America/Argentina/Buenos_Aires")
	return l
}

// suyo: el profesional vinculado a la persona de la sesión, o nil.
func suyo(ctx context.Context, tx pgx.Tx) (*string, string, error) {
	var id, nombre string
	err := tx.QueryRow(ctx, `SELECT p.id::text, p.nombre FROM profesionales p JOIN miembros m ON m.id = p.miembro_id
		WHERE m.clerk_id = $1 AND m.activo ORDER BY p.activo DESC, p.creado_en LIMIT 1`, sesion.De(ctx).ClerkID).Scan(&id, &nombre)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, "", nil
	}
	if err != nil {
		return nil, "", err
	}
	return &id, nombre, nil
}

// rangoFechas valida desde/hasta (AAAA-MM-DD, desde hoy, hasta 61 días) y los devuelve como horas del
// negocio: [desde, hasta+1).
func rangoFechas(desdeS, hastaS string, z *time.Location, ahora time.Time) (time.Time, time.Time, error) {
	if hastaS == "" {
		hastaS = desdeS
	}
	d, err1 := time.ParseInLocation(time.DateOnly, desdeS, z)
	h, err2 := time.ParseInLocation(time.DateOnly, hastaS, z)
	hoy := ahora.In(z).Format(time.DateOnly)
	if err1 != nil || err2 != nil || h.Before(d) || h.Sub(d) >= maxDias*24*time.Hour || desdeS < hoy {
		return d, h, problema.Invalido("Elegí desde y hasta qué día (AAAA-MM-DD, desde hoy, hasta 61 días).",
			problema.Campo{Ruta: "desde", Mensaje: "rango inválido"})
	}
	return d, h.AddDate(0, 0, 1), nil
}

// TurnoAfectado: un turno que cae en los días pedidos.
type TurnoAfectado struct {
	ID       string    `json:"id"`
	Codigo   string    `json:"codigo"`
	Inicio   time.Time `json:"inicio"`
	Fin      time.Time `json:"fin"`
	Servicio string    `json:"servicio"`
	Estado   string    `json:"estado"`
	Cliente  string    `json:"cliente"`
	Telefono string    `json:"telefono"`
}

func afectados(ctx context.Context, tx pgx.Tx, profesionalID string, desde, hasta time.Time) ([]TurnoAfectado, error) {
	filas, err := tx.Query(ctx, `SELECT t.id::text, t.codigo, t.inicio, t.fin, t.servicio, t.estado, c.nombre, c.telefono
		FROM turnos t JOIN clientes c ON c.id = t.cliente_id
		WHERE t.profesional_id::text = $1 AND t.estado IN `+activos+` AND t.inicio >= $2 AND t.inicio < $3 ORDER BY t.inicio`,
		profesionalID, desde, hasta)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(filas, pgx.RowToStructByPos[TurnoAfectado])
}

// turnosEnRango: «Ya tenés 3 turnos esos días». Un profesional ve los suyos; el resto pasa
// ?profesional= (sin él, el propio si tiene uno vinculado).
func (s *Servicio) turnosEnRango(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	q := r.URL.Query()
	var out []TurnoAfectado
	err := s.Pool.EnNegocio(ctx, sesion.De(ctx).NegocioID, func(tx pgx.Tx) error {
		propio, _, err := suyo(ctx, tx)
		if err != nil {
			return err
		}
		prof := q.Get("profesional")
		if sesion.De(ctx).Rol == sesion.Profesional || prof == "" {
			if propio == nil {
				return problema.Conflicto("sin_profesional", "Tu usuario no está vinculado a un profesional del equipo.")
			}
			prof = *propio
		}
		z := zona(ctx, tx)
		d, h, err := rangoFechas(q.Get("desde"), q.Get("hasta"), z, s.ahora())
		if err != nil {
			return err
		}
		out, err = afectados(ctx, tx, prof, d, h)
		return err
	})
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusOK, map[string]any{"cantidad": len(out), "turnos": out})
	return nil
}

func rangoTexto(desde, hasta string, z *time.Location) string {
	d, _ := time.ParseInLocation(time.DateOnly, desde, z)
	if desde == hasta {
		return "el " + avisos.Fecha(d)
	}
	h, _ := time.ParseInLocation(time.DateOnly, hasta, z)
	return "del " + avisos.Fecha(d) + " al " + avisos.Fecha(h)
}

// mayuscula: "del sábado…" → "Del sábado…" (empieza siempre con "el" o "del").
func mayuscula(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func cuantos(n int) string {
	if n == 1 {
		return "1 turno"
	}
	return strconv.Itoa(n) + " turnos"
}

// pedir: el profesional vinculado a la sesión pide días libres. Suma a la campanita.
func (s *Servicio) pedir(w http.ResponseWriter, r *http.Request) error {
	var in struct {
		Desde  string `json:"desde"`
		Hasta  string `json:"hasta"`
		Motivo string `json:"motivo"`
		Nota   string `json:"nota"`
	}
	if err := web.Leer(w, r, &in); err != nil {
		return err
	}
	if _, ok := motivos[in.Motivo]; !ok {
		return problema.Invalido("El motivo va como vacaciones, tramite, salud u otro.", problema.Campo{Ruta: "motivo", Mensaje: "inválido"})
	}
	in.Nota = strings.TrimSpace(in.Nota)
	if utf8.RuneCountInString(in.Nota) > 300 {
		return problema.Invalido("La nota es muy larga (hasta 300).", problema.Campo{Ruta: "nota", Mensaje: "muy larga"})
	}
	ctx := r.Context()
	var p Pedido
	err := s.Pool.EnNegocio(ctx, sesion.De(ctx).NegocioID, func(tx pgx.Tx) error {
		prof, nombre, err := suyo(ctx, tx)
		if err != nil {
			return err
		}
		if prof == nil {
			return problema.Conflicto("sin_profesional", "Tu usuario no está vinculado a un profesional del equipo.")
		}
		z := zona(ctx, tx)
		d, h, err := rangoFechas(in.Desde, in.Hasta, z, s.ahora())
		if err != nil {
			return err
		}
		desdeS, hastaS := d.Format(time.DateOnly), h.AddDate(0, 0, -1).Format(time.DateOnly)
		var choca bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM dias_libres WHERE profesional_id::text = $1
			AND estado IN ('pendiente', 'aprobado') AND desde <= $3::date AND hasta >= $2::date)`, *prof, desdeS, hastaS).Scan(&choca); err != nil {
			return err
		}
		if choca {
			return problema.Conflicto("superpuesto", "Ya pediste días libres que se cruzan con esos.")
		}
		ts, err := afectados(ctx, tx, *prof, d, h)
		if err != nil {
			return err
		}
		var id string
		if err := tx.QueryRow(ctx, `INSERT INTO dias_libres (negocio_id, profesional_id, pidio, desde, hasta, motivo, nota, turnos)
			VALUES (app.negocio_actual(), $1, $2, $3, $4, $5, $6, $7) RETURNING id::text`,
			*prof, sesion.De(ctx).ClerkID, desdeS, hastaS, in.Motivo, in.Nota, len(ts)).Scan(&id); err != nil {
			return err
		}
		if p, err = leer(ctx, tx, id, false); err != nil {
			return err
		}
		texto := mayuscula(rangoTexto(desdeS, hastaS, z)) + " · " + motivos[in.Motivo]
		if len(ts) > 0 {
			texto += " · " + cuantos(len(ts)) + " para mover"
		}
		if err := avisos.Notificar(ctx, tx, avisos.Notificacion{Tipo: avisos.NotiEquipo, Titulo: nombre + " pide días libres", Texto: texto + "."}); err != nil {
			return err
		}
		return auditoria.Anotar(ctx, tx, auditoria.Registro{Accion: "dias_libres.pedidos", Entidad: "dias_libres", EntidadID: id,
			Detalle: map[string]any{"desde": desdeS, "hasta": hastaS, "motivo": in.Motivo, "turnos": len(ts)}})
	})
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusCreated, p)
	return nil
}

// listar: los pedidos (del más nuevo al más viejo). Un profesional ve sólo los suyos. Filtro: estado
// (separados por coma).
func (s *Servicio) listar(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	out := []Pedido{}
	err := s.Pool.EnNegocio(ctx, sesion.De(ctx).NegocioID, func(tx pgx.Tx) error {
		cond := []string{"true"}
		var args []any
		if v := r.URL.Query().Get("estado"); v != "" {
			args = append(args, strings.Split(v, ","))
			cond = append(cond, "d.estado = ANY($"+strconv.Itoa(len(args))+")")
		}
		if sesion.De(ctx).Rol == sesion.Profesional {
			args = append(args, sesion.De(ctx).ClerkID)
			cond = append(cond, `d.profesional_id IN (SELECT p.id FROM profesionales p JOIN miembros m ON m.id = p.miembro_id
				WHERE m.clerk_id = $`+strconv.Itoa(len(args))+")")
		}
		filas, err := tx.Query(ctx, "SELECT "+columnas+desdeSQL+" WHERE "+strings.Join(cond, " AND ")+" ORDER BY d.creado_en DESC LIMIT 200", args...)
		if err != nil {
			return err
		}
		defer filas.Close()
		for filas.Next() {
			var p Pedido
			if err := escanear(filas, &p); err != nil {
				return err
			}
			out = append(out, p)
		}
		return filas.Err()
	})
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusOK, map[string]any{"pedidos": out})
	return nil
}

// puedeVer: un profesional sólo los suyos.
func puedeVer(ctx context.Context, tx pgx.Tx, p Pedido) error {
	if sesion.De(ctx).Rol != sesion.Profesional {
		return nil
	}
	propio, _, err := suyo(ctx, tx)
	if err != nil {
		return err
	}
	if propio == nil || *propio != p.ProfesionalID {
		return problema.NoEncontrado("No existe o no es de este negocio.")
	}
	return nil
}

// ver: el pedido con sus turnos afectados.
func (s *Servicio) ver(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	var out struct {
		Pedido
		TurnosAfectados []TurnoAfectado `json:"turnosAfectados"`
	}
	err := s.Pool.EnNegocio(ctx, sesion.De(ctx).NegocioID, func(tx pgx.Tx) (err error) {
		if out.Pedido, err = leer(ctx, tx, r.PathValue("id"), false); err != nil {
			return err
		}
		if err := puedeVer(ctx, tx, out.Pedido); err != nil {
			return err
		}
		d, h := limites(out.Pedido, zona(ctx, tx))
		out.TurnosAfectados, err = afectados(ctx, tx, out.ProfesionalID, d, h)
		return err
	})
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusOK, out)
	return nil
}

func limites(p Pedido, z *time.Location) (time.Time, time.Time) {
	d, _ := time.ParseInLocation(time.DateOnly, p.Desde, z)
	h, _ := time.ParseInLocation(time.DateOnly, p.Hasta, z)
	return d, h.AddDate(0, 0, 1)
}

// responder corre fn con el pedido pendiente bloqueado, y lo deja respondido con ese estado.
func (s *Servicio) responder(r *http.Request, w http.ResponseWriter, estado string, fn func(tx pgx.Tx, p *Pedido) error) (Pedido, error) {
	var in struct {
		Respuesta string `json:"respuesta"`
	}
	if r.ContentLength != 0 {
		if err := web.Leer(w, r, &in); err != nil {
			return Pedido{}, err
		}
	}
	in.Respuesta = strings.TrimSpace(in.Respuesta)
	if utf8.RuneCountInString(in.Respuesta) > 300 {
		return Pedido{}, problema.Invalido("La respuesta es muy larga (hasta 300).", problema.Campo{Ruta: "respuesta", Mensaje: "muy larga"})
	}
	ctx := r.Context()
	var p Pedido
	err := s.Pool.EnNegocio(ctx, sesion.De(ctx).NegocioID, func(tx pgx.Tx) (err error) {
		if p, err = leer(ctx, tx, r.PathValue("id"), true); err != nil {
			return err
		}
		if err := puedeVer(ctx, tx, p); err != nil {
			return err
		}
		if p.Estado != "pendiente" {
			return problema.Conflicto("ya_respondido", "Ese pedido ya se respondió.")
		}
		if _, err := tx.Exec(ctx, `UPDATE dias_libres SET estado = $2, respondio = $3, respuesta = $4, respondido_en = $5 WHERE id::text = $1`,
			p.ID, estado, sesion.De(ctx).ClerkID, in.Respuesta, s.ahora()); err != nil {
			return err
		}
		if err := fn(tx, &p); err != nil {
			return err
		}
		if err := auditoria.Anotar(ctx, tx, auditoria.Registro{Accion: "dias_libres." + estado, Entidad: "dias_libres", EntidadID: p.ID,
			Detalle: map[string]any{"desde": p.Desde, "hasta": p.Hasta, "profesional": p.Profesional}}); err != nil {
			return err
		}
		p, err = leer(ctx, tx, p.ID, false)
		return err
	})
	return p, err
}

// aprobar: bloquea al profesional esos días (los que no estaban ya bloqueados enteros), pasa sus
// turnos a la lista para reprogramar y deja de ofrecer los huecos de esos días a la lista de espera.
func (s *Servicio) aprobar(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	var lista []Reprogramar
	p, err := s.responder(r, w, "aprobado", func(tx pgx.Tx, p *Pedido) error {
		z := zona(ctx, tx)
		if _, err := tx.Exec(ctx, `INSERT INTO bloqueos (negocio_id, profesional_id, fecha, motivo)
			SELECT app.negocio_actual(), $1::uuid, f::date, $4 FROM generate_series($2::date, $3::date, interval '1 day') f
			WHERE NOT EXISTS (SELECT 1 FROM bloqueos b WHERE b.profesional_id = $1::uuid AND b.fecha = f::date AND b.desde IS NULL)`,
			p.ProfesionalID, p.Desde, p.Hasta, "Días libres ("+motivos[p.Motivo]+")"); err != nil {
			return err
		}
		d, h := limites(*p, z)
		if _, err := tx.Exec(ctx, `INSERT INTO reprogramar (negocio_id, turno_id, dia_libre_id)
			SELECT app.negocio_actual(), t.id, $4 FROM turnos t
			WHERE t.profesional_id::text = $1 AND t.estado IN `+activos+` AND t.inicio >= $2 AND t.inicio < $3
			ON CONFLICT (turno_id, dia_libre_id) DO NOTHING`, p.ProfesionalID, d, h, p.ID); err != nil {
			return err
		}
		if err := espera.Caer(ctx, tx, p.ProfesionalID, d, h, s.ahora()); err != nil {
			return err
		}
		var err error
		if lista, err = leerReprogramar(ctx, tx, "r.dia_libre_id::text = $1", p.ID); err != nil {
			return err
		}
		texto := mayuscula(rangoTexto(p.Desde, p.Hasta, z)) + "."
		if len(lista) > 0 {
			texto += " " + cuantos(len(lista)) + " pasan a la lista para reprogramar."
		}
		return avisos.Notificar(ctx, tx, avisos.Notificacion{Tipo: avisos.NotiEquipo, Titulo: "Aprobados los días libres de " + p.Profesional, Texto: texto})
	})
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusOK, map[string]any{"pedido": p, "reprogramar": lista})
	return nil
}

func (s *Servicio) rechazar(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	p, err := s.responder(r, w, "rechazado", func(tx pgx.Tx, p *Pedido) error {
		z := zona(ctx, tx)
		return avisos.Notificar(ctx, tx, avisos.Notificacion{Tipo: avisos.NotiEquipo, Titulo: "No se aprobaron los días libres de " + p.Profesional,
			Texto: mayuscula(rangoTexto(p.Desde, p.Hasta, z)) + "."})
	})
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusOK, p)
	return nil
}

// cancelar: quien lo pidió (o la gestión) lo retira mientras está pendiente.
func (s *Servicio) cancelar(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	if rol := sesion.De(ctx).Rol; rol == sesion.Recepcion {
		return problema.SinPermiso("Sólo quien lo pidió o la gestión del negocio pueden retirarlo.")
	}
	p, err := s.responder(r, w, "cancelado", func(pgx.Tx, *Pedido) error { return nil })
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusOK, p)
	return nil
}

// ---- Turnos para reprogramar

// Reprogramar: un turno que quedó dentro de unos días libres aprobados.
type Reprogramar struct {
	ID          string     `json:"id"`
	PedidoID    string     `json:"pedidoId"`
	Motivo      string     `json:"motivo"`
	ResueltoEn  *time.Time `json:"resueltoEn"`
	ResueltoPor *string    `json:"resueltoPor"`
	CreadoEn    time.Time  `json:"creadoEn"`
	Turno       struct {
		ID            string    `json:"id"`
		Codigo        string    `json:"codigo"`
		Inicio        time.Time `json:"inicio"`
		Fin           time.Time `json:"fin"`
		Servicio      string    `json:"servicio"`
		ProfesionalID *string   `json:"profesionalId"`
		Profesional   string    `json:"profesional"`
		Estado        string    `json:"estado"`
		Cliente       struct {
			Nombre   string `json:"nombre"`
			Telefono string `json:"telefono"`
			Email    string `json:"email"`
		} `json:"cliente"`
	} `json:"turno"`
}

func leerReprogramar(ctx context.Context, tx pgx.Tx, cond string, args ...any) ([]Reprogramar, error) {
	filas, err := tx.Query(ctx, `SELECT r.id::text, r.dia_libre_id::text, d.motivo, r.resuelto_en, r.resuelto_por, r.creado_en,
		t.id::text, t.codigo, t.inicio, t.fin, t.servicio, t.profesional_id::text, t.profesional, t.estado, c.nombre, c.telefono, c.email
		FROM reprogramar r JOIN dias_libres d ON d.id = r.dia_libre_id JOIN turnos t ON t.id = r.turno_id JOIN clientes c ON c.id = t.cliente_id
		WHERE `+cond+` ORDER BY t.inicio LIMIT 500`, args...)
	if err != nil {
		return nil, err
	}
	defer filas.Close()
	out := []Reprogramar{}
	for filas.Next() {
		var x Reprogramar
		t := &x.Turno
		if err := filas.Scan(&x.ID, &x.PedidoID, &x.Motivo, &x.ResueltoEn, &x.ResueltoPor, &x.CreadoEn, &t.ID, &t.Codigo, &t.Inicio, &t.Fin,
			&t.Servicio, &t.ProfesionalID, &t.Profesional, &t.Estado, &t.Cliente.Nombre, &t.Cliente.Telefono, &t.Cliente.Email); err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	return out, filas.Err()
}

// listarReprogramar: los que faltan mover (con ?todos=1, también los resueltos; ?pedido= de un pedido).
func (s *Servicio) listarReprogramar(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	q := r.URL.Query()
	var out []Reprogramar
	err := s.Pool.EnNegocio(ctx, sesion.De(ctx).NegocioID, func(tx pgx.Tx) (err error) {
		cond := []string{"true"}
		var args []any
		if q.Get("todos") != "1" {
			cond = append(cond, "r.resuelto_en IS NULL")
		}
		if v := q.Get("pedido"); v != "" {
			args = append(args, v)
			cond = append(cond, "r.dia_libre_id::text = $"+strconv.Itoa(len(args)))
		}
		out, err = leerReprogramar(ctx, tx, strings.Join(cond, " AND "), args...)
		return err
	})
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusOK, map[string]any{"turnos": out})
	return nil
}

// resolver: recepción lo marca resuelto (lo movió, lo habló con el cliente).
func (s *Servicio) resolver(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	var out []Reprogramar
	err := s.Pool.EnNegocio(ctx, sesion.De(ctx).NegocioID, func(tx pgx.Tx) (err error) {
		tag, err := tx.Exec(ctx, `UPDATE reprogramar SET resuelto_en = coalesce(resuelto_en, $2), resuelto_por = coalesce(resuelto_por, $3)
			WHERE id::text = $1`, r.PathValue("id"), s.ahora(), sesion.De(ctx).ClerkID)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return pgx.ErrNoRows
		}
		if out, err = leerReprogramar(ctx, tx, "r.id::text = $1", r.PathValue("id")); err != nil {
			return err
		}
		return auditoria.Anotar(ctx, tx, auditoria.Registro{Accion: "reprogramar.resuelto", Entidad: "reprogramar", EntidadID: r.PathValue("id")})
	})
	if err != nil {
		return err
	}
	if len(out) == 0 {
		return problema.NoEncontrado("No existe o no es de este negocio.")
	}
	web.JSON(w, http.StatusOK, out[0])
	return nil
}
