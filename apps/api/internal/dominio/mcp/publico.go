package mcp

// «La IA de tus clientes»: el servidor MCP público, sin clave. El asistente de IA de cualquier
// persona (Claude, ChatGPT, Gemini…) busca los negocios activos que lo prendieron en el panel, mira
// servicios, profesionales y horarios libres, y reserva con las mismas reglas que la página pública:
// penalidades por ausencias, seña con el link de Mercado Pago que abre la persona y el link
// /turno/{codigo} para verlo, cancelarlo o cambiarlo. Esos turnos llevan origen "ia".
//
// Mismo transporte que el del negocio (Streamable HTTP, JSON simple, sin sesión). Sin clave no hay
// registro por negocio: lo que se ve en el panel sale de los turnos con origen "ia". Cada IP tiene
// un tope de pedidos y otro, más chico, de reservas.

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/avisos"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/miweb"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/negocios"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/turnos"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/auditoria"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/problema"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/web"
)

// Topes por IP del servidor público (por defecto; los tests los achican).
const (
	PedidosPorMinuto = 60
	ReservasPorHora  = 5
)

// URLPublica: la del servidor público, la que la persona carga en su asistente.
func (s *Servicio) URLPublica() string { return s.url() + "/publico" }

// ---- Tope por IP

// limite cuenta pedidos por clave en una ventana fija. En memoria: la API es un solo proceso.
type limite struct {
	mu      sync.Mutex
	cuentas map[string]*cuenta
}

type cuenta struct {
	desde time.Time
	n     int
}

// permitir suma un pedido de clave y dice si entra en max por ventana.
func (l *limite) permitir(clave string, max int, ventana time.Duration, ahora time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.cuentas == nil {
		l.cuentas = map[string]*cuenta{}
	}
	// Limpieza de lo vencido cuando crece (no hace falta un barrido aparte).
	if len(l.cuentas) > 10000 {
		for k, c := range l.cuentas {
			if ahora.Sub(c.desde) > time.Hour {
				delete(l.cuentas, k)
			}
		}
	}
	c, ok := l.cuentas[clave]
	if !ok || ahora.Sub(c.desde) >= ventana {
		l.cuentas[clave] = &cuenta{desde: ahora, n: 1}
		return max > 0
	}
	c.n++
	return c.n <= max
}

// ipDe: la del cliente. La API sólo escucha detrás de Caddy, que sólo acepta a Cloudflare: el
// CF-Connecting-IP lo pone Cloudflare. Sin él (desarrollo, tests), la dirección de la conexión.
func ipDe(r *http.Request) string {
	if v := strings.TrimSpace(r.Header.Get("CF-Connecting-IP")); v != "" && net.ParseIP(v) != nil {
		return v
	}
	if h, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return h
	}
	return r.RemoteAddr
}

func (s *Servicio) topes() (pedidos, reservas int) {
	pedidos, reservas = PedidosPorMinuto, ReservasPorHora
	if s.TopePedidos > 0 {
		pedidos = s.TopePedidos
	}
	if s.TopeReservas > 0 {
		reservas = s.TopeReservas
	}
	return
}

func demasiados(w http.ResponseWriter, detalle string) {
	w.Header().Set("Retry-After", "60")
	problema.Escribir(w, &problema.Problema{Estado: http.StatusTooManyRequests, Codigo: "demasiados_pedidos",
		Titulo: "Demasiados pedidos", Detalle: detalle})
}

// ---- Transporte

// servirPublico: el POST del asistente de una persona. Sin clave y sin cookies, así que cualquier
// origen puede llamarlo (no hay nada de nadie que proteger del DNS rebinding).
func (s *Servicio) servirPublico(w http.ResponseWriter, r *http.Request) {
	if v := r.Header.Get("MCP-Protocol-Version"); v != "" && !slices.Contains(versiones, v) {
		problema.Escribir(w, problema.Invalido("No conocemos la versión "+v+" del protocolo MCP."))
		return
	}
	ip := ipDe(r)
	pedidos, _ := s.topes()
	if !s.limite.permitir("p:"+ip, pedidos, time.Minute, s.ahora()) {
		demasiados(w, "Hiciste muchos pedidos seguidos. Esperá un minuto.")
		return
	}
	s.rpc(w, r, func(ctx context.Context, m mensaje) *respuesta { return s.atenderPublico(ctx, ip, m) })
}

