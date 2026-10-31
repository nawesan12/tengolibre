package cobros

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/nawesan12/tengolibre/apps/api/internal/integraciones/mercadopago"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/problema"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/sesion"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/web"
)

// Rutas del dominio.
func (s *Servicio) Rutas(mux *http.ServeMux) {
	ses := s.Sesion
	g := func(h web.Manejador) http.HandlerFunc { return ses.Panel(sesion.Gestion, web.Envolver(h)) }
	duenia := func(h web.Manejador) http.HandlerFunc { return ses.Panel([]string{sesion.Duenio}, web.Envolver(h)) }
	// Cuenta de Mercado Pago. Desconectar no pasa por Escritura: con el panel bloqueado, sacar la
	// cuenta tiene que poder hacerse igual.
	mux.HandleFunc("GET /v1/panel/mercadopago", g(s.estado))
	mux.HandleFunc("POST /v1/panel/mercadopago/conectar", ses.Panel([]string{sesion.Duenio}, s.escritura(web.Envolver(s.conectarPanel))))
	mux.HandleFunc("POST /v1/panel/mercadopago/vuelta", duenia(s.vueltaPanel))
	mux.HandleFunc("DELETE /v1/panel/mercadopago", duenia(s.desconectarPanel))
	// Configuración de cobro.
	mux.HandleFunc("GET /v1/panel/cobros", g(s.leerConfigPanel))
	mux.HandleFunc("PUT /v1/panel/cobros", ses.Panel(sesion.Gestion, s.escritura(web.Envolver(s.guardarConfig))))
	// Pagos. Devolver tampoco se bloquea: es plata del cliente.
	mux.HandleFunc("GET /v1/panel/pagos", g(s.listar))
	mux.HandleFunc("POST /v1/panel/pagos/{id}/devolver", g(s.devolver))
	// Página pública y link del turno.
	mux.HandleFunc("GET /v1/p/{slug}/cobro", web.Envolver(s.configPublica))
	mux.HandleFunc("POST /v1/t/{codigo}/pagar", web.Envolver(s.pagar))
	// Webhook: pagos de los negocios y, si están, las suscripciones.
	receptores := map[string]mercadopago.Receptor{"payment": s.recibirPago}
	for k, v := range s.OtrosAvisos {
		receptores[k] = v
	}
	mux.HandleFunc("POST /v1/webhooks/mercadopago", mercadopago.Webhook(s.SecretoWebhook, s.ahora, s.Log, receptores))
}

func (s *Servicio) estado(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	var c Conexion
	err := s.Pool.EnNegocio(ctx, sesion.De(ctx).NegocioID, func(tx pgx.Tx) (err error) {
		c, err = leerConexion(ctx, tx)
		return err
	})
	if err != nil {
		return err
	}
	c.Disponible = s.Activo()
	web.JSON(w, http.StatusOK, c)
	return nil
}

func (s *Servicio) conectarPanel(w http.ResponseWriter, r *http.Request) error {
	ses := sesion.De(r.Context())
	u, err := s.conectar(r.Context(), ses.NegocioID, ses.ClerkID)
	if err != nil {
		return s.traducir(err)
	}
	web.JSON(w, http.StatusOK, map[string]string{"url": u})
	return nil
}

// vueltaPanel: el front recibe ?code&state de Mercado Pago y los manda acá con la sesión.
func (s *Servicio) vueltaPanel(w http.ResponseWriter, r *http.Request) error {
	var in struct {
		Codigo string `json:"codigo"`
		Estado string `json:"estado"`
	}
	if err := web.Leer(w, r, &in); err != nil {
		return err
	}
	ses := sesion.De(r.Context())
	if err := s.traducir(s.volver(r.Context(), ses.NegocioID, ses.ClerkID, in.Codigo, in.Estado)); err != nil {
		return err
	}
	return s.estado(w, r)
}

func (s *Servicio) desconectarPanel(w http.ResponseWriter, r *http.Request) error {
	if err := s.desconectar(r.Context(), sesion.De(r.Context()).NegocioID); err != nil {
		return err
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}

func (s *Servicio) leerConfigPanel(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	var c Config
	err := s.Pool.EnNegocio(ctx, sesion.De(ctx).NegocioID, func(tx pgx.Tx) (err error) {
		c, err = leerConfig(ctx, tx)
		return err
	})
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusOK, c)
	return nil
}

