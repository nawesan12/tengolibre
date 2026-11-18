package sesion

import (
	"context"

	"github.com/jackc/pgx/v5"

	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/problema"
)

// DeSuAgenda: la condición SQL (sobre el profesional_id de un turno, en la columna col) de «es de la
// agenda de quien pide», con su clerk_id en $n. La misma regla que /v1/panel/turnos para el rol
// profesional: los profesionales atados a su cuenta.
func DeSuAgenda(col, n string) string {
	return col + ` IN (SELECT p.id FROM profesionales p JOIN miembros m ON m.id = p.miembro_id WHERE m.clerk_id = ` + n + `)`
}

// TurnoPropio: un profesional sólo toca los turnos de su agenda; uno ajeno es 404 (como en turnos).
// Los demás roles, todos.
func TurnoPropio(ctx context.Context, tx pgx.Tx, turnoID string) error {
	ses := De(ctx)
	if ses.Rol != Profesional {
		return nil
	}
	var ok bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM turnos t WHERE t.id::text = $1 AND `+DeSuAgenda("t.profesional_id", "$2")+`)`,
		turnoID, ses.ClerkID).Scan(&ok); err != nil {
		return err
	}
	if !ok {
		return problema.NoEncontrado("No encontramos ese turno.")
	}
	return nil
}
