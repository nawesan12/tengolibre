-- +goose Up
-- +goose StatementBegin
-- Los próximos turnos de una persona en todos los negocios, por el mail verificado de su cuenta: quien
-- reserva con su cuenta abierta los ve en «Mi cuenta» (antes sólo tenía el link del mail). Recorre los
-- negocios activos fijando cada uno, como el resto de las funciones que cruzan negocios.
CREATE FUNCTION app.turnos_de_persona(p_email text, p_desde timestamptz)
RETURNS TABLE (codigo text, inicio timestamptz, fin timestamptz, estado text, servicio text, profesional text,
	negocio text, slug text, zona text)
LANGUAGE plpgsql SECURITY DEFINER SET search_path = public, app AS $$
DECLARE
	antes text := current_setting('app.negocio_id', true);
	n record;
BEGIN
	IF coalesce(trim(p_email), '') = '' THEN
		RETURN;
	END IF;
	PERFORM set_config('app.barrido', 'negocios', true);
	FOR n IN SELECT x.id, x.nombre, x.slug, x.zona_horaria FROM negocios x WHERE x.activo LOOP
		PERFORM set_config('app.barrido', '', true);
		PERFORM set_config('app.negocio_id', n.id::text, true);
		RETURN QUERY SELECT t.codigo, t.inicio, t.fin, t.estado, coalesce(s.nombre, ''), coalesce(p.nombre, ''),
				n.nombre, n.slug, n.zona_horaria
			FROM turnos t JOIN clientes c ON c.id = t.cliente_id
			LEFT JOIN servicios s ON s.id = t.servicio_id
			LEFT JOIN profesionales p ON p.id = t.profesional_id
			WHERE lower(c.email) = lower(p_email) AND t.fin >= p_desde AND t.estado <> 'cancelado';
		PERFORM set_config('app.barrido', 'negocios', true);
	END LOOP;
	PERFORM set_config('app.barrido', '', true);
	PERFORM set_config('app.negocio_id', coalesce(antes, ''), true);
END
$$;
-- +goose StatementEnd
REVOKE ALL ON FUNCTION app.turnos_de_persona(text, timestamptz) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION app.turnos_de_persona(text, timestamptz) TO tengolibre_app;

-- +goose Down
DROP FUNCTION app.turnos_de_persona(text, timestamptz);
