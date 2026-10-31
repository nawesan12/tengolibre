package espera_test

import (
	"context"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/avisos"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/catalogo"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/disponibilidad"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/espera"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/negocios"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/turnos"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/correo"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/prueba"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/servidor"
)

type P = prueba.Pedido

const (
	duenia = "user_duenia0001"
	sitio  = "https://tengolibre.com.ar"
)

func en(s string) time.Time {
	t, err := time.ParseInLocation("2006-01-02 15:04", s, disponibilidad.ZonaPorDefecto)
	if err != nil {
		panic(err)
	}
	return t
}

type entorno struct {
	*prueba.API
	negocio, servicio, ana, beto string
	mails                        *correo.Memoria
	env                          *avisos.Enviador
	es                           *espera.Servicio
}

// armar: un negocio con un servicio de 30' y dos profesionales que atienden los lunes de 9 a 11, la
// lista de espera escuchando los horarios liberados y reservando con turnos. Ahora: domingo 1/11 10:00.
func armar(t *testing.T) *entorno {
	e := &entorno{mails: &correo.Memoria{}}
	e.API = prueba.Nueva(t, en("2026-11-01 10:00"), func(a *prueba.API) []servidor.Dominio {
		app := a.Bases.App
		av := &avisos.Servicio{Pool: app, Sesion: a.Sesion, Ahora: a.Reloj(), AlLiberarse: espera.Liberado}
		tur := &turnos.Servicio{Pool: app, Sesion: a.Sesion, Ahora: a.Reloj(), Avisar: av.Avisar}
		e.es = &espera.Servicio{Pool: app, Sesion: a.Sesion, Ahora: a.Reloj(), Turnos: tur}
		return []servidor.Dominio{
			&negocios.Servicio{Pool: app, Sesion: a.Sesion},
			&catalogo.Servicio{Pool: app, Sesion: a.Sesion},
			tur, av, e.es,
		}
	})
	e.env = &avisos.Enviador{Pool: e.Bases.App, Correo: e.mails, Sitio: sitio}
	var n negocios.Negocio
	e.Esperar(201, P{Metodo: "POST", Ruta: "/v1/negocios", Quien: duenia, Cuerpo: map[string]string{"slug": "el-corte", "nombre": "El Corte"}}, &n)
	e.negocio = n.ID
	var s catalogo.Prestacion
	e.panel(201, "POST", "/v1/panel/servicios", map[string]any{"nombre": "Corte", "duracion": 30, "precio": 8000}, &s)
	e.servicio = s.ID
	horario := []disponibilidad.Franja{{Dia: 1, Desde: "09:00", Hasta: "11:00"}}
	var pa, pb catalogo.Profesional
	e.panel(201, "POST", "/v1/panel/profesionales", map[string]any{"nombre": "Ana", "servicios": []string{s.ID}, "horario": horario, "orden": 1}, &pa)
	e.panel(201, "POST", "/v1/panel/profesionales", map[string]any{"nombre": "Beto", "servicios": []string{s.ID}, "horario": horario, "orden": 2}, &pb)
	e.ana, e.beto = pa.ID, pb.ID
	return e
}

func (e *entorno) panel(estado int, metodo, ruta string, cuerpo, v any) prueba.Respuesta {
	e.T.Helper()
	return e.Esperar(estado, P{Metodo: metodo, Ruta: ruta, Quien: duenia, Negocio: e.negocio, Cuerpo: cuerpo}, v)
}

func (e *entorno) anotar(nombre, profesional, momento string) {
	e.T.Helper()
	e.Esperar(201, P{Metodo: "POST", Ruta: "/v1/p/el-corte/espera", Cuerpo: map[string]any{"servicioId": e.servicio,
		"profesionalId": profesional, "nombre": nombre, "email": strings.ToLower(nombre) + "@ejemplo.com", "telefono": "11" + strings.Repeat("5", len(nombre)),
		"desde": "2026-11-02", "momento": momento}}, nil)
}

