-- +goose Up
-- Referidos y cupones atados a lo que pasa con el turno, y el link del turno sin datos de otro.
--
-- 1. turnos.cliente_ajeno: la reserva (web o IA) dio el teléfono de un cliente que ya existía con un
--    email de confianza distinto del que se tipeó. Quien tiene ese link no probó ser el cliente: no
--    ve su código de referidos ni sus premios.
-- 2. El premio del referido se da cuando el turno queda confirmado o atendido (no al reservar: un
--    pendiente_pago que nunca se paga no premia). referido_usos.premiado_en lo marca, cupon_id es el
--    cupón que se dio y premio_desde, desde cuándo vale (lo pone la API con su reloj al reservar).
--    Los usos de antes ya estaban premiados.
-- 3. Al cancelar un turno vuelven el uso del cupón que usó y el lugar del referido (el premio que
--    todavía no se usó se anula). Los pendiente_pago vencidos los devuelve la próxima reserva con ese
--    cupón o código (no cambian de estado: ver promos.liberarVencidos).
ALTER TABLE turnos ADD COLUMN cliente_ajeno boolean NOT NULL DEFAULT false;

ALTER TABLE referido_usos ADD COLUMN premiado_en timestamptz, ADD COLUMN cupon_id uuid, ADD COLUMN premio_desde timestamptz;
ALTER TABLE referido_usos NO FORCE ROW LEVEL SECURITY;
UPDATE referido_usos SET premiado_en = creado_en;
ALTER TABLE referido_usos FORCE ROW LEVEL SECURITY;
GRANT UPDATE (premiado_en, cupon_id), DELETE ON referido_usos TO tengolibre_app;
GRANT DELETE ON cupon_usos TO tengolibre_app;
CREATE INDEX referido_usos_turno ON referido_usos (turno_id);
CREATE INDEX cupon_usos_turno ON cupon_usos (turno_id);

-- +goose StatementBegin
-- n caracteres al azar (n <= 16) del alfabeto de los códigos (sin 0/O ni 1/I), de gen_random_uuid.
CREATE FUNCTION app.codigo_al_azar(n int) RETURNS text
LANGUAGE sql VOLATILE AS $$
	SELECT string_agg(substr('ABCDEFGHJKLMNPQRSTUVWXYZ23456789', 1 + get_byte(x.b, i) % 32, 1), '' ORDER BY i)
	FROM (SELECT decode(md5(gen_random_uuid()::text || gen_random_uuid()::text), 'hex') AS b) x, generate_series(0, n - 1) i
$$;
-- +goose StatementEnd

-- +goose StatementBegin
-- Premia los usos de referido de un turno que todavía no se premiaron: un cupón de un uso para quien
-- refirió, de 90 días desde la reserva (premio_desde; sin él, creado_en).
CREATE FUNCTION app.premiar_referido(p_turno uuid) RETURNS void
LANGUAGE plpgsql AS $$
DECLARE
	u record;
	tipo text;
	valor bigint;
	c uuid;
BEGIN
	SELECT a.referidos_tipo, a.referidos_valor INTO tipo, valor
	FROM ajustes a JOIN turnos t ON t.negocio_id = a.negocio_id WHERE t.id = p_turno;
	FOR u IN SELECT ru.id, ru.negocio_id, ru.referido_id, coalesce(ru.premio_desde, ru.creado_en) AS desde, r.codigo FROM referido_usos ru
		JOIN referidos r ON r.id = ru.referido_id
		WHERE ru.turno_id = p_turno AND ru.premiado_en IS NULL FOR UPDATE OF ru
	LOOP
		c := NULL;
		IF tipo IS NOT NULL AND valor IS NOT NULL THEN
			INSERT INTO cupones (negocio_id, codigo, tipo, valor, max_usos, desde, hasta, referido_id)
			VALUES (u.negocio_id, 'REF-' || u.codigo || '-' || app.codigo_al_azar(8), tipo, valor, 1,
				u.desde, u.desde + interval '90 days', u.referido_id)
			RETURNING id INTO c;
		END IF;
		UPDATE referido_usos SET premiado_en = now(), cupon_id = c WHERE id = u.id;
	END LOOP;
END
$$;
-- +goose StatementEnd
GRANT EXECUTE ON FUNCTION app.premiar_referido(uuid) TO tengolibre_app;

-- +goose StatementBegin
CREATE FUNCTION app.promos_del_turno() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
	IF NEW.estado IS NOT DISTINCT FROM OLD.estado THEN
		RETURN NULL;
	END IF;
	IF NEW.estado IN ('confirmado', 'atendido') THEN
		PERFORM app.premiar_referido(NEW.id);
	ELSIF NEW.estado = 'cancelado' THEN
		WITH usados AS (DELETE FROM cupon_usos WHERE turno_id = NEW.id RETURNING cupon_id),
			n AS (SELECT cupon_id, count(*) AS cuantos FROM usados GROUP BY cupon_id)
		UPDATE cupones c SET usos = greatest(c.usos - n.cuantos, 0) FROM n WHERE c.id = n.cupon_id;
		-- El premio que ya se dio por este turno y no se usó, se anula; usado, queda.
		UPDATE cupones SET activo = false
		WHERE usos = 0 AND id IN (SELECT cupon_id FROM referido_usos WHERE turno_id = NEW.id AND cupon_id IS NOT NULL);
		DELETE FROM referido_usos ru WHERE ru.turno_id = NEW.id
			AND (ru.cupon_id IS NULL OR EXISTS (SELECT 1 FROM cupones c WHERE c.id = ru.cupon_id AND NOT c.activo));
	END IF;
	RETURN NULL;
END
$$;
-- +goose StatementEnd
CREATE TRIGGER promos AFTER UPDATE OF estado ON turnos FOR EACH ROW EXECUTE FUNCTION app.promos_del_turno();

-- +goose Down
DROP TRIGGER promos ON turnos;
DROP FUNCTION app.promos_del_turno();
DROP FUNCTION app.premiar_referido(uuid);
DROP FUNCTION app.codigo_al_azar(int);
DROP INDEX cupon_usos_turno;
DROP INDEX referido_usos_turno;
REVOKE DELETE ON cupon_usos FROM tengolibre_app;
REVOKE UPDATE (premiado_en, cupon_id), DELETE ON referido_usos FROM tengolibre_app;
ALTER TABLE referido_usos DROP COLUMN premio_desde, DROP COLUMN cupon_id, DROP COLUMN premiado_en;
ALTER TABLE turnos DROP COLUMN cliente_ajeno;
