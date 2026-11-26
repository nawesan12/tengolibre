package sugerencias

import (
	"context"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/arranque"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/suscripcion"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/db"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/problema"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/sesion"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/web"
)

// SinChequeo: cerrar una sugerencia y apagarlas andan aunque el panel esté en sólo lectura (es el
// derecho a retiro: nunca se traba).
var SinChequeo = []string{"/v1/panel/sugerencia", "/v1/panel/ajustes/sugerencias"}

// ParaSiempre: el «hasta» de «Ya la uso / No me interesa».
var ParaSiempre = time.Date(9999, 12, 31, 0, 0, 0, 0, time.UTC)

var zona, _ = time.LoadLocation("America/Argentina/Buenos_Aires")

// Servicio de las sugerencias. Activas: SUGERENCIAS_ACTIVAS (apagado, GET responde siempre null).
// Sitio: de donde se sirven las imágenes (el static de la web). Ahora es el reloj (tests).
type Servicio struct {
	Pool    *db.Pool
	Sesion  *sesion.Servicio
	Activas bool
	Sitio   string
	Ahora   func() time.Time
}

func (s *Servicio) ahora() time.Time {
	if s.Ahora != nil {
		return s.Ahora()
	}
	return time.Now()
}

// Rutas: ver la sugerencia la puede pedir cualquiera del equipo (a quien no gestiona le llega null);
// cerrarla, contar el clic y el interruptor, sólo la dueña o la encargada.
func (s *Servicio) Rutas(mux *http.ServeMux) {
	g := func(h web.Manejador) http.HandlerFunc { return s.Sesion.Panel(sesion.Gestion, web.Envolver(h)) }
	mux.HandleFunc("GET /v1/panel/sugerencia", s.Sesion.Panel(sesion.Agenda, web.Envolver(s.leer)))
	mux.HandleFunc("POST /v1/panel/sugerencia/{id}/cerrar", g(s.cerrar))
	mux.HandleFunc("POST /v1/panel/sugerencia/{id}/clic", g(s.clic))
	mux.HandleFunc("GET /v1/panel/ajustes/sugerencias", g(s.leerAjuste))
	mux.HandleFunc("PUT /v1/panel/ajustes/sugerencias", g(s.guardarAjuste))
}

// Leer arma los datos del negocio fijado en tx para las reglas.
func Leer(ctx context.Context, tx pgx.Tx, rol string, ahora time.Time) (Datos, error) {
	d := Datos{Rol: rol, Cerradas: map[string]time.Time{}}
	err := tx.QueryRow(ctx, `SELECT n.rubro, n.creado_en, n.sugerencias_apagadas,
		(SELECT count(*) FROM (SELECT 1 FROM turnos LIMIT 1) t)
		FROM negocios n WHERE n.id = app.negocio_actual()`).Scan(&d.Rubro, &d.CreadoEn, &d.Apagadas, &d.Turnos)
	if err != nil {
		return d, err
	}
	// El alta terminada es la misma guía «Para recibir turnos» del Inicio (todos los pasos obligatorios).
	e, err := arranque.Leer(ctx, tx)
	if err != nil {
		return d, err
	}
	d.AltaCompleta = e.Completo
	filas, err := tx.Query(ctx, "SELECT marca, hasta FROM sugerencias_cerradas")
	if err != nil {
		return d, err
	}
	defer filas.Close()
	for filas.Next() {
		var m string
		var h time.Time
		if err := filas.Scan(&m, &h); err != nil {
			return d, err
		}
		d.Cerradas[m] = h
	}
	if err := filas.Err(); err != nil {
		return d, err
	}
	a, err := suscripcion.LeerAcceso(ctx, tx, ahora)
	d.Bloqueado = a.Bloqueado
	return d, err
}