func (e *entorno) reservar(prof, hora, nombre, tel string) turnos.Turno {
	e.T.Helper()
	var t turnos.Turno
	e.Esperar(201, P{Metodo: "POST", Ruta: "/v1/p/el-corte/turnos", Cuerpo: map[string]any{"servicioId": e.servicio, "profesionalId": prof,
		"inicio": en("2026-11-02 " + hora).Format(time.RFC3339), "cliente": map[string]string{"nombre": nombre, "telefono": tel}}}, &t)
	return t
}

func (e *entorno) ofertas(q string) []espera.Oferta {
	e.T.Helper()
	var r struct {
		Ofertas []espera.Oferta `json:"ofertas"`
	}
	e.panel(200, "GET", "/v1/panel/espera/ofertas"+q, nil, &r)
	return r.Ofertas
}

// link: el token del último mail de la oferta que le llegó a esa persona.
func (e *entorno) link(email string) string {
	e.T.Helper()
	if _, err := e.env.Pasada(context.Background(), e.Ahora); err != nil {
		e.T.Fatal(err)
	}
	ms := e.mails.Para(email)
	if len(ms) == 0 {
		e.T.Fatalf("no le llegó nada a %s", email)
	}
	m := regexp.MustCompile(regexp.QuoteMeta(sitio+"/espera/") + `([A-Za-z0-9_-]+)`).FindStringSubmatch(ms[len(ms)-1].Texto)
	if m == nil {
		e.T.Fatalf("el mail no tiene el link: %s", ms[len(ms)-1].Texto)
	}
	return m[1]
}

func (e *entorno) campanita() []avisos.NotificacionPanel {
	e.T.Helper()
	var c struct {
		Notificaciones []avisos.NotificacionPanel `json:"notificaciones"`
	}
	e.panel(200, "GET", "/v1/panel/campanita", nil, &c)
	return c.Notificaciones
}

// notificacion: la de la campanita que empieza con ese título (varias de una misma transacción
// comparten la hora, así que no se mira el orden).
func (e *entorno) notificacion(titulo string) avisos.NotificacionPanel {
	e.T.Helper()
	cs := e.campanita()
	for _, c := range cs {
		if strings.HasPrefix(c.Titulo, titulo) {
			return c
		}
	}
	e.T.Fatalf("no hay «%s» en la campanita: %+v", titulo, cs)
	return avisos.NotificacionPanel{}
}

func (e *entorno) esperando() []espera.Anotado {
	e.T.Helper()
	var l struct {
		Espera []espera.Anotado `json:"espera"`
	}
	e.panel(200, "GET", "/v1/panel/espera", nil, &l)
	return l.Espera
}

