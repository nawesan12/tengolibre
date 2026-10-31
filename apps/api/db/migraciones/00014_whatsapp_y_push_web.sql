-- +goose Up
-- Propuesta 02 · Confirmación y recordatorios por WhatsApp, y Propuesta 10 · Avisos push en el
-- panel web.
--
-- WhatsApp: el día antes sale una plantilla con dos botones («Confirmo» / «Necesito cambiarlo») desde
-- el número de la plataforma; la respuesta llega por el webhook de Meta y cambia el turno en vivo. El
-- turno guarda de dónde vino la confirmación y si hay que llamar al cliente.
--
-- Push web: las suscripciones de Web Push del navegador, atadas a la sesión (como los teléfonos de
-- 00011), y qué tipos de la campanita le llegan a cada persona en cada negocio (también al iPhone y
-- a Android).

-- 1. El turno: de dónde vino la confirmación y si hay que llamar.
ALTER TABLE turnos
	-- whatsapp: tocó «Confirmo»; web: desde su link o pagando la seña; panel: lo confirmó el negocio.
	ADD COLUMN confirmado_por text CHECK (confirmado_por IN ('whatsapp', 'web', 'panel')),
	ADD COLUMN confirmado_en timestamptz,
	-- Tocó «Necesito cambiarlo».
	ADD COLUMN cambio_pedido_en timestamptz,
	-- Desde cuándo recepción tiene que llamarlo, y por qué: pidió cambiarlo, no contestó en las horas
	-- del negocio, o el WhatsApp no le llegó. NULL: no hay que llamar (o ya se llamó).
	ADD COLUMN llamar_desde timestamptz,
	ADD COLUMN llamar_motivo text CHECK (llamar_motivo IN ('cambio', 'sin_respuesta', 'no_llego')),
	ADD CHECK ((llamar_desde IS NULL) = (llamar_motivo IS NULL));
GRANT UPDATE (confirmado_por, confirmado_en, cambio_pedido_en, llamar_desde, llamar_motivo) ON turnos TO tengolibre_app;
CREATE INDEX turnos_llamar ON turnos (negocio_id, llamar_desde) WHERE llamar_desde IS NOT NULL;
-- Lo confirmado hasta hoy lo confirmó el panel (o el pago de la seña): no se sabe cuál, queda vacío.

-- +goose StatementBegin
-- Lo decide un trigger para no depender de cada UPDATE (como app.cuidar_email): al entrar en
-- confirmado, si el UPDATE no dijo de dónde (WhatsApp, el link y el pago de la seña lo dicen), fue el
-- panel. Al volver a pendiente o cancelarse, se borra. Un turno que se mueve de horario, se cancela o
-- se cierra ya no está para llamar.
CREATE FUNCTION app.cuidar_confirmacion() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
	IF NEW.estado = 'confirmado' AND (TG_OP = 'INSERT' OR OLD.estado <> 'confirmado') THEN
		IF TG_OP = 'INSERT' AND NEW.confirmado_por IS NULL
			OR TG_OP = 'UPDATE' AND NEW.confirmado_por IS NOT DISTINCT FROM OLD.confirmado_por THEN
			NEW.confirmado_por := CASE WHEN coalesce(current_setting('app.origen', true), '') = 'web' THEN 'web' ELSE 'panel' END;
		END IF;
		IF TG_OP = 'INSERT' OR NEW.confirmado_en IS NOT DISTINCT FROM OLD.confirmado_en THEN
			NEW.confirmado_en := now();
		END IF;
	ELSIF TG_OP = 'UPDATE' AND NEW.estado IN ('pendiente_pago', 'pendiente', 'cancelado') AND OLD.estado = 'confirmado' THEN
		NEW.confirmado_por := NULL;
		NEW.confirmado_en := NULL;
	END IF;
	IF TG_OP = 'UPDATE' AND NEW.inicio IS DISTINCT FROM OLD.inicio THEN
		NEW.cambio_pedido_en := NULL;
	END IF;
	IF TG_OP = 'UPDATE' AND (NEW.inicio IS DISTINCT FROM OLD.inicio OR NEW.estado IN ('cancelado', 'atendido', 'ausente')) THEN
		NEW.llamar_desde := NULL;
		NEW.llamar_motivo := NULL;
	END IF;
	RETURN NEW;
END
$$;
-- +goose StatementEnd
CREATE TRIGGER cuidar_confirmacion BEFORE INSERT OR UPDATE ON turnos FOR EACH ROW EXECUTE FUNCTION app.cuidar_confirmacion();

