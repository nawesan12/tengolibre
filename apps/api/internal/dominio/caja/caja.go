// Package caja: cobrar en el local y sumar productos (propuestas 07 y 08 del diseño v2). Al marcar un
// turno «Atendido» se abre su caja: lo que resta cobrar (precio del servicio − seña pagada − cupón −
// membresía − descuento de la caja, más los productos que se llevó), el medio de pago (o «Después»:
// queda debiendo) y se registra el cobro. También el catálogo de productos y el resumen de caja del
// día por medio de pago. La plata online (señas) la lleva cobros; acá sólo se lee.
package caja

import (
	"context"
	"errors"
	"net/http"
	"slices"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/disponibilidad"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/negocios"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/auditoria"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/db"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/problema"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/sesion"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/web"
)

// Servicio de la caja. Ahora es el reloj (tests); nil: time.Now.
type Servicio struct {
	Pool   *db.Pool
	Sesion *sesion.Servicio
	Ahora  func() time.Time
}

func (s *Servicio) ahora() time.Time {
	if s.Ahora != nil {
		return s.Ahora()
	}
	return time.Now()
}

// Cajeros: quienes ven el resumen de la caja del día (la plata de todo el negocio).
var Cajeros = []string{sesion.Duenio, sesion.Encargado, sesion.Recepcion}

// Medios de pago en el local. «despues»: queda debiendo.
var Medios = []string{"efectivo", "transferencia", "debito", "credito", "mercadopago", "despues"}

// Rutas del dominio.
func (s *Servicio) Rutas(mux *http.ServeMux) {
	a := func(h web.Manejador) http.HandlerFunc { return s.Sesion.Panel(sesion.Agenda, web.Envolver(h)) }
	g := func(h web.Manejador) http.HandlerFunc { return s.Sesion.Panel(sesion.Gestion, web.Envolver(h)) }
	c := func(h web.Manejador) http.HandlerFunc { return s.Sesion.Panel(Cajeros, web.Envolver(h)) }
	// Catálogo: lo ve quien cobra, lo arma quien administra.
	mux.HandleFunc("GET /v1/panel/productos", a(s.productos))
	mux.HandleFunc("POST /v1/panel/productos", g(s.crearProducto))
	mux.HandleFunc("PATCH /v1/panel/productos/{id}", g(s.editarProducto))
	mux.HandleFunc("DELETE /v1/panel/productos/{id}", g(s.borrarProducto))
	// La caja de un turno.
	mux.HandleFunc("GET /v1/panel/turnos/{id}/caja", a(s.verCaja))
	mux.HandleFunc("POST /v1/panel/turnos/{id}/caja", a(s.cobrar))
	mux.HandleFunc("POST /v1/panel/turnos/{id}/caja/saldar", a(s.saldar))
	mux.HandleFunc("DELETE /v1/panel/turnos/{id}/caja", g(s.anular))
	// El día.
	mux.HandleFunc("GET /v1/panel/caja", c(s.resumen))
}

func (s *Servicio) enNegocio(r *http.Request, fn func(context.Context, pgx.Tx) error) error {
	ctx := r.Context()
	return s.Pool.EnNegocio(ctx, sesion.De(ctx).NegocioID, func(tx pgx.Tx) error { return fn(ctx, tx) })
}

// Linea de la cuenta. Monto negativo: lo que se descuenta.
type Linea struct {
	// Tipo: servicio, sena, cupon, membresia, descuento o producto.
	Tipo       string  `json:"tipo"`
	Concepto   string  `json:"concepto"`
	Monto      int64   `json:"monto"`
	Cantidad   int     `json:"cantidad,omitempty"`
	ProductoID *string `json:"productoId,omitempty"`
}

// Cuenta de la caja de un turno.
type Cuenta struct {
	TurnoID  string `json:"turnoId"`
	Cliente  string `json:"cliente"`
	Servicio string `json:"servicio"`
	// EstadoTurno: la caja sólo se cobra con el turno atendido.
	EstadoTurno string `json:"estadoTurno"`
	// Estado de la caja: sin_cobrar, debe o cobrado.
	Estado    string     `json:"estado"`
	Medio     *string    `json:"medio"`
	CobradoEn *time.Time `json:"cobradoEn"`
	Lineas    []Linea    `json:"lineas"`
	// Total: lo que resta cobrar en el local (o lo que se cobró, si ya está cobrado).
	Total int64 `json:"total"`
	// Medios de pago posibles.
	Medios []string `json:"medios"`
}

