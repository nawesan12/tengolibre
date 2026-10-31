// Package espera: la lista de espera. El cliente se anota desde la página pública (un servicio, un
// profesional si le importa, entre qué días y en qué momento del día); el panel la ve y avisa a mano,
// y cuando se libera un horario que le sirve (un turno cancelado o reprogramado) se le ofrece a una
// persona por vez (ofertas.go).
package espera

import (
	"context"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"

	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/avisos"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/turnos"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/auditoria"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/db"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/problema"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/sesion"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/web"
)

// Servicio de la lista de espera. Ahora es el reloj (tests); nil: time.Now.
type Servicio struct {
	Pool   *db.Pool
	Sesion *sesion.Servicio
	Ahora  func() time.Time
	// Turnos reserva cuando alguien toma el lugar (o recepción se lo da). nil: tomar no anda (el
	// worker no lo necesita).
	Turnos *turnos.Servicio
}

func (s *Servicio) ahora() time.Time {
	if s.Ahora != nil {
		return s.Ahora()
	}
	return time.Now()
}

// Rutas del dominio.
func (s *Servicio) Rutas(mux *http.ServeMux) {
	mux.HandleFunc("POST /v1/p/{slug}/espera", web.Envolver(s.anotarse))
	m := func(h web.Manejador) http.HandlerFunc { return s.Sesion.Panel(avisos.Mostrador, web.Envolver(h)) }
	mux.HandleFunc("GET /v1/panel/espera", m(s.listar))
	mux.HandleFunc("POST /v1/panel/espera/{id}/avisar", m(s.avisarManual))
	mux.HandleFunc("DELETE /v1/panel/espera/{id}", m(s.borrar))
	// Ofrecer el hueco a una persona por vez.
	mux.HandleFunc("GET /v1/panel/espera/candidatos", m(s.candidatosPanel))
	mux.HandleFunc("GET /v1/panel/espera/ofertas", m(s.listarOfertas))
	mux.HandleFunc("POST /v1/panel/espera/ofertas", m(s.ofrecer))
	mux.HandleFunc("GET /v1/panel/espera/ofertas/{id}", m(s.verOferta))
	mux.HandleFunc("POST /v1/panel/espera/ofertas/{id}/frenar", m(s.frenar))
	mux.HandleFunc("POST /v1/panel/espera/ofertas/{id}/cargar", m(s.cargar))
	mux.HandleFunc("GET /v1/panel/ajustes/espera", m(s.verAjustes))
	mux.HandleFunc("PUT /v1/panel/ajustes/espera", s.Sesion.Panel(sesion.Gestion, web.Envolver(s.guardarAjustes)))
	// El link del mail (el token es el secreto).
	mux.HandleFunc("GET /v1/espera/{token}", web.Envolver(s.verLink))
	mux.HandleFunc("POST /v1/espera/{token}/tomar", web.Envolver(s.tomar))
	mux.HandleFunc("POST /v1/espera/{token}/pasar", web.Envolver(s.pasar))
}

// Anotado en la lista.
type Anotado struct {
	ID            string  `json:"id"`
	ServicioID    string  `json:"servicioId"`
	Servicio      string  `json:"servicio"`
	ProfesionalID *string `json:"profesionalId"`
	Profesional   *string `json:"profesional"`
	Nombre        string  `json:"nombre"`
	Email         string  `json:"email"`
	Telefono      string  `json:"telefono"`
	Desde         string  `json:"desde"`
	Hasta         string  `json:"hasta"`
	// Momento: cualquiera, manana (antes de las 13) o tarde.
	Momento   string     `json:"momento"`
	AvisadoEn *time.Time `json:"avisadoEn"`
	CreadoEn  time.Time  `json:"creadoEn"`
	// Resultado de la última vez que se le ofreció un hueco: ofrecida (tiene el link ahora), tomada,
	// paso, vencida (no respondió) o cancelada. nil: nunca se le ofreció.
	Resultado *string `json:"resultado"`
}

const columnas = `e.id::text, e.servicio_id::text, s.nombre, e.profesional_id::text, p.nombre, e.nombre, e.email, e.telefono,
	to_char(e.desde, 'YYYY-MM-DD'), to_char(e.hasta, 'YYYY-MM-DD'), e.momento, e.avisado_en, e.creado_en,
	(SELECT i.estado FROM oferta_intentos i WHERE i.espera_id = e.id AND i.ofrecida_en IS NOT NULL ORDER BY i.ofrecida_en DESC LIMIT 1)`

const desde = ` FROM espera e JOIN servicios s ON s.id = e.servicio_id LEFT JOIN profesionales p ON p.id = e.profesional_id`

func zona(ctx context.Context, tx pgx.Tx) *time.Location {
	var z string
	_ = tx.QueryRow(ctx, "SELECT zona_horaria FROM negocios WHERE id = app.negocio_actual()").Scan(&z)
	if l, err := time.LoadLocation(z); err == nil && z != "" {
		return l
	}
	l, _ := time.LoadLocation("America/Argentina/Buenos_Aires")
	return l
}

