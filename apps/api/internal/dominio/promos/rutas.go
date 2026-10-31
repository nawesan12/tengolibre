package promos

import (
	"context"
	"errors"
	"net/http"
	"regexp"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/catalogo"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/problema"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/sesion"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/web"
)

// Rutas del dominio.
func (s *Servicio) Rutas(mux *http.ServeMux) {
	ses := s.Sesion
	leer := func(h web.Manejador) http.HandlerFunc { return ses.Panel(sesion.Gestion, web.Envolver(h)) }
	escribir := func(h web.Manejador) http.HandlerFunc { return ses.Panel(sesion.Gestion, s.escritura(web.Envolver(h))) }
	mux.HandleFunc("GET /v1/panel/cupones", leer(s.cupones))
	mux.HandleFunc("POST /v1/panel/cupones", escribir(s.crearCupon))
	mux.HandleFunc("PATCH /v1/panel/cupones/{id}", escribir(s.editarCupon))
	mux.HandleFunc("DELETE /v1/panel/cupones/{id}", escribir(s.borrarCupon))
	mux.HandleFunc("GET /v1/panel/referidos", leer(s.referidos))
	mux.HandleFunc("GET /v1/panel/referidos/config", leer(s.leerConfig))
	mux.HandleFunc("PUT /v1/panel/referidos/config", escribir(s.guardarConfig))
	// Página pública: validar antes de reservar.
	mux.HandleFunc("POST /v1/p/{slug}/cupones/validar", web.Envolver(s.validarCupon))
	mux.HandleFunc("POST /v1/p/{slug}/referidos/validar", web.Envolver(s.validarReferido))
	// El código de referido de cada cliente sale del link de su turno (el código del link es el secreto).
	mux.HandleFunc("GET /v1/t/{codigo}/referido", web.Envolver(s.miReferido))
	mux.HandleFunc("POST /v1/t/{codigo}/referido", web.Envolver(s.crearReferido))
}

func (s *Servicio) enNegocio(r *http.Request, fn func(context.Context, pgx.Tx) error) error {
	ctx := r.Context()
	return s.Pool.EnNegocio(ctx, sesion.De(ctx).NegocioID, func(tx pgx.Tx) error { return fn(ctx, tx) })
}

func (s *Servicio) cupones(w http.ResponseWriter, r *http.Request) error {
	out := []Cupon{}
	err := s.enNegocio(r, func(ctx context.Context, tx pgx.Tx) error {
		filas, err := tx.Query(ctx, "SELECT "+columnasCupon+" FROM cupones ORDER BY creado_en DESC LIMIT 500")
		if err != nil {
			return err
		}
		defer filas.Close()
		for filas.Next() {
			var c Cupon
			if err := escanearCupon(filas, &c); err != nil {
				return err
			}
			out = append(out, c)
		}
		return filas.Err()
	})
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusOK, map[string]any{"cupones": out})
	return nil
}

var reCodigo = regexp.MustCompile(`^[A-Z0-9_-]{3,40}$`)

// EntradaCupon del panel. En PATCH, lo que no viene no cambia (el código no se edita).
type EntradaCupon struct {
	Codigo  *string    `json:"codigo"`
	Tipo    *string    `json:"tipo"`
	Valor   *int64     `json:"valor"`
	Minimo  *int64     `json:"minimo"`
	MaxUsos *int       `json:"maxUsos"`
	Desde   *time.Time `json:"desde"`
	Hasta   *time.Time `json:"hasta"`
	Activo  *bool      `json:"activo"`
	// Restricciones (franja.go). En PATCH, [] / "" / 0 las sacan.
	Dias           *[]int  `json:"dias"`
	HoraDesde      *string `json:"horaDesde"`
	HoraHasta      *string `json:"horaHasta"`
	AntiguedadDias *int    `json:"antiguedadDias"`
}

