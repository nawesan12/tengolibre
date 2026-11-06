package turnos

import (
	"context"
	"errors"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"

	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/catalogo"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/disponibilidad"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/miweb"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/negocios"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/resenas"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/problema"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/sesion"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/web"
)

// Rutas del dominio.
func (s *Servicio) Rutas(mux *http.ServeMux) {
	// Página pública del negocio (sin sesión).
	mux.HandleFunc("GET /v1/p/{slug}", web.Envolver(s.pagina))
	mux.HandleFunc("GET /v1/p/{slug}/libres", web.Envolver(s.libresPublico))
	mux.HandleFunc("GET /v1/p/{slug}/dias", web.Envolver(s.diasPublico))
	mux.HandleFunc("POST /v1/p/{slug}/turnos", s.Sesion.Opcional(web.Envolver(s.reservarPublico)))
	// Link del cliente (el código es el secreto).
	mux.HandleFunc("GET /v1/t/{codigo}", web.Envolver(s.verTurno))
	mux.HandleFunc("POST /v1/t/{codigo}/dispositivo", web.Envolver(s.seguirTurno))
	mux.HandleFunc("DELETE /v1/t/{codigo}/dispositivo", web.Envolver(s.dejarDeSeguir))
	mux.HandleFunc("POST /v1/t/{codigo}/cancelar", web.Envolver(s.cancelarCliente))
	mux.HandleFunc("POST /v1/t/{codigo}/reprogramar", web.Envolver(s.reprogramarCliente))
	mux.HandleFunc("GET /v1/t/{codigo}/repetir", web.Envolver(s.verRepetir))
	mux.HandleFunc("POST /v1/t/{codigo}/repetir", web.Envolver(s.reservarRepetir))
	// Agenda del panel.
	a := func(h web.Manejador) http.HandlerFunc { return s.Sesion.Panel(sesion.Agenda, web.Envolver(h)) }
	mux.HandleFunc("GET /v1/panel/turnos", a(s.agenda))
	mux.HandleFunc("POST /v1/panel/turnos", a(s.reservarPanel))
	mux.HandleFunc("GET /v1/panel/turnos/{id}", a(s.verPanel))
	mux.HandleFunc("PATCH /v1/panel/turnos/{id}", a(s.cambiar))
	mux.HandleFunc("POST /v1/panel/turnos/{id}/reprogramar", a(s.reprogramarPanel))
	mux.HandleFunc("GET /v1/panel/libres", a(s.libresPanel))
	// Turno fijo (serie.go).
	mux.HandleFunc("POST /v1/panel/series/previa", a(s.previaSerie))
	mux.HandleFunc("POST /v1/panel/series", a(s.reservarSerie))
	mux.HandleFunc("GET /v1/panel/series/{id}", a(s.verSerie))
	mux.HandleFunc("POST /v1/panel/turnos/{id}/cancelar-siguientes", a(s.cancelarSiguientes))
}

// ProfesionalPublico: lo que la página muestra de cada profesional (sin email ni teléfono).
type ProfesionalPublico struct {
	ID           string   `json:"id"`
	Nombre       string   `json:"nombre"`
	Especialidad string   `json:"especialidad"`
	Bio          string   `json:"bio"`
	Foto         *string  `json:"foto"`
	Servicios    []string `json:"servicios"`
}

