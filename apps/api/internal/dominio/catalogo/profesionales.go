package catalogo

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/disponibilidad"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/problema"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/sesion"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/web"
)

// Las horas salen de Postgres como time; to_char las deja en "HH24:MI".
const columnasProfesional = `p.id::text, p.miembro_id::text, p.nombre, p.especialidad, p.bio, p.foto, p.email, p.telefono, p.orden, p.activo,
	coalesce((SELECT array_agg(ps.servicio_id::text ORDER BY ps.servicio_id) FROM profesional_servicios ps WHERE ps.profesional_id = p.id), '{}'),
	coalesce((SELECT jsonb_agg(jsonb_build_object('dia', h.dia, 'desde', to_char(h.desde, 'HH24:MI'), 'hasta', to_char(h.hasta, 'HH24:MI'))
		ORDER BY h.dia, h.desde) FROM horarios h WHERE h.profesional_id = p.id), '[]')`

func escanearProfesional(f pgx.Row, p *Profesional) error {
	return f.Scan(&p.ID, &p.MiembroID, &p.Nombre, &p.Especialidad, &p.Bio, &p.Foto, &p.Email, &p.Telefono, &p.Orden, &p.Activo,
		&p.Servicios, &p.Horario)
}

// Profesionales del negocio fijado; soloActivos para la página pública.
func Profesionales(ctx context.Context, tx pgx.Tx, soloActivos bool) ([]Profesional, error) {
	q := "SELECT " + columnasProfesional + " FROM profesionales p"
	if soloActivos {
		q += " WHERE p.activo"
	}
	filas, err := tx.Query(ctx, q+" ORDER BY p.orden, p.nombre")
	if err != nil {
		return nil, err
	}
	defer filas.Close()
	out := []Profesional{}
	for filas.Next() {
		var p Profesional
		if err := escanearProfesional(filas, &p); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, filas.Err()
}

// LeerProfesional por id en el negocio fijado. paraReservar lo bloquea (FOR UPDATE) hasta el final
// de la transacción: dos reservas al mismo profesional se ponen en fila.
func LeerProfesional(ctx context.Context, tx pgx.Tx, id string, paraReservar bool) (Profesional, error) {
	var p Profesional
	q := "SELECT " + columnasProfesional + " FROM profesionales p WHERE p.id = $1"
	if paraReservar {
		q += " FOR UPDATE OF p"
	}
	err := escanearProfesional(tx.QueryRow(ctx, q, id), &p)
	return p, err
}

// Hace dice si el profesional hace el servicio.
func (p Profesional) Hace(servicioID string) bool {
	for _, s := range p.Servicios {
		if s == servicioID {
			return true
		}
	}
	return false
}

func (s *Servicio) profesionales(w http.ResponseWriter, r *http.Request) error {
	var ps []Profesional
	err := s.enNegocio(r, func(ctx context.Context, tx pgx.Tx) (err error) {
		ps, err = Profesionales(ctx, tx, false)
		return err
	})
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusOK, map[string]any{"profesionales": ps})
	return nil
}

func validarProfesional(p *Profesional) error {
	p.Nombre = strings.TrimSpace(p.Nombre)
	if err := largo("nombre", p.Nombre, 2, 60); err != nil {
		return err
	}
	if p.Foto != nil && *p.Foto == "" {
		p.Foto = nil
	}
	if p.MiembroID != nil && *p.MiembroID == "" {
		p.MiembroID = nil
	}
	return nil
}

func (s *Servicio) crearProfesional(w http.ResponseWriter, r *http.Request) error {
	p := Profesional{Activo: true}
	if err := web.Leer(w, r, &p); err != nil {
		return err
	}
	if err := validarProfesional(&p); err != nil {
		return err
	}
	err := s.enNegocio(r, func(ctx context.Context, tx pgx.Tx) error {
		err := tx.QueryRow(ctx, `INSERT INTO profesionales (negocio_id, miembro_id, nombre, especialidad, bio, foto, email, telefono, orden, activo)
			VALUES (app.negocio_actual(), $1, $2, $3, $4, $5, $6, $7, $8, $9) RETURNING id::text`,
			p.MiembroID, p.Nombre, p.Especialidad, p.Bio, p.Foto, p.Email, p.Telefono, p.Orden, p.Activo).Scan(&p.ID)
		if err != nil {
			return err
		}
		if err := ponerServicios(ctx, tx, p.ID, p.Servicios); err != nil {
			return err
		}
		if err := ponerHorario(ctx, tx, p.ID, p.Horario); err != nil {
			return err
		}
		p, err = LeerProfesional(ctx, tx, p.ID, false)
		return err
	})
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusCreated, p)
	return nil
}