// base: los números del turno tal como están hoy (precio, cupón, membresía, seña pagada).
type base struct {
	estado, servicio, cliente string
	precio, cupon, sena       int64
	membresia                 bool
}

func leerBase(ctx context.Context, tx pgx.Tx, turnoID string, bloquear bool) (base, error) {
	var b base
	q := `SELECT t.estado, t.servicio, c.nombre, t.precio, t.membresia_id IS NOT NULL,
		coalesce((SELECT sum(u.descuento) FROM cupon_usos u WHERE u.turno_id = t.id), 0)::bigint,
		coalesce((SELECT sum(p.monto - p.devuelto) FROM pagos p WHERE p.turno_id = t.id AND p.estado = 'aprobado'), 0)::bigint
		FROM turnos t JOIN clientes c ON c.id = t.cliente_id WHERE t.id::text = $1`
	if bloquear {
		q += " FOR UPDATE OF t"
	}
	err := tx.QueryRow(ctx, q, turnoID).Scan(&b.estado, &b.servicio, &b.cliente, &b.precio, &b.membresia, &b.cupon, &b.sena)
	if errors.Is(err, pgx.ErrNoRows) {
		return b, problema.NoEncontrado("No encontramos ese turno.")
	}
	return b, err
}

// numeros de la cuenta: precio de lista del servicio, y lo que lo baja.
type numeros struct {
	servicio, cupon, membresia, sena, descuento, productos int64
}

func (b base) numeros() numeros {
	n := numeros{servicio: b.precio + b.cupon, cupon: b.cupon, sena: b.sena}
	if b.membresia {
		n.membresia = b.precio
	}
	return n
}

// restaServicio: lo que queda del servicio después de cupón, membresía y seña (nunca negativo).
func (n numeros) restaServicio() int64 {
	return max(0, n.servicio-n.cupon-n.membresia-n.sena)
}

func (n numeros) total() int64 { return max(0, n.restaServicio()+n.productos-n.descuento) }

func (n numeros) lineas(servicio string) []Linea {
	ls := []Linea{{Tipo: "servicio", Concepto: servicio, Monto: n.servicio}}
	if n.cupon > 0 {
		ls = append(ls, Linea{Tipo: "cupon", Concepto: "Cupón", Monto: -n.cupon})
	}
	if n.membresia > 0 {
		ls = append(ls, Linea{Tipo: "membresia", Concepto: "Membresía", Monto: -n.membresia})
	}
	if n.sena > 0 {
		ls = append(ls, Linea{Tipo: "sena", Concepto: "Seña pagada (Mercado Pago)", Monto: -n.sena})
	}
	return ls
}

// cuenta del turno: la guardada si ya se cobró (o se debe); si no, la de hoy sin productos.
func cuenta(ctx context.Context, tx pgx.Tx, turnoID string) (Cuenta, error) {
	b, err := leerBase(ctx, tx, turnoID, false)
	if err != nil {
		return Cuenta{}, err
	}
	c := Cuenta{TurnoID: turnoID, Cliente: b.cliente, Servicio: b.servicio, EstadoTurno: b.estado, Estado: "sin_cobrar", Medios: Medios}
	var n numeros
	var cobroID string
	err = tx.QueryRow(ctx, `SELECT id::text, servicio, sena, descuento, descuento_caja, membresia, productos, total, medio, estado, cobrado_en
		FROM caja_cobros WHERE turno_id::text = $1`, turnoID).
		Scan(&cobroID, &n.servicio, &n.sena, &n.cupon, &n.descuento, &n.membresia, &n.productos, &c.Total, &c.Medio, &c.Estado, &c.CobradoEn)
	if errors.Is(err, pgx.ErrNoRows) {
		n = b.numeros()
		c.Lineas, c.Total = n.lineas(b.servicio), n.total()
		return c, nil
	}
	if err != nil {
		return c, err
	}
	c.Lineas = n.lineas(b.servicio)
	filas, err := tx.Query(ctx, "SELECT producto_id::text, nombre, precio, cantidad FROM caja_items WHERE cobro_id = $1 ORDER BY nombre", cobroID)
	if err != nil {
		return c, err
	}
	defer filas.Close()
	for filas.Next() {
		var l Linea
		var precio int64
		if err := filas.Scan(&l.ProductoID, &l.Concepto, &precio, &l.Cantidad); err != nil {
			return c, err
		}
		l.Tipo, l.Monto = "producto", precio*int64(l.Cantidad)
		c.Lineas = append(c.Lineas, l)
	}
	if n.descuento > 0 {
		c.Lineas = append(c.Lineas, Linea{Tipo: "descuento", Concepto: "Descuento", Monto: -n.descuento})
	}
	return c, filas.Err()
}

