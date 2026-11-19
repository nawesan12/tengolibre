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

// InicioDeMes en Argentina.
func InicioDeMes(ahora time.Time) time.Time {
	a := ahora.In(zonaAR)
	return time.Date(a.Year(), a.Month(), 1, 0, 0, 0, 0, zonaAR)
}

// WhatsAppDelMes: los automáticos incluidos en el plan que el negocio fijado usó este mes. Los que
// salieron del saldo de packs no cuentan (ya se pagaron aparte), y los que no salieron (descartados o
// fallidos) tampoco: es la misma regla que devuelve la unidad al saldo (app.whatsapp_devolver_saldo).
func WhatsAppDelMes(ctx context.Context, tx pgx.Tx, ahora time.Time) (int, error) {
	var n int
	err := tx.QueryRow(ctx, `SELECT count(*) FROM whatsapp_mensajes WHERE tipo IN ('confirmacion', 'recordatorio')
		AND NOT desde_saldo AND estado NOT IN ('descartado', 'fallido') AND creado_en >= $1`, InicioDeMes(ahora)).Scan(&n)
	return n, err
}

// WhatsAppDisponibles: cuántos automáticos más puede mandar el negocio fijado (los incluidos del mes
// que quedan más el saldo de packs). Para cortar antes de recorrer turnos y para el panel.
func WhatsAppDisponibles(ctx context.Context, tx pgx.Tx, ahora time.Time) (int, error) {
	v, err := Leer(ctx, tx, ahora)
	if err != nil {
		return 0, err
	}
	var saldo int
	if err := tx.QueryRow(ctx, "SELECT whatsapp_saldo FROM negocios WHERE id = app.negocio_actual()").Scan(&saldo); err != nil {
		return 0, err
	}
	if v.WhatsAppIncluidos == 0 {
		return saldo, nil
	}
	usados, err := WhatsAppDelMes(ctx, tx, ahora)
	if err != nil {
		return 0, err
	}
	return max(v.WhatsAppIncluidos-usados, 0) + saldo, nil
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
