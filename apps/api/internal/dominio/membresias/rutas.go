package membresias

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"

	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/catalogo"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/turnos"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/problema"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/sesion"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/web"
)

// Mostrador: quienes atienden a los socios (dan de alta, ajustan, renuevan). Los planes, Gestion.
var Mostrador = []string{sesion.Duenio, sesion.Encargado, sesion.Recepcion}

// Rutas del dominio.
func (s *Servicio) Rutas(mux *http.ServeMux) {
	ses := s.Sesion
	panel := func(roles []string, h web.Manejador) http.HandlerFunc { return ses.Panel(roles, web.Envolver(h)) }
	escribir := func(roles []string, h web.Manejador) http.HandlerFunc {
		return ses.Panel(roles, s.escritura(web.Envolver(h)))
	}
	mux.HandleFunc("GET /v1/panel/membresias/planes", panel(Mostrador, s.planes))
	mux.HandleFunc("POST /v1/panel/membresias/planes", escribir(sesion.Gestion, s.crearPlan))
	mux.HandleFunc("PATCH /v1/panel/membresias/planes/{id}", escribir(sesion.Gestion, s.editarPlan))
	mux.HandleFunc("DELETE /v1/panel/membresias/planes/{id}", escribir(sesion.Gestion, s.borrarPlan))
	mux.HandleFunc("GET /v1/panel/membresias/socios", panel(Mostrador, s.socios))
	mux.HandleFunc("POST /v1/panel/membresias/socios", escribir(Mostrador, s.alta))
	mux.HandleFunc("GET /v1/panel/membresias/socios/{id}", panel(Mostrador, s.socio))
	mux.HandleFunc("PATCH /v1/panel/membresias/socios/{id}", escribir(Mostrador, s.cambiar))
	// La página pública muestra los planes activos.
	mux.HandleFunc("GET /v1/p/{slug}/membresias", web.Envolver(s.planesPublicos))
	mux.HandleFunc("GET /v1/p/{slug}/mi-membresia", ses.Opcional(web.Envolver(s.miMembresia)))
}

func (s *Servicio) enNegocio(r *http.Request, fn func(context.Context, pgx.Tx) error) error {
	ctx := r.Context()
	return s.Pool.EnNegocio(ctx, sesion.De(ctx).NegocioID, func(tx pgx.Tx) error { return fn(ctx, tx) })
}

func listarPlanes(ctx context.Context, tx pgx.Tx, soloActivos bool) ([]Plan, error) {
	q := "SELECT " + columnasPlan + " FROM membresia_planes"
	if soloActivos {
		q += " WHERE activo"
	}
	filas, err := tx.Query(ctx, q+" ORDER BY orden, nombre")
	if err != nil {
		return nil, err
	}
	defer filas.Close()
	out := []Plan{}
	for filas.Next() {
		var p Plan
		if err := escanearPlan(filas, &p); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, filas.Err()
}

func (s *Servicio) planes(w http.ResponseWriter, r *http.Request) error {
	var out []Plan
	err := s.enNegocio(r, func(ctx context.Context, tx pgx.Tx) (err error) {
		out, err = listarPlanes(ctx, tx, false)
		return err
	})
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusOK, map[string]any{"planes": out})
	return nil
}

