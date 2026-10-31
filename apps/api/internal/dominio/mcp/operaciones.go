package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"

	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/avisos"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/catalogo"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/clientes"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/turnos"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/problema"
)

// Lo que devuelven las herramientas: corto y con las horas en la zona del negocio, que es lo que el
// asistente le va a contar a la persona.

// TurnoMCP: un turno para el asistente. El contacto del cliente va sólo con «Ver clientes».
type TurnoMCP struct {
	ID          string        `json:"id"`
	Dia         string        `json:"dia"`
	Inicio      string        `json:"inicio"`
	Fin         string        `json:"fin"`
	Servicio    string        `json:"servicio"`
	Profesional string        `json:"profesional"`
	Cliente     ClienteDeTurn `json:"cliente"`
	Estado      string        `json:"estado"`
	Precio      int64         `json:"precio"`
	Nota        string        `json:"nota,omitempty"`
}

// ClienteDeTurn: el cliente de un turno.
type ClienteDeTurn struct {
	ID       string `json:"id"`
	Nombre   string `json:"nombre"`
	Telefono string `json:"telefono,omitempty"`
	Email    string `json:"email,omitempty"`
}

func (c *conexion) turno(t turnos.Turno) TurnoMCP {
	out := TurnoMCP{ID: t.ID, Dia: avisos.Fecha(t.Inicio.In(c.zona)), Inicio: t.Inicio.In(c.zona).Format(time.RFC3339),
		Fin: t.Fin.In(c.zona).Format(time.RFC3339), Servicio: t.Servicio, Profesional: t.Profesional, Estado: t.Estado,
		Precio: t.Precio, Nota: t.Nota, Cliente: ClienteDeTurn{ID: t.Cliente.ID, Nombre: t.Cliente.Nombre}}
	if c.puede(VerClientes) {
		out.Cliente.Telefono, out.Cliente.Email = t.Cliente.Telefono, t.Cliente.Email
	}
	return out
}

func leerArgs(raw json.RawMessage, v any) error {
	if err := json.Unmarshal(raw, v); err != nil {
		return problema.Invalido("Los argumentos no son válidos: " + err.Error())
	}
	return nil
}

// resolver un servicio o profesional activo por id o por nombre (sin distinguir mayúsculas).
func resolver(ctx context.Context, tx pgx.Tx, tabla, v, que string) (id, nombre string, err error) {
	v = strings.TrimSpace(v)
	if v == "" {
		return "", "", nil
	}
	filas, err := tx.Query(ctx, "SELECT id::text, nombre FROM "+tabla+` WHERE activo AND (id::text = $1 OR lower(nombre) = lower($1))
		ORDER BY id::text = $1 DESC LIMIT 2`, v)
	if err != nil {
		return "", "", err
	}
	defer filas.Close()
	var ids, nombres []string
	for filas.Next() {
		var i, n string
		if err := filas.Scan(&i, &n); err != nil {
			return "", "", err
		}
		ids, nombres = append(ids, i), append(nombres, n)
	}
	if err := filas.Err(); err != nil {
		return "", "", err
	}
	switch {
	case len(ids) == 0:
		return "", "", problema.NoEncontrado("No encontramos " + que + " «" + v + "». Mirá ver_servicios_y_profesionales.")
	case len(ids) > 1 && ids[0] != v:
		return "", "", problema.Invalido("Hay más de " + que + " con ese nombre: usá el id.")
	}
	return ids[0], nombres[0], nil
}

func (s *Servicio) resolverAmbos(ctx context.Context, c *conexion, servicio, profesional string) (srv, prof, nomProf string, err error) {
	err = s.Pool.EnNegocio(ctx, c.negocio, func(tx pgx.Tx) error {
		var err error
		if srv, _, err = resolver(ctx, tx, "servicios", servicio, "un servicio"); err != nil {
			return err
		}
		prof, nomProf, err = resolver(ctx, tx, "profesionales", profesional, "un profesional")
		return err
	})
	return
}

// ---- Ver la agenda