func (in EntradaCupon) restricciones(c *Cupon) {
	if in.Dias != nil {
		c.Dias = *in.Dias
	}
	if in.HoraDesde != nil {
		c.HoraDesde = in.HoraDesde
	}
	if in.HoraHasta != nil {
		c.HoraHasta = in.HoraHasta
	}
	if in.AntiguedadDias != nil {
		c.AntiguedadDias = in.AntiguedadDias
	}
}

func validarCupon(c *Cupon) error {
	switch {
	case c.Tipo != "porcentaje" && c.Tipo != "fijo":
		return problema.Invalido("El tipo es porcentaje o fijo.", problema.Campo{Ruta: "tipo", Mensaje: "inválido"})
	case c.Valor <= 0 || (c.Tipo == "porcentaje" && c.Valor > 100):
		return problema.Invalido("El valor tiene que ser mayor a cero (y hasta 100 si es porcentaje).", problema.Campo{Ruta: "valor", Mensaje: "fuera de rango"})
	case c.Minimo != nil && *c.Minimo < 0:
		return problema.Invalido("El mínimo no puede ser negativo.", problema.Campo{Ruta: "minimo", Mensaje: "fuera de rango"})
	case c.MaxUsos != nil && *c.MaxUsos <= 0:
		return problema.Invalido("Los usos tienen que ser al menos uno.", problema.Campo{Ruta: "maxUsos", Mensaje: "fuera de rango"})
	case c.Hasta != nil && !c.Hasta.After(c.Desde):
		return problema.Invalido("El cupón tiene que terminar después de empezar.", problema.Campo{Ruta: "hasta", Mensaje: "fuera de rango"})
	}
	return validarRestricciones(c)
}

func (s *Servicio) crearCupon(w http.ResponseWriter, r *http.Request) error {
	var in EntradaCupon
	if err := web.Leer(w, r, &in); err != nil {
		return err
	}
	c := Cupon{Desde: s.ahora(), Activo: true, Minimo: in.Minimo, MaxUsos: in.MaxUsos, Hasta: in.Hasta}
	if in.Codigo != nil {
		c.Codigo = *in.Codigo
	}
	if in.Tipo != nil {
		c.Tipo = *in.Tipo
	}
	if in.Valor != nil {
		c.Valor = *in.Valor
	}
	if in.Desde != nil {
		c.Desde = *in.Desde
	}
	if in.Activo != nil {
		c.Activo = *in.Activo
	}
	in.restricciones(&c)
	err := s.enNegocio(r, func(ctx context.Context, tx pgx.Tx) error { return Crear(ctx, tx, &c) })
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusCreated, c)
	return nil
}

func (s *Servicio) editarCupon(w http.ResponseWriter, r *http.Request) error {
	var in EntradaCupon
	if err := web.Leer(w, r, &in); err != nil {
		return err
	}
	if in.Codigo != nil {
		return problema.Invalido("El código no se cambia: creá otro cupón.", problema.Campo{Ruta: "codigo", Mensaje: "no editable"})
	}
	var c Cupon
	err := s.enNegocio(r, func(ctx context.Context, tx pgx.Tx) error {
		if err := escanearCupon(tx.QueryRow(ctx, "SELECT "+columnasCupon+" FROM cupones WHERE id = $1 FOR UPDATE", r.PathValue("id")), &c); err != nil {
			return err
		}
		if in.Tipo != nil {
			c.Tipo = *in.Tipo
		}
		if in.Valor != nil {
			c.Valor = *in.Valor
		}
		if in.Minimo != nil {
			c.Minimo = in.Minimo
		}
		if in.MaxUsos != nil {
			c.MaxUsos = in.MaxUsos
		}
		if in.Desde != nil {
			c.Desde = *in.Desde
		}
		if in.Hasta != nil {
			c.Hasta = in.Hasta
		}
		if in.Activo != nil {
			c.Activo = *in.Activo
		}
		in.restricciones(&c)
		if err := validarCupon(&c); err != nil {
			return err
		}
		return escanearCupon(tx.QueryRow(ctx, `UPDATE cupones SET tipo = $2, valor = $3, minimo = $4, max_usos = $5, desde = $6, hasta = $7,
			activo = $8, dias = $9, hora_desde = $10::time, hora_hasta = $11::time, antiguedad_dias = $12 WHERE id = $1 RETURNING `+columnasCupon,
			c.ID, c.Tipo, c.Valor, c.Minimo, c.MaxUsos, c.Desde, c.Hasta, c.Activo, c.Dias, c.HoraDesde, c.HoraHasta, c.AntiguedadDias), &c)
	})
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusOK, c)
	return nil
}