func (s *Servicio) verCaja(w http.ResponseWriter, r *http.Request) error {
	var c Cuenta
	err := s.enNegocio(r, func(ctx context.Context, tx pgx.Tx) (err error) {
		c, err = cuenta(ctx, tx, r.PathValue("id"))
		return err
	})
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusOK, c)
	return nil
}

// EntradaCobro: lo que se llevó, el descuento de la caja y cómo paga.
type EntradaCobro struct {
	Productos []struct {
		ProductoID string `json:"productoId"`
		Cantidad   int    `json:"cantidad"`
	} `json:"productos"`
	Descuento int64  `json:"descuento"`
	Medio     string `json:"medio"`
}

func (s *Servicio) cobrar(w http.ResponseWriter, r *http.Request) error {
	var in EntradaCobro
	if err := web.Leer(w, r, &in); err != nil {
		return err
	}
	if !slices.Contains(Medios, in.Medio) {
		return problema.Invalido("Elegí cómo paga.", problema.Campo{Ruta: "medio", Mensaje: "inválido"})
	}
	if in.Descuento < 0 {
		return problema.Invalido("El descuento no puede ser negativo.", problema.Campo{Ruta: "descuento", Mensaje: "negativo"})
	}
	if len(in.Productos) > 30 {
		return problema.Invalido("Son muchos productos (hasta 30).", problema.Campo{Ruta: "productos", Mensaje: "demasiados"})
	}
	// Un mismo producto dos veces se suma.
	cantidades := map[string]int{}
	var orden []string
	for _, p := range in.Productos {
		if p.Cantidad < 1 || p.Cantidad > 99 {
			return problema.Invalido("La cantidad va de 1 a 99.", problema.Campo{Ruta: "productos.cantidad", Mensaje: "fuera de rango"})
		}
		if _, ok := cantidades[p.ProductoID]; !ok {
			orden = append(orden, p.ProductoID)
		}
		cantidades[p.ProductoID] += p.Cantidad
	}
	turnoID := r.PathValue("id")
	ahora := s.ahora()
	var c Cuenta
	err := s.enNegocio(r, func(ctx context.Context, tx pgx.Tx) error {
		b, err := leerBase(ctx, tx, turnoID, true)
		if err != nil {
			return err
		}
		if b.estado != "atendido" {
			return problema.Conflicto("no_atendido", "La caja se cobra con el turno atendido.")
		}
		var ya string
		err = tx.QueryRow(ctx, "SELECT estado FROM caja_cobros WHERE turno_id::text = $1", turnoID).Scan(&ya)
		if err == nil {
			if ya == "debe" {
				return problema.Conflicto("debe", "Este turno quedó debiendo: saldalo desde la caja.")
			}
			return problema.Conflicto("ya_cobrado", "Este turno ya está cobrado.")
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		n := b.numeros()
		type item struct {
			p        Producto
			cantidad int
		}
		var items []item
		for _, id := range orden {
			var p Producto
			err := escanearProducto(tx.QueryRow(ctx, "SELECT "+columnasProducto+" FROM productos WHERE id::text = $1 FOR UPDATE", id), &p)
			if errors.Is(err, pgx.ErrNoRows) || (err == nil && !p.Activo) {
				return problema.NoEncontrado("Uno de los productos ya no está a la venta.")
			}
			if err != nil {
				return err
			}
			q := cantidades[id]
			if p.Stock != nil && *p.Stock < q {
				return problema.Conflicto("sin_stock", "No queda stock suficiente de "+p.Nombre+".")
			}
			if p.Stock != nil {
				if _, err := tx.Exec(ctx, "UPDATE productos SET stock = stock - $2 WHERE id = $1", p.ID, q); err != nil {
					return err
				}
			}
			n.productos += p.Precio * int64(q)
			items = append(items, item{p, q})
		}
		if in.Descuento > n.restaServicio()+n.productos {
			return problema.Invalido("El descuento es más que lo que resta cobrar.", problema.Campo{Ruta: "descuento", Mensaje: "mayor al total"})
		}
		n.descuento = in.Descuento
		estado, cobradoEn := "cobrado", &ahora
		if in.Medio == "despues" {
			estado, cobradoEn = "debe", nil
		}
		var cobroID string
		if err := tx.QueryRow(ctx, `INSERT INTO caja_cobros (negocio_id, turno_id, servicio, sena, descuento, descuento_caja, membresia,
			productos, total, medio, estado, cobrado_en, clerk_id) VALUES (app.negocio_actual(), $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
			RETURNING id::text`, turnoID, n.servicio, n.sena, n.cupon, n.descuento, n.membresia, n.productos, n.total(), in.Medio, estado,
			cobradoEn, sesion.De(ctx).ClerkID).Scan(&cobroID); err != nil {
			return err
		}
		for _, it := range items {
			if _, err := tx.Exec(ctx, `INSERT INTO caja_items (negocio_id, cobro_id, producto_id, nombre, precio, cantidad)
				VALUES (app.negocio_actual(), $1, $2, $3, $4, $5)`, cobroID, it.p.ID, it.p.Nombre, it.p.Precio, it.cantidad); err != nil {
				return err
			}
		}
		if err := auditoria.Anotar(ctx, tx, auditoria.Registro{Accion: "caja." + estado, Entidad: "turno", EntidadID: turnoID,
			Detalle: map[string]any{"total": n.total(), "medio": in.Medio, "productos": n.productos}}); err != nil {
			return err
		}
		c, err = cuenta(ctx, tx, turnoID)
		return err
	})
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusCreated, c)
	return nil
}