func verAgenda(ctx context.Context, s *Servicio, c *conexion, raw json.RawMessage) (any, string, error) {
	var in struct {
		Desde, Hasta, ProfesionalID string
		IncluirCancelados           bool
	}
	if err := leerArgs(raw, &in); err != nil {
		return nil, "", err
	}
	desde := c.hoy(s.ahora())
	var err error
	if in.Desde != "" {
		if desde, err = c.dia(in.Desde); err != nil {
			return nil, "", err
		}
	}
	hasta := desde
	if in.Hasta != "" {
		if hasta, err = c.dia(in.Hasta); err != nil {
			return nil, "", err
		}
	}
	if hasta.Before(desde) || hasta.Sub(desde) > 31*24*time.Hour {
		return nil, "", problema.Invalido("hasta va después de desde, con hasta 31 días entre medio.")
	}
	out := []TurnoMCP{}
	err = s.Pool.EnNegocio(ctx, c.negocio, func(tx pgx.Tx) error {
		prof, _, err := resolver(ctx, tx, "profesionales", in.ProfesionalID, "un profesional")
		if err != nil {
			return err
		}
		ts, err := turnos.Agenda(ctx, tx, desde, hasta.AddDate(0, 0, 1), prof, "", 300)
		if err != nil {
			return err
		}
		for _, t := range ts {
			if t.Estado == "cancelado" && !in.IncluirCancelados {
				continue
			}
			out = append(out, c.turno(t))
		}
		return nil
	})
	if err != nil {
		return nil, "", err
	}
	rango := desde.Format(time.DateOnly)
	if !hasta.Equal(desde) {
		rango += " al " + hasta.Format(time.DateOnly)
	}
	return map[string]any{"desde": desde.Format(time.DateOnly), "hasta": hasta.Format(time.DateOnly), "turnos": out},
		"miró la agenda del " + rango, nil
}

// ---- Horarios libres

// LibreMCP: un horario libre y quiénes pueden atenderlo.
type LibreMCP struct {
	Hora          string     `json:"hora"`
	Inicio        string     `json:"inicio"`
	Profesionales []Nombrado `json:"profesionales"`
}

// Nombrado: un id con su nombre.
type Nombrado struct {
	ID     string `json:"id"`
	Nombre string `json:"nombre"`
}

func buscarLibres(ctx context.Context, s *Servicio, c *conexion, raw json.RawMessage) (any, string, error) {
	var in struct {
		ServicioID, Fecha, ProfesionalID string
		Dias                             int
	}
	if err := leerArgs(raw, &in); err != nil {
		return nil, "", err
	}
	if in.ServicioID == "" {
		return nil, "", problema.Invalido("Falta el servicio.")
	}
	desde, err := c.dia(in.Fecha)
	if err != nil {
		return nil, "", err
	}
	in.Dias = min(max(in.Dias, 1), 7)
	srv, prof, _, err := s.resolverAmbos(ctx, c, in.ServicioID, in.ProfesionalID)
	if err != nil {
		return nil, "", err
	}
	nombres := map[string]string{}
	var servicio string
	if err := s.Pool.EnNegocio(ctx, c.negocio, func(tx pgx.Tx) error {
		ps, err := catalogo.Profesionales(ctx, tx, true)
		if err != nil {
			return err
		}
		for _, p := range ps {
			nombres[p.ID] = p.Nombre
		}
		return tx.QueryRow(ctx, "SELECT nombre FROM servicios WHERE id = $1", srv).Scan(&servicio)
	}); err != nil {
		return nil, "", err
	}
	type dia struct {
		Fecha    string     `json:"fecha"`
		Dia      string     `json:"dia"`
		Horarios []LibreMCP `json:"horarios"`
	}
	out := []dia{}
	for i := 0; i < in.Dias; i++ {
		f := desde.AddDate(0, 0, i)
		hs, err := s.Turnos.Libres(ctx, c.negocio, srv, prof, f.Format(time.DateOnly))
		if err != nil {
			return nil, "", err
		}
		d := dia{Fecha: f.Format(time.DateOnly), Dia: avisos.Fecha(f), Horarios: []LibreMCP{}}
		for _, h := range hs {
			l := LibreMCP{Hora: h.Hora, Inicio: h.Inicio.In(c.zona).Format(time.RFC3339)}
			for _, p := range h.Profesionales {
				l.Profesionales = append(l.Profesionales, Nombrado{p, nombres[p]})
			}
			d.Horarios = append(d.Horarios, l)
		}
		out = append(out, d)
	}
	return map[string]any{"servicio": servicio, "dias": out}, "buscó horarios libres de " + servicio + " desde el " + desde.Format(time.DateOnly), nil
}

