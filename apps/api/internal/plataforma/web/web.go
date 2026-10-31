// Package web tiene lo que comparten los handlers: leer y escribir JSON y convertir errores en
// problem+json.
package web

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/problema"
)

// MaxCuerpo de un pedido JSON.
const MaxCuerpo = 1 << 20

// Leer decodifica el cuerpo en v. Un campo desconocido o un JSON roto es un 400.
func Leer(w http.ResponseWriter, r *http.Request, v any) error {
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, MaxCuerpo))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return problema.Invalido("El cuerpo del pedido no es un JSON válido: " + err.Error())
	}
	if dec.Decode(&struct{}{}) != io.EOF {
		return problema.Invalido("El cuerpo del pedido tiene más de un JSON.")
	}
	return nil
}

// JSON escribe v con el estado.
func JSON(w http.ResponseWriter, estado int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(estado)
	_ = json.NewEncoder(w).Encode(v)
}

// Error escribe err: un *problema.Problema tal cual, un "no hay filas" como 404, una violación de
// unicidad como 409 y el resto como 500 (registrado).
func Error(w http.ResponseWriter, r *http.Request, err error) {
	var p *problema.Problema
	var pg *pgconn.PgError
	switch {
	case errors.As(err, &p):
		problema.Escribir(w, p)
	case errors.Is(err, pgx.ErrNoRows):
		problema.Escribir(w, problema.NoEncontrado("No existe o no es de este negocio."))
	case errors.As(err, &pg) && pg.Code == "23505":
		problema.Escribir(w, problema.Conflicto("duplicado", "Ya hay uno igual."))
	case errors.As(err, &pg) && (pg.Code == "23514" || pg.Code == "22P02" || pg.Code == "22007" || pg.Code == "22008"):
		problema.Escribir(w, problema.Invalido("Algún dato está fuera de rango o tiene un formato inválido."))
	case errors.As(err, &pg) && pg.Code == "23503":
		problema.Escribir(w, problema.Conflicto("en_uso", "Hace referencia a algo que no existe, o todavía se usa."))
	default:
		slog.Error("error interno", "ruta", r.URL.Path, "metodo", r.Method, "error", err)
		problema.Escribir(w, problema.Interno())
	}
}

// Manejador es un handler que devuelve error; Envolver lo convierte en http.HandlerFunc.
type Manejador func(w http.ResponseWriter, r *http.Request) error

// Envolver escribe el error del manejador, si lo hubo.
func Envolver(h Manejador) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := h(w, r); err != nil {
			Error(w, r, err)
		}
	}
}
