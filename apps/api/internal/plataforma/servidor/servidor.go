// Package servidor arma el http.Handler de la API: rutas de salud y, más adelante, los dominios.
package servidor

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/db"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/problema"
)

// Dependencias del servidor. App puede ser nil en tests que no tocan la base.
type Dependencias struct {
	App     *db.Pool
	Log     *slog.Logger
	Version string
	// Dominios: cada uno monta sus rutas en el mux.
	Dominios []Dominio
	// Origenes que pueden llamar desde el navegador (CORS). Vacío: ninguno.
	Origenes []string
}

// Dominio es cualquier servicio que sabe montar sus rutas.
type Dominio interface{ Rutas(*http.ServeMux) }

// Nuevo arma el handler con todas las rutas.
func Nuevo(d Dependencias) http.Handler {
	if d.Log == nil {
		d.Log = slog.Default()
	}
	mux := http.NewServeMux()
	// Vivo: el proceso responde. No toca la base, para que un corte de Postgres no reinicie la API.
	mux.HandleFunc("GET /salud/vivo", func(w http.ResponseWriter, _ *http.Request) {
		escribirJSON(w, http.StatusOK, map[string]string{"estado": "vivo"})
	})
	// Listo: puede atender pedidos (la base responde).
	mux.HandleFunc("GET /salud/listo", func(w http.ResponseWriter, r *http.Request) {
		if d.App == nil {
			problema.Escribir(w, problema.Interno())
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if err := d.App.Ping(ctx); err != nil {
			d.Log.Warn("salud: la base no responde", "error", err)
			w.Header().Set("Retry-After", "5")
			problema.Escribir(w, &problema.Problema{Estado: http.StatusServiceUnavailable, Codigo: "sin_base",
				Titulo: "No disponible", Detalle: "La base de datos no responde."})
			return
		}
		escribirJSON(w, http.StatusOK, map[string]string{"estado": "listo"})
	})
	mux.HandleFunc("GET /v1/version", func(w http.ResponseWriter, _ *http.Request) {
		escribirJSON(w, http.StatusOK, map[string]string{"version": d.Version})
	})
	for _, dom := range d.Dominios {
		dom.Rutas(mux)
	}
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		problema.Escribir(w, problema.NoEncontrado("Esa ruta no existe."))
	})
	return recuperar(d.Log, cors(d.Origenes, mux))
}

// recuperar convierte un panic en un 500 con problem+json y lo registra, sin tirar el proceso.
func recuperar(log *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if v := recover(); v != nil {
				if v == http.ErrAbortHandler {
					panic(v)
				}
				log.Error("panic en un handler", "ruta", r.URL.Path, "valor", v)
				problema.Escribir(w, problema.Interno())
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// cors deja llamar desde los orígenes permitidos (el front en otro subdominio) con el token de Clerk.
func cors(origenes []string, next http.Handler) http.Handler {
	permitido := map[string]bool{}
	for _, o := range origenes {
		permitido[o] = true
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if o := r.Header.Get("Origin"); o != "" && permitido[o] {
			h := w.Header()
			h.Set("Access-Control-Allow-Origin", o)
			h.Add("Vary", "Origin")
			h.Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Negocio")
			h.Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE")
			h.Set("Access-Control-Max-Age", "600")
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func escribirJSON(w http.ResponseWriter, estado int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(estado)
	_ = json.NewEncoder(w).Encode(v)
}
