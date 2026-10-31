// Package prueba arma la API entera contra una base de prueba y hace pedidos como un front:
// los tests de los dominios la usan para probar de punta a punta.
package prueba

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/db/pruebadb"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/servidor"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/sesion"
)

// API levantada para un test.
type API struct {
	T       testing.TB
	Bases   *pruebadb.Bases
	Sesion  *sesion.Servicio
	Handler http.Handler
	// Ahora es el reloj que comparten los dominios (lo puede mover el test).
	Ahora time.Time
}

// Reloj devuelve una función que lee a.Ahora (para el campo Ahora de los servicios).
func (a *API) Reloj() func() time.Time { return func() time.Time { return a.Ahora } }

// Nueva levanta una base y arma los dominios con dominios(a) (que reciben a.Sesion y la base).
func Nueva(t testing.TB, ahora time.Time, dominios func(a *API) []servidor.Dominio) *API {
	b := pruebadb.Nueva(t)
	a := &API{T: t, Bases: b, Sesion: &sesion.Servicio{Pool: b.App, Verificador: sesion.Falso{}}, Ahora: ahora}
	a.Handler = servidor.Nuevo(servidor.Dependencias{App: b.App, Dominios: dominios(a)})
	return a
}

// Pedido a la API. Quien vacío: sin sesión. Negocio vacío: sin X-Negocio.
type Pedido struct {
	Metodo, Ruta string
	Cuerpo       any
	Quien        string
	Negocio      string
}

// Respuesta de un pedido.
type Respuesta struct {
	Estado int
	Cuerpo []byte
}

// Leer decodifica el cuerpo en v (falla el test si no se puede).
func (r Respuesta) Leer(t testing.TB, v any) {
	t.Helper()
	if err := json.Unmarshal(r.Cuerpo, v); err != nil {
		t.Fatalf("respuesta %d no es JSON (%v): %s", r.Estado, err, r.Cuerpo)
	}
}

// Hacer manda el pedido.
func (a *API) Hacer(p Pedido) Respuesta {
	a.T.Helper()
	var cuerpo io.Reader
	if p.Cuerpo != nil {
		b, err := json.Marshal(p.Cuerpo)
		if err != nil {
			a.T.Fatal(err)
		}
		cuerpo = bytes.NewReader(b)
	}
	req := httptest.NewRequest(p.Metodo, p.Ruta, cuerpo)
	if p.Cuerpo != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if p.Quien != "" {
		req.Header.Set("Authorization", "Bearer dev:"+p.Quien)
	}
	if p.Negocio != "" {
		req.Header.Set("X-Negocio", p.Negocio)
	}
	rec := httptest.NewRecorder()
	a.Handler.ServeHTTP(rec, req)
	return Respuesta{Estado: rec.Code, Cuerpo: rec.Body.Bytes()}
}

// Esperar manda el pedido, exige el estado y decodifica en v (si no es nil).
func (a *API) Esperar(estado int, p Pedido, v any) Respuesta {
	a.T.Helper()
	r := a.Hacer(p)
	if r.Estado != estado {
		a.T.Fatalf("%s %s: %d, quería %d: %s", p.Metodo, p.Ruta, r.Estado, estado, r.Cuerpo)
	}
	if v != nil {
		r.Leer(a.T, v)
	}
	return r
}