func (s *Servicio) atenderPublico(ctx context.Context, ip string, m mensaje) *respuesta {
	res := abrir(m)
	if res == nil || res.Error != nil {
		return res
	}
	var err *errorRPC
	switch m.Metodo {
	case "initialize":
		res.Result, err = inicializarPublico(m.Params)
	case "ping":
		res.Result = map[string]any{}
	case "tools/list":
		res.Result = map[string]any{"tools": listarPublicas()}
	case "tools/call":
		res.Result, err = s.llamarPublica(ctx, ip, m.Params)
	default:
		err = &errorRPC{errMetodo, "Método desconocido: " + m.Metodo}
	}
	if err != nil {
		res.Result, res.Error = nil, err
	}
	return res
}

func inicializarPublico(params json.RawMessage) (any, *errorRPC) {
	var in struct {
		ProtocolVersion string `json:"protocolVersion"`
	}
	if len(params) > 0 && json.Unmarshal(params, &in) != nil {
		return nil, &errorRPC{errParametro, "Los parámetros de initialize no son válidos."}
	}
	version := versiones[0]
	for _, v := range versiones {
		if v == in.ProtocolVersion {
			version = v
		}
	}
	return map[string]any{
		"protocolVersion": version,
		"capabilities":    map[string]any{"tools": map[string]any{"listChanged": false}},
		"serverInfo":      map[string]any{"name": "tengolibre-publico", "title": "Tengo Libre · Sacar turno", "version": "1.0.0"},
		"instructions": "Tengo Libre es un sistema de turnos para barberías, salones, estética y otros negocios de servicios de " +
			"Argentina. Con estas herramientas ayudás a una persona a encontrar un negocio y sacar turno: buscar_negocios, " +
			"después ver_servicios_y_profesionales y buscar_horarios_libres con el slug del negocio. Las fechas van como " +
			"AAAA-MM-DD y las horas como HH:MM, en la hora del negocio. Antes de reservar_turno, contale a la persona qué vas " +
			"a reservar (servicio, día, hora, con quién y el precio) y esperá que te diga que sí; pedile su nombre y un " +
			"teléfono o un email. Si la reserva pide seña, pasale el link de pago: el turno se guarda sólo hasta que pague. " +
			"Siempre pasale el link del turno, donde lo puede ver, cambiar o cancelar.",
	}, nil
}

// ---- Herramientas

type publica struct {
	nombre, titulo, descripcion string
	esquema                     map[string]any
	escribe                     bool
	correr                      func(ctx context.Context, s *Servicio, ip string, args json.RawMessage) (any, error)
}

var publicas []publica