func validarPlan(ctx context.Context, tx pgx.Tx, p *Plan) error {
	p.Nombre, p.Descripcion = strings.TrimSpace(p.Nombre), strings.TrimSpace(p.Descripcion)
	switch n := utf8.RuneCountInString(p.Nombre); {
	case n < 2 || n > 60:
		return problema.Invalido("Poné un nombre (de 2 a 60).", problema.Campo{Ruta: "nombre", Mensaje: "largo inválido"})
	case utf8.RuneCountInString(p.Descripcion) > 400:
		return problema.Invalido("La descripción es muy larga (hasta 400).", problema.Campo{Ruta: "descripcion", Mensaje: "largo inválido"})
	case p.Precio <= 0 || p.Precio > 100_000_000:
		return problema.Invalido("El precio tiene que ser mayor a cero.", problema.Campo{Ruta: "precio", Mensaje: "fuera de rango"})
	case p.Visitas < 0 || p.Visitas > 1000:
		return problema.Invalido("Las visitas van de 0 a 1000.", problema.Campo{Ruta: "visitas", Mensaje: "fuera de rango"})
	case p.DuracionDias < 1 || p.DuracionDias > 365:
		return problema.Invalido("La duración va de 1 a 365 días.", problema.Campo{Ruta: "duracionDias", Mensaje: "fuera de rango"})
	case p.PrioridadDias < 0 || p.PrioridadDias > 90:
		return problema.Invalido("La prioridad va de 0 a 90 días.", problema.Campo{Ruta: "prioridadDias", Mensaje: "fuera de rango"})
	case len(p.Beneficios) > 12:
		return problema.Invalido("Hasta 12 beneficios.", problema.Campo{Ruta: "beneficios", Mensaje: "demasiados"})
	case p.TopeSocios != nil && (*p.TopeSocios < 1 || *p.TopeSocios > 100_000):
		return problema.Invalido("El cupo de socios tiene que ser al menos uno.", problema.Campo{Ruta: "topeSocios", Mensaje: "fuera de rango"})
	case p.Orden < 0 || p.Orden > 999:
		return problema.Invalido("El orden va de 0 a 999.", problema.Campo{Ruta: "orden", Mensaje: "fuera de rango"})
	}
	for i, b := range p.Beneficios {
		p.Beneficios[i] = strings.TrimSpace(b)
		if n := utf8.RuneCountInString(p.Beneficios[i]); n < 1 || n > 120 {
			return problema.Invalido("Cada beneficio va de 1 a 120 letras.", problema.Campo{Ruta: "beneficios", Mensaje: "largo inválido"})
		}
	}
	if p.Beneficios == nil {
		p.Beneficios = []string{}
	}
	if p.Servicios == nil {
		p.Servicios = []string{}
	}
	// Los servicios cubiertos tienen que ser de este negocio (el array no tiene FK).
	for _, id := range p.Servicios {
		if _, err := catalogo.LeerServicio(ctx, tx, id); err != nil {
			return problema.Invalido("Algún servicio no existe.", problema.Campo{Ruta: "servicios", Mensaje: "no existe"})
		}
	}
	return nil
}

func (s *Servicio) guardarPlan(ctx context.Context, tx pgx.Tx, p *Plan) error {
	if err := validarPlan(ctx, tx, p); err != nil {
		return err
	}
	if p.ID == "" {
		return escanearPlan(tx.QueryRow(ctx, `INSERT INTO membresia_planes (negocio_id, nombre, descripcion, precio, visitas, duracion_dias,
			prioridad_dias, beneficios, tope_socios, servicios, activo, orden)
			VALUES (app.negocio_actual(), $1, $2, $3, $4, $5, $6, $7, $8, $9::uuid[], $10, $11) RETURNING `+columnasPlan,
			p.Nombre, p.Descripcion, p.Precio, p.Visitas, p.DuracionDias, p.PrioridadDias, p.Beneficios, p.TopeSocios, p.Servicios,
			p.Activo, p.Orden), p)
	}
	return escanearPlan(tx.QueryRow(ctx, `UPDATE membresia_planes SET nombre = $2, descripcion = $3, precio = $4, visitas = $5,
		duracion_dias = $6, prioridad_dias = $7, beneficios = $8, tope_socios = $9, servicios = $10::uuid[], activo = $11, orden = $12
		WHERE id = $1 RETURNING `+columnasPlan, p.ID, p.Nombre, p.Descripcion, p.Precio, p.Visitas, p.DuracionDias, p.PrioridadDias,
		p.Beneficios, p.TopeSocios, p.Servicios, p.Activo, p.Orden), p)
}

func (s *Servicio) crearPlan(w http.ResponseWriter, r *http.Request) error {
	p := Plan{DuracionDias: 30, Activo: true}
	if err := web.Leer(w, r, &p); err != nil {
		return err
	}
	p.ID = ""
	if err := s.enNegocio(r, func(ctx context.Context, tx pgx.Tx) error { return s.guardarPlan(ctx, tx, &p) }); err != nil {
		return err
	}
	web.JSON(w, http.StatusCreated, p)
	return nil
}