func (s *Servicio) guardarConfig(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	var c Config
	if err := web.Leer(w, r, &c); err != nil {
		return err
	}
	switch c.Modo {
	case "sin_cobro", "total":
	case "porcentaje":
		if c.SenaPorcentaje < 1 || c.SenaPorcentaje > 100 {
			return problema.Invalido("La seña va de 1 a 100 %.", problema.Campo{Ruta: "senaPorcentaje", Mensaje: "fuera de rango"})
		}
	case "fija":
		if c.SenaFija <= 0 {
			return problema.Invalido("Poné el monto de la seña.", problema.Campo{Ruta: "senaFija", Mensaje: "requerido"})
		}
	default:
		return problema.Invalido("El modo es sin_cobro, porcentaje, fija o total.", problema.Campo{Ruta: "modo", Mensaje: "inválido"})
	}
	if c.SenaPorcentaje == 0 {
		c.SenaPorcentaje = 50
	}
	err := s.Pool.EnNegocio(ctx, sesion.De(ctx).NegocioID, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `UPDATE ajustes SET cobro_modo = $1, sena_porcentaje = $2, sena_fija = $3, actualizado_en = now()
			WHERE negocio_id = app.negocio_actual()`, c.Modo, c.SenaPorcentaje, max(c.SenaFija, 0))
		return err
	})
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusOK, c)
	return nil
}

// Pago como lo lista el panel.
type Pago struct {
	ID          string     `json:"id"`
	TurnoID     string     `json:"turnoId"`
	Codigo      string     `json:"codigo"`
	Servicio    string     `json:"servicio"`
	Cliente     string     `json:"cliente"`
	Inicio      time.Time  `json:"inicio"`
	TurnoEstado string     `json:"turnoEstado"`
	Monto       int64      `json:"monto"`
	Modo        string     `json:"modo"`
	Estado      string     `json:"estado"`
	Devuelto    int64      `json:"devuelto"`
	MPPago      *int64     `json:"mpPago"`
	VenceEn     *time.Time `json:"venceEn"`
	// Vencido: nunca se pagó y el horario ya se liberó.
	Vencido  bool      `json:"vencido"`
	CreadoEn time.Time `json:"creadoEn"`
}

func (s *Servicio) listar(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	cond := []string{"true"}
	args := []any{}
	if v := r.URL.Query().Get("estado"); v != "" {
		args = append(args, strings.Split(v, ","))
		cond = append(cond, "p.estado = ANY($1)")
	}
	out := []Pago{}
	err := s.Pool.EnNegocio(ctx, sesion.De(ctx).NegocioID, func(tx pgx.Tx) error {
		filas, err := tx.Query(ctx, `SELECT p.id::text, t.id::text, t.codigo, t.servicio, c.nombre, t.inicio, t.estado, p.monto, p.modo,
				p.estado, p.devuelto, p.mp_pago, t.vence_en, p.creado_en
			FROM pagos p JOIN turnos t ON t.id = p.turno_id JOIN clientes c ON c.id = t.cliente_id
			WHERE `+strings.Join(cond, " AND ")+` ORDER BY p.creado_en DESC LIMIT 200`, args...)
		if err != nil {
			return err
		}
		defer filas.Close()
		for filas.Next() {
			var p Pago
			if err := filas.Scan(&p.ID, &p.TurnoID, &p.Codigo, &p.Servicio, &p.Cliente, &p.Inicio, &p.TurnoEstado, &p.Monto, &p.Modo,
				&p.Estado, &p.Devuelto, &p.MPPago, &p.VenceEn, &p.CreadoEn); err != nil {
				return err
			}
			p.Vencido = p.TurnoEstado == "pendiente_pago" && p.VenceEn != nil && !p.VenceEn.After(s.ahora())
			out = append(out, p)
		}
		return filas.Err()
	})
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusOK, map[string]any{"pagos": out})
	return nil
}

