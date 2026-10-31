// Package directorio: explorar (el directorio público de negocios, directory.ts de Jiku) y las
// sugerencias de horario a partir de lo que un cliente suele reservar (smart-scheduling.ts).
package directorio

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"

	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/miweb"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/db"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/problema"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/sesion"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/web"
)

// Servicio del directorio.
type Servicio struct {
	Pool   *db.Pool
	Sesion *sesion.Servicio
	// Ahora: el reloj (nil: time.Now).
	Ahora func() time.Time
}

func (s *Servicio) ahora() time.Time {
	if s.Ahora == nil {
		return time.Now()
	}
	return s.Ahora()
}

// Ficha de un negocio en el directorio: sólo lo público.
type Ficha struct {
	Slug        string   `json:"slug"`
	Nombre      string   `json:"nombre"`
	Descripcion string   `json:"descripcion"`
	Logo        *string  `json:"logo"`
	Portada     *string  `json:"portada"`
	Ciudad      string   `json:"ciudad"`
	Provincia   string   `json:"provincia"`
	Rubro       string   `json:"rubro"`
	Servicios   []string `json:"servicios"`
	// Desde: el precio más bajo ("desde $X").
	Desde *int64 `json:"desde"`
}

// Pagina de resultados.
type Pagina struct {
	Negocios []Ficha `json:"negocios"`
	Total    int     `json:"total"`
	Pagina   int     `json:"pagina"`
	Limite   int     `json:"limite"`
	Paginas  int     `json:"paginas"`
}

// Ciudad con negocios para mostrar.
type Ciudad struct {
	Ciudad    string `json:"ciudad"`
	Provincia string `json:"provincia"`
	Cantidad  int    `json:"cantidad"`
}

// Rutas del dominio.
func (s *Servicio) Rutas(mux *http.ServeMux) {
	mux.HandleFunc("GET /v1/explorar", web.Envolver(s.explorar))
	mux.HandleFunc("GET /v1/explorar/ciudades", web.Envolver(s.ciudades))
	mux.HandleFunc("GET /v1/explorar/rubros", func(w http.ResponseWriter, _ *http.Request) {
		web.JSON(w, http.StatusOK, map[string]any{"rubros": miweb.Rubros})
	})
	// Sugerencias: desde el link del cliente (el código es el secreto) y desde el panel.
	mux.HandleFunc("GET /v1/t/{codigo}/sugerencias", web.Envolver(s.sugerenciasCliente))
	mux.HandleFunc("GET /v1/panel/clientes/{id}/sugerencias", s.Sesion.Panel(sesion.Agenda, web.Envolver(s.sugerenciasPanel)))
}

func filtro(q string, max int) (string, error) {
	q = strings.TrimSpace(q)
	if utf8.RuneCountInString(q) > max {
		return "", problema.Invalido("La búsqueda es demasiado larga.")
	}
	return q, nil
}

func entero(v string, def, lo, hi int) int {
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return max(lo, min(n, hi))
}

func (s *Servicio) explorar(w http.ResponseWriter, r *http.Request) error {
	q := r.URL.Query()
	var f [5]string
	for i, k := range []string{"ciudad", "provincia", "rubro", "categoria", "q"} {
		v, err := filtro(q.Get(k), 80)
		if err != nil {
			return err
		}
		f[i] = v
	}
	p := Pagina{Pagina: entero(q.Get("pagina"), 1, 1, 1000), Limite: entero(q.Get("limite"), 20, 1, 50)}
	ctx := r.Context()
	var crudo []byte
	err := s.Pool.EnTx(ctx, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, "SELECT app.directorio($1, $2, $3, $4, $5, $6, $7)",
			f[0], f[1], f[2], f[3], f[4], p.Limite, (p.Pagina-1)*p.Limite).Scan(&crudo)
	})
	if err != nil {
		return err
	}
	if err := json.Unmarshal(crudo, &p); err != nil {
		return err
	}
	p.Paginas = (p.Total + p.Limite - 1) / p.Limite
	// Se cachea poco: lo nuevo aparece enseguida y un pico de visitas no pega siempre en la base.
	w.Header().Set("Cache-Control", "public, max-age=60")
	escribir(w, p)
	return nil
}

// escribir es web.JSON sin pisar el Cache-Control.
func escribir(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(v)
}

func (s *Servicio) ciudades(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	out := []Ciudad{}
	err := s.Pool.EnTx(ctx, func(tx pgx.Tx) error {
		filas, err := tx.Query(ctx, "SELECT ciudad, provincia, cantidad FROM app.directorio_ciudades()")
		if err != nil {
			return err
		}
		out, err = pgx.CollectRows(filas, pgx.RowToStructByPos[Ciudad])
		return err
	})
	if err != nil {
		return err
	}
	w.Header().Set("Cache-Control", "public, max-age=300")
	escribir(w, map[string]any{"ciudades": out})
	return nil
}
