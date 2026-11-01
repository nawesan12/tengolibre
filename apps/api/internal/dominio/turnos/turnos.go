// Package turnos: la reserva desde la página pública, el link del cliente (ver, cancelar,
// reprogramar) y la agenda del panel. La disponibilidad la calcula el paquete disponibilidad; acá se
// cargan los datos y se reserva con el profesional bloqueado, así dos personas no se llevan el
// mismo horario.
package turnos

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"

	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/catalogo"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/clientes"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/disponibilidad"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/negocios"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/db"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/problema"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/sesion"
)

// Servicio de turnos. Ahora es el reloj (tests); nil: time.Now.
type Servicio struct {
	Pool   *db.Pool
	Sesion *sesion.Servicio
	Ahora  func() time.Time
	// Avisar se llama después de confirmar cada cambio (reserva, cancelación, reprogramación, estado).
	// Lo engancha el dominio de avisos; nil: no se avisa.
	Avisar func(ctx context.Context, negocioID, turnoID, evento string)
	// Extras: membresías, cupones y cobros (ver extras.go). nil: la reserva no toca plata.
	Extras []Extra
}

func (s *Servicio) ahora() time.Time {
	if s.Ahora != nil {
		return s.Ahora()
	}
	return time.Now()
}

func (s *Servicio) avisar(ctx context.Context, negocioID, turnoID, evento string) {
	if s.Avisar != nil {
		s.Avisar(context.WithoutCancel(ctx), negocioID, turnoID, evento)
	}
}

// Cliente que reserva.
type Cliente struct {
	ID       string `json:"id,omitempty"`
	Nombre   string `json:"nombre"`
	Telefono string `json:"telefono"`
	Email    string `json:"email"`
	// Nacimiento (opcional, AAAA-MM-DD): se guarda si el cliente todavía no tenía. Sólo en el pedido.
	Nacimiento string `json:"nacimiento,omitempty"`
}

// Turno como lo ve el panel (el cliente ve lo mismo salvo el id de cliente).
type Turno struct {
	ID            string     `json:"id"`
	Codigo        string     `json:"codigo"`
	ServicioID    *string    `json:"servicioId"`
	Servicio      string     `json:"servicio"`
	ProfesionalID *string    `json:"profesionalId"`
	Profesional   string     `json:"profesional"`
	Cliente       Cliente    `json:"cliente"`
	Inicio        time.Time  `json:"inicio"`
	Fin           time.Time  `json:"fin"`
	Precio        int64      `json:"precio"`
	Estado        string     `json:"estado"`
	Origen        string     `json:"origen"`
	Nota          string     `json:"nota"`
	VenceEn       *time.Time `json:"venceEn"`
	CreadoEn      time.Time  `json:"creadoEn"`
	// SerieID: el turno fijo del que es parte (ver serie.go). null: un turno suelto.
	SerieID *string `json:"serieId"`
	// De dónde vino la confirmación (whatsapp, web, panel) y cuándo: el talón dice «Confirmó por
	// WhatsApp · 10:02». CambioPedidoEn: tocó «Necesito cambiarlo».
	ConfirmadoPor  *string    `json:"confirmadoPor"`
	ConfirmadoEn   *time.Time `json:"confirmadoEn"`
	CambioPedidoEn *time.Time `json:"cambioPedidoEn"`
	// Para recepción (el cliente no lo ve): desde cuándo hay que llamarlo y por qué (cambio,
	// sin_respuesta, no_llego).
	LlamarDesde  *time.Time `json:"llamarDesde"`
	LlamarMotivo *string    `json:"llamarMotivo"`
	// Sólo en la respuesta de la reserva: lo que descontó el cupón y el pago que queda pendiente.
	Descuento int64 `json:"descuento,omitempty"`
	Pago      *Pago `json:"pago,omitempty"`
}