// editarPlan: lo que no viene en el cuerpo queda como estaba.
func (s *Servicio) editarPlan(w http.ResponseWriter, r *http.Request) error {
	var p Plan
	err := s.enNegocio(r, func(ctx context.Context, tx pgx.Tx) error {
		if err := escanearPlan(tx.QueryRow(ctx, "SELECT "+columnasPlan+" FROM membresia_planes WHERE id = $1 FOR UPDATE", r.PathValue("id")), &p); err != nil {
			return err
		}
		id := p.ID
		if err := web.Leer(w, r, &p); err != nil {
			return err
		}
		p.ID = id
		return s.guardarPlan(ctx, tx, &p)
	})
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusOK, p)
	return nil
}

// borrarPlan: uno con socios se desactiva (las membresías lo siguen nombrando).
func (s *Servicio) borrarPlan(w http.ResponseWriter, r *http.Request) error {
	err := s.enNegocio(r, func(ctx context.Context, tx pgx.Tx) error {
		var socios bool
		if err := tx.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM membresias WHERE plan_id = $1)", r.PathValue("id")).Scan(&socios); err != nil {
			return err
		}
		q := "DELETE FROM membresia_planes WHERE id = $1"
		if socios {
			q = "UPDATE membresia_planes SET activo = false WHERE id = $1"
		}
		tag, err := tx.Exec(ctx, q, r.PathValue("id"))
		if err == nil && tag.RowsAffected() == 0 {
			return pgx.ErrNoRows
		}
		return err
	})
	if err != nil {
		return err
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}

// Socio: una membresía como la lista el panel.
type Socio struct {
	ID        string    `json:"id"`
	PlanID    string    `json:"planId"`
	Plan      string    `json:"plan"`
	Visitas   int       `json:"visitasDelPlan"`
	ClienteID string    `json:"clienteId"`
	Cliente   string    `json:"cliente"`
	Telefono  string    `json:"telefono"`
	Desde     time.Time `json:"desde"`
	Hasta     time.Time `json:"hasta"`
	// Estado: activa, vencida (por el reloj) o cancelada.
	Estado string `json:"estado"`
	Quedan int    `json:"quedan"`
	Nota   string `json:"nota"`
}

// Credito: un movimiento de visitas.
type Credito struct {
	Cantidad int       `json:"cantidad"`
	Motivo   string    `json:"motivo"`
	TurnoID  *string   `json:"turnoId"`
	ClerkID  *string   `json:"clerkId"`
	CreadoEn time.Time `json:"creadoEn"`
}

const columnasSocio = `m.id::text, p.id::text, p.nombre, p.visitas, c.id::text, c.nombre, c.telefono, m.desde, m.hasta,
	CASE WHEN m.estado = 'cancelada' THEN 'cancelada' WHEN m.hasta <= $1 THEN 'vencida' ELSE 'activa' END,
	(SELECT coalesce(sum(k.cantidad), 0) FROM membresia_creditos k WHERE k.membresia_id = m.id), m.nota`

const desdeSocios = ` FROM membresias m JOIN membresia_planes p ON p.id = m.plan_id JOIN clientes c ON c.id = m.cliente_id`

func escanearSocio(f pgx.Row, x *Socio) error {
	return f.Scan(&x.ID, &x.PlanID, &x.Plan, &x.Visitas, &x.ClienteID, &x.Cliente, &x.Telefono, &x.Desde, &x.Hasta, &x.Estado, &x.Quedan, &x.Nota)
}

func leerSocio(ctx context.Context, tx pgx.Tx, id string, ahora time.Time) (Socio, error) {
	var x Socio
	err := escanearSocio(tx.QueryRow(ctx, "SELECT "+columnasSocio+desdeSocios+" WHERE m.id = $2", ahora, id), &x)
	return x, err
}

func (s *Servicio) socios(w http.ResponseWriter, r *http.Request) error {
	estado := r.URL.Query().Get("estado")
	if estado == "" {
		estado = "activa"
	}
	cond := map[string]string{
		"activa":    "m.estado = 'activa' AND m.hasta > $1",
		"vencida":   "m.estado = 'activa' AND m.hasta <= $1",
		"cancelada": "m.estado = 'cancelada'",
		"todas":     "true",
	}[estado]
	if cond == "" {
		return problema.Invalido("El estado es activa, vencida, cancelada o todas.")
	}
	out := []Socio{}
	err := s.enNegocio(r, func(ctx context.Context, tx pgx.Tx) error {
		filas, err := tx.Query(ctx, "SELECT "+columnasSocio+desdeSocios+" WHERE "+cond+" ORDER BY m.hasta LIMIT 500", s.ahora())
		if err != nil {
			return err
		}
		defer filas.Close()
		for filas.Next() {
			var x Socio
			if err := escanearSocio(filas, &x); err != nil {
				return err
			}
			out = append(out, x)
		}
		return filas.Err()
	})
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusOK, map[string]any{"socios": out})
	return nil
}

