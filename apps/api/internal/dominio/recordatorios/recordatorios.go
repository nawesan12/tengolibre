// Package recordatorios: los recordatorios automáticos a los clientes (propuesta 12 del diseño v2,
// en Clientes › Recordatorios automáticos): el saludo de cumpleaños y «hace mucho que no venís».
// Cada uno se prende por negocio, con un texto corto propio y un cupón opcional. El trabajo del
// worker los encola en la cola de avisos (uno por cliente por ocasión, con la clave del aviso) y el
// mail se arma al mandarlo con el Armador de este paquete: si el cliente se dio de baja o ya
// reservó, se descarta.
package recordatorios

import (
	"context"
	"errors"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"

	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/avisos"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/promos"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/auditoria"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/correo"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/db"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/problema"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/sesion"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/trabajos"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/web"
)

// Tipos de los recordatorios automáticos (también son tipos de aviso).
const (
	Cumpleanios = avisos.Cumpleanios
	Extraniamos = avisos.Extraniamos
)

// Servicio de recordatorios. Sitio: el público, para los links de los mails. Ahora es el reloj
// (tests); nil: time.Now.
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

// Rutas del dominio: lo ve el mostrador (dueña, encargados, recepción); lo cambia la gestión.
func (s *Servicio) Rutas(mux *http.ServeMux) {
	ses := s.Sesion
	mux.HandleFunc("GET /v1/panel/recordatorios", ses.Panel(avisos.Mostrador, web.Envolver(s.recordatorios)))
	mux.HandleFunc("PUT /v1/panel/recordatorios/{tipo}", ses.Panel(sesion.Gestion, web.Envolver(s.guardarRecordatorio)))
	mux.HandleFunc("GET /v1/panel/recordatorios/{tipo}/vista-previa", ses.Panel(avisos.Mostrador, web.Envolver(s.vistaPrevia)))
}

// Trabajo del worker: encola los que tocan (cada media hora).
func (s *Servicio) Trabajo() trabajos.Trabajo {
	return trabajos.Trabajo{Nombre: "recordatorios_automaticos", Cada: 30 * time.Minute, Correr: s.Encolar}
}

// Armadores para avisos.Enviador.
func Armadores() map[string]avisos.Armador {
	return map[string]avisos.Armador{Cumpleanios: armar, Extraniamos: armar}
}

// HoraAutomaticos: desde qué hora del negocio salen (el día que tocan).
const HoraAutomaticos = 10

// DiasExtraniamosPorDefecto: días desde el último turno atendido.
const DiasExtraniamosPorDefecto = 60

// Textos por defecto, si el negocio no escribió el suyo.
var textosPorDefecto = map[string]string{
	Cumpleanios: "Que tengas un muy lindo día. Cuando quieras darte un gusto, te esperamos.",
	Extraniamos: "Cuando quieras volver, tenemos lugar. Reservás en un minuto desde acá.",
}

// MaxTextoRecordatorio: el texto propio es corto (va en un párrafo del mail).
const MaxTextoRecordatorio = 300

// CuponDelMail: lo que muestra el panel del cupón elegido.
type CuponDelMail struct {
	ID        string `json:"id"`
	Codigo    string `json:"codigo"`
	Descuento string `json:"descuento"`
	Vigente   bool   `json:"vigente"`
}

// Recordatorio automático de un negocio.
type Recordatorio struct {
	Tipo    string        `json:"tipo"`
	Titulo  string        `json:"titulo"`
	Activo  bool          `json:"activo"`
	Texto   string        `json:"texto"`
	CuponID *string       `json:"cuponId"`
	Cupon   *CuponDelMail `json:"cupon"`
	// Dias: sólo «hace mucho que no venís» (días desde el último turno atendido).
	Dias  int    `json:"dias"`
	Regla string `json:"regla"`
}

// Destinatario de esta semana.
type Destinatario struct {
	ClienteID string `json:"clienteId"`
	Nombre    string `json:"nombre"`
	Email     string `json:"email"`
	// Fecha (AAAA-MM-DD, del negocio): el día que le toca.
	Fecha    string `json:"fecha"`
	Detalle  string `json:"detalle"`
	Enviado  bool   `json:"enviado"`
	clave    string
	semanas  int
	servicio string
}

