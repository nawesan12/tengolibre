// Package cobros: el negocio cobra la reserva con su propia cuenta de Mercado Pago. La conexión por
// OAuth (tokens cifrados, renovación), la configuración (sin cobro, seña por porcentaje, seña fija o
// total), el cobro de la reserva pública con Checkout Pro, el webhook firmado que lo concilia, el
// listado de pagos del panel y las devoluciones.
//
// La reserva con cobro queda pendiente_pago con vence_en: el motor de disponibilidad ya deja de
// contarla al vencer, así que no hace falta un cron que libere horarios.
package cobros

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/nawesan12/tengolibre/apps/api/internal/integraciones/mercadopago"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/cifrado"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/db"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/problema"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/sesion"
)

// VencePago: cuánto tiempo guarda el horario una reserva sin pagar (PENDING_PAYMENT_TTL_MINUTES).
const VencePago = 15 * time.Minute

// RutaVuelta del OAuth en el front: tiene que coincidir con la configurada en la aplicación.
const RutaVuelta = "/panel/mercadopago/vuelta"

// Errores que el panel y el link del turno traducen.
var (
	ErrApagado     = errors.New("cobros: Mercado Pago no está configurado")
	ErrSinConexion = errors.New("cobros: el negocio no conectó Mercado Pago")
	ErrReautorizar = errors.New("cobros: hay que volver a conectar Mercado Pago")
	ErrEstado      = errors.New("cobros: state inválido o vencido")
)

// Servicio de cobros.
type Servicio struct {
	Pool   *db.Pool
	Sesion *sesion.Servicio
	// Cliente de Mercado Pago (nil: el de verdad).
	Cliente *mercadopago.Cliente
	// Aplicacion: el OAuth de TengoLibre. Vuelta vacía: Sitio + RutaVuelta.
	Aplicacion mercadopago.Aplicacion
	// SecretoWebhook: la clave secreta de los webhooks (Tus integraciones › Webhooks).
	SecretoWebhook string
	// Cifra los tokens. Sin cifrador, los negocios no pueden conectar su cuenta.
	Cifra *cifrado.Cifrador
	// Sitio del front: vueltas del checkout y del OAuth.
	Sitio string
	// Escritura: el chequeo de sólo lectura de la suscripción (suscripcion.Servicio.Escritura). nil: no
	// se chequea.
	Escritura func(http.HandlerFunc) http.HandlerFunc
	// OtrosAvisos: receptores del webhook que no son pagos (las suscripciones). El webhook es uno solo.
	OtrosAvisos map[string]mercadopago.Receptor
	// Avisar como en turnos (el pago confirma el turno). nil: no se avisa.
	Avisar func(ctx context.Context, negocioID, turnoID, evento string)
	Ahora  func() time.Time
	Log    *slog.Logger
}

func (s *Servicio) ahora() time.Time {
	if s.Ahora != nil {
		return s.Ahora()
	}
	return time.Now()
}

func (s *Servicio) log() *slog.Logger {
	if s.Log != nil {
		return s.Log
	}
	return slog.Default()
}

func (s *Servicio) cliente() *mercadopago.Cliente {
	if s.Cliente != nil {
		return s.Cliente
	}
	return &mercadopago.Cliente{}
}

func (s *Servicio) aplicacion() mercadopago.Aplicacion {
	a := s.Aplicacion
	if a.Vuelta == "" {
		a.Vuelta = s.Sitio + RutaVuelta
	}
	return a
}

func (s *Servicio) avisar(ctx context.Context, negocioID, turnoID, evento string) {
	if s.Avisar != nil {
		s.Avisar(context.WithoutCancel(ctx), negocioID, turnoID, evento)
	}
}

// Activo: la aplicación tiene credenciales para conectar negocios.
func (s *Servicio) Activo() bool {
	return s != nil && s.Aplicacion.ID != "" && s.Aplicacion.Secreto != "" && s.Cifra != nil
}

func (s *Servicio) escritura(h http.HandlerFunc) http.HandlerFunc {
	if s.Escritura == nil {
		return h
	}
	return s.Escritura(h)
}

// traducir los errores de Mercado Pago para el panel y el link del turno. Nunca devuelve lo que
// respondió Mercado Pago.
func (s *Servicio) traducir(err error) error {
	var emp *mercadopago.ErrorMP
	switch {
	case err == nil:
		return nil
	case errors.Is(err, ErrApagado), errors.Is(err, ErrSinConexion):
		return problema.Conflicto("sin_cobros", "El negocio todavía no conectó Mercado Pago.")
	case errors.Is(err, ErrReautorizar):
		return problema.Conflicto("reconectar", "Hay que volver a conectar Mercado Pago.")
	case errors.Is(err, ErrEstado):
		return problema.Invalido("La conexión venció. Probá de nuevo.")
	case errors.As(err, &emp):
		s.log().Warn("mercadopago: la API respondió con error", "estado", emp.Estado, "cuerpo", mercadopago.Recortar(emp.Cuerpo, 300))
		return &problema.Problema{Estado: http.StatusBadGateway, Codigo: "mercadopago", Titulo: "Mercado Pago no respondió",
			Detalle: "Probá de nuevo en un rato."}
	}
	return err
}

