// Package cobrosprueba arma, para los tests de punta a punta de la plata, la API con todos los
// dominios enganchados como en cmd/api (turnos con sus extras, cobros, promos, membresías,
// suscripción, caja y reportes) contra un Mercado Pago falso. Lo usan los tests de cobros, promos,
// membresias, suscripcion y caja.
package cobrosprueba

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/caja"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/catalogo"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/cobros"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/disponibilidad"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/membresias"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/negocios"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/promos"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/reportes"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/suscripcion"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/turnos"
	"github.com/nawesan12/tengolibre/apps/api/internal/integraciones/mercadopago"
	"github.com/nawesan12/tengolibre/apps/api/internal/integraciones/mercadopago/mpfalso"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/cifrado"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/prueba"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/servidor"
)

// Duenia del negocio de prueba.
const Duenia = "user_duenia0001"

// P es un pedido a la API.
type P = prueba.Pedido

// Entorno de un test.
type Entorno struct {
	*prueba.API
	MP *mpfalso.MP
	// El negocio "el-corte": un servicio Corte de 30' a $8000 y Ana, que atiende los lunes de 9 a 13.
	Negocio, Servicio, Profesional string
	// Avisos que mandaron los dominios (turno → eventos).
	Avisos map[string][]string
	// Cobros, para llamar a lo que no tiene ruta (la renovación de tokens).
	Cobros *cobros.Servicio
}

// En una hora de la zona del negocio ("2026-11-02 09:00").
func En(s string) time.Time {
	t, err := time.ParseInLocation("2006-01-02 15:04", s, disponibilidad.ZonaPorDefecto)
	if err != nil {
		panic(err)
	}
	return t
}

// Armar levanta todo. Ahora: domingo 1/11/2026 a las 10.
func Armar(t *testing.T) *Entorno { return ArmarCon(t, nil) }

// ArmarCon es Armar más los dominios de otro test (p. ej. la lista de espera), que reciben el
// servicio de turnos con sus extras.
func ArmarCon(t *testing.T, mas func(a *prueba.API, tur *turnos.Servicio) []servidor.Dominio) *Entorno {
	e := &Entorno{Avisos: map[string][]string{}}
	e.MP = mpfalso.Nuevo(t)
	clave := make([]byte, 32)
	_, _ = rand.Read(clave)
	cifra, err := cifrado.Nuevo(base64.StdEncoding.EncodeToString(clave))
	if err != nil {
		t.Fatal(err)
	}
	avisar := func(_ context.Context, _, turno, evento string) { e.Avisos[turno] = append(e.Avisos[turno], evento) }
	e.API = prueba.Nueva(t, En("2026-11-01 10:00"), func(a *prueba.API) []servidor.Dominio {
		app, reloj := a.Bases.App, a.Reloj()
		sus := &suscripcion.Servicio{Pool: app, Sesion: a.Sesion, Cliente: e.MP.Cliente(), TokenPlataforma: mpfalso.TokenPlataforma,
			Sitio: "https://tengolibre.com.ar", Ahora: reloj}
		// Como en cmd/api: toda escritura del panel pasa por la suscripción.
		a.Sesion.Escritura, a.Sesion.SinChequeoDeEscritura = sus.Escritura, suscripcion.SinChequeo
		cob := &cobros.Servicio{Pool: app, Sesion: a.Sesion, Cliente: e.MP.Cliente(),
			Aplicacion:     mercadopago.Aplicacion{ID: "123", Secreto: "secreto-app"},
			SecretoWebhook: mpfalso.Secreto, Cifra: cifra, Sitio: "https://tengolibre.com.ar", Escritura: sus.Escritura,
			OtrosAvisos: sus.Receptores(), Avisar: avisar, Ahora: reloj}
		e.Cobros = cob
		pro := &promos.Servicio{Pool: app, Sesion: a.Sesion, Escritura: sus.Escritura, Ahora: reloj}
		mem := &membresias.Servicio{Pool: app, Sesion: a.Sesion, Escritura: sus.Escritura, Ahora: reloj}
		// El orden de los extras importa: la membresía cubre, el cupón descuenta, el cobro cobra lo que queda.
		tur := &turnos.Servicio{Pool: app, Sesion: a.Sesion, Ahora: reloj, Avisar: avisar, Extras: []turnos.Extra{mem, pro, cob}}
		ds := []servidor.Dominio{
			&negocios.Servicio{Pool: app, Sesion: a.Sesion},
			&catalogo.Servicio{Pool: app, Sesion: a.Sesion},
			tur, cob, pro, mem, sus,
			&caja.Servicio{Pool: app, Sesion: a.Sesion, Ahora: reloj},
			&reportes.Servicio{Pool: app, Sesion: a.Sesion, Ahora: reloj},
		}
		if mas != nil {
			ds = append(ds, mas(a, tur)...)
		}
		return ds
	})
	var n negocios.Negocio
	e.Esperar(http.StatusCreated, P{Metodo: "POST", Ruta: "/v1/negocios", Quien: Duenia,
		Cuerpo: map[string]string{"slug": "el-corte", "nombre": "El Corte"}}, &n)
	e.Negocio = n.ID
	var s catalogo.Prestacion
	e.Panel(201, "POST", "/v1/panel/servicios", map[string]any{"nombre": "Corte", "duracion": 30, "precio": 8000}, &s)
	e.Servicio = s.ID
	var p catalogo.Profesional
	e.Panel(201, "POST", "/v1/panel/profesionales", map[string]any{"nombre": "Ana", "servicios": []string{s.ID},
		"horario": []disponibilidad.Franja{{Dia: 1, Desde: "09:00", Hasta: "13:00"}}}, &p)
	e.Profesional = p.ID
	return e
}

