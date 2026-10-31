package clientes

import (
	"context"
	"net/http"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"

	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/problema"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/sesion"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/web"
)

// Nota interna sobre un cliente: la ve el equipo, nunca el cliente.
type Nota struct {
	ID        string    `json:"id"`
	Texto     string    `json:"texto"`
	MiembroID *string   `json:"miembroId"`
	Autor     string    `json:"autor"`
	CreadoEn  time.Time `json:"creadoEn"`
}

// miembroDeLaSesion: el id y el nombre (o el email, si no tiene) de quien llama, en el negocio fijado.
func miembroDeLaSesion(ctx context.Context, tx pgx.Tx) (id, nombre string, err error) {
	err = tx.QueryRow(ctx, "SELECT id::text, CASE WHEN nombre <> '' THEN nombre ELSE email END FROM miembros WHERE clerk_id = $1",
		sesion.De(ctx).ClerkID).Scan(&id, &nombre)
	return id, nombre, err
}

func (s *Servicio) notas(w http.ResponseWriter, r *http.Request) error {
	var out []Nota
	err := s.enNegocio(r, func(ctx context.Context, tx pgx.Tx) error {
		if err := existe(ctx, tx, r.PathValue("id")); err != nil {
			return err
		}
		filas, err := tx.Query(ctx, `SELECT id::text, texto, miembro_id::text, autor, creado_en FROM notas_cliente
			WHERE cliente_id = $1 ORDER BY creado_en DESC, id`, r.PathValue("id"))
		if err != nil {
			return err
		}
		out, err = pgx.CollectRows(filas, pgx.RowToStructByPos[Nota])
		return err
	})
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusOK, map[string]any{"notas": out})
	return nil
}

func (s *Servicio) crearNota(w http.ResponseWriter, r *http.Request) error {
	var in struct {
		Texto string `json:"texto"`
	}
	if err := web.Leer(w, r, &in); err != nil {
		return err
	}
	in.Texto = strings.TrimSpace(in.Texto)
	if n := utf8.RuneCountInString(in.Texto); n < 1 || n > 2000 {
		return problema.Invalido("La nota va de 1 a 2000 letras.", problema.Campo{Ruta: "texto", Mensaje: "largo inválido"})
	}
	var n Nota
	err := s.enNegocio(r, func(ctx context.Context, tx pgx.Tx) error {
		if err := existe(ctx, tx, r.PathValue("id")); err != nil {
			return err
		}
		miembro, autor, err := miembroDeLaSesion(ctx, tx)
		if err != nil {
			return err
		}
		return tx.QueryRow(ctx, `INSERT INTO notas_cliente (negocio_id, cliente_id, miembro_id, autor, texto)
			VALUES (app.negocio_actual(), $1, $2, $3, $4) RETURNING id::text, texto, miembro_id::text, autor, creado_en`,
			r.PathValue("id"), miembro, autor, in.Texto).Scan(&n.ID, &n.Texto, &n.MiembroID, &n.Autor, &n.CreadoEn)
	})
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusCreated, n)
	return nil
}

// borrarNota: la borra quien la escribió, o alguien de gestión.
func (s *Servicio) borrarNota(w http.ResponseWriter, r *http.Request) error {
	err := s.enNegocio(r, func(ctx context.Context, tx pgx.Tx) error {
		var autor *string
		err := tx.QueryRow(ctx, "SELECT miembro_id::text FROM notas_cliente WHERE id = $1 AND cliente_id = $2",
			r.PathValue("nota"), r.PathValue("id")).Scan(&autor)
		if err == pgx.ErrNoRows {
			return problema.NoEncontrado("Esa nota no existe.")
		}
		if err != nil {
			return err
		}
		if !slices.Contains(sesion.Gestion, sesion.De(ctx).Rol) {
			yo, _, err := miembroDeLaSesion(ctx, tx)
			if err != nil {
				return err
			}
			if autor == nil || *autor != yo {
				return problema.SinPermiso("Sólo podés borrar tus propias notas.")
			}
		}
		_, err = tx.Exec(ctx, "DELETE FROM notas_cliente WHERE id = $1", r.PathValue("nota"))
		return err
	})
	if err != nil {
		return err
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}
