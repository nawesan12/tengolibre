-- +goose Up
-- Revisión del comité (18/11): lo que necesita la base.
--
-- 1. ajustes.ia_activa: la IA (Anthropic, un proveedor nuevo con transferencia a EE. UU.) la prende la
--    dueña de cada negocio. Hasta entonces el asistente contesta sólo con el intérprete propio.
-- 2. asistente_propuestas: al resolverse se vacían los argumentos (traen los datos del cliente).
-- 3. notificaciones.profesional_id: de quién es una notificación del equipo (días libres), para que un
--    profesional vea en la campanita sólo lo suyo.
-- 4. Links anteriores: mientras el negocio que lo dejó siga activo, un link anterior no se libera
--    (los QR impresos duran más de 90 días). Para no acaparar, cada negocio retiene sus últimos 5:
--    con el sexto, el más viejo queda libre. De un negocio inactivo, se libera a los 90 días. El alta
--    de un negocio con un slug que fue link anterior (ya liberado) borra esa fila.
-- 5. Acento: el DEFAULT cian (#22d3ee) no era de ninguna paleta y casi no se leía sobre crema: pasa
--    al del handoff (#f0a98c) y los que lo tenían por defecto se acomodan a su color principal.
-- 6. Purga: suma asistente_propuestas (30 días), mcp_registro (180 días) y las sesiones vencidas
--    hace más de 7 días. suprimir_cliente también limpia lo del asistente y del MCP.

-- 1
ALTER TABLE ajustes ADD COLUMN ia_activa boolean NOT NULL DEFAULT false;
GRANT UPDATE (ia_activa) ON ajustes TO tengolibre_app;

-- 2
GRANT UPDATE (argumentos) ON asistente_propuestas TO tengolibre_app;

-- 3
ALTER TABLE notificaciones ADD COLUMN profesional_id uuid;
CREATE INDEX notificaciones_profesional ON notificaciones (profesional_id) WHERE profesional_id IS NOT NULL;

-- 4
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION app.estado_slug(p_slug text, p_negocio uuid) RETURNS text
LANGUAGE plpgsql SECURITY DEFINER SET search_path = public, app AS $$
DECLARE
	actual uuid;
	anterior uuid;
	dejado timestamptz;
	activo boolean;
BEGIN
	PERFORM set_config('app.slug', p_slug, true);
	SELECT id INTO actual FROM negocios WHERE slug = p_slug;
	SELECT s.negocio_id, s.dejado_en, g.activo INTO anterior, dejado, activo
		FROM slugs_anteriores s JOIN negocios g ON g.id = s.negocio_id WHERE s.slug = p_slug;
	PERFORM set_config('app.slug', '', true);
	IF actual IS NOT NULL THEN
		RETURN CASE WHEN actual = p_negocio THEN 'propia' ELSE 'tomada' END;
	END IF;
	IF anterior IS NOT NULL AND anterior IS DISTINCT FROM p_negocio AND (activo OR dejado > now() - interval '90 days') THEN
		RETURN 'tomada';
	END IF;
	RETURN 'libre';
END
$$;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION app.cambiar_slug(p_nuevo text) RETURNS text
LANGUAGE plpgsql SECURITY DEFINER SET search_path = public, app AS $$
DECLARE
	yo uuid := app.negocio_actual();
	viejo text;
	cambios int;
	sobra text;
BEGIN
	IF yo IS NULL THEN
		RAISE EXCEPTION 'sin negocio' USING ERRCODE = '42501';
	END IF;
	-- Dos negocios que piden el mismo slug a la vez: uno espera al otro.
	PERFORM pg_advisory_xact_lock(hashtext('slug:' || p_nuevo));
	SELECT slug INTO viejo FROM negocios WHERE id = yo FOR UPDATE;
	IF viejo = p_nuevo THEN
		RETURN 'igual';
	END IF;
	IF app.estado_slug(p_nuevo, yo) <> 'libre' THEN
		RETURN 'tomada';
	END IF;
	SELECT count(*) INTO cambios FROM cambios_de_link WHERE negocio_id = yo AND creado_en > now() - interval '30 days';
	IF cambios >= 5 THEN
		RETURN 'tope';
	END IF;
	-- El nuevo deja de ser un link anterior (propio, o de otro ya liberado).
	PERFORM set_config('app.slug', p_nuevo, true);
	DELETE FROM slugs_anteriores WHERE slug = p_nuevo;
	-- El viejo queda apuntando acá (si ya era anterior de otro, ahora es de este negocio).
	PERFORM set_config('app.slug', viejo, true);
	INSERT INTO slugs_anteriores (slug, negocio_id) VALUES (viejo, yo)
		ON CONFLICT (slug) DO UPDATE SET negocio_id = excluded.negocio_id, dejado_en = now();
	PERFORM set_config('app.slug', '', true);
	-- Se retienen los últimos 5: el sexto libera el más viejo.
	FOR sobra IN SELECT a.slug FROM slugs_anteriores a WHERE a.negocio_id = yo ORDER BY a.dejado_en DESC OFFSET 5 LOOP
		PERFORM set_config('app.slug', sobra, true);
		DELETE FROM slugs_anteriores WHERE slug = sobra;
	END LOOP;
	PERFORM set_config('app.slug', '', true);
	UPDATE negocios SET slug = p_nuevo WHERE id = yo;
	INSERT INTO cambios_de_link (negocio_id, de, a) VALUES (yo, viejo, p_nuevo);
	RETURN 'ok';
END
$$;
-- +goose StatementEnd

-- +goose StatementBegin
-- Para el alta: con el slug en fila (advisory lock hasta el final de la transacción), dice si está
-- libre y, si fue el link anterior de otro ya liberado, borra esa fila (si no, el negocio viejo lo
-- seguía mostrando y, con el nuevo inactivo, negocio_publico lo volvía a resolver al viejo).
CREATE FUNCTION app.tomar_slug_para_alta(p_slug text) RETURNS text
LANGUAGE plpgsql SECURITY DEFINER SET search_path = public, app AS $$
DECLARE
	e text;
BEGIN
	PERFORM pg_advisory_xact_lock(hashtext('slug:' || p_slug));
	e := app.estado_slug(p_slug, NULL);
	IF e = 'libre' THEN
		PERFORM set_config('app.slug', p_slug, true);
		DELETE FROM slugs_anteriores WHERE slug = p_slug;
		PERFORM set_config('app.slug', '', true);
	END IF;
	RETURN e;
END
$$;
-- +goose StatementEnd
REVOKE ALL ON FUNCTION app.tomar_slug_para_alta(text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION app.tomar_slug_para_alta(text) TO tengolibre_app;

-- 5
ALTER TABLE negocios ALTER COLUMN color_acento SET DEFAULT '#f0a98c';
ALTER TABLE negocios NO FORCE ROW LEVEL SECURITY;
UPDATE negocios SET color_acento = CASE color_primario
		WHEN '#7a3b5e' THEN '#e9a6b8'
		WHEN '#183c44' THEN '#f0a98c'
		WHEN '#1f2a24' THEN '#d9b26a'
		WHEN '#b4532a' THEN '#f2c38b'
		WHEN '#3d5a40' THEN '#e3b48e'
		WHEN '#1d3557' THEN '#f4b183'
		ELSE '#f0a98c' END
	WHERE color_acento = '#22d3ee';
ALTER TABLE negocios FORCE ROW LEVEL SECURITY;

-- 6
CREATE POLICY purga ON asistente_propuestas TO tengolibre_migrador USING (app.barrido() = 'purga');
CREATE POLICY purga ON mcp_registro TO tengolibre_migrador USING (app.barrido() = 'purga');

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION app.purgar(p_avisos timestamptz, p_auditoria timestamptz, p_vencidos timestamptz, p_tanda int)
RETURNS TABLE (tabla text, borradas bigint)
LANGUAGE plpgsql SECURITY DEFINER SET search_path = public, app AS $$
DECLARE
	n bigint;
BEGIN
	PERFORM set_config('app.barrido', 'purga', true);
	DELETE FROM avisos WHERE id IN (SELECT a.id FROM avisos a WHERE a.creado_en < p_avisos AND a.estado <> 'pendiente' LIMIT p_tanda);
	GET DIAGNOSTICS n = ROW_COUNT;
	tabla := 'avisos'; borradas := n; RETURN NEXT;
	DELETE FROM notificaciones WHERE id IN (SELECT x.id FROM notificaciones x WHERE x.creado_en < p_avisos LIMIT p_tanda);
	GET DIAGNOSTICS n = ROW_COUNT;
	tabla := 'notificaciones'; borradas := n; RETURN NEXT;
	DELETE FROM auditoria WHERE id IN (SELECT x.id FROM auditoria x WHERE x.creado_en < p_auditoria LIMIT p_tanda);
	GET DIAGNOSTICS n = ROW_COUNT;
	tabla := 'auditoria'; borradas := n; RETURN NEXT;
	DELETE FROM resena_tokens WHERE token IN (SELECT x.token FROM resena_tokens x WHERE x.vence_en < p_vencidos LIMIT p_tanda);
	GET DIAGNOSTICS n = ROW_COUNT;
	tabla := 'resena_tokens'; borradas := n; RETURN NEXT;
	DELETE FROM invitaciones WHERE id IN (SELECT x.id FROM invitaciones x WHERE x.vence_en < p_vencidos LIMIT p_tanda);
	GET DIAGNOSTICS n = ROW_COUNT;
	tabla := 'invitaciones'; borradas := n; RETURN NEXT;
	DELETE FROM espera WHERE id IN (SELECT x.id FROM espera x WHERE x.hasta < p_vencidos::date LIMIT p_tanda);
	GET DIAGNOSTICS n = ROW_COUNT;
	tabla := 'espera'; borradas := n; RETURN NEXT;
	-- p_vencidos es hoy menos 30 días.
	-- Las propuestas del asistente traen el nombre del cliente en el resumen: 30 días.
	DELETE FROM asistente_propuestas WHERE id IN (SELECT x.id FROM asistente_propuestas x
		WHERE x.creada_en < p_vencidos LIMIT p_tanda);
	GET DIAGNOSTICS n = ROW_COUNT;
	tabla := 'asistente_propuestas'; borradas := n; RETURN NEXT;
	-- Lo que hizo cada asistente conectado por MCP: 180 días.
	DELETE FROM mcp_registro WHERE id IN (SELECT x.id FROM mcp_registro x WHERE x.creado_en < p_vencidos - interval '150 days' LIMIT p_tanda);
	GET DIAGNOSTICS n = ROW_COUNT;
	tabla := 'mcp_registro'; borradas := n; RETURN NEXT;
	PERFORM set_config('app.barrido', '', true);
	-- Sesiones vencidas hace más de una semana (antes sólo se borraban al volver a entrar).
	DELETE FROM auth.sesiones WHERE hash IN (SELECT x.hash FROM auth.sesiones x WHERE x.vence_en < p_vencidos + interval '23 days' LIMIT p_tanda);
	GET DIAGNOSTICS n = ROW_COUNT;
	tabla := 'sesiones'; borradas := n; RETURN NEXT;
END
$$;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION app.suprimir_cliente(p_cliente uuid, p_ahora timestamptz) RETURNS text
LANGUAGE plpgsql SECURITY DEFINER SET search_path = public, app AS $$
DECLARE
	c clientes;
	ts uuid[];
BEGIN
	SELECT * INTO c FROM clientes WHERE id = p_cliente AND negocio_id = app.negocio_actual() AND suprimido_en IS NULL FOR UPDATE;
	IF NOT FOUND THEN
		RETURN NULL;
	END IF;
	IF EXISTS (SELECT 1 FROM turnos WHERE cliente_id = c.id AND estado IN ('pendiente_pago', 'pendiente', 'confirmado')
		AND fin > p_ahora AND (estado <> 'pendiente_pago' OR vence_en > p_ahora)) THEN
		RETURN 'con_turnos';
	END IF;
	SELECT coalesce(array_agg(id), '{}') INTO ts FROM turnos WHERE cliente_id = c.id;

	DELETE FROM espera WHERE (c.telefono <> '' AND telefono = c.telefono) OR (c.email <> '' AND lower(email) = lower(c.email));
	DELETE FROM resenas WHERE cliente_id = c.id OR turno_id = ANY (ts);
	DELETE FROM avisos WHERE estado = 'pendiente' AND (turno_id = ANY (ts) OR (c.email <> '' AND lower(para) = lower(c.email)));
	UPDATE avisos SET para = '', datos = '{}' WHERE turno_id = ANY (ts) OR (c.email <> '' AND lower(para) = lower(c.email));
	UPDATE auditoria SET detalle = NULL WHERE entidad = 'cliente' AND entidad_id = c.id::text;
	-- Lo del asistente y de los asistentes conectados por MCP (00035): el nombre en los resúmenes y
	-- los argumentos que lo mencionan.
	UPDATE asistente_propuestas SET resumen = replace(resumen, c.nombre, 'Cliente borrado'), argumentos = '{}'
		WHERE (c.nombre <> '' AND position(c.nombre IN resumen) > 0) OR argumentos::text LIKE '%' || c.id::text || '%'
			OR (c.telefono <> '' AND argumentos::text LIKE '%' || c.telefono || '%');
	UPDATE mcp_registro SET resumen = replace(resumen, c.nombre, 'Cliente borrado'), argumentos = NULL
		WHERE (c.nombre <> '' AND position(c.nombre IN resumen) > 0) OR argumentos::text LIKE '%' || c.id::text || '%'
			OR (c.telefono <> '' AND argumentos::text LIKE '%' || c.telefono || '%');

	IF cardinality(ts) = 0 THEN
		-- Lo demás se va en cascada (etiquetas, notas, penalidades, membresías, referidos,
		-- preferencias, turnos fijos).
		DELETE FROM clientes WHERE id = c.id;
		RETURN 'borrado';
	END IF;

	DELETE FROM whatsapp_mensajes WHERE turno_id = ANY (ts) AND estado = 'pendiente';
	UPDATE whatsapp_mensajes SET para = '', datos = '{}' WHERE turno_id = ANY (ts);
	DELETE FROM turno_dispositivos WHERE turno_id = ANY (ts);
	DELETE FROM turno_actividades WHERE turno_id = ANY (ts);
	DELETE FROM resena_tokens WHERE turno_id = ANY (ts);
	UPDATE notificaciones SET titulo = replace(titulo, c.nombre, 'Cliente borrado'), texto = replace(texto, c.nombre, 'Cliente borrado')
		WHERE turno_id = ANY (ts);
	UPDATE turnos SET nota = '', codigo = replace(gen_random_uuid()::text, '-', '') || replace(gen_random_uuid()::text, '-', ''),
		actualizado_en = now() WHERE id = ANY (ts);

	DELETE FROM cliente_etiquetas WHERE cliente_id = c.id;
	DELETE FROM notas_cliente WHERE cliente_id = c.id;
	DELETE FROM penalidades WHERE cliente_id = c.id;
	DELETE FROM membresias WHERE cliente_id = c.id;
	DELETE FROM referidos WHERE cliente_id = c.id;
	DELETE FROM preferencias WHERE cliente_id = c.id;
	DELETE FROM series WHERE cliente_id = c.id;
	UPDATE clientes SET nombre = 'Cliente borrado', telefono = '', email = '', nacimiento = NULL, suprimido_en = now()
		WHERE id = c.id;
	RETURN 'anonimizado';
END
$$;
-- +goose StatementEnd

-- 7. Calendarios: el feed iCal lo protege sólo el token, y quien gestiona ve todas las URLs. Cuando
--    alguien deja de gestionar (baja, cambio de rol o cuenta borrada) se cortan todos los calendarios
--    del negocio; si se da de baja a un profesional, los suyos. La campanita lo avisa.
-- +goose StatementBegin
CREATE FUNCTION app.cortar_calendarios() RETURNS trigger
LANGUAGE plpgsql AS $$
DECLARE
	n bigint := 0;
	gestionaba boolean := OLD.activo AND OLD.rol IN ('duenio', 'encargado');
	gestiona boolean := NEW.activo AND NEW.rol IN ('duenio', 'encargado');
BEGIN
	IF gestionaba AND NOT gestiona THEN
		DELETE FROM calendarios WHERE negocio_id = NEW.negocio_id;
		GET DIAGNOSTICS n = ROW_COUNT;
	ELSIF OLD.activo AND NOT NEW.activo THEN
		DELETE FROM calendarios WHERE negocio_id = NEW.negocio_id
			AND profesional_id IN (SELECT p.id FROM profesionales p WHERE p.miembro_id = NEW.id);
		GET DIAGNOSTICS n = ROW_COUNT;
	END IF;
	-- Quien gestionaba veía todos los links: se cortan todos. Si no, sólo los de su agenda.
	IF n > 0 THEN
		INSERT INTO notificaciones (negocio_id, tipo, titulo, texto)
			VALUES (NEW.negocio_id, 'equipo', 'Cortamos los calendarios suscriptos',
				left('Por seguridad, al cambiar el acceso de ' || coalesce(nullif(NEW.nombre, ''), NEW.email) ||
					CASE WHEN gestionaba AND NOT gestiona THEN ', los links de los calendarios dejaron de andar.'
						ELSE ', los links de su agenda dejaron de andar.' END ||
					' Volvé a conectarlos desde Integraciones.', 500));
	END IF;
	RETURN NEW;
END
$$;
-- +goose StatementEnd
CREATE TRIGGER cortar_calendarios AFTER UPDATE OF rol, activo ON miembros FOR EACH ROW EXECUTE FUNCTION app.cortar_calendarios();

-- +goose Down
DROP TRIGGER cortar_calendarios ON miembros;
DROP FUNCTION app.cortar_calendarios();
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION app.purgar(p_avisos timestamptz, p_auditoria timestamptz, p_vencidos timestamptz, p_tanda int)
RETURNS TABLE (tabla text, borradas bigint)
LANGUAGE plpgsql SECURITY DEFINER SET search_path = public, app AS $$
DECLARE
	n bigint;
BEGIN
	PERFORM set_config('app.barrido', 'purga', true);
	DELETE FROM avisos WHERE id IN (SELECT a.id FROM avisos a WHERE a.creado_en < p_avisos AND a.estado <> 'pendiente' LIMIT p_tanda);
	GET DIAGNOSTICS n = ROW_COUNT;
	tabla := 'avisos'; borradas := n; RETURN NEXT;
	DELETE FROM notificaciones WHERE id IN (SELECT x.id FROM notificaciones x WHERE x.creado_en < p_avisos LIMIT p_tanda);
	GET DIAGNOSTICS n = ROW_COUNT;
	tabla := 'notificaciones'; borradas := n; RETURN NEXT;
	DELETE FROM auditoria WHERE id IN (SELECT x.id FROM auditoria x WHERE x.creado_en < p_auditoria LIMIT p_tanda);
	GET DIAGNOSTICS n = ROW_COUNT;
	tabla := 'auditoria'; borradas := n; RETURN NEXT;
	DELETE FROM resena_tokens WHERE token IN (SELECT x.token FROM resena_tokens x WHERE x.vence_en < p_vencidos LIMIT p_tanda);
	GET DIAGNOSTICS n = ROW_COUNT;
	tabla := 'resena_tokens'; borradas := n; RETURN NEXT;
	DELETE FROM invitaciones WHERE id IN (SELECT x.id FROM invitaciones x WHERE x.vence_en < p_vencidos LIMIT p_tanda);
	GET DIAGNOSTICS n = ROW_COUNT;
	tabla := 'invitaciones'; borradas := n; RETURN NEXT;
	DELETE FROM espera WHERE id IN (SELECT x.id FROM espera x WHERE x.hasta < p_vencidos::date LIMIT p_tanda);
	GET DIAGNOSTICS n = ROW_COUNT;
	tabla := 'espera'; borradas := n; RETURN NEXT;
	PERFORM set_config('app.barrido', '', true);
END
$$;
-- +goose StatementEnd
DROP POLICY purga ON mcp_registro;
DROP POLICY purga ON asistente_propuestas;
ALTER TABLE negocios ALTER COLUMN color_acento SET DEFAULT '#22d3ee';
DROP FUNCTION app.tomar_slug_para_alta(text);
DROP INDEX notificaciones_profesional;
ALTER TABLE notificaciones DROP COLUMN profesional_id;
REVOKE UPDATE (argumentos) ON asistente_propuestas FROM tengolibre_app;
REVOKE UPDATE (ia_activa) ON ajustes FROM tengolibre_app;
ALTER TABLE ajustes DROP COLUMN ia_activa;
