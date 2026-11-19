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
	ADD COLUMN profesionales_extra int NOT NULL DEFAULT 0 CHECK (profesionales_extra BETWEEN 0 AND 100);
GRANT UPDATE (periodo, profesionales_extra) ON suscripciones TO tengolibre_app;

ALTER TABLE suscripciones FORCE ROW LEVEL SECURITY;
ALTER TABLE negocios FORCE ROW LEVEL SECURITY;

CREATE TABLE whatsapp_packs (
	id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
	negocio_id uuid NOT NULL REFERENCES negocios (id) ON DELETE CASCADE,
	-- external_reference de la orden: empieza con el negocio.
	referencia text NOT NULL UNIQUE CHECK (length(referencia) <= 120),
	cantidad int NOT NULL CHECK (cantidad > 0),
	monto bigint NOT NULL CHECK (monto > 0),
	estado text NOT NULL DEFAULT 'pendiente' CHECK (estado IN ('pendiente', 'acreditado', 'cancelado')),
	mp_orden text UNIQUE CHECK (length(mp_orden) <= 64),
	link text CHECK (length(link) <= 2048),
	creado_en timestamptz NOT NULL DEFAULT now(),
	acreditado_en timestamptz
);
CREATE INDEX whatsapp_packs_negocio ON whatsapp_packs (negocio_id, creado_en DESC);
SELECT app.aplicar_rls('whatsapp_packs');
GRANT SELECT, INSERT ON whatsapp_packs TO tengolibre_app;
GRANT UPDATE (estado, mp_orden, link, acreditado_en) ON whatsapp_packs TO tengolibre_app;

-- +goose StatementBegin
-- El plan que rige para el negocio fijado: durante la prueba, Equipo (o Centro si ya lo pagó); con
-- una suscripción que vale, la suya; si no, sin_plan (el panel queda en sólo lectura y los topes son
-- los de Inicial). La condición de «vale» es la de suscripcion.vigenteSQL.
CREATE FUNCTION app.plan_vigente(p_ahora timestamptz) RETURNS text
LANGUAGE sql STABLE SET search_path = public, app AS $$
	SELECT CASE
		WHEN n.prueba_hasta > p_ahora AND NOT (v.vale AND n.plan = 'centro') THEN 'equipo'
		WHEN v.vale AND n.plan IN ('inicial', 'equipo', 'centro') THEN n.plan
		WHEN v.vale THEN 'inicial'
		ELSE 'sin_plan'
	END
	FROM negocios n, LATERAL (SELECT EXISTS (SELECT 1 FROM suscripciones s WHERE s.negocio_id = n.id AND (
		(s.estado = 'autorizada' AND (s.origen NOT IN ('apple', 'google') OR s.proximo_cobro > p_ahora - interval '1 day'))
		OR (s.estado = 'pausada' AND (s.gracia_hasta IS NULL OR s.gracia_hasta > p_ahora)))) AS vale) v
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
DROP TABLE whatsapp_packs;
ALTER TABLE negocios NO FORCE ROW LEVEL SECURITY;
ALTER TABLE suscripciones NO FORCE ROW LEVEL SECURITY;
ALTER TABLE suscripciones DROP COLUMN periodo, DROP COLUMN profesionales_extra, DROP CONSTRAINT suscripciones_plan_check;
UPDATE suscripciones SET plan = 'completo' WHERE plan IN ('equipo', 'centro');
ALTER TABLE suscripciones ADD CONSTRAINT suscripciones_plan_check CHECK (plan IN ('inicial', 'completo'));
ALTER TABLE negocios DROP COLUMN profesionales_extra, DROP COLUMN whatsapp_saldo, DROP CONSTRAINT negocios_plan_check,
	ALTER COLUMN prueba_hasta SET DEFAULT now() + interval '7 days';
UPDATE negocios SET plan = 'completo' WHERE plan IN ('equipo', 'centro');
ALTER TABLE negocios ADD CONSTRAINT negocios_plan_check CHECK (plan IN ('sin_plan', 'inicial', 'completo'));
ALTER TABLE suscripciones FORCE ROW LEVEL SECURITY;
ALTER TABLE negocios FORCE ROW LEVEL SECURITY;
