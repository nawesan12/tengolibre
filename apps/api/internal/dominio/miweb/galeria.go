package miweb

import (
	"context"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"

	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/problema"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/web"
)

func (s *Servicio) galeria(w http.ResponseWriter, r *http.Request) error {
	var fs []Foto
	err := s.enNegocio(r, func(ctx context.Context, tx pgx.Tx) (err error) {
		fs, err = Galeria(ctx, tx)
		return err
	})
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusOK, map[string]any{"fotos": fs})
	return nil
}

func epigrafe(v string) (string, error) {
	v = strings.TrimSpace(v)
	if utf8.RuneCountInString(v) > 200 {
		return "", campo("epigrafe", "hasta 200 letras")
	}
	return v, nil
}

// sumarFoto agrega una foto al final. La URL es la que devolvió la subida de archivos (o cualquier
// link https a una imagen).
func (s *Servicio) sumarFoto(w http.ResponseWriter, r *http.Request) error {
	var in struct {
		URL      string `json:"url"`
		Epigrafe string `json:"epigrafe"`
	}
	if err := web.Leer(w, r, &in); err != nil {
		return err
	}
	in.URL = strings.TrimSpace(in.URL)
	if in.URL == "" {
		return campo("url", "falta la imagen")
	}
	if err := URLWeb("url", in.URL, 2048); err != nil {
		return err
	}
	var err error
	if in.Epigrafe, err = epigrafe(in.Epigrafe); err != nil {
		return err
	}
	var f Foto
	err = s.enNegocio(r, func(ctx context.Context, tx pgx.Tx) error {
		var n, ultimo int
		if err := tx.QueryRow(ctx, "SELECT count(*), coalesce(max(orden), -1) FROM fotos").Scan(&n, &ultimo); err != nil {
			return err
		}
		if n >= MaxFotos {
			return problema.Conflicto("galeria_llena", "La galería ya tiene el máximo de fotos: sacá alguna para sumar otra.")
		}
		return tx.QueryRow(ctx, `INSERT INTO fotos (negocio_id, url, epigrafe, orden) VALUES (app.negocio_actual(), $1, $2, $3)
			RETURNING id::text, url, epigrafe, orden`, in.URL, in.Epigrafe, ultimo+1).Scan(&f.ID, &f.URL, &f.Epigrafe, &f.Orden)
	})
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusCreated, f)
	return nil
}

func (s *Servicio) editarFoto(w http.ResponseWriter, r *http.Request) error {
	var in struct {
		Epigrafe string `json:"epigrafe"`
	}
	if err := web.Leer(w, r, &in); err != nil {
		return err
	}
	var err error
	if in.Epigrafe, err = epigrafe(in.Epigrafe); err != nil {
		return err
	}
	var f Foto
	err = s.enNegocio(r, func(ctx context.Context, tx pgx.Tx) error {
		return tx.QueryRow(ctx, "UPDATE fotos SET epigrafe = $2 WHERE id = $1 RETURNING id::text, url, epigrafe, orden",
			r.PathValue("id"), in.Epigrafe).Scan(&f.ID, &f.URL, &f.Epigrafe, &f.Orden)
	})
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusOK, f)
	return nil
}

func (s *Servicio) borrarFoto(w http.ResponseWriter, r *http.Request) error {
	err := s.enNegocio(r, func(ctx context.Context, tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, "DELETE FROM fotos WHERE id = $1", r.PathValue("id"))
		if err == nil && tag.RowsAffected() == 0 {
			return pgx.ErrNoRows
		}
		return err
	})
	if err != nil {
		return err
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}

// ordenar recibe todas las fotos en el orden nuevo (arrastrar y soltar): tienen que estar todas y
// una sola vez, así un pedido viejo no deja dos fotos en el mismo lugar.
func (s *Servicio) ordenar(w http.ResponseWriter, r *http.Request) error {
	var in struct {
		IDs []string `json:"ids"`
	}
	if err := web.Leer(w, r, &in); err != nil {
		return err
	}
	var fs []Foto
	err := s.enNegocio(r, func(ctx context.Context, tx pgx.Tx) error {
		actuales, err := Galeria(ctx, tx)
		if err != nil {
			return err
		}
		vistos := map[string]bool{}
		for _, id := range in.IDs {
			vistos[id] = true
		}
		mismas := len(vistos) == len(in.IDs) && len(in.IDs) == len(actuales)
		for _, f := range actuales {
			mismas = mismas && vistos[f.ID]
		}
		if !mismas {
			return problema.Conflicto("galeria_cambio", "La galería cambió mientras la ordenabas: recargá y probá de nuevo.")
		}
		if _, err := tx.Exec(ctx, "UPDATE fotos SET orden = array_position($1::uuid[], id) - 1", in.IDs); err != nil {
			return err
		}
		fs, err = Galeria(ctx, tx)
		return err
	})
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusOK, map[string]any{"fotos": fs})
	return nil
}

func (s *Servicio) publico(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	var id *string
	err := s.Pool.EnTx(ctx, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, "SELECT app.negocio_publico($1)::text", r.PathValue("slug")).Scan(&id)
	})
	if err != nil {
		return err
	}
	if id == nil {
		return problema.NoEncontrado("No hay un negocio con esa dirección.")
	}
	var out struct {
		Web
		Galeria []Foto `json:"galeria"`
	}
	err = s.Pool.EnNegocio(ctx, *id, func(tx pgx.Tx) (err error) {
		if out.Web, err = Leer(ctx, tx); err != nil {
			return err
		}
		out.Galeria, err = Galeria(ctx, tx)
		return err
	})
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusOK, out)
	return nil
}
