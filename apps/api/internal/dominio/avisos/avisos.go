// Package avisos: la cola de mails (outbox), la campanita del panel, las preferencias del cliente y
// los trabajos del worker que encolan avisos (recordatorios, ausentes, resumen diario, purga).
//
// Todo aviso se encola en una transacción con el negocio fijado (Encolar) y lo manda el worker
// (Enviador). El mail se arma al mandarlo, con los datos de ese momento: un turno que se canceló
// antes de que saliera su recordatorio lo descarta en vez de mandarlo.
package avisos

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/auditoria"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/db"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/sesion"
)

// Tipos de aviso (la columna avisos.tipo).
const (
	Reservado    = "reservado"
	Recordatorio = "recordatorio"
	Cancelado    = "cancelado"
	Reprogramado = "reprogramado"
	Resena       = "resena"
	Espera       = "espera"
	Invitacion   = "invitacion"
	Resumen      = "resumen"
	// Los de otros dominios, que se arman con su Armador: los recordatorios automáticos
	// (recordatorios) y el reporte mensual (reportes).
	Cumpleanios = "cumpleanios"
	Extraniamos = "extraniamos"
	Mensual     = "mensual"
)

// Aviso a encolar. Para puede ir vacío en los de un turno (se usa el email del cliente al armarlo).
// Clave (opcional) evita encolar dos veces lo mismo en el negocio.
type Aviso struct {
	Tipo, Para, TurnoID, Clave string
	Datos                      map[string]any
}

// Encolar agrega el aviso en la transacción (con el negocio fijado), para mandarse desde ahora.
// Devuelve false si ya había uno con la misma clave.
func Encolar(ctx context.Context, tx pgx.Tx, a Aviso, ahora time.Time) (bool, error) {
	if a.Datos == nil {
		a.Datos = map[string]any{}
	}
	datos, err := json.Marshal(a.Datos)
	if err != nil {
		return false, err
	}
	tag, err := tx.Exec(ctx, `INSERT INTO avisos (negocio_id, tipo, para, turno_id, datos, clave, proximo_intento)
		VALUES (app.negocio_actual(), $1, $2, nullif($3, '')::uuid, $4, nullif($5, ''), $6)
		ON CONFLICT (negocio_id, clave) DO NOTHING`, a.Tipo, a.Para, a.TurnoID, datos, a.Clave, ahora)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() == 1, nil
}

// Tipos de la campanita (la columna notificaciones.tipo).
const (
	NotiReserva        = "reserva"
	NotiCancelacion    = "cancelacion"
	NotiReprogramacion = "reprogramacion"
	NotiResena         = "resena"
	NotiEspera         = "espera"
	NotiEquipo         = "equipo"
	NotiAusente        = "ausente"
	NotiAvisoFallido   = "aviso_fallido"
)

// Notificacion de la campanita.
type Notificacion struct {
	Tipo, Titulo, Texto, TurnoID string
}

// Notificar agrega una notificación a la campanita del negocio fijado.
func Notificar(ctx context.Context, tx pgx.Tx, n Notificacion) error {
	_, err := tx.Exec(ctx, `INSERT INTO notificaciones (negocio_id, tipo, titulo, texto, turno_id)
		VALUES (app.negocio_actual(), $1, left($2, 120), left($3, 500), nullif($4, '')::uuid)`, n.Tipo, n.Titulo, n.Texto, n.TurnoID)
	return err
}

