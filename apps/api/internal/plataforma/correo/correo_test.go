package correo_test

import (
	"bytes"
	"context"
	"io"
	"mime"
	"mime/multipart"
	"net/mail"
	"strings"
	"testing"
	"time"

	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/correo"
)

// El mail con adjunto se arma como mixed { alternative { texto, html }, adjunto } y se lee de vuelta.
func TestArmarConAdjunto(t *testing.T) {
	m := correo.Mensaje{Para: "lucia@ejemplo.com", Asunto: "Turno reservado — El Corte", Texto: "Hola Lucía",
		HTML: "<p>Hola Lucía</p>" + strings.Repeat("x", 2000), Baja: "https://tengolibre.com.ar/baja",
		Adjuntos: []correo.Adjunto{{Nombre: "turno.ics", Tipo: "text/calendar; method=PUBLISH", Datos: []byte("BEGIN:VCALENDAR")}}}
	crudo, err := correo.Armar("TengoLibre <hola@tengolibre.com.ar>", m, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	msj, err := mail.ReadMessage(bytes.NewReader(crudo))
	if err != nil {
		t.Fatal(err)
	}
	dec := new(mime.WordDecoder)
	asunto, _ := dec.DecodeHeader(msj.Header.Get("Subject"))
	if asunto != m.Asunto || msj.Header.Get("List-Unsubscribe") != "<https://tengolibre.com.ar/baja>" {
		t.Fatalf("encabezados: %v", msj.Header)
	}
	for _, linea := range strings.Split(string(crudo), "\r\n") {
		if len(linea) > 998 {
			t.Fatal("una línea pasa el límite de SMTP")
		}
	}
	_, params, _ := mime.ParseMediaType(msj.Header.Get("Content-Type"))
	r := multipart.NewReader(msj.Body, params["boundary"])
	var tipos []string
	for {
		p, err := r.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		tipos = append(tipos, p.Header.Get("Content-Type"))
	}
	if len(tipos) != 2 || !strings.HasPrefix(tipos[0], "multipart/alternative") || !strings.HasPrefix(tipos[1], "text/calendar") {
		t.Fatalf("partes: %v", tipos)
	}
}

func TestSinInyeccionDeEncabezados(t *testing.T) {
	if _, err := correo.Armar("a@b.c", correo.Mensaje{Para: "x@y.z\r\nBcc: otro@z.z"}, time.Now()); err == nil {
		t.Fatal("aceptó un salto de línea en Para")
	}
}

func TestMemoria(t *testing.T) {
	m := &correo.Memoria{}
	_ = m.Enviar(context.Background(), correo.Mensaje{Para: "a@b.c"})
	if len(m.Para("a@b.c")) != 1 || m.Cantidad() != 1 {
		t.Fatal("memoria")
	}
}