func (s *Servicio) editarProfesional(w http.ResponseWriter, r *http.Request) error {
	var p Profesional
	err := s.enNegocio(r, func(ctx context.Context, tx pgx.Tx) error {
		var err error
		if p, err = LeerProfesional(ctx, tx, r.PathValue("id"), false); err != nil {
			return err
		}
		id, servicios, horario := p.ID, p.Servicios, p.Horario
		if err := web.Leer(w, r, &p); err != nil {
			return err
		}
		// Servicios y horario tienen su propio PUT.
		p.ID, p.Servicios, p.Horario = id, servicios, horario
		if err := validarProfesional(&p); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `UPDATE profesionales SET miembro_id = $2, nombre = $3, especialidad = $4, bio = $5, foto = $6,
			email = $7, telefono = $8, orden = $9, activo = $10 WHERE id = $1`,
			p.ID, p.MiembroID, p.Nombre, p.Especialidad, p.Bio, p.Foto, p.Email, p.Telefono, p.Orden, p.Activo); err != nil {
			return err
		}
		p, err = LeerProfesional(ctx, tx, p.ID, false)
		return err
	})
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusOK, p)
	return nil
}

func ponerServicios(ctx context.Context, tx pgx.Tx, id string, servicios []string) error {
	if _, err := tx.Exec(ctx, "DELETE FROM profesional_servicios WHERE profesional_id = $1", id); err != nil {
		return err
	}
	if len(servicios) == 0 {
		return nil
	}
	_, err := tx.Exec(ctx, `INSERT INTO profesional_servicios (negocio_id, profesional_id, servicio_id)
		SELECT app.negocio_actual(), $1, s FROM unnest($2::uuid[]) s ON CONFLICT DO NOTHING`, id, servicios)
	return err
}

// validarHorario: horas bien formadas y franjas del mismo día que no se pisan.
func validarHorario(hs []disponibilidad.Franja) error {
	for i, f := range hs {
		if f.Dia < 0 || f.Dia > 6 {
			return problema.Invalido("El día va de 0 (domingo) a 6 (sábado).")
		}
		d, err1 := time.Parse("15:04", f.Desde)
		h, err2 := time.Parse("15:04", f.Hasta)
		if err1 != nil || err2 != nil || !h.After(d) {
			return problema.Invalido("Cada franja va de una hora HH:MM a otra más tarde.")
		}
		for _, o := range hs[i+1:] {
			od, _ := time.Parse("15:04", o.Desde)
			oh, _ := time.Parse("15:04", o.Hasta)
			if o.Dia == f.Dia && d.Before(oh) && od.Before(h) {
				return problema.Invalido("Hay franjas del mismo día que se pisan.")
			}
		}
	}
	return nil
}

func ponerHorario(ctx context.Context, tx pgx.Tx, id string, hs []disponibilidad.Franja) error {
	if err := validarHorario(hs); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, "DELETE FROM horarios WHERE profesional_id = $1", id); err != nil {
		return err
	}
	for _, f := range hs {
		if _, err := tx.Exec(ctx, "INSERT INTO horarios (negocio_id, profesional_id, dia, desde, hasta) VALUES (app.negocio_actual(), $1, $2, $3, $4)",
			id, f.Dia, f.Desde, f.Hasta); err != nil {
			return err
		}
	}
	return nil
}

// reemplazar corre fn sobre un profesional que existe y devuelve el profesional actualizado.
func (s *Servicio) reemplazar(w http.ResponseWriter, r *http.Request, fn func(ctx context.Context, tx pgx.Tx, id string) error) error {
	var p Profesional
	err := s.enNegocio(r, func(ctx context.Context, tx pgx.Tx) error {
		var err error
		if p, err = LeerProfesional(ctx, tx, r.PathValue("id"), false); err != nil {
			return err
		}
		if err := fn(ctx, tx, p.ID); err != nil {
			return err
		}
		p, err = LeerProfesional(ctx, tx, p.ID, false)
		return err
	})
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusOK, p)
	return nil
}

func (s *Servicio) asignarServicios(w http.ResponseWriter, r *http.Request) error {
	var in struct {
		Servicios []string `json:"servicios"`
	}
	if err := web.Leer(w, r, &in); err != nil {
		return err
	}
	return s.reemplazar(w, r, func(ctx context.Context, tx pgx.Tx, id string) error {
		return ponerServicios(ctx, tx, id, in.Servicios)
	})
}

