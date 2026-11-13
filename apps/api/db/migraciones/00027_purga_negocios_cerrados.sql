-- +goose Up
-- Negocios cerrados: a los 90 días del cierre se borran, con sus clientes (lo que dice la política).
-- Lo único que se conserva son los cobros, sin datos de nadie, en auth.cobros_archivados (fuera de
-- la RLS por negocio y sin permisos para la app: lo lee sólo la plataforma).
--
-- cerrado_en lo lleva un trigger, así vale para cualquier cierre (la baja de la cuenta, el admin) y
-- se vacía si el negocio vuelve a abrir. Los que ya estaban cerrados cuentan desde esta migración.
ALTER TABLE negocios ADD COLUMN cerrado_en timestamptz;
ALTER TABLE negocios NO FORCE ROW LEVEL SECURITY;
UPDATE negocios SET cerrado_en = now() WHERE NOT activo;
ALTER TABLE negocios FORCE ROW LEVEL SECURITY;
CREATE INDEX negocios_cerrados ON negocios (cerrado_en) WHERE NOT activo;

-- +goose StatementBegin
CREATE FUNCTION app.marcar_cierre() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
	IF NEW.activo IS DISTINCT FROM OLD.activo THEN
		NEW.cerrado_en := CASE WHEN NEW.activo THEN NULL ELSE now() END;
	END IF;
	RETURN NEW;
END
$$;
-- +goose StatementEnd
CREATE TRIGGER marcar_cierre BEFORE UPDATE OF activo ON negocios FOR EACH ROW EXECUTE FUNCTION app.marcar_cierre();

