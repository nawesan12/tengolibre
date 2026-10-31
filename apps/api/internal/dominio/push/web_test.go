package push_test

import (
	"context"
	"crypto/ecdh"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/avisos"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/negocios"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/push"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/fcm"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/prueba"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/servidor"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/webpush"
)

type P = prueba.Pedido

const (
	duenia = "user_duenia0001"
	profe  = "user_profe00001"
)

// navegadorFalso devuelve una suscripción con claves de verdad (como la de PushSubscription.toJSON()).
func navegadorFalso(t *testing.T, endpoint string) map[string]any {
	t.Helper()
	k, err := ecdh.P256().GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	a := make([]byte, 16)
	_, _ = rand.Read(a)
	b64 := base64.RawURLEncoding
	return map[string]any{"endpoint": endpoint, "expirationTime": nil, "aparato": "Chrome · Mac",
		"keys": map[string]string{"p256dh": b64.EncodeToString(k.PublicKey().Bytes()), "auth": b64.EncodeToString(a)}}
}

// webFalso y fcmFalso guardan lo que se manda; un endpoint en vencidos responde que ya no existe.
type webFalso struct {
	mu       sync.Mutex
	envios   []webpush.Envio
	vencidos []string
}

func (f *webFalso) Enviar(_ context.Context, e webpush.Envio) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.envios = append(f.envios, e)
	if slices.Contains(f.vencidos, e.Endpoint) {
		return webpush.ErrVencida
	}
	return nil
}

type fcmFalso struct{ envios []fcm.Envio }

func (f *fcmFalso) Enviar(_ context.Context, e fcm.Envio) error {
	f.envios = append(f.envios, e)
	return nil
}

// con manda un pedido con la sesión propia (prueba.Hacer arma "dev:…", que no la tiene).
func con(a *prueba.API, metodo, ruta, token, negocio string, cuerpo any) *httptest.ResponseRecorder {
	var b strings.Builder
	if cuerpo != nil {
		_ = json.NewEncoder(&b).Encode(cuerpo)
	}
	req := httptest.NewRequest(metodo, ruta, strings.NewReader(b.String()))
	req.Header.Set("Authorization", "Bearer "+token)
	if negocio != "" {
		req.Header.Set("X-Negocio", negocio)
	}
	rec := httptest.NewRecorder()
	a.Handler.ServeHTTP(rec, req)
	return rec
}

func sesionPropia(t *testing.T, a *prueba.API, quien string) string {
	t.Helper()
	var r struct {
		Token string `json:"token"`
	}
	a.Esperar(201, P{Metodo: "POST", Ruta: "/v1/auth/clerk", Cuerpo: map[string]string{"token": "dev:" + quien}}, &r)
	return r.Token
}

