package whatsapp

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestE164(t *testing.T) {
	bien := map[string]string{
		"11 2345-6789":        "+5491123456789",
		"011 15 2345-6789":    "+5491123456789",
		"(011) 15-2345-6789":  "+5491123456789",
		"+54 9 11 2345 6789":  "+5491123456789",
		"+54 11 2345 6789":    "+5491123456789",
		"5491123456789":       "+5491123456789",
		"0054 9 11 2345-6789": "+5491123456789",
		"0351 155 123456":     "+5493515123456",
		"351 15 512 3456":     "+5493515123456",
		"02966 15 123456":     "+5492966123456",
		"+54 9 0351 512-3456": "+5493515123456",
		"+598 99 123 456":     "+59899123456",
		"+1 (415) 555-0100":   "+14155550100",
	}
	for in, quiero := range bien {
		got, err := E164(in)
		if err != nil || got != quiero {
			t.Errorf("E164(%q) = %q, %v; quería %q", in, got, err, quiero)
		}
	}
	for _, mal := range []string{"", "15 2345 6789", "1234", "+0 123", "abc", "11 2345 678"} {
		if got, err := E164(mal); err == nil {
			t.Errorf("E164(%q) = %q: tenía que fallar", mal, got)
		}
	}
}

func TestFirma(t *testing.T) {
	cuerpo := []byte(`{"object":"whatsapp_business_account"}`)
	f := Firmar("secreto", cuerpo)
	if !FirmaValida("secreto", cuerpo, f) {
		t.Fatal("la firma propia no valida")
	}
	if FirmaValida("otro", cuerpo, f) || FirmaValida("secreto", append(cuerpo, ' '), f) ||
		FirmaValida("secreto", cuerpo, strings.TrimPrefix(f, "sha256=")) || FirmaValida("", cuerpo, f) {
		t.Fatal("validó una firma que no es")
	}
}

func TestInterpretar(t *testing.T) {
	cuerpo := `{"object":"whatsapp_business_account","entry":[{"id":"1","changes":[{"field":"messages","value":{
		"messaging_product":"whatsapp","metadata":{"display_phone_number":"5491100000000","phone_number_id":"PNID"},
		"messages":[
			{"from":"5491123456789","id":"wamid.R1","timestamp":"1793530920","type":"button",
			 "context":{"from":"5491100000000","id":"wamid.O1"},"button":{"payload":"confirmo:abc","text":"Confirmo"}},
			{"from":"5491123456789","id":"wamid.R2","timestamp":"1793530921","type":"interactive",
			 "interactive":{"type":"button_reply","button_reply":{"id":"cambiar:abc","title":"Necesito cambiarlo"}}},
			{"from":"5491123456789","id":"wamid.R3","timestamp":"1793530922","type":"text","text":{"body":"hola"}}],
		"statuses":[
			{"id":"wamid.O1","status":"delivered","timestamp":"1793530900","recipient_id":"5491123456789"},
			{"id":"wamid.O2","status":"failed","timestamp":"1793530901","errors":[{"code":131026,"title":"Message undeliverable"}]},
			{"id":"wamid.O3","status":"failed","timestamp":"1793530902","errors":[{"code":132001,"title":"Template does not exist"}]}]
	}}]}]}`
	rs, es, err := Interpretar([]byte(cuerpo))
	if err != nil {
		t.Fatal(err)
	}
	if len(rs) != 2 || rs[0].Payload != "confirmo:abc" || rs[0].Contexto != "wamid.O1" || rs[0].TelefonoID != "PNID" ||
		rs[1].Payload != "cambiar:abc" || rs[1].Texto != "Necesito cambiarlo" || rs[0].Cuando.Unix() != 1793530920 {
		t.Fatalf("respuestas: %+v", rs)
	}
	if len(es) != 3 || es[0].Estado != "delivered" || es[0].Error != nil || !errors.Is(es[1].Error, ErrDestinoInvalido) {
		t.Fatalf("estados: %+v", es)
	}
	var e *Error
	if !errors.As(es[2].Error, &e) || e.Codigo != 132001 || e.Reintentable() {
		t.Fatalf("estado fallido: %+v", es[2].Error)
	}
}