// VistaPrevia del mail tal cual llega.
type VistaPrevia struct {
	Para   string `json:"para"`
	Asunto string `json:"asunto"`
	HTML   string `json:"html"`
	Texto  string `json:"texto"`
}

// RecordatorioPanel: el recordatorio con su vista previa y a quiénes les toca esta semana.
type RecordatorioPanel struct {
	Recordatorio
	VistaPrevia VistaPrevia    `json:"vistaPrevia"`
	Semana      []Destinatario `json:"semana"`
	// SemanaTexto: «Esta semana: 4 cumpleaños».
	SemanaTexto string `json:"semanaTexto"`
}

func regla(tipo string, dias int) string {
	if tipo == Cumpleanios {
		return "Le llega el día de su cumple, a las " + strconv.Itoa(HoraAutomaticos) + ", si vino al menos una vez."
	}
	return "Le llega cuando pasan " + strconv.Itoa(dias) + " días desde su último turno atendido y no tiene otro reservado."
}

func tituloDe(tipo string) string {
	if tipo == Cumpleanios {
		return "Cumpleaños"
	}
	return "«Hace mucho que no venís»"
}

// leerRecordatorio del negocio fijado (sin fila: apagado y con el texto por defecto).
func leerRecordatorio(ctx context.Context, tx pgx.Tx, tipo string, ahora time.Time) (Recordatorio, error) {
	r := Recordatorio{Tipo: tipo, Titulo: tituloDe(tipo), Dias: DiasExtraniamosPorDefecto}
	err := tx.QueryRow(ctx, "SELECT activo, texto, cupon_id::text, dias FROM recordatorios WHERE negocio_id = app.negocio_actual() AND tipo = $1", tipo).
		Scan(&r.Activo, &r.Texto, &r.CuponID, &r.Dias)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return r, err
	}
	if r.Texto == "" {
		r.Texto = textosPorDefecto[tipo]
	}
	r.Regla = regla(tipo, r.Dias)
	if r.CuponID != nil {
		c, err := promos.Leer(ctx, tx, *r.CuponID)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return r, err
		}
		if err == nil {
			r.Cupon = &CuponDelMail{ID: c.ID, Codigo: c.Codigo, Descuento: promos.DescuentoEnPalabras(c), Vigente: promos.Vigente(c, ahora)}
		}
	}
	return r, nil
}

// cumpleEn: el día del cumple de nacimiento en el año dado (el 29/2, el 28 en los años no bisiestos).
func cumpleEn(nacimiento time.Time, anio int, zona *time.Location) time.Time {
	m, d := nacimiento.Month(), nacimiento.Day()
	if m == time.February && d == 29 && time.Date(anio, 3, 0, 0, 0, 0, 0, zona).Day() != 29 {
		d = 28
	}
	return time.Date(anio, m, d, 0, 0, 0, 0, zona)
}