// Token para los links públicos (reseña, invitación, baja): nadie adivina el de otro.
func Token() string {
	b := make([]byte, 18)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

// Links del front (apps/web) que van en los mails.
func LinkTurno(sitio, codigo string) string       { return sitio + "/turno/" + codigo }
func LinkReserva(sitio, slug string) string       { return sitio + "/" + slug }
func LinkResena(sitio, token string) string       { return sitio + "/resena/" + token }
func LinkInvitacion(sitio, token string) string   { return sitio + "/invitacion/" + token }
func LinkPreferencias(sitio, token string) string { return sitio + "/mi-cuenta/avisos/" + token }
func LinkAgenda(sitio, fecha string) string       { return sitio + "/panel?fecha=" + fecha }
func LinkOferta(sitio, token string) string       { return sitio + "/espera/" + token }
func LinkMensual(sitio, mes string) string        { return sitio + "/panel/reportes/mensual?mes=" + mes }

// Liberado es un horario que quedó libre (un turno cancelado, o el horario viejo de uno
// reprogramado): la lista de espera lo escucha.
type Liberado struct {
	ServicioID, ProfesionalID string
	Servicio, Profesional     string
	Inicio                    time.Time
	Zona                      *time.Location
	// Ahora: la hora del cambio (el reloj del servicio de avisos).
	Ahora time.Time
}

// Servicio de avisos. Ahora es el reloj (tests); nil: time.Now.
type Servicio struct {
	Pool   *db.Pool
	Sesion *sesion.Servicio
	Ahora  func() time.Time
	// AlLiberarse se llama, en la misma transacción que el aviso, cuando un cambio de un turno deja un
	// horario libre (lo engancha la lista de espera). nil: nadie escucha.
	AlLiberarse func(ctx context.Context, tx pgx.Tx, l Liberado) error
}

func (s *Servicio) ahora() time.Time {
	if s.Ahora != nil {
		return s.Ahora()
	}
	return time.Now()
}

// Avisar es la función para turnos.Servicio.Avisar: encola el mail que corresponde al evento, suma
// la campanita si lo hizo el cliente, avisa a la lista de espera si se liberó un horario y, si vino
// del panel, lo anota en la auditoría. Corre después de que turnos confirmó el cambio, en su propia
// transacción; un error queda en el log (el cambio del turno ya está hecho).
func (s *Servicio) Avisar(ctx context.Context, negocioID, turnoID, evento string) {
	err := s.Pool.EnNegocio(ctx, negocioID, func(tx pgx.Tx) error {
		return s.avisar(ctx, tx, turnoID, evento, s.ahora())
	})
	if err != nil {
		slog.Error("avisos: no se pudo encolar el aviso del turno", "negocio", negocioID, "turno", turnoID, "evento", evento, "error", err)
	}
}

func (s *Servicio) avisar(ctx context.Context, tx pgx.Tx, turnoID, evento string, ahora time.Time) error {
	t, err := leerTurno(ctx, tx, turnoID)
	if err != nil {
		return err
	}
	n, err := leerNegocio(ctx, tx)
	if err != nil {
		return err
	}
	quien := sesion.De(ctx).ClerkID
	cuando := Fecha(t.Inicio.In(n.zona)) + " a las " + Hora(t.Inicio.In(n.zona))
	inicio := map[string]any{"inicio": t.Inicio.UTC().Format(time.RFC3339Nano)}
	switch evento {
	case "reservado", "reservado_espera":
		if _, err := Encolar(ctx, tx, Aviso{Tipo: Reservado, TurnoID: t.ID, Clave: "reservado:" + t.ID, Datos: inicio}, ahora); err != nil {
			return err
		}
		// Lo tomó alguien de la lista de espera desde su link: la campanita cuenta que se llenó el hueco.
		if evento == "reservado_espera" {
			if err := Notificar(ctx, tx, Notificacion{Tipo: NotiEspera, TurnoID: t.ID,
				Titulo: "Se llenó el hueco de las " + Hora(t.Inicio.In(n.zona)),
				Texto:  t.ClienteNombre + " lo tomó desde la lista de espera: " + t.Servicio + " con " + t.Profesional + ", el " + cuando + "."}); err != nil {
				return err
			}
		} else if quien == "" {
			if err := Notificar(ctx, tx, Notificacion{Tipo: NotiReserva, TurnoID: t.ID, Titulo: "Nueva reserva de " + t.ClienteNombre,
				Texto: t.Servicio + " con " + t.Profesional + ", el " + cuando + "."}); err != nil {
				return err
			}
		}
	case "cancelado", "cancelado_cliente":
		if _, err := Encolar(ctx, tx, Aviso{Tipo: Cancelado, TurnoID: t.ID, Clave: "cancelado:" + t.ID, Datos: inicio}, ahora); err != nil {
			return err
		}
		if evento == "cancelado_cliente" || quien == "" {
			if err := Notificar(ctx, tx, Notificacion{Tipo: NotiCancelacion, TurnoID: t.ID, Titulo: t.ClienteNombre + " canceló su turno",
				Texto: t.Servicio + " con " + t.Profesional + ", el " + cuando + "."}); err != nil {
				return err
			}
		}
		if err := s.liberar(ctx, tx, t, t.Inicio, n, ahora); err != nil {
			return err
		}
	case "reprogramado":
		// El horario que tenía: el del último aviso del turno (turnos ya guardó el nuevo).
		anterior, err := ultimoInicio(ctx, tx, t.ID)
		if err != nil {
			return err
		}
		datos := map[string]any{"inicio": inicio["inicio"]}
		if anterior != nil {
			datos["anterior"] = anterior.UTC().Format(time.RFC3339Nano)
		}
		if _, err := Encolar(ctx, tx, Aviso{Tipo: Reprogramado, TurnoID: t.ID, Datos: datos}, ahora); err != nil {
			return err
		}
		if quien == "" {
			if err := Notificar(ctx, tx, Notificacion{Tipo: NotiReprogramacion, TurnoID: t.ID, Titulo: t.ClienteNombre + " cambió su turno",
				Texto: t.Servicio + " con " + t.Profesional + ", ahora el " + cuando + "."}); err != nil {
				return err
			}
		}
		if anterior != nil && !anterior.Equal(t.Inicio) {
			if err := s.liberar(ctx, tx, t, *anterior, n, ahora); err != nil {
				return err
			}
		}
	}
	if quien != "" {
		return auditoria.Anotar(ctx, tx, auditoria.Registro{Quien: quien, Accion: "turno." + evento, Entidad: "turno", EntidadID: t.ID,
			Detalle: map[string]any{"estado": t.Estado, "inicio": t.Inicio, "cliente": t.ClienteNombre}})
	}
	return nil
}

func (s *Servicio) liberar(ctx context.Context, tx pgx.Tx, t turno, inicio time.Time, n negocio, ahora time.Time) error {
	if s.AlLiberarse == nil || t.ServicioID == nil || !inicio.After(ahora) {
		return nil
	}
	l := Liberado{ServicioID: *t.ServicioID, Servicio: t.Servicio, Profesional: t.Profesional, Inicio: inicio, Zona: n.zona, Ahora: ahora}
	if t.ProfesionalID != nil {
		l.ProfesionalID = *t.ProfesionalID
	}
	return s.AlLiberarse(ctx, tx, l)
}

func ultimoInicio(ctx context.Context, tx pgx.Tx, turnoID string) (*time.Time, error) {
	var v string
	err := tx.QueryRow(ctx, `SELECT datos->>'inicio' FROM avisos
		WHERE turno_id = $1 AND tipo IN ('reservado', 'reprogramado') AND datos ? 'inicio'
		ORDER BY creado_en DESC LIMIT 1`, turnoID).Scan(&v)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	t, err := time.Parse(time.RFC3339Nano, v)
	if err != nil {
		return nil, nil
	}
	return &t, nil
}

// turno con lo que necesitan los mails.
type turno struct {
	ID, Codigo                string
	ServicioID, ProfesionalID *string
	Servicio, Profesional     string
	Estado, Origen            string
	Inicio, Fin               time.Time
	Precio                    int64
	ClienteID, ClienteNombre  string
	ClienteEmail              string
}

func leerTurno(ctx context.Context, tx pgx.Tx, id string) (turno, error) {
	var t turno
	err := tx.QueryRow(ctx, `SELECT t.id::text, t.codigo, t.servicio_id::text, t.profesional_id::text, t.servicio, t.profesional,
		t.estado, t.origen, t.inicio, t.fin, t.precio, c.id::text, c.nombre, c.email
		FROM turnos t JOIN clientes c ON c.id = t.cliente_id WHERE t.id = $1`, id).Scan(
		&t.ID, &t.Codigo, &t.ServicioID, &t.ProfesionalID, &t.Servicio, &t.Profesional, &t.Estado, &t.Origen,
		&t.Inicio, &t.Fin, &t.Precio, &t.ClienteID, &t.ClienteNombre, &t.ClienteEmail)
	return t, err
}

// negocio fijado, con lo que necesitan los mails.
type negocio struct {
	Nombre, Slug, Direccion, Ciudad, Telefono string
	zona                                      *time.Location
}

func (n negocio) lugar() string {
	switch {
	case n.Direccion != "" && n.Ciudad != "":
		return n.Direccion + ", " + n.Ciudad
	case n.Direccion != "":
		return n.Direccion
	}
	return n.Ciudad
}

// NegocioDelMail: lo que necesita un mail del negocio fijado (para los armadores de otros dominios).
type NegocioDelMail struct {
	Nombre, Slug string
	Zona         *time.Location
}

// LeerNegocioDelMail del negocio fijado.
func LeerNegocioDelMail(ctx context.Context, tx pgx.Tx) (NegocioDelMail, error) {
	n, err := leerNegocio(ctx, tx)
	return NegocioDelMail{Nombre: n.Nombre, Slug: n.Slug, Zona: n.zona}, err
}

// PrimerNombre: "Lucía Fernández" → "Lucía".
func PrimerNombre(nombre string) string { return primerNombre(nombre) }

func leerNegocio(ctx context.Context, tx pgx.Tx) (negocio, error) {
	var n negocio
	var zona string
	err := tx.QueryRow(ctx, `SELECT nombre, slug, direccion, ciudad, telefono, zona_horaria FROM negocios WHERE id = app.negocio_actual()`).
		Scan(&n.Nombre, &n.Slug, &n.Direccion, &n.Ciudad, &n.Telefono, &zona)
	if err != nil {
		return n, err
	}
	if n.zona, err = time.LoadLocation(zona); err != nil {
		n.zona, _ = time.LoadLocation("America/Argentina/Buenos_Aires")
	}
	return n, nil
}

// Preferencias de un cliente (con su token, que se crea la primera vez que se pide). La
// confirmación de un turno nuevo no depende de ninguna: es el comprobante y sale siempre.
type Preferencias struct {
	Token string `json:"-"`
	// Email: los cambios en sus turnos (cancelado, reprogramado).
	Email         bool `json:"email"`
	Recordatorios bool `json:"recordatorios"`
	Resena        bool `json:"resena"`
	ListaEspera   bool `json:"listaEspera"`
	// Novedades: los recordatorios automáticos del negocio (cumpleaños, «hace mucho que no venís»).
	Novedades bool `json:"novedades"`
}

// PreferenciasDe devuelve las preferencias del cliente en el negocio fijado (las crea si no hay).
func PreferenciasDe(ctx context.Context, tx pgx.Tx, clienteID string) (Preferencias, error) {
	var p Preferencias
	if _, err := tx.Exec(ctx, `INSERT INTO preferencias (cliente_id, negocio_id, token) VALUES ($1, app.negocio_actual(), $2)
		ON CONFLICT (cliente_id) DO NOTHING`, clienteID, Token()); err != nil {
		return p, err
	}
	err := tx.QueryRow(ctx, "SELECT token, email, recordatorios, resena, lista_espera, novedades FROM preferencias WHERE cliente_id = $1", clienteID).
		Scan(&p.Token, &p.Email, &p.Recordatorios, &p.Resena, &p.ListaEspera, &p.Novedades)
	return p, err
}