func (s *Servicio) guardarHorario(w http.ResponseWriter, r *http.Request) error {
	var in struct {
		Horario []disponibilidad.Franja `json:"horario"`
	}
	if err := web.Leer(w, r, &in); err != nil {
		return err
	}
	return s.reemplazar(w, r, func(ctx context.Context, tx pgx.Tx, id string) error { return ponerHorario(ctx, tx, id, in.Horario) })
}

// puedeTocar: un profesional (rol) sólo maneja sus propios bloqueos y excepciones.
func puedeTocar(ctx context.Context, tx pgx.Tx, profesionalID string) error {
	ses := sesion.De(ctx)
	if ses.Rol != sesion.Profesional {
		return nil
	}
	var suyo bool
	err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM profesionales p JOIN miembros m ON m.id = p.miembro_id
		WHERE p.id = $1 AND m.clerk_id = $2)`, profesionalID, ses.ClerkID).Scan(&suyo)
	if err != nil {
		return err
	}
	if !suyo {
		return problema.SinPermiso("Sólo podés cambiar tu propia agenda.")
	}
	return nil
}

// ---- Bloqueos por fecha

// Bloqueos de los profesionales entre dos fechas (inclusive), en el negocio fijado.
func Bloqueos(ctx context.Context, tx pgx.Tx, profesionalID, desde, hasta string) ([]disponibilidad.Bloqueo, error) {
	filas, err := tx.Query(ctx, `SELECT id::text, to_char(fecha, 'YYYY-MM-DD'), to_char(desde, 'HH24:MI'), to_char(hasta, 'HH24:MI'), motivo
		FROM bloqueos WHERE profesional_id = $1 AND fecha BETWEEN $2 AND $3 ORDER BY fecha, desde NULLS FIRST`, profesionalID, desde, hasta)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(filas, pgx.RowToStructByPos[disponibilidad.Bloqueo])
}

func rango(r *http.Request) (string, string) {
	desde, hasta := r.URL.Query().Get("desde"), r.URL.Query().Get("hasta")
	if desde == "" {
		desde = time.Now().In(disponibilidad.ZonaPorDefecto).Format(time.DateOnly)
	}
	if hasta == "" {
		hasta = "9999-12-31"
	}
	return desde, hasta
}

func (s *Servicio) bloqueos(w http.ResponseWriter, r *http.Request) error {
	var bs []disponibilidad.Bloqueo
	desde, hasta := rango(r)
	err := s.enNegocio(r, func(ctx context.Context, tx pgx.Tx) (err error) {
		bs, err = Bloqueos(ctx, tx, r.PathValue("id"), desde, hasta)
		return err
	})
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusOK, map[string]any{"bloqueos": bs})
	return nil
}

func (s *Servicio) crearBloqueo(w http.ResponseWriter, r *http.Request) error {
	var b disponibilidad.Bloqueo
	if err := web.Leer(w, r, &b); err != nil {
		return err
	}
	if (b.Desde == nil) != (b.Hasta == nil) {
		return problema.Invalido("Un bloqueo parcial lleva desde y hasta; sin ninguno es el día entero.")
	}
	err := s.enNegocio(r, func(ctx context.Context, tx pgx.Tx) error {
		if err := puedeTocar(ctx, tx, r.PathValue("id")); err != nil {
			return err
		}
		return tx.QueryRow(ctx, `INSERT INTO bloqueos (negocio_id, profesional_id, fecha, desde, hasta, motivo)
			VALUES (app.negocio_actual(), $1, $2, $3, $4, $5) RETURNING id::text`,
			r.PathValue("id"), b.Fecha, b.Desde, b.Hasta, b.Motivo).Scan(&b.ID)
	})
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusCreated, b)
	return nil
}

func (s *Servicio) borrarBloqueo(w http.ResponseWriter, r *http.Request) error {
	err := s.enNegocio(r, func(ctx context.Context, tx pgx.Tx) error {
		if err := puedeTocar(ctx, tx, r.PathValue("id")); err != nil {
			return err
		}
		tag, err := tx.Exec(ctx, "DELETE FROM bloqueos WHERE id = $1 AND profesional_id = $2", r.PathValue("bloqueo"), r.PathValue("id"))
		if err != nil {
			return err
		}
		return noExiste(tag)
	})
	if err != nil {
		return err
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}

// ---- Bloqueos semanales

// Semanales de un profesional en el negocio fijado.
func Semanales(ctx context.Context, tx pgx.Tx, profesionalID string) ([]disponibilidad.Semanal, error) {
	filas, err := tx.Query(ctx, "SELECT dia, to_char(hora, 'HH24:MI') FROM bloqueos_semanales WHERE profesional_id = $1 ORDER BY dia, hora", profesionalID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(filas, pgx.RowToStructByPos[disponibilidad.Semanal])
}

func (s *Servicio) semanales(w http.ResponseWriter, r *http.Request) error {
	var ss []disponibilidad.Semanal
	err := s.enNegocio(r, func(ctx context.Context, tx pgx.Tx) (err error) {
		ss, err = Semanales(ctx, tx, r.PathValue("id"))
		return err
	})
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusOK, map[string]any{"semanales": ss})
	return nil
}

func (s *Servicio) guardarSemanales(w http.ResponseWriter, r *http.Request) error {
	var in struct {
		Semanales []disponibilidad.Semanal `json:"semanales"`
	}
	if err := web.Leer(w, r, &in); err != nil {
		return err
	}
	var ss []disponibilidad.Semanal
	err := s.enNegocio(r, func(ctx context.Context, tx pgx.Tx) error {
		if _, err := LeerProfesional(ctx, tx, r.PathValue("id"), false); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, "DELETE FROM bloqueos_semanales WHERE profesional_id = $1", r.PathValue("id")); err != nil {
			return err
		}
		for _, b := range in.Semanales {
			if _, err := tx.Exec(ctx, `INSERT INTO bloqueos_semanales (negocio_id, profesional_id, dia, hora) VALUES (app.negocio_actual(), $1, $2, $3)
				ON CONFLICT DO NOTHING`, r.PathValue("id"), b.Dia, b.Hora); err != nil {
				return err
			}
		}
		var err error
		ss, err = Semanales(ctx, tx, r.PathValue("id"))
		return err
	})
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusOK, map[string]any{"semanales": ss})
	return nil
}

// ---- Excepciones por fecha y hora

// Excepciones de un profesional entre dos fechas, en el negocio fijado.
func Excepciones(ctx context.Context, tx pgx.Tx, profesionalID, desde, hasta string) ([]disponibilidad.Excepcion, error) {
	filas, err := tx.Query(ctx, `SELECT to_char(fecha, 'YYYY-MM-DD'), to_char(hora, 'HH24:MI'), tipo FROM excepciones
		WHERE profesional_id = $1 AND fecha BETWEEN $2 AND $3 ORDER BY fecha, hora`, profesionalID, desde, hasta)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(filas, pgx.RowToStructByPos[disponibilidad.Excepcion])
}

func (s *Servicio) excepciones(w http.ResponseWriter, r *http.Request) error {
	var es []disponibilidad.Excepcion
	desde, hasta := rango(r)
	err := s.enNegocio(r, func(ctx context.Context, tx pgx.Tx) (err error) {
		es, err = Excepciones(ctx, tx, r.PathValue("id"), desde, hasta)
		return err
	})
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusOK, map[string]any{"excepciones": es})
	return nil
}

func (s *Servicio) guardarExcepcion(w http.ResponseWriter, r *http.Request) error {
	var e disponibilidad.Excepcion
	if err := web.Leer(w, r, &e); err != nil {
		return err
	}
	if e.Tipo != "bloqueado" && e.Tipo != "libre" {
		return problema.Invalido("El tipo es bloqueado o libre.")
	}
	err := s.enNegocio(r, func(ctx context.Context, tx pgx.Tx) error {
		if err := puedeTocar(ctx, tx, r.PathValue("id")); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `INSERT INTO excepciones (negocio_id, profesional_id, fecha, hora, tipo) VALUES (app.negocio_actual(), $1, $2, $3, $4)
			ON CONFLICT (profesional_id, fecha, hora) DO UPDATE SET tipo = excluded.tipo`, r.PathValue("id"), e.Fecha, e.Hora, e.Tipo)
		return err
	})
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusOK, e)
	return nil
}

func (s *Servicio) borrarExcepcion(w http.ResponseWriter, r *http.Request) error {
	err := s.enNegocio(r, func(ctx context.Context, tx pgx.Tx) error {
		if err := puedeTocar(ctx, tx, r.PathValue("id")); err != nil {
			return err
		}
		tag, err := tx.Exec(ctx, "DELETE FROM excepciones WHERE profesional_id = $1 AND fecha = $2 AND hora = $3",
			r.PathValue("id"), r.PathValue("fecha"), r.PathValue("hora"))
		if err != nil {
			return err
		}
		return noExiste(tag)
	})
	if err != nil {
		return err
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}
