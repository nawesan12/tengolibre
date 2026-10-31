// Package calendario: la agenda del negocio (o de un profesional) como calendario para suscribirse
// desde Google Calendar, Apple o Outlook. Es un link secreto que el calendario relee solo; no hace
// falta conectar ninguna cuenta.
package calendario

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/db"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/problema"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/sesion"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/web"
)

// Servicio de calendarios. Sitio arma los links (el front reenvía /cal/… a la API). Ahora es el
// reloj (tests); nil: time.Now.
type Servicio struct {
	Pool   *db.Pool
	Sesion *sesion.Servicio
	Sitio  string
	Ahora  func() time.Time
}

func (s *Servicio) ahora() time.Time {
	if s.Ahora != nil {
		return s.Ahora()
	}
	return time.Now()
}

// Rutas del dominio.
func (s *Servicio) Rutas(mux *http.ServeMux) {
	a := func(h web.Manejador) http.HandlerFunc { return s.Sesion.Panel(sesion.Agenda, web.Envolver(h)) }
	mux.HandleFunc("GET /v1/panel/calendarios", a(s.listar))
	mux.HandleFunc("POST /v1/panel/calendarios", a(s.crear))
	mux.HandleFunc("DELETE /v1/panel/calendarios/{id}", a(s.borrar))
	mux.HandleFunc("GET /v1/cal/{archivo}", web.Envolver(s.feed))
}

// Calendario como lo ve el panel. Sin URL: todavía no se creó.
type Calendario struct {
	ID            *string `json:"id"`
	ProfesionalID *string `json:"profesionalId"`
	Nombre        string  `json:"nombre"`
	URL           *string `json:"url"`
}

func gestiona(rol string) bool { return rol == sesion.Duenio || rol == sesion.Encargado }

// propio: el profesional del miembro que pide (los que no gestionan sólo ven el suyo).
func propio(ctx context.Context, tx pgx.Tx, clerk string) (string, error) {
	var id string
	err := tx.QueryRow(ctx, `SELECT p.id::text FROM profesionales p JOIN miembros m ON m.id = p.miembro_id
		WHERE m.clerk_id = $1 AND p.activo LIMIT 1`, clerk).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	return id, err
}

func (s *Servicio) url(token string) *string {
	u := strings.TrimSuffix(s.Sitio, "/") + "/cal/" + token + ".ics"
	return &u
}

func (s *Servicio) listar(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	ses := sesion.De(ctx)
	out := []Calendario{}
	err := s.Pool.EnNegocio(ctx, ses.NegocioID, func(tx pgx.Tx) error {
		solo := ""
		if !gestiona(ses.Rol) {
			var err error
			if solo, err = propio(ctx, tx, ses.ClerkID); err != nil || solo == "" {
				return err
			}
		}
		filas, err := tx.Query(ctx, `SELECT c.id::text, x.id, x.nombre, c.token FROM (
				SELECT NULL::uuid AS id, 'Todo el negocio' AS nombre, 0 AS orden, '' AS orden2 WHERE $1 = ''
				UNION ALL
				SELECT p.id, p.nombre, 1, p.nombre FROM profesionales p WHERE p.activo AND ($1 = '' OR p.id::text = $1)
			) x LEFT JOIN calendarios c ON c.profesional_id IS NOT DISTINCT FROM x.id
			ORDER BY x.orden, x.orden2`, solo)
		if err != nil {
			return err
		}
		defer filas.Close()
		for filas.Next() {
			var c Calendario
			var token *string
			if err := filas.Scan(&c.ID, &c.ProfesionalID, &c.Nombre, &token); err != nil {
				return err
			}
			if token != nil {
				c.URL = s.url(*token)
			}
			out = append(out, c)
		}
		return filas.Err()
	})
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusOK, map[string]any{"calendarios": out})
	return nil
}

// permitido dice si quien pide puede tocar el calendario de ese profesional ("" = el del negocio).
func permitido(ctx context.Context, tx pgx.Tx, ses sesion.Sesion, profesional string) error {
	if gestiona(ses.Rol) {
		return nil
	}
	p, err := propio(ctx, tx, ses.ClerkID)
	if err != nil {
		return err
	}
	if profesional == "" || p != profesional {
		return &problema.Problema{Estado: http.StatusForbidden, Codigo: "sin_permiso", Titulo: "No podés hacer esto",
			Detalle: "Sólo podés manejar el calendario de tus turnos."}
	}
	return nil
}

// crear arma el calendario, o lo regenera si ya estaba (el link viejo deja de andar).
func (s *Servicio) crear(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	ses := sesion.De(ctx)
	var in struct {
		ProfesionalID string `json:"profesionalId"`
	}
	if err := web.Leer(w, r, &in); err != nil {
		return err
	}
	b := make([]byte, 24)
	_, _ = rand.Read(b)
	token := base64.RawURLEncoding.EncodeToString(b)
	var c Calendario
	err := s.Pool.EnNegocio(ctx, ses.NegocioID, func(tx pgx.Tx) error {
		if err := permitido(ctx, tx, ses, in.ProfesionalID); err != nil {
			return err
		}
		var prof *string
		c.Nombre = "Todo el negocio"
		if in.ProfesionalID != "" {
			prof = &in.ProfesionalID
			err := tx.QueryRow(ctx, "SELECT nombre FROM profesionales WHERE id::text = $1 AND activo", in.ProfesionalID).Scan(&c.Nombre)
			if errors.Is(err, pgx.ErrNoRows) {
				return problema.NoEncontrado("Ese profesional no existe.")
			}
			if err != nil {
				return err
			}
		}
		if _, err := tx.Exec(ctx, "DELETE FROM calendarios WHERE profesional_id IS NOT DISTINCT FROM $1::uuid", prof); err != nil {
			return err
		}
		c.ProfesionalID = prof
		return tx.QueryRow(ctx, `INSERT INTO calendarios (negocio_id, profesional_id, token) VALUES (app.negocio_actual(), $1, $2)
			RETURNING id::text`, prof, token).Scan(&c.ID)
	})
	if err != nil {
		return err
	}
	c.URL = s.url(token)
	web.JSON(w, http.StatusCreated, c)
	return nil
}