// ---- Lista de espera

func verEspera(ctx context.Context, s *Servicio, c *conexion, _ json.RawMessage) (any, string, error) {
	type anotado struct {
		ID          string  `json:"id"`
		Nombre      string  `json:"nombre"`
		Telefono    string  `json:"telefono,omitempty"`
		Email       string  `json:"email,omitempty"`
		Servicio    string  `json:"servicio"`
		Profesional *string `json:"profesional"`
		Desde       string  `json:"desde"`
		Hasta       string  `json:"hasta"`
	}
	out := []anotado{}
	err := s.Pool.EnNegocio(ctx, c.negocio, func(tx pgx.Tx) error {
		filas, err := tx.Query(ctx, `SELECT e.id::text, e.nombre, e.telefono, e.email, s.nombre, p.nombre,
				to_char(e.desde, 'YYYY-MM-DD'), to_char(e.hasta, 'YYYY-MM-DD')
			FROM espera e JOIN servicios s ON s.id = e.servicio_id LEFT JOIN profesionales p ON p.id = e.profesional_id
			WHERE e.avisado_en IS NULL AND e.hasta >= $1::date ORDER BY e.desde, e.creado_en LIMIT 100`,
			c.hoy(s.ahora()).Format(time.DateOnly))
		if err != nil {
			return err
		}
		defer filas.Close()
		for filas.Next() {
			var a anotado
			if err := filas.Scan(&a.ID, &a.Nombre, &a.Telefono, &a.Email, &a.Servicio, &a.Profesional, &a.Desde, &a.Hasta); err != nil {
				return err
			}
			if !c.puede(VerClientes) {
				a.Telefono, a.Email = "", ""
			}
			out = append(out, a)
		}
		return filas.Err()
	})
	if err != nil {
		return nil, "", err
	}
	return map[string]any{"espera": out}, "miró la lista de espera", nil
}

// ---- Servicios y profesionales

func verCatalogo(ctx context.Context, s *Servicio, c *conexion, _ json.RawMessage) (any, string, error) {
	type servicio struct {
		ID          string `json:"id"`
		Nombre      string `json:"nombre"`
		Descripcion string `json:"descripcion,omitempty"`
		Duracion    int    `json:"duracionMinutos"`
		Precio      int64  `json:"precio"`
	}
	type profesional struct {
		ID           string   `json:"id"`
		Nombre       string   `json:"nombre"`
		Especialidad string   `json:"especialidad,omitempty"`
		Servicios    []string `json:"servicios"`
		Horario      []string `json:"horario"`
	}
	dias := []string{"domingo", "lunes", "martes", "miércoles", "jueves", "viernes", "sábado"}
	var ss []servicio
	var ps []profesional
	err := s.Pool.EnNegocio(ctx, c.negocio, func(tx pgx.Tx) error {
		srvs, err := catalogo.Servicios(ctx, tx, true)
		if err != nil {
			return err
		}
		ss = []servicio{}
		for _, x := range srvs {
			ss = append(ss, servicio{x.ID, x.Nombre, x.Descripcion, x.Duracion, x.Precio})
		}
		profs, err := catalogo.Profesionales(ctx, tx, true)
		if err != nil {
			return err
		}
		ps = []profesional{}
		for _, p := range profs {
			x := profesional{ID: p.ID, Nombre: p.Nombre, Especialidad: p.Especialidad, Servicios: p.Servicios, Horario: []string{}}
			for _, f := range p.Horario {
				if f.Dia >= 0 && f.Dia < len(dias) {
					x.Horario = append(x.Horario, dias[f.Dia]+" de "+f.Desde+" a "+f.Hasta)
				}
			}
			ps = append(ps, x)
		}
		return nil
	})
	if err != nil {
		return nil, "", err
	}
	return map[string]any{"servicios": ss, "profesionales": ps}, "miró los servicios y profesionales", nil
}

// ---- Crear, confirmar, reprogramar y cancelar