func (s *Servicio) pagina(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	id, err := s.negocioPublico(ctx, r.PathValue("slug"))
	if err != nil {
		return err
	}
	var out struct {
		Negocio             negocios.Negocio      `json:"negocio"`
		Categorias          []catalogo.Categoria  `json:"categorias"`
		Servicios           []catalogo.Prestacion `json:"servicios"`
		Profesionales       []ProfesionalPublico  `json:"profesionales"`
		DiasVista           int                   `json:"diasVista"`
		PoliticaCancelacion string                `json:"politicaCancelacion"`
		Web                 miweb.Web             `json:"web"`
		Galeria             []miweb.Foto          `json:"galeria"`
		Resenas             resenas.Resumen       `json:"resenas"`
		// HorarioAtencion: cuándo abre el negocio, la unión de los horarios de sus profesionales activos.
		HorarioAtencion []disponibilidad.Franja `json:"horarioAtencion"`
	}
	err = s.Pool.EnNegocio(ctx, id, func(tx pgx.Tx) error {
		var err error
		if out.Negocio, err = negocios.Leer(ctx, tx); err != nil {
			return err
		}
		if out.Categorias, err = catalogo.Categorias(ctx, tx); err != nil {
			return err
		}
		if out.Servicios, err = catalogo.Servicios(ctx, tx, true); err != nil {
			return err
		}
		ps, err := catalogo.Profesionales(ctx, tx, true)
		if err != nil {
			return err
		}
		out.Profesionales = []ProfesionalPublico{}
		out.HorarioAtencion = HorarioAtencion(ps)
		for _, p := range ps {
			out.Profesionales = append(out.Profesionales, ProfesionalPublico{p.ID, p.Nombre, p.Especialidad, p.Bio, p.Foto, p.Servicios})
		}
		a, err := negocios.LeerAjustes(ctx, tx)
		if err != nil {
			return err
		}
		out.DiasVista, out.PoliticaCancelacion = a.DiasVista, a.PoliticaCancelacion
		if out.Web, err = miweb.Leer(ctx, tx); err != nil {
			return err
		}
		if out.Galeria, err = miweb.Galeria(ctx, tx); err != nil {
			return err
		}
		out.Resenas, err = resenas.Promedio(ctx, tx)
		return err
	})
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusOK, out)
	return nil
}

func (s *Servicio) libresEn(ctx context.Context, negocioID string, r *http.Request) ([]HorarioLibre, error) {
	q := r.URL.Query()
	if q.Get("servicio") == "" || q.Get("fecha") == "" {
		return nil, problema.Invalido("Faltan servicio y fecha.")
	}
	var out []HorarioLibre
	err := s.Pool.EnNegocio(ctx, negocioID, func(tx pgx.Tx) error {
		c, err := cargarContexto(ctx, tx)
		if err != nil {
			return err
		}
		out, err = libres(ctx, tx, c, q.Get("servicio"), q.Get("profesional"), q.Get("fecha"), s.ahora())
		return err
	})
	return out, err
}

func (s *Servicio) libresPublico(w http.ResponseWriter, r *http.Request) error {
	id, err := s.negocioPublico(r.Context(), r.PathValue("slug"))
	if err != nil {
		return err
	}
	hs, err := s.libresEn(r.Context(), id, r)
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusOK, map[string]any{"horarios": hs})
	return nil
}

func (s *Servicio) libresPanel(w http.ResponseWriter, r *http.Request) error {
	hs, err := s.libresEn(r.Context(), sesion.De(r.Context()).NegocioID, r)
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusOK, map[string]any{"horarios": hs})
	return nil
}

// Dia del calendario: si tiene algún horario libre.
type Dia struct {
	Fecha  string `json:"fecha"`
	Libres int    `json:"libres"`
}

// diasPublico: para el calendario, cuántos horarios libres tiene cada día de la vista (o de los
// `dias` pedidos, hasta 62, desde `desde`).
func (s *Servicio) diasPublico(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	id, err := s.negocioPublico(ctx, r.PathValue("slug"))
	if err != nil {
		return err
	}
	q := r.URL.Query()
	if q.Get("servicio") == "" {
		return problema.Invalido("Falta el servicio.")
	}
	ahora := s.ahora()
	out := []Dia{}
	err = s.Pool.EnNegocio(ctx, id, func(tx pgx.Tx) error {
		c, err := cargarContexto(ctx, tx)
		if err != nil {
			return err
		}
		desde := ahora.In(c.zona)
		if v := q.Get("desde"); v != "" {
			if desde, err = time.ParseInLocation(time.DateOnly, v, c.zona); err != nil {
				return problema.Invalido("desde va como AAAA-MM-DD.")
			}
		}
		n := c.ajustes.DiasVista
		if v, err := strconv.Atoi(q.Get("dias")); err == nil && v > 0 {
			n = min(v, 62)
		}
		for i := 0; i < n; i++ {
			fecha := desde.AddDate(0, 0, i).Format(time.DateOnly)
			hs, err := libres(ctx, tx, c, q.Get("servicio"), q.Get("profesional"), fecha, ahora)
			if err != nil {
				return err
			}
			out = append(out, Dia{Fecha: fecha, Libres: len(hs)})
		}
		return nil
	})
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusOK, map[string]any{"dias": out})
	return nil
}

