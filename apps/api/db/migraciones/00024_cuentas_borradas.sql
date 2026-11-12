-- +goose Up
-- Borrar la cuenta (lo exigen Google Play y la App Store). La persona la borra desde la app o la web
-- (DELETE /v1/auth/cuenta): la cuenta se desactiva en el momento y sus datos personales se borran a
-- los 30 días (la purga del worker). En esos 30 días se recupera a mano (tengolibrectl
-- recuperar-cuenta) si escribe a hola@. No hay tabla de personas: la identidad es el clerk_id, y lo
-- suyo está repartido en miembros, sesiones, dispositivos y las columnas «quién lo hizo».
--
-- La app sólo lee la tabla (el ingreso pregunta si hay una baja sin purgar). Escribir, y tocar lo de
-- todos los negocios de la persona, va por las funciones de abajo (SECURITY DEFINER: fijan cada
-- negocio para pasar la RLS, como app.admin_cambiar_negocio).
CREATE TABLE auth.cuentas_borradas (
	clerk_id text PRIMARY KEY CHECK (clerk_id ~ '^user_[A-Za-z0-9]{10,64}$'),
	-- Para encontrarla cuando escribe a hola@. La purga lo vacía.
	email text NOT NULL DEFAULT '' CHECK (length(email) <= 120),
	borrada_en timestamptz NOT NULL DEFAULT now(),
	-- Cuándo se borraron sus datos personales. Desde ahí, si vuelve a entrar, es una cuenta nueva.
	purgada_en timestamptz,
	-- Lo que la baja apagó, para poder recuperarla tal cual: sus membresías activas y los negocios
	-- donde era la única dueña.
	miembros uuid[] NOT NULL DEFAULT '{}',
	negocios uuid[] NOT NULL DEFAULT '{}'
);
CREATE INDEX cuentas_borradas_pendientes ON auth.cuentas_borradas (borrada_en) WHERE purgada_en IS NULL;
GRANT SELECT ON auth.cuentas_borradas TO tengolibre_app;

-- +goose StatementBegin
-- La baja: desactiva sus membresías, cierra (activo = false) los negocios donde era la única dueña
-- activa, revoca las claves del asistente de IA que creó, y borra sus sesiones (con ellas se van los
-- dispositivos de push y las suscripciones web, ON DELETE CASCADE) y lo que quede atado a su
-- clerk_id. Devuelve los negocios que cerró. Si ya estaba borrada sin purgar, no cambia la fecha.
CREATE FUNCTION app.borrar_cuenta(p_clerk text, p_email text) RETURNS uuid[]
LANGUAGE plpgsql SECURITY DEFINER SET search_path = public, app AS $$
DECLARE
	antes text := current_setting('app.negocio_id', true);
	m record;
	apagados uuid[] := '{}';
	cerrados uuid[] := '{}';