// Conexion es lo que ve el panel de la cuenta de Mercado Pago.
type Conexion struct {
	// Disponible: TengoLibre tiene la aplicación configurada.
	Disponible bool  `json:"disponible"`
	Conectada  bool  `json:"conectada"`
	Usuario    int64 `json:"usuario,omitempty"`
	// Cuenta: el nombre de la cuenta conectada (razón social, marca o persona), si Mercado Pago lo dio.
	Cuenta      string     `json:"cuenta,omitempty"`
	Desde       *time.Time `json:"desde,omitempty"`
	VenceEn     *time.Time `json:"venceEn,omitempty"`
	Reautorizar bool       `json:"reautorizar"`
	UltimoError string     `json:"ultimoError,omitempty"`
}

// leerConexion del negocio fijado.
func leerConexion(ctx context.Context, tx pgx.Tx) (Conexion, error) {
	var c Conexion
	var desde, vence time.Time
	err := tx.QueryRow(ctx, `SELECT mp_usuario, cuenta, conectada_en, vence_en, reautorizar, ultimo_error FROM mp_conexiones
		WHERE negocio_id = app.negocio_actual()`).Scan(&c.Usuario, &c.Cuenta, &desde, &vence, &c.Reautorizar, &c.UltimoError)
	if errors.Is(err, pgx.ErrNoRows) {
		return c, nil
	}
	if err != nil {
		return c, err
	}
	c.Conectada, c.Desde, c.VenceEn = true, &desde, &vence
	return c, nil
}

// conectar guarda un state de un solo uso (atado al negocio y a la persona) y devuelve la URL de
// autorización de Mercado Pago.
func (s *Servicio) conectar(ctx context.Context, negocio, clerkID string) (string, error) {
	if !s.Activo() {
		return "", ErrApagado
	}
	estado, verificador := mercadopago.Azar(), mercadopago.Azar()+mercadopago.Azar()
	err := s.Pool.EnNegocio(ctx, negocio, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, "DELETE FROM mp_estados WHERE vence_en < now()"); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `INSERT INTO mp_estados (estado_hash, negocio_id, clerk_id, verificador, vence_en)
			VALUES ($1, app.negocio_actual(), $2, $3, now() + interval '10 minutes')`, mercadopago.Hash(estado), clerkID, verificador)
		return err
	})
	if err != nil {
		return "", err
	}
	return s.cliente().URLAutorizacion(s.aplicacion(), estado, verificador), nil
}

// volver consume el state, canjea el código y guarda los tokens cifrados.
func (s *Servicio) volver(ctx context.Context, negocio, clerkID, codigo, estado string) error {
	if !s.Activo() {
		return ErrApagado
	}
	if codigo == "" || estado == "" {
		return ErrEstado
	}
	var verificador string
	err := s.Pool.EnNegocio(ctx, negocio, func(tx pgx.Tx) error {
		err := tx.QueryRow(ctx, `UPDATE mp_estados SET usado_en = now()
			WHERE estado_hash = $1 AND clerk_id = $2 AND usado_en IS NULL AND vence_en > now()
			RETURNING verificador`, mercadopago.Hash(estado), clerkID).Scan(&verificador)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrEstado
		}
		return err
	})
	if err != nil {
		return err
	}
	tk, err := s.cliente().Canjear(ctx, s.aplicacion(), codigo, verificador)
	if err != nil {
		return err
	}
	if err := s.guardarTokens(ctx, negocio, tk, true); err != nil {
		return err
	}
	// El nombre de la cuenta, para el panel. Si Mercado Pago no lo da, queda el número de usuario.
	u, err := s.cliente().LeerCuenta(ctx, tk.AccessToken)
	if err != nil {
		s.log().Warn("cobros: no se pudo leer la cuenta de Mercado Pago", "negocio", negocio, "error", err)
		return nil
	}
	return s.Pool.EnNegocio(ctx, negocio, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, "UPDATE mp_conexiones SET cuenta = $1 WHERE negocio_id = app.negocio_actual()", u.Nombre())
		return err
	})
}

func (s *Servicio) guardarTokens(ctx context.Context, negocio string, tk mercadopago.Tokens, nueva bool) error {
	acceso, err := s.Cifra.Cifrar("mp:"+negocio, []byte(tk.AccessToken))
	if err != nil {
		return err
	}
	renov, err := s.Cifra.Cifrar("mp:"+negocio, []byte(tk.RefreshToken))
	if err != nil {
		return err
	}
	vence := s.ahora().Add(time.Duration(tk.ExpiresIn) * time.Second)
	return s.Pool.EnNegocio(ctx, negocio, func(tx pgx.Tx) error {
		if nueva {
			// Otra cuenta de Mercado Pago reemplaza a la anterior.
			_, err := tx.Exec(ctx, `INSERT INTO mp_conexiones (negocio_id, mp_usuario, acceso, renovacion, vence_en, alcances)
				VALUES (app.negocio_actual(), $1, $2, $3, $4, $5)
				ON CONFLICT (negocio_id) DO UPDATE SET mp_usuario = $1, acceso = $2, renovacion = $3, vence_en = $4, alcances = $5,
					conectada_en = now(), renovada_en = NULL, reautorizar = false, ultimo_error = '', cuenta = ''`,
				tk.UserID, acceso, renov, vence, tk.Scope)
			return err
		}
		// Un refresh puede rotar el refresh token: se reemplazan los dos juntos.
		_, err := tx.Exec(ctx, `UPDATE mp_conexiones SET acceso = $1, renovacion = $2, vence_en = $3, renovada_en = $4,
			reautorizar = false, ultimo_error = '' WHERE negocio_id = app.negocio_actual()`, acceso, renov, vence, s.ahora())
		return err
	})
}