// candidatos del recordatorio con fecha en [desde, hasta] (medianoches del negocio, inclusive):
// clientes con email que no se dieron de baja. ahora es para «sin turnos futuros».
func candidatos(ctx context.Context, tx pgx.Tx, r Recordatorio, zona *time.Location, desde, hasta, ahora time.Time) ([]Destinatario, error) {
	var out []Destinatario
	if r.Tipo == Cumpleanios {
		filas, err := tx.Query(ctx, `SELECT c.id::text, c.nombre, c.email, c.nacimiento FROM clientes c
			LEFT JOIN preferencias p ON p.cliente_id = c.id
			WHERE c.nacimiento IS NOT NULL AND c.email <> '' AND coalesce(p.novedades, true)
			AND EXISTS (SELECT 1 FROM turnos t WHERE t.cliente_id = c.id AND t.estado = 'atendido')
			ORDER BY c.nombre LIMIT 20000`)
		if err != nil {
			return nil, err
		}
		defer filas.Close()
		for filas.Next() {
			var d Destinatario
			var nac time.Time
			if err := filas.Scan(&d.ClienteID, &d.Nombre, &d.Email, &nac); err != nil {
				return nil, err
			}
			for anio := desde.Year(); anio <= hasta.Year(); anio++ {
				c := cumpleEn(nac, anio, zona)
				if c.Before(desde) || c.After(hasta) {
					continue
				}
				d.Fecha = c.Format(time.DateOnly)
				d.Detalle = "Cumple el " + avisos.Fecha(c)
				if edad := anio - nac.Year(); edad > 0 && edad < 120 {
					d.Detalle += " (" + strconv.Itoa(edad) + ")"
				}
				d.clave = Cumpleanios + ":" + d.ClienteID + ":" + strconv.Itoa(anio)
				out = append(out, d)
			}
		}
		return out, filas.Err()
	}
	// «Hace mucho que no venís»: el último atendido cayó r.Dias antes de algún día de la ventana.
	filas, err := tx.Query(ctx, `SELECT c.id::text, c.nombre, c.email, u.turno_id, u.inicio, u.servicio FROM clientes c
		LEFT JOIN preferencias p ON p.cliente_id = c.id
		JOIN LATERAL (SELECT t.id::text AS turno_id, t.inicio, t.servicio FROM turnos t
			WHERE t.cliente_id = c.id AND t.estado = 'atendido' ORDER BY t.inicio DESC LIMIT 1) u ON true
		WHERE c.email <> '' AND coalesce(p.novedades, true) AND u.inicio >= $1 AND u.inicio < $2
		AND NOT EXISTS (SELECT 1 FROM turnos t WHERE t.cliente_id = c.id AND t.inicio >= $3
			AND (t.estado IN ('pendiente', 'confirmado') OR (t.estado = 'pendiente_pago' AND t.vence_en > $3)))
		ORDER BY u.inicio LIMIT 5000`, desde.AddDate(0, 0, -r.Dias), hasta.AddDate(0, 0, 1-r.Dias), ahora)
	if err != nil {
		return nil, err
	}
	defer filas.Close()
	for filas.Next() {
		var d Destinatario
		var turno string
		var ultimo time.Time
		if err := filas.Scan(&d.ClienteID, &d.Nombre, &d.Email, &turno, &ultimo, &d.servicio); err != nil {
			return nil, err
		}
		u := ultimo.In(zona)
		toca := time.Date(u.Year(), u.Month(), u.Day(), 0, 0, 0, 0, zona).AddDate(0, 0, r.Dias)
		d.Fecha = toca.Format(time.DateOnly)
		d.semanas = int(math.Round(float64(r.Dias) / 7))
		d.Detalle = "Último turno el " + avisos.Fecha(u) + ": " + d.servicio
		d.clave = Extraniamos + ":" + d.ClienteID + ":" + turno
		out = append(out, d)
	}
	return out, filas.Err()
}

// marcarEnviados: los que ya tienen su aviso en la cola (por la clave).
func marcarEnviados(ctx context.Context, tx pgx.Tx, ds []Destinatario) error {
	if len(ds) == 0 {
		return nil
	}
	claves := make([]string, len(ds))
	for i, d := range ds {
		claves[i] = d.clave
	}
	filas, err := tx.Query(ctx, "SELECT clave FROM avisos WHERE clave = ANY($1)", claves)
	if err != nil {
		return err
	}
	ya, err := pgx.CollectRows(filas, pgx.RowTo[string])
	if err != nil {
		return err
	}
	hay := map[string]bool{}
	for _, c := range ya {
		hay[c] = true
	}
	for i := range ds {
		ds[i].Enviado = hay[ds[i].clave]
	}
	return nil
}

// datosMail: lo que varía de un mail a otro.
type datosMail struct {
	nombre, servicio string
	semanas          int
	cupon            *promos.Cupon
}

