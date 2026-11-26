package sugerencias

import (
	"slices"
	"time"

	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/miweb"
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

// Reglas, en orden de prioridad (se muestra la primera que aplica). Tengo Libre → Tu Local Web: con
// cualquier rubro cargado (todos venden algo además del servicio: productos de pelo, cremas, etc.).
var Reglas = []Regla{
	{Sugerencia: "tl-tlw", Rubros: miweb.Rubros},
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
	// Apagadas: el interruptor «Sugerencias» de Ajustes.
	Apagadas bool
	// Cerradas: hasta cuándo está cerrada cada marca.
	Cerradas map[string]time.Time
}

// Elegir devuelve la sugerencia que le toca ver, o nil.
func Elegir(d Datos, ahora time.Time) *Sugerencia {
	if !slices.Contains(sesion.Gestion, d.Rol) || d.Apagadas || d.Bloqueado || d.Rubro == "" {
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