// borrarCupon: uno ya usado se desactiva en vez de borrarse, para no perder los descuentos hechos.
func (s *Servicio) borrarCupon(w http.ResponseWriter, r *http.Request) error {
	err := s.enNegocio(r, func(ctx context.Context, tx pgx.Tx) error {
		var usos int
		if err := tx.QueryRow(ctx, "SELECT usos FROM cupones WHERE id = $1", r.PathValue("id")).Scan(&usos); err != nil {
			return err
		}
		q := "DELETE FROM cupones WHERE id = $1"
		if usos > 0 {
			q = "UPDATE cupones SET activo = false WHERE id = $1"
		}
		_, err := tx.Exec(ctx, q, r.PathValue("id"))
		return err
	})
	if err != nil {
		return err
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}

// Referido como lo lista el panel.
type Referido struct {
	ID       string    `json:"id"`
	Codigo   string    `json:"codigo"`
	Cliente  string    `json:"cliente"`
	Usos     int       `json:"usos"`
	CreadoEn time.Time `json:"creadoEn"`
}

func (s *Servicio) referidos(w http.ResponseWriter, r *http.Request) error {
	out := []Referido{}
	err := s.enNegocio(r, func(ctx context.Context, tx pgx.Tx) error {
		filas, err := tx.Query(ctx, `SELECT r.id::text, r.codigo, c.nombre, (SELECT count(*) FROM referido_usos u WHERE u.referido_id = r.id), r.creado_en
			FROM referidos r JOIN clientes c ON c.id = r.cliente_id ORDER BY r.creado_en DESC LIMIT 500`)
		if err != nil {
			return err
		}
		out, err = pgx.CollectRows(filas, pgx.RowToStructByPos[Referido])
		return err
	})
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusOK, map[string]any{"referidos": out})
	return nil
}

func (s *Servicio) leerConfig(w http.ResponseWriter, r *http.Request) error {
	var a AjustesReferidos
	err := s.enNegocio(r, func(ctx context.Context, tx pgx.Tx) (err error) {
		a, err = leerReferidos(ctx, tx)
		return err
	})
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusOK, a)
	return nil
}

func (s *Servicio) guardarConfig(w http.ResponseWriter, r *http.Request) error {
	var a AjustesReferidos
	if err := web.Leer(w, r, &a); err != nil {
		return err
	}
	if (a.Tipo == nil) != (a.Valor == nil) {
		return problema.Invalido("La recompensa lleva tipo y valor juntos.", problema.Campo{Ruta: "tipo", Mensaje: "incompleto"})
	}
	if a.Tipo != nil {
		c := Cupon{Tipo: *a.Tipo, Valor: *a.Valor}
		if err := validarCupon(&c); err != nil {
			return err
		}
	}
	if a.Tope != nil && *a.Tope <= 0 {
		return problema.Invalido("El tope tiene que ser al menos uno.", problema.Campo{Ruta: "tope", Mensaje: "fuera de rango"})
	}
	err := s.enNegocio(r, func(ctx context.Context, tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `UPDATE ajustes SET referidos_activo = $1, referidos_tipo = $2, referidos_valor = $3, referidos_tope = $4,
			actualizado_en = now() WHERE negocio_id = app.negocio_actual()`, a.Activo, a.Tipo, a.Valor, a.Tope)
		return err
	})
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusOK, a)
	return nil
}