// devolver un pago aprobado, total (sin monto) o en parte. La devolución va por la misma cuenta que
// cobró: sin conexión, no se marca nada (sería decirle a la dueña que se devolvió algo que no).
// Devuelto entero, el turno se cancela (como Jiku).
func (s *Servicio) devolver(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	var in struct {
		Monto int64 `json:"monto"`
	}
	if r.ContentLength != 0 {
		if err := web.Leer(w, r, &in); err != nil {
			return err
		}
	}
	if in.Monto < 0 {
		return problema.Invalido("El monto no puede ser negativo.")
	}
	negocio := sesion.De(ctx).NegocioID
	var turnoID, estado string
	var monto, devuelto int64
	var mpPago *int64
	err := s.Pool.EnNegocio(ctx, negocio, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, "SELECT turno_id::text, estado, monto, devuelto, mp_pago FROM pagos WHERE id = $1", r.PathValue("id")).
			Scan(&turnoID, &estado, &monto, &devuelto, &mpPago)
	})
	if err != nil {
		return err
	}
	if estado != "aprobado" || mpPago == nil {
		return problema.Conflicto("estado", "Sólo se devuelve un pago aprobado.")
	}
	resta := monto - devuelto
	if in.Monto == 0 {
		in.Monto = resta
	}
	if in.Monto > resta {
		return problema.Invalido("Se puede devolver hasta $" + strconv.FormatInt(resta, 10) + ".")
	}
	token, err := s.tokenDe(ctx, negocio)
	if err != nil {
		return s.traducir(err)
	}
	parcial := int64(0)
	if in.Monto < resta || devuelto > 0 {
		parcial = in.Monto
	}
	clave := r.PathValue("id") + ":" + strconv.FormatInt(devuelto, 10) + ":" + strconv.FormatInt(in.Monto, 10)
	if err := s.cliente().Devolver(ctx, token, *mpPago, parcial, clave); err != nil {
		return s.traducir(err)
	}
	total := devuelto+in.Monto >= monto
	var t Pago
	err = s.Pool.EnNegocio(context.WithoutCancel(ctx), negocio, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `UPDATE pagos SET devuelto = least(monto, devuelto + $2), estado = CASE WHEN $3 THEN 'devuelto' ELSE estado END,
			actualizado_en = now() WHERE id = $1`, r.PathValue("id"), in.Monto, total); err != nil {
			return err
		}
		if total {
			if _, err := tx.Exec(ctx, `UPDATE turnos SET estado = 'cancelado', vence_en = NULL, actualizado_en = now()
				WHERE id = $1 AND estado IN ('pendiente_pago', 'pendiente', 'confirmado')`, turnoID); err != nil {
				return err
			}
		}
		return tx.QueryRow(ctx, "SELECT id::text, estado, monto, devuelto FROM pagos WHERE id = $1", r.PathValue("id")).
			Scan(&t.ID, &t.Estado, &t.Monto, &t.Devuelto)
	})
	if err != nil {
		return err
	}
	if total {
		s.avisar(ctx, negocio, turnoID, "cancelado")
	}
	web.JSON(w, http.StatusOK, t)
	return nil
}

// configPublica: lo que la página muestra antes de reservar (payment-config de Jiku). Sin la cuenta
// conectada, el negocio no cobra aunque lo tenga configurado.
func (s *Servicio) configPublica(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	var id *string
	if err := s.Pool.EnTx(ctx, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, "SELECT app.negocio_publico($1)::text", r.PathValue("slug")).Scan(&id)
	}); err != nil {
		return err
	}
	if id == nil {
		return problema.NoEncontrado("No encontramos ese negocio.")
	}
	var c Config
	err := s.Pool.EnNegocio(ctx, *id, func(tx pgx.Tx) error {
		var err error
		if c, err = leerConfig(ctx, tx); err != nil {
			return err
		}
		ok, err := s.puedeCobrar(ctx, tx)
		if !ok {
			c.Modo = "sin_cobro"
		}
		return err
	})
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusOK, c)
	return nil
}

// pagar: el link del turno pide (o vuelve a pedir) el link de pago.
func (s *Servicio) pagar(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	var negocio *string
	if err := s.Pool.EnTx(ctx, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, "SELECT app.negocio_del_turno($1)::text", r.PathValue("codigo")).Scan(&negocio)
	}); err != nil {
		return err
	}
	if negocio == nil {
		return problema.NoEncontrado("No encontramos ese turno.")
	}
	var turnoID string
	var monto int64
	err := s.Pool.EnNegocio(ctx, *negocio, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, "SELECT t.id::text, p.monto FROM turnos t JOIN pagos p ON p.turno_id = t.id WHERE t.codigo = $1",
			r.PathValue("codigo")).Scan(&turnoID, &monto)
	})
	if err == pgx.ErrNoRows {
		return errSinPago
	}
	if err != nil {
		return err
	}
	u, err := s.linkDePago(ctx, *negocio, turnoID)
	if err != nil {
		return s.traducir(err)
	}
	web.JSON(w, http.StatusOK, map[string]any{"url": u, "monto": monto})
	return nil
}