const columnasTurno = `t.id::text, t.codigo, t.servicio_id::text, t.servicio, t.profesional_id::text, t.profesional,
	c.id::text, c.nombre, c.telefono, c.email, t.inicio, t.fin, t.precio, t.estado, t.origen, t.nota, t.vence_en, t.creado_en,
	t.serie_id::text, t.confirmado_por, t.confirmado_en, t.cambio_pedido_en, t.llamar_desde, t.llamar_motivo`

const desdeTurnos = " FROM turnos t JOIN clientes c ON c.id = t.cliente_id"

func escanearTurno(f pgx.Row, t *Turno) error {
	return f.Scan(&t.ID, &t.Codigo, &t.ServicioID, &t.Servicio, &t.ProfesionalID, &t.Profesional,
		&t.Cliente.ID, &t.Cliente.Nombre, &t.Cliente.Telefono, &t.Cliente.Email,
		&t.Inicio, &t.Fin, &t.Precio, &t.Estado, &t.Origen, &t.Nota, &t.VenceEn, &t.CreadoEn,
		&t.SerieID, &t.ConfirmadoPor, &t.ConfirmadoEn, &t.CambioPedidoEn, &t.LlamarDesde, &t.LlamarMotivo)
}

func leerTurno(ctx context.Context, tx pgx.Tx, donde string, arg any, bloquear bool) (Turno, error) {
	var t Turno
	q := "SELECT " + columnasTurno + desdeTurnos + " WHERE " + donde
	if bloquear {
		q += " FOR UPDATE OF t"
	}
	err := escanearTurno(tx.QueryRow(ctx, q, arg), &t)
	return t, err
}