func crearTurno(ctx context.Context, s *Servicio, c *conexion, raw json.RawMessage) (any, string, error) {
	var in struct {
		ServicioID, Fecha, Hora, ProfesionalID, ClienteID, Nota string
		Cliente                                                 *turnos.Cliente
	}
	if err := leerArgs(raw, &in); err != nil {
		return nil, "", err
	}
	if in.ServicioID == "" {
		return nil, "", problema.Invalido("Falta el servicio.")
	}
	inicio, err := c.momento(in.Fecha, in.Hora)
	if err != nil {
		return nil, "", err
	}
	srv, prof, _, err := s.resolverAmbos(ctx, c, in.ServicioID, in.ProfesionalID)
	if err != nil {
		return nil, "", err
	}
	p := turnos.Pedido{ServicioID: srv, ProfesionalID: prof, Inicio: inicio, Nota: in.Nota}
	switch {
	case in.ClienteID != "":
		if err := s.Pool.EnNegocio(ctx, c.negocio, func(tx pgx.Tx) error {
			return tx.QueryRow(ctx, "SELECT id::text FROM clientes WHERE id::text = $1", strings.TrimSpace(in.ClienteID)).Scan(&p.ClienteID)
		}); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, "", problema.NoEncontrado("No encontramos ese cliente. Buscalo con buscar_clientes.")
			}
			return nil, "", err
		}
	case in.Cliente != nil:
		p.Cliente = *in.Cliente
	default:
		return nil, "", problema.Invalido("Falta el cliente: un clienteId o los datos de uno nuevo (cliente).")
	}
	t, err := s.Turnos.Reservar(ctx, c.negocio, p)
	if err != nil {
		return nil, "", err
	}
	frase := t.Servicio + " para " + t.Cliente.Nombre + " con " + t.Profesional + ", el " + c.cuando(t.Inicio) + "."
	s.anotarCambio(ctx, c, avisos.NotiReserva, "El asistente cargó un turno", frase, t.ID, "turno_creado")
	return map[string]any{"turno": c.turno(t)}, "cargó " + frase, nil
}

func turnoID(raw json.RawMessage) (string, error) {
	var in struct{ TurnoID string }
	if err := leerArgs(raw, &in); err != nil {
		return "", err
	}
	if strings.TrimSpace(in.TurnoID) == "" {
		return "", problema.Invalido("Falta el turnoId.")
	}
	return strings.TrimSpace(in.TurnoID), nil
}

func confirmarTurno(ctx context.Context, s *Servicio, c *conexion, raw json.RawMessage) (any, string, error) {
	return cambiarEstado(ctx, s, c, raw, "confirmado")
}

func cancelarTurno(ctx context.Context, s *Servicio, c *conexion, raw json.RawMessage) (any, string, error) {
	return cambiarEstado(ctx, s, c, raw, "cancelado")
}

func cambiarEstado(ctx context.Context, s *Servicio, c *conexion, raw json.RawMessage, estado string) (any, string, error) {
	id, err := turnoID(raw)
	if err != nil {
		return nil, "", err
	}
	t, antes, err := s.Turnos.CambiarEstado(ctx, c.negocio, id, estado)
	if err != nil {
		return nil, "", err
	}
	frase := t.Servicio + " de " + t.Cliente.Nombre + " con " + t.Profesional + ", el " + c.cuando(t.Inicio) + "."
	verbo, titulo, tipo := "confirmó ", "El asistente confirmó un turno", avisos.NotiReserva
	if estado == "cancelado" {
		verbo, titulo, tipo = "canceló ", "El asistente canceló un turno", avisos.NotiCancelacion
	}
	if antes != t.Estado {
		s.anotarCambio(ctx, c, tipo, titulo, frase, t.ID, "turno_"+estado)
	} else {
		verbo = "ya estaba " + estado + ": "
	}
	return map[string]any{"turno": c.turno(t)}, verbo + frase, nil
}

func reprogramarTurno(ctx context.Context, s *Servicio, c *conexion, raw json.RawMessage) (any, string, error) {
	var in struct{ TurnoID, Fecha, Hora, ProfesionalID string }
	if err := leerArgs(raw, &in); err != nil {
		return nil, "", err
	}
	if strings.TrimSpace(in.TurnoID) == "" {
		return nil, "", problema.Invalido("Falta el turnoId.")
	}
	inicio, err := c.momento(in.Fecha, in.Hora)
	if err != nil {
		return nil, "", err
	}
	_, prof, _, err := s.resolverAmbos(ctx, c, "", in.ProfesionalID)
	if err != nil {
		return nil, "", err
	}
	antes, err := s.Turnos.Ver(ctx, c.negocio, strings.TrimSpace(in.TurnoID))
	if err != nil {
		return nil, "", err
	}
	t, err := s.Turnos.Reprogramar(ctx, c.negocio, antes.ID, inicio, prof)
	if err != nil {
		return nil, "", err
	}
	frase := t.Servicio + " de " + t.Cliente.Nombre + " del " + c.cuando(antes.Inicio) + " al " + c.cuando(t.Inicio) +
		", con " + t.Profesional + "."
	s.anotarCambio(ctx, c, avisos.NotiReprogramacion, "El asistente movió un turno", frase, t.ID, "turno_reprogramado")
	return map[string]any{"turno": c.turno(t)}, "movió " + frase, nil
}