// desconectar borra los tokens. (La autorización se revoca desde la cuenta de Mercado Pago.)
func (s *Servicio) desconectar(ctx context.Context, negocio string) error {
	return s.Pool.EnNegocio(ctx, negocio, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, "DELETE FROM mp_conexiones WHERE negocio_id = app.negocio_actual()")
		return err
	})
}

// tokenDe devuelve el token del negocio, renovándolo si vence en menos de una semana. Si la
// renovación falla, la conexión queda para reautorizar: nunca se usa otro token.
func (s *Servicio) tokenDe(ctx context.Context, negocio string) (string, error) {
	if !s.Activo() {
		return "", ErrApagado
	}
	var acceso, renov []byte
	var vence time.Time
	var reaut bool
	err := s.Pool.EnNegocio(ctx, negocio, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, "SELECT acceso, renovacion, vence_en, reautorizar FROM mp_conexiones WHERE negocio_id = app.negocio_actual()").
			Scan(&acceso, &renov, &vence, &reaut)
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrSinConexion
	}
	if err != nil {
		return "", err
	}
	if reaut {
		return "", ErrReautorizar
	}
	if vence.After(s.ahora().Add(7 * 24 * time.Hour)) {
		tk, err := s.Cifra.Descifrar("mp:"+negocio, acceso)
		return string(tk), err
	}
	return s.renovar(ctx, negocio, renov)
}

func (s *Servicio) renovar(ctx context.Context, negocio string, renovCifrado []byte) (string, error) {
	r, err := s.Cifra.Descifrar("mp:"+negocio, renovCifrado)
	if err != nil {
		return "", err
	}
	tk, err := s.cliente().Renovar(ctx, s.aplicacion(), string(r))
	if err != nil {
		s.log().Warn("mercadopago: no se pudo renovar el token", "negocio", negocio, "error", err)
		_ = s.Pool.EnNegocio(context.WithoutCancel(ctx), negocio, func(tx pgx.Tx) error {
			_, err := tx.Exec(ctx, `UPDATE mp_conexiones SET reautorizar = true, ultimo_error = $1 WHERE negocio_id = app.negocio_actual()`,
				"La renovación del permiso falló: volvé a conectar la cuenta.")
			return err
		})
		return "", ErrReautorizar
	}
	if err := s.guardarTokens(ctx, negocio, tk, false); err != nil {
		return "", err
	}
	return tk.AccessToken, nil
}

// RenovarPorVencer renueva los tokens que vencen en menos de 30 días (mercadopago-renewal.ts): sin
// esto, cada negocio dejaría de poder cobrar a los seis meses de conectar, sin aviso. Hasta 20 por
// pasada. Lo llama RenovarCada.
func (s *Servicio) RenovarPorVencer(ctx context.Context) (renovadas, fallidas int, err error) {
	if !s.Activo() {
		return 0, 0, nil
	}
	var ids []string
	err = s.Pool.EnTx(ctx, func(tx pgx.Tx) error {
		filas, err := tx.Query(ctx, "SELECT id::text FROM app.mp_por_vencer($1, 20) AS id", s.ahora().Add(30*24*time.Hour))
		if err != nil {
			return err
		}
		ids, err = pgx.CollectRows(filas, pgx.RowTo[string])
		return err
	})
	if err != nil {
		return 0, 0, err
	}
	for _, n := range ids {
		var renov []byte
		err := s.Pool.EnNegocio(ctx, n, func(tx pgx.Tx) error {
			return tx.QueryRow(ctx, "SELECT renovacion FROM mp_conexiones WHERE negocio_id = app.negocio_actual()").Scan(&renov)
		})
		if err == nil {
			_, err = s.renovar(ctx, n, renov)
		}
		if err != nil {
			fallidas++
			continue
		}
		renovadas++
	}
	return renovadas, fallidas, nil
}

// RenovarCada corre RenovarPorVencer cada tanto hasta que ctx se cancele (va en una goroutine de
// cmd/api).
func (s *Servicio) RenovarCada(ctx context.Context, cada time.Duration) {
	t := time.NewTicker(cada)
	defer t.Stop()
	for {
		if r, f, err := s.RenovarPorVencer(ctx); err != nil {
			s.log().Error("mercadopago: renovación de tokens", "error", err)
		} else if r+f > 0 {
			s.log().Info("mercadopago: tokens renovados", "renovados", r, "fallidos", f)
		}
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
	}
}
