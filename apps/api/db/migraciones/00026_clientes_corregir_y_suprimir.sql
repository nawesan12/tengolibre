-- +goose Up
-- Los derechos del cliente (Ley 25.326, art. 16): corregir su teléfono y pedir que se borren sus
-- datos. El negocio es el responsable de su base de clientes y lo hace desde el panel o las apps.
--
-- Corregir: la app ya editaba nombre, email y nacimiento; ahora también el teléfono (sigue siendo
-- único por negocio: lo frena clientes_telefono).
GRANT UPDATE (telefono) ON clientes TO tengolibre_app;

-- Suprimir: si el cliente no tiene turnos, se borra entero. Si tiene, los turnos pasados se quedan
-- (son la agenda y la plata del negocio) y el cliente queda como «Cliente borrado», sin teléfono,
-- email ni nacimiento, con suprimido_en. El listado y la búsqueda no lo muestran.
ALTER TABLE clientes ADD COLUMN suprimido_en timestamptz;

-- +goose StatementBegin
-- La supresión de un cliente del negocio fijado. Corre como dueña de las tablas (SECURITY DEFINER),
-- pero dentro del negocio de la sesión: las políticas «negocio» siguen filtrando. Con turnos por
-- venir (después de p_ahora) no hace nada y devuelve 'con_turnos' (el negocio los cancela antes, con su aviso y la
-- oferta del hueco). Si no, devuelve 'borrado' o 'anonimizado'; NULL si el cliente no existe o ya
-- estaba suprimido.
--
-- Al anonimizar:
--  · el cliente: nombre «Cliente borrado», sin teléfono, email ni nacimiento;
--  · sus turnos: sin nota y con otro código (el link de Mi turno deja de andar), sin los
--    dispositivos de la app que seguían el turno ni los tokens de reseña;
--  · se borran sus etiquetas, notas, penalidades, membresías, referidos, preferencias de mails,
--    turnos fijos, reseñas, anotaciones en la lista de espera (por teléfono o email) y los mails y
--    WhatsApp que quedaban por salir; a los que ya salieron se les borra el destinatario y los datos;
--  · la campanita deja de nombrarlo y la auditoría del cliente queda sin detalle.
-- Los cobros (señas y caja) no tienen datos del cliente: se quedan con el turno.
CREATE FUNCTION app.suprimir_cliente(p_cliente uuid, p_ahora timestamptz) RETURNS text
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
REVOKE ALL ON FUNCTION app.suprimir_cliente(uuid, timestamptz) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION app.suprimir_cliente(uuid, timestamptz) TO tengolibre_app;

-- +goose Down
DROP FUNCTION app.suprimir_cliente(uuid, timestamptz);
ALTER TABLE clientes DROP COLUMN suprimido_en;
REVOKE UPDATE (telefono) ON clientes FROM tengolibre_app;