// cartaAutomatica arma el mail del recordatorio.
func cartaAutomatica(sitio string, n avisos.NegocioDelMail, r Recordatorio, d datosMail, baja string) avisos.Carta {
	nombre := avisos.PrimerNombre(d.nombre)
	c := avisos.Carta{Encabezado: n.Nombre, Baja: baja, Boton: "Reservar turno", Link: avisos.LinkReserva(sitio, n.Slug)}
	if r.Tipo == Cumpleanios {
		c.Asunto, c.Titulo = "¡Feliz cumple, "+nombre+"!", "¡Feliz cumple, "+nombre+"!"
		c.Parrafos = []string{r.Texto}
	} else {
		c.Asunto, c.Titulo = "Te extrañamos, "+nombre, "Te extrañamos, "+nombre
		hace := "Hace " + strconv.Itoa(d.semanas) + " semanas de tu último turno."
		if d.servicio != "" {
			hace = "Hace " + strconv.Itoa(d.semanas) + " semanas de tu último " + strings.ToLower(d.servicio) + "."
		}
		c.Parrafos = []string{hace, r.Texto}
	}
	if k := d.cupon; k != nil {
		regalo := "Tu regalo"
		if r.Tipo == Extraniamos {
			regalo = "Para la vuelta"
		}
		c.Filas = []avisos.Fila{{Etiqueta: regalo, Valor: promos.DescuentoEnPalabras(*k) + " de descuento"}, {Etiqueta: "Código", Valor: k.Codigo}}
		if k.Hasta != nil {
			c.Filas = append(c.Filas, avisos.Fila{Etiqueta: "Vale hasta", Valor: avisos.Fecha(k.Hasta.In(n.Zona))})
		}
		if k.Condiciones != "" {
			c.Filas = append(c.Filas, avisos.Fila{Etiqueta: "Condiciones", Valor: k.Condiciones})
		}
		c.Link += "?cupon=" + url.QueryEscape(k.Codigo)
	}
	c.Notas = []string{"Te escribimos porque sos cliente de " + n.Nombre + "."}
	return c
}

// cuponVigente del recordatorio (nil si no tiene o ya no sirve: el mail sale sin cupón).
func cuponVigente(ctx context.Context, tx pgx.Tx, r Recordatorio, ahora time.Time) (*promos.Cupon, error) {
	if r.CuponID == nil {
		return nil, nil
	}
	c, err := promos.Leer(ctx, tx, *r.CuponID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil || !promos.Vigente(c, ahora) {
		return nil, err
	}
	return &c, nil
}

// ---- El trabajo del worker

// tandaAutomaticos: tope por negocio y pasada (el resto sale en la siguiente).
const tandaAutomaticos = 300

// Automaticos encola los recordatorios automáticos que tocan hoy (desde las 10 del negocio). El
// cumpleaños sale sólo el día; «hace mucho que no venís» recupera hasta 6 días atrás (si el worker
// estuvo parado, o el cliente acaba de pasar el umbral un domingo). La clave del aviso hace que cada
// ocasión salga una sola vez: un cumpleaños por año, un «te extrañamos» por último turno.
func (s *Servicio) Encolar(ctx context.Context, ahora time.Time) (any, error) {
	encolados := 0
	err := trabajos.PorNegocio(ctx, s.Pool, func(_ string, tx pgx.Tx) error {
		n, err := avisos.LeerNegocioDelMail(ctx, tx)
		if err != nil {
			return err
		}
		local := ahora.In(n.Zona)
		if local.Hour() < HoraAutomaticos {
			return nil
		}
		hoy := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, n.Zona)
		for _, tipo := range []string{Cumpleanios, Extraniamos} {
			r, err := leerRecordatorio(ctx, tx, tipo, ahora)
			if err != nil {
				return err
			}
			if !r.Activo {
				continue
			}
			desde := hoy
			if tipo == Extraniamos {
				desde = hoy.AddDate(0, 0, -6)
			}
			ds, err := candidatos(ctx, tx, r, n.Zona, desde, hoy, ahora)
			if err != nil {
				return err
			}
			hechos := 0
			for _, d := range ds {
				if hechos >= tandaAutomaticos {
					break
				}
				ok, err := avisos.Encolar(ctx, tx, avisos.Aviso{Tipo: tipo, Para: d.Email, Clave: d.clave,
					Datos: map[string]any{"cliente": d.ClienteID, "fecha": d.Fecha}}, ahora)
				if err != nil {
					return err
				}
				if ok {
					encolados++
					hechos++
				}
			}
		}
		return nil
	})
	return map[string]int{"encolados": encolados}, err
}

