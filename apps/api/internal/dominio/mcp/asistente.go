package mcp

import (
	"context"
	"encoding/json"
	"slices"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/turnos"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/problema"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/sesion"
)

// Operador: las herramientas del MCP para el asistente propio del panel (dominio asistente), sin
// pasar por HTTP ni por una clave. Corre con la sesión de quien pregunta y con lo que su rol puede
// hacer en el panel: un profesional, sólo su agenda.
type Operador struct {
	s *Servicio
	c *conexion
}

// permisosDeRol: lo mismo que el panel deja hacer a cada rol (los números son de la gestión; los
// datos de contacto, de quien atiende el mostrador).
func permisosDeRol(rol string) []string {
	switch rol {
	case sesion.Duenio, sesion.Encargado:
		return []string{VerAgenda, MoverTurnos, CancelarTurnos, VerClientes, VerNumeros}
	case sesion.Recepcion:
		return []string{VerAgenda, MoverTurnos, CancelarTurnos, VerClientes}
	case sesion.Profesional:
		return []string{VerAgenda, MoverTurnos, CancelarTurnos}
	}
	return nil
}

// Operador para la sesión del contexto (la que dejó sesion.Panel).
func (s *Servicio) Operador(ctx context.Context) (*Operador, error) {
	ses := sesion.De(ctx)
	nombre := strings.TrimSpace(ses.Persona.Nombre)
	if nombre == "" {
		nombre = "el asistente del panel"
	} else {
		nombre += " con el asistente"
	}
	c := &conexion{negocio: ses.NegocioID, nombre: nombre, quien: ses.ClerkID, permisos: permisosDeRol(ses.Rol),
		limitado: ses.Rol == sesion.Profesional, paraIA: true}
	err := s.Pool.EnNegocio(ctx, ses.NegocioID, func(tx pgx.Tx) error {
		var zona string
		if err := tx.QueryRow(ctx, "SELECT zona_horaria FROM negocios WHERE id = app.negocio_actual()").Scan(&zona); err != nil {
			return err
		}
		var err error
		if c.zona, err = time.LoadLocation(zona); err != nil {
			c.zona = time.FixedZone("UTC-3", -3*3600)
		}
		if !c.limitado {
			return nil
		}
		filas, err := tx.Query(ctx, `SELECT p.id::text FROM profesionales p JOIN miembros m ON m.id = p.miembro_id
			WHERE m.clerk_id = $1 ORDER BY p.orden, p.id`, ses.ClerkID)
		if err != nil {
			return err
		}
		c.propios, err = pgx.CollectRows(filas, pgx.RowTo[string])
		return err
	})
	if err != nil {
		return nil, err
	}
	return &Operador{s: s, c: c}, nil
}

// Zona horaria del negocio.
func (o *Operador) Zona() *time.Location { return o.c.zona }

// Limitado: es un profesional que ve sólo su agenda.
func (o *Operador) Limitado() bool { return o.c.limitado }

// Propios: los profesionales de la agenda de un profesional (rol).
func (o *Operador) Propios() []string { return o.c.propios }

// Permitido: ese profesional es de la agenda de quien pregunta.
func (o *Operador) Permitido(profesional string) bool { return o.c.permitido(profesional) }

// HerramientaPanel: una herramienta del MCP como la ve el asistente.
type HerramientaPanel struct {
	Nombre, Descripcion string
	Esquema             map[string]any
	Escribe             bool
}

// Herramientas que puede usar quien pregunta. La lista de espera es del mostrador.
func (o *Operador) Herramientas() []HerramientaPanel {
	var out []HerramientaPanel
	for _, h := range herramientas {
		if o.Puede(h.nombre) {
			out = append(out, HerramientaPanel{Nombre: h.nombre, Descripcion: h.descripcion, Esquema: h.esquema, Escribe: h.escribe})
		}
	}
	return out
}

// Puede usar esa herramienta.
func (o *Operador) Puede(nombre string) bool {
	h := buscarHerramienta(nombre)
	if h == nil || !o.c.puede(h.permiso) {
		return false
	}
	return !(o.c.limitado && nombre == "ver_lista_de_espera")
}

// Escribe: la herramienta cambia algo (va como propuesta).
func Escribe(nombre string) bool {
	h := buscarHerramienta(nombre)
	return h != nil && h.escribe
}

// Leer corre una herramienta de lectura.
func (o *Operador) Leer(ctx context.Context, nombre string, args json.RawMessage) (any, error) {
	h := buscarHerramienta(nombre)
	if h == nil || !o.Puede(nombre) {
		return nil, problema.SinPermiso("No podés usar «" + nombre + "».")
	}
	if h.escribe {
		return nil, problema.Invalido("«" + nombre + "» cambia algo: va como propuesta.")
	}
	if len(args) == 0 {
		args = json.RawMessage("{}")
	}
	res, _, err := h.correr(ctx, o.s, o.c, args)
	return res, err
}

