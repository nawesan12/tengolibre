package planes

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
)

// Los recordatorios automáticos por WhatsApp (confirmación y recordatorio; las respuestas dentro
// de la conversación no cuentan): Centro trae 150 por mes y, fuera de eso, se usan los packs
// comprados (negocios.whatsapp_saldo, no vencen). Sin ninguno de los dos, no se manda: el turno
// sigue teniendo el recordatorio por mail y push, que no tiene límite.

var zonaAR = func() *time.Location {
	z, err := time.LoadLocation("America/Argentina/Buenos_Aires")
	if err != nil {
		return time.FixedZone("UTC-3", -3*3600)
	}
	return z
}()

// inicioDeMes en Argentina.
func inicioDeMes(ahora time.Time) time.Time {
	a := ahora.In(zonaAR)
	return time.Date(a.Year(), a.Month(), 1, 0, 0, 0, 0, zonaAR)
}

// WhatsAppDelMes: los automáticos que el negocio fijado ya encoló este mes (los descartados por un
// teléfono que no servía no cuentan).
func WhatsAppDelMes(ctx context.Context, tx pgx.Tx, ahora time.Time) (int, error) {
	var n int
	err := tx.QueryRow(ctx, `SELECT count(*) FROM whatsapp_mensajes WHERE tipo IN ('confirmacion', 'recordatorio')
		AND estado <> 'descartado' AND creado_en >= $1`, inicioDeMes(ahora)).Scan(&n)
	return n, err
}

// ConsumirWhatsApp: si el negocio fijado puede mandar un automático más. Primero los incluidos del
// mes; después descuenta uno del saldo de packs (en la misma transacción que lo encola: si el
// mensaje no se encola, DevolverWhatsApp). desdeSaldo dice de dónde salió.
func ConsumirWhatsApp(ctx context.Context, tx pgx.Tx, ahora time.Time) (ok, desdeSaldo bool, err error) {
	// En fila por negocio: dos corridas a la vez no se pasan del tope.
	if _, err := tx.Exec(ctx, "SELECT pg_advisory_xact_lock(hashtext('whatsapp:' || app.negocio_actual()::text))"); err != nil {
		return false, false, err
	}
	v, err := Leer(ctx, tx, ahora)
	if err != nil {
		return false, false, err
	}
	if v.WhatsAppIncluidos > 0 {
		usados, err := WhatsAppDelMes(ctx, tx, ahora)
		if err != nil {
			return false, false, err
		}
		if usados < v.WhatsAppIncluidos {
			return true, false, nil
		}
	}
	tag, err := tx.Exec(ctx, "UPDATE negocios SET whatsapp_saldo = whatsapp_saldo - 1 WHERE id = app.negocio_actual() AND whatsapp_saldo > 0")
	if err != nil {
		return false, false, err
	}
	return tag.RowsAffected() == 1, true, nil
}

// DevolverWhatsApp: el mensaje que se descontó del saldo al final no se encoló.
func DevolverWhatsApp(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx, "UPDATE negocios SET whatsapp_saldo = whatsapp_saldo + 1 WHERE id = app.negocio_actual()")
	return err
}
