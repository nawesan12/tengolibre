package avisos

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/auditoria"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/db"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/trabajos"
)

// Trabajos del worker que encolan avisos o mueven turnos. Cada uno recibe la hora de la pasada.
type Trabajos struct {
	Pool *db.Pool
	// DespuesDeMarcar se llama por cada turno que el trabajo de ausentes marcó, con evento "ausente",
	// después de confirmar (como turnos.Servicio.Avisar: ahí se enganchan las penalidades). nil: nada.
	DespuesDeMarcar func(ctx context.Context, negocioID, turnoID, evento string)
}

// Ventana del recordatorio: más ancha que el intervalo del trabajo, así ninguna reserva cae entre
// dos pasadas. La clave del aviso (turno + horario) hace inofensivas las pasadas superpuestas, y un
// turno reprogramado vuelve a tener su recordatorio.
const (
	recordatorioDesde = 23 * time.Hour
	recordatorioHasta = 25 * time.Hour
)

// Recordatorios encola el recordatorio de los turnos que empiezan en unas 24 horas.
func (s *Trabajos) Recordatorios(ctx context.Context, ahora time.Time) (any, error) {
	encolados := 0
	err := trabajos.PorNegocio(ctx, s.Pool, func(_ string, tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `INSERT INTO avisos (negocio_id, tipo, turno_id, datos, clave, proximo_intento)
			SELECT t.negocio_id, 'recordatorio', t.id, jsonb_build_object('inicio', to_char(t.inicio AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS.US"Z"')),
				'recordatorio:' || t.id || ':' || extract(epoch FROM t.inicio)::bigint, $1
			FROM turnos t JOIN clientes c ON c.id = t.cliente_id
			WHERE t.estado IN ('pendiente', 'confirmado') AND t.inicio >= $2 AND t.inicio <= $3 AND c.email <> ''
			ORDER BY t.inicio LIMIT 500
			ON CONFLICT (negocio_id, clave) DO NOTHING`, ahora, ahora.Add(recordatorioDesde), ahora.Add(recordatorioHasta))
		encolados += int(tag.RowsAffected())
		return err
	})
	return map[string]int{"encolados": encolados}, err
}

// Gracia antes de marcar ausente: quien cierra el turno un rato tarde no le genera una ausencia al
// cliente (GRACE_MINUTES de Jiku).
const graciaAusente = 2 * time.Hour

// Ausentes marca ausente, en los negocios que lo pidieron (ajustes.marcar_ausentes, que se cambia desde el panel de ausencias), los turnos
// pendientes o confirmados que terminaron hace más de dos horas sin que nadie los cierre.
func (s *Trabajos) Ausentes(ctx context.Context, ahora time.Time) (any, error) {
	type marcado struct{ negocio, turno string }
	var marcados []marcado
	err := trabajos.PorNegocio(ctx, s.Pool, func(negocio string, tx pgx.Tx) error {
		var activo bool
		if err := tx.QueryRow(ctx, "SELECT marcar_ausentes FROM ajustes WHERE negocio_id = app.negocio_actual()").Scan(&activo); err != nil || !activo {
			return err
		}
		filas, err := tx.Query(ctx, `UPDATE turnos t SET estado = 'ausente', actualizado_en = $2
			FROM clientes c WHERE c.id = t.cliente_id AND t.id IN (
				SELECT id FROM turnos WHERE estado IN ('pendiente', 'confirmado') AND fin < $1 ORDER BY fin LIMIT 200 FOR UPDATE SKIP LOCKED)
			RETURNING t.id::text, c.nombre, t.servicio`, ahora.Add(-graciaAusente), ahora)
		if err != nil {
			return err
		}
		type fila struct{ ID, Cliente, Servicio string }
		fs, err := pgx.CollectRows(filas, pgx.RowToStructByPos[fila])
		if err != nil {
			return err
		}
		for _, f := range fs {
			if err := Notificar(ctx, tx, Notificacion{Tipo: NotiAusente, TurnoID: f.ID, Titulo: "Marcamos ausente a " + f.Cliente,
				Texto: "Su turno de " + f.Servicio + " terminó y nadie lo cerró. Si vino, cambialo a atendido desde la agenda."}); err != nil {
				return err
			}
			if err := auditoria.Anotar(ctx, tx, auditoria.Registro{Accion: "turno.ausente_automatico", Entidad: "turno", EntidadID: f.ID}); err != nil {
				return err
			}
			marcados = append(marcados, marcado{negocio, f.ID})
		}
		return nil
	})
	if s.DespuesDeMarcar != nil {
		for _, m := range marcados {
			s.DespuesDeMarcar(ctx, m.negocio, m.turno, "ausente")
		}
	}
	return map[string]int{"marcados": len(marcados)}, err
}

