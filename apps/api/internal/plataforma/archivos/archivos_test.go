package archivos_test

import (
	"bytes"
	"image"
	"image/png"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/negocios"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/archivos"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/prueba"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/servidor"
)

type P = prueba.Pedido

func pngDe(t *testing.T, ancho, alto int) []byte {
	t.Helper()
	var b bytes.Buffer
	if err := png.Encode(&b, image.NewRGBA(image.Rect(0, 0, ancho, alto))); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

// subir manda un multipart con el archivo; tipo es el Content-Type que "dice" el navegador.
func subir(a *prueba.API, quien, negocio, nombre, tipo string, cuerpo []byte) prueba.Respuesta {
	a.T.Helper()
	var b bytes.Buffer
	mw := multipart.NewWriter(&b)
	h := make(map[string][]string)
	h["Content-Disposition"] = []string{`form-data; name="archivo"; filename="` + nombre + `"`}
	h["Content-Type"] = []string{tipo}
	parte, _ := mw.CreatePart(h)
	_, _ = parte.Write(cuerpo)
	_ = mw.Close()
	req := httptest.NewRequest("POST", "/v1/panel/archivos", &b)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("Authorization", "Bearer dev:"+quien)
	req.Header.Set("X-Negocio", negocio)
	rec := httptest.NewRecorder()
	a.Handler.ServeHTTP(rec, req)
	return prueba.Respuesta{Estado: rec.Code, Cuerpo: rec.Body.Bytes()}
}

func TestSubirServirYBorrar(t *testing.T) {
	carpeta := t.TempDir()
	a := prueba.Nueva(t, time.Now(), func(a *prueba.API) []servidor.Dominio {
		return []servidor.Dominio{
			&negocios.Servicio{Pool: a.Bases.App, Sesion: a.Sesion},
			&archivos.Servicio{Pool: a.Bases.App, Sesion: a.Sesion, Carpeta: carpeta, URLBase: "https://api.prueba/archivos"},
		}
	})
	var uno, dos negocios.Negocio
	a.Esperar(201, P{Metodo: "POST", Ruta: "/v1/negocios", Quien: "user_duenia0001", Cuerpo: map[string]string{"slug": "uno", "nombre": "Uno"}}, &uno)
	a.Esperar(201, P{Metodo: "POST", Ruta: "/v1/negocios", Quien: "user_duenia0002", Cuerpo: map[string]string{"slug": "dos", "nombre": "Dos"}}, &dos)

	// Lo que no es imagen no entra, diga lo que diga el nombre o el Content-Type.
	for _, c := range []struct {
		nombre string
		cuerpo []byte
		codigo string
	}{
		{"foto.png", []byte("<script>alert(1)</script>"), "tipo_no_aceptado"},
		{"foto.jpg", []byte("%PDF-1.4 no soy una foto"), "tipo_no_aceptado"},
		{"foto.svg", []byte(`<svg xmlns="http://www.w3.org/2000/svg"></svg>`), "tipo_no_aceptado"},
		{"rota.png", append([]byte("\x89PNG\r\n\x1a\n"), make([]byte, 40)...), "imagen_rota"},
	} {
		r := subir(a, "user_duenia0001", uno.ID, c.nombre, "image/png", c.cuerpo)
		if r.Estado != http.StatusUnprocessableEntity || !strings.Contains(string(r.Cuerpo), c.codigo) {
			t.Errorf("%s: %d %s, quería 422 %s", c.nombre, r.Estado, r.Cuerpo, c.codigo)
		}
	}
	// Muy grande.
	if r := subir(a, "user_duenia0001", uno.ID, "enorme.png", "image/png", make([]byte, archivos.MaxBytes+10)); r.Estado != 422 {
		t.Errorf("una imagen de más de 5 MB: %d %s", r.Estado, r.Cuerpo)
	}
	// Sin sesión del negocio, nada.
	if r := subir(a, "user_duenia0002", uno.ID, "x.png", "image/png", pngDe(t, 4, 4)); r.Estado != 403 {
		t.Errorf("subir a un negocio ajeno: %d", r.Estado)
	}

	r := subir(a, "user_duenia0001", uno.ID, "logo.png", "application/octet-stream", pngDe(t, 40, 20))
	if r.Estado != 201 {
		t.Fatalf("subir: %d %s", r.Estado, r.Cuerpo)
	}
	var arch archivos.Archivo
	r.Leer(t, &arch)
	if arch.Tipo != "image/png" || arch.Ancho != 40 || arch.Alto != 20 || !strings.HasSuffix(arch.Nombre, ".png") ||
		arch.URL != "https://api.prueba/archivos/"+uno.ID+"/"+arch.Nombre {
		t.Fatalf("archivo: %+v", arch)
	}
	if _, err := os.Stat(filepath.Join(carpeta, uno.ID, arch.Nombre)); err != nil {
		t.Fatal("no quedó en el disco:", err)
	}

	// Se sirve sin sesión, con caché larga y el tipo bien puesto.
	req := httptest.NewRequest("GET", "/archivos/"+uno.ID+"/"+arch.Nombre, nil)
	rec := httptest.NewRecorder()
	a.Handler.ServeHTTP(rec, req)
	if rec.Code != 200 || rec.Header().Get("Content-Type") != "image/png" || !strings.Contains(rec.Header().Get("Cache-Control"), "immutable") ||
		rec.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("servir: %d %v", rec.Code, rec.Header())
	}
	for _, ruta := range []string{"/archivos/" + uno.ID + "/..%2F..%2Fetc", "/archivos/../" + arch.Nombre, "/archivos/" + dos.ID + "/" + arch.Nombre} {
		rec := httptest.NewRecorder()
		a.Handler.ServeHTTP(rec, httptest.NewRequest("GET", ruta, nil))
		if rec.Code == 200 {
			t.Errorf("%s: 200", ruta)
		}
	}

	// El otro negocio no lo ve ni lo puede borrar.
	var lista struct {
		Archivos []archivos.Archivo `json:"archivos"`
	}
	a.Esperar(200, P{Metodo: "GET", Ruta: "/v1/panel/archivos", Quien: "user_duenia0002", Negocio: dos.ID}, &lista)
	if len(lista.Archivos) != 0 {
		t.Fatal("el otro negocio ve el archivo")
	}
	a.Esperar(404, P{Metodo: "DELETE", Ruta: "/v1/panel/archivos/" + arch.ID, Quien: "user_duenia0002", Negocio: dos.ID}, nil)
	a.Esperar(200, P{Metodo: "GET", Ruta: "/v1/panel/archivos", Quien: "user_duenia0001", Negocio: uno.ID}, &lista)
	if len(lista.Archivos) != 1 {
		t.Fatal("el dueño tiene que ver su archivo")
	}

	a.Esperar(204, P{Metodo: "DELETE", Ruta: "/v1/panel/archivos/" + arch.ID, Quien: "user_duenia0001", Negocio: uno.ID}, nil)
	if _, err := os.Stat(filepath.Join(carpeta, uno.ID, arch.Nombre)); !os.IsNotExist(err) {
		t.Fatal("el archivo sigue en el disco")
	}
	rec = httptest.NewRecorder()
	a.Handler.ServeHTTP(rec, httptest.NewRequest("GET", "/archivos/"+uno.ID+"/"+arch.Nombre, nil))
	if rec.Code != 404 {
		t.Fatalf("borrado y se sigue sirviendo: %d", rec.Code)
	}
}