// limpiarTelefono como en turnos: dígitos y un + inicial, así el socio es el mismo cliente que reserva.
func limpiarTelefono(t string) string {
	var b strings.Builder
	for i, r := range strings.TrimSpace(t) {
		if (r >= '0' && r <= '9') || (r == '+' && i == 0) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// alta de un socio: un cliente existente, o uno buscado por teléfono, o uno nuevo. Las visitas del
// período se dan en la misma transacción.
func (s *Servicio) alta(w http.ResponseWriter, r *http.Request) error {
	var in struct {
		PlanID    string `json:"planId"`
		ClienteID string `json:"clienteId"`
		Nombre    string `json:"nombre"`
		Telefono  string `json:"telefono"`
		Email     string `json:"email"`
		Nota      string `json:"nota"`
	}
	if err := web.Leer(w, r, &in); err != nil {
		return err
	}
	in.Nombre, in.Telefono, in.Email, in.Nota = strings.TrimSpace(in.Nombre), limpiarTelefono(in.Telefono),
		strings.ToLower(strings.TrimSpace(in.Email)), strings.TrimSpace(in.Nota)
	if utf8.RuneCountInString(in.Nota) > 300 {
		return problema.Invalido("La nota es muy larga (hasta 300).")
	}
	ahora := s.ahora()
	ses := sesion.De(r.Context())
	var x Socio
	err := s.enNegocio(r, func(ctx context.Context, tx pgx.Tx) error {
		var p Plan
		err := escanearPlan(tx.QueryRow(ctx, "SELECT "+columnasPlan+" FROM membresia_planes WHERE id = $1 AND activo", in.PlanID), &p)
		if errors.Is(err, pgx.ErrNoRows) {
			return problema.NoEncontrado("Ese plan no existe o no está activo.")
		}
		if err != nil {
			return err
		}
		if p.TopeSocios != nil {
			var activos int
			if err := tx.QueryRow(ctx, "SELECT count(*) FROM membresias WHERE plan_id = $1 AND estado = 'activa' AND hasta > $2",
				p.ID, ahora).Scan(&activos); err != nil {
				return err
			}
			if activos >= *p.TopeSocios {
				return problema.Conflicto("cupo_lleno", "Este plan ya llegó a su cupo de socios.")
			}
		}
		cliente := in.ClienteID
		switch {
		case cliente != "":
			if err := tx.QueryRow(ctx, "SELECT id::text FROM clientes WHERE id = $1 AND suprimido_en IS NULL", cliente).Scan(&cliente); err != nil {
				return problema.NoEncontrado("Ese cliente no existe.")
			}
		case in.Telefono != "":
			if n := utf8.RuneCountInString(in.Nombre); n < 1 || n > 80 {
				return problema.Invalido("Escribí el nombre del socio.", problema.Campo{Ruta: "nombre", Mensaje: "requerido"})
			}
			if err := tx.QueryRow(ctx, `INSERT INTO clientes (negocio_id, nombre, telefono, email) VALUES (app.negocio_actual(), $1, $2, $3)
				ON CONFLICT (negocio_id, telefono) WHERE telefono <> '' DO UPDATE SET email = CASE WHEN excluded.email <> '' THEN excluded.email ELSE clientes.email END
				RETURNING id::text`, in.Nombre, in.Telefono, in.Email).Scan(&cliente); err != nil {
				return err
			}
		default:
			return problema.Invalido("Elegí un cliente o dejá su teléfono.", problema.Campo{Ruta: "telefono", Mensaje: "requerido"})
		}
		var ya bool
		if err := tx.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM membresias WHERE cliente_id = $1 AND estado = 'activa' AND hasta > $2)",
			cliente, ahora).Scan(&ya); err != nil {
			return err
		}
		if ya {
			return problema.Conflicto("ya_socio", "Esa persona ya tiene una membresía activa.")
		}
		var id string
		if err := tx.QueryRow(ctx, `INSERT INTO membresias (negocio_id, plan_id, cliente_id, desde, hasta, nota)
			VALUES (app.negocio_actual(), $1, $2, $3, $4, $5) RETURNING id::text`,
			p.ID, cliente, ahora, ahora.AddDate(0, 0, p.DuracionDias), in.Nota).Scan(&id); err != nil {
			return err
		}
		if p.Visitas > 0 {
			if _, err := tx.Exec(ctx, `INSERT INTO membresia_creditos (negocio_id, membresia_id, cantidad, motivo, clerk_id)
				VALUES (app.negocio_actual(), $1, $2, $3, $4)`, id, p.Visitas, "Alta en "+p.Nombre, ses.ClerkID); err != nil {
				return err
			}
		}
		x, err = leerSocio(ctx, tx, id, ahora)
		return err
	})
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusCreated, x)
	return nil
}