// A partir de qué hora (la del negocio) sale el resumen de mañana: a las siete de la tarde la agenda
// ya está armada y todavía se está a tiempo de mover algo.
const resumenDesdeHora = 19

// Resumen encola "Mañana tenés N turnos" para las dueñas de cada negocio que lo tiene prendido. La
// clave (día + persona) hace que sólo salga una vez por día aunque el trabajo corra toda la tarde;
// un día sin turnos se descarta al armarse.
func (s *Trabajos) Resumen(ctx context.Context, ahora time.Time) (any, error) {
	encolados := 0
	err := trabajos.PorNegocio(ctx, s.Pool, func(_ string, tx pgx.Tx) error {
		n, err := leerNegocio(ctx, tx)
		if err != nil {
			return err
		}
		local := ahora.In(n.zona)
		if local.Hour() < resumenDesdeHora {
			return nil
		}
		var prendido bool
		if err := tx.QueryRow(ctx, "SELECT resumen_diario FROM ajustes WHERE negocio_id = app.negocio_actual()").Scan(&prendido); err != nil || !prendido {
			return err
		}
		manana := local.AddDate(0, 0, 1).Format(time.DateOnly)
		filas, err := tx.Query(ctx, "SELECT id::text, email FROM miembros WHERE rol = 'duenio' AND activo AND email <> ''")
		if err != nil {
			return err
		}
		type duenia struct{ ID, Email string }
		ds, err := pgx.CollectRows(filas, pgx.RowToStructByPos[duenia])
		if err != nil {
			return err
		}
		for _, d := range ds {
			ok, err := Encolar(ctx, tx, Aviso{Tipo: Resumen, Para: d.Email, Clave: "resumen:" + manana + ":" + d.ID,
				Datos: map[string]any{"fecha": manana}}, ahora)
			if err != nil {
				return err
			}
			if ok {
				encolados++
			}
		}
		return nil
	})
	return map[string]int{"encolados": encolados}, err
}

// Cuánto se guarda cada cosa (RETENCION de Jiku): el registro de envíos y la campanita 60 días, la
// auditoría 180 (es lo que se mira cuando algo no cierra), y los links vencidos 30 días más.
const (
	retencionAvisos    = 60 * 24 * time.Hour
	retencionAuditoria = 180 * 24 * time.Hour
	retencionVencidos  = 30 * 24 * time.Hour
	tandaPurga         = 5000
)

// Purgar borra, por tandas, lo que ya nadie va a leer.
func (s *Trabajos) Purgar(ctx context.Context, ahora time.Time) (any, error) {
	out := map[string]int64{}
	err := s.Pool.EnTx(ctx, func(tx pgx.Tx) error {
		filas, err := tx.Query(ctx, "SELECT tabla, borradas FROM app.purgar($1, $2, $3, $4)",
			ahora.Add(-retencionAvisos), ahora.Add(-retencionAuditoria), ahora.Add(-retencionVencidos), tandaPurga)
		if err != nil {
			return err
		}
		defer filas.Close()
		for filas.Next() {
			var tabla string
			var n int64
			if err := filas.Scan(&tabla, &n); err != nil {
				return err
			}
			out[tabla] = n
		}
		return filas.Err()
	})
	return out, err
}

// Lista arma los trabajos del worker de este paquete más el envío de la cola (con los nombres que
// quedan en corridas). Los de otros dominios (pedidos de reseña) se suman al lado.
func (s *Trabajos) Lista(e *Enviador) []trabajos.Trabajo {
	return []trabajos.Trabajo{
		// Manda los nuevos y los reintentos que ya cumplieron su espera.
		{Nombre: "envios", Cada: 5 * time.Second, Correr: e.Correr},
		{Nombre: "recordatorios", Cada: 5 * time.Minute, Correr: s.Recordatorios},
		{Nombre: "ausentes", Cada: 15 * time.Minute, Correr: s.Ausentes},
		{Nombre: "resumen", Cada: 30 * time.Minute, Correr: s.Resumen},
		{Nombre: "purga", Cada: 24 * time.Hour, Correr: s.Purgar},
	}
}