func (s *Servicio) reservarPublico(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	id, err := s.negocioPublico(ctx, r.PathValue("slug"))
	if err != nil {
		return err
	}
	var in Pedido
	if err := web.Leer(w, r, &in); err != nil {
		return err
	}
	in.Forzar = false
	in.ClienteID = ""
	email := sesion.De(ctx).Persona.Email
	if in.UsarMembresia && email == "" {
		return problema.SinSesion()
	}
	var t Turno
	err = s.Pool.EnNegocio(ctx, id, func(tx pgx.Tx) (err error) {
		// Lo que se cargue desde acá no es de confianza (ver app.cuidar_email).
		if _, err := tx.Exec(ctx, "SELECT set_config('app.origen', 'web', true)"); err != nil {
			return err
		}
		if in.UsarMembresia {
			if in.ClienteID, err = SocioDe(ctx, tx, email, s.ahora()); err != nil {
				return err
			}
		}
		if t, err = reservar(ctx, tx, in, "web", s.ahora()); err != nil {
			return err
		}
		return s.alReservar(ctx, tx, &t, in, "web")
	})
	if err != nil {
		return err
	}
	s.despuesDeReservar(ctx, id, &t)
	s.avisar(ctx, id, t.ID, "reservado")
	web.JSON(w, http.StatusCreated, t)
	return nil
}

// SocioDe es el cliente del negocio fijado con ese email (de confianza) y una membresía vigente.
func SocioDe(ctx context.Context, tx pgx.Tx, email string, ahora time.Time) (string, error) {
	var id string
	err := tx.QueryRow(ctx, `SELECT c.id::text FROM clientes c JOIN membresias m ON m.cliente_id = c.id
		WHERE c.email_confiable AND lower(c.email) = lower($1) AND m.estado = 'activa' AND m.desde <= $2 AND m.hasta > $2
		ORDER BY m.hasta DESC LIMIT 1`, email, ahora).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", problema.Conflicto("sin_membresia", "No encontramos una membresía vigente con tu email en este negocio.")
	}
	return id, err
}

func (s *Servicio) reservarPanel(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	var in Pedido
	if err := web.Leer(w, r, &in); err != nil {
		return err
	}
	negocio := sesion.De(ctx).NegocioID
	var t Turno
	err := s.Pool.EnNegocio(ctx, negocio, func(tx pgx.Tx) (err error) {
		if t, err = reservar(ctx, tx, in, "panel", s.ahora()); err != nil {
			return err
		}
		return s.alReservar(ctx, tx, &t, in, "panel")
	})
	if err != nil {
		return err
	}
	s.despuesDeReservar(ctx, negocio, &t)
	s.avisar(ctx, negocio, t.ID, "reservado")
	web.JSON(w, http.StatusCreated, t)
	return nil
}

// TurnoCliente: lo que ve quien tiene el link.
type TurnoCliente struct {
	Turno
	Negocio            negocios.Negocio `json:"negocio"`
	PuedeCancelar      bool             `json:"puedeCancelar"`
	CancelarHastaHoras int              `json:"cancelarHastaHoras"`
	// Sena: lo cobrado por adelantado (null: no hubo cobro).
	Sena *Sena `json:"sena"`
	// Referencia: código corto para decir en el local (el código del link es el secreto).
	Referencia   string `json:"referencia"`
	ConMembresia bool   `json:"conMembresia"`
}