// ---- Clientes

// ClienteMCP: un cliente con su resumen.
type ClienteMCP struct {
	ID           string  `json:"id"`
	Nombre       string  `json:"nombre"`
	Telefono     string  `json:"telefono"`
	Email        string  `json:"email"`
	Visitas      int     `json:"visitas"`
	UltimaVisita *string `json:"ultimaVisita"`
	ProximoTurno *string `json:"proximoTurno"`
	Ausencias    int     `json:"ausencias"`
	Penalizado   bool    `json:"penalizado"`
}

func (c *conexion) fecha(t *time.Time) *string {
	if t == nil {
		return nil
	}
	v := t.In(c.zona).Format(time.RFC3339)
	return &v
}

func (c *conexion) cliente(x clientes.Cliente) ClienteMCP {
	return ClienteMCP{ID: x.ID, Nombre: x.Nombre, Telefono: x.Telefono, Email: x.Email, Visitas: x.Visitas,
		UltimaVisita: c.fecha(x.UltimaVisita), ProximoTurno: c.fecha(x.ProximoTurno), Ausencias: x.Ausencias, Penalizado: x.Penalizado}
}

// patron para ILIKE, sin que % o _ hagan de comodín.
func patron(q string) string {
	r := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return "%" + r.Replace(q) + "%"
}

func buscarClientes(ctx context.Context, s *Servicio, c *conexion, raw json.RawMessage) (any, string, error) {
	var in struct {
		Texto  string
		Limite int
	}
	if err := leerArgs(raw, &in); err != nil {
		return nil, "", err
	}
	in.Texto = strings.TrimSpace(in.Texto)
	if utf8.RuneCountInString(in.Texto) < 2 {
		return nil, "", problema.Invalido("Escribí al menos dos letras o números para buscar.")
	}
	if in.Limite <= 0 {
		in.Limite = 10
	}
	in.Limite = min(in.Limite, 50)
	var digitos strings.Builder
	for _, r := range in.Texto {
		if r >= '0' && r <= '9' {
			digitos.WriteRune(r)
		}
	}
	tel := ""
	if digitos.Len() >= 3 {
		tel = patron(digitos.String())
	}
	ahora := s.ahora()
	out := []ClienteMCP{}
	err := s.Pool.EnNegocio(ctx, c.negocio, func(tx pgx.Tx) error {
		filas, err := tx.Query(ctx, `SELECT c.id::text FROM clientes c
			WHERE c.nombre ILIKE $1 OR c.email ILIKE $1 OR ($2 <> '' AND c.telefono LIKE $2)
			ORDER BY lower(c.nombre), c.id LIMIT $3`, patron(in.Texto), tel, in.Limite)
		if err != nil {
			return err
		}
		ids, err := pgx.CollectRows(filas, pgx.RowTo[string])
		if err != nil {
			return err
		}
		for _, id := range ids {
			x, err := clientes.LeerCliente(ctx, tx, id, ahora)
			if err != nil {
				return err
			}
			out = append(out, c.cliente(x))
		}
		return nil
	})
	if err != nil {
		return nil, "", err
	}
	return map[string]any{"clientes": out}, "buscó clientes", nil
}