func (s *Servicio) anotarse(w http.ResponseWriter, r *http.Request) error {
	var in struct {
		ServicioID    string `json:"servicioId"`
		ProfesionalID string `json:"profesionalId"`
		Nombre        string `json:"nombre"`
		Email         string `json:"email"`
		Telefono      string `json:"telefono"`
		Desde         string `json:"desde"`
		Hasta         string `json:"hasta"`
		Momento       string `json:"momento"`
	}
	if err := web.Leer(w, r, &in); err != nil {
		return err
	}
	if in.Momento == "" {
		in.Momento = "cualquiera"
	}
	if in.Momento != "cualquiera" && in.Momento != "manana" && in.Momento != "tarde" {
		return problema.Invalido("El momento va como cualquiera, manana o tarde.", problema.Campo{Ruta: "momento", Mensaje: "inválido"})
	}
	in.Nombre, in.Email, in.Telefono = strings.TrimSpace(in.Nombre), strings.ToLower(strings.TrimSpace(in.Email)), strings.TrimSpace(in.Telefono)
	if n := utf8.RuneCountInString(in.Nombre); n < 1 || n > 80 {
		return problema.Invalido("Falta tu nombre.", problema.Campo{Ruta: "nombre", Mensaje: "requerido"})
	}
	// El aviso sale por mail: sin email no hay a quién avisarle.
	if !strings.Contains(in.Email, "@") || len(in.Email) > 120 {
		return problema.Invalido("Dejá un email para avisarte.", problema.Campo{Ruta: "email", Mensaje: "inválido"})
	}
	if len(in.Telefono) > 30 {
		return problema.Invalido("Revisá el teléfono.", problema.Campo{Ruta: "telefono", Mensaje: "inválido"})
	}
	if in.Hasta == "" {
		in.Hasta = in.Desde
	}
	ctx := r.Context()
	var negocio *string
	err := s.Pool.EnTx(ctx, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, "SELECT app.negocio_publico($1)::text", r.PathValue("slug")).Scan(&negocio)
	})
	if err != nil {
		return err
	}
	if negocio == nil {
		return problema.NoEncontrado("No encontramos ese negocio.")
	}
	var a Anotado
	err = s.Pool.EnNegocio(ctx, *negocio, func(tx pgx.Tx) error {
		z := zona(ctx, tx)
		d, err1 := time.ParseInLocation(time.DateOnly, in.Desde, z)
		h, err2 := time.ParseInLocation(time.DateOnly, in.Hasta, z)
		hoy := s.ahora().In(z).Format(time.DateOnly)
		if err1 != nil || err2 != nil || h.Before(d) || h.Sub(d) > 31*24*time.Hour || in.Hasta < hoy {
			return problema.Invalido("Elegí desde y hasta qué día (AAAA-MM-DD, hasta un mes, desde hoy).",
				problema.Campo{Ruta: "desde", Mensaje: "rango inválido"})
		}
		if in.Desde < hoy {
			in.Desde = hoy
		}
		var activo bool
		var servicio string
		if err := tx.QueryRow(ctx, "SELECT activo, nombre FROM servicios WHERE id::text = $1", in.ServicioID).Scan(&activo, &servicio); err != nil || !activo {
			return problema.NoEncontrado("Ese servicio no existe o no se puede reservar.")
		}
		if in.ProfesionalID != "" {
			var hace bool
			if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM profesional_servicios ps JOIN profesionales p ON p.id = ps.profesional_id
				WHERE p.id::text = $1 AND p.activo AND ps.servicio_id::text = $2)`, in.ProfesionalID, in.ServicioID).Scan(&hace); err != nil {
				return err
			}
			if !hace {
				return problema.NoEncontrado("Ese profesional no hace ese servicio.")
			}
		}
		var id string
		if err := tx.QueryRow(ctx, `INSERT INTO espera (negocio_id, servicio_id, profesional_id, nombre, email, telefono, desde, hasta, momento)
			VALUES (app.negocio_actual(), $1, nullif($2, '')::uuid, $3, $4, $5, $6, $7, $8) RETURNING id::text`,
			in.ServicioID, in.ProfesionalID, in.Nombre, in.Email, in.Telefono, in.Desde, in.Hasta, in.Momento).Scan(&id); err != nil {
			return err
		}
		if err := leer(ctx, tx, id, &a); err != nil {
			return err
		}
		rango := "el " + avisos.Fecha(d)
		if in.Hasta != in.Desde {
			rango = "entre el " + avisos.Fecha(d) + " y el " + avisos.Fecha(h)
		}
		return avisos.Notificar(ctx, tx, avisos.Notificacion{Tipo: avisos.NotiEspera, Titulo: in.Nombre + " se anotó en la lista de espera",
			Texto: "Para " + servicio + ", " + rango + "."})
	})
	if err != nil {
		return err
	}
	// Al cliente no le mostramos su id: no tiene nada que hacer con él.
	web.JSON(w, http.StatusCreated, map[string]any{"servicio": a.Servicio, "desde": a.Desde, "hasta": a.Hasta, "momento": a.Momento})
	return nil
}

// primerNombre: "Martina Gómez" → "Martina".
func primerNombre(nombre string) string {
	if f := strings.Fields(nombre); len(f) > 0 {
		return f[0]
	}
	return nombre
}

func leer(ctx context.Context, tx pgx.Tx, id string, a *Anotado) error {
	return escanear(tx.QueryRow(ctx, "SELECT "+columnas+desde+" WHERE e.id::text = $1", id), a)
}

func escanear(f pgx.Row, a *Anotado) error {
	return f.Scan(&a.ID, &a.ServicioID, &a.Servicio, &a.ProfesionalID, &a.Profesional, &a.Nombre, &a.Email, &a.Telefono,
		&a.Desde, &a.Hasta, &a.Momento, &a.AvisadoEn, &a.CreadoEn, &a.Resultado)
}

// listar: los que siguen esperando (sin aviso y con el rango vigente); con ?todos=1, también los
// avisados y vencidos.
func (s *Servicio) listar(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	out := []Anotado{}
	err := s.Pool.EnNegocio(ctx, sesion.De(ctx).NegocioID, func(tx pgx.Tx) error {
		cond := "true"
		var args []any
		if r.URL.Query().Get("todos") != "1" {
			cond = "e.avisado_en IS NULL AND e.hasta >= $1::date"
			args = append(args, s.ahora().In(zona(ctx, tx)).Format(time.DateOnly))
		}
		filas, err := tx.Query(ctx, "SELECT "+columnas+desde+" WHERE "+cond+" ORDER BY e.desde, e.creado_en LIMIT 200", args...)
		if err != nil {
			return err
		}
		defer filas.Close()
		for filas.Next() {
			var a Anotado
			if err := escanear(filas, &a); err != nil {
				return err
			}
			out = append(out, a)
		}
		return filas.Err()
	})
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusOK, map[string]any{"espera": out})
	return nil
}

// avisarManual: el negocio le avisa a alguien de la lista (con un horario puntual si lo tiene, y una
// nota opcional que va en el mail).
func (s *Servicio) avisarManual(w http.ResponseWriter, r *http.Request) error {
	var in struct {
		Inicio *time.Time `json:"inicio"`
		Nota   string     `json:"nota"`
	}
	if err := web.Leer(w, r, &in); err != nil {
		return err
	}
	in.Nota = strings.TrimSpace(in.Nota)
	if utf8.RuneCountInString(in.Nota) > 300 {
		return problema.Invalido("La nota es muy larga (hasta 300).")
	}
	ctx := r.Context()
	ahora := s.ahora()
	var a Anotado
	err := s.Pool.EnNegocio(ctx, sesion.De(ctx).NegocioID, func(tx pgx.Tx) error {
		if err := escanear(tx.QueryRow(ctx, "SELECT "+columnas+desde+" WHERE e.id::text = $1 FOR UPDATE OF e", r.PathValue("id")), &a); err != nil {
			return err
		}
		datos := map[string]any{"espera": a.ID}
		if in.Inicio != nil {
			datos["inicio"] = in.Inicio.UTC().Format(time.RFC3339Nano)
		}
		if a.Profesional != nil {
			datos["profesional"] = *a.Profesional
		}
		if in.Nota != "" {
			datos["nota"] = in.Nota
		}
		if _, err := avisos.Encolar(ctx, tx, avisos.Aviso{Tipo: avisos.Espera, Para: a.Email, Datos: datos}, ahora); err != nil {
			return err
		}
		if err := tx.QueryRow(ctx, "UPDATE espera SET avisado_en = $2 WHERE id::text = $1 RETURNING avisado_en", a.ID, ahora).Scan(&a.AvisadoEn); err != nil {
			return err
		}
		return auditoria.Anotar(ctx, tx, auditoria.Registro{Accion: "espera.avisada", Entidad: "espera", EntidadID: a.ID,
			Detalle: map[string]any{"email": a.Email, "inicio": in.Inicio}})
	})
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusOK, a)
	return nil
}

func (s *Servicio) borrar(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	err := s.Pool.EnNegocio(ctx, sesion.De(ctx).NegocioID, func(tx pgx.Tx) error {
		var email string
		if err := tx.QueryRow(ctx, "DELETE FROM espera WHERE id::text = $1 RETURNING email", r.PathValue("id")).Scan(&email); err != nil {
			return err
		}
		return auditoria.Anotar(ctx, tx, auditoria.Registro{Accion: "espera.borrada", Entidad: "espera", EntidadID: r.PathValue("id"),
			Detalle: map[string]any{"email": email}})
	})
	if err != nil {
		return err
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}