// SocioConCreditos: la membresía con su historial completo.
type SocioConCreditos struct {
	Socio
	Creditos []Credito `json:"creditos"`
}

func leerConCreditos(ctx context.Context, tx pgx.Tx, id string, ahora time.Time) (SocioConCreditos, error) {
	var out SocioConCreditos
	var err error
	if out.Socio, err = leerSocio(ctx, tx, id, ahora); err != nil {
		return out, err
	}
	filas, err := tx.Query(ctx, `SELECT cantidad, motivo, turno_id::text, clerk_id, creado_en FROM membresia_creditos
		WHERE membresia_id = $1 ORDER BY creado_en DESC LIMIT 100`, id)
	if err != nil {
		return out, err
	}
	out.Creditos, err = pgx.CollectRows(filas, pgx.RowToStructByPos[Credito])
	return out, err
}

func (s *Servicio) socio(w http.ResponseWriter, r *http.Request) error {
	var out SocioConCreditos
	err := s.enNegocio(r, func(ctx context.Context, tx pgx.Tx) (err error) {
		out, err = leerConCreditos(ctx, tx, r.PathValue("id"), s.ahora())
		return err
	})
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusOK, out)
	return nil
}

// cambiar un socio: ajustar visitas (+/-), renovar el período (desde hoy, con las visitas del plan)
// o cancelar / reactivar.
func (s *Servicio) cambiar(w http.ResponseWriter, r *http.Request) error {
	var in struct {
		Ajustar int     `json:"ajustar"`
		Motivo  string  `json:"motivo"`
		Renovar bool    `json:"renovar"`
		Estado  *string `json:"estado"`
	}
	if err := web.Leer(w, r, &in); err != nil {
		return err
	}
	in.Motivo = strings.TrimSpace(in.Motivo)
	switch {
	case in.Ajustar < -100 || in.Ajustar > 100:
		return problema.Invalido("El ajuste va de -100 a 100.", problema.Campo{Ruta: "ajustar", Mensaje: "fuera de rango"})
	case utf8.RuneCountInString(in.Motivo) > 120:
		return problema.Invalido("El motivo es muy largo (hasta 120).", problema.Campo{Ruta: "motivo", Mensaje: "largo inválido"})
	case in.Estado != nil && *in.Estado != "activa" && *in.Estado != "cancelada":
		return problema.Invalido("El estado es activa o cancelada.", problema.Campo{Ruta: "estado", Mensaje: "inválido"})
	}
	ahora := s.ahora()
	clerk := sesion.De(r.Context()).ClerkID
	var out SocioConCreditos
	err := s.enNegocio(r, func(ctx context.Context, tx pgx.Tx) error {
		var planID string
		var visitas, dias int
		var nombre string
		err := tx.QueryRow(ctx, `SELECT m.plan_id::text, p.visitas, p.duracion_dias, p.nombre FROM membresias m
			JOIN membresia_planes p ON p.id = m.plan_id WHERE m.id = $1 FOR UPDATE OF m`, r.PathValue("id")).Scan(&planID, &visitas, &dias, &nombre)
		if err != nil {
			return err
		}
		id := r.PathValue("id")
		if in.Ajustar != 0 {
			saldo, err := Saldo(ctx, tx, id)
			if err != nil {
				return err
			}
			// Un saldo negativo no es una deuda que alguien pueda cobrar.
			if saldo+in.Ajustar < 0 {
				return problema.Invalido("No le quedan tantas visitas para restar.")
			}
			motivo := in.Motivo
			if motivo == "" {
				motivo = "Ajuste manual"
			}
			if _, err := tx.Exec(ctx, `INSERT INTO membresia_creditos (negocio_id, membresia_id, cantidad, motivo, clerk_id)
				VALUES (app.negocio_actual(), $1, $2, $3, $4)`, id, in.Ajustar, motivo, clerk); err != nil {
				return err
			}
		}
		if in.Renovar {
			if _, err := tx.Exec(ctx, "UPDATE membresias SET desde = $2, hasta = $3, estado = 'activa' WHERE id = $1",
				id, ahora, ahora.AddDate(0, 0, dias)); err != nil {
				return err
			}
			if visitas > 0 {
				if _, err := tx.Exec(ctx, `INSERT INTO membresia_creditos (negocio_id, membresia_id, cantidad, motivo, clerk_id)
					VALUES (app.negocio_actual(), $1, $2, 'Renovación del período', $3)`, id, visitas, clerk); err != nil {
					return err
				}
			}
		}
		if in.Estado != nil {
			if _, err := tx.Exec(ctx, "UPDATE membresias SET estado = $2 WHERE id = $1", id, *in.Estado); err != nil {
				return err
			}
		}
		out, err = leerConCreditos(ctx, tx, id, ahora)
		return err
	})
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusOK, out)
	return nil
}

