-- +goose Up
-- Los planes del 19/11/2026: Inicial ($6.900), Equipo ($14.900, el destacado) y Centro ($29.900).
-- «completo» pasa a «equipo» (era el plan de arriba con el mismo precio). La prueba dura 14 días.
--  · negocios.profesionales_extra: los profesionales que Centro suma arriba de 25 ($1.900 c/u).
--  · negocios.whatsapp_saldo: recordatorios por WhatsApp comprados en packs (no vencen); los 150
--    de Centro se cuentan por mes sobre whatsapp_mensajes.
--  · suscripciones.periodo: mensual o anual (el anual cobra 10 meses y vale 12).
--  · whatsapp_packs: la compra de un pack (una orden de Mercado Pago de la cuenta de la plataforma).
--  · app.plan_vigente / app.tope_profesionales: el plan que rige (la misma regla que
--    planes.Leer) y cuántos profesionales activos toman turnos. Bajar de plan no borra nada: los
--    que sobran (los más nuevos) quedan en el panel, fuera del plan, hasta subir o liberar lugar.
ALTER TABLE negocios NO FORCE ROW LEVEL SECURITY;
ALTER TABLE suscripciones NO FORCE ROW LEVEL SECURITY;

ALTER TABLE negocios DROP CONSTRAINT negocios_plan_check;
UPDATE negocios SET plan = 'equipo' WHERE plan = 'completo';
ALTER TABLE negocios ADD CONSTRAINT negocios_plan_check CHECK (plan IN ('sin_plan', 'inicial', 'equipo', 'centro')),
	ALTER COLUMN prueba_hasta SET DEFAULT now() + interval '14 days',
	ADD COLUMN profesionales_extra int NOT NULL DEFAULT 0 CHECK (profesionales_extra BETWEEN 0 AND 100),
	ADD COLUMN whatsapp_saldo int NOT NULL DEFAULT 0 CHECK (whatsapp_saldo >= 0);
GRANT UPDATE (profesionales_extra, whatsapp_saldo) ON negocios TO tengolibre_app;

ALTER TABLE suscripciones DROP CONSTRAINT suscripciones_plan_check;
UPDATE suscripciones SET plan = 'equipo' WHERE plan = 'completo';
ALTER TABLE suscripciones ADD CONSTRAINT suscripciones_plan_check CHECK (plan IN ('inicial', 'equipo', 'centro')),
	ADD COLUMN periodo text NOT NULL DEFAULT 'mensual' CHECK (periodo IN ('mensual', 'anual')),
	ADD COLUMN profesionales_extra int NOT NULL DEFAULT 0 CHECK (profesionales_extra BETWEEN 0 AND 100),
	-- Bajar de plan (o sacar extras) no rige ya: se cobra lo de antes hasta el próximo cobro y ahí
	-- pasa (ConciliarCobro). Lo pagado se usa entero.
	ADD COLUMN plan_siguiente text CHECK (plan_siguiente IN ('inicial', 'equipo', 'centro')),
	ADD COLUMN extras_siguiente int CHECK (extras_siguiente BETWEEN 0 AND 100),
	-- Un cambio a la vez (leer, Mercado Pago, escribir): lo toma cambiar/extras y lo suelta al terminar.
	ADD COLUMN cambiando_hasta timestamptz;
GRANT UPDATE (periodo, profesionales_extra, plan_siguiente, extras_siguiente, cambiando_hasta) ON suscripciones TO tengolibre_app;

ALTER TABLE suscripciones FORCE ROW LEVEL SECURITY;
ALTER TABLE negocios FORCE ROW LEVEL SECURITY;