-- 2. Ajustes del negocio (Configuración › Avisos).
ALTER TABLE ajustes
	ADD COLUMN whatsapp_confirmacion boolean NOT NULL DEFAULT false,
	-- A qué hora (la del negocio) del día anterior sale la confirmación.
	ADD COLUMN whatsapp_hora time NOT NULL DEFAULT '10:00' CHECK (whatsapp_hora BETWEEN '07:00' AND '20:00'),
	-- Si no contesta en tantas horas, se lo marca a recepción para llamar.
	ADD COLUMN whatsapp_sin_respuesta_horas int NOT NULL DEFAULT 4 CHECK (whatsapp_sin_respuesta_horas BETWEEN 1 AND 12),
	-- El recordatorio de 24 horas antes también por WhatsApp (además del mail).
	ADD COLUMN whatsapp_recordatorio boolean NOT NULL DEFAULT false;
GRANT UPDATE (whatsapp_confirmacion, whatsapp_hora, whatsapp_sin_respuesta_horas, whatsapp_recordatorio) ON ajustes TO tengolibre_app;

-- 3. Tipos nuevos de la campanita: confirmó por WhatsApp, pidió cambiarlo, no contestó. La lista se
-- arma con la que haya (otras migraciones también suman tipos), así el orden no importa.
-- +goose StatementBegin
DO $$
DECLARE
	c record;
	tipos text[] := ARRAY['confirmacion', 'cambio_pedido', 'sin_respuesta'];
BEGIN
	FOR c IN SELECT conname, pg_get_constraintdef(oid) AS def FROM pg_constraint
		WHERE conrelid = 'notificaciones'::regclass AND contype = 'c' AND pg_get_constraintdef(oid) ~ '\mtipo\M'
	LOOP
		tipos := tipos || ARRAY(SELECT m[1] FROM regexp_matches(c.def, '''([a-z_]+)''', 'g') AS m);
		EXECUTE format('ALTER TABLE notificaciones DROP CONSTRAINT %I', c.conname);
	END LOOP;
	-- Con IN (…), el catálogo lo guarda como ARRAY['…'::text, …], igual que el CHECK de 00005.
	EXECUTE format('ALTER TABLE notificaciones ADD CONSTRAINT notificaciones_tipo_check CHECK (tipo IN (%s))',
		(SELECT string_agg(quote_literal(t), ', ' ORDER BY t) FROM (SELECT DISTINCT unnest(tipos) AS t) x));
END
$$;
-- +goose StatementEnd

-- 4. La cola de WhatsApp (como la de mails): confirmaciones, recordatorios y las respuestas a lo que
-- tocó el cliente. El mensaje se arma al mandarlo, con los datos de ese momento. El token va en el
-- payload de los botones: es lo que vuelve en el webhook y dice de qué turno es.
CREATE TABLE whatsapp_mensajes (
	id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
	negocio_id uuid NOT NULL REFERENCES negocios (id) ON DELETE CASCADE,
	turno_id uuid REFERENCES turnos (id) ON DELETE CASCADE,
	tipo text NOT NULL CHECK (tipo IN ('confirmacion', 'recordatorio', 'respuesta')),
	-- E.164. Vacío sólo si se descartó porque el teléfono no servía.
	para text NOT NULL CHECK (para = '' OR para ~ '^\+[1-9][0-9]{7,14}$'),
	token text NOT NULL UNIQUE CHECK (token ~ '^[A-Za-z0-9_-]{16,64}$'),
	clave text CHECK (length(clave) <= 200),
	datos jsonb NOT NULL DEFAULT '{}',
	estado text NOT NULL DEFAULT 'pendiente' CHECK (estado IN ('pendiente', 'enviado', 'descartado', 'fallido')),
	intentos int NOT NULL DEFAULT 0,
	proximo_intento timestamptz NOT NULL,
	ultimo_error text CHECK (length(ultimo_error) <= 1000),
	-- El id que da Meta (wamid): con él llegan los estados (entregado, leído, falló).
	wamid text UNIQUE CHECK (length(wamid) <= 200),
	enviado_en timestamptz,
	entregado_en timestamptz,
	leido_en timestamptz,
	-- Lo que tocó el cliente, el wamid de su respuesta (Meta repite webhooks) y cuándo.
	respuesta text CHECK (respuesta IN ('confirmo', 'cambiar')),
	respuesta_id text CHECK (length(respuesta_id) <= 200),
	respondido_en timestamptz,
	-- Cuándo se dio por no contestado (y se marcó para llamar).
	sin_respuesta_en timestamptz,
	creado_en timestamptz NOT NULL DEFAULT now(),
	UNIQUE (negocio_id, clave)
);
CREATE INDEX whatsapp_pendientes ON whatsapp_mensajes (proximo_intento) WHERE estado = 'pendiente';
CREATE INDEX whatsapp_negocio ON whatsapp_mensajes (negocio_id, creado_en DESC);
CREATE INDEX whatsapp_turno ON whatsapp_mensajes (turno_id);
CREATE INDEX whatsapp_sin_respuesta ON whatsapp_mensajes (negocio_id, enviado_en)
	WHERE tipo = 'confirmacion' AND estado = 'enviado' AND respuesta IS NULL AND sin_respuesta_en IS NULL;
