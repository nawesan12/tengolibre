-- +goose Up
-- «Tus turnos» sin turnos ajenos (revisión del comité).
--
-- Antes, app.turnos_de_persona traía todo turno cuyo cliente tuviera el mail de la cuenta, aunque ese
-- mail lo hubiera tipeado cualquiera en la web: con el teléfono de una clienta cargada sin mail y un
-- mail propio, la reserva le completaba la ficha y «Tus turnos» listaba todos sus turnos con el código.
--
-- 1. turnos.email_reserva: el mail con que se reservó ESE turno desde la web (o la lista de espera, o
--    el asistente de una persona). A ese mail le llega el comprobante con el link, así que quien lo
--    controla ya conoce el código: verlo en su cuenta no le muestra nada nuevo.
-- 2. app.turnos_de_persona trae un turno si el mail de la cuenta es el de la reserva o es el mail de
--    confianza de la ficha (cargado por el negocio). Nunca por un mail que la web le pegó a una ficha.
--    Sin los pendiente_pago vencidos (el horario ya no está guardado).
-- 3. Primero los negocios donde hay algo con ese mail (dos índices y una política de barrido de sólo
--    lectura), después el recorrido: no se recorre la plataforma entera en cada carga de Mi cuenta.
ALTER TABLE turnos ADD COLUMN email_reserva text NOT NULL DEFAULT '' CHECK (length(email_reserva) <= 320);
CREATE INDEX turnos_email_reserva ON turnos (lower(email_reserva)) WHERE email_reserva <> '';
CREATE INDEX clientes_email_confiable_global ON clientes (lower(email)) WHERE email_confiable;

CREATE POLICY barrido_turnos_de_persona ON turnos FOR SELECT TO tengolibre_migrador
	USING (app.barrido() = 'turnos_de_persona');
CREATE POLICY barrido_turnos_de_persona ON clientes FOR SELECT TO tengolibre_migrador
	USING (app.barrido() = 'turnos_de_persona');

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION app.turnos_de_persona(p_email text, p_desde timestamptz)
RETURNS TABLE (codigo text, inicio timestamptz, fin timestamptz, estado text, servicio text, profesional text,
	negocio text, slug text, zona text)
LANGUAGE plpgsql SECURITY DEFINER SET search_path = public, app AS $$
DECLARE
	antes text := current_setting('app.negocio_id', true);
	mail text := lower(trim(coalesce(p_email, '')));
	ids uuid[];
	n record;
BEGIN
	IF mail = '' THEN
		RETURN;
	END IF;
	PERFORM set_config('app.barrido', 'turnos_de_persona', true);
	SELECT coalesce(array_agg(DISTINCT x.negocio_id), '{}') INTO ids FROM (
		SELECT t.negocio_id FROM turnos t WHERE t.email_reserva <> '' AND lower(t.email_reserva) = mail
		UNION SELECT c.negocio_id FROM clientes c WHERE c.email_confiable AND lower(c.email) = mail
	) x;
	PERFORM set_config('app.barrido', 'negocios', true);
	FOR n IN SELECT x.id, x.nombre, x.slug, x.zona_horaria FROM negocios x WHERE x.activo AND x.id = ANY (ids) LOOP
		PERFORM set_config('app.barrido', '', true);
		PERFORM set_config('app.negocio_id', n.id::text, true);
		RETURN QUERY SELECT t.codigo, t.inicio, t.fin, t.estado, coalesce(s.nombre, ''), coalesce(p.nombre, ''),
				n.nombre, n.slug, n.zona_horaria
			FROM turnos t JOIN clientes c ON c.id = t.cliente_id
			LEFT JOIN servicios s ON s.id = t.servicio_id
			LEFT JOIN profesionales p ON p.id = t.profesional_id
			WHERE t.fin >= p_desde AND t.estado <> 'cancelado'
				AND NOT (t.estado = 'pendiente_pago' AND t.vence_en IS NOT NULL AND t.vence_en <= p_desde)
				AND ((t.email_reserva <> '' AND lower(t.email_reserva) = mail)
					OR (c.email_confiable AND lower(c.email) = mail));
		PERFORM set_config('app.barrido', 'negocios', true);
	END LOOP;
	PERFORM set_config('app.barrido', '', true);
	PERFORM set_config('app.negocio_id', coalesce(antes, ''), true);
END
$$;
-- +goose StatementEnd

-- +goose Down
DROP POLICY barrido_turnos_de_persona ON clientes;
DROP POLICY barrido_turnos_de_persona ON turnos;
DROP INDEX clientes_email_confiable_global;
DROP INDEX turnos_email_reserva;
ALTER TABLE turnos DROP COLUMN email_reserva;
