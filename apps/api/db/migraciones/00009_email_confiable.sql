-- +goose Up
-- El email de un cliente es «de confianza» cuando lo cargó el negocio (panel), no cuando lo tipeó
-- cualquiera en la página pública, donde el cliente se busca por teléfono. Sólo un email de
-- confianza identifica a quien ingresó con ese email verificado (para usar su membresía al
-- reservar): si no, bastaría con escribir el teléfono de un socio y un email propio.
--
-- Lo decide un trigger para no depender de cada INSERT/UPDATE: la reserva pública fija
-- `app.origen = 'web'` en su transacción. Desde la web, además, un email de confianza no se pisa.
ALTER TABLE clientes ADD COLUMN email_confiable boolean NOT NULL DEFAULT false;
CREATE INDEX clientes_email_confiable ON clientes (negocio_id, lower(email)) WHERE email_confiable;

-- +goose StatementBegin
CREATE FUNCTION app.cuidar_email() RETURNS trigger
LANGUAGE plpgsql AS $$
DECLARE
	web boolean := coalesce(current_setting('app.origen', true), '') = 'web';
BEGIN
	IF TG_OP = 'UPDATE' THEN
		IF NEW.email IS NOT DISTINCT FROM OLD.email THEN
			NEW.email_confiable := OLD.email_confiable;
			RETURN NEW;
		END IF;
		IF web AND OLD.email_confiable THEN
			NEW.email := OLD.email;
			NEW.email_confiable := true;
			RETURN NEW;
		END IF;
	END IF;
	NEW.email_confiable := NEW.email <> '' AND NOT web;
	RETURN NEW;
END
$$;
-- +goose StatementEnd

CREATE TRIGGER cuidar_email BEFORE INSERT OR UPDATE ON clientes FOR EACH ROW EXECUTE FUNCTION app.cuidar_email();

-- +goose Down
DROP TRIGGER cuidar_email ON clientes;
DROP FUNCTION app.cuidar_email();
DROP INDEX clientes_email_confiable;
ALTER TABLE clientes DROP COLUMN email_confiable;