CREATE TABLE whatsapp_packs (
	id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
	negocio_id uuid NOT NULL REFERENCES negocios (id) ON DELETE CASCADE,
	-- external_reference de la orden: empieza con el negocio.
	referencia text NOT NULL UNIQUE CHECK (length(referencia) <= 120),
	cantidad int NOT NULL CHECK (cantidad > 0),
	monto bigint NOT NULL CHECK (monto > 0),
	-- devuelto: Mercado Pago lo devolvió o hubo un contracargo (se descuenta del saldo lo que quede).
	estado text NOT NULL DEFAULT 'pendiente' CHECK (estado IN ('pendiente', 'acreditado', 'cancelado', 'devuelto')),
	mp_orden text UNIQUE CHECK (length(mp_orden) <= 64),
	link text CHECK (length(link) <= 2048),
	creado_en timestamptz NOT NULL DEFAULT now(),
	acreditado_en timestamptz
);
CREATE INDEX whatsapp_packs_negocio ON whatsapp_packs (negocio_id, creado_en DESC);
SELECT app.aplicar_rls('whatsapp_packs');
GRANT SELECT, INSERT ON whatsapp_packs TO tengolibre_app;
GRANT UPDATE (estado, mp_orden, link, acreditado_en) ON whatsapp_packs TO tengolibre_app;

-- Subir de plan (o sumar profesionales extra) a mitad de un período pago: se cobra ya la diferencia
-- por los días que faltan (una orden de la cuenta de la plataforma, como los packs) y el plan nuevo
-- rige recién cuando esa orden se acredita. monto_nuevo: lo que el preapproval cobra desde ahí.
CREATE TABLE suscripcion_subidas (
	id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
	negocio_id uuid NOT NULL REFERENCES negocios (id) ON DELETE CASCADE,
	suscripcion_id uuid NOT NULL REFERENCES suscripciones (id) ON DELETE CASCADE,
	referencia text NOT NULL UNIQUE CHECK (length(referencia) <= 120),
	plan text NOT NULL CHECK (plan IN ('inicial', 'equipo', 'centro')),
	profesionales_extra int NOT NULL DEFAULT 0 CHECK (profesionales_extra BETWEEN 0 AND 100),
	monto_nuevo bigint NOT NULL CHECK (monto_nuevo > 0),
	diferencia bigint NOT NULL CHECK (diferencia > 0),
	-- sin_aplicar: se pagó pero la suscripción ya no estaba (se canceló o cambió): hay que devolverlo.
	estado text NOT NULL DEFAULT 'pendiente' CHECK (estado IN ('pendiente', 'acreditado', 'cancelado', 'devuelto', 'sin_aplicar')),
	mp_orden text UNIQUE CHECK (length(mp_orden) <= 64),
	link text CHECK (length(link) <= 2048),
	creado_en timestamptz NOT NULL DEFAULT now(),
	acreditado_en timestamptz
);
CREATE INDEX suscripcion_subidas_negocio ON suscripcion_subidas (negocio_id, creado_en DESC);
SELECT app.aplicar_rls('suscripcion_subidas');
GRANT SELECT, INSERT ON suscripcion_subidas TO tengolibre_app;
GRANT UPDATE (estado, mp_orden, link, acreditado_en) ON suscripcion_subidas TO tengolibre_app;

-- Las órdenes de la plataforma que siguen pendientes (packs y subidas): el worker las vuelve a leer
-- por si el aviso de Mercado Pago no llegó. Sólo los ids de los negocios, durante el barrido.
CREATE POLICY barrido_ordenes ON whatsapp_packs FOR SELECT TO tengolibre_migrador USING (app.barrido() = 'ordenes_plataforma');
CREATE POLICY barrido_ordenes ON suscripcion_subidas FOR SELECT TO tengolibre_migrador USING (app.barrido() = 'ordenes_plataforma');
-- +goose StatementBegin
CREATE FUNCTION app.negocios_con_ordenes_pendientes(p_desde timestamptz) RETURNS SETOF uuid
LANGUAGE plpgsql SECURITY DEFINER SET search_path = public, app AS $$
BEGIN
	PERFORM set_config('app.barrido', 'ordenes_plataforma', true);
	RETURN QUERY SELECT DISTINCT x.negocio_id FROM (
		SELECT w.negocio_id FROM whatsapp_packs w WHERE w.estado IN ('pendiente', 'cancelado') AND w.mp_orden IS NOT NULL AND w.creado_en > p_desde
		UNION ALL
		SELECT u.negocio_id FROM suscripcion_subidas u WHERE u.estado IN ('pendiente', 'cancelado') AND u.mp_orden IS NOT NULL AND u.creado_en > p_desde) x;
	PERFORM set_config('app.barrido', '', true);
END
$$;
-- +goose StatementEnd
REVOKE ALL ON FUNCTION app.negocios_con_ordenes_pendientes(timestamptz) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION app.negocios_con_ordenes_pendientes(timestamptz) TO tengolibre_app;

