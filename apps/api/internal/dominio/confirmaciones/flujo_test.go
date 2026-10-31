package confirmaciones_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/avisos"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/catalogo"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/confirmaciones"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/disponibilidad"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/negocios"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/turnos"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/prueba"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/servidor"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/whatsapp"
)

const (
	duenia  = "user_duenia0001"
	sitio   = "https://tengolibre.com.ar"
	secreto = "secreto-de-la-app"
	pnid    = "PNID"
	// Este número «no tiene WhatsApp» para el Meta falso.
	sinWhatsApp = "5491199990000"
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

// meta hace de la Graph API: guarda lo que le mandan y responde con un wamid nuevo (o con el error
// de un número sin WhatsApp).
type meta struct {
	mu       sync.Mutex
	mensajes []map[string]any
}

func (m *meta) servidor(t *testing.T) *httptest.Server {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer token-meta" || r.URL.Path != "/"+whatsapp.VersionDefecto+"/"+pnid+"/messages" {
			http.Error(w, `{"error":{"message":"bad auth","code":190}}`, 401)
			return
		}
		var x map[string]any
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &x)
		m.mu.Lock()
		m.mensajes = append(m.mensajes, x)
		n := len(m.mensajes)
		m.mu.Unlock()
		if x["to"] == sinWhatsApp {
			w.WriteHeader(400)
			_, _ = io.WriteString(w, `{"error":{"message":"Message undeliverable","code":131026}}`)
			return
		}
		_, _ = fmt.Fprintf(w, `{"messaging_product":"whatsapp","messages":[{"id":"wamid.%d"}]}`, n)
	}))
	t.Cleanup(srv.Close)
	return srv
}

// para devuelve los mensajes mandados a ese número (sin +), en orden.
func (m *meta) para(numero string) []map[string]any {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []map[string]any
	for _, x := range m.mensajes {
		if x["to"] == numero {
			out = append(out, x)
		}
	}
	return out
}

func (m *meta) cuantos() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.mensajes)
}

type entorno struct {
	*prueba.API
	negocio, servicio, ana string
	meta                   *meta
	tj                     *confirmaciones.Trabajos
}

func armar(t *testing.T) *entorno {
	e := &entorno{meta: &meta{}}
	e.API = prueba.Nueva(t, en("2026-11-01 08:00"), func(a *prueba.API) []servidor.Dominio {
		app := a.Bases.App
		av := &avisos.Servicio{Pool: app, Sesion: a.Sesion, Ahora: a.Reloj()}
		return []servidor.Dominio{
			&negocios.Servicio{Pool: app, Sesion: a.Sesion},
			&catalogo.Servicio{Pool: app, Sesion: a.Sesion},
			&turnos.Servicio{Pool: app, Sesion: a.Sesion, Ahora: a.Reloj(), Avisar: av.Avisar},
			av,
			&confirmaciones.Servicio{Pool: app, Sesion: a.Sesion, Ahora: a.Reloj(), Sitio: sitio, Disponible: true,
				TelefonoID: pnid, SecretoApp: secreto, Verificacion: "verificame"},
		}
	})
	srv := e.meta.servidor(t)
	cli := &whatsapp.Cliente{Token: "token-meta", TelefonoID: pnid, Base: srv.URL, HTTP: srv.Client()}
	e.tj = confirmaciones.Nuevos(e.Bases.App, cli, sitio, nil)
	var n negocios.Negocio
	e.Esperar(http.StatusCreated, P{Metodo: "POST", Ruta: "/v1/negocios", Quien: duenia,
		Cuerpo: map[string]string{"slug": "el-corte", "nombre": "El Corte"}}, &n)
	e.negocio = n.ID
	var s catalogo.Prestacion
	e.panel(201, "POST", "/v1/panel/servicios", map[string]any{"nombre": "Corte", "duracion": 30, "precio": 8000}, &s)
	e.servicio = s.ID
	var pa catalogo.Profesional
	e.panel(201, "POST", "/v1/panel/profesionales", map[string]any{"nombre": "Ana", "servicios": []string{s.ID},
		"horario": []disponibilidad.Franja{{Dia: 1, Desde: "09:00", Hasta: "12:00"}}}, &pa)
	e.ana = pa.ID
	return e
}