func (s *Servicio) leer(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	ses := sesion.De(ctx)
	ahora := s.ahora()
	var out *Sugerencia
	if s.Activas && slices.Contains(sesion.Gestion, ses.Rol) {
		var d Datos
		err := s.Pool.EnNegocio(ctx, ses.NegocioID, func(tx pgx.Tx) (err error) {
			d, err = Leer(ctx, tx, ses.Rol, ahora)
			return err
		})
		if err != nil {
			return err
		}
		if out = Elegir(d, ahora); out != nil {
			base := strings.TrimSuffix(s.Sitio, "/")
			out.Imagen, out.Banner = base+out.Imagen, base+out.Banner
			s.contar(ctx, out.ID, ahora, "mostradas")
		}
	}
	web.JSON(w, http.StatusOK, map[string]any{"sugerencia": out})
	return nil
}

// contar suma uno al contador del día (agregado, sin negocio ni persona). Si falla, no importa.
func (s *Servicio) contar(ctx context.Context, id string, ahora time.Time, columna string) {
	if columna != "mostradas" && columna != "clics" {
		return
	}
	dia := ahora.In(zona).Format(time.DateOnly)
	_ = s.Pool.EnTx(ctx, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `INSERT INTO sugerencias_metricas (dia, sugerencia_id, `+columna+`) VALUES ($1, $2, 1)
			ON CONFLICT (dia, sugerencia_id) DO UPDATE SET `+columna+` = sugerencias_metricas.`+columna+` + 1`, dia, id)
		return err
	})
}

func (s *Servicio) cerrar(w http.ResponseWriter, r *http.Request) error {
	sug, ok := DelCatalogo(r.PathValue("id"))
	if !ok {
		return problema.NoEncontrado("Esa sugerencia no existe.")
	}
	var in struct {
		Hasta string `json:"hasta"`
	}
	if err := web.Leer(w, r, &in); err != nil {
		return err
	}
	var hasta time.Time
	switch in.Hasta {
	case "30d":
		hasta = s.ahora().AddDate(0, 0, DiasOculta)
	case "siempre":
		hasta = ParaSiempre
	default:
		return problema.Invalido("Revisá hasta.", problema.Campo{Ruta: "hasta", Mensaje: "tiene que ser 30d o siempre"})
	}
	ctx := r.Context()
	err := s.Pool.EnNegocio(ctx, sesion.De(ctx).NegocioID, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `INSERT INTO sugerencias_cerradas (negocio_id, marca, hasta, creada_en)
			VALUES (app.negocio_actual(), $1, $2, $3)
			ON CONFLICT (negocio_id, marca) DO UPDATE SET hasta = greatest(sugerencias_cerradas.hasta, excluded.hasta), creada_en = excluded.creada_en`,
			sug.Marca, hasta, s.ahora())
		return err
	})
	if err != nil {
		return err
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}

func (s *Servicio) clic(w http.ResponseWriter, r *http.Request) error {
	sug, ok := DelCatalogo(r.PathValue("id"))
	if !ok {
		return problema.NoEncontrado("Esa sugerencia no existe.")
	}
	s.contar(r.Context(), sug.ID, s.ahora(), "clics")
	w.WriteHeader(http.StatusNoContent)
	return nil
}

// Ajuste: el interruptor «Sugerencias».
type Ajuste struct {
	Activas bool `json:"activas"`
}

func (s *Servicio) leerAjuste(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	var apagadas bool
	err := s.Pool.EnNegocio(ctx, sesion.De(ctx).NegocioID, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, "SELECT sugerencias_apagadas FROM negocios WHERE id = app.negocio_actual()").Scan(&apagadas)
	})
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusOK, Ajuste{Activas: !apagadas})
	return nil
}

func (s *Servicio) guardarAjuste(w http.ResponseWriter, r *http.Request) error {
	var in Ajuste
	if err := web.Leer(w, r, &in); err != nil {
		return err
	}
	ctx := r.Context()
	err := s.Pool.EnNegocio(ctx, sesion.De(ctx).NegocioID, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, "UPDATE negocios SET sugerencias_apagadas = $1 WHERE id = app.negocio_actual()", !in.Activas)
		return err
	})
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusOK, in)
	return nil
}