-- Los WhatsApp que salieron del saldo de packs: si no se mandan (se descartan, fallan o se borran
-- sin salir), la unidad vuelve al saldo. Los incluidos de Centro se cuentan aparte (sólo los que no
-- son del saldo y salieron o están por salir), así que la misma regla vale para los dos.
ALTER TABLE whatsapp_mensajes ADD COLUMN desde_saldo boolean NOT NULL DEFAULT false;
-- +goose StatementBegin
CREATE FUNCTION app.whatsapp_devolver_saldo() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
	IF TG_OP = 'DELETE' THEN
		IF OLD.desde_saldo AND OLD.estado = 'pendiente' THEN
			UPDATE negocios SET whatsapp_saldo = whatsapp_saldo + 1 WHERE id = OLD.negocio_id;
		END IF;
		RETURN OLD;
	END IF;
	IF NEW.desde_saldo AND OLD.estado NOT IN ('descartado', 'fallido') AND NEW.estado IN ('descartado', 'fallido') THEN
		UPDATE negocios SET whatsapp_saldo = whatsapp_saldo + 1 WHERE id = NEW.negocio_id;
	END IF;
	RETURN NEW;
END
$$;
-- +goose StatementEnd
CREATE TRIGGER whatsapp_devolver_saldo AFTER UPDATE OF estado OR DELETE ON whatsapp_mensajes
	FOR EACH ROW EXECUTE FUNCTION app.whatsapp_devolver_saldo();

-- La campanita avisa una vez por mes que los WhatsApp automáticos no salen por falta de cupo.
-- +goose StatementBegin
DO $$
DECLARE
	c record;
	tipos text[] := ARRAY['whatsapp_sin_cupo'];
BEGIN
	FOR c IN SELECT conname, pg_get_constraintdef(oid) AS def FROM pg_constraint
		WHERE conrelid = 'notificaciones'::regclass AND contype = 'c' AND pg_get_constraintdef(oid) ~ '\mtipo\M'
	LOOP
		tipos := tipos || ARRAY(SELECT m[1] FROM regexp_matches(c.def, '''([a-z_]+)''', 'g') AS m);
		EXECUTE format('ALTER TABLE notificaciones DROP CONSTRAINT %I', c.conname);
	END LOOP;
	EXECUTE format('ALTER TABLE notificaciones ADD CONSTRAINT notificaciones_tipo_check CHECK (tipo IN (%s))',
		(SELECT string_agg(quote_literal(t), ', ' ORDER BY t) FROM (SELECT DISTINCT unnest(tipos) AS t) x));
END
$$;
-- +goose StatementEnd

-- La purga de negocios cerrados archiva también lo que el negocio le pagó a Tengo Libre aparte de la
-- suscripción: los packs de WhatsApp y las diferencias por subir de plan.
ALTER TABLE auth.cobros_archivados DROP CONSTRAINT cobros_archivados_origen_check,
	ADD CONSTRAINT cobros_archivados_origen_check CHECK (origen IN ('sena', 'caja', 'suscripcion', 'whatsapp_pack', 'subida_plan'));
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION app.purgar_negocios(p_antes timestamptz, p_tanda int) RETURNS uuid[]
LANGUAGE plpgsql SECURITY DEFINER SET search_path = public, app AS $$
DECLARE
	antes text := current_setting('app.negocio_id', true);
	x uuid;
	borrados uuid[] := '{}';