// Sena del turno como la ve el cliente.
type Sena struct {
	Monto int64 `json:"monto"`
	// Estado: aprobado, pendiente, devuelto o rechazado.
	Estado string `json:"estado"`
}

// estadosSena: del estado del pago a lo que ve el cliente.
var estadosSena = map[string]string{"pendiente": "pendiente", "en_proceso": "pendiente", "aprobado": "aprobado",
	"rechazado": "rechazado", "cancelado": "rechazado", "devuelto": "devuelto"}

// Referencia corta de un turno: los primeros 6 caracteres del id, en mayúsculas. El id no es secreto
// (el panel lo usa), así que no abre el link del cliente.
func Referencia(id string) string {
	if len(id) < 6 {
		return strings.ToUpper(id)
	}
	return strings.ToUpper(id[:6])
}

// HorarioAtencion une los horarios semanales de los profesionales (activos): por día, las franjas
// que se pisan o se tocan quedan en una sola. Ordenado por día y hora.
func HorarioAtencion(ps []catalogo.Profesional) []disponibilidad.Franja {
	var todas []disponibilidad.Franja
	for _, p := range ps {
		if p.Activo {
			todas = append(todas, p.Horario...)
		}
	}
	sort.Slice(todas, func(i, j int) bool {
		if todas[i].Dia != todas[j].Dia {
			return todas[i].Dia < todas[j].Dia
		}
		return todas[i].Desde < todas[j].Desde
	})
	out := []disponibilidad.Franja{}
	for _, f := range todas {
		f.Desde, f.Hasta = hhmmDe(f.Desde), hhmmDe(f.Hasta)
		if n := len(out); n > 0 && out[n-1].Dia == f.Dia && f.Desde <= out[n-1].Hasta {
			if f.Hasta > out[n-1].Hasta {
				out[n-1].Hasta = f.Hasta
			}
			continue
		}
		out = append(out, f)
	}
	return out
}

// hhmmDe: "HH:MM:SS" (como puede venir de Postgres) a "HH:MM".
func hhmmDe(s string) string {
	if len(s) > 5 {
		return s[:5]
	}
	return s
}

// delCliente corre fn con el turno del link bloqueado.
func (s *Servicio) delCliente(ctx context.Context, cod string, fn func(tx pgx.Tx, t *TurnoCliente) error) (TurnoCliente, string, error) {
	var out TurnoCliente
	negocio, err := s.negocioDelTurno(ctx, cod)
	if err != nil {
		return out, "", err
	}
	err = s.Pool.EnNegocio(ctx, negocio, func(tx pgx.Tx) error {
		var err error
		if out.Turno, err = leerTurno(ctx, tx, "t.codigo = $1", cod, true); err != nil {
			return err
		}
		if out.Negocio, err = negocios.Leer(ctx, tx); err != nil {
			return err
		}
		a, err := negocios.LeerAjustes(ctx, tx)
		if err != nil {
			return err
		}
		out.CancelarHastaHoras = a.CancelarHastaHoras
		if err := fn(tx, &out); err != nil {
			return err
		}
		out.Referencia = Referencia(out.ID)
		var monto *int64
		var estado *string
		if err := tx.QueryRow(ctx, `SELECT t.membresia_id IS NOT NULL, p.monto, p.estado FROM turnos t
			LEFT JOIN pagos p ON p.turno_id = t.id WHERE t.id = $1`, out.ID).Scan(&out.ConMembresia, &monto, &estado); err != nil {
			return err
		}
		out.Sena = nil
		if monto != nil && estado != nil {
			out.Sena = &Sena{Monto: *monto, Estado: estadosSena[*estado]}
		}
		out.PuedeCancelar = (out.Estado == "pendiente" || out.Estado == "confirmado" || out.Estado == "pendiente_pago") &&
			s.ahora().Add(time.Duration(a.CancelarHastaHoras)*time.Hour).Before(out.Inicio)
		return nil
	})
	out.Cliente.ID = ""
	out.LlamarDesde, out.LlamarMotivo = nil, nil
	return out, negocio, err
}