// falso hace de la Graph API: guarda el último cuerpo y responde lo que diga responder.
type falso struct {
	ultimo     map[string]any
	auth, ruta string
	responder  func() (int, string)
}

func (f *falso) servidor(t *testing.T) *httptest.Server {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.auth, f.ruta = r.Header.Get("Authorization"), r.URL.Path
		b, _ := io.ReadAll(r.Body)
		f.ultimo = nil
		_ = json.Unmarshal(b, &f.ultimo)
		estado, cuerpo := f.responder()
		w.WriteHeader(estado)
		_, _ = io.WriteString(w, cuerpo)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestEnviarPlantilla(t *testing.T) {
	f := &falso{responder: func() (int, string) {
		return 200, `{"messaging_product":"whatsapp","contacts":[{"input":"5491123456789","wa_id":"5491123456789"}],"messages":[{"id":"wamid.NUEVO"}]}`
	}}
	srv := f.servidor(t)
	c := &Cliente{Token: "tok", TelefonoID: "PNID", Base: srv.URL, HTTP: srv.Client()}
	id, err := c.EnviarPlantilla(context.Background(), Plantilla{Para: "+5491123456789", Nombre: "tengolibre_confirmacion",
		Cuerpo: []string{"Sofía", "lunes 2"}, Botones: []Boton{{Payload: "confirmo:x"}, {Payload: "cambiar:x"}}})
	if err != nil || id != "wamid.NUEVO" {
		t.Fatalf("envío: %q %v", id, err)
	}
	if f.auth != "Bearer tok" || f.ruta != "/"+VersionDefecto+"/PNID/messages" || f.ultimo["to"] != "5491123456789" {
		t.Fatalf("pedido: %s %s %v", f.auth, f.ruta, f.ultimo)
	}
	tpl := f.ultimo["template"].(map[string]any)
	comps := tpl["components"].([]any)
	if tpl["name"] != "tengolibre_confirmacion" || tpl["language"].(map[string]any)["code"] != "es_AR" || len(comps) != 3 {
		t.Fatalf("plantilla: %v", tpl)
	}
	b1 := comps[2].(map[string]any)
	if b1["sub_type"] != "quick_reply" || b1["index"] != "1" || b1["parameters"].([]any)[0].(map[string]any)["payload"] != "cambiar:x" {
		t.Fatalf("botón: %v", b1)
	}

	// Errores: número sin WhatsApp, límite (reintentable), plantilla mal (no reintentable).
	casos := []struct {
		estado     int
		cuerpo     string
		destino    bool
		reintentar bool
	}{
		{400, `{"error":{"message":"Message undeliverable","code":131026}}`, true, false},
		{400, `{"error":{"message":"Spam rate limit hit","code":131048}}`, false, true},
		{500, `{"error":{"message":"Service unavailable","code":2}}`, false, true},
		{404, `{"error":{"message":"Template name does not exist","code":132001,"error_data":{"details":"tengolibre_x"}}}`, false, false},
	}
	for _, k := range casos {
		f.responder = func() (int, string) { return k.estado, k.cuerpo }
		_, err := c.EnviarTexto(context.Background(), "+5491123456789", "hola")
		var e *Error
		switch {
		case k.destino && !errors.Is(err, ErrDestinoInvalido):
			t.Errorf("%s: %v", k.cuerpo, err)
		case !k.destino && (!errors.As(err, &e) || e.Reintentable() != k.reintentar):
			t.Errorf("%s: %v", k.cuerpo, err)
		}
	}
	if f.ultimo["type"] != "text" {
		t.Fatalf("texto: %v", f.ultimo)
	}
	var nada *Cliente
	if id, err := nada.EnviarTexto(context.Background(), "+1", "x"); id != "" || err != nil || nada.Activo() || Desde("", "x", "") != nil {
		t.Fatal("un cliente sin configurar tiene que quedar apagado")
	}
}