func (s *Servicio) negocioPublico(ctx context.Context, slug string) (string, error) {
	var id *string
	if err := s.Pool.EnTx(ctx, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, "SELECT app.negocio_publico($1)::text", slug).Scan(&id)
	}); err != nil {
		return "", err
	}
	if id == nil {
		return "", problema.NoEncontrado("No encontramos ese negocio.")
	}
	return *id, nil
}

// Validacion de un cupón para la página pública.
type Validacion struct {
	Valido bool   `json:"valido"`
	Motivo string `json:"motivo,omitempty"`
	Codigo string `json:"codigo,omitempty"`
	Tipo   string `json:"tipo,omitempty"`
	Valor  int64  `json:"valor,omitempty"`
	// Descuento sobre el servicio pedido, si vino.
	Descuento int64 `json:"descuento,omitempty"`
	// Condiciones del cupón en palabras (franja, antigüedad). La antigüedad se chequea al reservar.
	Condiciones string `json:"condiciones,omitempty"`
}

func (s *Servicio) validarCupon(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	id, err := s.negocioPublico(ctx, r.PathValue("slug"))
	if err != nil {
		return err
	}
	var in struct {
		Codigo     string `json:"codigo"`
		ServicioID string `json:"servicioId"`
		// Inicio (opcional): el horario elegido, para chequear la franja del cupón.
		Inicio *time.Time `json:"inicio"`
	}
	if err := web.Leer(w, r, &in); err != nil {
		return err
	}
	out := Validacion{Motivo: "Cupón no encontrado"}
	if in.Codigo == "" {
		web.JSON(w, http.StatusOK, out)
		return nil
	}
	err = s.Pool.EnNegocio(ctx, id, func(tx pgx.Tx) error {
		var precio *int64
		if in.ServicioID != "" {
			srv, err := catalogo.LeerServicio(ctx, tx, in.ServicioID)
			if err != nil {
				return problema.NoEncontrado("Ese servicio no existe.")
			}
			precio = &srv.Precio
		}
		c, motivo, err := vigente(ctx, tx, in.Codigo, precio, s.ahora())
		if err != nil {
			return err
		}
		if motivo == "" && in.Inicio != nil && (len(c.Dias) > 0 || c.HoraDesde != nil) {
			z, err := zona(ctx, tx)
			if err != nil {
				return err
			}
			if !EnFranja(c, in.Inicio.In(z)) {
				motivo = "Este cupón vale sólo " + Franja(c.Dias, c.HoraDesde, c.HoraHasta)
			}
		}
		out = Validacion{Valido: motivo == "", Motivo: motivo, Condiciones: c.Condiciones}
		if out.Valido {
			out.Codigo, out.Tipo, out.Valor = c.Codigo, c.Tipo, c.Valor
			if precio != nil {
				out.Descuento = Descuento(*precio, c.Tipo, c.Valor)
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusOK, out)
	return nil
}

func (s *Servicio) validarReferido(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	id, err := s.negocioPublico(ctx, r.PathValue("slug"))
	if err != nil {
		return err
	}
	var in struct {
		Codigo string `json:"codigo"`
	}
	if err := web.Leer(w, r, &in); err != nil {
		return err
	}
	valido := false
	err = s.Pool.EnNegocio(ctx, id, func(tx pgx.Tx) error {
		a, err := leerReferidos(ctx, tx)
		if err != nil || !a.Activo || in.Codigo == "" {
			return err
		}
		return tx.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM referidos WHERE codigo = $1)", Normalizar(in.Codigo)).Scan(&valido)
	})
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusOK, map[string]bool{"valido": valido})
	return nil
}

// delTurno corre fn con el negocio y el cliente del link del turno.
func (s *Servicio) delTurno(ctx context.Context, cod string, fn func(tx pgx.Tx, cliente string) error) error {
	var negocio *string
	if err := s.Pool.EnTx(ctx, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, "SELECT app.negocio_del_turno($1)::text", cod).Scan(&negocio)
	}); err != nil {
		return err
	}
	if negocio == nil {
		return problema.NoEncontrado("No encontramos ese turno.")
	}
	return s.Pool.EnNegocio(ctx, *negocio, func(tx pgx.Tx) error {
		var cliente string
		if err := tx.QueryRow(ctx, "SELECT cliente_id::text FROM turnos WHERE codigo = $1", cod).Scan(&cliente); err != nil {
			return err
		}
		return fn(tx, cliente)
	})
}