func (e *entorno) panel(estado int, metodo, ruta string, cuerpo, v any) prueba.Respuesta {
	e.T.Helper()
	return e.Esperar(estado, P{Metodo: metodo, Ruta: ruta, Quien: duenia, Negocio: e.negocio, Cuerpo: cuerpo}, v)
}

func (e *entorno) reservar(inicio, nombre, telefono string) turnos.Turno {
	e.T.Helper()
	var t turnos.Turno
	e.Esperar(201, P{Metodo: "POST", Ruta: "/v1/p/el-corte/turnos", Cuerpo: map[string]any{
		"servicioId": e.servicio, "profesionalId": e.ana, "inicio": en(inicio).Format(time.RFC3339),
		"cliente": map[string]string{"nombre": nombre, "telefono": telefono}}}, &t)
	return t
}

func (e *entorno) turno(id string) turnos.Turno {
	e.T.Helper()
	var t turnos.Turno
	e.panel(200, "GET", "/v1/panel/turnos/"+id, nil, &t)
	return t
}

func (e *entorno) correr(f func(context.Context, time.Time) (any, error)) map[string]int {
	e.T.Helper()
	r, err := f(context.Background(), e.Ahora)
	if err != nil {
		e.T.Fatal(err)
	}
	b, _ := json.Marshal(r)
	var out map[string]int
	_ = json.Unmarshal(b, &out)
	return out
}

func (e *entorno) enviar() confirmaciones.ResultadoEnvio {
	e.T.Helper()
	r, err := e.tj.Enviar(context.Background(), e.Ahora)
	if err != nil {
		e.T.Fatal(err)
	}
	return r
}

// webhook manda el cuerpo firmado (o con la firma dada) y devuelve el estado.
func (e *entorno) webhook(cuerpo string, firma ...string) int {
	e.T.Helper()
	req := httptest.NewRequest("POST", "/v1/webhooks/whatsapp", strings.NewReader(cuerpo))
	f := whatsapp.Firmar(secreto, []byte(cuerpo))
	if len(firma) > 0 {
		f = firma[0]
	}
	req.Header.Set("X-Hub-Signature-256", f)
	rec := httptest.NewRecorder()
	e.Handler.ServeHTTP(rec, req)
	return rec.Code
}

func boton(id, contexto, payload, texto string) string {
	return fmt.Sprintf(`{"object":"whatsapp_business_account","entry":[{"id":"WABA","changes":[{"field":"messages","value":{
		"messaging_product":"whatsapp","metadata":{"phone_number_id":%q},
		"messages":[{"from":"5491123456789","id":%q,"timestamp":"1793530920","type":"button","context":{"id":%q},
		"button":{"payload":%q,"text":%q}}]}}]}]}`, pnid, id, contexto, payload, texto)
}

func estado(wamid, st string) string {
	return fmt.Sprintf(`{"object":"whatsapp_business_account","entry":[{"id":"WABA","changes":[{"field":"messages","value":{
		"messaging_product":"whatsapp","metadata":{"phone_number_id":%q},
		"statuses":[{"id":%q,"status":%q,"timestamp":"1793530930","recipient_id":"5491123456789"}]}}]}]}`, pnid, wamid, st)
}

// payload saca lo que se mandó en el botón i de la plantilla.
func payload(t *testing.T, m map[string]any, i int) string {
	t.Helper()
	for _, c := range m["template"].(map[string]any)["components"].([]any) {
		c := c.(map[string]any)
		if c["type"] == "button" && c["index"] == fmt.Sprint(i) {
			return c["parameters"].([]any)[0].(map[string]any)["payload"].(string)
		}
	}
	t.Fatalf("sin botón %d: %v", i, m)
	return ""
}

func cuerpo(m map[string]any) []string {
	var out []string
	for _, c := range m["template"].(map[string]any)["components"].([]any) {
		c := c.(map[string]any)
		if c["type"] == "body" {
			for _, p := range c["parameters"].([]any) {
				out = append(out, p.(map[string]any)["text"].(string))
			}
		}
	}
	return out
}

func (e *entorno) campanita(tipo string) []avisos.NotificacionPanel {
	e.T.Helper()
	var r struct {
		Notificaciones []avisos.NotificacionPanel `json:"notificaciones"`
	}
	e.panel(200, "GET", "/v1/panel/campanita", nil, &r)
	var out []avisos.NotificacionPanel
	for _, n := range r.Notificaciones {
		if n.Tipo == tipo {
			out = append(out, n)
		}
	}
	return out
}