func init() {
	negocio := texto("El slug del negocio (sale de buscar_negocios), por ejemplo «el-corte».")
	publicas = []publica{
		{nombre: "buscar_negocios", titulo: "Buscar negocios", correr: buscarNegocios,
			descripcion: "Busca negocios en Tengo Libre que toman turnos por asistentes de IA: por rubro, ciudad o texto " +
				"(nombre, descripción o un servicio, como «corte» o «uñas»). Devuelve el slug que piden las demás herramientas.",
			esquema: objeto(nil, map[string]any{
				"texto":  texto("Parte del nombre del negocio o de un servicio."),
				"rubro":  map[string]any{"type": "string", "enum": miweb.Rubros, "description": "El rubro."},
				"ciudad": texto("Ciudad o provincia."),
				"limite": map[string]any{"type": "integer", "minimum": 1, "maximum": 20, "description": "Cuántos traer. Por defecto, 10."},
			})},
		{nombre: "ver_servicios_y_profesionales", titulo: "Ver servicios y profesionales", correr: verNegocio,
			descripcion: "Muestra un negocio: dónde queda, su política de cancelación, los servicios (duración y precio) y " +
				"los profesionales (qué servicios hacen y en qué horario atienden), con los ids que pide la reserva.",
			esquema: objeto([]string{"negocio"}, map[string]any{"negocio": negocio})},
		{nombre: "buscar_horarios_libres", titulo: "Buscar horarios libres", correr: libresPublicos,
			descripcion: "Busca los horarios libres de un servicio de un negocio, con cualquier profesional o con uno en " +
				"particular, en un día o en los que siguen (hasta 7). Respeta las reglas de reserva del negocio.",
			esquema: objeto([]string{"negocio", "servicioId", "fecha"}, map[string]any{
				"negocio":       negocio,
				"servicioId":    texto("El servicio (id o nombre exacto)."),
				"fecha":         fecha("Día, AAAA-MM-DD."),
				"dias":          map[string]any{"type": "integer", "minimum": 1, "maximum": 7, "description": "Cuántos días mirar desde la fecha. Por defecto, 1."},
				"profesionalId": texto("Sólo con este profesional (id o nombre)."),
			})},
		{nombre: "reservar_turno", titulo: "Reservar un turno", escribe: true, correr: reservarPublica,
			descripcion: "Reserva un turno a nombre de la persona, como desde la página del negocio. El horario tiene que " +
				"estar libre (buscalo antes). Devuelve el link del turno y, si el negocio pide seña, el link de Mercado Pago " +
				"para pagarla: sin pagar, el horario se libera. Confirmá con la persona antes de usarla.",
			esquema: objeto([]string{"negocio", "servicioId", "fecha", "hora", "nombre"}, map[string]any{
				"negocio":       negocio,
				"servicioId":    texto("El servicio (id o nombre exacto)."),
				"fecha":         fecha("Día, AAAA-MM-DD."),
				"hora":          hora("Hora de inicio, HH:MM, en la hora del negocio."),
				"profesionalId": texto("Con quién (id o nombre). Sin esto, el primero libre."),
				"nombre":        texto("Nombre y apellido de la persona."),
				"telefono":      texto("Su teléfono (hace falta esto o el email)."),
				"email":         texto("Su email (hace falta esto o el teléfono)."),
				"nota":          texto("Algo para el negocio (hasta 500 caracteres)."),
			})},
	}
}

func listarPublicas() []map[string]any {
	out := []map[string]any{}
	for _, h := range publicas {
		out = append(out, map[string]any{
			"name": h.nombre, "title": h.titulo, "description": h.descripcion, "inputSchema": h.esquema,
			"annotations": map[string]any{"title": h.titulo, "readOnlyHint": !h.escribe, "destructiveHint": false,
				"idempotentHint": !h.escribe, "openWorldHint": true},
		})
	}
	return out
}

func (s *Servicio) llamarPublica(ctx context.Context, ip string, params json.RawMessage) (any, *errorRPC) {
	var in struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	}
	if err := json.Unmarshal(params, &in); err != nil || in.Name == "" {
		return nil, &errorRPC{errParametro, "tools/call necesita name."}
	}
	if len(in.Arguments) == 0 || string(in.Arguments) == "null" {
		in.Arguments = json.RawMessage("{}")
	}
	var h *publica
	for i := range publicas {
		if publicas[i].nombre == in.Name {
			h = &publicas[i]
		}
	}
	if h == nil {
		return nil, &errorRPC{errParametro, "La herramienta " + in.Name + " no existe."}
	}
	res, err := h.correr(ctx, s, ip, in.Arguments)
	if err != nil {
		return resultado(map[string]any{"error": explicar(err, h.nombre)}, true), nil
	}
	return resultado(res, false), nil
}