// armarAutomatico: el mail de un recordatorio automático, con los datos de ahora.
func armar(ctx context.Context, tx pgx.Tx, f avisos.EnCola, ahora time.Time) (correo.Mensaje, string, error) {
	sin := correo.Mensaje{Para: f.Para}
	n, err := avisos.LeerNegocioDelMail(ctx, tx)
	if err != nil {
		return sin, "", err
	}
	r, err := leerRecordatorio(ctx, tx, f.Tipo, ahora)
	if err != nil {
		return sin, "", err
	}
	if !r.Activo {
		return sin, "el negocio apagó este recordatorio", nil
	}
	var nombre, email string
	err = tx.QueryRow(ctx, "SELECT nombre, email FROM clientes WHERE id = $1", f.Datos["cliente"]).Scan(&nombre, &email)
	if errors.Is(err, pgx.ErrNoRows) {
		return sin, "el cliente ya no existe", nil
	}
	if err != nil {
		return sin, "", err
	}
	sin.Para = email
	if email == "" {
		return sin, "el cliente no dejó email", nil
	}
	p, err := avisos.PreferenciasDe(ctx, tx, f.Datos["cliente"])
	if err != nil {
		return sin, "", err
	}
	if !p.Novedades {
		return sin, "el cliente pidió no recibir novedades", nil
	}
	d := datosMail{nombre: nombre}
	if f.Tipo == Extraniamos {
		var ultimo *time.Time
		var servicio *string
		var futuro bool
		if err := tx.QueryRow(ctx, `SELECT (SELECT max(inicio) FROM turnos WHERE cliente_id = $1 AND estado = 'atendido'),
			(SELECT servicio FROM turnos WHERE cliente_id = $1 AND estado = 'atendido' ORDER BY inicio DESC LIMIT 1),
			EXISTS (SELECT 1 FROM turnos WHERE cliente_id = $1 AND inicio >= $2
				AND (estado IN ('pendiente', 'confirmado') OR (estado = 'pendiente_pago' AND vence_en > $2)))`,
			f.Datos["cliente"], ahora).Scan(&ultimo, &servicio, &futuro); err != nil {
			return sin, "", err
		}
		if futuro {
			return sin, "ya tiene otro turno reservado", nil
		}
		if ultimo == nil || ultimo.After(ahora.AddDate(0, 0, -r.Dias+1)) {
			return sin, "volvió hace poco", nil
		}
		d.semanas = int(math.Round(ahora.Sub(*ultimo).Hours() / 24 / 7))
		if servicio != nil {
			d.servicio = *servicio
		}
	}
	if d.cupon, err = cuponVigente(ctx, tx, r, ahora); err != nil {
		return sin, "", err
	}
	return cartaAutomatica(f.Sitio, n, r, d, avisos.LinkPreferencias(f.Sitio, p.Token)).Mensaje(email), "", nil
}

// ---- Panel

// sitio para los links de la vista previa.
func (s *Servicio) sitio() string {
	if s.Sitio != "" {
		return s.Sitio
	}
	return "https://tengolibre.com.ar"
}

func tipoValido(t string) bool { return t == Cumpleanios || t == Extraniamos }