// saldar lo que quedó debiendo, con el medio con que paga ahora. Entra en la caja del día en que se
// paga.
func (s *Servicio) saldar(w http.ResponseWriter, r *http.Request) error {
	var in struct {
		Medio string `json:"medio"`
	}
	if err := web.Leer(w, r, &in); err != nil {
		return err
	}
	if !slices.Contains(Medios, in.Medio) || in.Medio == "despues" {
		return problema.Invalido("Elegí cómo paga.", problema.Campo{Ruta: "medio", Mensaje: "inválido"})
	}
	turnoID := r.PathValue("id")
	var c Cuenta
	err := s.enNegocio(r, func(ctx context.Context, tx pgx.Tx) error {
		var estado string
		var total int64
		err := tx.QueryRow(ctx, "SELECT estado, total FROM caja_cobros WHERE turno_id::text = $1 FOR UPDATE", turnoID).Scan(&estado, &total)
		if errors.Is(err, pgx.ErrNoRows) {
			return problema.Conflicto("sin_cobrar", "Este turno todavía no pasó por la caja.")
		}
		if err != nil {
			return err
		}
		if estado != "debe" {
			return problema.Conflicto("ya_cobrado", "Este turno ya está cobrado.")
		}
		if _, err := tx.Exec(ctx, `UPDATE caja_cobros SET medio = $2, estado = 'cobrado', cobrado_en = $3, clerk_id = $4, actualizado_en = now()
			WHERE turno_id::text = $1`, turnoID, in.Medio, s.ahora(), sesion.De(ctx).ClerkID); err != nil {
			return err
		}
		if err := auditoria.Anotar(ctx, tx, auditoria.Registro{Accion: "caja.saldado", Entidad: "turno", EntidadID: turnoID,
			Detalle: map[string]any{"total": total, "medio": in.Medio}}); err != nil {
			return err
		}
		c, err = cuenta(ctx, tx, turnoID)
		return err
	})
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusOK, c)
	return nil
}

// anular el cobro (se cargó mal): devuelve el stock y la caja vuelve a sin_cobrar.
func (s *Servicio) anular(w http.ResponseWriter, r *http.Request) error {
	turnoID := r.PathValue("id")
	var c Cuenta
	err := s.enNegocio(r, func(ctx context.Context, tx pgx.Tx) error {
		var cobroID, medio string
		var total int64
		err := tx.QueryRow(ctx, "SELECT id::text, medio, total FROM caja_cobros WHERE turno_id::text = $1 FOR UPDATE", turnoID).
			Scan(&cobroID, &medio, &total)
		if errors.Is(err, pgx.ErrNoRows) {
			return problema.NoEncontrado("Este turno no tiene un cobro para anular.")
		}
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `UPDATE productos p SET stock = p.stock + i.cantidad
			FROM (SELECT producto_id, sum(cantidad)::int AS cantidad FROM caja_items WHERE cobro_id = $1 AND producto_id IS NOT NULL GROUP BY 1) i
			WHERE p.id = i.producto_id AND p.stock IS NOT NULL`, cobroID); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, "DELETE FROM caja_cobros WHERE id = $1", cobroID); err != nil {
			return err
		}
		if err := auditoria.Anotar(ctx, tx, auditoria.Registro{Accion: "caja.anulado", Entidad: "turno", EntidadID: turnoID,
			Detalle: map[string]any{"total": total, "medio": medio}}); err != nil {
			return err
		}
		c, err = cuenta(ctx, tx, turnoID)
		return err
	})
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusOK, c)
	return nil
}