// negocioIA: del slug a la conexión del negocio, sólo si está activo y tiene la IA pública prendida.
func (s *Servicio) negocioIA(ctx context.Context, slug string) (*conexion, string, error) {
	slug = strings.ToLower(strings.TrimSpace(slug))
	noEsta := problema.NoEncontrado("No encontramos ese negocio, o no toma turnos por asistentes de IA. Buscalo con buscar_negocios.")
	if slug == "" {
		return nil, "", problema.Invalido("Falta el negocio (su slug, de buscar_negocios).")
	}
	var id *string
	if err := s.Pool.EnTx(ctx, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, "SELECT app.negocio_publico($1)::text", slug).Scan(&id)
	}); err != nil {
		return nil, "", err
	}
	if id == nil {
		return nil, "", noEsta
	}
	c := &conexion{negocio: *id, nombre: "IA pública"}
	var activa bool
	var zona, nombre string
	if err := s.Pool.EnNegocio(ctx, *id, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, "SELECT ia_publica, zona_horaria, nombre FROM negocios WHERE id = app.negocio_actual()").Scan(&activa, &zona, &nombre)
	}); err != nil {
		return nil, "", err
	}
	if !activa {
		return nil, "", noEsta
	}
	z, err := time.LoadLocation(zona)
	if err != nil {
		z = time.FixedZone("UTC-3", -3*3600)
	}
	c.zona = z
	return c, nombre, nil
}

// sinNegocio: los argumentos sin "negocio", para pasárselos a las herramientas del negocio.
func sinNegocio(raw json.RawMessage) (string, json.RawMessage, error) {
	var m map[string]json.RawMessage
	if err := json.Unmarshal(raw, &m); err != nil {
		return "", nil, problema.Invalido("Los argumentos no son válidos: " + err.Error())
	}
	var slug string
	if v, ok := m["negocio"]; ok {
		if err := json.Unmarshal(v, &slug); err != nil {
			return "", nil, problema.Invalido("negocio va como texto (el slug).")
		}
	}
	delete(m, "negocio")
	b, _ := json.Marshal(m)
	return slug, b, nil
}

// FichaIA: un negocio para el asistente.
type FichaIA struct {
	Slug        string   `json:"slug"`
	Nombre      string   `json:"nombre"`
	Descripcion string   `json:"descripcion,omitempty"`
	Rubro       string   `json:"rubro,omitempty"`
	Ciudad      string   `json:"ciudad,omitempty"`
	Provincia   string   `json:"provincia,omitempty"`
	Direccion   string   `json:"direccion,omitempty"`
	Servicios   []string `json:"servicios"`
	Desde       *int64   `json:"precioDesde"`
	Pagina      string   `json:"pagina"`
}

func buscarNegocios(ctx context.Context, s *Servicio, _ string, raw json.RawMessage) (any, error) {
	var in struct {
		Texto, Rubro, Ciudad string
		Limite               int
	}
	if err := leerArgs(raw, &in); err != nil {
		return nil, err
	}
	for _, v := range []*string{&in.Texto, &in.Rubro, &in.Ciudad} {
		if *v = strings.TrimSpace(*v); len([]rune(*v)) > 80 {
			return nil, problema.Invalido("La búsqueda es demasiado larga.")
		}
	}
	if in.Limite <= 0 {
		in.Limite = 10
	}
	var crudo []byte
	if err := s.Pool.EnTx(ctx, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, "SELECT app.directorio_ia($1, $2, $3, $4)", in.Rubro, in.Ciudad, in.Texto, min(in.Limite, 20)).Scan(&crudo)
	}); err != nil {
		return nil, err
	}
	out := []FichaIA{}
	if err := json.Unmarshal(crudo, &out); err != nil {
		return nil, err
	}
	for i := range out {
		out[i].Pagina = s.sitio() + "/" + out[i].Slug
	}
	return map[string]any{"negocios": out}, nil
}

func verNegocio(ctx context.Context, s *Servicio, _ string, raw json.RawMessage) (any, error) {
	slug, resto, err := sinNegocio(raw)
	if err != nil {
		return nil, err
	}
	c, _, err := s.negocioIA(ctx, slug)
	if err != nil {
		return nil, err
	}
	cat, _, err := verCatalogo(ctx, s, c, resto)
	if err != nil {
		return nil, err
	}
	out := cat.(map[string]any)
	if err := s.Pool.EnNegocio(ctx, c.negocio, func(tx pgx.Tx) error {
		n, err := negocios.Leer(ctx, tx)
		if err != nil {
			return err
		}
		a, err := negocios.LeerAjustes(ctx, tx)
		if err != nil {
			return err
		}
		out["negocio"] = map[string]any{"slug": n.Slug, "nombre": n.Nombre, "descripcion": n.Descripcion, "direccion": n.Direccion,
			"ciudad": n.Ciudad, "pagina": s.sitio() + "/" + n.Slug}
		out["politicaCancelacion"] = a.PoliticaCancelacion
		out["cancelarHastaHorasAntes"] = a.CancelarHastaHoras
		out["diasQueSePuedenReservar"] = a.DiasVista
		return nil
	}); err != nil {
		return nil, err
	}
	return out, nil
}