// Hacer corre una herramienta que escribe (la propuesta ya confirmada). Pasa por el mismo chequeo de
// sólo lectura que el panel.
func (o *Operador) Hacer(ctx context.Context, nombre string, args json.RawMessage) (any, string, error) {
	h := buscarHerramienta(nombre)
	if h == nil || !o.Puede(nombre) {
		return nil, "", problema.SinPermiso("No podés usar «" + nombre + "».")
	}
	if err := o.s.puedeEscribir(ctx, h); err != nil {
		return nil, "", err
	}
	return h.correr(ctx, o.s, o.c, args)
}

// Explicar un error de una herramienta en una frase.
func Explicar(err error, herramienta string) string { return explicar(err, herramienta) }

// Previa de una acción: valida lo que se puede validar sin escribir (que el turno exista y sea de su
// agenda, que el horario esté libre) y arma la frase que ve la persona antes de confirmar. Devuelve
// los argumentos con los ids ya resueltos, que son los que se van a usar al confirmar.
func (o *Operador) Previa(ctx context.Context, nombre string, raw json.RawMessage) (string, json.RawMessage, error) {
	if !o.Puede(nombre) || !Escribe(nombre) {
		return "", nil, problema.SinPermiso("No podés usar «" + nombre + "».")
	}
	c, s := o.c, o.s
	switch nombre {
	case "confirmar_turno", "cancelar_turno":
		id, err := turnoID(raw)
		if err != nil {
			return "", nil, err
		}
		t, err := o.turno(ctx, id)
		if err != nil {
			return "", nil, err
		}
		estado := "confirmado"
		if nombre == "cancelar_turno" {
			estado = "cancelado"
		}
		if t.Estado == estado {
			return "", nil, problema.Conflicto("estado", "Ese turno ya está "+estado+".")
		}
		if !turnos.PuedePasar(t.Estado, estado) {
			return "", nil, problema.Conflicto("estado", "Un turno "+t.Estado+" no se puede pasar a "+estado+".")
		}
		if estado == "confirmado" {
			if err := s.Turnos.SePuedeConfirmar(ctx, c.negocio, t); err != nil {
				return "", nil, err
			}
		}
		frase := "Confirmar el turno de " + t.Cliente.Nombre + " (" + t.Servicio + " con " + t.Profesional + "), el " + c.cuando(t.Inicio) + "."
		if estado == "cancelado" {
			frase = "Cancelar el turno de " + t.Cliente.Nombre + " (" + t.Servicio + " con " + t.Profesional + "), el " + c.cuando(t.Inicio) +
				". Le avisamos al cliente y a la lista de espera."
		}
		return frase, mustJSON(map[string]any{"turnoId": t.ID}), nil
	case "reprogramar_turno":
		var in struct{ TurnoID, Fecha, Hora, ProfesionalID string }
		if err := leerArgs(raw, &in); err != nil {
			return "", nil, err
		}
		t, err := o.turno(ctx, strings.TrimSpace(in.TurnoID))
		if err != nil {
			return "", nil, err
		}
		inicio, err := c.momento(in.Fecha, in.Hora)
		if err != nil {
			return "", nil, err
		}
		_, prof, nomProf, err := s.resolverAmbos(ctx, c, "", in.ProfesionalID)
		if err != nil {
			return "", nil, err
		}
		if prof != "" && !c.permitido(prof) {
			return "", nil, problema.Invalido("Sólo podés mover el turno dentro de tu agenda.")
		}
		destino := prof
		if destino == "" && t.ProfesionalID != nil {
			destino, nomProf = *t.ProfesionalID, t.Profesional
		}
		if inicio.Equal(t.Inicio) && destino == deref(t.ProfesionalID) {
			return "", nil, problema.Invalido("El turno ya está en ese horario.")
		}
		// Si el horario nuevo no pisa el actual, tiene que estar libre (si lo pisa, lo decide reprogramar).
		dur := t.Fin.Sub(t.Inicio)
		pisa := inicio.Before(t.Fin) && t.Inicio.Before(inicio.Add(dur)) && destino == deref(t.ProfesionalID)
		if !pisa && t.ServicioID != nil {
			if err := o.libre(ctx, *t.ServicioID, destino, inicio); err != nil {
				return "", nil, err
			}
		}
		frase := "Mover el turno de " + t.Cliente.Nombre + " (" + t.Servicio + ") del " + c.cuando(t.Inicio) + " al " + c.cuando(inicio)
		if prof != "" && prof != deref(t.ProfesionalID) {
			frase += ", con " + nomProf
		}
		args := map[string]any{"turnoId": t.ID, "fecha": inicio.In(c.zona).Format(time.DateOnly), "hora": inicio.In(c.zona).Format("15:04")}
		if prof != "" {
			args["profesionalId"] = prof
		}
		return frase + ". Le avisamos al cliente.", mustJSON(args), nil
	case "crear_turno":
		var in struct {
			ServicioID, Fecha, Hora, ProfesionalID, ClienteID, Nota string
			Cliente                                                 *turnos.Cliente
		}
		if err := leerArgs(raw, &in); err != nil {
			return "", nil, err
		}
		if in.ServicioID == "" {
			return "", nil, problema.Invalido("Falta el servicio.")
		}
		inicio, err := c.momento(in.Fecha, in.Hora)
		if err != nil {
			return "", nil, err
		}
		srv, prof, nomProf, err := s.resolverAmbos(ctx, c, in.ServicioID, in.ProfesionalID)
		if err != nil {
			return "", nil, err
		}
		if prof, err = c.profesionalPropio(prof); err != nil {
			return "", nil, err
		}
		if err := o.libre(ctx, srv, prof, inicio); err != nil {
			return "", nil, err
		}
		var servicio, cliente string
		args := map[string]any{"servicioId": srv, "fecha": inicio.In(c.zona).Format(time.DateOnly), "hora": inicio.In(c.zona).Format("15:04")}
		err = s.Pool.EnNegocio(ctx, c.negocio, func(tx pgx.Tx) error {
			if err := tx.QueryRow(ctx, "SELECT nombre FROM servicios WHERE id = $1", srv).Scan(&servicio); err != nil {
				return err
			}
			if prof != "" && nomProf == "" {
				if err := tx.QueryRow(ctx, "SELECT nombre FROM profesionales WHERE id = $1", prof).Scan(&nomProf); err != nil {
					return err
				}
			}
			switch {
			case in.ClienteID != "":
				var id string
				if err := tx.QueryRow(ctx, "SELECT id::text, nombre FROM clientes WHERE id::text = $1 AND suprimido_en IS NULL",
					strings.TrimSpace(in.ClienteID)).Scan(&id, &cliente); err != nil {
					return problema.NoEncontrado("No encontramos ese cliente.")
				}
				args["clienteId"] = id
			case in.Cliente != nil && strings.TrimSpace(in.Cliente.Nombre) != "":
				// Cliente nuevo: el resumen muestra también el teléfono y el email, así quien confirma ve
				// exactamente qué datos se van a cargar (pueden venir de un texto de un tercero).
				cliente = strings.TrimSpace(in.Cliente.Nombre)
				var datos []string
				if tel := strings.TrimSpace(in.Cliente.Telefono); tel != "" {
					datos = append(datos, "tel. "+tel)
				}
				if mail := strings.TrimSpace(in.Cliente.Email); mail != "" {
					datos = append(datos, mail)
				}
				if len(datos) > 0 {
					cliente += " (" + strings.Join(datos, ", ") + ")"
				} else {
					cliente += " (sin teléfono ni email)"
				}
				args["cliente"] = in.Cliente
			default:
				return problema.Invalido("Falta el cliente: un clienteId o los datos de uno nuevo.")
			}
			return nil
		})
		if err != nil {
			return "", nil, err
		}
		if prof != "" {
			args["profesionalId"] = prof
		}
		if in.Nota != "" {
			args["nota"] = in.Nota
		}
		frase := "Cargar " + servicio + " para " + cliente
		if nomProf != "" {
			frase += " con " + nomProf
		}
		return frase + ", el " + c.cuando(inicio) + ". Le avisamos al cliente.", mustJSON(args), nil
	}
	return "", nil, problema.Invalido("«" + nombre + "» no se puede proponer.")
}