// Movimiento de la caja del día.
type Movimiento struct {
	TurnoID   string    `json:"turnoId"`
	Cliente   string    `json:"cliente"`
	Servicio  string    `json:"servicio"`
	Medio     string    `json:"medio"`
	Total     int64     `json:"total"`
	Productos int64     `json:"productos"`
	CobradoEn time.Time `json:"cobradoEn"`
}

// Resumen de la caja de un día.
type Resumen struct {
	Fecha string `json:"fecha"`
	// PorMedio: lo cobrado en el local, por medio de pago (siempre con todos los medios).
	PorMedio map[string]int64 `json:"porMedio"`
	// Caja: lo cobrado en el local; Senas: las señas online aprobadas ese día (netas de devoluciones);
	// Total: las dos cosas.
	Caja  int64 `json:"caja"`
	Senas int64 `json:"senas"`
	Total int64 `json:"total"`
	// Productos: cuánto de la caja fue de productos.
	Productos int64 `json:"productos"`
	Cobros    int   `json:"cobros"`
	// Debe: lo que quedó debiendo de los turnos de ese día.
	Debe        int64        `json:"debe"`
	Movimientos []Movimiento `json:"movimientos"`
}

// zonaDe del negocio fijado.
func zonaDe(ctx context.Context, tx pgx.Tx) (*time.Location, error) {
	n, err := negocios.Leer(ctx, tx)
	if err != nil {
		return nil, err
	}
	z, err := time.LoadLocation(n.ZonaHoraria)
	if err != nil {
		z = disponibilidad.ZonaPorDefecto
	}
	return z, nil
}

// SenasEntre: las señas online aprobadas en [ini, fin), netas de lo devuelto. La fecha es la del
// pago creado (se aprueba dentro de los 15 minutos que guarda el horario). Lo usa también reportes.
const SenasEntre = `SELECT coalesce(sum(monto - devuelto), 0)::bigint FROM pagos
	WHERE estado = 'aprobado' AND creado_en >= $1 AND creado_en < $2`

func (s *Servicio) resumen(w http.ResponseWriter, r *http.Request) error {
	out := Resumen{PorMedio: map[string]int64{}, Movimientos: []Movimiento{}}
	err := s.enNegocio(r, func(ctx context.Context, tx pgx.Tx) error {
		zona, err := zonaDe(ctx, tx)
		if err != nil {
			return err
		}
		a := s.ahora().In(zona)
		dia := time.Date(a.Year(), a.Month(), a.Day(), 0, 0, 0, 0, zona)
		if v := r.URL.Query().Get("fecha"); v != "" {
			if dia, err = time.ParseInLocation(time.DateOnly, v, zona); err != nil {
				return problema.Invalido("fecha va como AAAA-MM-DD.")
			}
		}
		fin := dia.AddDate(0, 0, 1)
		out.Fecha = dia.Format(time.DateOnly)
		for _, m := range Medios {
			if m != "despues" {
				out.PorMedio[m] = 0
			}
		}
		filas, err := tx.Query(ctx, `SELECT k.turno_id::text, c.nombre, t.servicio, k.medio, k.total, k.productos, k.cobrado_en
			FROM caja_cobros k JOIN turnos t ON t.id = k.turno_id JOIN clientes c ON c.id = t.cliente_id
			WHERE k.estado = 'cobrado' AND k.cobrado_en >= $1 AND k.cobrado_en < $2 ORDER BY k.cobrado_en`, dia, fin)
		if err != nil {
			return err
		}
		movs, err := pgx.CollectRows(filas, pgx.RowToStructByPos[Movimiento])
		if err != nil {
			return err
		}
		for _, m := range movs {
			out.PorMedio[m.Medio] += m.Total
			out.Caja += m.Total
			out.Productos += m.Productos
		}
		out.Movimientos = append(out.Movimientos, movs...)
		out.Cobros = len(movs)
		if err := tx.QueryRow(ctx, SenasEntre, dia, fin).Scan(&out.Senas); err != nil {
			return err
		}
		out.Total = out.Caja + out.Senas
		return tx.QueryRow(ctx, `SELECT coalesce(sum(k.total), 0)::bigint FROM caja_cobros k JOIN turnos t ON t.id = k.turno_id
			WHERE k.estado = 'debe' AND t.inicio >= $1 AND t.inicio < $2`, dia, fin).Scan(&out.Debe)
	})
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusOK, out)
	return nil
}
