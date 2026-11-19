-- +goose Up
-- 1. turnos.profesional_elegida: la clienta eligió a la profesional (o tocó «Cualquiera» y se la
--    asignamos). Al reprogramar desde su link, con «Cualquiera» se ofrecen los horarios de todas y se
--    reasigna; antes quedaba atada a la que le tocó.
-- 2. La profesional que arma el alta para la dueña quedaba sin su miembro: en Equipo, la dueña
--    figuraba «Sin cuenta del panel». Los negocios con una sola profesional suelta y una sola dueña
--    las unen (el alta nueva ya manda el miembro).
ALTER TABLE turnos ADD COLUMN profesional_elegida boolean NOT NULL DEFAULT true;

ALTER TABLE profesionales NO FORCE ROW LEVEL SECURITY;
ALTER TABLE miembros NO FORCE ROW LEVEL SECURITY;
UPDATE profesionales p SET miembro_id = m.id
	FROM miembros m
	WHERE m.negocio_id = p.negocio_id AND m.rol = 'duenio' AND m.activo AND p.miembro_id IS NULL
		AND (SELECT count(*) FROM profesionales x WHERE x.negocio_id = p.negocio_id) = 1
		AND (SELECT count(*) FROM miembros y WHERE y.negocio_id = p.negocio_id AND y.rol = 'duenio' AND y.activo) = 1
		AND NOT EXISTS (SELECT 1 FROM profesionales z WHERE z.miembro_id = m.id);
ALTER TABLE miembros FORCE ROW LEVEL SECURITY;
ALTER TABLE profesionales FORCE ROW LEVEL SECURITY;

-- 3. El push del worker (cada 10 s) abría una transacción por negocio activo aunque no hubiera nada:
--    ahora recorre sólo los que tienen notificaciones por mandar (con el índice parcial de 00011).
CREATE POLICY barrido_push ON notificaciones FOR SELECT TO tengolibre_migrador
	USING (app.barrido() = 'push_pendiente' AND push);
-- +goose StatementBegin
CREATE FUNCTION app.negocios_con_push() RETURNS SETOF uuid
LANGUAGE plpgsql SECURITY DEFINER SET search_path = public, app AS $$
DECLARE
	ids uuid[];
BEGIN
	PERFORM set_config('app.barrido', 'push_pendiente', true);
	SELECT coalesce(array_agg(DISTINCT x.negocio_id), '{}') INTO ids FROM notificaciones x WHERE x.push;
	PERFORM set_config('app.barrido', 'negocios', true);
	RETURN QUERY SELECT n.id FROM negocios n WHERE n.activo AND n.id = ANY (ids);
	PERFORM set_config('app.barrido', '', true);
END
$$;
-- +goose StatementEnd
REVOKE ALL ON FUNCTION app.negocios_con_push() FROM PUBLIC;
GRANT EXECUTE ON FUNCTION app.negocios_con_push() TO tengolibre_app;

-- +goose Down
DROP FUNCTION app.negocios_con_push();
DROP POLICY barrido_push ON notificaciones;
ALTER TABLE turnos DROP COLUMN profesional_elegida;
