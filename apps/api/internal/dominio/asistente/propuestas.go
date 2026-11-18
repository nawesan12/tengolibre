package asistente

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/avisos"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/mcp"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/problema"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/sesion"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/web"
)

// VidaPropuesta: cuánto espera una propuesta a que la confirmen (después, se vuelve a pedir: la
// agenda pudo cambiar).
const VidaPropuesta = 15 * time.Minute

// Propuesta: una acción que espera el «Confirmar» de quien preguntó.
type Propuesta struct {
	ID      string    `json:"id"`
	Accion  string    `json:"accion"`
	Resumen string    `json:"resumen"`
	VenceEn time.Time `json:"venceEn"`
}

// OfrecerALaLista: la acción propia del asistente (no está en el MCP).
const OfrecerALaLista = "ofrecer_a_la_lista"

func (s *Servicio) proponer(ctx context.Context, accion, resumen string, args json.RawMessage) (Propuesta, error) {
	ses := sesion.De(ctx)
	p := Propuesta{Accion: accion, Resumen: recortar(resumen, 300), VenceEn: s.ahora().Add(VidaPropuesta)}
	err := s.Pool.EnNegocio(ctx, ses.NegocioID, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `INSERT INTO asistente_propuestas (negocio_id, clerk_id, accion, argumentos, resumen, creada_en, vence_en)
			VALUES (app.negocio_actual(), $1, $2, $3, $4, $5, $6) RETURNING id::text`,
			ses.ClerkID, accion, args, p.Resumen, s.ahora(), p.VenceEn).Scan(&p.ID)
	})
	return p, err
}

func recortar(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n-1]) + "…"
}

type pedidoPropuesta struct {
	PropuestaID string `json:"propuestaId"`
}

// tomar marca la propuesta (de quien la pide, vigente y sin resolver) con el estado y devuelve qué
// había que hacer. Lo que corre después va en otra transacción.
func (s *Servicio) tomar(ctx context.Context, id, estado string) (accion string, args json.RawMessage, resumen string, err error) {
	ses := sesion.De(ctx)
	id = strings.TrimSpace(id)
	if id == "" {
		return "", nil, "", problema.Invalido("Falta propuestaId.")
	}
	ahora := s.ahora()
	err = s.Pool.EnNegocio(ctx, ses.NegocioID, func(tx pgx.Tx) error {
		err := tx.QueryRow(ctx, `UPDATE asistente_propuestas SET estado = $3, resuelta_en = $4
			WHERE id::text = $1 AND clerk_id = $2 AND estado IS NULL AND vence_en > $4 RETURNING accion, argumentos, resumen`,
			id, ses.ClerkID, estado, ahora).Scan(&accion, &args, &resumen)
		if !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		// ¿Por qué no? Para decirlo bien.
		var actual *string
		var vence time.Time
		if e := tx.QueryRow(ctx, "SELECT estado, vence_en FROM asistente_propuestas WHERE id::text = $1 AND clerk_id = $2", id, ses.ClerkID).
			Scan(&actual, &vence); e != nil {
			return problema.NoEncontrado("No encontramos esa propuesta.")
		}
		if actual != nil {
			return problema.Conflicto("propuesta_resuelta", "Esa propuesta ya se "+map[string]string{"confirmada": "confirmó",
				"descartada": "descartó", "fallida": "intentó y no se pudo"}[*actual]+".")
		}
		return problema.Conflicto("propuesta_vencida", "Pasó mucho tiempo y la agenda pudo cambiar: volvé a pedirlo.")
	})
	return accion, args, resumen, err
}

// confirmar: corre la acción de la propuesta con la sesión (y el rol) de quien la confirma.
func (s *Servicio) confirmar(w http.ResponseWriter, r *http.Request) error {
	var in pedidoPropuesta
	if err := web.Leer(w, r, &in); err != nil {
		return err
	}
	ctx := r.Context()
	ses := sesion.De(ctx)
	op, err := s.MCP.Operador(ctx)
	if err != nil {
		return err
	}
	// El sólo lectura se mira antes de gastar la propuesta.
	if err := op.PuedeEscribir(ctx); err != nil {
		return err
	}
	accion, args, resumen, err := s.tomar(ctx, in.PropuestaID, "confirmada")
	if err != nil {
		return err
	}
	var res any
	if accion == OfrecerALaLista {
		res, err = s.ofrecer(ctx, op, args)
	} else {
		res, _, err = op.Hacer(ctx, accion, args)
	}
	if err != nil {
		if e := s.Pool.EnNegocio(context.WithoutCancel(ctx), ses.NegocioID, func(tx pgx.Tx) error {
			_, err := tx.Exec(ctx, "UPDATE asistente_propuestas SET estado = 'fallida' WHERE id::text = $1", in.PropuestaID)
			return err
		}); e != nil {
			return e
		}
		var p *problema.Problema
		if errors.As(err, &p) {
			return err
		}
		return problema.Conflicto("no_se_pudo", mcp.Explicar(err, accion))
	}
	if s.Cache != nil {
		s.Cache.Olvidar(ses.NegocioID)
	}
	web.JSON(w, http.StatusOK, map[string]any{"respuesta": "Listo. " + hecho(resumen), "resultado": res})
	return nil
}

