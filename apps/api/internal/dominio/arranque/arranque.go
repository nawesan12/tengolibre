// Package arranque: el onboarding del panel, qué le falta configurar al negocio para que lo puedan
// reservar y para que su página se vea completa (onboarding/status de Jiku, con más pasos).
package arranque

import (
	"net/http"

	"github.com/jackc/pgx/v5"

	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/db"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/sesion"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/web"
)

// Servicio del onboarding.
type Servicio struct {
	Pool   *db.Pool
	Sesion *sesion.Servicio
}

// Paso del onboarding.
type Paso struct {
	Clave  string `json:"clave"`
	Titulo string `json:"titulo"`
	Listo  bool   `json:"listo"`
	// Obligatorio: sin él el negocio no está completo (los otros mejoran la página).
	Obligatorio bool `json:"obligatorio"`
}

// Estado del onboarding.
type Estado struct {
	// Completo: todos los obligatorios listos.
	Completo bool `json:"completo"`
	// Reservable: hay algo que un cliente puede reservar (lo mismo que pide el directorio).
	Reservable bool   `json:"reservable"`
	Pasos      []Paso `json:"pasos"`
}

// Rutas del dominio.
func (s *Servicio) Rutas(mux *http.ServeMux) {
	mux.HandleFunc("GET /v1/panel/arranque", s.Sesion.Panel(sesion.Agenda, web.Envolver(s.estado)))
}

func (s *Servicio) estado(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	var perfil, contacto, ubicacion, logo, servicios, profesionales, horarios, asignados, fotos bool
	err := s.Pool.EnNegocio(ctx, sesion.De(ctx).NegocioID, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `SELECT
			n.descripcion <> '',
			n.telefono <> '' OR n.whatsapp <> '',
			n.direccion <> '' AND n.ciudad <> '',
			n.logo IS NOT NULL,
			EXISTS (SELECT 1 FROM servicios WHERE activo),
			EXISTS (SELECT 1 FROM profesionales WHERE activo),
			EXISTS (SELECT 1 FROM profesionales p WHERE p.activo AND EXISTS (SELECT 1 FROM horarios h WHERE h.profesional_id = p.id)),
			EXISTS (SELECT 1 FROM profesional_servicios ps JOIN profesionales p ON p.id = ps.profesional_id
				JOIN servicios s ON s.id = ps.servicio_id WHERE p.activo AND s.activo
				AND EXISTS (SELECT 1 FROM horarios h WHERE h.profesional_id = p.id)),
			EXISTS (SELECT 1 FROM fotos)
			FROM negocios n WHERE n.id = app.negocio_actual()`).Scan(
			&perfil, &contacto, &ubicacion, &logo, &servicios, &profesionales, &horarios, &asignados, &fotos)
	})
	if err != nil {
		return err
	}
	e := Estado{Reservable: asignados, Pasos: []Paso{
		{"servicios", "Cargá tus servicios con duración y precio", servicios, true},
		{"profesionales", "Sumá a quienes atienden", profesionales, true},
		{"horarios", "Poné el horario de cada profesional", horarios, true},
		// Sin esto, con todo lo anterior listo, igual no hay horarios libres: nadie hace nada.
		{"asignaciones", "Elegí qué servicios hace cada profesional", asignados, true},
		{"perfil", "Escribí una descripción del negocio", perfil, true},
		{"contacto", "Dejá un teléfono o WhatsApp", contacto, true},
		{"ubicacion", "Poné la dirección y la ciudad", ubicacion, true},
		{"logo", "Subí tu logo", logo, false},
		{"fotos", "Sumá fotos a la galería", fotos, false},
	}}
	e.Completo = true
	for _, p := range e.Pasos {
		if p.Obligatorio && !p.Listo {
			e.Completo = false
		}
	}
	web.JSON(w, http.StatusOK, e)
	return nil
}