CREATE INDEX whatsapp_creado ON whatsapp_mensajes (creado_en);
SELECT app.aplicar_rls('whatsapp_mensajes');
GRANT SELECT, INSERT ON whatsapp_mensajes TO tengolibre_app;
GRANT UPDATE (para, estado, intentos, proximo_intento, ultimo_error, wamid, enviado_en, entregado_en, leido_en,
	respuesta, respuesta_id, respondido_en, sin_respuesta_en) ON whatsapp_mensajes TO tengolibre_app;

CREATE POLICY barrido ON whatsapp_mensajes FOR SELECT TO tengolibre_migrador
	USING (app.barrido() = 'whatsapp');
-- +goose StatementBegin
-- Los mensajes que toca mandar, de todos los negocios (como app.avisos_pendientes).
CREATE FUNCTION app.whatsapp_pendientes(p_ahora timestamptz, p_limite int)
RETURNS TABLE (mensaje_id uuid, negocio_id uuid)
LANGUAGE plpgsql SECURITY DEFINER SET search_path = public, app AS $$
BEGIN
	PERFORM set_config('app.barrido', 'whatsapp', true);
	RETURN QUERY SELECT m.id, m.negocio_id FROM whatsapp_mensajes m
		WHERE m.estado = 'pendiente' AND m.proximo_intento <= p_ahora
		ORDER BY m.proximo_intento LIMIT p_limite;
	PERFORM set_config('app.barrido', '', true);
END
$$;
-- +goose StatementEnd
REVOKE ALL ON FUNCTION app.whatsapp_pendientes(timestamptz, int) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION app.whatsapp_pendientes(timestamptz, int) TO tengolibre_app;

-- El webhook de Meta no trae negocio: del token del botón (o del wamid del mensaje respondido o del
-- estado) al negocio, mirando sólo esa fila.
CREATE POLICY por_token ON whatsapp_mensajes FOR SELECT TO tengolibre_migrador
	USING (token = current_setting('app.wa_token', true) OR wamid = current_setting('app.wa_id', true));
-- +goose StatementBegin
CREATE FUNCTION app.negocio_de_whatsapp(p_token text, p_wamid text) RETURNS uuid
LANGUAGE plpgsql SECURITY DEFINER SET search_path = public, app AS $$
DECLARE
	n uuid;
BEGIN
	PERFORM set_config('app.wa_token', coalesce(p_token, ''), true);
	PERFORM set_config('app.wa_id', coalesce(p_wamid, ''), true);
	SELECT negocio_id INTO n FROM whatsapp_mensajes
		WHERE (p_token <> '' AND token = p_token) OR (p_wamid <> '' AND wamid = p_wamid) LIMIT 1;
	PERFORM set_config('app.wa_token', '', true);
	PERFORM set_config('app.wa_id', '', true);
	RETURN n;
