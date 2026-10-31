// Package db es el único lugar que habla con el pool de Postgres. Todo acceso a datos de un negocio
// pasa por EnNegocio, que abre una transacción y fija el negocio con set_config(..., true): el valor
// vive sólo en esa transacción, así que un pool compartido nunca filtra un negocio a otro.
// Un olvido devuelve vacío (RLS), nunca datos ajenos.
package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Pool es la conexión de un rol.
type Pool struct{ *pgxpool.Pool }

// Abrir conecta con la URL de un rol y verifica que responda.
func Abrir(ctx context.Context, url string) (*Pool, error) {
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, fmt.Errorf("url de la base: %w", err)
	}
	p, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, err
	}
	if err := p.Ping(ctx); err != nil {
		p.Close()
		return nil, fmt.Errorf("la base no responde: %w", err)
	}
	return &Pool{p}, nil
}

// ErrSinNegocio: se pidió un acceso por negocio sin negocio.
var ErrSinNegocio = errors.New("acceso a datos sin negocio")

// EnNegocio corre fn en una transacción con el negocio fijado. Si fn devuelve error, se revierte.
func (p *Pool) EnNegocio(ctx context.Context, negocioID string, fn func(pgx.Tx) error) error {
	if negocioID == "" {
		return ErrSinNegocio
	}
	return p.enTx(ctx, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, "SELECT set_config('app.negocio_id', $1, true)", negocioID); err != nil {
			return err
		}
		return fn(tx)
	})
}

// EnTx corre fn en una transacción sin negocio: sólo para lo que no depende de uno (el alta).
func (p *Pool) EnTx(ctx context.Context, fn func(pgx.Tx) error) error {
	return p.enTx(ctx, fn)
}

func (p *Pool) enTx(ctx context.Context, fn func(pgx.Tx) error) (err error) {
	tx, err := p.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback(context.WithoutCancel(ctx))
		}
	}()
	if err = fn(tx); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