// panelDe arma el recordatorio con la vista previa y la semana (hoy y los 6 días que siguen).
func (s *Servicio) panelDe(ctx context.Context, tx pgx.Tx, r Recordatorio, ahora time.Time) (RecordatorioPanel, error) {
	out := RecordatorioPanel{Recordatorio: r, Semana: []Destinatario{}}
	n, err := avisos.LeerNegocioDelMail(ctx, tx)
	if err != nil {
		return out, err
	}
	local := ahora.In(n.Zona)
	hoy := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, n.Zona)
	ds, err := candidatos(ctx, tx, r, n.Zona, hoy, hoy.AddDate(0, 0, 6), ahora)
	if err != nil {
		return out, err
	}
	if err := marcarEnviados(ctx, tx, ds); err != nil {
		return out, err
	}
	if ds != nil {
		out.Semana = ds
	}
	switch {
	case r.Tipo == Cumpleanios:
		out.SemanaTexto = "Esta semana: " + strconv.Itoa(len(ds)) + " cumpleaños"
	case len(ds) == 1:
		out.SemanaTexto = "Esta semana: 1 cliente"
	default:
		out.SemanaTexto = "Esta semana: " + strconv.Itoa(len(ds)) + " clientes"
	}
	// La vista previa: con el primero de la semana, o con un ejemplo.
	d := datosMail{nombre: "Sofía", semanas: int(math.Round(float64(r.Dias) / 7)), servicio: "corte"}
	para := "sofia@ejemplo.com"
	if len(ds) > 0 {
		d.nombre, para = ds[0].Nombre, ds[0].Email
		if ds[0].servicio != "" {
			d.servicio = ds[0].servicio
		}
	} else if r.Tipo == Extraniamos {
		var srv string
		if err := tx.QueryRow(ctx, "SELECT coalesce((SELECT servicio FROM turnos GROUP BY servicio ORDER BY count(*) DESC LIMIT 1), '')").
			Scan(&srv); err != nil {
			return out, err
		}
		if srv != "" {
			d.servicio = srv
		}
	}
	if d.cupon, err = cuponVigente(ctx, tx, r, ahora); err != nil {
		return out, err
	}
	m := cartaAutomatica(s.sitio(), n, r, d, avisos.LinkPreferencias(s.sitio(), "ejemplo")).Mensaje(para)
	out.VistaPrevia = VistaPrevia{Para: m.Para, Asunto: m.Asunto, HTML: m.HTML, Texto: m.Texto}
	return out, nil
}

func (s *Servicio) recordatorios(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	ahora := s.ahora()
	out := []RecordatorioPanel{}
	err := s.Pool.EnNegocio(ctx, sesion.De(ctx).NegocioID, func(tx pgx.Tx) error {
		for _, tipo := range []string{Cumpleanios, Extraniamos} {
			rec, err := leerRecordatorio(ctx, tx, tipo, ahora)
			if err != nil {
				return err
			}
			p, err := s.panelDe(ctx, tx, rec, ahora)
			if err != nil {
				return err
			}
			out = append(out, p)
		}
		return nil
	})
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusOK, map[string]any{"recordatorios": out})
	return nil
}

// EntradaRecordatorio: lo que se cambia. Lo que no viene, queda. cuponId "" saca el cupón; texto ""
// vuelve al texto por defecto.
type EntradaRecordatorio struct {
	Activo  *bool   `json:"activo"`
	Texto   *string `json:"texto"`
	CuponID *string `json:"cuponId"`
	Dias    *int    `json:"dias"`
}

// aplicar valida la entrada sobre el recordatorio guardado.
func (in EntradaRecordatorio) aplicar(ctx context.Context, tx pgx.Tx, r *Recordatorio) error {
	if in.Activo != nil {
		r.Activo = *in.Activo
	}
	if in.Texto != nil {
		t := strings.TrimSpace(*in.Texto)
		if utf8.RuneCountInString(t) > MaxTextoRecordatorio {
			return problema.Invalido("El texto es corto: hasta 300 letras.", problema.Campo{Ruta: "texto", Mensaje: "muy largo"})
		}
		if t == "" {
			t = textosPorDefecto[r.Tipo]
		}
		r.Texto = t
	}
	if in.Dias != nil {
		if *in.Dias < 14 || *in.Dias > 365 {
			return problema.Invalido("Los días van de 14 a 365.", problema.Campo{Ruta: "dias", Mensaje: "fuera de rango"})
		}
		r.Dias = *in.Dias
	}
	if in.CuponID != nil {
		r.CuponID, r.Cupon = nil, nil
		if *in.CuponID != "" {
			var ok bool
			if err := tx.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM cupones WHERE id::text = $1)", *in.CuponID).Scan(&ok); err != nil {
				return err
			}
			if !ok {
				return problema.Invalido("Ese cupón no existe.", problema.Campo{Ruta: "cuponId", Mensaje: "no existe"})
			}
			r.CuponID = in.CuponID
		}
	}
	r.Regla = regla(r.Tipo, r.Dias)
	return nil
}