BEGIN
	FOR x IN SELECT p.id FROM app.negocios_para_purgar(p_antes) p WHERE NOT p.retenido LIMIT p_tanda LOOP
		PERFORM set_config('app.negocio_id', x::text, true);
		INSERT INTO auth.cobros_archivados (negocio_id, origen, referencia, monto, devuelto, estado, medio, fecha)
			SELECT x, 'sena', concat_ws(' ', p.referencia, p.mp_orden, p.mp_transaccion, p.mp_pago::text), p.monto, p.devuelto,
				p.estado, 'mercadopago', p.creado_en
			FROM pagos p WHERE p.estado IN ('aprobado', 'devuelto');
		INSERT INTO auth.cobros_archivados (negocio_id, origen, monto, estado, medio, fecha)
			SELECT x, 'caja', c.total, c.estado, c.medio, coalesce(c.cobrado_en, c.creado_en) FROM caja_cobros c;
		INSERT INTO auth.cobros_archivados (negocio_id, origen, referencia, monto, estado, medio, fecha)
			SELECT x, 'suscripcion', concat_ws(' ', s.referencia, s.mp_id, s.apple_original, s.google_token), s.monto, s.estado,
				s.origen, coalesce(s.ultimo_cobro, s.creado_en)
			FROM suscripciones s WHERE s.ultimo_cobro IS NOT NULL;
		INSERT INTO auth.cobros_archivados (negocio_id, origen, referencia, monto, estado, medio, fecha)
			SELECT x, 'whatsapp_pack', concat_ws(' ', w.referencia, w.mp_orden), w.monto, w.estado, 'mercadopago',
				coalesce(w.acreditado_en, w.creado_en)
			FROM whatsapp_packs w WHERE w.estado IN ('acreditado', 'devuelto');
		INSERT INTO auth.cobros_archivados (negocio_id, origen, referencia, monto, estado, medio, fecha)
			SELECT x, 'subida_plan', concat_ws(' ', u.referencia, u.mp_orden), u.diferencia, u.estado, 'mercadopago',
				coalesce(u.acreditado_en, u.creado_en)
			FROM suscripcion_subidas u WHERE u.estado IN ('acreditado', 'devuelto', 'sin_aplicar');
		DELETE FROM turnos;
		DELETE FROM negocios WHERE id = x;
		borrados := borrados || x;
	END LOOP;
	PERFORM set_config('app.negocio_id', coalesce(antes, ''), true);
	RETURN borrados;
END
$$;
-- +goose StatementEnd

-- +goose StatementBegin
-- El plan que rige para el negocio fijado: durante la prueba, Equipo (o Centro si ya se cobró: una
-- suscripción a Centro que todavía no cobró nada, porque la prueba no se cobra, no da Centro gratis);
-- con una suscripción que vale, la suya; si no, sin_plan (el panel queda en sólo lectura y los topes
-- son los de Inicial). La condición de «vale» es la de suscripcion.vigenteSQL.
CREATE FUNCTION app.plan_vigente(p_ahora timestamptz) RETURNS text
LANGUAGE sql STABLE SET search_path = public, app AS $$
	SELECT CASE
		WHEN n.prueba_hasta > p_ahora AND NOT (n.plan = 'centro' AND v.centro_cobrado) THEN 'equipo'
		WHEN v.vale AND n.plan IN ('inicial', 'equipo', 'centro') THEN n.plan
		WHEN v.vale THEN 'inicial'
		ELSE 'sin_plan'
	END
	FROM negocios n, LATERAL (SELECT count(*) > 0 AS vale,
			coalesce(bool_or(s.plan = 'centro' AND s.ultimo_cobro IS NOT NULL), false) AS centro_cobrado
		FROM suscripciones s WHERE s.negocio_id = n.id AND (
		(s.estado = 'autorizada' AND (s.origen NOT IN ('apple', 'google') OR s.proximo_cobro > p_ahora - interval '1 day'))
		OR (s.estado = 'pausada' AND (s.gracia_hasta IS NULL OR s.gracia_hasta > p_ahora)))) v
	WHERE n.id = app.negocio_actual()
$$;
-- +goose StatementEnd

-- +goose StatementBegin
-- Cuántos profesionales activos toman turnos con el plan que rige (planes.Plan.Profesionales).
CREATE FUNCTION app.tope_profesionales(p_ahora timestamptz) RETURNS int
LANGUAGE sql STABLE SET search_path = public, app AS $$
	SELECT CASE app.plan_vigente(p_ahora)
		WHEN 'centro' THEN 25 + (SELECT profesionales_extra FROM negocios WHERE id = app.negocio_actual())
		WHEN 'equipo' THEN 12
		ELSE 5
	END
$$;
-- +goose StatementEnd

-- +goose StatementBegin
-- Las estadísticas del admin: el MRR cuenta el anual como un doceavo de lo que se cobró.
CREATE OR REPLACE FUNCTION app.admin_estadisticas(p_mes timestamptz, p_anterior timestamptz, p_ahora timestamptz,
	p_hoy timestamptz, p_manana timestamptz) RETURNS jsonb