func (s *Servicio) planesPublicos(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	var id *string
	if err := s.Pool.EnTx(ctx, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, "SELECT app.negocio_publico($1)::text", r.PathValue("slug")).Scan(&id)
	}); err != nil {
		return err
	}
	if id == nil {
		return problema.NoEncontrado("No encontramos ese negocio.")
	}
	var out []Plan
	err := s.Pool.EnNegocio(ctx, *id, func(tx pgx.Tx) (err error) {
		out, err = listarPlanes(ctx, tx, true)
		return err
	})
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusOK, map[string]any{"planes": out})
	return nil
}

// MiMembresia: lo que ve en la reserva el socio que ingresó.
type MiMembresia struct {
	Plan      string    `json:"plan"`
	Hasta     time.Time `json:"hasta"`
	Saldo     int       `json:"saldo"`
	Servicios []string  `json:"servicios"`
}

// miMembresia responde la membresía vigente de quien ingresó (por su email verificado y el email de
// confianza del cliente), o null: sin sesión, sin membresía o en otro negocio.
func (s *Servicio) miMembresia(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	email := sesion.De(ctx).Persona.Email
	var out *MiMembresia
	if email != "" {
		var id *string
		if err := s.Pool.EnTx(ctx, func(tx pgx.Tx) error {
			return tx.QueryRow(ctx, "SELECT app.negocio_publico($1)::text", r.PathValue("slug")).Scan(&id)
		}); err != nil {
			return err
		}
		if id == nil {
			return problema.NoEncontrado("No encontramos ese negocio.")
		}
		err := s.Pool.EnNegocio(ctx, *id, func(tx pgx.Tx) error {
			cliente, err := turnos.SocioDe(ctx, tx, email, s.ahora())
			if err != nil {
				if p, ok := errors.AsType[*problema.Problema](err); ok && p.Codigo == "sin_membresia" {
					return nil
				}
				return err
			}
			var m MiMembresia
			var mid string
			if err := tx.QueryRow(ctx, `SELECT m.id::text, p.nombre, m.hasta, p.servicios::text[] FROM membresias m
				JOIN membresia_planes p ON p.id = m.plan_id
				WHERE m.cliente_id = $1 AND m.estado = 'activa' AND m.desde <= $2 AND m.hasta > $2
				ORDER BY m.hasta DESC LIMIT 1`, cliente, s.ahora()).Scan(&mid, &m.Plan, &m.Hasta, &m.Servicios); err != nil {
				return err
			}
			if m.Saldo, err = Saldo(ctx, tx, mid); err != nil {
				return err
			}
			out = &m
			return nil
		})
		if err != nil {
			return err
		}
	}
	web.JSON(w, http.StatusOK, map[string]any{"membresia": out})
	return nil
}
