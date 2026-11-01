// Package invitaciones: sumar al equipo a alguien que todavía no tiene cuenta. La dueña lo invita
// por email (con un rol), le llega un link con un token, y al aceptarlo con su sesión de Clerk se
// crea el miembro con ese rol. A quien ya tiene cuenta se lo suma directo (negocios, POST /equipo).
package invitaciones

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/avisos"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/auditoria"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/db"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/problema"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/sesion"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/web"
)

// Servicio de invitaciones. Ahora es el reloj (tests); nil: time.Now.
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

// El link sirve una semana.
const vigencia = 7 * 24 * time.Hour

// Rutas del dominio.
func (s *Servicio) Rutas(mux *http.ServeMux) {
	ses := s.Sesion
	duenio := []string{sesion.Duenio}
	mux.HandleFunc("GET /v1/panel/invitaciones", ses.Panel(sesion.Gestion, web.Envolver(s.listar)))
	mux.HandleFunc("POST /v1/panel/invitaciones", ses.Panel(duenio, web.Envolver(s.invitar)))
	mux.HandleFunc("DELETE /v1/panel/invitaciones/{id}", ses.Panel(duenio, web.Envolver(s.borrar)))
	// El link de la invitación: verla no pide sesión; aceptarla, sí.
	mux.HandleFunc("GET /v1/invitaciones/{token}", web.Envolver(s.ver))
	mux.HandleFunc("POST /v1/invitaciones/{token}/aceptar", ses.Persona(web.Envolver(s.aceptar)))
}

// Invitacion como la ve el panel (sin el token: el link sólo lo tiene quien recibió el mail).
type Invitacion struct {
	ID         string     `json:"id"`
	Email      string     `json:"email"`
	Rol        string     `json:"rol"`
	Invito     string     `json:"invito"`
	VenceEn    time.Time  `json:"venceEn"`
	AceptadaEn *time.Time `json:"aceptadaEn"`
	CreadoEn   time.Time  `json:"creadoEn"`
}

const columnas = "id::text, email, rol, invito, vence_en, aceptada_en, creado_en"

var roles = map[string]bool{sesion.Encargado: true, sesion.Recepcion: true, sesion.Profesional: true}

func (s *Servicio) invitar(w http.ResponseWriter, r *http.Request) error {
	var in struct {
		Email string `json:"email"`
		Rol   string `json:"rol"`
	}
	if err := web.Leer(w, r, &in); err != nil {
		return err
	}
	in.Email = strings.ToLower(strings.TrimSpace(in.Email))
	if !strings.Contains(in.Email, "@") || len(in.Email) > 120 || strings.ContainsAny(in.Email, " \r\n<>") {
		return problema.Invalido("Revisá el email.", problema.Campo{Ruta: "email", Mensaje: "inválido"})
	}
	// Dueña no se invita: el traspaso del negocio es otra cosa.
	if !roles[in.Rol] {
		return problema.Invalido("Rol desconocido.", problema.Campo{Ruta: "rol", Mensaje: "encargado, recepcion o profesional"})
	}
	ctx := r.Context()
	ahora := s.ahora()
	var x Invitacion
	err := s.Pool.EnNegocio(ctx, sesion.De(ctx).NegocioID, func(tx pgx.Tx) error {
		var yaEsta bool
		if err := tx.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM miembros WHERE lower(email) = $1 AND activo)", in.Email).Scan(&yaEsta); err != nil {
			return err
		}
		if yaEsta {
			return problema.Conflicto("ya_es_miembro", "Esa persona ya es parte del equipo.")
		}
		// Invitar de nuevo reemplaza la pendiente (el link viejo deja de servir).
		if _, err := tx.Exec(ctx, "DELETE FROM invitaciones WHERE email = $1 AND aceptada_en IS NULL", in.Email); err != nil {
			return err
		}
		token := avisos.Token()
		if err := tx.QueryRow(ctx, `INSERT INTO invitaciones (negocio_id, email, rol, token, invito, vence_en)
			VALUES (app.negocio_actual(), $1, $2, $3, $4, $5) RETURNING `+columnas, in.Email, in.Rol, token, sesion.De(ctx).ClerkID, ahora.Add(vigencia)).
			Scan(&x.ID, &x.Email, &x.Rol, &x.Invito, &x.VenceEn, &x.AceptadaEn, &x.CreadoEn); err != nil {
			return err
		}
		if _, err := avisos.Encolar(ctx, tx, avisos.Aviso{Tipo: avisos.Invitacion, Para: in.Email, Clave: "invitacion:" + x.ID,
			Datos: map[string]any{"token": token}}, ahora); err != nil {
			return err
		}
		return auditoria.Anotar(ctx, tx, auditoria.Registro{Accion: "equipo.invitado", Entidad: "invitacion", EntidadID: x.ID,
			Detalle: map[string]any{"email": in.Email, "rol": in.Rol}})
	})
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusCreated, x)
	return nil
}

// listar: las pendientes y las aceptadas de los últimos 30 días.
func (s *Servicio) listar(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	out := []Invitacion{}
	err := s.Pool.EnNegocio(ctx, sesion.De(ctx).NegocioID, func(tx pgx.Tx) error {
		filas, err := tx.Query(ctx, "SELECT "+columnas+` FROM invitaciones
			WHERE aceptada_en IS NULL OR aceptada_en > $1 ORDER BY creado_en DESC LIMIT 100`, s.ahora().Add(-30*24*time.Hour))
		if err != nil {
			return err
		}
		out, err = pgx.CollectRows(filas, pgx.RowToStructByPos[Invitacion])
		return err
	})
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusOK, map[string]any{"invitaciones": out})
	return nil
}