BEGIN
	PERFORM set_config('app.clerk_id', p_clerk, true);
	FOR m IN SELECT id, negocio_id, rol FROM miembros WHERE clerk_id = p_clerk AND activo LOOP
		PERFORM set_config('app.negocio_id', m.negocio_id::text, true);
		IF m.rol = 'duenio' AND NOT EXISTS (
			SELECT 1 FROM miembros o WHERE o.negocio_id = m.negocio_id AND o.rol = 'duenio' AND o.activo AND o.clerk_id <> p_clerk
		) THEN
			UPDATE negocios SET activo = false WHERE id = m.negocio_id AND activo;
			IF FOUND THEN
				cerrados := cerrados || m.negocio_id;
			END IF;
		END IF;
		UPDATE miembros SET activo = false WHERE id = m.id;
		apagados := apagados || m.id;
		UPDATE mcp_tokens SET revocado_en = now() WHERE creado_por = p_clerk AND revocado_en IS NULL;
	END LOOP;
	PERFORM set_config('app.negocio_id', coalesce(antes, ''), true);
	PERFORM set_config('app.clerk_id', '', true);

	DELETE FROM auth.sesiones WHERE clerk_id = p_clerk;
	DELETE FROM auth.dispositivos WHERE clerk_id = p_clerk;
	DELETE FROM auth.suscripciones_web WHERE clerk_id = p_clerk;
	DELETE FROM auth.oauth_autorizaciones WHERE clerk_id = p_clerk AND canjeada_en IS NULL;

	INSERT INTO auth.cuentas_borradas AS c (clerk_id, email, miembros, negocios)
		VALUES (p_clerk, left(p_email, 120), apagados, cerrados)
		ON CONFLICT (clerk_id) DO UPDATE SET
			-- Borrada otra vez después de purgada (volvió a entrar como nueva): empieza de cero.
			borrada_en = CASE WHEN c.purgada_en IS NULL THEN c.borrada_en ELSE now() END,
			purgada_en = NULL,
			email = excluded.email,
			miembros = CASE WHEN c.purgada_en IS NULL THEN c.miembros ELSE '{}' END || excluded.miembros,
			negocios = CASE WHEN c.purgada_en IS NULL THEN c.negocios ELSE '{}' END || excluded.negocios;
	RETURN cerrados;
