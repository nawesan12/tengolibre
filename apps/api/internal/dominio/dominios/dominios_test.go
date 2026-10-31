package dominios_test

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/dominios"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/negocios"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/prueba"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/servidor"
)

type P = prueba.Pedido

// dnsFalso: registros cargados a mano.
type dnsFalso struct {
	mu    sync.Mutex
	txt   map[string][]string
	cname map[string]string
	a     map[string][]string
}

var errNX = errors.New("no such host")

func (d *dnsFalso) LookupTXT(_ context.Context, n string) ([]string, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if v, ok := d.txt[n]; ok {
		return v, nil
	}
	return nil, errNX
}

func (d *dnsFalso) LookupCNAME(_ context.Context, n string) (string, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if v, ok := d.cname[n]; ok {
		return v, nil
	}
	return n + ".", nil // como net: sin CNAME devuelve el mismo nombre
}

func (d *dnsFalso) LookupHost(_ context.Context, n string) ([]string, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if v, ok := d.a[n]; ok {
		return v, nil
	}
	return nil, errNX
}

type respuesta struct {
	Dominio *dominios.Dominio `json:"dominio"`
}

func txtDe(t *testing.T, d *dominios.Dominio) string {
	t.Helper()
	for _, r := range d.Registros {
		if r.Tipo == "TXT" {
			return r.Valor
		}
	}
	t.Fatalf("sin TXT: %+v", d.Registros)
	return ""
}

func TestNormalizar(t *testing.T) {
	for entrada, quiero := range map[string]string{
		"  HTTPS://Turnos.MiBarberia.com/reservar?x=1 ": "turnos.mibarberia.com",
		"mibarberia.com.ar.":                            "mibarberia.com.ar",
		"mibarberia.com:443":                            "mibarberia.com",
	} {
		if d, err := dominios.Normalizar(entrada, "tengolibre.com.ar"); err != nil || d != quiero {
			t.Errorf("%q: %q %v, quería %q", entrada, d, err, quiero)
		}
	}
	for _, malo := range []string{"", "localhost", "tengolibre.com.ar", "x.tengolibre.com.ar", "mi barberia.com", "-a.com", "a..com", "1.2.3.4"} {
		if _, err := dominios.Normalizar(malo, "tengolibre.com.ar"); err == nil {
			t.Errorf("%q tendría que ser inválido", malo)
		}
	}
}