func (s *Servicio) borrar(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	ses := sesion.De(ctx)
	err := s.Pool.EnNegocio(ctx, ses.NegocioID, func(tx pgx.Tx) error {
		var prof *string
		err := tx.QueryRow(ctx, "SELECT profesional_id::text FROM calendarios WHERE id::text = $1", r.PathValue("id")).Scan(&prof)
		if errors.Is(err, pgx.ErrNoRows) {
			return problema.NoEncontrado("Ese calendario no existe.")
		}
		if err != nil {
			return err
		}
		p := ""
		if prof != nil {
			p = *prof
		}
		if err := permitido(ctx, tx, ses, p); err != nil {
			return err
		}
		_, err = tx.Exec(ctx, "DELETE FROM calendarios WHERE id::text = $1", r.PathValue("id"))
		return err
	})
	if err != nil {
		return err
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}

// Ventana del calendario: un mes para atrás y seis para adelante.
const (
	atras    = 30 * 24 * time.Hour
	adelante = 183 * 24 * time.Hour
	tope     = 3000
)

var estados = map[string]string{"pendiente_pago": "Esperando el pago", "pendiente": "Pendiente", "confirmado": "Confirmado",
	"atendido": "Atendido", "ausente": "No vino"}

func (s *Servicio) feed(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	token, ok := strings.CutSuffix(r.PathValue("archivo"), ".ics")
	if !ok {
		return problema.NoEncontrado("No encontramos ese calendario.")
	}
	var negocio *string
	if err := s.Pool.EnTx(ctx, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, "SELECT app.negocio_del_calendario($1)::text", token).Scan(&negocio)
	}); err != nil {
		return err
	}
	if negocio == nil {
		return problema.NoEncontrado("No encontramos ese calendario.")
	}
	ahora := s.ahora()
	var nombre string
	var eventos []Evento
	err := s.Pool.EnNegocio(ctx, *negocio, func(tx pgx.Tx) error {
		var prof *string
		var nomProf *string
		if err := tx.QueryRow(ctx, `SELECT n.nombre, c.profesional_id::text, p.nombre FROM calendarios c JOIN negocios n ON n.id = c.negocio_id
			LEFT JOIN profesionales p ON p.id = c.profesional_id WHERE c.token = $1`, token).Scan(&nombre, &prof, &nomProf); err != nil {
			return err
		}
		if nomProf != nil {
			nombre += " · " + *nomProf
		}
		filas, err := tx.Query(ctx, `SELECT t.id::text, t.servicio, t.profesional, t.inicio, t.fin, t.estado, t.nota, t.actualizado_en,
				c.nombre, c.telefono, c.email
			FROM turnos t JOIN clientes c ON c.id = t.cliente_id
			WHERE t.estado <> 'cancelado' AND (t.estado <> 'pendiente_pago' OR t.vence_en IS NULL OR t.vence_en > $1)
				AND t.inicio >= $2 AND t.inicio < $3 AND ($4::uuid IS NULL OR t.profesional_id = $4::uuid)
			ORDER BY t.inicio LIMIT $5`, ahora, ahora.Add(-atras), ahora.Add(adelante), prof, tope)
		if err != nil {
			return err
		}
		defer filas.Close()
		for filas.Next() {
			var id, servicio, profesional, estado, nota, cliente, tel, email string
			var e Evento
			if err := filas.Scan(&id, &servicio, &profesional, &e.Inicio, &e.Fin, &estado, &nota, &e.Cambio, &cliente, &tel, &email); err != nil {
				return err
			}
			e.UID = id + "@tengolibre.com.ar"
			e.Titulo = servicio + " · " + cliente
			e.Tentativo = estado == "pendiente" || estado == "pendiente_pago"
			e.URL = strings.TrimSuffix(s.Sitio, "/") + "/panel/turnos/" + id
			var d []string
			if prof == nil && profesional != "" {
				d = append(d, "Con "+profesional)
			}
			if tel != "" {
				d = append(d, "Teléfono: "+tel)
			}
			if email != "" {
				d = append(d, "Email: "+email)
			}
			d = append(d, "Estado: "+estados[estado])
			if nota != "" {
				d = append(d, "Nota: "+nota)
			}
			e.Descripcion = strings.Join(d, "\n")
			eventos = append(eventos, e)
		}
		return filas.Err()
	})
	if err != nil {
		return err
	}
	w.Header().Set("Content-Type", "text/calendar; charset=utf-8")
	w.Header().Set("Cache-Control", "private, max-age=300")
	w.Header().Set("X-Robots-Tag", "noindex")
	_, err = w.Write(ICS(nombre, eventos))
	return err
}