func (e *entorno) paraLlamar() map[string]string {
	e.T.Helper()
	var r struct {
		Turnos []confirmaciones.ParaLlamar `json:"turnos"`
	}
	e.panel(200, "GET", "/v1/panel/para-llamar", nil, &r)
	out := map[string]string{}
	for _, t := range r.Turnos {
		out[t.TurnoID] = t.Motivo
	}
	return out
}

func TestConfirmacionPorWhatsApp(t *testing.T) {
	e := armar(t)

	// Ajustes: apagado por defecto, a las 10 y 4 horas; sólo la gestión los cambia.
	var a confirmaciones.Ajustes
	e.panel(200, "GET", "/v1/panel/ajustes/whatsapp", nil, &a)
	if !a.Disponible || a.Confirmacion || a.Hora != "10:00" || a.SinRespuestaHoras != 4 || a.Recordatorio {
		t.Fatalf("ajustes por defecto: %+v", a)
	}
	e.panel(400, "PUT", "/v1/panel/ajustes/whatsapp", map[string]any{"confirmacion": true, "hora": "22:00", "sinRespuestaHoras": 4}, nil)
	e.panel(400, "PUT", "/v1/panel/ajustes/whatsapp", map[string]any{"confirmacion": true, "hora": "10:00", "sinRespuestaHoras": 0}, nil)
	e.panel(200, "PUT", "/v1/panel/ajustes/whatsapp", map[string]any{"confirmacion": true, "hora": "10:00", "sinRespuestaHoras": 4}, &a)
	if !a.Confirmacion {
		t.Fatalf("ajustes guardados: %+v", a)
	}

	// Mañana (lunes 2) hay cinco turnos: Sofía, Nadia, Lola (no contesta), Tomás (sin WhatsApp) y
	// Pía (con un teléfono que no es un celular).
	sofia := e.reservar("2026-11-02 10:30", "Sofía Méndez", "11 15-2345-6789")
	nadia := e.reservar("2026-11-02 09:00", "Nadia Paz", "0351 155 123456")
	lola := e.reservar("2026-11-02 09:30", "Lola Fernández", "+54 9 11 3456-7890")
	tomas := e.reservar("2026-11-02 10:00", "Tomás Aguirre", "11 9999-0000")
	e.reservar("2026-11-02 11:00", "Pía Ruiz", "1234")
	if sofia.ConfirmadoPor != nil || sofia.Estado != "pendiente" {
		t.Fatalf("una reserva web no está confirmada: %+v", sofia)
	}

	// Antes de las 10 no sale nada; a las 10, las cinco (Pía, descartada).
	if r := e.correr(e.tj.Confirmaciones); r["encolados"] != 0 {
		t.Fatalf("antes de hora: %v", r)
	}
	e.Ahora = en("2026-11-01 10:00")
	if r := e.correr(e.tj.Confirmaciones); r["encolados"] != 4 {
		t.Fatalf("a las 10: %v", r)
	}
	if r := e.correr(e.tj.Confirmaciones); r["encolados"] != 0 {
		t.Fatalf("una segunda pasada no encola de nuevo: %v", r)
	}
	if r := e.enviar(); r.Enviados != 3 || r.Fallidos != 1 {
		t.Fatalf("envío: %+v", r)
	}
	ms := e.meta.para("5491123456789")
	if len(ms) != 1 {
		t.Fatalf("a Sofía: %v", ms)
	}
	tpl := ms[0]["template"].(map[string]any)
	if tpl["name"] != confirmaciones.PlantillaConfirmacion || tpl["language"].(map[string]any)["code"] != "es_AR" ||
		strings.Join(cuerpo(ms[0]), "|") != "Sofía|lunes 2|10:30|El Corte|corte|Ana" {
		t.Fatalf("plantilla: %v", ms[0])
	}
	confirmo, cambiar := payload(t, ms[0], 0), payload(t, ms[0], 1)
	if !strings.HasPrefix(confirmo, "confirmo:") || !strings.HasPrefix(cambiar, "cambiar:") {
		t.Fatalf("botones: %s %s", confirmo, cambiar)
	}
	// Tomás no tiene WhatsApp: queda para llamar y la campanita lo dice.
	if e.paraLlamar()[tomas.ID] != "no_llego" || len(e.campanita("aviso_fallido")) != 1 {
		t.Fatalf("sin WhatsApp: %v", e.paraLlamar())
	}
	var lista struct {
		Mensajes []confirmaciones.Mensaje `json:"mensajes"`
	}
	e.panel(200, "GET", "/v1/panel/whatsapp/mensajes?tipo=confirmacion", nil, &lista)
	estados := map[string]int{}
	for _, m := range lista.Mensajes {
		estados[m.Estado]++
	}
	if estados["enviado"] != 3 || estados["fallido"] != 1 || estados["descartado"] != 1 {
		t.Fatalf("registro: %v", estados)
	}

	// El webhook: la verificación de Meta, y una firma que no es.
	if r := e.Hacer(P{Metodo: "GET", Ruta: "/v1/webhooks/whatsapp?hub.mode=subscribe&hub.verify_token=verificame&hub.challenge=123"}); r.Estado != 200 || string(r.Cuerpo) != "123" {
		t.Fatalf("verificación: %d %s", r.Estado, r.Cuerpo)
	}
	e.Esperar(403, P{Metodo: "GET", Ruta: "/v1/webhooks/whatsapp?hub.mode=subscribe&hub.verify_token=otro&hub.challenge=123"}, nil)
	if c := e.webhook(boton("wamid.R1", "wamid.1", confirmo, "Confirmo"), "sha256=00"); c != 401 {
		t.Fatalf("firma mala: %d", c)
	}

	// Sofía toca «Confirmo»: el turno se confirma por WhatsApp, avisa a la campanita y le contestamos
	// con el link. Meta repite el webhook: no pasa dos veces.
	e.Ahora = en("2026-11-01 10:02")
	for range 2 {
		if c := e.webhook(boton("wamid.R1", "wamid.1", confirmo, "Confirmo")); c != 200 {
			t.Fatalf("webhook: %d", c)
		}
	}
	ts := e.turno(sofia.ID)
	if ts.Estado != "confirmado" || ts.ConfirmadoPor == nil || *ts.ConfirmadoPor != "whatsapp" || !ts.ConfirmadoEn.Equal(e.Ahora) {
		t.Fatalf("Sofía: %+v", ts)
	}
	if ns := e.campanita("confirmacion"); len(ns) != 1 || ns[0].Titulo != "Sofía Méndez confirmó por WhatsApp" || *ns[0].TurnoID != sofia.ID {
		t.Fatalf("campanita: %+v", ns)
	}
	if r := e.enviar(); r.Enviados != 1 {
		t.Fatalf("respuesta: %+v", r)
	}
	ms = e.meta.para("5491123456789")
	if len(ms) != 2 || ms[1]["type"] != "text" ||
		!strings.Contains(ms[1]["text"].(map[string]any)["body"].(string), sitio+"/turno/"+sofia.Codigo) {
		t.Fatalf("respuesta a Sofía: %v", ms)
	}
	// Desde el link del cliente se ve la confirmación, pero no lo de llamar.
	var tc map[string]any
	e.Esperar(200, P{Metodo: "GET", Ruta: "/v1/t/" + sofia.Codigo}, &tc)
	if tc["confirmadoPor"] != "whatsapp" || tc["llamarDesde"] != nil {
		t.Fatalf("link del cliente: %v", tc)
	}

	// Nadia pide cambiarlo: queda para llamar, con su campanita y la respuesta con el link.
	mn := e.meta.para("5493515123456")
	if c := e.webhook(boton("wamid.R2", "", payload(t, mn[0], 1), "Necesito cambiarlo")); c != 200 {
		t.Fatalf("webhook Nadia: %d", c)
	}
	tn := e.turno(nadia.ID)
	if tn.Estado != "pendiente" || tn.CambioPedidoEn == nil || tn.LlamarMotivo == nil || *tn.LlamarMotivo != "cambio" {
		t.Fatalf("Nadia: %+v", tn)
	}
	if ns := e.campanita("cambio_pedido"); len(ns) != 1 || !strings.Contains(ns[0].Texto, "Llamalo al 0351") {
		t.Fatalf("campanita de Nadia: %+v", ns)
	}
	e.enviar()
	if mn = e.meta.para("5493515123456"); len(mn) != 2 || !strings.Contains(mn[1]["text"].(map[string]any)["body"].(string), "/turno/"+nadia.Codigo) {
		t.Fatalf("respuesta a Nadia: %v", mn)
	}

	// Entregado y leído llegan como estados.
	ml := e.meta.para("5491134567890")
	var wamidLola string
	e.panel(200, "GET", "/v1/panel/whatsapp/mensajes?turno="+lola.ID, nil, &lista)
	for _, m := range lista.Mensajes {
		if m.Tipo == "confirmacion" {
			wamidLola = m.ID
		}
	}
	if len(ml) != 1 || wamidLola == "" {
		t.Fatalf("Lola: %v %v", ml, lista)
	}
	for i, m := range e.meta.mensajes {
		if m["to"] == "5491134567890" {
			if c := e.webhook(estado(fmt.Sprintf("wamid.%d", i+1), "read")); c != 200 {
				t.Fatalf("estado: %d", c)
			}
		}
	}
	e.panel(200, "GET", "/v1/panel/whatsapp/mensajes?turno="+lola.ID, nil, &lista)
	if lista.Mensajes[0].LeidoEn == nil || lista.Mensajes[0].EntregadoEn == nil {
		t.Fatalf("leído: %+v", lista.Mensajes[0])
	}

	// Cuatro horas sin contestar: Lola queda para llamar (Sofía confirmó, Nadia ya estaba).
	e.Ahora = en("2026-11-01 13:59")
	if r := e.correr(e.tj.SinRespuesta); r["marcados"] != 0 {
		t.Fatalf("antes de las 4 horas: %v", r)
	}
	e.Ahora = en("2026-11-01 14:01")
	if r := e.correr(e.tj.SinRespuesta); r["marcados"] != 1 {
		t.Fatalf("sin respuesta: %v", r)
	}
	llamar := e.paraLlamar()
	if llamar[lola.ID] != "sin_respuesta" || llamar[nadia.ID] != "cambio" || llamar[sofia.ID] != "" || len(e.campanita("sin_respuesta")) != 1 {
		t.Fatalf("para llamar: %v", llamar)
	}
	// Recepción llamó a Nadia; Lola confirma desde su link (y deja de estar para llamar).
	e.panel(204, "POST", "/v1/panel/turnos/"+nadia.ID+"/llamado", nil, nil)
	var c confirmaciones.Confirmacion
	e.Esperar(200, P{Metodo: "POST", Ruta: "/v1/t/" + lola.Codigo + "/confirmar"}, &c)
	if c.Estado != "confirmado" || *c.ConfirmadoPor != "web" {
		t.Fatalf("confirmar desde el link: %+v", c)
	}
	if llamar = e.paraLlamar(); len(llamar) != 1 || llamar[tomas.ID] == "" {
		t.Fatalf("después de llamar: %v", llamar)
	}

	// Lo que confirma el panel queda como panel; volver a pendiente lo borra.
	var tp turnos.Turno
	e.panel(200, "PATCH", "/v1/panel/turnos/"+nadia.ID, map[string]string{"estado": "confirmado"}, &tp)
	if tp.ConfirmadoPor == nil || *tp.ConfirmadoPor != "panel" || tp.ConfirmadoEn == nil {
		t.Fatalf("confirmado por el panel: %+v", tp)
	}
	e.panel(200, "PATCH", "/v1/panel/turnos/"+nadia.ID, map[string]string{"estado": "pendiente"}, &tp)
	if tp.ConfirmadoPor != nil || tp.ConfirmadoEn != nil {
		t.Fatalf("de nuevo pendiente: %+v", tp)
	}

	// Sofía vuelve a tocar «Confirmo» (otro webhook): no se contesta de nuevo.
	antes := e.meta.cuantos()
	if c := e.webhook(boton("wamid.R9", "wamid.1", confirmo, "Confirmo")); c != 200 {
		t.Fatalf("repetido: %d", c)
	}
	if e.enviar(); e.meta.cuantos() != antes {
		t.Fatal("un botón ya procesado no contesta de nuevo")
	}
	// Lola cancela y después toca «Confirmo»: le contestamos que ya no está vigente.
	e.panel(200, "PATCH", "/v1/panel/turnos/"+lola.ID, map[string]string{"estado": "cancelado"}, nil)
	if c := e.webhook(boton("wamid.R10", "", payload(t, ml[0], 0), "Confirmo")); c != 200 {
		t.Fatalf("Lola: %d", c)
	}
	e.enviar()
	if ml = e.meta.para("5491134567890"); len(ml) != 2 || !strings.Contains(ml[1]["text"].(map[string]any)["body"].(string), "ya no está vigente") {
		t.Fatalf("respuesta a Lola: %v", ml)
	}
	if e.turno(lola.ID).Estado != "cancelado" {
		t.Fatal("un botón no revive un turno cancelado")
	}
}