func libresPublicos(ctx context.Context, s *Servicio, _ string, raw json.RawMessage) (any, error) {
	slug, resto, err := sinNegocio(raw)
	if err != nil {
		return nil, err
	}
	c, _, err := s.negocioIA(ctx, slug)
	if err != nil {
		return nil, err
	}
	out, _, err := buscarLibres(ctx, s, c, resto)
	return out, err
}

func reservarPublica(ctx context.Context, s *Servicio, ip string, raw json.RawMessage) (any, error) {
	slug, resto, err := sinNegocio(raw)
	if err != nil {
		return nil, err
	}
	var in struct {
		ServicioID, Fecha, Hora, ProfesionalID, Nombre, Telefono, Email, Nota string
	}
	if err := leerArgs(resto, &in); err != nil {
		return nil, err
	}
	if in.ServicioID == "" {
		return nil, problema.Invalido("Falta el servicio.")
	}
	if strings.TrimSpace(in.Nombre) == "" {
		return nil, problema.Invalido("Falta el nombre de la persona.")
	}
	if strings.TrimSpace(in.Telefono) == "" && strings.TrimSpace(in.Email) == "" {
		return nil, problema.Invalido("Falta un teléfono o un email de la persona, para avisarle del turno.")
	}
	c, nombre, err := s.negocioIA(ctx, slug)
	if err != nil {
		return nil, err
	}
	inicio, err := c.momento(in.Fecha, in.Hora)
	if err != nil {
		return nil, err
	}
	srv, prof, _, err := s.resolverAmbos(ctx, c, in.ServicioID, in.ProfesionalID)
	if err != nil {
		return nil, err
	}
	// El tope de reservas se cuenta recién con un pedido completo: equivocarse en un dato no gasta.
	if _, reservas := s.topes(); !s.limite.permitir("r:"+ip, reservas, time.Hour, s.ahora()) {
		return nil, &problema.Problema{Estado: http.StatusTooManyRequests, Codigo: "demasiadas_reservas",
			Titulo: "Demasiadas reservas", Detalle: "Ya se hicieron muchas reservas desde esta conexión. Probá en un rato o reservá en la página del negocio."}
	}
	t, err := s.Turnos.ReservarIA(ctx, c.negocio, turnos.Pedido{ServicioID: srv, ProfesionalID: prof, Inicio: inicio, Nota: in.Nota,
		Cliente: turnos.Cliente{Nombre: in.Nombre, Telefono: in.Telefono, Email: in.Email}})
	if err != nil {
		return nil, err
	}
	link := s.sitio() + "/turno/" + t.Codigo
	out := map[string]any{
		"negocio": nombre,
		"turno": map[string]any{"codigo": t.Codigo, "dia": avisos.Fecha(t.Inicio.In(c.zona)), "inicio": t.Inicio.In(c.zona).Format(time.RFC3339),
			"servicio": t.Servicio, "profesional": t.Profesional, "precio": t.Precio, "estado": t.Estado},
		"link": link,
	}
	switch {
	case t.Pago != nil && t.Pago.URL != "":
		p := map[string]any{"monto": t.Pago.Monto, "modo": t.Pago.Modo, "url": t.Pago.URL}
		if t.VenceEn != nil {
			p["pagarAntesDe"] = t.VenceEn.In(c.zona).Format(time.RFC3339)
		}
		out["pago"] = p
		out["mensaje"] = "El turno queda guardado hasta que pague la seña con el link de Mercado Pago. Después, todo en " + link + "."
	case t.Pago != nil:
		out["pago"] = map[string]any{"monto": t.Pago.Monto, "modo": t.Pago.Modo, "url": link}
		out["mensaje"] = "El turno pide seña: el link para pagarla está en " + link + "."
	default:
		out["mensaje"] = "Listo: el turno quedó reservado. Lo puede ver, cambiar o cancelar en " + link + "."
	}
	return out, nil
}