func (s *Servicio) guardarRecordatorio(w http.ResponseWriter, r *http.Request) error {
	tipo := r.PathValue("tipo")
	if !tipoValido(tipo) {
		return problema.NoEncontrado("Ese recordatorio no existe: es cumpleanios o extraniamos.")
	}
	var in EntradaRecordatorio
	if err := web.Leer(w, r, &in); err != nil {
		return err
	}
	ctx := r.Context()
	ahora := s.ahora()
	var out RecordatorioPanel
	err := s.Pool.EnNegocio(ctx, sesion.De(ctx).NegocioID, func(tx pgx.Tx) error {
		antes, err := leerRecordatorio(ctx, tx, tipo, ahora)
		if err != nil {
			return err
		}
		rec := antes
		if err := in.aplicar(ctx, tx, &rec); err != nil {
			return err
		}
		texto := rec.Texto
		if texto == textosPorDefecto[tipo] {
			texto = ""
		}
		if _, err := tx.Exec(ctx, `INSERT INTO recordatorios (negocio_id, tipo, activo, texto, cupon_id, dias)
			VALUES (app.negocio_actual(), $1, $2, $3, $4::uuid, $5)
			ON CONFLICT (negocio_id, tipo) DO UPDATE SET activo = excluded.activo, texto = excluded.texto, cupon_id = excluded.cupon_id,
				dias = excluded.dias, actualizado_en = now()`, tipo, rec.Activo, texto, rec.CuponID, rec.Dias); err != nil {
			return err
		}
		if rec, err = leerRecordatorio(ctx, tx, tipo, ahora); err != nil {
			return err
		}
		if err := auditoria.Anotar(ctx, tx, auditoria.Registro{Accion: "recordatorio." + tipo, Entidad: "recordatorio",
			Detalle: map[string]any{"antes": antes, "despues": rec}}); err != nil {
			return err
		}
		out, err = s.panelDe(ctx, tx, rec, ahora)
		return err
	})
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusOK, out)
	return nil
}

// vistaPrevia del mail con un borrador (?texto=&cuponId=&dias=), sin guardar nada.
func (s *Servicio) vistaPrevia(w http.ResponseWriter, r *http.Request) error {
	tipo := r.PathValue("tipo")
	if !tipoValido(tipo) {
		return problema.NoEncontrado("Ese recordatorio no existe: es cumpleanios o extraniamos.")
	}
	q := r.URL.Query()
	var in EntradaRecordatorio
	if q.Has("texto") {
		t := q.Get("texto")
		in.Texto = &t
	}
	if q.Has("cuponId") {
		c := q.Get("cuponId")
		in.CuponID = &c
	}
	if v := q.Get("dias"); v != "" {
		d, err := strconv.Atoi(v)
		if err != nil {
			return problema.Invalido("dias va como número.")
		}
		in.Dias = &d
	}
	ctx := r.Context()
	ahora := s.ahora()
	var out VistaPrevia
	err := s.Pool.EnNegocio(ctx, sesion.De(ctx).NegocioID, func(tx pgx.Tx) error {
		rec, err := leerRecordatorio(ctx, tx, tipo, ahora)
		if err != nil {
			return err
		}
		if err := in.aplicar(ctx, tx, &rec); err != nil {
			return err
		}
		p, err := s.panelDe(ctx, tx, rec, ahora)
		out = p.VistaPrevia
		return err
	})
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusOK, out)
	return nil
}
