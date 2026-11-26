package sugerencias

import (
	"slices"
	"time"

	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/sesion"
)

// Tiempos de las reglas.
const (
	// DiasDeGracia: la cuenta nueva no ve sugerencias (primero, que aprenda a usar Tengo Libre).
	DiasDeGracia = 14
	// DiasOculta: «Ocultar por 30 días».
	DiasOculta = 30
)

// Regla: qué sugerencia se activa con qué rubros del negocio (ids de miweb.Rubros).
type Regla struct {
	Sugerencia string
	Rubros     []string
}

// RubrosTLW: los rubros que venden algo además del servicio (productos de pelo, cremas, suplementos,
// etc.). Lista explícita de miweb.Rubros sin «salud»: a un consultorio no se le ofrece «Vendé tus
// productos online» (decisión del 26/11/2026). Un rubro nuevo en miweb no entra solo.
var RubrosTLW = []string{"barberia", "peluqueria", "estetica", "unias", "cejas-pestanias", "maquillaje",
	"depilacion", "masajes", "spa", "tatuajes", "entrenamiento", "otro"}

// Reglas, en orden de prioridad (se muestra la primera que aplica).
var Reglas = []Regla{
	{Sugerencia: "tl-tlw", Rubros: RubrosTLW},
}

// Datos del negocio y de quien mira que deciden si se muestra algo.
type Datos struct {
	Rol      string
	Rubro    string
	CreadoEn time.Time
	// AltaCompleta: la guía «Para recibir turnos» está terminada (arranque.Estado.Completo: todos los
	// pasos obligatorios, no sólo que haya algo reservable).
	AltaCompleta bool
	// Turnos: cuántos turnos tuvo el negocio (0: la cuenta está vacía).
	Turnos int
	// Bloqueado: la prueba terminó sin suscripción (el Inicio le pide elegir un plan).
	Bloqueado bool
	// Apagadas: el interruptor «Sugerencias» de Configuración. Sólo vale con Pago: sin una suscripción
	// que pague, el interruptor no existe y las sugerencias se ven aunque lo haya apagado antes (pedido de
	// Nahuel, 26/11/2026). El retiro gratuito del art. 27 es «Ya la uso / No me interesa» en la tarjeta.
	Apagadas bool
	// Pago: hay una suscripción vigente (Inicial, Equipo o Centro). La prueba de 14 días no cuenta.
	Pago bool
	// Cerradas: hasta cuándo está cerrada cada marca.
	Cerradas map[string]time.Time
}

// Elegir devuelve la sugerencia que le toca ver, o nil.
func Elegir(d Datos, ahora time.Time) *Sugerencia {
	if !slices.Contains(sesion.Gestion, d.Rol) || (d.Apagadas && d.Pago) || d.Bloqueado || d.Rubro == "" {
		return nil
	}
	if ahora.Sub(d.CreadoEn) < DiasDeGracia*24*time.Hour || !d.AltaCompleta || d.Turnos == 0 {
		return nil
	}
	for _, r := range Reglas {
		if !slices.Contains(r.Rubros, d.Rubro) {
			continue
		}
		s, ok := DelCatalogo(r.Sugerencia)
		if !ok {
			continue
		}
		if hasta, cerrada := d.Cerradas[s.Marca]; cerrada && ahora.Before(hasta) {
			continue
		}
		return &s
	}
	return nil
}