func (s *Servicio) verTurno(w http.ResponseWriter, r *http.Request) error {
	t, _, err := s.delCliente(r.Context(), r.PathValue("codigo"), func(pgx.Tx, *TurnoCliente) error { return nil })
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusOK, t)
	return nil
}

func (s *Servicio) cancelarCliente(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	t, negocio, err := s.delCliente(ctx, r.PathValue("codigo"), func(tx pgx.Tx, t *TurnoCliente) error {
		if t.Estado == "cancelado" {
			return nil
		}
		if t.Estado != "pendiente" && t.Estado != "confirmado" && t.Estado != "pendiente_pago" {
			return problema.Conflicto("estado", "Este turno ya no se puede cancelar.")
		}
		if !s.ahora().Add(time.Duration(t.CancelarHastaHoras) * time.Hour).Before(t.Inicio) {
			return problema.Conflicto("tarde", "Ya pasó el plazo para cancelar online. Escribile al negocio.")
		}
		if _, err := tx.Exec(ctx, "UPDATE turnos SET estado = 'cancelado', actualizado_en = now() WHERE id = $1", t.ID); err != nil {
			return err
		}
		t.Estado = "cancelado"
		return s.alCancelar(ctx, tx, t.ID)
	})
	if err != nil {
		return err
	}
	s.avisar(ctx, negocio, t.ID, "cancelado_cliente")
	web.JSON(w, http.StatusOK, t)
	return nil
}

func (s *Servicio) reprogramarCliente(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	var in struct {
		Inicio        time.Time `json:"inicio"`
		ProfesionalID string    `json:"profesionalId"`
	}
	if err := web.Leer(w, r, &in); err != nil {
		return err
	}
	t, negocio, err := s.delCliente(ctx, r.PathValue("codigo"), func(tx pgx.Tx, t *TurnoCliente) error {
		if !s.ahora().Add(time.Duration(t.CancelarHastaHoras) * time.Hour).Before(t.Inicio) {
			return problema.Conflicto("tarde", "Ya pasó el plazo para cambiar el turno online. Escribile al negocio.")
		}
		nuevo, err := reprogramar(ctx, tx, t.Turno, in.Inicio, in.ProfesionalID, s.ahora(), false)
		if err != nil {
			return err
		}
		t.Turno = nuevo
		return nil
	})
	if err != nil {
		return err
	}
	s.avisar(ctx, negocio, t.ID, "reprogramado")
	web.JSON(w, http.StatusOK, t)
	return nil
}