func (s *Servicio) borrar(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	err := s.Pool.EnNegocio(ctx, sesion.De(ctx).NegocioID, func(tx pgx.Tx) error {
		var email string
		if err := tx.QueryRow(ctx, "DELETE FROM invitaciones WHERE id::text = $1 AND aceptada_en IS NULL RETURNING email", r.PathValue("id")).
			Scan(&email); err != nil {
			return err
		}
		return auditoria.Anotar(ctx, tx, auditoria.Registro{Accion: "equipo.invitacion_borrada", Entidad: "invitacion",
			EntidadID: r.PathValue("id"), Detalle: map[string]any{"email": email}})
	})
	if err != nil {
		return err
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}

// Publica: lo que ve quien abre el link.
type Publica struct {
	Negocio  string `json:"negocio"`
	Slug     string `json:"slug"`
	Email    string `json:"email"`
	Rol      string `json:"rol"`
	Vencida  bool   `json:"vencida"`
	Aceptada bool   `json:"aceptada"`
	// InvitadoPor: el nombre de quien la mandó (vacío si ya no está en el equipo o no tiene nombre).
	InvitadoPor string    `json:"invitadoPor,omitempty"`
	VenceEn     time.Time `json:"venceEn"`
}

type fila struct {
	Publica
	id      string
	vence   time.Time
	negocio string
}

func (s *Servicio) conInvitacion(ctx context.Context, token string, fn func(tx pgx.Tx, f *fila) error) (fila, error) {
	var f fila
	negocio, err := avisos.NegocioPorToken(ctx, s.Pool.EnTx, "invitacion", token)
	if err != nil {
		return f, err
	}
	f.negocio = negocio
	err = s.Pool.EnNegocio(ctx, negocio, func(tx pgx.Tx) error {
		if err := tx.QueryRow(ctx, `SELECT i.id::text, n.nombre, n.slug, i.email, i.rol, i.vence_en, i.aceptada_en IS NOT NULL,
				coalesce((SELECT m.nombre FROM miembros m WHERE m.clerk_id = i.invito AND i.invito <> '' AND m.activo LIMIT 1), '')
			FROM invitaciones i JOIN negocios n ON n.id = i.negocio_id WHERE i.token = $1 FOR UPDATE OF i`, token).
			Scan(&f.id, &f.Negocio, &f.Slug, &f.Email, &f.Rol, &f.vence, &f.Aceptada, &f.InvitadoPor); err != nil {
			return err
		}
		f.VenceEn = f.vence
		f.Vencida = !f.vence.After(s.ahora())
		return fn(tx, &f)
	})
	return f, err
}

func (s *Servicio) ver(w http.ResponseWriter, r *http.Request) error {
	f, err := s.conInvitacion(r.Context(), r.PathValue("token"), func(pgx.Tx, *fila) error { return nil })
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusOK, f.Publica)
	return nil
}

// aceptar: con la sesión de Clerk, crea el miembro (o lo reactiva) con el rol de la invitación. La
// cuenta tiene que ser la del email invitado: el link reenviado a otra persona no le sirve.
func (s *Servicio) aceptar(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	clerkID := sesion.De(ctx).ClerkID
	u, err := s.Sesion.Verificador.Usuario(ctx, clerkID)
	if err != nil {
		return err
	}
	nombre := strings.TrimSpace(u.Nombre + " " + u.Apellido)
	var rol string
	f, err := s.conInvitacion(ctx, r.PathValue("token"), func(tx pgx.Tx, f *fila) error {
		if f.Aceptada {
			return problema.Conflicto("aceptada", "Esta invitación ya se usó.")
		}
		if f.Vencida {
			return problema.Conflicto("vencida", "Esta invitación venció. Pedile a quien te invitó que te mande otra.")
		}
		if !strings.EqualFold(strings.TrimSpace(u.Email), f.Email) {
			return problema.SinPermiso("La invitación es para " + f.Email + ". Ingresá con esa cuenta.")
		}
		// Si ya era miembro (inactivo, o con otro rol), toma el de la invitación; una dueña sigue siendo dueña.
		if err := tx.QueryRow(ctx, `INSERT INTO miembros (negocio_id, clerk_id, email, nombre, rol) VALUES (app.negocio_actual(), $1, $2, $3, $4)
			ON CONFLICT (negocio_id, clerk_id) DO UPDATE SET activo = true, email = excluded.email,
				rol = CASE WHEN miembros.rol = 'duenio' THEN 'duenio' ELSE excluded.rol END
			RETURNING rol`, clerkID, f.Email, nombre, f.Rol).Scan(&rol); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, "UPDATE invitaciones SET aceptada_en = $2, aceptada_por = $3 WHERE id::text = $1", f.id, s.ahora(), clerkID); err != nil {
			return err
		}
		quien := nombre
		if quien == "" {
			quien = f.Email
		}
		if err := avisos.Notificar(ctx, tx, avisos.Notificacion{Tipo: avisos.NotiEquipo, Titulo: quien + " se sumó al equipo",
			Texto: "Aceptó la invitación como " + f.Rol + "."}); err != nil {
			return err
		}
		return auditoria.Anotar(ctx, tx, auditoria.Registro{Quien: clerkID, Accion: "equipo.invitacion_aceptada", Entidad: "invitacion",
			EntidadID: f.id, Detalle: map[string]any{"email": f.Email, "rol": rol}})
	})
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusOK, map[string]any{"negocio": sesion.Negocio{ID: f.negocio, Slug: f.Slug, Nombre: f.Negocio, Rol: rol}})
	return nil
}