// Panel hace un pedido como la dueña y exige el estado.
func (e *Entorno) Panel(estado int, metodo, ruta string, cuerpo, v any) prueba.Respuesta {
	e.T.Helper()
	return e.Esperar(estado, P{Metodo: metodo, Ruta: ruta, Quien: Duenia, Negocio: e.Negocio, Cuerpo: cuerpo}, v)
}

// ConectarMP hace el OAuth entero: conectar, la vuelta con el state de la URL y el canje.
func (e *Entorno) ConectarMP() {
	e.T.Helper()
	var c struct {
		URL string `json:"url"`
	}
	e.Panel(200, "POST", "/v1/panel/mercadopago/conectar", nil, &c)
	u, err := url.Parse(c.URL)
	if err != nil || u.Query().Get("state") == "" || u.Query().Get("code_challenge_method") != "S256" {
		e.T.Fatalf("url de autorización: %q", c.URL)
	}
	var est cobros.Conexion
	e.Panel(200, "POST", "/v1/panel/mercadopago/vuelta", map[string]string{"codigo": "TG-codigo", "estado": u.Query().Get("state")}, &est)
	if !est.Conectada || est.Usuario != e.MP.Usuario {
		e.T.Fatalf("conexión: %+v", est)
	}
}

// Reservar desde la página pública a una hora del lunes 2/11 ("09:00"), con extras opcionales.
func (e *Entorno) Reservar(estado int, hora, telefono string, extra map[string]any) turnos.Turno {
	e.T.Helper()
	cuerpo := map[string]any{"servicioId": e.Servicio, "profesionalId": e.Profesional, "inicio": En("2026-11-02 " + hora).Format(time.RFC3339),
		"cliente": map[string]string{"nombre": "Cliente " + telefono, "telefono": telefono}}
	for k, v := range extra {
		cuerpo[k] = v
	}
	var t turnos.Turno
	var v any
	if estado == http.StatusCreated {
		v = &t
	}
	e.Esperar(estado, P{Metodo: "POST", Ruta: "/v1/p/el-corte/turnos", Cuerpo: cuerpo}, v)
	return t
}

// Avisar manda un aviso firmado de Mercado Pago (del vendedor conectado) y devuelve el estado.
func (e *Entorno) Avisar(tipo, id string) int {
	return mpfalso.Avisar(e.Handler, tipo, id, e.MP.Usuario, e.Ahora)
}

// SQL corre una sentencia como superusuario (para preparar casos, como una prueba vencida).
func (e *Entorno) SQL(q string, args ...any) {
	e.T.Helper()
	ctx := context.Background()
	c, err := pgx.Connect(ctx, e.Bases.AdminURL)
	if err != nil {
		e.T.Fatal(err)
	}
	defer c.Close(ctx)
	if _, err := c.Exec(ctx, q, args...); err != nil {
		e.T.Fatal(err)
	}
}

// Turno del panel por id.
func (e *Entorno) Turno(id string) turnos.Turno {
	e.T.Helper()
	var t turnos.Turno
	e.Panel(200, "GET", "/v1/panel/turnos/"+id, nil, &t)
	return t
}