END
$$;
-- +goose StatementEnd
REVOKE ALL ON FUNCTION app.borrar_cuenta(text, text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION app.borrar_cuenta(text, text) TO tengolibre_app;

-- +goose StatementBegin
-- La purga, a los 30 días (p_antes = ahora - 30 días): de cada cuenta borrada sin purgar, borra lo
-- que dice quién es la persona y deja lo del negocio sin ella:
--  · sus membresías (en todos los negocios), y las notas de clientes que escribió quedan sin autor;
--  · el profesional atado a su usuario queda suelto y sin su mail, teléfono ni foto (el nombre se
--    queda: es lo que el negocio cargó y lo que muestra su agenda pasada);
--  · los «quién lo hizo» (auditoría, caja, turnos fijos, visitas de membresías, claves de IA) quedan
--    sin su clerk_id; sus preferencias de avisos, lo que vio de la campanita, sus sucursales y los
--    estados de OAuth de Mercado Pago se borran;
--  · el email de la baja se vacía y queda purgada_en.
-- Devuelve cuántas cuentas purgó.
CREATE FUNCTION app.purgar_cuentas(p_antes timestamptz, p_tanda int) RETURNS int
LANGUAGE plpgsql SECURITY DEFINER SET search_path = public, app AS $$
DECLARE
	antes text := current_setting('app.negocio_id', true);
	c record;
	m record;
	n int := 0;
BEGIN
	FOR c IN SELECT clerk_id FROM auth.cuentas_borradas
		WHERE purgada_en IS NULL AND borrada_en < p_antes ORDER BY borrada_en LIMIT p_tanda FOR UPDATE SKIP LOCKED LOOP
		PERFORM set_config('app.clerk_id', c.clerk_id, true);
		FOR m IN SELECT id, negocio_id FROM miembros WHERE clerk_id = c.clerk_id LOOP
			PERFORM set_config('app.negocio_id', m.negocio_id::text, true);
			UPDATE notas_cliente SET autor = '' WHERE miembro_id = m.id;
			UPDATE profesionales SET email = '', telefono = '', foto = '' WHERE miembro_id = m.id;
			-- Las FK de notas y profesionales sueltan el miembro solas (ON DELETE SET NULL).
			DELETE FROM miembros WHERE id = m.id;
			UPDATE auditoria SET clerk_id = '' WHERE clerk_id = c.clerk_id;
			UPDATE caja_cobros SET clerk_id = '' WHERE clerk_id = c.clerk_id;
			UPDATE series SET clerk_id = '' WHERE clerk_id = c.clerk_id;
			UPDATE membresia_creditos SET clerk_id = NULL WHERE clerk_id = c.clerk_id;
			UPDATE mcp_tokens SET creado_por = '' WHERE creado_por = c.clerk_id;
			DELETE FROM mp_estados WHERE clerk_id = c.clerk_id;
			DELETE FROM notificaciones_vistas WHERE clerk_id = c.clerk_id;
			DELETE FROM push_preferencias WHERE clerk_id = c.clerk_id;
		END LOOP;
		DELETE FROM grupos WHERE duenio = c.clerk_id;
		PERFORM set_config('app.clerk_id', '', true);
		-- Lo que la baja ya borró, por si quedó algo (una sesión abierta en el mismo instante).
		DELETE FROM auth.sesiones WHERE clerk_id = c.clerk_id;
		DELETE FROM auth.dispositivos WHERE clerk_id = c.clerk_id;
		DELETE FROM auth.suscripciones_web WHERE clerk_id = c.clerk_id;
		UPDATE auth.oauth_autorizaciones SET clerk_id = NULL WHERE clerk_id = c.clerk_id;
		UPDATE auth.cuentas_borradas SET purgada_en = now(), email = '', miembros = '{}', negocios = '{}'
			WHERE clerk_id = c.clerk_id;
		n := n + 1;
	END LOOP;
	PERFORM set_config('app.negocio_id', coalesce(antes, ''), true);
	RETURN n;
END
$$;
-- +goose StatementEnd
REVOKE ALL ON FUNCTION app.purgar_cuentas(timestamptz, int) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION app.purgar_cuentas(timestamptz, int) TO tengolibre_app;

-- +goose StatementBegin
-- Recuperar una cuenta borrada sin purgar (la persona escribió a hola@ dentro de los 30 días): vuelve
-- a prender sus membresías y los negocios que la baja cerró, y borra la baja. Por email o clerk_id.
-- Devuelve el clerk_id, o NULL si no hay una baja que se pueda recuperar. Sólo la corre el migrador
-- (tengolibrectl recuperar-cuenta): la app no tiene permiso.
CREATE FUNCTION app.recuperar_cuenta(p_quien text) RETURNS text
LANGUAGE plpgsql SECURITY DEFINER SET search_path = public, app AS $$
DECLARE
	antes text := current_setting('app.negocio_id', true);
	c auth.cuentas_borradas;
	m record;
	x uuid;
BEGIN
	SELECT * INTO c FROM auth.cuentas_borradas
		WHERE purgada_en IS NULL AND (clerk_id = p_quien OR (p_quien <> '' AND lower(email) = lower(p_quien)))
		ORDER BY borrada_en DESC LIMIT 1 FOR UPDATE;
	IF NOT FOUND THEN
		RETURN NULL;
	END IF;
	PERFORM set_config('app.clerk_id', c.clerk_id, true);
	FOR m IN SELECT id, negocio_id FROM miembros WHERE id = ANY (c.miembros) LOOP
		PERFORM set_config('app.negocio_id', m.negocio_id::text, true);
		UPDATE miembros SET activo = true WHERE id = m.id;
	END LOOP;
	FOREACH x IN ARRAY c.negocios LOOP
		PERFORM set_config('app.negocio_id', x::text, true);
		UPDATE negocios SET activo = true WHERE id = x;
	END LOOP;
	PERFORM set_config('app.clerk_id', '', true);
	PERFORM set_config('app.negocio_id', coalesce(antes, ''), true);
	DELETE FROM auth.cuentas_borradas WHERE clerk_id = c.clerk_id;
	RETURN c.clerk_id;
END
$$;
-- +goose StatementEnd
REVOKE ALL ON FUNCTION app.recuperar_cuenta(text) FROM PUBLIC;

-- +goose Down
DROP FUNCTION app.recuperar_cuenta(text);
DROP FUNCTION app.purgar_cuentas(timestamptz, int);
DROP FUNCTION app.borrar_cuenta(text, text);
DROP TABLE auth.cuentas_borradas;
