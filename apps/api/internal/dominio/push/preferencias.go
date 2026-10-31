package push

import (
	"context"
	"errors"
	"net/http"
	"regexp"
	"slices"

	"github.com/jackc/pgx/v5"

	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/auditoria"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/problema"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/sesion"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/web"
)

// Tipo de la campanita como lo muestra la pantalla de preferencias.
type Tipo struct {
	Tipo   string `json:"tipo"`
	Nombre string `json:"nombre"`
}

// Tipos de la campanita que se pueden elegir, en el orden en que se muestran. Un tipo que no está
// acá (lo sumó otro dominio) llega igual a quien no lo apagó: lo guardado es lo apagado.
var Tipos = []Tipo{
	{"reserva", "Reservas nuevas"},
	{"cancelacion", "Cancelaciones"},
	{"reprogramacion", "Cambios de horario"},
	{"confirmacion", "Confirmaciones por WhatsApp"},
	{"cambio_pedido", "Pedidos para cambiar el turno"},
	{"sin_respuesta", "Clientes que no contestaron el WhatsApp"},
	{"resena", "Reseñas"},
	{"espera", "Lista de espera"},
	{"ausente", "Ausentes"},
	{"equipo", "Equipo"},
	{"aviso_fallido", "Avisos que no salieron"},
}

// ApagadosPorRol: lo que no le llega a cada rol mientras no elija otra cosa. La dueña y quien
// encarga ven todo; recepción, todo menos el equipo; una profesional, lo de la agenda y las reseñas.
var ApagadosPorRol = map[string][]string{
	sesion.Duenio:      {},
	sesion.Encargado:   {},
	sesion.Recepcion:   {"equipo"},
	sesion.Profesional: {"confirmacion", "cambio_pedido", "sin_respuesta", "espera", "ausente", "equipo", "aviso_fallido"},
}

// Apagado dice si el tipo no le llega a la persona: apagados es lo que eligió (nil: lo de su rol).
func Apagado(tipo, rol string, apagados []string) bool {
	if apagados == nil {
		apagados = ApagadosPorRol[rol]
	}
	return slices.Contains(apagados, tipo)
}

// TipoPreferencia: un tipo con si le llega.
type TipoPreferencia struct {
	Tipo
	Activo bool `json:"activo"`
}

// Preferencias de push de una persona en un negocio.
type Preferencias struct {
	Rol string `json:"rol"`
	// PorDefecto: no eligió nada, valen las de su rol.
	PorDefecto bool              `json:"porDefecto"`
	Apagados   []string          `json:"apagados"`
	Tipos      []TipoPreferencia `json:"tipos"`
}

func (s *Servicio) rutasPreferencias(mux *http.ServeMux) {
	mux.HandleFunc("GET /v1/panel/push/preferencias", s.Sesion.Panel(sesion.Agenda, web.Envolver(s.verPreferencias)))
	mux.HandleFunc("PUT /v1/panel/push/preferencias", s.Sesion.Panel(sesion.Agenda, web.Envolver(s.guardarPreferencias)))
}

func leerPreferencias(ctx context.Context, tx pgx.Tx, clerkID, rol string) (Preferencias, error) {
	p := Preferencias{Rol: rol}
	err := tx.QueryRow(ctx, "SELECT apagados FROM push_preferencias WHERE clerk_id = $1", clerkID).Scan(&p.Apagados)
	if errors.Is(err, pgx.ErrNoRows) {
		p.PorDefecto, p.Apagados, err = true, slices.Clone(ApagadosPorRol[rol]), nil
	}
	if err != nil {
		return p, err
	}
	if p.Apagados == nil {
		p.Apagados = []string{}
	}
	for _, t := range Tipos {
		p.Tipos = append(p.Tipos, TipoPreferencia{Tipo: t, Activo: !slices.Contains(p.Apagados, t.Tipo)})
	}
	return p, nil
}

func (s *Servicio) verPreferencias(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	ses := sesion.De(ctx)
	var p Preferencias
	err := s.Pool.EnNegocio(ctx, ses.NegocioID, func(tx pgx.Tx) (err error) {
		p, err = leerPreferencias(ctx, tx, ses.ClerkID, ses.Rol)
		return err
	})
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusOK, p)
	return nil
}

var tipoValido = regexp.MustCompile(`^[a-z_]{1,40}$`)

// guardarPreferencias: {"apagados": [...]} guarda lo que no le llega; {"porDefecto": true} vuelve a
// lo de su rol. Vale para el navegador, el iPhone y Android de esa persona en este negocio.
func (s *Servicio) guardarPreferencias(w http.ResponseWriter, r *http.Request) error {
	var in struct {
		Apagados   []string `json:"apagados"`
		PorDefecto bool     `json:"porDefecto"`
	}
	if err := web.Leer(w, r, &in); err != nil {
		return err
	}
	if !in.PorDefecto && in.Apagados == nil {
		return problema.Invalido("Mandá apagados (la lista de tipos que no te llegan) o porDefecto.",
			problema.Campo{Ruta: "apagados", Mensaje: "falta"})
	}
	apagados := []string{}
	for _, t := range in.Apagados {
		if !tipoValido.MatchString(t) {
			return problema.Invalido("Ese tipo de aviso no existe.", problema.Campo{Ruta: "apagados", Mensaje: t})
		}
		if !slices.Contains(apagados, t) {
			apagados = append(apagados, t)
		}
	}
	if len(apagados) > 40 {
		return problema.Invalido("Son demasiados tipos.", problema.Campo{Ruta: "apagados", Mensaje: "hasta 40"})
	}
	ctx := r.Context()
	ses := sesion.De(ctx)
	var p Preferencias
	err := s.Pool.EnNegocio(ctx, ses.NegocioID, func(tx pgx.Tx) error {
		var err error
		if in.PorDefecto {
			_, err = tx.Exec(ctx, "DELETE FROM push_preferencias WHERE clerk_id = $1", ses.ClerkID)
		} else {
			_, err = tx.Exec(ctx, `INSERT INTO push_preferencias (negocio_id, clerk_id, apagados) VALUES (app.negocio_actual(), $1, $2)
				ON CONFLICT (negocio_id, clerk_id) DO UPDATE SET apagados = excluded.apagados, actualizado_en = now()`, ses.ClerkID, apagados)
		}
		if err != nil {
			return err
		}
		if p, err = leerPreferencias(ctx, tx, ses.ClerkID, ses.Rol); err != nil {
			return err
		}
		return auditoria.Anotar(ctx, tx, auditoria.Registro{Accion: "push.preferencias", Entidad: "push",
			Detalle: map[string]any{"apagados": p.Apagados, "porDefecto": p.PorDefecto}})
	})
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusOK, p)
	return nil
}