func TestRecordatorioPorWhatsApp(t *testing.T) {
	e := armar(t)
	e.Ahora = en("2026-11-01 10:00")
	// Con la confirmación apagada y el recordatorio prendido, sale 24 horas antes, con el link.
	e.panel(200, "PUT", "/v1/panel/ajustes/whatsapp", map[string]any{"confirmacion": false, "hora": "10:00", "sinRespuestaHoras": 4, "recordatorio": true}, nil)
	t1 := e.reservar("2026-11-02 10:30", "Sofía Méndez", "11 2345-6789")
	t2 := e.reservar("2026-11-02 10:00", "Nadia Paz", "11 2345-0000")
	if r := e.correr(e.tj.Confirmaciones); r["encolados"] != 0 {
		t.Fatalf("confirmación apagada: %v", r)
	}
	if r := e.correr(e.tj.Recordatorios); r["encolados"] != 2 {
		t.Fatalf("recordatorios: %v", r)
	}
	// Uno se cancela antes de que salga: se descarta.
	e.panel(200, "PATCH", "/v1/panel/turnos/"+t2.ID, map[string]string{"estado": "cancelado"}, nil)
	if r := e.enviar(); r.Enviados != 1 || r.Descartados != 1 {
		t.Fatalf("envío: %+v", r)
	}
	m := e.meta.para("5491123456789")[0]
	tpl := m["template"].(map[string]any)
	comps := tpl["components"].([]any)
	boton := comps[len(comps)-1].(map[string]any)
	if tpl["name"] != confirmaciones.PlantillaRecordatorio || boton["sub_type"] != "url" ||
		boton["parameters"].([]any)[0].(map[string]any)["text"] != t1.Codigo ||
		strings.Join(cuerpo(m), "|") != "Sofía|El Corte|Corte|lunes 2 de noviembre|10:30" {
		t.Fatalf("recordatorio: %v", m)
	}
	// Con la confirmación prendida, el recordatorio no sale (la confirmación ya recuerda).
	e.panel(200, "PUT", "/v1/panel/ajustes/whatsapp", map[string]any{"confirmacion": true, "hora": "10:00", "sinRespuestaHoras": 4, "recordatorio": true}, nil)
	e.reservar("2026-11-02 09:30", "Lola Fernández", "11 3456-7890")
	if r := e.correr(e.tj.Recordatorios); r["encolados"] != 0 {
		t.Fatalf("con confirmación: %v", r)
	}
}