func TestDominioPropioDePuntaAPunta(t *testing.T) {
	dns := &dnsFalso{txt: map[string][]string{}, cname: map[string]string{}, a: map[string][]string{}}
	a := prueba.Nueva(t, time.Now(), func(a *prueba.API) []servidor.Dominio {
		return []servidor.Dominio{
			&negocios.Servicio{Pool: a.Bases.App, Sesion: a.Sesion},
			&dominios.Servicio{Pool: a.Bases.App, Sesion: a.Sesion, Resolver: dns, Destino: "dominios.tengolibre.com.ar", IP: "187.41.102.190"},
		}
	})
	const ana, beto = "user_anaduenia01", "user_betoduenio1"
	var uno, dos negocios.Negocio
	a.Esperar(201, P{Metodo: "POST", Ruta: "/v1/negocios", Quien: ana, Cuerpo: map[string]string{"slug": "uno", "nombre": "Uno"}}, &uno)
	a.Esperar(201, P{Metodo: "POST", Ruta: "/v1/negocios", Quien: beto, Cuerpo: map[string]string{"slug": "dos", "nombre": "Dos"}}, &dos)

	var r respuesta
	a.Esperar(200, P{Metodo: "GET", Ruta: "/v1/panel/dominio", Quien: ana, Negocio: uno.ID}, &r)
	if r.Dominio != nil {
		t.Fatal("sin dominio todavía")
	}
	a.Esperar(400, P{Metodo: "PUT", Ruta: "/v1/panel/dominio", Quien: ana, Negocio: uno.ID, Cuerpo: map[string]string{"dominio": "tengolibre.com.ar"}}, nil)
	a.Esperar(200, P{Metodo: "PUT", Ruta: "/v1/panel/dominio", Quien: ana, Negocio: uno.ID, Cuerpo: map[string]string{"dominio": "https://Turnos.Uno.com/"}}, &r)
	if r.Dominio == nil || r.Dominio.Nombre != "turnos.uno.com" || r.Dominio.Estado != "pendiente" || len(r.Dominio.Registros) != 2 {
		t.Fatalf("alta: %+v", r.Dominio)
	}
	txtUno := txtDe(t, r.Dominio)
	// Volver a ponerlo no cambia el token (ya puede estar cargado en el DNS).
	a.Esperar(200, P{Metodo: "PUT", Ruta: "/v1/panel/dominio", Quien: ana, Negocio: uno.ID, Cuerpo: map[string]string{"dominio": "turnos.uno.com"}}, &r)
	if txtDe(t, r.Dominio) != txtUno {
		t.Fatal("el token cambió")
	}

	// Sin DNS: error con lo que falta.
	a.Esperar(200, P{Metodo: "POST", Ruta: "/v1/panel/dominio/verificar", Quien: ana, Negocio: uno.ID}, &r)
	if r.Dominio.Estado != "error" || !strings.Contains(r.Dominio.Detalle, "TXT") {
		t.Fatalf("sin DNS: %+v", r.Dominio)
	}
	// Con el TXT pero sin apuntar.
	dns.txt["_tengolibre.turnos.uno.com"] = []string{"otra-cosa", txtUno}
	a.Esperar(200, P{Metodo: "POST", Ruta: "/v1/panel/dominio/verificar", Quien: ana, Negocio: uno.ID}, &r)
	if r.Dominio.Estado != "error" || !strings.Contains(r.Dominio.Detalle, "CNAME") {
		t.Fatalf("sin CNAME: %+v", r.Dominio)
	}
	a.Esperar(404, P{Metodo: "GET", Ruta: "/v1/host/turnos.uno.com"}, nil)
	// Con el CNAME: verificado, y el host lleva al negocio (con www también).
	dns.cname["turnos.uno.com"] = "Dominios.TengoLibre.com.ar."
	a.Esperar(200, P{Metodo: "POST", Ruta: "/v1/panel/dominio/verificar", Quien: ana, Negocio: uno.ID}, &r)
	if r.Dominio.Estado != "verificado" || r.Dominio.VerificadoEn == nil || r.Dominio.Detalle != "" {
		t.Fatalf("verificado: %+v", r.Dominio)
	}
	var h map[string]string
	for _, host := range []string{"turnos.uno.com", "TURNOS.UNO.COM:443", "www.turnos.uno.com"} {
		a.Esperar(200, P{Metodo: "GET", Ruta: "/v1/host/" + host}, &h)
		if h["slug"] != "uno" {
			t.Fatalf("%s: %v", host, h)
		}
	}

	// Otro negocio puede anotar el mismo dominio (nadie lo traba), pero no verificarlo: su token
	// no está, y aunque estuviera, el dominio ya es de uno.
	a.Esperar(200, P{Metodo: "PUT", Ruta: "/v1/panel/dominio", Quien: beto, Negocio: dos.ID, Cuerpo: map[string]string{"dominio": "turnos.uno.com"}}, &r)
	txtDos := txtDe(t, r.Dominio)
	a.Esperar(200, P{Metodo: "POST", Ruta: "/v1/panel/dominio/verificar", Quien: beto, Negocio: dos.ID}, &r)
	if r.Dominio.Estado != "error" {
		t.Fatal("dos verificó con el TXT de uno")
	}
	dns.txt["_tengolibre.turnos.uno.com"] = append(dns.txt["_tengolibre.turnos.uno.com"], txtDos)
	a.Esperar(409, P{Metodo: "POST", Ruta: "/v1/panel/dominio/verificar", Quien: beto, Negocio: dos.ID}, nil)
	// Ni ve el de uno.
	a.Esperar(403, P{Metodo: "GET", Ruta: "/v1/panel/dominio", Quien: beto, Negocio: uno.ID}, nil)

	// Un dominio raíz con A al VPS también vale.
	a.Esperar(200, P{Metodo: "PUT", Ruta: "/v1/panel/dominio", Quien: beto, Negocio: dos.ID, Cuerpo: map[string]string{"dominio": "dos.com.ar"}}, &r)
	if len(r.Dominio.Registros) != 3 || r.Dominio.Registros[2].Tipo != "A" || r.Dominio.Registros[2].Valor != "187.41.102.190" {
		t.Fatalf("raíz: %+v", r.Dominio.Registros)
	}
	dns.txt["_tengolibre.dos.com.ar"] = []string{txtDe(t, r.Dominio)}
	dns.a["dos.com.ar"] = []string{"187.41.102.190"}
	a.Esperar(200, P{Metodo: "POST", Ruta: "/v1/panel/dominio/verificar", Quien: beto, Negocio: dos.ID}, &r)
	if r.Dominio.Estado != "verificado" {
		t.Fatalf("raíz con A: %+v", r.Dominio)
	}

	// Un negocio desactivado deja de responder por su dominio.
	conn, err := pgx.Connect(context.Background(), a.Bases.AdminURL)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close(context.Background())
	if _, err := conn.Exec(context.Background(), "UPDATE negocios SET activo = false WHERE id = $1", dos.ID); err != nil {
		t.Fatal(err)
	}
	a.Esperar(404, P{Metodo: "GET", Ruta: "/v1/host/dos.com.ar"}, nil)

	// Baja.
	a.Esperar(200, P{Metodo: "DELETE", Ruta: "/v1/panel/dominio", Quien: ana, Negocio: uno.ID}, &r)
	if r.Dominio != nil {
		t.Fatal("tendría que quedar sin dominio")
	}
	a.Esperar(404, P{Metodo: "GET", Ruta: "/v1/host/turnos.uno.com"}, nil)
	// Sin dominio no hay qué verificar.
	a.Esperar(404, P{Metodo: "POST", Ruta: "/v1/panel/dominio/verificar", Quien: ana, Negocio: uno.ID}, nil)
}
