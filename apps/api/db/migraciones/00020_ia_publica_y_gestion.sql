-- +goose Up
-- «La IA de tus clientes»: el servidor MCP público (/v1/mcp/publico, sin clave) deja que el
-- asistente de IA de cualquier persona encuentre los negocios que lo prendieron y reserve, con las
-- mismas reglas que la página pública. Esos turnos llevan origen 'ia' (se cuentan en el panel).
ALTER TABLE negocios ADD COLUMN ia_publica boolean NOT NULL DEFAULT false;
GRANT UPDATE (ia_publica) ON negocios TO tengolibre_app;

ALTER TABLE turnos DROP CONSTRAINT turnos_origen_check;
ALTER TABLE turnos ADD CONSTRAINT turnos_origen_check CHECK (origen IN ('web', 'panel', 'ia'));
-- «N reservas llegaron por un asistente de IA este mes» y «lo último que hizo».
CREATE INDEX turnos_ia ON turnos (negocio_id, creado_en DESC) WHERE origen = 'ia';

-- +goose StatementBegin
-- Buscar negocios para la IA pública: como app.directorio, sólo los que la tienen prendida.
CREATE FUNCTION app.directorio_ia(p_rubro text, p_ciudad text, p_texto text, p_limite int) RETURNS jsonb
LANGUAGE plpgsql SECURITY DEFINER SET search_path = public, app AS $$
DECLARE
	out jsonb;
BEGIN
	PERFORM set_config('app.plataforma', 'si', true);
	WITH elegidos AS (
		SELECT n.* FROM negocios n
		WHERE n.activo AND n.ia_publica AND app.es_reservable(n.id)
			AND (p_ciudad = '' OR strpos(lower(n.ciudad), lower(p_ciudad)) > 0 OR strpos(lower(n.provincia), lower(p_ciudad)) > 0)
			AND (p_rubro = '' OR n.rubro = p_rubro)
			AND (p_texto = '' OR strpos(lower(n.nombre), lower(p_texto)) > 0 OR strpos(lower(n.descripcion), lower(p_texto)) > 0
				OR EXISTS (SELECT 1 FROM servicios s WHERE s.negocio_id = n.id AND s.activo AND strpos(lower(s.nombre), lower(p_texto)) > 0))
		ORDER BY n.nombre, n.id LIMIT greatest(1, least(p_limite, 20))
	)
	SELECT coalesce(jsonb_agg(jsonb_build_object(
		'slug', p.slug, 'nombre', p.nombre, 'descripcion', p.descripcion, 'ciudad', p.ciudad, 'provincia', p.provincia,
		'direccion', p.direccion, 'rubro', p.rubro,
		'servicios', to_jsonb(ARRAY(SELECT s.nombre FROM servicios s WHERE s.negocio_id = p.id AND s.activo ORDER BY s.orden, s.nombre LIMIT 5)),
		'desde', (SELECT min(s.precio) FROM servicios s WHERE s.negocio_id = p.id AND s.activo)
	) ORDER BY p.nombre, p.id), '[]'::jsonb) INTO out FROM elegidos p;
	PERFORM set_config('app.plataforma', '', true);
	RETURN out;
END
$$;
-- +goose StatementEnd
REVOKE ALL ON FUNCTION app.directorio_ia(text, text, text, int) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION app.directorio_ia(text, text, text, int) TO tengolibre_app;

-- Mercado Pago: el nombre de la cuenta conectada («LUMI ESTÉTICA SRL»), para el chip del panel.
ALTER TABLE mp_conexiones ADD COLUMN cuenta text NOT NULL DEFAULT '' CHECK (length(cuenta) <= 120);

-- Suscripción: cambiar de plan (el precio nuevo va al preapproval) y las altas que reemplazan a otra
-- (pagar la cuota que falló, cambiar la tarjeta, reactivar): cuando la nueva queda autorizada, la
-- vieja se cancela.
ALTER TABLE suscripciones ADD COLUMN reemplaza uuid REFERENCES suscripciones (id);
GRANT UPDATE (plan, monto) ON suscripciones TO tengolibre_app;

-- +goose Down
REVOKE UPDATE (plan, monto) ON suscripciones FROM tengolibre_app;
ALTER TABLE suscripciones DROP COLUMN reemplaza;
ALTER TABLE mp_conexiones DROP COLUMN cuenta;
DROP FUNCTION app.directorio_ia(text, text, text, int);
DROP INDEX turnos_ia;
-- Los turnos que ya llegaron por la IA quedan como estaban (la restricción vieja no los revisa).
ALTER TABLE turnos DROP CONSTRAINT turnos_origen_check;
ALTER TABLE turnos ADD CONSTRAINT turnos_origen_check CHECK (origen IN ('web', 'panel')) NOT VALID;
ALTER TABLE negocios DROP COLUMN ia_publica;