// agenda del panel: turnos entre desde y hasta (fechas, inclusive; por defecto hoy), opcionalmente
// de un profesional y de un estado. Un profesional (rol) ve sólo los suyos.
func (s *Servicio) agenda(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	ses := sesion.De(ctx)
	q := r.URL.Query()
	out := []Turno{}
	err := s.Pool.EnNegocio(ctx, ses.NegocioID, func(tx pgx.Tx) error {
		c, err := cargarContexto(ctx, tx)
		if err != nil {
			return err
		}
		hoy := s.ahora().In(c.zona).Format(time.DateOnly)
		desdeS, hastaS := q.Get("desde"), q.Get("hasta")
		if desdeS == "" {
			desdeS = hoy
		}
		if hastaS == "" {
			hastaS = desdeS
		}
		desde, err1 := time.ParseInLocation(time.DateOnly, desdeS, c.zona)
		hasta, err2 := time.ParseInLocation(time.DateOnly, hastaS, c.zona)
		if err1 != nil || err2 != nil || hasta.Before(desde) || hasta.Sub(desde) > 92*24*time.Hour {
			return problema.Invalido("desde y hasta van como AAAA-MM-DD, en orden y con hasta 92 días.")
		}
		cond := []string{"t.inicio >= $1", "t.inicio < $2"}
		args := []any{desde, hasta.AddDate(0, 0, 1)}
		if v := q.Get("profesional"); v != "" {
			args = append(args, v)
			cond = append(cond, "t.profesional_id = $"+strconv.Itoa(len(args)))
		}
		if v := q.Get("estado"); v != "" {
			args = append(args, strings.Split(v, ","))
			cond = append(cond, "t.estado = ANY($"+strconv.Itoa(len(args))+")")
		}
		if v := q.Get("cliente"); v != "" {
			args = append(args, v)
			cond = append(cond, "t.cliente_id = $"+strconv.Itoa(len(args)))
		}
		if ses.Rol == sesion.Profesional {
			args = append(args, ses.ClerkID)
			cond = append(cond, `t.profesional_id IN (SELECT p.id FROM profesionales p JOIN miembros m ON m.id = p.miembro_id
				WHERE m.clerk_id = $`+strconv.Itoa(len(args))+")")
		}
		filas, err := tx.Query(ctx, "SELECT "+columnasTurno+desdeTurnos+" WHERE "+strings.Join(cond, " AND ")+" ORDER BY t.inicio", args...)
		if err != nil {
			return err
		}
		defer filas.Close()
		for filas.Next() {
			var t Turno
			if err := escanearTurno(filas, &t); err != nil {
				return err
			}
			out = append(out, t)
		}
		return filas.Err()
	})
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusOK, map[string]any{"turnos": out})
	return nil
}

func (s *Servicio) verPanel(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	var t Turno
	err := s.Pool.EnNegocio(ctx, sesion.De(ctx).NegocioID, func(tx pgx.Tx) (err error) {
		t, err = leerTurno(ctx, tx, "t.id = $1", r.PathValue("id"), false)
		return err
	})
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusOK, t)
	return nil
}

func (s *Servicio) cambiar(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	var in struct {
		Estado *string `json:"estado"`
		Nota   *string `json:"nota"`
	}
	if err := web.Leer(w, r, &in); err != nil {
		return err
	}
	if in.Nota != nil && utf8.RuneCountInString(*in.Nota) > 500 {
		return problema.Invalido("La nota es muy larga (hasta 500).")
	}
	negocio := sesion.De(ctx).NegocioID
	var t Turno
	var antes string
	err := s.Pool.EnNegocio(ctx, negocio, func(tx pgx.Tx) error {
		var err error
		if t, err = leerTurno(ctx, tx, "t.id = $1", r.PathValue("id"), true); err != nil {
			return err
		}
		antes = t.Estado
		if in.Estado != nil && !PuedePasar(t.Estado, *in.Estado) {
			return problema.Conflicto("estado", "No se puede pasar de "+t.Estado+" a "+*in.Estado+".")
		}
		if _, err := tx.Exec(ctx, `UPDATE turnos SET estado = coalesce($2, estado), nota = coalesce($3, nota),
			vence_en = CASE WHEN coalesce($2, estado) = 'pendiente_pago' THEN vence_en END, actualizado_en = now() WHERE id = $1`,
			t.ID, in.Estado, in.Nota); err != nil {
			return err
		}
		if in.Estado != nil && *in.Estado == "cancelado" && antes != "cancelado" {
			if err := s.alCancelar(ctx, tx, t.ID); err != nil {
				return err
			}
		}
		t, err = leerTurno(ctx, tx, "t.id = $1", t.ID, false)
		return err
	})
	if err != nil {
		return err
	}
	if t.Estado != antes {
		s.avisar(ctx, negocio, t.ID, t.Estado)
	}
	web.JSON(w, http.StatusOK, t)
	return nil
}

