package cuenta

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/db"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/trabajos"
)

// DiasNegocioCerrado: los días entre el cierre de un negocio y el borrado de sus datos (el negocio y
// sus clientes). Los cobros quedan, sin datos de nadie, en auth.cobros_archivados (00027).
const DiasNegocioCerrado = 90

// NegocioParaPurgar: un negocio cerrado que la purga borraría. Retenido: tiene una suscripción que
// todavía cobra (autorizada o pausada) y la purga lo saltea hasta que se dé de baja.
type NegocioParaPurgar struct {
	ID        string
	Slug      string
	Nombre    string
	CerradoEn time.Time
	Clientes  int64
	Turnos    int64
	Cobros    int64
	Retenido  bool
}

// SQLParaPurgar lista los negocios cerrados antes de $1. Lo usa también tengolibrectl (como el
// migrador): la función no borra nada.
const SQLParaPurgar = `SELECT id::text, slug, nombre, cerrado_en, clientes, turnos, cobros, retenido FROM app.negocios_para_purgar($1)`

// Consultor: lo que tienen pgx.Tx y pgx.Conn para leer filas.
type Consultor interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

// ParaPurgar devuelve los negocios que la purga borraría a la hora ahora.
func ParaPurgar(ctx context.Context, q Consultor, ahora time.Time) ([]NegocioParaPurgar, error) {
	filas, err := q.Query(ctx, SQLParaPurgar, ahora.AddDate(0, 0, -DiasNegocioCerrado))
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(filas, pgx.RowToStructByPos[NegocioParaPurgar])
}

// PurgarNegocios borra, de a tandas, los negocios cerrados hace más de DiasNegocioCerrado (los no
// retenidos) y, si carpeta no está vacía, sus imágenes (<carpeta>/<negocio>). Devuelve los ids.
func PurgarNegocios(ctx context.Context, pool *db.Pool, ahora time.Time, carpeta string) ([]string, error) {
	var todos []string
	for {
		var ids []string
		err := pool.EnTx(ctx, func(tx pgx.Tx) error {
			return tx.QueryRow(ctx, "SELECT app.purgar_negocios($1, 10)::text[]", ahora.AddDate(0, 0, -DiasNegocioCerrado)).Scan(&ids)
		})
		if err != nil {
			return todos, err
		}
		for _, id := range ids {
			if carpeta == "" {
				continue
			}
			// Las imágenes no están en la base: si no se pueden borrar, se anota y la purga sigue.
			if err := os.RemoveAll(filepath.Join(carpeta, filepath.Base(id))); err != nil {
				slog.Warn("purga de negocios: no se borraron las imágenes", "negocio", id, "error", err)
			}
		}
		todos = append(todos, ids...)
		if len(ids) < 10 {
			return todos, nil
		}
	}
}

// TrabajoNegocios es la purga de negocios cerrados para el worker: una vez por día. Va detrás de
// TENGOLIBRE_PURGA_NEGOCIOS=1 (apagada por defecto): el worker sólo la suma si está prendida.
func TrabajoNegocios(pool *db.Pool, carpeta string) trabajos.Trabajo {
	return trabajos.Trabajo{Nombre: "purga_negocios", Cada: 24 * time.Hour, Correr: func(ctx context.Context, ahora time.Time) (any, error) {
		ids, err := PurgarNegocios(ctx, pool, ahora, carpeta)
		return map[string]any{"purgados": len(ids), "negocios": ids}, err
	}}
}