func TestWhatsAppSinConfigurar(t *testing.T) {
	e := prueba.Nueva(t, en("2026-11-01 08:00"), func(a *prueba.API) []servidor.Dominio {
		return []servidor.Dominio{&negocios.Servicio{Pool: a.Bases.App, Sesion: a.Sesion},
			&confirmaciones.Servicio{Pool: a.Bases.App, Sesion: a.Sesion, Sitio: sitio}}
	})
	var n negocios.Negocio
	e.Esperar(201, P{Metodo: "POST", Ruta: "/v1/negocios", Quien: duenia, Cuerpo: map[string]string{"slug": "apagado", "nombre": "Apagado"}}, &n)
	var a confirmaciones.Ajustes
	e.Esperar(200, P{Metodo: "GET", Ruta: "/v1/panel/ajustes/whatsapp", Quien: duenia, Negocio: n.ID}, &a)
	if a.Disponible {
		t.Fatal("sin configuración no está disponible")
	}
	e.Esperar(409, P{Metodo: "PUT", Ruta: "/v1/panel/ajustes/whatsapp", Quien: duenia, Negocio: n.ID,
		Cuerpo: map[string]any{"confirmacion": true, "hora": "10:00", "sinRespuestaHoras": 4}}, nil)
	e.Esperar(404, P{Metodo: "POST", Ruta: "/v1/webhooks/whatsapp", Cuerpo: map[string]string{}}, nil)
	e.Esperar(403, P{Metodo: "GET", Ruta: "/v1/webhooks/whatsapp?hub.mode=subscribe&hub.verify_token=&hub.challenge=1"}, nil)
	if confirmaciones.Nuevos(e.Bases.App, nil, sitio, nil).Lista() != nil {
		t.Fatal("sin cliente no hay trabajos")
	}
}