func verCliente(ctx context.Context, s *Servicio, c *conexion, raw json.RawMessage) (any, string, error) {
	var in struct{ ClienteID string }
	if err := leerArgs(raw, &in); err != nil {
		return nil, "", err
	}
	if strings.TrimSpace(in.ClienteID) == "" {
		return nil, "", problema.Invalido("Falta el clienteId.")
	}
	ahora := s.ahora()
	var ficha struct {
		ClienteMCP
		Gastado int64      `json:"gastado"`
		Turnos  []TurnoMCP `json:"ultimosTurnos"`
	}
	err := s.Pool.EnNegocio(ctx, c.negocio, func(tx pgx.Tx) error {
		if _, err := clientes.MarcarAusentes(ctx, tx, ahora); err != nil {
			return err
		}
		x, err := clientes.LeerCliente(ctx, tx, strings.TrimSpace(in.ClienteID), ahora)
		if err != nil {
			return err
		}
		ficha.ClienteMCP, ficha.Gastado = c.cliente(x), x.Gastado
		// Los últimos 20, del más nuevo al más viejo (incluye los que vienen).
		ts, err := turnos.Agenda(ctx, tx, time.Time{}, ahora.AddDate(2, 0, 0), "", x.ID, 1000)
		if err != nil {
			return err
		}
		ficha.Turnos = []TurnoMCP{}
		for i := len(ts) - 1; i >= 0 && len(ficha.Turnos) < 20; i-- {
			ficha.Turnos = append(ficha.Turnos, c.turno(ts[i]))
		}
		return nil
	})
	if err != nil {
		return nil, "", err
	}
	return ficha, "miró la ficha de " + ficha.Nombre, nil
}

// ---- Números

func resumenNumeros(ctx context.Context, s *Servicio, c *conexion, raw json.RawMessage) (any, string, error) {
	var in struct{ Desde, Hasta string }
	if err := leerArgs(raw, &in); err != nil {
		return nil, "", err
	}
	ahora := s.ahora()
	hoy := c.hoy(ahora)
	desde := time.Date(hoy.Year(), hoy.Month(), 1, 0, 0, 0, 0, c.zona)
	hasta := hoy
	var err error
	if in.Desde != "" {
		if desde, err = c.dia(in.Desde); err != nil {
			return nil, "", err
		}
	}
	if in.Hasta != "" {
		if hasta, err = c.dia(in.Hasta); err != nil {
			return nil, "", err
		}
	}
	if hasta.Before(desde) || hasta.Sub(desde) > 366*24*time.Hour {
		return nil, "", problema.Invalido("hasta va después de desde, con hasta un año entre medio.")
	}
	var r struct {
		Desde          string `json:"desde"`
		Hasta          string `json:"hasta"`
		Turnos         int    `json:"turnos"`
		Atendidos      int    `json:"atendidos"`
		Ausentes       int    `json:"ausentes"`
		Cancelados     int    `json:"cancelados"`
		PorVenir       int    `json:"porVenir"`
		Facturado      int64  `json:"facturado"`
		ClientesNuevos int    `json:"clientesNuevos"`
		Nota           string `json:"nota"`
	}
	r.Desde, r.Hasta = desde.Format(time.DateOnly), hasta.Format(time.DateOnly)
	r.Nota = "Turnos no cuenta los cancelados. Facturado: la suma de los precios de los turnos atendidos."
	fin := hasta.AddDate(0, 0, 1)
	err = s.Pool.EnNegocio(ctx, c.negocio, func(tx pgx.Tx) error {
		if _, err := clientes.MarcarAusentes(ctx, tx, ahora); err != nil {
			return err
		}
		if err := tx.QueryRow(ctx, `SELECT count(*) FILTER (WHERE estado <> 'cancelado'),
				count(*) FILTER (WHERE estado = 'atendido'), count(*) FILTER (WHERE estado = 'ausente'),
				count(*) FILTER (WHERE estado = 'cancelado'),
				count(*) FILTER (WHERE estado IN ('pendiente', 'confirmado', 'pendiente_pago')),
				coalesce(sum(precio) FILTER (WHERE estado = 'atendido'), 0)
			FROM turnos WHERE inicio >= $1 AND inicio < $2 AND NOT (estado = 'pendiente_pago' AND vence_en <= now())`, desde, fin).
			Scan(&r.Turnos, &r.Atendidos, &r.Ausentes, &r.Cancelados, &r.PorVenir, &r.Facturado); err != nil {
			return err
		}
		return tx.QueryRow(ctx, "SELECT count(*) FROM clientes WHERE creado_en >= $1 AND creado_en < $2", desde, fin).Scan(&r.ClientesNuevos)
	})
	if err != nil {
		return nil, "", err
	}
	return r, "miró los números del " + r.Desde + " al " + r.Hasta, nil
}