// hecho: «Mover el turno de Juana…» → «Movimos el turno de Juana…».
func hecho(resumen string) string {
	for _, x := range [][2]string{{"Mover ", "Movimos "}, {"Cancelar ", "Cancelamos "}, {"Confirmar ", "Confirmamos "},
		{"Cargar ", "Cargamos "}, {"Ofrecer ", "Ofrecimos "}} {
		if strings.HasPrefix(resumen, x[0]) {
			r := x[1] + strings.TrimPrefix(resumen, x[0])
			// «Le avisamos…» ya está dicho en presente: queda.
			return r
		}
	}
	return resumen
}

func (s *Servicio) descartar(w http.ResponseWriter, r *http.Request) error {
	var in pedidoPropuesta
	if err := web.Leer(w, r, &in); err != nil {
		return err
	}
	if _, _, _, err := s.tomar(r.Context(), in.PropuestaID, "descartada"); err != nil {
		return err
	}
	web.JSON(w, http.StatusOK, map[string]any{"respuesta": "Listo, no hago nada."})
	return nil
}

// ---- Ofrecer un hueco a la lista de espera

type pedidoOferta struct {
	ServicioID    string `json:"servicioId"`
	ProfesionalID string `json:"profesionalId"`
	Fecha         string `json:"fecha"`
	Hora          string `json:"hora"`
}

var esquemaOferta = map[string]any{"type": "object", "additionalProperties": false,
	"required": []string{"servicioId", "profesionalId", "fecha", "hora"},
	"properties": map[string]any{
		"servicioId":    map[string]any{"type": "string", "description": "El servicio (id)."},
		"profesionalId": map[string]any{"type": "string", "description": "El profesional (id)."},
		"fecha":         map[string]any{"type": "string", "description": "Día, AAAA-MM-DD.", "pattern": `^\d{4}-\d{2}-\d{2}$`},
		"hora":          map[string]any{"type": "string", "description": "Hora de inicio, HH:MM.", "pattern": `^\d{1,2}:\d{2}$`},
	}}

const descripcionOferta = "Ofrece un horario libre a la lista de espera: se le avisa a una persona por vez, en orden, hasta " +
	"que alguien lo toma. Sirve cuando se libera un hueco (si se cancela un turno, la lista ya se avisa sola). Necesita " +
	"servicio, profesional, fecha y hora; el horario tiene que estar libre y alguien de la lista tiene que buscarlo."

func (s *Servicio) leerOferta(op *mcp.Operador, raw json.RawMessage) (pedidoOferta, time.Time, error) {
	var in pedidoOferta
	if err := json.Unmarshal(raw, &in); err != nil {
		return in, time.Time{}, problema.Invalido("Los argumentos no son válidos.")
	}
	inicio, err := time.ParseInLocation("2006-01-02 15:04", strings.TrimSpace(in.Fecha)+" "+pad(strings.TrimSpace(in.Hora)), op.Zona())
	if err != nil || in.ServicioID == "" || in.ProfesionalID == "" {
		return in, inicio, problema.Invalido("Faltan servicioId, profesionalId, fecha (AAAA-MM-DD) y hora (HH:MM).")
	}
	return in, inicio, nil
}

func pad(h string) string {
	if len(h) == 4 {
		return "0" + h
	}
	return h
}

// previaOferta: cuántos lo buscan y la frase para confirmar.
func (s *Servicio) previaOferta(ctx context.Context, op *mcp.Operador, raw json.RawMessage) (string, json.RawMessage, error) {
	if !op.Puede("ver_lista_de_espera") || s.Espera == nil {
		return "", nil, problema.SinPermiso("La lista de espera la maneja el mostrador.")
	}
	in, inicio, err := s.leerOferta(op, raw)
	if err != nil {
		return "", nil, err
	}
	n, err := s.Espera.Candidatos(ctx, sesion.De(ctx).NegocioID, in.ServicioID, in.ProfesionalID, inicio)
	if err != nil {
		return "", nil, err
	}
	if n == 0 {
		return "", nil, problema.Conflicto("sin_espera", "Nadie de la lista de espera busca ese horario.")
	}
	quien := "1 persona que lo busca"
	if n > 1 {
		quien = "las " + itoa(n) + " personas que lo buscan, una por vez"
	}
	return "Ofrecer el " + avisos.Fecha(inicio) + " a las " + avisos.Hora(inicio) + " a " + quien + ".", raw, nil
}

func (s *Servicio) ofrecer(ctx context.Context, op *mcp.Operador, raw json.RawMessage) (any, error) {
	if !op.Puede("ver_lista_de_espera") || s.Espera == nil {
		return nil, problema.SinPermiso("La lista de espera la maneja el mostrador.")
	}
	in, inicio, err := s.leerOferta(op, raw)
	if err != nil {
		return nil, err
	}
	return s.Espera.Ofrecer(ctx, sesion.De(ctx).NegocioID, in.ServicioID, in.ProfesionalID, inicio)
}

func itoa(n int) string {
	b, _ := json.Marshal(n)
	return string(b)
}