// ---- Panel: prender, apagar y contar

// UltimaIA: una reserva que hizo el asistente de una persona, para «Lo último que hizo la IA».
type UltimaIA struct {
	TurnoID  string    `json:"turnoId"`
	Resumen  string    `json:"resumen"`
	CreadoEn time.Time `json:"creadoEn"`
}

// IAPublica como la ve el panel.
type IAPublica struct {
	Activo bool `json:"activo"`
	// ReservasMes: los turnos que llegaron por la IA este mes (sin los que vencieron sin pagar).
	ReservasMes int        `json:"reservasMes"`
	Ultimas     []UltimaIA `json:"ultimas"`
	URL         string     `json:"url"`
}

func (s *Servicio) leerPublico(w http.ResponseWriter, r *http.Request) error {
	out := IAPublica{Ultimas: []UltimaIA{}, URL: s.URLPublica()}
	err := s.enNegocio(r, func(ctx context.Context, tx pgx.Tx) error {
		var zona string
		if err := tx.QueryRow(ctx, "SELECT ia_publica, zona_horaria FROM negocios WHERE id = app.negocio_actual()").Scan(&out.Activo, &zona); err != nil {
			return err
		}
		z, err := time.LoadLocation(zona)
		if err != nil {
			z = time.FixedZone("UTC-3", -3*3600)
		}
		// El mes con el reloj de la base, el mismo que puso creado_en.
		if err := tx.QueryRow(ctx, `SELECT count(*) FROM turnos WHERE origen = 'ia'
			AND creado_en >= date_trunc('month', now() AT TIME ZONE $1) AT TIME ZONE $1
			AND NOT (estado = 'pendiente_pago' AND vence_en <= now())`, z.String()).Scan(&out.ReservasMes); err != nil {
			return err
		}
		filas, err := tx.Query(ctx, `SELECT t.id::text, t.servicio, t.inicio, t.creado_en FROM turnos t
			WHERE t.origen = 'ia' ORDER BY t.creado_en DESC LIMIT 5`)
		if err != nil {
			return err
		}
		defer filas.Close()
		for filas.Next() {
			var u UltimaIA
			var servicio string
			var inicio time.Time
			if err := filas.Scan(&u.TurnoID, &servicio, &inicio, &u.CreadoEn); err != nil {
				return err
			}
			// Como en el diseño: «reservó Laminado el jueves 5, 15:30».
			en := inicio.In(z)
			u.Resumen = "reservó " + servicio + " el " + strings.SplitN(avisos.Fecha(en), " de ", 2)[0] + ", " + avisos.Hora(en)
			out.Ultimas = append(out.Ultimas, u)
		}
		return filas.Err()
	})
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusOK, out)
	return nil
}

func (s *Servicio) guardarPublico(w http.ResponseWriter, r *http.Request) error {
	var in struct {
		Activo *bool `json:"activo"`
	}
	if err := web.Leer(w, r, &in); err != nil {
		return err
	}
	if in.Activo == nil {
		return problema.Invalido("Falta activo.", problema.Campo{Ruta: "activo", Mensaje: "requerido"})
	}
	err := s.enNegocio(r, func(ctx context.Context, tx pgx.Tx) error {
		var antes bool
		if err := tx.QueryRow(ctx, "SELECT ia_publica FROM negocios WHERE id = app.negocio_actual() FOR UPDATE").Scan(&antes); err != nil {
			return err
		}
		if antes == *in.Activo {
			return nil
		}
		if _, err := tx.Exec(ctx, "UPDATE negocios SET ia_publica = $1 WHERE id = app.negocio_actual()", *in.Activo); err != nil {
			return err
		}
		accion := "ia_publica.apagada"
		if *in.Activo {
			accion = "ia_publica.prendida"
		}
		return auditoria.Anotar(ctx, tx, auditoria.Registro{Accion: accion, Entidad: "ajustes"})
	})
	if err != nil {
		return err
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}