func TestOfrecerElHuecoSolo(t *testing.T) {
	e := armar(t)
	ctx := context.Background()
	t1 := e.reservar(e.ana, "09:00", "Lucía", "1111")
	// Les sirve el lunes a la mañana con Ana: Martina, Lola (cualquiera) y Nora. A Camila sólo a la tarde.
	e.anotar("Martina", e.ana, "manana")
	e.anotar("Lola", "", "")
	e.anotar("Camila", e.ana, "tarde")
	e.anotar("Nora", e.ana, "cualquiera")
	e.Esperar(400, P{Metodo: "POST", Ruta: "/v1/p/el-corte/espera", Cuerpo: map[string]any{"servicioId": e.servicio,
		"nombre": "Mal", "email": "mal@ejemplo.com", "desde": "2026-11-02", "momento": "noche"}}, nil)

	// Lucía cancela: el hueco se le ofrece sólo a Martina, con 30 minutos (el ajuste por defecto).
	e.Esperar(200, P{Metodo: "POST", Ruta: "/v1/t/" + t1.Codigo + "/cancelar"}, nil)
	os := e.ofertas("")
	if len(os) != 1 || os[0].Estado != "ofreciendo" || os[0].Origen != "automatica" || len(os[0].Cola) != 3 ||
		os[0].Actual == nil || os[0].Actual.Nombre != "Martina" || !os[0].Actual.VenceEn.Equal(en("2026-11-01 10:30")) ||
		os[0].Sigue == nil || os[0].Sigue.Nombre != "Lola" || os[0].Precio != 8000 || !os[0].Inicio.Equal(en("2026-11-02 09:00")) {
		t.Fatalf("ofertas: %+v", os)
	}
	oferta := os[0].ID
	tokMartina := e.link("martina@ejemplo.com")
	if n := len(e.mails.Para("lola@ejemplo.com")); n != 0 {
		t.Fatalf("a Lola no le toca todavía y le llegaron %d", n)
	}
	m := e.mails.Para("martina@ejemplo.com")[0]
	if m.Asunto != "Se liberó lo que esperabas — El Corte" || !strings.Contains(m.Texto, "antes de las 10:30") || !strings.Contains(m.Texto, "Precio: $ 8.000") {
		t.Fatalf("mail: %s", m.Texto)
	}
	var pub espera.OfertaPublica
	e.Esperar(200, P{Metodo: "GET", Ruta: "/v1/espera/" + tokMartina}, &pub)
	if pub.Estado != "ofrecida" || pub.Nombre != "Martina" || pub.Negocio.Nombre != "El Corte" || pub.Servicio != "Corte" || pub.Profesional != "Ana" ||
		pub.VenceEn == nil || pub.Turno != nil {
		t.Fatalf("link: %+v", pub)
	}
	e.Esperar(404, P{Metodo: "GET", Ruta: "/v1/espera/token-inventado-0000000"}, nil)

	// «No me sirve»: pasa a Lola y Martina vuelve a esperar.
	e.Esperar(200, P{Metodo: "POST", Ruta: "/v1/espera/" + tokMartina + "/pasar"}, &pub)
	if pub.Estado != "paso" {
		t.Fatalf("pasar: %+v", pub)
	}
	e.Esperar(409, P{Metodo: "POST", Ruta: "/v1/espera/" + tokMartina + "/tomar"}, nil)
	var o espera.Oferta
	e.panel(200, "GET", "/v1/panel/espera/ofertas/"+oferta, nil, &o)
	if o.Actual == nil || o.Actual.Nombre != "Lola" || o.Sigue == nil || o.Sigue.Nombre != "Nora" || o.Cola[0].Estado != "paso" {
		t.Fatalf("después de pasar: %+v", o)
	}
	tokLola := e.link("lola@ejemplo.com")
	var volvio bool
	for _, a := range e.esperando() {
		if a.Nombre == "Martina" && a.AvisadoEn == nil && a.Resultado != nil && *a.Resultado == "paso" {
			volvio = true
		}
	}
	if !volvio {
		t.Fatalf("Martina no volvió a esperar: %+v", e.esperando())
	}

	// Lola no responde: a los 31 minutos el worker se lo pasa a Nora (Martina ya pasó este horario).
	e.Ahora = e.Ahora.Add(31 * time.Minute)
	if r, err := e.es.Avanzar(ctx, e.Ahora); err != nil || r.(map[string]int)["avanzadas"] != 1 {
		t.Fatalf("worker: %v %v", r, err)
	}
	e.Esperar(200, P{Metodo: "GET", Ruta: "/v1/espera/" + tokLola}, &pub)
	if pub.Estado != "vencida" {
		t.Fatalf("Lola: %+v", pub)
	}
	e.Esperar(409, P{Metodo: "POST", Ruta: "/v1/espera/" + tokLola + "/tomar"}, nil)
	// Una segunda pasada no toca nada: Nora tiene su link vigente.
	if r, _ := e.es.Avanzar(ctx, e.Ahora); r.(map[string]int)["avanzadas"] != 0 {
		t.Fatalf("segunda pasada: %v", r)
	}

	// Nora lo toma: queda su turno (como una reserva de la página) y la campanita lo cuenta.
	tokNora := e.link("nora@ejemplo.com")
	var tn turnos.Turno
	e.Esperar(201, P{Metodo: "POST", Ruta: "/v1/espera/" + tokNora + "/tomar"}, &tn)
	if tn.Cliente.Nombre != "Nora" || tn.Estado != "pendiente" || tn.Origen != "web" || !tn.Inicio.Equal(en("2026-11-02 09:00")) || tn.Profesional != "Ana" {
		t.Fatalf("turno de Nora: %+v", tn)
	}
	e.Esperar(409, P{Metodo: "POST", Ruta: "/v1/espera/" + tokNora + "/tomar"}, nil)
	e.Esperar(200, P{Metodo: "GET", Ruta: "/v1/espera/" + tokNora}, &pub)
	if pub.Estado != "tomada" || pub.Turno == nil || *pub.Turno != tn.Codigo {
		t.Fatalf("link de Nora: %+v", pub)
	}
	e.panel(200, "GET", "/v1/panel/espera/ofertas/"+oferta, nil, &o)
	if o.Estado != "tomada" || o.TurnoID == nil || *o.TurnoID != tn.ID {
		t.Fatalf("oferta tomada: %+v", o)
	}
	if c := e.notificacion("Se llenó el hueco de las 09:00"); c.Tipo != avisos.NotiEspera || !strings.Contains(c.Texto, "Nora") {
		t.Fatalf("campanita: %+v", c)
	}
	// Con sus datos de la lista: le llega la confirmación del turno al mismo email.
	if _, err := e.env.Pasada(ctx, e.Ahora); err != nil {
		t.Fatal(err)
	}
	if m := e.mails.Para("nora@ejemplo.com"); len(m) != 2 || m[1].Asunto != "Turno reservado — El Corte" {
		t.Fatalf("mails de Nora: %+v", m)
	}

	// Si el horario se ocupa por otro lado, la oferta se cae y quien la tenía vuelve a esperar.
	e.anotar("Eva", "", "manana")
	t2 := e.reservar(e.beto, "10:00", "Pedro", "2222")
	e.panel(200, "PATCH", "/v1/panel/turnos/"+t2.ID, map[string]string{"estado": "cancelado"}, nil)
	os = e.ofertas("")
	if len(os) != 1 || os[0].Actual == nil || os[0].Actual.Nombre != "Eva" || os[0].Profesional != "Beto" {
		t.Fatalf("hueco de Beto: %+v", os)
	}
	tokEva := e.link("eva@ejemplo.com")
	e.reservar(e.beto, "10:00", "Quique", "3333")
	e.Esperar(200, P{Metodo: "GET", Ruta: "/v1/espera/" + tokEva}, &pub)
	if pub.Estado != "cerrada" {
		t.Fatalf("Eva: %+v", pub)
	}
	if os := e.ofertas("?fecha=2026-11-02"); len(os) != 2 || os[1].Estado != "caida" {
		t.Fatalf("del día: %+v", os)
	}

	// Lo dispara recepción: con el ajuste apagado, un horario liberado sólo suma a la campanita.
	e.panel(400, "PUT", "/v1/panel/ajustes/espera", map[string]any{"minutos": 1, "automatica": false}, nil)
	var aj espera.AjustesEspera
	e.panel(200, "PUT", "/v1/panel/ajustes/espera", map[string]any{"minutos": 15, "automatica": false}, &aj)
	e.panel(200, "GET", "/v1/panel/ajustes/espera", nil, &aj)
	if aj.Minutos != 15 || aj.Automatica {
		t.Fatalf("ajustes: %+v", aj)
	}
	t3 := e.reservar(e.ana, "10:30", "Rosa", "4444")
	e.Esperar(200, P{Metodo: "POST", Ruta: "/v1/t/" + t3.Codigo + "/cancelar"}, nil)
	if len(e.ofertas("")) != 0 {
		t.Fatal("con el ajuste apagado no se ofrece solo")
	}
	if c := e.notificacion("Se liberó las 10:30 con Ana"); !strings.Contains(c.Texto, "2 personas esperan") {
		t.Fatalf("campanita: %+v", c)
	}
	hueco := map[string]any{"servicioId": e.servicio, "profesionalId": e.ana, "inicio": en("2026-11-02 10:30").Format(time.RFC3339)}
	var cand struct {
		Espera []espera.Anotado `json:"espera"`
	}
	e.panel(200, "GET", "/v1/panel/espera/candidatos?servicio="+e.servicio+"&profesional="+e.ana+"&inicio="+
		strings.ReplaceAll(en("2026-11-02 10:30").Format(time.RFC3339), "+", "%2B"), nil, &cand)
	if len(cand.Espera) != 2 || cand.Espera[0].Nombre != "Martina" || cand.Espera[1].Nombre != "Eva" {
		t.Fatalf("candidatos: %+v", cand.Espera)
	}
	e.panel(201, "POST", "/v1/panel/espera/ofertas", hueco, &o)
	if o.Origen != "panel" || o.Actual == nil || o.Actual.Nombre != "Martina" || !o.Actual.VenceEn.Equal(e.Ahora.Add(15*time.Minute)) {
		t.Fatalf("ofrecer: %+v", o)
	}
	e.panel(409, "POST", "/v1/panel/espera/ofertas", hueco, nil)
	// Frenar: Martina vuelve a esperar.
	e.panel(200, "POST", "/v1/panel/espera/ofertas/"+o.ID+"/frenar", nil, &o)
	if o.Estado != "frenada" || o.Actual != nil || o.Cola[0].Estado != "cancelada" {
		t.Fatalf("frenar: %+v", o)
	}
	e.panel(409, "POST", "/v1/panel/espera/ofertas/"+o.ID+"/frenar", nil, nil)
	// «Cargarlo yo» dándoselo a Eva: turno confirmado del panel.
	e.panel(201, "POST", "/v1/panel/espera/ofertas", hueco, &o)
	var eva string
	for _, c := range o.Cola {
		if c.Nombre == "Eva" {
			eva = *c.EsperaID
		}
	}
	var cargado struct {
		Oferta espera.Oferta `json:"oferta"`
		Turno  *turnos.Turno `json:"turno"`
	}
	e.panel(200, "POST", "/v1/panel/espera/ofertas/"+o.ID+"/cargar", map[string]string{"esperaId": eva}, &cargado)
	if cargado.Oferta.Estado != "cargada" || cargado.Turno == nil || cargado.Turno.Cliente.Nombre != "Eva" || cargado.Turno.Estado != "confirmado" ||
		cargado.Oferta.Cola[0].Estado != "cancelada" || cargado.Oferta.Cola[1].Estado != "tomada" {
		t.Fatalf("cargar: %+v", cargado)
	}
	// Nadie más espera ese horario.
	e.panel(409, "POST", "/v1/panel/espera/ofertas", map[string]any{"servicioId": e.servicio, "profesionalId": e.beto,
		"inicio": en("2026-11-02 09:30").Format(time.RFC3339)}, nil)
}

// Cuando se termina la cola sin que nadie lo tome, la oferta queda agotada y la campanita lo dice.
func TestOfertaAgotada(t *testing.T) {
	e := armar(t)
	t1 := e.reservar(e.ana, "09:00", "Lucía", "1111")
	e.anotar("Martina", e.ana, "")
	e.Esperar(200, P{Metodo: "POST", Ruta: "/v1/t/" + t1.Codigo + "/cancelar"}, nil)
	tok := e.link("martina@ejemplo.com")
	e.Esperar(200, P{Metodo: "POST", Ruta: "/v1/espera/" + tok + "/pasar"}, nil)
	if os := e.ofertas("?fecha=2026-11-02"); len(os) != 1 || os[0].Estado != "agotada" {
		t.Fatalf("ofertas: %+v", os)
	}
	e.notificacion("Nadie de la lista tomó el hueco de las 09:00")
}