// MiReferido: el código de un cliente, cuántas veces se usó y los cupones que le dio.
type MiReferido struct {
	Codigo         *string `json:"codigo"`
	Referidos      int     `json:"referidos"`
	Recompensas    []Cupon `json:"recompensas"`
	ProgramaActivo bool    `json:"programaActivo"`
}

func leerMiReferido(ctx context.Context, tx pgx.Tx, cliente string) (MiReferido, error) {
	out := MiReferido{Recompensas: []Cupon{}}
	a, err := leerReferidos(ctx, tx)
	if err != nil {
		return out, err
	}
	out.ProgramaActivo = a.Activo
	var id string
	err = tx.QueryRow(ctx, `SELECT r.id::text, r.codigo, (SELECT count(*) FROM referido_usos u WHERE u.referido_id = r.id)
		FROM referidos r WHERE r.cliente_id = $1`, cliente).Scan(&id, &out.Codigo, &out.Referidos)
	if errors.Is(err, pgx.ErrNoRows) {
		return out, nil
	}
	if err != nil {
		return out, err
	}
	filas, err := tx.Query(ctx, "SELECT "+columnasCupon+" FROM cupones WHERE referido_id = $1 ORDER BY creado_en DESC", id)
	if err != nil {
		return out, err
	}
	defer filas.Close()
	for filas.Next() {
		var c Cupon
		if err := escanearCupon(filas, &c); err != nil {
			return out, err
		}
		out.Recompensas = append(out.Recompensas, c)
	}
	return out, filas.Err()
}

func (s *Servicio) miReferido(w http.ResponseWriter, r *http.Request) error {
	var out MiReferido
	err := s.delTurno(r.Context(), r.PathValue("codigo"), func(tx pgx.Tx, cliente string) (err error) {
		out, err = leerMiReferido(r.Context(), tx, cliente)
		return err
	})
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusOK, out)
	return nil
}

// crearReferido: el código del cliente (uno por cliente; si ya tiene, el mismo).
func (s *Servicio) crearReferido(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	var out MiReferido
	err := s.delTurno(ctx, r.PathValue("codigo"), func(tx pgx.Tx, cliente string) error {
		a, err := leerReferidos(ctx, tx)
		if err != nil {
			return err
		}
		if !a.Activo {
			return problema.Conflicto("sin_referidos", "Este negocio no tiene programa de referidos.")
		}
		// Un choque de código (raro) se reintenta; el del cliente ya existente lo deja como está.
		for i := 0; i < 10; i++ {
			tag, err := tx.Exec(ctx, `INSERT INTO referidos (negocio_id, cliente_id, codigo) VALUES (app.negocio_actual(), $1, $2)
				ON CONFLICT DO NOTHING`, cliente, codigoReferido())
			if err != nil {
				return err
			}
			var tiene bool
			if err := tx.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM referidos WHERE cliente_id = $1)", cliente).Scan(&tiene); err != nil {
				return err
			}
			if tag.RowsAffected() > 0 || tiene {
				break
			}
		}
		out, err = leerMiReferido(ctx, tx, cliente)
		return err
	})
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusOK, out)
	return nil
}