func (s *Servicio) reprogramarPanel(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	var in struct {
		Inicio        time.Time `json:"inicio"`
		ProfesionalID string    `json:"profesionalId"`
		Forzar        bool      `json:"forzar"`
	}
	if err := web.Leer(w, r, &in); err != nil {
		return err
	}
	negocio := sesion.De(ctx).NegocioID
	var t Turno
	err := s.Pool.EnNegocio(ctx, negocio, func(tx pgx.Tx) error {
		actual, err := leerTurno(ctx, tx, "t.id = $1", r.PathValue("id"), true)
		if err != nil {
			return err
		}
		t, err = reprogramar(ctx, tx, actual, in.Inicio, in.ProfesionalID, s.ahora(), in.Forzar)
		return err
	})
	if err != nil {
		return err
	}
	s.avisar(ctx, negocio, t.ID, "reprogramado")
	web.JSON(w, http.StatusOK, t)
	return nil
}

var (
	tokenIPhone  = regexp.MustCompile(`^[0-9a-f]{64,200}$`)
	tokenAndroid = regexp.MustCompile(`^[A-Za-z0-9_:-]{20,}$`)
)

// tokenDe normaliza el token de un teléfono: el de APNs (hex) en minúsculas; el de FCM tal cual
// (distingue mayúsculas).
func tokenDe(t string) string {
	t = strings.TrimSpace(t)
	if tokenIPhone.MatchString(strings.ToLower(t)) {
		return strings.ToLower(t)
	}
	return t
}

// seguirTurno: el teléfono de la clienta (la app en modo clienta) se anota para que le lleguen los
// avisos de este turno: {"token": hex de APNs, "entorno": "sandbox|produccion"} desde el iPhone, o
// {"plataforma": "android", "token": <FCM>} desde Android. Tener el código alcanza (es el mismo link
// que recibe por mail).
func (s *Servicio) seguirTurno(w http.ResponseWriter, r *http.Request) error {
	var in struct {
		Plataforma string `json:"plataforma"`
		Token      string `json:"token"`
		Entorno    string `json:"entorno"`
	}
	if err := web.Leer(w, r, &in); err != nil {
		return err
	}
	in.Token = strings.TrimSpace(in.Token)
	if in.Plataforma == "android" {
		if len(in.Token) > 4096 || !tokenAndroid.MatchString(in.Token) {
			return problema.Invalido("El token del teléfono no es válido.", problema.Campo{Ruta: "token", Mensaje: "formato"})
		}
	} else {
		in.Plataforma = "ios"
		in.Token = strings.ToLower(in.Token)
		if !tokenIPhone.MatchString(in.Token) {
			return problema.Invalido("El token del iPhone no es válido.", problema.Campo{Ruta: "token", Mensaje: "hexadecimal"})
		}
	}
	if in.Entorno != "sandbox" {
		in.Entorno = "produccion"
	}
	ctx := r.Context()
	_, _, err := s.delCliente(ctx, r.PathValue("codigo"), func(tx pgx.Tx, t *TurnoCliente) error {
		_, err := tx.Exec(ctx, `INSERT INTO turno_dispositivos (negocio_id, turno_id, token, entorno, plataforma)
			VALUES (app.negocio_actual(), $1, $2, $3, $4) ON CONFLICT (turno_id, token) DO UPDATE SET entorno = excluded.entorno`,
			t.ID, in.Token, in.Entorno, in.Plataforma)
		return err
	})
	if err != nil {
		return err
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}

// dejarDeSeguir: {"token": …} deja de mandarle los avisos de este turno a ese teléfono.
func (s *Servicio) dejarDeSeguir(w http.ResponseWriter, r *http.Request) error {
	var in struct {
		Token string `json:"token"`
	}
	if err := web.Leer(w, r, &in); err != nil {
		return err
	}
	ctx := r.Context()
	_, _, err := s.delCliente(ctx, r.PathValue("codigo"), func(tx pgx.Tx, t *TurnoCliente) error {
		_, err := tx.Exec(ctx, "DELETE FROM turno_dispositivos WHERE turno_id = $1 AND token = $2", t.ID, tokenDe(in.Token))
		return err
	})
	if err != nil {
		return err
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}