// turno de la agenda de quien pregunta (uno ajeno no existe).
func (o *Operador) turno(ctx context.Context, id string) (turnos.Turno, error) {
	if id == "" {
		return turnos.Turno{}, problema.Invalido("Falta el turnoId.")
	}
	t, err := o.s.Turnos.Ver(ctx, o.c.negocio, id)
	if err != nil {
		return t, err
	}
	if !o.c.permitidoP(t.ProfesionalID) {
		return t, problema.NoEncontrado("No encontramos ese turno.")
	}
	return t, nil
}

// libre: el horario está entre los libres del servicio (con ese profesional, si lo hay).
func (o *Operador) libre(ctx context.Context, servicio, profesional string, inicio time.Time) error {
	hs, err := o.s.Turnos.Libres(ctx, o.c.negocio, servicio, profesional, inicio.In(o.c.zona).Format(time.DateOnly))
	if err != nil {
		return err
	}
	for _, h := range hs {
		if h.Inicio.Equal(inicio) && (profesional == "" || slices.Contains(h.Profesionales, profesional)) {
			return nil
		}
	}
	return problema.Conflicto("horario_ocupado", "Ese horario no está libre ("+o.c.cuando(inicio)+").")
}

// Cuando: «jueves 20 de noviembre a las 17:00», en la zona del negocio.
func (o *Operador) Cuando(t time.Time) string { return o.c.cuando(t) }

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func mustJSON(v any) json.RawMessage {
	b, _ := json.Marshal(v)
	return b
}

// PuedeEscribir: el panel no está en sólo lectura (la prueba vencida sin suscripción).
func (o *Operador) PuedeEscribir(ctx context.Context) error {
	return o.s.puedeEscribir(ctx, &herramienta{escribe: true})
}
