package avisos_test

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
	"net/http"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/avisos"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/catalogo"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/disponibilidad"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/espera"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/invitaciones"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/negocios"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/resenas"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/turnos"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/auditoria"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/correo"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/prueba"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/servidor"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/trabajos"
)

const (
	duenia     = "user_duenia0001"
	mailDuenia = "duenia0001@prueba.tengolibre.com.ar"
	sitio      = "https://tengolibre.com.ar"
)

type P = prueba.Pedido

var z = disponibilidad.ZonaPorDefecto

func en(s string) time.Time {
	t, err := time.ParseInLocation("2006-01-02 15:04", s, z)
	if err != nil {
		panic(err)
	}
	return t
}

// entorno: la API con todos los dominios (turnos con Avisar enganchado y la lista de espera escuchando
// los horarios liberados), el enviador con un correo falso y los trabajos, todos con el reloj del test.
type entorno struct {
	*prueba.API
	negocio, servicio, ana, beto string
	mails                        *correo.Memoria
	env                          *avisos.Enviador
	tj                           *avisos.Trabajos
	rs                           *resenas.Servicio
	marcados                     []string
}

// armar: un negocio con un servicio de 30' y dos profesionales que atienden los lunes de 9 a 11.
// Ahora: domingo 1/11/2026 a las 10.
func armar(t *testing.T) *entorno {
	e := &entorno{mails: &correo.Memoria{}}
	e.API = prueba.Nueva(t, en("2026-11-01 10:00"), func(a *prueba.API) []servidor.Dominio {
		app := a.Bases.App
		av := &avisos.Servicio{Pool: app, Sesion: a.Sesion, Ahora: a.Reloj(), AlLiberarse: espera.Liberado}
		e.rs = &resenas.Servicio{Pool: app, Sesion: a.Sesion, Ahora: a.Reloj()}
		return []servidor.Dominio{
			&negocios.Servicio{Pool: app, Sesion: a.Sesion},
			&catalogo.Servicio{Pool: app, Sesion: a.Sesion},
			&turnos.Servicio{Pool: app, Sesion: a.Sesion, Ahora: a.Reloj(), Avisar: av.Avisar},
			av, e.rs,
			&espera.Servicio{Pool: app, Sesion: a.Sesion, Ahora: a.Reloj()},
			&invitaciones.Servicio{Pool: app, Sesion: a.Sesion, Ahora: a.Reloj()},
			&auditoria.Servicio{Pool: app, Sesion: a.Sesion},
		}
	})
	app := e.Bases.App
	e.env = &avisos.Enviador{Pool: app, Correo: e.mails, Sitio: sitio}
	e.tj = &avisos.Trabajos{Pool: app, DespuesDeMarcar: func(_ context.Context, _, turno, evento string) {
		e.marcados = append(e.marcados, turno+":"+evento)
	}}
	var n negocios.Negocio
	e.Esperar(http.StatusCreated, P{Metodo: "POST", Ruta: "/v1/negocios", Quien: duenia,
		Cuerpo: map[string]string{"slug": "el-corte", "nombre": "El Corte"}}, &n)
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

func (e *entorno) reservar(inicio, telefono, email string) turnos.Turno {
	e.T.Helper()
	var t turnos.Turno
	e.Esperar(201, P{Metodo: "POST", Ruta: "/v1/p/el-corte/turnos", Cuerpo: map[string]any{
		"servicioId": e.servicio, "profesionalId": e.ana, "inicio": en(inicio).Format(time.RFC3339),
		"cliente": map[string]string{"nombre": "Lucía Fernández", "telefono": telefono, "email": email}}}, &t)
	return t
}

// enviar hace una pasada del enviador a la hora del test.
func (e *entorno) enviar() avisos.ResultadoEnvio {
	e.T.Helper()
	r, err := e.env.Pasada(context.Background(), e.Ahora)
	if err != nil {
		e.T.Fatal(err)
	}
	return r
}

func (e *entorno) ultimo(para string) correo.Mensaje {
	e.T.Helper()
	ms := e.mails.Para(para)
	if len(ms) == 0 {
		e.T.Fatalf("no le llegó nada a %s", para)
	}
	return ms[len(ms)-1]
}

func (e *entorno) avisos(q string) []avisos.Registro {
	e.T.Helper()
	var r struct {
		Avisos []avisos.Registro `json:"avisos"`
	}
	e.panel(200, "GET", "/v1/panel/avisos"+q, nil, &r)
	return r.Avisos
}

type campanita struct {
	Notificaciones []avisos.NotificacionPanel `json:"notificaciones"`
	SinLeer        int                        `json:"sinLeer"`
}

// link saca del texto del mail el token de un link (/resena/, /invitacion/, /mi-cuenta/avisos/).
func link(t *testing.T, texto, prefijo string) string {
	t.Helper()
	m := regexp.MustCompile(regexp.QuoteMeta(sitio+prefijo) + `([A-Za-z0-9_-]+)`).FindStringSubmatch(texto)
	if m == nil {
		t.Fatalf("el mail no tiene un link %s: %s", prefijo, texto)
	}
	return m[1]
}

func TestAvisosDeUnTurno(t *testing.T) {
	e := armar(t)
	ctx := context.Background()
	lucia := "lucia@ejemplo.com"

	// Alguien se anota en la lista de espera para el lunes con Ana.
	e.Esperar(201, P{Metodo: "POST", Ruta: "/v1/p/el-corte/espera", Cuerpo: map[string]any{"servicioId": e.servicio,
		"profesionalId": e.ana, "nombre": "Martina", "email": "martina@ejemplo.com", "desde": "2026-11-02"}}, nil)
	e.Esperar(400, P{Metodo: "POST", Ruta: "/v1/p/el-corte/espera", Cuerpo: map[string]any{"servicioId": e.servicio,
		"nombre": "Sin mail", "desde": "2026-11-02"}}, nil)

	// La reserva desde la página encola la confirmación y suma a la campanita.
	t1 := e.reservar("2026-11-02 09:00", "1111", lucia)
	var c campanita
	e.panel(200, "GET", "/v1/panel/campanita", nil, &c)
	if c.SinLeer != 2 || !strings.Contains(c.Notificaciones[0].Titulo, "Nueva reserva de Lucía") {
		t.Fatalf("campanita: %+v", c)
	}
	e.panel(204, "POST", "/v1/panel/campanita/vista", nil, nil)
	e.panel(200, "GET", "/v1/panel/campanita/sin-leer", nil, &c)
	if c.SinLeer != 0 {
		t.Fatalf("después de mirarla quedan %d sin leer", c.SinLeer)
	}

	if r := e.enviar(); r.Enviados != 1 {
		t.Fatalf("pasada: %+v", r)
	}
	m := e.ultimo(lucia)
	if m.Asunto != "Turno reservado — El Corte" || len(m.Adjuntos) != 1 || !strings.Contains(string(m.Adjuntos[0].Datos), "DTSTART:20261102T120000Z") ||
		!strings.Contains(m.Texto, "Fecha: lunes 2 de noviembre") || !strings.Contains(m.Texto, "Hora: 09:00") ||
		!strings.Contains(m.Texto, sitio+"/turno/"+t1.Codigo) || !strings.Contains(m.HTML, "Listo, tu turno quedó reservado") {
		t.Fatalf("confirmación: %+v", m)
	}
	baja := link(t, m.Texto, "/mi-cuenta/avisos/")
	if m.Baja != sitio+"/mi-cuenta/avisos/"+baja {
		t.Fatalf("la baja no va en List-Unsubscribe: %q", m.Baja)
	}
	// Una segunda pasada no manda nada más.
	if r := e.enviar(); r.Enviados != 0 {
		t.Fatalf("segunda pasada: %+v", r)
	}

	// El recordatorio: el lunes a las 9 está a 23 horas. Dos pasadas encolan uno solo.
	for i := 0; i < 2; i++ {
		if _, err := e.tj.Recordatorios(ctx, e.Ahora); err != nil {
			t.Fatal(err)
		}
	}
	e.enviar()
	if m := e.ultimo(lucia); m.Asunto != "Mañana tenés turno — El Corte" || len(e.mails.Para(lucia)) != 2 {
		t.Fatalf("recordatorio: %+v", m)
	}

	// La clienta lo pasa a las 10:30: le llega el cambio, y a la que esperaba ese día se le ofrecen las 9.
	e.Esperar(200, P{Metodo: "POST", Ruta: "/v1/t/" + t1.Codigo + "/reprogramar",
		Cuerpo: map[string]any{"inicio": en("2026-11-02 10:30").Format(time.RFC3339)}}, nil)
	e.enviar()
	// Lo cambió ella: «Cambiaste tu turno», con el código corto, el plazo y la respuesta al negocio.
	if m := e.ultimo(lucia); m.Asunto != "Cambiaste tu turno — El Corte" || !strings.Contains(m.Texto, "Antes era el lunes 2 de noviembre a las 09:00") ||
		!strings.Contains(m.Texto, "Hora: 10:30") || !strings.Contains(m.Texto, "mostralo al llegar") || m.ResponderA != mailDuenia {
		t.Fatalf("reprogramado: %s", m.Texto)
	}
	if m := e.ultimo("martina@ejemplo.com"); m.Asunto != "Se liberó lo que esperabas — El Corte" || !strings.Contains(m.Texto, "Hora: 09:00") || !strings.Contains(m.Texto, sitio+"/espera/") {
		t.Fatalf("lista de espera: %s", m.Texto)
	}
	var lista struct {
		Espera []espera.Anotado `json:"espera"`
	}
	e.panel(200, "GET", "/v1/panel/espera", nil, &lista)
	if len(lista.Espera) != 0 {
		t.Fatal("la avisada ya no está esperando")
	}

	// Cancela: le llega la cancelación y la campanita lo cuenta.
	e.Esperar(200, P{Metodo: "POST", Ruta: "/v1/t/" + t1.Codigo + "/cancelar"}, nil)
	e.enviar()
	if m := e.ultimo(lucia); m.Asunto != "Turno cancelado — El Corte" || !strings.Contains(m.Texto, sitio+"/el-corte") {
		t.Fatalf("cancelado: %+v", m)
	}
	e.panel(200, "GET", "/v1/panel/campanita", nil, &c)
	if c.SinLeer != 2 || c.Notificaciones[0].Tipo != avisos.NotiCancelacion {
		t.Fatalf("campanita: %+v", c)
	}

	// Desde el link de baja apaga todo: la confirmación de un turno nuevo sale igual (es el
	// comprobante), pero los cambios ya no (quedan descartados).
	var pref avisos.PreferenciasPublicas
	e.Esperar(200, P{Metodo: "GET", Ruta: "/v1/preferencias/" + baja}, &pref)
	if pref.Negocio != "El Corte" || pref.Cliente != "Lucía" || pref.Correo != lucia || !pref.Email || !pref.Resena || !pref.ListaEspera || !pref.Novedades {
		t.Fatalf("preferencias: %+v", pref)
	}
	e.Esperar(200, P{Metodo: "PUT", Ruta: "/v1/preferencias/" + baja, Cuerpo: map[string]bool{"email": false, "recordatorios": false,
		"resena": false, "listaEspera": false, "novedades": false}}, &pref)
	if pref.Email || pref.Recordatorios || pref.Resena || pref.ListaEspera || pref.Novedades {
		t.Fatalf("preferencias apagadas: %+v", pref)
	}
	// Una llave sola no toca las demás.
	e.Esperar(200, P{Metodo: "PUT", Ruta: "/v1/preferencias/" + baja, Cuerpo: map[string]bool{"novedades": true}}, &pref)
	if !pref.Novedades || pref.Resena || pref.Email {
		t.Fatalf("una llave: %+v", pref)
	}
	e.Esperar(400, P{Metodo: "PUT", Ruta: "/v1/preferencias/" + baja, Cuerpo: map[string]bool{"otra": true}}, nil)
	e.Esperar(404, P{Metodo: "GET", Ruta: "/v1/preferencias/no-existe-este-token"}, nil)
	antes := e.mails.Cantidad()
	t2 := e.reservar("2026-11-02 09:30", "1111", lucia)
	if r := e.enviar(); r.Enviados != 1 || e.mails.Cantidad() != antes+1 || e.ultimo(lucia).Asunto != "Turno reservado — El Corte" {
		t.Fatalf("la confirmación con los mails apagados: %+v", r)
	}
	e.Esperar(200, P{Metodo: "POST", Ruta: "/v1/t/" + t2.Codigo + "/cancelar"}, nil)
	if r := e.enviar(); r.Descartados != 1 || e.mails.Cantidad() != antes+1 {
		t.Fatalf("con los mails apagados: %+v", r)
	}
	as := e.avisos("?estado=descartado")
	if len(as) != 1 || as[0].UltimoError == nil || *as[0].UltimoError != "el cliente pidió no recibir mails" || *as[0].Cliente != "Lucía Fernández" {
		t.Fatalf("registro: %+v", as)
	}

	// Reenviar uno que ya salió lo vuelve a mandar; uno de otro negocio no se encuentra.
	enviados := e.avisos("?estado=enviado&tipo=reservado")
	if len(enviados) != 2 {
		t.Fatalf("enviados: %+v", enviados)
	}
	e.panel(202, "POST", "/v1/panel/avisos/"+enviados[0].ID+"/reenviar", nil, nil)
	e.panel(409, "POST", "/v1/panel/avisos/"+enviados[0].ID+"/reenviar", nil, nil)
	e.Esperar(201, P{Metodo: "POST", Ruta: "/v1/negocios", Quien: "user_ajeno00001", Cuerpo: map[string]string{"slug": "otro", "nombre": "Otro"}}, nil)
	var yo struct {
		Negocios []struct{ ID string } `json:"negocios"`
	}
	e.Esperar(200, P{Metodo: "GET", Ruta: "/v1/yo", Quien: "user_ajeno00001"}, &yo)
	e.Esperar(404, P{Metodo: "POST", Ruta: "/v1/panel/avisos/" + enviados[0].ID + "/reenviar", Quien: "user_ajeno00001", Negocio: yo.Negocios[0].ID}, nil)
	var ajeno struct {
		Avisos []avisos.Registro `json:"avisos"`
	}
	e.Esperar(200, P{Metodo: "GET", Ruta: "/v1/panel/avisos", Quien: "user_ajeno00001", Negocio: yo.Negocios[0].ID}, &ajeno)
	if len(ajeno.Avisos) != 0 {
		t.Fatal("otro negocio ve avisos ajenos")
	}
	// Turno cancelado: el reenvío de su confirmación se descarta en vez de mandarse.
	if r := e.enviar(); r.Descartados != 1 {
		t.Fatalf("reenvío de un turno cancelado: %+v", r)
	}

	// Lo que hizo el panel queda en la auditoría.
	var aud struct {
		Entradas []auditoria.Entrada `json:"entradas"`
	}
	e.panel(200, "GET", "/v1/panel/auditoria?entidad=aviso", nil, &aud)
	if len(aud.Entradas) != 1 || aud.Entradas[0].Accion != "aviso.reenviado" || aud.Entradas[0].Nombre == "" {
		t.Fatalf("auditoría: %+v", aud.Entradas)
	}
}

func TestReintentosHastaFallar(t *testing.T) {
	e := armar(t)
	e.mails.Fallar = errors.New("el servidor de correo no responde")
	e.reservar("2026-11-02 09:00", "1111", "lucia@ejemplo.com")
	// Cada intento fallido espera el doble; antes de la espera no se reintenta.
	for i := 1; i < avisos.MaxIntentos; i++ {
		if r := e.enviar(); r.Reintentos != 1 {
			t.Fatalf("intento %d: %+v", i, r)
		}
		e.Ahora = e.Ahora.Add(avisos.EsperaReintento(i) - time.Second)
		if r := e.enviar(); r.Reintentos+r.Enviados+r.Fallidos != 0 {
			t.Fatalf("reintentó antes de la espera: %+v", r)
		}
		e.Ahora = e.Ahora.Add(time.Second)
	}
	if r := e.enviar(); r.Fallidos != 1 {
		t.Fatalf("último intento: %+v", r)
	}
	as := e.avisos("?estado=fallido")
	if len(as) != 1 || as[0].Intentos != avisos.MaxIntentos || *as[0].UltimoError != "el servidor de correo no responde" {
		t.Fatalf("fallido: %+v", as)
	}
	var c campanita
	e.panel(200, "GET", "/v1/panel/campanita", nil, &c)
	if c.Notificaciones[0].Tipo != avisos.NotiAvisoFallido || !strings.Contains(c.Notificaciones[0].Titulo, "lucia@ejemplo.com") {
		t.Fatalf("campanita: %+v", c.Notificaciones)
	}
	// Arreglado el correo, el reenvío manual sale.
	e.mails.Fallar = nil
	e.panel(202, "POST", "/v1/panel/avisos/"+as[0].ID+"/reenviar", nil, nil)
	if r := e.enviar(); r.Enviados != 1 {
		t.Fatalf("reenvío: %+v", r)
	}
}

func TestResenas(t *testing.T) {
	e := armar(t)
	ctx := context.Background()
	t1 := e.reservar("2026-11-02 09:00", "1111", "lucia@ejemplo.com")
	e.reservar("2026-11-02 09:30", "2222", "")
	e.enviar()
	e.panel(200, "PATCH", "/v1/panel/turnos/"+t1.ID, map[string]string{"estado": "atendido"}, nil)

	// Terminó a las 9:30; con la demora por defecto (2 h), a las 10 todavía no se pide y a las 11:30 sí.
	e.Ahora = en("2026-11-02 10:00")
	if r, _ := e.rs.Pedir(ctx, e.Ahora); r.(map[string]int)["pedidos"] != 0 {
		t.Fatalf("pidió antes de la demora: %v", r)
	}
	e.Ahora = en("2026-11-02 11:30")
	for i := 0; i < 2; i++ {
		if _, err := e.rs.Pedir(ctx, e.Ahora); err != nil {
			t.Fatal(err)
		}
	}
	e.enviar()
	ms := e.mails.Para("lucia@ejemplo.com")
	m := ms[len(ms)-1]
	if len(ms) != 2 || m.Asunto != "¿Cómo fue tu visita? — El Corte" {
		t.Fatalf("pedido de reseña: %d mails, %q", len(ms), m.Asunto)
	}
	token := link(t, m.Texto, "/resena/")

	var ped resenas.Pedido
	e.Esperar(200, P{Metodo: "GET", Ruta: "/v1/resenas/" + token}, &ped)
	if ped.Negocio != "El Corte" || ped.Servicio != "Corte" || ped.Resena != nil || ped.Vencido {
		t.Fatalf("pedido: %+v", ped)
	}
	e.Esperar(400, P{Metodo: "POST", Ruta: "/v1/resenas/" + token, Cuerpo: map[string]any{"puntaje": 6}}, nil)
	e.Esperar(201, P{Metodo: "POST", Ruta: "/v1/resenas/" + token, Cuerpo: map[string]any{"puntaje": 4, "comentario": "Muy bien"}}, nil)
	e.Esperar(409, P{Metodo: "POST", Ruta: "/v1/resenas/" + token, Cuerpo: map[string]any{"puntaje": 5}}, nil)

	type publicas struct {
		resenas.Resumen
		Resenas []resenas.Publica `json:"resenas"`
	}
	var pub publicas
	e.Esperar(200, P{Metodo: "GET", Ruta: "/v1/p/el-corte/resenas"}, &pub)
	if pub.Promedio != 4 || pub.Cantidad != 1 || pub.Resenas[0].Nombre != "Lucía F." {
		t.Fatalf("públicas: %+v", pub)
	}

	// El panel responde y la oculta: deja de contar en la página.
	var lista struct {
		Resenas      []resenas.Resena `json:"resenas"`
		Distribucion map[string]int   `json:"distribucion"`
	}
	e.panel(200, "GET", "/v1/panel/resenas", nil, &lista)
	// Cuántas hay de cada puntaje (las visibles), siempre los cinco.
	if len(lista.Distribucion) != 5 || lista.Distribucion["4"] != 1 || lista.Distribucion["5"] != 0 {
		t.Fatalf("distribución: %+v", lista.Distribucion)
	}
	var x resenas.Resena
	e.panel(200, "PATCH", "/v1/panel/resenas/"+lista.Resenas[0].ID, map[string]any{"respuesta": "¡Gracias!"}, &x)
	if x.Respuesta != "¡Gracias!" || x.RespondidaEn == nil {
		t.Fatalf("respuesta: %+v", x)
	}
	e.panel(200, "PATCH", "/v1/panel/resenas/"+x.ID, map[string]any{"visible": false}, nil)
	e.Esperar(200, P{Metodo: "GET", Ruta: "/v1/p/el-corte/resenas"}, &pub)
	if pub.Cantidad != 0 || len(pub.Resenas) != 0 {
		t.Fatalf("oculta y se ve: %+v", pub)
	}
	var c campanita
	e.panel(200, "GET", "/v1/panel/campanita", nil, &c)
	if c.Notificaciones[0].Tipo != avisos.NotiResena || !strings.Contains(c.Notificaciones[0].Titulo, "★★★★☆") {
		t.Fatalf("campanita: %+v", c.Notificaciones[0])
	}

	// La auditoría lo registra, y el CSV también.
	r := e.panel(200, "GET", "/v1/panel/auditoria/exportar?entidad=resena", nil, nil)
	csv := string(r.Cuerpo)
	if !strings.Contains(csv, "resena.ocultada") || !strings.Contains(csv, "resena.respondida") || !strings.HasPrefix(csv, "\xef\xbb\xbffecha,") {
		t.Fatalf("csv: %s", csv)
	}
	// Una recepcionista no modera.
	e.panel(201, "POST", "/v1/panel/equipo", map[string]string{"clerkId": "user_recep00001", "rol": "recepcion"}, nil)
	e.Esperar(403, P{Metodo: "PATCH", Ruta: "/v1/panel/resenas/" + x.ID, Quien: "user_recep00001", Negocio: e.negocio,
		Cuerpo: map[string]any{"visible": true}}, nil)
}

func TestTrabajosDelWorker(t *testing.T) {
	e := armar(t)
	ctx := context.Background()
	e.panel(200, "PUT", "/v1/panel/ajustes/avisos", map[string]any{"resenaDemoraHoras": 3, "resumenDiario": true}, nil)
	// Marcar ausentes solos se prende desde el panel de ausencias (dominio clientes, no montado acá).
	if err := e.Bases.App.EnNegocio(ctx, e.negocio, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, "UPDATE ajustes SET marcar_ausentes = true WHERE negocio_id = app.negocio_actual()")
		return err
	}); err != nil {
		t.Fatal(err)
	}
	t1 := e.reservar("2026-11-02 09:00", "1111", "lucia@ejemplo.com")
	t2 := e.reservar("2026-11-02 10:00", "2222", "")
	e.panel(200, "PATCH", "/v1/panel/turnos/"+t2.ID, map[string]string{"estado": "atendido"}, nil)

	// El resumen sale a partir de las 19 (hora del negocio), una sola vez, a la dueña.
	e.Ahora = en("2026-11-01 18:00")
	if r, _ := e.tj.Resumen(ctx, e.Ahora); r.(map[string]int)["encolados"] != 0 {
		t.Fatalf("resumen antes de hora: %v", r)
	}
	e.Ahora = en("2026-11-01 19:30")
	if r, _ := e.tj.Resumen(ctx, e.Ahora); r.(map[string]int)["encolados"] != 1 {
		t.Fatalf("resumen: %v", r)
	}
	if r, _ := e.tj.Resumen(ctx, e.Ahora.Add(time.Hour)); r.(map[string]int)["encolados"] != 0 {
		t.Fatalf("resumen repetido: %v", r)
	}
	e.enviar()
	m := e.ultimo(mailDuenia)
	if m.Asunto != "Mañana tenés 1 turno — El Corte" || !strings.Contains(m.Texto, "09:00: Lucía Fernández · Corte con Ana") {
		t.Fatalf("resumen: %s\n%s", m.Asunto, m.Texto)
	}

	// Ausentes: el de las 9 (pendiente) terminó 9:30; con dos horas de gracia, a las 11 no y a las 12 sí.
	// El atendido no se toca.
	if r, _ := e.tj.Ausentes(ctx, en("2026-11-02 11:00")); r.(map[string]int)["marcados"] != 0 {
		t.Fatalf("ausente antes de la gracia: %v", r)
	}
	if r, _ := e.tj.Ausentes(ctx, en("2026-11-02 12:00")); r.(map[string]int)["marcados"] != 1 {
		t.Fatalf("ausentes: %v", r)
	}
	var x turnos.Turno
	e.panel(200, "GET", "/v1/panel/turnos/"+t1.ID, nil, &x)
	if x.Estado != "ausente" || len(e.marcados) != 1 || e.marcados[0] != t1.ID+":ausente" {
		t.Fatalf("ausente: %s, ganchos %v", x.Estado, e.marcados)
	}

	// El planificador: corre lo que toca, lo registra, y en la misma hora no repite.
	p := &trabajos.Planificador{Pool: e.Bases.App, Ahora: e.Reloj(), Trabajos: append(e.tj.Lista(e.env), e.rs.Trabajo(),
		trabajos.Trabajo{Nombre: "roto", Cada: time.Hour, Correr: func(context.Context, time.Time) (any, error) { panic("se rompió") }})}
	e.Ahora = en("2026-11-02 13:00")
	corridos, err := p.Pasada(ctx)
	if err != nil || len(corridos) != 7 {
		t.Fatalf("primera pasada: %v %v", corridos, err)
	}
	if corridos, _ := p.Pasada(ctx); len(corridos) != 0 {
		t.Fatalf("repitió: %v", corridos)
	}
	e.Ahora = e.Ahora.Add(10 * time.Second)
	if corridos, _ := p.Pasada(ctx); len(corridos) != 1 || corridos[0] != "envios" {
		t.Fatalf("a los 10 segundos: %v", corridos)
	}
	c, err := trabajos.LeerCorrida(ctx, e.Bases.App, "roto")
	if err != nil || c.UltimoError == nil || !strings.Contains(*c.UltimoError, "se rompió") {
		t.Fatalf("corrida rota: %+v %v", c, err)
	}
	c, _ = trabajos.LeerCorrida(ctx, e.Bases.App, "envios")
	if c.Corridas != 2 || c.UltimoError != nil {
		t.Fatalf("corrida de envíos: %+v", c)
	}

	// La purga: a los 200 días, los avisos ya resueltos y la auditoría vieja se van.
	r, err := e.tj.Purgar(ctx, time.Now().Add(200*24*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	borradas := r.(map[string]int64)
	if borradas["avisos"] == 0 || borradas["auditoria"] == 0 || borradas["notificaciones"] == 0 {
		t.Fatalf("purga: %v", borradas)
	}
	if len(e.avisos("")) != 0 {
		t.Fatal("quedaron avisos")
	}
}

func TestInvitaciones(t *testing.T) {
	e := armar(t)
	nueva := "user_nueva00001"
	mailNueva := "nueva00001@prueba.tengolibre.com.ar"
	e.panel(400, "POST", "/v1/panel/invitaciones", map[string]string{"email": mailNueva, "rol": "duenio"}, nil)
	var inv invitaciones.Invitacion
	e.panel(201, "POST", "/v1/panel/invitaciones", map[string]string{"email": "Nueva00001@prueba.tengolibre.com.ar", "rol": "recepcion"}, &inv)
	// Invitar de nuevo reemplaza la anterior (y el link viejo deja de servir).
	e.enviar()
	viejo := link(t, e.ultimo(mailNueva).Texto, "/invitacion/")
	e.panel(201, "POST", "/v1/panel/invitaciones", map[string]string{"email": mailNueva, "rol": "recepcion"}, &inv)
	e.enviar()
	m := e.ultimo(mailNueva)
	if m.Asunto != "Te invitaron a El Corte — Tengo Libre" || !strings.Contains(m.Texto, "con el rol de recepción") {
		t.Fatalf("invitación: %s\n%s", m.Asunto, m.Texto)
	}
	token := link(t, m.Texto, "/invitacion/")
	e.Esperar(404, P{Metodo: "GET", Ruta: "/v1/invitaciones/" + viejo}, nil)

	var pub invitaciones.Publica
	e.Esperar(200, P{Metodo: "GET", Ruta: "/v1/invitaciones/" + token}, &pub)
	if pub.Negocio != "El Corte" || pub.Rol != "recepcion" || pub.Email != mailNueva || pub.Aceptada ||
		pub.InvitadoPor == "" || !strings.Contains(m.Texto, pub.InvitadoPor+" te invitó") || !pub.VenceEn.Equal(inv.VenceEn) {
		t.Fatalf("pública: %+v", pub)
	}
	// Sin sesión no se acepta; con otra cuenta tampoco.
	e.Esperar(401, P{Metodo: "POST", Ruta: "/v1/invitaciones/" + token + "/aceptar"}, nil)
	e.Esperar(403, P{Metodo: "POST", Ruta: "/v1/invitaciones/" + token + "/aceptar", Quien: "user_otra000001"}, nil)
	e.Esperar(200, P{Metodo: "POST", Ruta: "/v1/invitaciones/" + token + "/aceptar", Quien: nueva}, nil)
	e.Esperar(409, P{Metodo: "POST", Ruta: "/v1/invitaciones/" + token + "/aceptar", Quien: nueva}, nil)

	var yo struct {
		Negocios []map[string]string `json:"negocios"`
	}
	e.Esperar(200, P{Metodo: "GET", Ruta: "/v1/yo", Quien: nueva}, &yo)
	if len(yo.Negocios) != 1 || yo.Negocios[0]["rol"] != "recepcion" {
		t.Fatalf("yo: %+v", yo)
	}
	// Ya es del equipo: no se la vuelve a invitar, y la recepción no invita.
	e.panel(409, "POST", "/v1/panel/invitaciones", map[string]string{"email": mailNueva, "rol": "encargado"}, nil)
	e.Esperar(403, P{Metodo: "POST", Ruta: "/v1/panel/invitaciones", Quien: nueva, Negocio: e.negocio,
		Cuerpo: map[string]string{"email": "x@y.com", "rol": "recepcion"}}, nil)
	// Una invitación vencida no se acepta.
	e.panel(201, "POST", "/v1/panel/invitaciones", map[string]string{"email": "tarde0001@prueba.tengolibre.com.ar", "rol": "profesional"}, nil)
	e.enviar()
	tarde := link(t, e.ultimo("tarde0001@prueba.tengolibre.com.ar").Texto, "/invitacion/")
	e.Ahora = e.Ahora.Add(8 * 24 * time.Hour)
	e.Esperar(409, P{Metodo: "POST", Ruta: "/v1/invitaciones/" + tarde + "/aceptar", Quien: "user_tarde0001"}, nil)

	var c campanita
	e.panel(200, "GET", "/v1/panel/campanita", nil, &c)
	if c.Notificaciones[0].Tipo != avisos.NotiEquipo {
		t.Fatalf("campanita: %+v", c.Notificaciones)
	}
}

// Quien reserva con su cuenta de Tengo Libre abierta (la página reenvía la sesión) también le avisa al negocio:
// antes sólo avisaban las reservas sin sesión y al socio no le llegaba nada.
func TestReservaConSesionAvisaAlNegocio(t *testing.T) {
	e := armar(t)
	var c campanita
	e.Esperar(201, P{Metodo: "POST", Ruta: "/v1/p/el-corte/turnos", Quien: "user_clienta00001", Cuerpo: map[string]any{
		"servicioId": e.servicio, "profesionalId": e.ana, "inicio": en("2026-11-02 10:00").Format(time.RFC3339),
		"cliente": map[string]string{"nombre": "Rocío Paz", "telefono": "3333", "email": "rocio@ejemplo.com"}}}, nil)
	e.panel(200, "GET", "/v1/panel/campanita", nil, &c)
	if c.SinLeer != 1 || !strings.Contains(c.Notificaciones[0].Titulo, "Nueva reserva de Rocío") {
		t.Fatalf("campanita: %+v", c)
	}
	// Lo que carga la dueña desde el panel no se le avisa a ella misma.
	e.panel(201, "POST", "/v1/panel/turnos", map[string]any{"servicioId": e.servicio, "profesionalId": e.ana,
		"inicio": en("2026-11-02 09:00").Format(time.RFC3339), "cliente": map[string]string{"nombre": "Juana", "telefono": "4444"}}, nil)
	e.panel(200, "GET", "/v1/panel/campanita", nil, &c)
	if c.SinLeer != 1 {
		t.Fatalf("el turno del panel sumó a la campanita: %+v", c)
	}
}

// Quien reservó con su mail ve el turno en «Mi cuenta», en cualquier negocio; los cancelados no.
func TestMisTurnos(t *testing.T) {
	e := armar(t)
	quien := "user_rociopaz001"
	mail := "rociopaz001@prueba.tengolibre.com.ar"
	e.Esperar(201, P{Metodo: "POST", Ruta: "/v1/p/el-corte/turnos", Cuerpo: map[string]any{
		"servicioId": e.servicio, "profesionalId": e.ana, "inicio": en("2026-11-30 10:00").Format(time.RFC3339),
		"cliente": map[string]string{"nombre": "Rocío Paz", "telefono": "3333", "email": mail}}}, nil)
	var r struct {
		Turnos []struct{ Negocio, Slug, Servicio, Codigo string } `json:"turnos"`
	}
	e.Esperar(200, P{Metodo: "GET", Ruta: "/v1/yo/turnos", Quien: quien}, &r)
	if len(r.Turnos) != 1 || r.Turnos[0].Slug != "el-corte" || r.Turnos[0].Codigo == "" {
		t.Fatalf("mis turnos: %+v", r)
	}
	e.Esperar(200, P{Metodo: "GET", Ruta: "/v1/yo/turnos", Quien: "user_otrapersona01"}, &r)
	if len(r.Turnos) != 0 {
		t.Fatalf("otra persona ve turnos ajenos: %+v", r)
	}
}

// «Tus turnos» no muestra los turnos de otra persona: con el teléfono de una clienta que el negocio
// cargó sin mail y un mail propio, la reserva web no le pega ese mail a la ficha, y quien reservó ve
// sólo el turno que sacó él. Lo que el negocio cargó con un mail (de confianza) sí lo ve su dueña.
func TestMisTurnosNoMuestraTurnosAjenos(t *testing.T) {
	e := armar(t)
	atacante := "user_atacante0001"
	mail := "atacante0001@prueba.tengolibre.com.ar"
	// La clienta, cargada por el panel sólo con teléfono, con un turno.
	e.panel(201, "POST", "/v1/panel/turnos", map[string]any{"servicioId": e.servicio, "profesionalId": e.ana,
		"inicio": en("2026-11-23 09:00").Format(time.RFC3339), "cliente": map[string]string{"nombre": "Marta Gil", "telefono": "5555"}}, nil)
	// Alguien reserva en la página con ese teléfono y su mail.
	e.Esperar(201, P{Metodo: "POST", Ruta: "/v1/p/el-corte/turnos", Cuerpo: map[string]any{
		"servicioId": e.servicio, "profesionalId": e.ana, "inicio": en("2026-11-23 10:00").Format(time.RFC3339),
		"cliente": map[string]string{"nombre": "Marta Gil", "telefono": "5555", "email": mail}}}, nil)
	var r struct {
		Turnos []struct{ Inicio time.Time } `json:"turnos"`
	}
	e.Esperar(200, P{Metodo: "GET", Ruta: "/v1/yo/turnos", Quien: atacante}, &r)
	if len(r.Turnos) != 1 || !r.Turnos[0].Inicio.Equal(en("2026-11-23 10:00")) {
		t.Fatalf("con el teléfono de otra se ven sus turnos: %+v", r)
	}
	// Otro turno de la clienta cargado después por el panel tampoco aparece.
	e.panel(201, "POST", "/v1/panel/turnos", map[string]any{"servicioId": e.servicio, "profesionalId": e.ana,
		"inicio": en("2026-11-30 09:00").Format(time.RFC3339), "cliente": map[string]string{"nombre": "Marta Gil", "telefono": "5555"}}, nil)
	e.Esperar(200, P{Metodo: "GET", Ruta: "/v1/yo/turnos", Quien: atacante}, &r)
	if len(r.Turnos) != 1 {
		t.Fatalf("aparece un turno del panel: %+v", r)
	}
	// El mail de la ficha sigue vacío (no le llegan los recordatorios de la clienta).
	var email string
	if err := e.Bases.App.EnNegocio(context.Background(), e.negocio, func(tx pgx.Tx) error {
		return tx.QueryRow(context.Background(), "SELECT email FROM clientes WHERE telefono = '5555'").Scan(&email)
	}); err != nil {
		t.Fatal(err)
	}
	if email != "" {
		t.Fatalf("la reserva web completó el mail de la ficha: %q", email)
	}
	// Un turno que el negocio cargó con el mail de la clienta (de confianza) sí lo ve ella.
	duena := "user_lauraok00001"
	e.panel(201, "POST", "/v1/panel/turnos", map[string]any{"servicioId": e.servicio, "profesionalId": e.ana,
		"inicio": en("2026-11-30 10:00").Format(time.RFC3339), "cliente": map[string]string{"nombre": "Laura",
			"telefono": "6666", "email": "lauraok00001@prueba.tengolibre.com.ar"}}, nil)
	e.Esperar(200, P{Metodo: "GET", Ruta: "/v1/yo/turnos", Quien: duena}, &r)
	if len(r.Turnos) != 1 {
		t.Fatalf("el turno con mail de confianza no aparece: %+v", r)
	}
}

// La seña pagada tarde (el horario ya era de otra) y la de un turno ya cancelado: a la clienta le llega
// un mail que habla de su plata (no el «Tu turno fue cancelado» genérico) y al negocio, la campanita.
func TestPagoTardeYPagoDeCancelado(t *testing.T) {
	e := armar(t)
	av := &avisos.Servicio{Pool: e.Bases.App, Ahora: e.Reloj()}
	mail := "rosa@ejemplo.com"
	var t1, t2 turnos.Turno
	e.panel(201, "POST", "/v1/panel/turnos", map[string]any{"servicioId": e.servicio, "profesionalId": e.ana,
		"inicio": en("2026-11-02 09:00").Format(time.RFC3339), "cliente": map[string]string{"nombre": "Rosa Díaz", "telefono": "7777", "email": mail}}, &t1)
	e.panel(201, "POST", "/v1/panel/turnos", map[string]any{"servicioId": e.servicio, "profesionalId": e.ana,
		"inicio": en("2026-11-02 10:00").Format(time.RFC3339), "cliente": map[string]string{"nombre": "Rosa Díaz", "telefono": "7777", "email": mail}}, &t2)
	e.enviar()
	av.Avisar(context.Background(), e.negocio, t1.ID, "pago_sin_lugar")
	e.enviar()
	if m := e.ultimo(mail); m.Asunto != "Tu seña llegó tarde — El Corte" || !strings.Contains(m.Texto, "te va a devolver la seña") {
		t.Fatalf("pago tarde: %s %s", m.Asunto, m.Texto)
	}
	av.Avisar(context.Background(), e.negocio, t2.ID, "pago_de_cancelado")
	e.enviar()
	if m := e.ultimo(mail); !strings.Contains(m.Asunto, "turno cancelado") || !strings.Contains(m.Texto, "ya estaba cancelado") {
		t.Fatalf("pago de cancelado: %s %s", m.Asunto, m.Texto)
	}
	var c campanita
	e.panel(200, "GET", "/v1/panel/campanita", nil, &c)
	titulos := ""
	for _, n := range c.Notificaciones {
		titulos += n.Titulo + "|"
	}
	if !strings.Contains(titulos, "pagó la seña tarde") || !strings.Contains(titulos, "pagó la seña de un turno cancelado") {
		t.Fatalf("campanita: %s", titulos)
	}
}

// Sin email no se encola ningún mail del turno (antes quedaba «En cola» y después «Descartado»).
func TestSinEmailNoSeEncola(t *testing.T) {
	e := armar(t)
	var t1 turnos.Turno
	e.panel(201, "POST", "/v1/panel/turnos", map[string]any{"servicioId": e.servicio, "profesionalId": e.ana,
		"inicio": en("2026-11-02 09:00").Format(time.RFC3339), "cliente": map[string]string{"nombre": "Sin Mail", "telefono": "8888"}}, &t1)
	var n int
	e.Fila("SELECT count(*) FROM avisos WHERE turno_id = $1", []any{t1.ID}, &n)
	if n != 0 {
		t.Fatalf("se encolaron %d avisos para una clienta sin email", n)
	}
}

// El panel anota en la lista de espera a quien lo pidió por teléfono (sin campanita: lo cargó el equipo).
func TestAnotarEnEsperaDesdeElPanel(t *testing.T) {
	e := armar(t)
	var a espera.Anotado
	e.panel(201, "POST", "/v1/panel/espera", map[string]any{"servicioId": e.servicio, "nombre": "Carla Ruiz",
		"email": "carla@ejemplo.com", "telefono": "1155550099", "desde": "2026-11-02"}, &a)
	if a.Nombre != "Carla Ruiz" || a.Desde != "2026-11-02" || a.Hasta != "2026-11-02" {
		t.Fatalf("anotada: %+v", a)
	}
	e.panel(400, "POST", "/v1/panel/espera", map[string]any{"servicioId": e.servicio, "nombre": "Sin Mail", "desde": "2026-11-02"}, nil)
	var c campanita
	e.panel(200, "GET", "/v1/panel/campanita", nil, &c)
	if c.SinLeer != 0 {
		t.Fatalf("la carga del panel sumó a la campanita: %+v", c)
	}
}
