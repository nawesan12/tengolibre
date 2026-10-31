package db

import (
	"context"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
)

// Escuchar recibe los NOTIFY de un canal y llama a fn con cada carga, hasta que se cancele ctx.
// Usa una conexión del pool para él solo; si se corta, se reconecta (con espera creciente).
func (p *Pool) Escuchar(ctx context.Context, canal string, fn func(carga string)) {
	espera := time.Second
	for ctx.Err() == nil {
		err := p.escuchar(ctx, canal, fn, func() { espera = time.Second })
		if ctx.Err() != nil {
			return
		}
		slog.Warn("tiempo real: se cortó la escucha, reconecto", "canal", canal, "error", err, "en", espera)
		select {
		case <-ctx.Done():
			return
		case <-time.After(espera):
		}
		espera = min(espera*2, time.Minute)
	}
}

func (p *Pool) escuchar(ctx context.Context, canal string, fn func(string), conectado func()) error {
	pc, err := p.Acquire(ctx)
	if err != nil {
		return err
	}
	// La conexión queda en LISTEN: se saca del pool y se cierra al terminar.
	c := pc.Hijack()
	defer c.Close(context.WithoutCancel(ctx))
	if _, err := c.Exec(ctx, "LISTEN "+pgx.Identifier{canal}.Sanitize()); err != nil {
		return err
	}
	conectado()
	for {
		n, err := c.WaitForNotification(ctx)
		if err != nil {
			return err
		}
		fn(n.Payload)
	}
}