func TestPushWebYPreferencias(t *testing.T) {
	pub, priv, err := webpush.GenerarClaves()
	if err != nil {
		t.Fatal(err)
	}
	cli, err := webpush.Desde(pub, priv, "mailto:hola@tengolibre.com.ar")
	if err != nil {
		t.Fatal(err)
	}
	a := prueba.Nueva(t, time.Now(), func(a *prueba.API) []servidor.Dominio {
		return []servidor.Dominio{a.Sesion, &negocios.Servicio{Pool: a.Bases.App, Sesion: a.Sesion},
			&push.Servicio{Pool: a.Bases.App, Sesion: a.Sesion, ClaveVAPID: cli.ClavePublica()}}
	})
	ctx := context.Background()
	var n negocios.Negocio
	a.Esperar(201, P{Metodo: "POST", Ruta: "/v1/negocios", Quien: duenia, Cuerpo: map[string]string{"slug": "lumi", "nombre": "Lumi"}}, &n)
	err = a.Bases.App.EnNegocio(ctx, n.ID, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, "INSERT INTO miembros (negocio_id, clerk_id, rol) VALUES (app.negocio_actual(), $1, 'profesional')", profe)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}

	// La clave pública es pública; la tarjeta de Inicio pregunta si este navegador ya está suscripto.
	var v struct {
		Clave  string `json:"clave"`
		Activo bool   `json:"activo"`
	}
	a.Esperar(200, P{Metodo: "GET", Ruta: "/v1/push/vapid"}, &v)
	if v.Clave != pub || !v.Activo {
		t.Fatalf("vapid: %+v", v)
	}
	td, tp := sesionPropia(t, a, duenia), sesionPropia(t, a, profe)
	var est push.EstadoWeb
	if rec := con(a, "GET", "/v1/yo/suscripciones-web", td, "", nil); rec.Code != 200 || json.Unmarshal(rec.Body.Bytes(), &est) != nil || est.Suscripta {
		t.Fatalf("estado inicial: %d %s", rec.Code, rec.Body)
	}

	// Suscribirse: sólo con la sesión propia y de un servicio de push conocido.
	epD, epP := "https://fcm.googleapis.com/fcm/send/duenia", "https://web.push.apple.com/profe"
	a.Esperar(403, P{Metodo: "POST", Ruta: "/v1/yo/suscripciones-web", Quien: duenia, Cuerpo: navegadorFalso(t, epD)}, nil)
	if rec := con(a, "POST", "/v1/yo/suscripciones-web", td, "", navegadorFalso(t, "https://malo.example.com/x")); rec.Code != 400 {
		t.Fatalf("endpoint desconocido: %d", rec.Code)
	}
	for _, x := range []struct{ token, ep string }{{td, epD}, {tp, epP}, {tp, epP}} {
		if rec := con(a, "POST", "/v1/yo/suscripciones-web", x.token, "", navegadorFalso(t, x.ep)); rec.Code != 204 {
			t.Fatalf("suscribir: %d %s", rec.Code, rec.Body)
		}
	}
	if rec := con(a, "GET", "/v1/yo/suscripciones-web", td, "", nil); json.Unmarshal(rec.Body.Bytes(), &est) != nil || !est.Suscripta || est.Clave != pub {
		t.Fatalf("estado: %s", rec.Body)
	}
	// La profesional también tiene la app en Android.
	if rec := con(a, "POST", "/v1/yo/dispositivos", tp, "", map[string]string{"plataforma": "android", "token": "fcm:token_de_la_profe_123"}); rec.Code != 204 {
		t.Fatalf("android: %d", rec.Code)
	}

	// Preferencias: la dueña ve todo por defecto y apaga las reseñas; la profesional, lo de su rol.
	var pd, pp push.Preferencias
	if rec := con(a, "GET", "/v1/panel/push/preferencias", td, n.ID, nil); rec.Code != 200 || json.Unmarshal(rec.Body.Bytes(), &pd) != nil ||
		!pd.PorDefecto || len(pd.Apagados) != 0 || len(pd.Tipos) != len(push.Tipos) {
		t.Fatalf("preferencias de la dueña: %s", rec.Body)
	}
	if rec := con(a, "PUT", "/v1/panel/push/preferencias", td, n.ID, map[string]any{"apagados": []string{"resena"}}); rec.Code != 200 ||
		json.Unmarshal(rec.Body.Bytes(), &pd) != nil || pd.PorDefecto || !slices.Equal(pd.Apagados, []string{"resena"}) {
		t.Fatalf("guardar: %d %s", rec.Code, rec.Body)
	}
	if rec := con(a, "PUT", "/v1/panel/push/preferencias", td, n.ID, map[string]any{"apagados": []string{"Reserva!"}}); rec.Code != 400 {
		t.Fatalf("tipo inválido: %d", rec.Code)
	}
	if rec := con(a, "GET", "/v1/panel/push/preferencias", tp, n.ID, nil); json.Unmarshal(rec.Body.Bytes(), &pp) != nil ||
		pp.Rol != "profesional" || !pp.PorDefecto || !slices.Contains(pp.Apagados, "equipo") {
		t.Fatalf("preferencias de la profesional: %s", rec.Body)
	}

	// Llegan cuatro notificaciones; el worker manda a cada uno lo suyo.
	err = a.Bases.App.EnNegocio(ctx, n.ID, func(tx pgx.Tx) error {
		for _, x := range []avisos.Notificacion{
			{Tipo: "reserva", Titulo: "Nueva reserva de Florencia", Texto: "Lifting hoy 16:30 con Abril"},
			{Tipo: "resena", Titulo: "Nueva reseña", Texto: "5 estrellas"},
			{Tipo: "equipo", Titulo: "Abril aceptó la invitación"},
			{Tipo: "confirmacion", Titulo: "Sofía confirmó por WhatsApp"},
		} {
			if err := avisos.Notificar(ctx, tx, x); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	web := &webFalso{vencidos: []string{epD}}
	f := &fcmFalso{}
	env := push.Nuevo(a.Bases.App, nil, nil, nil)
	env.Web, env.FCM = web, f
	if _, err := env.Correr(ctx, time.Now()); err != nil {
		t.Fatal(err)
	}
	titulos := func(ep string) []string {
		var out []string
		for _, e := range web.envios {
			if e.Endpoint == ep {
				var p struct{ Title, URL, Negocio, Tipo string }
				_ = json.Unmarshal(e.Payload, &p)
				if p.URL != "/panel/notificaciones" || p.Negocio != n.ID {
					t.Errorf("payload: %s", e.Payload)
				}
				out = append(out, p.Title)
			}
		}
		slices.Sort(out)
		return out
	}
	if got := titulos(epD); strings.Join(got, "|") != "Abril aceptó la invitación|Nueva reserva de Florencia|Sofía confirmó por WhatsApp" {
		t.Fatalf("a la dueña: %v", got)
	}
	if got := titulos(epP); strings.Join(got, "|") != "Nueva reserva de Florencia|Nueva reseña" {
		t.Fatalf("a la profesional: %v", got)
	}
	var android []string
	for _, e := range f.envios {
		android = append(android, e.Titulo)
	}
	slices.Sort(android)
	if strings.Join(android, "|") != "Nueva reserva de Florencia|Nueva reseña" {
		t.Fatalf("al Android de la profesional: %v", android)
	}

	// La suscripción vencida de la dueña se borró; la profesional se da de baja sola.
	if rec := con(a, "GET", "/v1/yo/suscripciones-web", td, "", nil); json.Unmarshal(rec.Body.Bytes(), &est) != nil || est.Suscripta {
		t.Fatalf("vencida: %s", rec.Body)
	}
	if rec := con(a, "DELETE", "/v1/yo/suscripciones-web", tp, "", nil); rec.Code != 204 {
		t.Fatalf("baja: %d", rec.Code)
	}
	if rec := con(a, "GET", "/v1/yo/suscripciones-web", tp, "", nil); json.Unmarshal(rec.Body.Bytes(), &est) != nil || est.Suscripta {
		t.Fatalf("después de la baja: %s", rec.Body)
	}
	// Volver a lo de su rol.
	if rec := con(a, "PUT", "/v1/panel/push/preferencias", td, n.ID, map[string]any{"porDefecto": true}); rec.Code != 200 ||
		json.Unmarshal(rec.Body.Bytes(), &pd) != nil || !pd.PorDefecto || len(pd.Apagados) != 0 {
		t.Fatalf("por defecto: %s", rec.Body)
	}
}

func TestPushWebApagado(t *testing.T) {
	a := prueba.Nueva(t, time.Now(), func(a *prueba.API) []servidor.Dominio {
		return []servidor.Dominio{a.Sesion, &push.Servicio{Pool: a.Bases.App, Sesion: a.Sesion}}
	})
	var v struct {
		Activo bool `json:"activo"`
	}
	a.Esperar(200, P{Metodo: "GET", Ruta: "/v1/push/vapid"}, &v)
	if v.Activo {
		t.Fatal("sin claves está apagado")
	}
	if rec := con(a, "POST", "/v1/yo/suscripciones-web", sesionPropia(t, a, duenia), "",
		navegadorFalso(t, "https://fcm.googleapis.com/fcm/send/x")); rec.Code != 409 {
		t.Fatalf("suscribir apagado: %d", rec.Code)
	}
}

func TestPayloadWeb(t *testing.T) {
	var p map[string]string
	_ = json.Unmarshal(push.PayloadWeb(push.Aviso{Tipo: "reserva", Titulo: "Nueva reserva", Ruta: "turnos/abc", Negocio: "n1", Colapso: "noti-1"}), &p)
	if p["url"] != "/panel/turnos/abc" || p["title"] != "Nueva reserva" || p["tag"] != "noti-1" || p["negocio"] != "n1" || p["tipo"] != "reserva" {
		t.Fatalf("payload: %v", p)
	}
	if !push.Apagado("equipo", "recepcion", nil) || push.Apagado("equipo", "duenio", nil) || push.Apagado("equipo", "recepcion", []string{}) {
		t.Fatal("apagados por rol")
	}
}
