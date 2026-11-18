-- +goose Up
-- Cambiar el link (tengolibre.com.ar/<slug>) sin romper los viejos: el slug que se deja queda en
-- slugs_anteriores y app.negocio_publico lo sigue resolviendo al negocio (la web redirige con 301
-- al nuevo). Los QR y carteles impresos siguen andando. Un slug anterior queda retenido para su
-- negocio 90 días: nadie más lo puede tomar en ese tiempo. Después, si otro lo toma, gana el actual.
-- (00031 la puede tomar otra rama: por eso 00032.)

CREATE TABLE slugs_anteriores (
	slug text PRIMARY KEY CHECK (slug ~ '^[a-z0-9]([a-z0-9-]{0,46}[a-z0-9])?$'),
	negocio_id uuid NOT NULL REFERENCES negocios (id) ON DELETE CASCADE,
	-- Hasta cuándo fue el link del negocio: desde ahí corren los 90 días de retención.
	dejado_en timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX slugs_anteriores_negocio ON slugs_anteriores (negocio_id);
SELECT app.aplicar_rls('slugs_anteriores');
GRANT SELECT ON slugs_anteriores TO tengolibre_app;
-- Las funciones de abajo (como el migrador) ven y tocan sólo el slug que están buscando.
CREATE POLICY por_slug ON slugs_anteriores TO tengolibre_migrador
	USING (slug = current_setting('app.slug', true)) WITH CHECK (true);
-- Y del slug anterior al negocio: sólo el que lo tuvo.
CREATE POLICY por_slug_anterior ON negocios FOR SELECT TO tengolibre_migrador
	USING (id IN (SELECT negocio_id FROM slugs_anteriores WHERE slug = current_setting('app.slug', true)));

-- Cada cambio, para el tope (5 cada 30 días) y para mostrar los links viejos que siguen andando.
CREATE TABLE cambios_de_link (
	id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
	negocio_id uuid NOT NULL REFERENCES negocios (id) ON DELETE CASCADE,
	de text NOT NULL,
	a text NOT NULL,
	creado_en timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX cambios_de_link_negocio ON cambios_de_link (negocio_id, creado_en DESC);
SELECT app.aplicar_rls('cambios_de_link');
GRANT SELECT ON cambios_de_link TO tengolibre_app;

-- +goose StatementBegin
-- La página pública: el slug actual de un negocio activo; si no hay, el negocio que lo tuvo antes.
CREATE OR REPLACE FUNCTION app.negocio_publico(p_slug text) RETURNS uuid
LANGUAGE plpgsql SECURITY DEFINER SET search_path = public, app AS $$
DECLARE
	n uuid;
BEGIN
	PERFORM set_config('app.slug', p_slug, true);
	SELECT id INTO n FROM negocios WHERE slug = p_slug AND activo;
	IF n IS NULL THEN
		SELECT g.id INTO n FROM slugs_anteriores s JOIN negocios g ON g.id = s.negocio_id
			WHERE s.slug = p_slug AND g.activo;
	END IF;
	PERFORM set_config('app.slug', '', true);
	RETURN n;
END
$$;
-- +goose StatementEnd

-- +goose StatementBegin
-- ¿Lo puede usar p_negocio (NULL: un negocio nuevo)? 'libre', 'propia' (es su link de ahora) o
-- 'tomada' (es el link de otro, o lo dejó otro hace menos de 90 días). Los reservados los ve Go.
CREATE FUNCTION app.estado_slug(p_slug text, p_negocio uuid) RETURNS text
LANGUAGE plpgsql SECURITY DEFINER SET search_path = public, app AS $$
DECLARE
	actual uuid;
	anterior uuid;
	dejado timestamptz;
BEGIN
	PERFORM set_config('app.slug', p_slug, true);
	SELECT id INTO actual FROM negocios WHERE slug = p_slug;
	SELECT negocio_id, dejado_en INTO anterior, dejado FROM slugs_anteriores WHERE slug = p_slug;
	PERFORM set_config('app.slug', '', true);
	IF actual IS NOT NULL THEN
		RETURN CASE WHEN actual = p_negocio THEN 'propia' ELSE 'tomada' END;
	END IF;
	IF anterior IS NOT NULL AND anterior IS DISTINCT FROM p_negocio AND dejado > now() - interval '90 days' THEN
		RETURN 'tomada';
	END IF;
	RETURN 'libre';
END
$$;
-- +goose StatementEnd
REVOKE ALL ON FUNCTION app.estado_slug(text, uuid) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION app.estado_slug(text, uuid) TO tengolibre_app;

-- +goose StatementBegin
-- Cambia el link del negocio fijado. Go ya validó formato, reservados y rol; acá va lo que necesita
-- ver a los otros negocios: que esté libre y el tope. El viejo pasa a slugs_anteriores.
CREATE FUNCTION app.cambiar_slug(p_nuevo text) RETURNS text
LANGUAGE plpgsql SECURITY DEFINER SET search_path = public, app AS $$
DECLARE
	yo uuid := app.negocio_actual();
	viejo text;
	cambios int;
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
	-- El nuevo deja de ser un link anterior (propio, o de otro con la retención vencida).
	PERFORM set_config('app.slug', p_nuevo, true);
	DELETE FROM slugs_anteriores WHERE slug = p_nuevo;
	-- El viejo queda apuntando acá (si ya era anterior de otro, ahora es de este negocio).
	PERFORM set_config('app.slug', viejo, true);
	INSERT INTO slugs_anteriores (slug, negocio_id) VALUES (viejo, yo)
		ON CONFLICT (slug) DO UPDATE SET negocio_id = excluded.negocio_id, dejado_en = now();
	PERFORM set_config('app.slug', '', true);
	UPDATE negocios SET slug = p_nuevo WHERE id = yo;
	INSERT INTO cambios_de_link (negocio_id, de, a) VALUES (yo, viejo, p_nuevo);
	RETURN 'ok';
END
$$;
-- +goose StatementEnd
REVOKE ALL ON FUNCTION app.cambiar_slug(text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION app.cambiar_slug(text) TO tengolibre_app;

-- +goose Down
DROP FUNCTION app.cambiar_slug(text);
DROP FUNCTION app.estado_slug(text, uuid);
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION app.negocio_publico(p_slug text) RETURNS uuid
LANGUAGE plpgsql SECURITY DEFINER SET search_path = public, app AS $$
DECLARE
	n uuid;
BEGIN
	PERFORM set_config('app.slug', p_slug, true);
	SELECT id INTO n FROM negocios WHERE slug = p_slug AND activo;
	PERFORM set_config('app.slug', '', true);
	RETURN n;
END
$$;
-- +goose StatementEnd
DROP POLICY por_slug_anterior ON negocios;
DROP TABLE cambios_de_link;
DROP TABLE slugs_anteriores;
