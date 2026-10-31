// Package trabajos corre los trabajos periódicos del worker (recordatorios, pedidos de reseña,
// purga…) y deja registro de cada corrida en la tabla corridas (JobRun en Jiku).
//
// El reclamo es el avance de ultima_en: un UPDATE condicional que sólo gana si pasó el intervalo.
// Dos workers a la vez no corren el mismo trabajo, y un worker que muere a mitad de una corrida
// cuesta, como mucho, un intervalo salteado (no hay una marca de "corriendo" que quede trabada).
package trabajos

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/db"
)

// Trabajo periódico. Correr recibe la hora de la pasada (el reloj del planificador) y devuelve un
// resultado que se guarda en la corrida (lo que mandó, lo que salteó).
type Trabajo struct {
	Nombre string
	Cada   time.Duration
	Correr func(ctx context.Context, ahora time.Time) (any, error)
}

// Planificador. Ahora es el reloj (tests); nil: time.Now.
type Planificador struct {
	Pool     *db.Pool
	Trabajos []Trabajo
	Ahora    func() time.Time
	Log      *slog.Logger
}

func (p *Planificador) ahora() time.Time {
	if p.Ahora != nil {
		return p.Ahora()
	}
	return time.Now()
}

func (p *Planificador) log() *slog.Logger {
	if p.Log != nil {
		return p.Log
	}
	return slog.Default()
}

// Correr hace una pasada cada `cada` hasta que se cancela el contexto.
func (p *Planificador) Correr(ctx context.Context, cada time.Duration) {
	t := time.NewTicker(cada)
	defer t.Stop()
	for {
		if _, err := p.Pasada(ctx); err != nil && ctx.Err() == nil {
			p.log().Error("trabajos: pasada", "error", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
	}
}

// Pasada corre los trabajos a los que les toca y devuelve sus nombres. El error de un trabajo queda
// en su corrida y no frena a los demás.
func (p *Planificador) Pasada(ctx context.Context) ([]string, error) {
	var corridos []string
	for _, t := range p.Trabajos {
		ahora := p.ahora()
		toca, err := p.reclamar(ctx, t, ahora)
		if err != nil {
			return corridos, err
		}
		if !toca {
			continue
		}
		res, errT := p.correrUno(ctx, t, ahora)
		if errT != nil {
			p.log().Error("trabajo fallido", "trabajo", t.Nombre, "error", errT)
		}
		if err := p.registrar(ctx, t.Nombre, res, errT); err != nil {
			return corridos, err
		}
		corridos = append(corridos, t.Nombre)
	}
	return corridos, nil
}

// correrUno: un panic cuenta como error del trabajo, no tira el worker.
func (p *Planificador) correrUno(ctx context.Context, t Trabajo, ahora time.Time) (res any, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic: %v", r)
		}
	}()
	return t.Correr(ctx, ahora)
}

func (p *Planificador) reclamar(ctx context.Context, t Trabajo, ahora time.Time) (bool, error) {
	toca := false
	err := p.Pool.EnTx(ctx, func(tx pgx.Tx) error {
		var n string
		err := tx.QueryRow(ctx, `INSERT INTO corridas (nombre, ultima_en, corridas) VALUES ($1, $2, 1)
			ON CONFLICT (nombre) DO UPDATE SET ultima_en = excluded.ultima_en, corridas = corridas.corridas + 1
			WHERE corridas.ultima_en + make_interval(secs => $3) <= excluded.ultima_en
			RETURNING nombre`, t.Nombre, ahora, t.Cada.Seconds()).Scan(&n)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		toca = err == nil
		return err
	})
	return toca, err
}

func (p *Planificador) registrar(ctx context.Context, nombre string, res any, errT error) error {
	var resultado []byte
	if res != nil {
		resultado, _ = json.Marshal(res)
	}
	var msj *string
	if errT != nil {
		m := errT.Error()
		if len(m) > 2000 {
			m = m[:2000]
		}
		msj = &m
	}
	return p.Pool.EnTx(ctx, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `UPDATE corridas SET termino_en = now(), ultimo_error = $2, resultado = $3 WHERE nombre = $1`,
			nombre, msj, resultado)
		return err
	})
}

// Corrida registrada (para tests y diagnóstico).
type Corrida struct {
	Nombre      string
	UltimaEn    time.Time
	UltimoError *string
	Resultado   []byte
	Corridas    int
}

// LeerCorrida devuelve el registro de un trabajo.
func LeerCorrida(ctx context.Context, pool *db.Pool, nombre string) (Corrida, error) {
	var c Corrida
	err := pool.EnTx(ctx, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, "SELECT nombre, ultima_en, ultimo_error, resultado, corridas FROM corridas WHERE nombre = $1", nombre).
			Scan(&c.Nombre, &c.UltimaEn, &c.UltimoError, &c.Resultado, &c.Corridas)
	})
	return c, err
}

// Negocios activos (para recorrerlos).
func Negocios(ctx context.Context, pool *db.Pool) ([]string, error) {
	var out []string
	err := pool.EnTx(ctx, func(tx pgx.Tx) error {
		filas, err := tx.Query(ctx, "SELECT id::text FROM app.negocios_activos() AS id")
		if err != nil {
			return err
		}
		out, err = pgx.CollectRows(filas, pgx.RowTo[string])
		return err
	})
	return out, err
}

// PorNegocio corre fn en cada negocio activo, cada uno en su transacción con el negocio fijado. Un
// negocio que falla no frena a los demás: los errores se juntan.
func PorNegocio(ctx context.Context, pool *db.Pool, fn func(negocioID string, tx pgx.Tx) error) error {
	ns, err := Negocios(ctx, pool)
	if err != nil {
		return err
	}
	var errs []error
	for _, n := range ns {
		if err := pool.EnNegocio(ctx, n, func(tx pgx.Tx) error { return fn(n, tx) }); err != nil {
			errs = append(errs, fmt.Errorf("negocio %s: %w", n, err))
		}
	}
	return errors.Join(errs...)
}
