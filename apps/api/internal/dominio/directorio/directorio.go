// Package directorio: explorar (el directorio público de negocios, directory.ts de Jiku) y las
// sugerencias de horario a partir de lo que un cliente suele reservar (smart-scheduling.ts).
package directorio

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"

	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/miweb"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/turnos"
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

	mu     sync.Mutex
	libres map[string]libreGuardado
}

// libreGuardado: el próximo horario libre de un negocio, calculado hace poco (como mucho el minuto
// que la respuesta queda en caché): un pico de visitas no corre el motor en cada pedido.
type libreGuardado struct {
	inicio *time.Time
	hasta  time.Time
}

const (
	// diasLibre: el próximo libre se busca hoy y mañana, nada más.
	diasLibre = 2
	// maxConLugarHoy: con conLugarHoy=1 el motor corre sobre los candidatos (los que atienden hoy),
	// como mucho éstos; total cuenta entre ellos.
	maxConLugarHoy = 200
	// enParalelo: cuántos negocios se calculan a la vez.
	enParalelo = 4
)

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
	// Promedio de las reseñas visibles (un decimal) y cuántas son; sin reseñas, null y 0.
	Promedio *float64 `json:"promedio"`
	Resenas  int      `json:"resenas"`
	// ProximoLibre: el primer horario que se puede reservar hoy o mañana (null: ninguno).
	ProximoLibre *time.Time `json:"proximoLibre"`
	ZonaHoraria  string     `json:"zonaHoraria"`
}

// fichaCruda: la ficha como la devuelve app.directorio, con el id (no sale en la respuesta).
type fichaCruda struct {
	ID string `json:"id"`
	Ficha
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
	conLugarHoy := q.Get("conLugarHoy") == "1"
	p := Pagina{Pagina: entero(q.Get("pagina"), 1, 1, 1000), Limite: entero(q.Get("limite"), 20, 1, 50)}
	limite, desde := p.Limite, (p.Pagina-1)*p.Limite
	pagDesde := desde
	if conLugarHoy {
		// Lo que filtra es el motor: se traen los candidatos y se pagina acá.
		limite, desde = maxConLugarHoy, 0
	}
	ctx := r.Context()
	ahora := s.ahora()
	var crudo []byte
	err := s.Pool.EnTx(ctx, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, "SELECT app.directorio($1, $2, $3, $4, $5, $6, $7, $8, $9)",
			f[0], f[1], f[2], f[3], f[4], conLugarHoy, ahora, limite, desde).Scan(&crudo)
	})
	if err != nil {
		return err
	}
	var res struct {
		Total    int          `json:"total"`
		Negocios []fichaCruda `json:"negocios"`
	}
	if err := json.Unmarshal(crudo, &res); err != nil {
		return err
	}
	dias := diasLibre
	if conLugarHoy {
		dias = 1 // primero, sólo hoy: es lo que filtra
	}
	s.proximos(ctx, res.Negocios, ahora, dias)
	if conLugarHoy {
		con := res.Negocios[:0]
		for _, n := range res.Negocios {
			if n.ProximoLibre != nil {
				con = append(con, n)
			}
		}
		res.Total = len(con)
		res.Negocios = con[min(pagDesde, len(con)):min(pagDesde+p.Limite, len(con))]
	}
	p.Total = res.Total
	p.Negocios = make([]Ficha, len(res.Negocios))
	for i, n := range res.Negocios {
		p.Negocios[i] = n.Ficha
	}
	p.Paginas = (p.Total + p.Limite - 1) / p.Limite
	// Se cachea poco: lo nuevo aparece enseguida y un pico de visitas no pega siempre en la base.
	w.Header().Set("Cache-Control", "public, max-age=60")
	escribir(w, p)
	return nil
}

// proximos completa ProximoLibre de cada ficha con el motor (turnos.ProximoLibre), con el negocio
// fijado, de a enParalelo. Con dias=1 busca sólo hoy. Si uno falla, queda sin dato: explorar sigue.
func (s *Servicio) proximos(ctx context.Context, fs []fichaCruda, ahora time.Time, dias int) {
	sem := make(chan struct{}, enParalelo)
	var wg sync.WaitGroup
	for i := range fs {
		wg.Add(1)
		sem <- struct{}{}
		go func(f *fichaCruda) {
			defer func() { <-sem; wg.Done() }()
			t, err := s.proximoLibre(ctx, f.ID, ahora, dias)
			if err != nil {
				slog.Warn("directorio: no se pudo calcular el próximo libre", "negocio", f.ID, "error", err)
				return
			}
			f.ProximoLibre = t
		}(&fs[i])
	}
	wg.Wait()
}

func (s *Servicio) proximoLibre(ctx context.Context, negocio string, ahora time.Time, dias int) (*time.Time, error) {
	clave := negocio + ":" + strconv.Itoa(dias)
	s.mu.Lock()
	g, ok := s.libres[clave]
	s.mu.Unlock()
	// Un horario que ya pasó no sirve aunque el guardado no haya vencido.
	if ok && ahora.Before(g.hasta) && (g.inicio == nil || g.inicio.After(ahora)) {
		return g.inicio, nil
	}
	var t *time.Time
	err := s.Pool.EnNegocio(ctx, negocio, func(tx pgx.Tx) (err error) {
		t, err = turnos.ProximoLibre(ctx, tx, ahora, dias)
		return err
	})
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	if s.libres == nil || len(s.libres) > 5000 {
		s.libres = map[string]libreGuardado{}
	}
	s.libres[clave] = libreGuardado{inicio: t, hasta: ahora.Add(time.Minute)}
	s.mu.Unlock()
	return t, nil
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