LANGUAGE plpgsql SECURITY DEFINER SET search_path = public, app AS $$
DECLARE
	out jsonb;
	hoy bigint;
	web bigint;
	terminadas bigint;
	convertidas bigint;
BEGIN
	PERFORM set_config('app.plataforma', 'si', true);
	SELECT count(*), count(*) FILTER (WHERE origen = 'web') INTO hoy, web
		FROM turnos WHERE inicio >= p_hoy AND inicio < p_manana AND estado <> 'cancelado';
	-- Conversión: de las pruebas que terminaron en los últimos 90 días, cuántas llegaron a pagar.
	SELECT count(*), count(*) FILTER (WHERE EXISTS (SELECT 1 FROM suscripciones s WHERE s.negocio_id = n.id
			AND (s.estado IN ('autorizada', 'pausada') OR s.ultimo_cobro IS NOT NULL)))
		INTO terminadas, convertidas
		FROM negocios n WHERE n.prueba_hasta > p_ahora - interval '90 days' AND n.prueba_hasta <= p_ahora;
	SELECT jsonb_build_object(
		'negocios', (SELECT count(*) FROM negocios),
		'activos', (SELECT count(*) FROM negocios WHERE activo),
		'enPrueba', (SELECT count(*) FROM negocios WHERE activo AND prueba_hasta > p_ahora),
		'negociosDelMes', (SELECT count(*) FROM negocios WHERE creado_en >= p_mes),
		'personas', (SELECT count(DISTINCT clerk_id) FROM miembros WHERE activo),
		'turnosDelMes', (SELECT count(*) FROM turnos WHERE inicio >= p_mes AND estado <> 'cancelado'),
		'turnosMesAnterior', (SELECT count(*) FROM turnos WHERE inicio >= p_anterior AND inicio < p_mes AND estado <> 'cancelado'),
		'ingresosDelMes', (SELECT coalesce(sum(precio), 0) FROM turnos WHERE inicio >= p_mes AND estado = 'atendido'),
		'dominiosVerificados', (SELECT count(*) FROM dominios WHERE estado = 'verificado'),
		'recientes', coalesce((SELECT jsonb_agg(jsonb_build_object('id', r.id, 'slug', r.slug, 'nombre', r.nombre, 'creadoEn', r.creado_en)
			ORDER BY r.creado_en DESC) FROM (SELECT * FROM negocios ORDER BY creado_en DESC LIMIT 5) r), '[]'::jsonb),
		'turnosHoy', hoy,
		'porcentajeWeb', CASE WHEN hoy > 0 THEN round(100.0 * web / hoy)::int END,
		'suscripcionesPagas', (SELECT count(DISTINCT negocio_id) FROM suscripciones WHERE estado = 'autorizada'),
		'mrr', (SELECT coalesce(sum(CASE WHEN periodo = 'anual' THEN round(monto / 12.0) ELSE monto END), 0)::bigint FROM suscripciones WHERE estado = 'autorizada'),
		'pruebasPorVencer', (SELECT count(*) FROM negocios n WHERE n.activo AND n.prueba_hasta > p_ahora
			AND n.prueba_hasta <= p_ahora + interval '3 days'
			AND NOT EXISTS (SELECT 1 FROM suscripciones s WHERE s.negocio_id = n.id AND s.estado = 'autorizada')),
		'conversion', CASE WHEN terminadas > 0 THEN round(100.0 * convertidas / terminadas)::int END
	) INTO out;
	PERFORM set_config('app.plataforma', '', true);
	RETURN out;
END
$$;
-- +goose StatementEnd

-- +goose Down
DELETE FROM notificaciones WHERE tipo = 'whatsapp_sin_cupo';
-- +goose StatementBegin
DO $$
DECLARE
	c record;
	tipos text[] := '{}';