END
$$;
-- +goose StatementEnd
REVOKE ALL ON FUNCTION app.negocio_de_whatsapp(text, text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION app.negocio_de_whatsapp(text, text) TO tengolibre_app;

-- La purga de la cola de WhatsApp (app.purgar es de 00005 y no la conoce): lo resuelto y viejo.
CREATE POLICY purga ON whatsapp_mensajes TO tengolibre_migrador USING (app.barrido() = 'purga_whatsapp');
-- +goose StatementBegin
CREATE FUNCTION app.purgar_whatsapp(p_antes timestamptz, p_tanda int) RETURNS bigint
LANGUAGE plpgsql SECURITY DEFINER SET search_path = public, app AS $$
DECLARE
	n bigint;
BEGIN
	PERFORM set_config('app.barrido', 'purga_whatsapp', true);
	DELETE FROM whatsapp_mensajes WHERE id IN (SELECT m.id FROM whatsapp_mensajes m
		WHERE m.creado_en < p_antes AND m.estado <> 'pendiente' LIMIT p_tanda);
	GET DIAGNOSTICS n = ROW_COUNT;
	PERFORM set_config('app.barrido', '', true);
	RETURN n;
END
$$;
-- +goose StatementEnd
REVOKE ALL ON FUNCTION app.purgar_whatsapp(timestamptz, int) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION app.purgar_whatsapp(timestamptz, int) TO tengolibre_app;

-- 5. Push web: cada navegador donde alguien activó los avisos del panel, atado a la sesión con que
-- entró (cerrarla o borrarla se lleva la suscripción, como auth.dispositivos).
CREATE TABLE auth.suscripciones_web (
	id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
	clerk_id text NOT NULL CHECK (clerk_id ~ '^user_[A-Za-z0-9]{10,64}$'),
	sesion_hash bytea NOT NULL REFERENCES auth.sesiones (hash) ON DELETE CASCADE,
	-- La URL del servicio de push (FCM, Apple, Mozilla, Windows): webpush.Normalizar la valida.
	endpoint text NOT NULL UNIQUE CHECK (endpoint ~ '^https://' AND length(endpoint) <= 1000),
	-- Las claves del navegador en base64url sin relleno: el punto P-256 (65 bytes) y el secreto (16).
	p256dh text NOT NULL CHECK (p256dh ~ '^[A-Za-z0-9_-]{87}$'),
	auth text NOT NULL CHECK (auth ~ '^[A-Za-z0-9_-]{22}$'),
	-- Cómo se ve en la lista («Chrome · Mac»).
	aparato text NOT NULL DEFAULT '' CHECK (length(aparato) <= 60),
	creado_en timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX suscripciones_web_clerk ON auth.suscripciones_web (clerk_id);
CREATE INDEX suscripciones_web_sesion ON auth.suscripciones_web (sesion_hash);
GRANT SELECT, INSERT, DELETE ON auth.suscripciones_web TO tengolibre_app;

-- 6. Qué tipos de la campanita le llegan por push a cada persona en cada negocio (web, iPhone y
-- Android). Se guarda lo apagado: sin fila, valen los de su rol; un tipo nuevo llega prendido.
CREATE TABLE push_preferencias (
	negocio_id uuid NOT NULL REFERENCES negocios (id) ON DELETE CASCADE,
	clerk_id text NOT NULL CHECK (clerk_id ~ '^user_[A-Za-z0-9]{10,64}$'),
	apagados text[] NOT NULL DEFAULT '{}' CHECK (cardinality(apagados) <= 40),
	actualizado_en timestamptz NOT NULL DEFAULT now(),
	PRIMARY KEY (negocio_id, clerk_id)
);
SELECT app.aplicar_rls('push_preferencias');
GRANT SELECT, INSERT, DELETE ON push_preferencias TO tengolibre_app;
GRANT UPDATE (apagados, actualizado_en) ON push_preferencias TO tengolibre_app;

-- +goose Down
DROP TABLE push_preferencias;
DROP TABLE auth.suscripciones_web;
DROP FUNCTION app.purgar_whatsapp(timestamptz, int);
DROP FUNCTION app.negocio_de_whatsapp(text, text);
DROP FUNCTION app.whatsapp_pendientes(timestamptz, int);
DROP TABLE whatsapp_mensajes;
DELETE FROM notificaciones WHERE tipo IN ('confirmacion', 'cambio_pedido', 'sin_respuesta');
-- +goose StatementBegin
DO $$
DECLARE
	c record;
	tipos text[] := '{}';
BEGIN
	FOR c IN SELECT conname, pg_get_constraintdef(oid) AS def FROM pg_constraint
		WHERE conrelid = 'notificaciones'::regclass AND contype = 'c' AND pg_get_constraintdef(oid) ~ '\mtipo\M'
	LOOP
		tipos := tipos || ARRAY(SELECT m[1] FROM regexp_matches(c.def, '''([a-z_]+)''', 'g') AS m);
		EXECUTE format('ALTER TABLE notificaciones DROP CONSTRAINT %I', c.conname);
	END LOOP;
	EXECUTE format('ALTER TABLE notificaciones ADD CONSTRAINT notificaciones_tipo_check CHECK (tipo IN (%s))',
		(SELECT string_agg(quote_literal(t), ', ' ORDER BY t) FROM (SELECT DISTINCT unnest(tipos) AS t) x
			WHERE t NOT IN ('confirmacion', 'cambio_pedido', 'sin_respuesta')));
END
$$;
-- +goose StatementEnd
ALTER TABLE ajustes DROP COLUMN whatsapp_confirmacion, DROP COLUMN whatsapp_hora, DROP COLUMN whatsapp_sin_respuesta_horas,
	DROP COLUMN whatsapp_recordatorio;
DROP TRIGGER cuidar_confirmacion ON turnos;
DROP FUNCTION app.cuidar_confirmacion();
DROP INDEX turnos_llamar;
ALTER TABLE turnos DROP COLUMN confirmado_por, DROP COLUMN confirmado_en, DROP COLUMN cambio_pedido_en,
	DROP COLUMN llamar_desde, DROP COLUMN llamar_motivo;