-- Los cobros de los negocios purgados. origen: sena (Mercado Pago, la seña del turno), caja (lo que
-- se cobró en el local) o suscripcion (lo que el negocio le pagó a TengoLibre). referencia: los ids
-- de Mercado Pago, Apple o Google, para cruzarlo con sus comprobantes. Sin cliente, turno ni email.
CREATE TABLE auth.cobros_archivados (
	id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
	negocio_id uuid NOT NULL,
	origen text NOT NULL CHECK (origen IN ('sena', 'caja', 'suscripcion')),
	referencia text NOT NULL DEFAULT '',
	monto bigint NOT NULL,
	devuelto bigint NOT NULL DEFAULT 0,
	estado text NOT NULL,
	medio text NOT NULL DEFAULT '',
	fecha timestamptz NOT NULL,
	archivado_en timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX cobros_archivados_negocio ON auth.cobros_archivados (negocio_id, fecha);

-- Encontrar los cerrados de todos los negocios: el migrador sólo los ve durante el barrido.
CREATE POLICY purga_negocios ON negocios FOR SELECT TO tengolibre_migrador USING (app.barrido() = 'purga_negocios');

-- +goose StatementBegin
-- Los negocios que la purga borraría con p_antes (ahora - 90 días): cerrados antes de esa fecha. Un
-- negocio con una suscripción que todavía cobra (autorizada o pausada) no se purga (retenido):
-- primero hay que darla de baja en Mercado Pago o la tienda. No borra nada.
CREATE FUNCTION app.negocios_para_purgar(p_antes timestamptz)
RETURNS TABLE (id uuid, slug text, nombre text, cerrado_en timestamptz, clientes bigint, turnos bigint, cobros bigint, retenido boolean)
LANGUAGE plpgsql SECURITY DEFINER SET search_path = public, app AS $$
DECLARE
	antes text := current_setting('app.negocio_id', true);
	n record;
BEGIN
	PERFORM set_config('app.barrido', 'purga_negocios', true);
	FOR n IN SELECT x.id, x.slug, x.nombre, x.cerrado_en FROM negocios x
		WHERE NOT x.activo AND x.cerrado_en < p_antes ORDER BY x.cerrado_en LOOP
		PERFORM set_config('app.negocio_id', n.id::text, true);
		id := n.id; slug := n.slug; nombre := n.nombre; cerrado_en := n.cerrado_en;
		SELECT count(*) INTO clientes FROM public.clientes;
		SELECT count(*) INTO turnos FROM public.turnos;
		SELECT (SELECT count(*) FROM pagos) + (SELECT count(*) FROM caja_cobros) + (SELECT count(*) FROM suscripciones) INTO cobros;
		SELECT EXISTS (SELECT 1 FROM suscripciones s WHERE s.estado IN ('autorizada', 'pausada')) INTO retenido;
		RETURN NEXT;
	END LOOP;
	PERFORM set_config('app.negocio_id', coalesce(antes, ''), true);
	PERFORM set_config('app.barrido', '', true);
END
$$;
-- +goose StatementEnd
REVOKE ALL ON FUNCTION app.negocios_para_purgar(timestamptz) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION app.negocios_para_purgar(timestamptz) TO tengolibre_app;

-- +goose StatementBegin
-- La purga: de a p_tanda, los negocios cerrados antes de p_antes y no retenidos. Copia sus cobros a
-- auth.cobros_archivados y borra el negocio: todo lo demás (clientes, turnos, equipo, web, mails,
-- campanita, auditoría…) se va en cascada. Los turnos van primero porque frenan el borrado de los
-- clientes (ON DELETE RESTRICT). Devuelve los ids borrados (el worker borra además sus imágenes).
CREATE FUNCTION app.purgar_negocios(p_antes timestamptz, p_tanda int) RETURNS uuid[]
LANGUAGE plpgsql SECURITY DEFINER SET search_path = public, app AS $$
DECLARE
	antes text := current_setting('app.negocio_id', true);
	x uuid;
	borrados uuid[] := '{}';
BEGIN
	FOR x IN SELECT p.id FROM app.negocios_para_purgar(p_antes) p WHERE NOT p.retenido LIMIT p_tanda LOOP
		PERFORM set_config('app.negocio_id', x::text, true);
		INSERT INTO auth.cobros_archivados (negocio_id, origen, referencia, monto, devuelto, estado, medio, fecha)
			SELECT x, 'sena', concat_ws(' ', p.referencia, p.mp_orden, p.mp_transaccion, p.mp_pago::text), p.monto, p.devuelto,
				p.estado, 'mercadopago', p.creado_en
			FROM pagos p WHERE p.estado IN ('aprobado', 'devuelto');
		INSERT INTO auth.cobros_archivados (negocio_id, origen, monto, estado, medio, fecha)
			SELECT x, 'caja', c.total, c.estado, c.medio, coalesce(c.cobrado_en, c.creado_en) FROM caja_cobros c;
		INSERT INTO auth.cobros_archivados (negocio_id, origen, referencia, monto, estado, medio, fecha)
			SELECT x, 'suscripcion', concat_ws(' ', s.referencia, s.mp_id, s.apple_original, s.google_token), s.monto, s.estado,
				s.origen, coalesce(s.ultimo_cobro, s.creado_en)
			FROM suscripciones s WHERE s.ultimo_cobro IS NOT NULL;
		DELETE FROM turnos;
		DELETE FROM negocios WHERE id = x;
		borrados := borrados || x;
	END LOOP;
	PERFORM set_config('app.negocio_id', coalesce(antes, ''), true);
	RETURN borrados;
END
$$;
-- +goose StatementEnd
REVOKE ALL ON FUNCTION app.purgar_negocios(timestamptz, int) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION app.purgar_negocios(timestamptz, int) TO tengolibre_app;

-- +goose Down
DROP FUNCTION app.purgar_negocios(timestamptz, int);
DROP FUNCTION app.negocios_para_purgar(timestamptz);
DROP POLICY purga_negocios ON negocios;
DROP TABLE auth.cobros_archivados;
DROP TRIGGER marcar_cierre ON negocios;
DROP FUNCTION app.marcar_cierre();
DROP INDEX negocios_cerrados;
ALTER TABLE negocios DROP COLUMN cerrado_en;