BEGIN
	FOR c IN SELECT conname, pg_get_constraintdef(oid) AS def FROM pg_constraint
		WHERE conrelid = 'notificaciones'::regclass AND contype = 'c' AND pg_get_constraintdef(oid) ~ '\mtipo\M'
	LOOP
		tipos := tipos || ARRAY(SELECT m[1] FROM regexp_matches(c.def, '''([a-z_]+)''', 'g') AS m WHERE m[1] <> 'whatsapp_sin_cupo');
		EXECUTE format('ALTER TABLE notificaciones DROP CONSTRAINT %I', c.conname);
	END LOOP;
	EXECUTE format('ALTER TABLE notificaciones ADD CONSTRAINT notificaciones_tipo_check CHECK (tipo IN (%s))',
		(SELECT string_agg(quote_literal(t), ', ' ORDER BY t) FROM (SELECT DISTINCT unnest(tipos) AS t) x));
END
$$;
-- +goose StatementEnd
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION app.admin_estadisticas(p_mes timestamptz, p_anterior timestamptz, p_ahora timestamptz,
	p_hoy timestamptz, p_manana timestamptz) RETURNS jsonb
LANGUAGE plpgsql SECURITY DEFINER SET search_path = public, app AS $$
DECLARE
	out jsonb;
	hoy bigint;
	web bigint;
	terminadas bigint;
	convertidas bigint;
BEGIN
	PERFORM set_config('app.plataforma', 'si', true);
	SELECT count(*), count(*) FILTER (WHERE origen = 'web') INTO hoy, web
		FROM turnos WHERE inicio >= p_hoy AND inicio < p_manana AND estado <> 'cancelado';
	-- Conversión: de las pruebas que terminaron en los últimos 90 días, cuántas llegaron a pagar.
	SELECT count(*), count(*) FILTER (WHERE EXISTS (SELECT 1 FROM suscripciones s WHERE s.negocio_id = n.id
			AND (s.estado IN ('autorizada', 'pausada') OR s.ultimo_cobro IS NOT NULL)))
		INTO terminadas, convertidas
		FROM negocios n WHERE n.prueba_hasta > p_ahora - interval '90 days' AND n.prueba_hasta <= p_ahora;
	SELECT jsonb_build_object(
		'negocios', (SELECT count(*) FROM negocios),
		'activos', (SELECT count(*) FROM negocios WHERE activo),
		'enPrueba', (SELECT count(*) FROM negocios WHERE activo AND prueba_hasta > p_ahora),
		'negociosDelMes', (SELECT count(*) FROM negocios WHERE creado_en >= p_mes),
		'personas', (SELECT count(DISTINCT clerk_id) FROM miembros WHERE activo),
		'turnosDelMes', (SELECT count(*) FROM turnos WHERE inicio >= p_mes AND estado <> 'cancelado'),
		'turnosMesAnterior', (SELECT count(*) FROM turnos WHERE inicio >= p_anterior AND inicio < p_mes AND estado <> 'cancelado'),
		'ingresosDelMes', (SELECT coalesce(sum(precio), 0) FROM turnos WHERE inicio >= p_mes AND estado = 'atendido'),
		'dominiosVerificados', (SELECT count(*) FROM dominios WHERE estado = 'verificado'),
		'recientes', coalesce((SELECT jsonb_agg(jsonb_build_object('id', r.id, 'slug', r.slug, 'nombre', r.nombre, 'creadoEn', r.creado_en)
			ORDER BY r.creado_en DESC) FROM (SELECT * FROM negocios ORDER BY creado_en DESC LIMIT 5) r), '[]'::jsonb),
		'turnosHoy', hoy,
		'porcentajeWeb', CASE WHEN hoy > 0 THEN round(100.0 * web / hoy)::int END,
		'suscripcionesPagas', (SELECT count(DISTINCT negocio_id) FROM suscripciones WHERE estado = 'autorizada'),
		'mrr', (SELECT coalesce(sum(monto), 0) FROM suscripciones WHERE estado = 'autorizada'),
		'pruebasPorVencer', (SELECT count(*) FROM negocios n WHERE n.activo AND n.prueba_hasta > p_ahora
			AND n.prueba_hasta <= p_ahora + interval '3 days'
			AND NOT EXISTS (SELECT 1 FROM suscripciones s WHERE s.negocio_id = n.id AND s.estado = 'autorizada')),
		'conversion', CASE WHEN terminadas > 0 THEN round(100.0 * convertidas / terminadas)::int END
	) INTO out;
	PERFORM set_config('app.plataforma', '', true);
	RETURN out;
