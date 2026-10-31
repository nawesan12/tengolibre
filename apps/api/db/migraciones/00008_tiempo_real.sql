-- +goose Up
-- Tiempo real: cada cambio de lo que el panel muestra en vivo avisa por pg_notify en el canal
-- `tengolibre_eventos`. La API escucha ese canal y lo reparte por SSE a los paneles abiertos del
-- negocio. El aviso lleva sólo qué cambió (tabla, id, operación), nunca datos del cliente: el panel
-- vuelve a pedir lo que necesite a la API, que aplica los permisos de siempre.

-- +goose StatementBegin
CREATE FUNCTION app.avisar_cambio() RETURNS trigger
LANGUAGE plpgsql AS $$
DECLARE
	fila record := coalesce(NEW, OLD);
BEGIN
	PERFORM pg_notify('tengolibre_eventos', json_build_object(
		'negocio', fila.negocio_id, 'tipo', TG_TABLE_NAME, 'id', fila.id, 'op', lower(TG_OP))::text);
	RETURN NULL;
END
$$;
-- +goose StatementEnd

CREATE TRIGGER tiempo_real AFTER INSERT OR UPDATE OR DELETE ON turnos FOR EACH ROW EXECUTE FUNCTION app.avisar_cambio();
CREATE TRIGGER tiempo_real AFTER INSERT OR UPDATE OR DELETE ON pagos FOR EACH ROW EXECUTE FUNCTION app.avisar_cambio();
CREATE TRIGGER tiempo_real AFTER INSERT OR UPDATE OR DELETE ON notificaciones FOR EACH ROW EXECUTE FUNCTION app.avisar_cambio();
CREATE TRIGGER tiempo_real AFTER INSERT OR UPDATE OR DELETE ON espera FOR EACH ROW EXECUTE FUNCTION app.avisar_cambio();
CREATE TRIGGER tiempo_real AFTER INSERT OR UPDATE OR DELETE ON resenas FOR EACH ROW EXECUTE FUNCTION app.avisar_cambio();
CREATE TRIGGER tiempo_real AFTER INSERT OR UPDATE OR DELETE ON clientes FOR EACH ROW EXECUTE FUNCTION app.avisar_cambio();

-- +goose Down
DROP TRIGGER tiempo_real ON clientes;
DROP TRIGGER tiempo_real ON resenas;
DROP TRIGGER tiempo_real ON espera;
DROP TRIGGER tiempo_real ON notificaciones;
DROP TRIGGER tiempo_real ON pagos;
DROP TRIGGER tiempo_real ON turnos;
DROP FUNCTION app.avisar_cambio();
