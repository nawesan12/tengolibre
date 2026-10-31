-- +goose Up
-- Calendarios para suscribirse (iCal): un link secreto por negocio (todos los turnos) o por
-- profesional (los suyos), que Google Calendar, Apple o Outlook leen solos cada tanto. El token es
-- el secreto: regenerarlo es borrar la fila y crear otra.
CREATE TABLE calendarios (
	id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
	negocio_id uuid NOT NULL REFERENCES negocios (id) ON DELETE CASCADE,
	-- NULL: el del negocio entero.
	profesional_id uuid,
	token text NOT NULL UNIQUE CHECK (token ~ '^[A-Za-z0-9_-]{32,64}$'),
	creado_en timestamptz NOT NULL DEFAULT now(),
	UNIQUE NULLS NOT DISTINCT (negocio_id, profesional_id),
	FOREIGN KEY (negocio_id, profesional_id) REFERENCES profesionales (negocio_id, id) ON DELETE CASCADE
);
SELECT app.aplicar_rls('calendarios');
GRANT SELECT, INSERT, DELETE ON calendarios TO tengolibre_app;

CREATE POLICY por_token ON calendarios FOR SELECT TO tengolibre_migrador
	USING (token = current_setting('app.cal_token', true));

-- +goose StatementBegin
-- Del token al negocio, antes de fijarlo (como app.negocio_del_turno).
CREATE FUNCTION app.negocio_del_calendario(p_token text) RETURNS uuid
LANGUAGE plpgsql SECURITY DEFINER SET search_path = public, app AS $$
DECLARE
	n uuid;
BEGIN
	PERFORM set_config('app.cal_token', p_token, true);
	SELECT negocio_id INTO n FROM calendarios WHERE token = p_token;
	PERFORM set_config('app.cal_token', '', true);
	RETURN n;
END
$$;
-- +goose StatementEnd
REVOKE ALL ON FUNCTION app.negocio_del_calendario(text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION app.negocio_del_calendario(text) TO tengolibre_app;

-- +goose Down
DROP FUNCTION app.negocio_del_calendario(text);
DROP TABLE calendarios;