END
$$;
-- +goose StatementEnd
DROP FUNCTION app.tope_profesionales(timestamptz);
DROP FUNCTION app.plan_vigente(timestamptz);
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION app.purgar_negocios(p_antes timestamptz, p_tanda int) RETURNS uuid[]
LANGUAGE plpgsql SECURITY DEFINER SET search_path = public, app AS $$
DECLARE
	antes text := current_setting('app.negocio_id', true);
	x uuid;
	borrados uuid[] := '{}';
BEGIN
	FOR x IN SELECT p.id FROM app.negocios_para_purgar(p_antes) p WHERE NOT p.retenido LIMIT p_tanda LOOP
		PERFORM set_config('app.negocio_id', x::text, true);
		INSERT INTO auth.cobros_archivados (negocio_id, origen, referencia, monto, devuelto, estado, medio, fecha)
			SELECT x, 'sena', concat_ws(' ', p.referencia, p.mp_orden, p.mp_transaccion, p.mp_pago::text), p.monto, p.devuelto,
				p.estado, 'mercadopago', p.creado_en
			FROM pagos p WHERE p.estado IN ('aprobado', 'devuelto');
		INSERT INTO auth.cobros_archivados (negocio_id, origen, monto, estado, medio, fecha)
			SELECT x, 'caja', c.total, c.estado, c.medio, coalesce(c.cobrado_en, c.creado_en) FROM caja_cobros c;
		INSERT INTO auth.cobros_archivados (negocio_id, origen, referencia, monto, estado, medio, fecha)
			SELECT x, 'suscripcion', concat_ws(' ', s.referencia, s.mp_id, s.apple_original, s.google_token), s.monto, s.estado,
				s.origen, coalesce(s.ultimo_cobro, s.creado_en)
			FROM suscripciones s WHERE s.ultimo_cobro IS NOT NULL;
		DELETE FROM turnos;
		DELETE FROM negocios WHERE id = x;
		borrados := borrados || x;
	END LOOP;
	PERFORM set_config('app.negocio_id', coalesce(antes, ''), true);
	RETURN borrados;
END
$$;
-- +goose StatementEnd
DELETE FROM auth.cobros_archivados WHERE origen IN ('whatsapp_pack', 'subida_plan');
ALTER TABLE auth.cobros_archivados DROP CONSTRAINT cobros_archivados_origen_check,
	ADD CONSTRAINT cobros_archivados_origen_check CHECK (origen IN ('sena', 'caja', 'suscripcion'));
DROP TRIGGER whatsapp_devolver_saldo ON whatsapp_mensajes;
DROP FUNCTION app.whatsapp_devolver_saldo();
ALTER TABLE whatsapp_mensajes DROP COLUMN desde_saldo;
DROP FUNCTION app.negocios_con_ordenes_pendientes(timestamptz);
DROP TABLE suscripcion_subidas;
DROP TABLE whatsapp_packs;
ALTER TABLE negocios NO FORCE ROW LEVEL SECURITY;
ALTER TABLE suscripciones NO FORCE ROW LEVEL SECURITY;
ALTER TABLE suscripciones DROP COLUMN periodo, DROP COLUMN profesionales_extra, DROP COLUMN plan_siguiente, DROP COLUMN extras_siguiente,
	DROP COLUMN cambiando_hasta, DROP CONSTRAINT suscripciones_plan_check;
UPDATE suscripciones SET plan = 'completo' WHERE plan IN ('equipo', 'centro');
ALTER TABLE suscripciones ADD CONSTRAINT suscripciones_plan_check CHECK (plan IN ('inicial', 'completo'));
ALTER TABLE negocios DROP COLUMN profesionales_extra, DROP COLUMN whatsapp_saldo, DROP CONSTRAINT negocios_plan_check,
	ALTER COLUMN prueba_hasta SET DEFAULT now() + interval '7 days';
UPDATE negocios SET plan = 'completo' WHERE plan IN ('equipo', 'centro');
ALTER TABLE negocios ADD CONSTRAINT negocios_plan_check CHECK (plan IN ('sin_plan', 'inicial', 'completo'));
ALTER TABLE suscripciones FORCE ROW LEVEL SECURITY;
ALTER TABLE negocios FORCE ROW LEVEL SECURITY;