func codigo() string {
	b := make([]byte, 18)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

// contexto de reserva del negocio fijado: zona y reglas.
type contexto struct {
	zona    *time.Location
	ajustes negocios.Ajustes
}

func cargarContexto(ctx context.Context, tx pgx.Tx) (contexto, error) {
	n, err := negocios.Leer(ctx, tx)
	if err != nil {
		return contexto{}, err
	}
	a, err := negocios.LeerAjustes(ctx, tx)
	if err != nil {
		return contexto{}, err
	}
	z, err := time.LoadLocation(n.ZonaHoraria)
	if err != nil {
		z = disponibilidad.ZonaPorDefecto
	}
	return contexto{zona: z, ajustes: a}, nil
}

func (c contexto) reglas() disponibilidad.Reglas {
	return disponibilidad.Reglas{Intervalo: c.ajustes.Intervalo, AnticipacionMin: c.ajustes.AnticipacionMin,
		DiasVista: c.ajustes.DiasVista, MargenMin: c.ajustes.MargenMin}
}

// datosDelDia arma lo que el motor necesita de un profesional para una fecha. sinTurno deja afuera
// un turno (el que se está reprogramando, para que no se tape a sí mismo).
func datosDelDia(ctx context.Context, tx pgx.Tx, p catalogo.Profesional, fecha string, zona *time.Location, sinTurno string) (disponibilidad.Profesional, error) {
	return datosDeDias(ctx, tx, p, fecha, fecha, zona, sinTurno)
}

// datosDeDias: lo mismo para un rango de fechas (desde y hasta incluidas), con las mismas consultas.
func datosDeDias(ctx context.Context, tx pgx.Tx, p catalogo.Profesional, desde, hasta string, zona *time.Location, sinTurno string) (disponibilidad.Profesional, error) {
	d := disponibilidad.Profesional{ID: p.ID, Horario: p.Horario}
	var err error
	if d.Bloqueos, err = catalogo.Bloqueos(ctx, tx, p.ID, desde, hasta); err != nil {
		return d, err
	}
	if d.Semanales, err = catalogo.Semanales(ctx, tx, p.ID); err != nil {
		return d, err
	}
	if d.Excepciones, err = catalogo.Excepciones(ctx, tx, p.ID, desde, hasta); err != nil {
		return d, err
	}
	ini, err1 := time.ParseInLocation(time.DateOnly, desde, zona)
	fin, err2 := time.ParseInLocation(time.DateOnly, hasta, zona)
	if err1 != nil || err2 != nil {
		return d, problema.Invalido("La fecha va como AAAA-MM-DD.")
	}
	// Un día de margen de cada lado: un turno de la noche anterior con margen puede asomar.
	filas, err := tx.Query(ctx, `SELECT inicio, fin, estado, vence_en FROM turnos
		WHERE profesional_id = $1 AND estado <> 'cancelado' AND inicio < $3 AND fin > $2 AND id::text <> $4`,
		p.ID, ini.AddDate(0, 0, -1), fin.AddDate(0, 0, 2), sinTurno)
	if err != nil {
		return d, err
	}
	d.Turnos, err = pgx.CollectRows(filas, pgx.RowToStructByPos[disponibilidad.Ocupado])
	return d, err
}

// ProximoLibre es el primer horario que se puede reservar en el negocio fijado, de hoy a dias−1
// días adelante (en su zona), o nil. Por profesional mira su servicio activo más corto: si ése no
// entra, ninguno entra. Lo usa el directorio, acotado a pocos días y a los negocios de la página.
func ProximoLibre(ctx context.Context, tx pgx.Tx, ahora time.Time, dias int) (*time.Time, error) {
	c, err := cargarContexto(ctx, tx)
	if err != nil {
		return nil, err
	}
	srvs, err := catalogo.Servicios(ctx, tx, true)
	if err != nil {
		return nil, err
	}
	ps, err := catalogo.Profesionales(ctx, tx, true)
	if err != nil {
		return nil, err
	}
	hoy := ahora.In(c.zona)
	fechas := make([]string, dias)
	for i := range fechas {
		fechas[i] = time.Date(hoy.Year(), hoy.Month(), hoy.Day()+i, 0, 0, 0, 0, c.zona).Format(time.DateOnly)
	}
	type cargado struct {
		d        disponibilidad.Profesional
		duracion int
	}
	var cs []cargado
	for _, p := range ps {
		duracion := 0
		for _, s := range srvs {
			if p.Hace(s.ID) && (duracion == 0 || s.Duracion < duracion) {
				duracion = s.Duracion
			}
		}
		if duracion == 0 || len(p.Horario) == 0 {
			continue
		}
		d, err := datosDeDias(ctx, tx, p, fechas[0], fechas[len(fechas)-1], c.zona, "")
		if err != nil {
			return nil, err
		}
		cs = append(cs, cargado{d, duracion})
	}
	for _, fecha := range fechas {
		var mejor *time.Time
		for _, x := range cs {
			for _, h := range disponibilidad.Dia(disponibilidad.Consulta{Profesional: x.d, Fecha: fecha, Duracion: x.duracion,
				Reglas: c.reglas(), Ahora: ahora, Zona: c.zona}) {
				if h.Libre {
					if mejor == nil || h.Inicio.Before(*mejor) {
						i := h.Inicio
						mejor = &i
					}
					break
				}
			}
		}
		if mejor != nil {
			return mejor, nil
		}
	}
	return nil, nil
}

// HorarioLibre de la página pública: a qué hora, y qué profesionales están libres.
type HorarioLibre struct {
	Hora          string    `json:"hora"`
	Inicio        time.Time `json:"inicio"`
	Profesionales []string  `json:"profesionales"`
}

// libres de un servicio en una fecha: con un profesional, los de él; sin, la unión de todos los que
// hacen el servicio (en el orden del negocio).
func libres(ctx context.Context, tx pgx.Tx, c contexto, servicioID, profesionalID, fecha string, ahora time.Time) ([]HorarioLibre, error) {
	srv, err := catalogo.LeerServicio(ctx, tx, servicioID)
	if err != nil || !srv.Activo {
		return nil, problema.NoEncontrado("Ese servicio no existe o no se puede reservar.")
	}
	ps, err := catalogo.Profesionales(ctx, tx, true)
	if err != nil {
		return nil, err
	}
	out := []HorarioLibre{}
	indice := map[string]int{}
	for _, p := range ps {
		if !p.Hace(srv.ID) || (profesionalID != "" && p.ID != profesionalID) {
			continue
		}
		d, err := datosDelDia(ctx, tx, p, fecha, c.zona, "")
		if err != nil {
			return nil, err
		}
		for _, h := range disponibilidad.Dia(disponibilidad.Consulta{Profesional: d, Fecha: fecha, Duracion: srv.Duracion,
			Reglas: c.reglas(), Ahora: ahora, Zona: c.zona}) {
			if !h.Libre {
				continue
			}
			i, ok := indice[h.Hora]
			if !ok {
				i = len(out)
				indice[h.Hora] = i
				out = append(out, HorarioLibre{Hora: h.Hora, Inicio: h.Inicio})
			}
			out[i].Profesionales = append(out[i].Profesionales, p.ID)
		}
	}
	ordenar(out)
	return out, nil
}

func ordenar(hs []HorarioLibre) {
	for i := 1; i < len(hs); i++ {
		for j := i; j > 0 && hs[j].Inicio.Before(hs[j-1].Inicio); j-- {
			hs[j], hs[j-1] = hs[j-1], hs[j]
		}
	}
}

func validarCliente(c *Cliente) error {
	c.Nombre, c.Telefono, c.Email = strings.TrimSpace(c.Nombre), limpiarTelefono(c.Telefono), strings.ToLower(strings.TrimSpace(c.Email))
	if n := utf8.RuneCountInString(c.Nombre); n < 1 || n > 80 {
		return problema.Invalido("Falta tu nombre.", problema.Campo{Ruta: "cliente.nombre", Mensaje: "requerido"})
	}
	if c.Telefono == "" && c.Email == "" {
		return problema.Invalido("Dejá un teléfono o un email para avisarte.", problema.Campo{Ruta: "cliente.telefono", Mensaje: "requerido"})
	}
	if c.Email != "" && (!strings.Contains(c.Email, "@") || len(c.Email) > 120) {
		return problema.Invalido("Revisá el email.", problema.Campo{Ruta: "cliente.email", Mensaje: "inválido"})
	}
	if len(c.Telefono) > 30 {
		return problema.Invalido("Revisá el teléfono.", problema.Campo{Ruta: "cliente.telefono", Mensaje: "inválido"})
	}
	return nil
}

// limpiarTelefono deja dígitos y un + inicial: así "11 2345-6789" y "1123456789" son el mismo cliente.
func limpiarTelefono(t string) string {
	var b strings.Builder
	for i, r := range strings.TrimSpace(t) {
		if (r >= '0' && r <= '9') || (r == '+' && i == 0) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// clienteDe busca al cliente por teléfono (o crea uno) en el negocio fijado.
// La fecha de nacimiento sólo se completa si faltaba (nunca pisa la que cargó el negocio).
func clienteDe(ctx context.Context, tx pgx.Tx, c Cliente) (string, error) {
	var id string
	nacimiento := &c.Nacimiento
	if c.Nacimiento == "" {
		nacimiento = nil
	}
	if c.Telefono != "" {
		err := tx.QueryRow(ctx, `INSERT INTO clientes (negocio_id, nombre, telefono, email, nacimiento) VALUES (app.negocio_actual(), $1, $2, $3, $4::date)
			ON CONFLICT (negocio_id, telefono) WHERE telefono <> '' DO UPDATE SET email = CASE WHEN excluded.email <> '' THEN excluded.email ELSE clientes.email END,
				nacimiento = coalesce(clientes.nacimiento, excluded.nacimiento)
			RETURNING id::text`, c.Nombre, c.Telefono, c.Email, nacimiento).Scan(&id)
		return id, err
	}
	err := tx.QueryRow(ctx, "SELECT id::text FROM clientes WHERE telefono = '' AND email = $1 ORDER BY creado_en LIMIT 1", c.Email).Scan(&id)
	if err == pgx.ErrNoRows {
		err = tx.QueryRow(ctx, "INSERT INTO clientes (negocio_id, nombre, email, nacimiento) VALUES (app.negocio_actual(), $1, $2, $3::date) RETURNING id::text",
			c.Nombre, c.Email, nacimiento).Scan(&id)
	} else if err == nil && nacimiento != nil {
		_, err = tx.Exec(ctx, "UPDATE clientes SET nacimiento = $2::date WHERE id = $1 AND nacimiento IS NULL", id, nacimiento)
	}
	return id, err
}

// Pedido de reserva.
type Pedido struct {
	ServicioID    string    `json:"servicioId"`
	ProfesionalID string    `json:"profesionalId"`
	Inicio        time.Time `json:"inicio"`
	Cliente       Cliente   `json:"cliente"`
	Nota          string    `json:"nota"`
	// Forzar (sólo panel): reserva aunque el horario no figure libre (sobreturno).
	Forzar bool `json:"forzar,omitempty"`
	// Cupon y Referido: los códigos que trae quien reserva (los aplica el extra de promos).
	Cupon    string `json:"cupon,omitempty"`
	Referido string `json:"referido,omitempty"`
	// UsarMembresia (sólo web, con sesión): el turno va a nombre del socio que ingresó y gasta una
	// visita de su membresía. Lo resuelve reservarPublico, que deja ClienteID.
	UsarMembresia bool   `json:"usarMembresia,omitempty"`
	ClienteID     string `json:"-"`
	// SerieID: el turno fijo al que pertenece (lo pone reservarSerie).
	SerieID string `json:"-"`
	// reglas, si no es nil, reemplaza las del negocio (el turno fijo mira más allá de la vista).
	reglas *disponibilidad.Reglas
	// SinAnticipacion (sólo desde otro dominio): no aplica la anticipación mínima. La lista de espera
	// ofrece huecos que se liberaron a último momento: el negocio ya decidió ofrecerlos.
	SinAnticipacion bool `json:"-"`
}

// elegir el profesional para el horario: el pedido, o el primero libre en el orden del negocio.
// Bloquea al elegido hasta el final de la transacción y recalcula ahí adentro.
func elegir(ctx context.Context, tx pgx.Tx, c contexto, srv catalogo.Prestacion, pedido string, inicio time.Time, ahora time.Time,
	forzar bool, sinTurno string) (catalogo.Profesional, error) {
	return elegirCon(ctx, tx, c, c.reglas(), srv, pedido, inicio, ahora, forzar, sinTurno)
}

func elegirCon(ctx context.Context, tx pgx.Tx, c contexto, reglas disponibilidad.Reglas, srv catalogo.Prestacion, pedido string,
	inicio time.Time, ahora time.Time, forzar bool, sinTurno string) (catalogo.Profesional, error) {
	fecha := inicio.In(c.zona).Format(time.DateOnly)
	var candidatos []string
	if pedido != "" {
		candidatos = []string{pedido}
	} else {
		ps, err := catalogo.Profesionales(ctx, tx, true)
		if err != nil {
			return catalogo.Profesional{}, err
		}
		for _, p := range ps {
			if p.Hace(srv.ID) {
				candidatos = append(candidatos, p.ID)
			}
		}
	}
	for _, id := range candidatos {
		p, err := catalogo.LeerProfesional(ctx, tx, id, true)
		if err == pgx.ErrNoRows {
			continue
		}
		if err != nil {
			return p, err
		}
		if !p.Activo || !p.Hace(srv.ID) {
			continue
		}
		if forzar {
			return p, nil
		}
		d, err := datosDelDia(ctx, tx, p, fecha, c.zona, sinTurno)
		if err != nil {
			return p, err
		}
		if disponibilidad.Libre(disponibilidad.Consulta{Profesional: d, Fecha: fecha, Duracion: srv.Duracion,
			Reglas: reglas, Ahora: ahora, Zona: c.zona}, inicio) {
			return p, nil
		}
	}
	return catalogo.Profesional{}, problema.Conflicto("horario_ocupado", "Ese horario ya no está libre. Elegí otro.")
}

// reservar crea el turno en el negocio fijado.
func reservar(ctx context.Context, tx pgx.Tx, in Pedido, origen string, ahora time.Time) (Turno, error) {
	// Con ClienteID (el socio que ingresó) los datos de contacto son los suyos: no se piden.
	if err := validarCliente(&in.Cliente); err != nil && in.ClienteID == "" {
		return Turno{}, err
	}
	if utf8.RuneCountInString(in.Nota) > 500 {
		return Turno{}, problema.Invalido("La nota es muy larga (hasta 500).")
	}
	if in.Cliente.Nacimiento != "" {
		n, err := clientes.ValidarNacimiento(&in.Cliente.Nacimiento, ahora, "cliente.nacimiento")
		if err != nil {
			return Turno{}, err
		}
		in.Cliente.Nacimiento = *n
	}
	c, err := cargarContexto(ctx, tx)
	if err != nil {
		return Turno{}, err
	}
	if in.SinAnticipacion {
		c.ajustes.AnticipacionMin = 0
	}
	srv, err := catalogo.LeerServicio(ctx, tx, in.ServicioID)
	if err != nil || !srv.Activo {
		return Turno{}, problema.NoEncontrado("Ese servicio no existe o no se puede reservar.")
	}
	reglas := c.reglas()
	if in.reglas != nil {
		reglas = *in.reglas
	}
	p, err := elegirCon(ctx, tx, c, reglas, srv, in.ProfesionalID, in.Inicio, ahora, in.Forzar, "")
	if err != nil {
		return Turno{}, err
	}
	clienteID := in.ClienteID
	if clienteID == "" {
		if clienteID, err = clienteDe(ctx, tx, in.Cliente); err != nil {
			return Turno{}, err
		}
	}
	// Con una penalidad vigente (por ausencias, o puesta a mano) no se reserva desde la web; desde el
	// panel sí, porque ahí decide el negocio.
	if origen == "web" {
		penalizado, err := clientes.Penalizado(ctx, tx, clienteID)
		if err != nil {
			return Turno{}, err
		}
		if penalizado {
			return Turno{}, problema.Conflicto("cliente_penalizado",
				"Por ausencias anteriores no podés reservar online por ahora. Escribile al negocio.")
		}
	}
	estado := "pendiente"
	if origen == "panel" {
		estado = "confirmado"
	}
	var id string
	err = tx.QueryRow(ctx, `INSERT INTO turnos (negocio_id, codigo, servicio_id, servicio, profesional_id, profesional, cliente_id,
		inicio, fin, precio, estado, origen, nota, serie_id) VALUES (app.negocio_actual(), $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12,
		nullif($13, '')::uuid) RETURNING id::text`, codigo(), srv.ID, srv.Nombre, p.ID, p.Nombre, clienteID,
		in.Inicio, in.Inicio.Add(time.Duration(srv.Duracion)*time.Minute), srv.Precio, estado, origen, strings.TrimSpace(in.Nota),
		in.SerieID).Scan(&id)
	if err != nil {
		return Turno{}, err
	}
	return leerTurno(ctx, tx, "t.id = $1", id, false)
}

// reprogramar mueve el turno a otro horario (y, si se pide, a otro profesional) con las mismas reglas
// que una reserva nueva.
func reprogramar(ctx context.Context, tx pgx.Tx, t Turno, inicio time.Time, profesionalID string, ahora time.Time, forzar bool) (Turno, error) {
	if t.Estado != "pendiente" && t.Estado != "confirmado" {
		return t, problema.Conflicto("estado", "Sólo se reprograma un turno pendiente o confirmado.")
	}
	if t.ServicioID == nil {
		return t, problema.Conflicto("sin_servicio", "El servicio de este turno ya no existe: sacá uno nuevo.")
	}
	c, err := cargarContexto(ctx, tx)
	if err != nil {
		return t, err
	}
	srv, err := catalogo.LeerServicio(ctx, tx, *t.ServicioID)
	if err != nil {
		return t, err
	}
	if profesionalID == "" && t.ProfesionalID != nil {
		profesionalID = *t.ProfesionalID
	}
	p, err := elegir(ctx, tx, c, srv, profesionalID, inicio, ahora, forzar, t.ID)
	if err != nil {
		return t, err
	}
	if _, err := tx.Exec(ctx, `UPDATE turnos SET inicio = $2, fin = $3, profesional_id = $4, profesional = $5, actualizado_en = now()
		WHERE id = $1`, t.ID, inicio, inicio.Add(time.Duration(srv.Duracion)*time.Minute), p.ID, p.Nombre); err != nil {
		return t, err
	}
	return leerTurno(ctx, tx, "t.id = $1", t.ID, false)
}

// Pasos de estado permitidos desde el panel.
var pasos = map[string]map[string]bool{
	"pendiente_pago": {"pendiente": true, "confirmado": true, "cancelado": true},
	"pendiente":      {"confirmado": true, "cancelado": true, "atendido": true, "ausente": true},
	"confirmado":     {"pendiente": true, "cancelado": true, "atendido": true, "ausente": true},
	"atendido":       {"ausente": true, "confirmado": true},
	"ausente":        {"atendido": true, "confirmado": true},
}

// PuedePasar dice si el panel puede llevar un turno de un estado a otro.
func PuedePasar(de, a string) bool { return de == a || pasos[de][a] }

// negocioPublico fija el negocio del slug, o 404.
func (s *Servicio) negocioPublico(ctx context.Context, slug string) (string, error) {
	var id *string
	err := s.Pool.EnTx(ctx, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, "SELECT app.negocio_publico($1)::text", slug).Scan(&id)
	})
	if err != nil {
		return "", err
	}
	if id == nil {
		return "", problema.NoEncontrado("No encontramos ese negocio.")
	}
	return *id, nil
}

// negocioDelTurno: del código del link al negocio, o 404.
func (s *Servicio) negocioDelTurno(ctx context.Context, cod string) (string, error) {
	var id *string
	err := s.Pool.EnTx(ctx, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, "SELECT app.negocio_del_turno($1)::text", cod).Scan(&id)
	})
	if err != nil {
		return "", err
	}
	if id == nil {
		return "", problema.NoEncontrado("No encontramos ese turno.")
	}
	return *id, nil
}

// Disponible dice si el profesional tiene libre ese horario para el servicio, en el negocio fijado,
// sin la anticipación mínima (la lista de espera lo usa para saber si un hueco sigue para ofrecer).
func Disponible(ctx context.Context, tx pgx.Tx, servicioID, profesionalID string, inicio, ahora time.Time) (bool, error) {
	c, err := cargarContexto(ctx, tx)
	if err != nil {
		return false, err
	}
	c.ajustes.AnticipacionMin = 0
	srv, err := catalogo.LeerServicio(ctx, tx, servicioID)
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && !srv.Activo) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	p, err := catalogo.LeerProfesional(ctx, tx, profesionalID, false)
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && (!p.Activo || !p.Hace(srv.ID))) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	fecha := inicio.In(c.zona).Format(time.DateOnly)
	d, err := datosDelDia(ctx, tx, p, fecha, c.zona, "")
	if err != nil {
		return false, err
	}
	return disponibilidad.Libre(disponibilidad.Consulta{Profesional: d, Fecha: fecha, Duracion: srv.Duracion,
		Reglas: c.reglas(), Ahora: ahora, Zona: c.zona}, inicio), nil
}
