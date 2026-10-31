-- +goose Up
-- Avisos y trabajos en segundo plano (Notification, NotificationPreference, Review, ReviewToken,
-- WaitlistEntry, JobRun y AuditLog en Jiku): la cola de mails, la campanita del panel, las
-- preferencias del cliente, las reseñas, la lista de espera, las invitaciones al equipo, la
-- auditoría y el registro de corridas del worker.
--
-- El worker usa el rol de la app: lo que cruza negocios (qué avisos están pendientes, qué negocios
-- hay, la purga) pasa por funciones SECURITY DEFINER que fijan app.barrido y lo limpian al salir, y
-- las políticas del migrador sólo dejan ver lo de ese barrido.

-- +goose StatementBegin
CREATE FUNCTION app.barrido() RETURNS text
LANGUAGE sql STABLE AS $$
	SELECT coalesce(current_setting('app.barrido', true), '')
$$;
-- +goose StatementEnd

-- Ajustes de avisos (reviewRequestDelayHours, dailyDigestEnabled). Marcar ausentes solos
-- (noShowAutoMark) es `marcar_ausentes`, de 00003: lo cambia el panel de ausencias y lo usa el worker.
ALTER TABLE ajustes
	ADD COLUMN resena_demora_horas int NOT NULL DEFAULT 2 CHECK (resena_demora_horas BETWEEN 0 AND 168),
	ADD COLUMN resumen_diario boolean NOT NULL DEFAULT true;
GRANT UPDATE (resena_demora_horas, resumen_diario) ON ajustes TO tengolibre_app;

-- Los negocios activos, para los trabajos que recorren todos (después cada uno corre con su negocio
-- fijado).
CREATE POLICY barrido ON negocios FOR SELECT TO tengolibre_migrador
	USING (app.barrido() = 'negocios');
-- +goose StatementBegin
CREATE FUNCTION app.negocios_activos() RETURNS SETOF uuid
LANGUAGE plpgsql SECURITY DEFINER SET search_path = public, app AS $$
BEGIN
	PERFORM set_config('app.barrido', 'negocios', true);
	RETURN QUERY SELECT n.id FROM negocios n WHERE n.activo ORDER BY n.creado_en;
	PERFORM set_config('app.barrido', '', true);
END
$$;
-- +goose StatementEnd
REVOKE ALL ON FUNCTION app.negocios_activos() FROM PUBLIC;
GRANT EXECUTE ON FUNCTION app.negocios_activos() TO tengolibre_app;

-- La cola de mails (outbox + Notification). Se escribe en la misma transacción que lo que la causa
-- y la consume el worker: pendiente → enviado, o descartado (ya no aplica: el turno se canceló, el
-- cliente no quiere mails), o fallido después de los reintentos. El mail se arma al mandarlo, con los
-- datos de ese momento; datos guarda sólo lo que no se puede volver a leer (un token, el horario
-- que tenía el turno).
CREATE TABLE avisos (
	id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
	negocio_id uuid NOT NULL REFERENCES negocios (id) ON DELETE CASCADE,
	tipo text NOT NULL CHECK (tipo IN ('reservado', 'recordatorio', 'cancelado', 'reprogramado', 'resena', 'espera',
		'invitacion', 'resumen')),
	-- A quién se mandó (o se va a mandar). Los de un turno lo completan al armarse.
	para text NOT NULL DEFAULT '' CHECK (length(para) <= 120),
	turno_id uuid REFERENCES turnos (id) ON DELETE SET NULL,
	datos jsonb NOT NULL DEFAULT '{}',
	-- Para no encolar dos veces lo mismo (el recordatorio de un turno, el resumen de un día).
	clave text CHECK (length(clave) <= 200),
	estado text NOT NULL DEFAULT 'pendiente' CHECK (estado IN ('pendiente', 'enviado', 'descartado', 'fallido')),
	intentos int NOT NULL DEFAULT 0,
	proximo_intento timestamptz NOT NULL,
	ultimo_error text CHECK (length(ultimo_error) <= 1000),
	enviado_en timestamptz,
	creado_en timestamptz NOT NULL DEFAULT now(),
	UNIQUE (negocio_id, clave)
);
CREATE INDEX avisos_pendientes ON avisos (proximo_intento) WHERE estado = 'pendiente';
CREATE INDEX avisos_negocio ON avisos (negocio_id, creado_en DESC);
CREATE INDEX avisos_turno ON avisos (turno_id);
-- La purga barre por fecha, sin negocio.
CREATE INDEX avisos_creado ON avisos (creado_en);
SELECT app.aplicar_rls('avisos');
GRANT SELECT, INSERT ON avisos TO tengolibre_app;
GRANT UPDATE (para, estado, intentos, proximo_intento, ultimo_error, enviado_en) ON avisos TO tengolibre_app;
CREATE POLICY barrido ON avisos FOR SELECT TO tengolibre_migrador
	USING (app.barrido() = 'avisos');

-- +goose StatementBegin
-- Los avisos que toca mandar, de todos los negocios. El worker después toma cada uno con su negocio
-- fijado (y FOR UPDATE SKIP LOCKED: dos workers no mandan el mismo).
CREATE FUNCTION app.avisos_pendientes(p_ahora timestamptz, p_limite int)
RETURNS TABLE (aviso_id uuid, negocio_id uuid)
LANGUAGE plpgsql SECURITY DEFINER SET search_path = public, app AS $$
BEGIN
	PERFORM set_config('app.barrido', 'avisos', true);
	RETURN QUERY SELECT a.id, a.negocio_id FROM avisos a
		WHERE a.estado = 'pendiente' AND a.proximo_intento <= p_ahora
		ORDER BY a.proximo_intento LIMIT p_limite;
	PERFORM set_config('app.barrido', '', true);
END
$$;
-- +goose StatementEnd
REVOKE ALL ON FUNCTION app.avisos_pendientes(timestamptz, int) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION app.avisos_pendientes(timestamptz, int) TO tengolibre_app;

-- La campanita del panel: lo que pasó en el negocio (una reserva desde la página, una cancelación
-- del cliente, una reseña, un mail que no salió).
CREATE TABLE notificaciones (
	id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
	negocio_id uuid NOT NULL REFERENCES negocios (id) ON DELETE CASCADE,
	tipo text NOT NULL CHECK (tipo IN ('reserva', 'cancelacion', 'reprogramacion', 'resena', 'espera', 'equipo',
		'ausente', 'aviso_fallido')),
	titulo text NOT NULL CHECK (length(titulo) BETWEEN 1 AND 120),
	texto text NOT NULL DEFAULT '' CHECK (length(texto) <= 500),
	turno_id uuid REFERENCES turnos (id) ON DELETE SET NULL,
	creado_en timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX notificaciones_negocio ON notificaciones (negocio_id, creado_en DESC);
CREATE INDEX notificaciones_creado ON notificaciones (creado_en);
SELECT app.aplicar_rls('notificaciones');
GRANT SELECT, INSERT ON notificaciones TO tengolibre_app;

-- Hasta cuándo miró cada persona la campanita (notificationsSeenAt, pero por persona: que la dueña
-- abra la suya no le apaga el número a la recepcionista).
CREATE TABLE notificaciones_vistas (
	negocio_id uuid NOT NULL REFERENCES negocios (id) ON DELETE CASCADE,
	clerk_id text NOT NULL CHECK (length(clerk_id) <= 80),
	vista_en timestamptz NOT NULL,
	PRIMARY KEY (negocio_id, clerk_id)
);
SELECT app.aplicar_rls('notificaciones_vistas');
GRANT SELECT, INSERT ON notificaciones_vistas TO tengolibre_app;
GRANT UPDATE (vista_en) ON notificaciones_vistas TO tengolibre_app;

-- Qué mails quiere recibir cada cliente (NotificationPreference). El token es el link de baja que
-- va al pie de cada mail: no hace falta cuenta para usarlo.
CREATE TABLE preferencias (
	cliente_id uuid PRIMARY KEY,
	negocio_id uuid NOT NULL REFERENCES negocios (id) ON DELETE CASCADE,
	token text NOT NULL UNIQUE CHECK (token ~ '^[A-Za-z0-9_-]{16,64}$'),
	email boolean NOT NULL DEFAULT true,
	recordatorios boolean NOT NULL DEFAULT true,
	actualizado_en timestamptz NOT NULL DEFAULT now(),
	FOREIGN KEY (negocio_id, cliente_id) REFERENCES clientes (negocio_id, id) ON DELETE CASCADE
);
SELECT app.aplicar_rls('preferencias');
GRANT SELECT, INSERT ON preferencias TO tengolibre_app;
GRANT UPDATE (email, recordatorios, actualizado_en) ON preferencias TO tengolibre_app;

-- El pedido de reseña de un turno atendido (ReviewToken): uno por turno, vence a los 7 días.
CREATE TABLE resena_tokens (
	token text PRIMARY KEY CHECK (token ~ '^[A-Za-z0-9_-]{16,64}$'),
	negocio_id uuid NOT NULL REFERENCES negocios (id) ON DELETE CASCADE,
	turno_id uuid NOT NULL UNIQUE REFERENCES turnos (id) ON DELETE CASCADE,
	vence_en timestamptz NOT NULL,
	usado_en timestamptz,
	creado_en timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX resena_tokens_negocio ON resena_tokens (negocio_id);
SELECT app.aplicar_rls('resena_tokens');
GRANT SELECT, INSERT ON resena_tokens TO tengolibre_app;
GRANT UPDATE (usado_en) ON resena_tokens TO tengolibre_app;

-- Reseñas (Review). Copian el nombre del cliente, el servicio y el profesional, como los turnos.
CREATE TABLE resenas (
	id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
	negocio_id uuid NOT NULL REFERENCES negocios (id) ON DELETE CASCADE,
	turno_id uuid NOT NULL UNIQUE REFERENCES turnos (id) ON DELETE CASCADE,
	cliente_id uuid,
	nombre text NOT NULL CHECK (length(nombre) BETWEEN 1 AND 80),
	servicio text NOT NULL CHECK (length(servicio) <= 80),
	profesional text NOT NULL CHECK (length(profesional) <= 60),
	puntaje int NOT NULL CHECK (puntaje BETWEEN 1 AND 5),
	comentario text NOT NULL DEFAULT '' CHECK (length(comentario) <= 1000),
	respuesta text NOT NULL DEFAULT '' CHECK (length(respuesta) <= 1000),
	respondida_en timestamptz,
	visible boolean NOT NULL DEFAULT true,
	creado_en timestamptz NOT NULL DEFAULT now(),
	FOREIGN KEY (negocio_id, cliente_id) REFERENCES clientes (negocio_id, id) ON DELETE SET NULL (cliente_id)
);
CREATE INDEX resenas_negocio ON resenas (negocio_id, creado_en DESC);
SELECT app.aplicar_rls('resenas');
GRANT SELECT, INSERT ON resenas TO tengolibre_app;
GRANT UPDATE (respuesta, respondida_en, visible) ON resenas TO tengolibre_app;

-- Lista de espera (WaitlistEntry): un servicio, un profesional si importa, y entre qué días. El aviso
-- sale por mail, así que el email es obligatorio.
CREATE TABLE espera (
	id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
	negocio_id uuid NOT NULL REFERENCES negocios (id) ON DELETE CASCADE,
	servicio_id uuid NOT NULL,
	profesional_id uuid,
	nombre text NOT NULL CHECK (length(nombre) BETWEEN 1 AND 80),
	email text NOT NULL CHECK (length(email) BETWEEN 3 AND 120),
	telefono text NOT NULL DEFAULT '' CHECK (length(telefono) <= 30),
	desde date NOT NULL,
	hasta date NOT NULL,
	avisado_en timestamptz,
	creado_en timestamptz NOT NULL DEFAULT now(),
	CHECK (hasta >= desde AND hasta - desde <= 31),
	FOREIGN KEY (negocio_id, servicio_id) REFERENCES servicios (negocio_id, id) ON DELETE CASCADE,
	FOREIGN KEY (negocio_id, profesional_id) REFERENCES profesionales (negocio_id, id) ON DELETE SET NULL (profesional_id)
);
CREATE INDEX espera_servicio ON espera (negocio_id, servicio_id, desde) WHERE avisado_en IS NULL;
SELECT app.aplicar_rls('espera');
GRANT SELECT, INSERT, DELETE ON espera TO tengolibre_app;
GRANT UPDATE (avisado_en) ON espera TO tengolibre_app;

-- Invitaciones al equipo para quien todavía no tiene cuenta: se acepta con la sesión de Clerk y ahí
-- se crea el miembro con el rol de la invitación. Una pendiente por email y negocio.
CREATE TABLE invitaciones (
	id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
	negocio_id uuid NOT NULL REFERENCES negocios (id) ON DELETE CASCADE,
	email text NOT NULL CHECK (length(email) BETWEEN 3 AND 120),
	rol text NOT NULL CHECK (rol IN ('encargado', 'recepcion', 'profesional')),
	token text NOT NULL UNIQUE CHECK (token ~ '^[A-Za-z0-9_-]{16,64}$'),
	invito text NOT NULL DEFAULT '' CHECK (length(invito) <= 80),
	vence_en timestamptz NOT NULL,
	aceptada_en timestamptz,
	aceptada_por text CHECK (length(aceptada_por) <= 80),
	creado_en timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX invitaciones_pendiente ON invitaciones (negocio_id, email) WHERE aceptada_en IS NULL;
SELECT app.aplicar_rls('invitaciones');
GRANT SELECT, INSERT, DELETE ON invitaciones TO tengolibre_app;
GRANT UPDATE (aceptada_en, aceptada_por) ON invitaciones TO tengolibre_app;

-- Los links públicos (baja de mails, reseña, invitación): del token al negocio. Cada política deja
-- ver sólo la fila del token que fijó la función.
CREATE POLICY por_token ON preferencias FOR SELECT TO tengolibre_migrador
	USING (token = current_setting('app.token', true));
CREATE POLICY por_token ON resena_tokens FOR SELECT TO tengolibre_migrador
	USING (token = current_setting('app.token', true));
CREATE POLICY por_token ON invitaciones FOR SELECT TO tengolibre_migrador
	USING (token = current_setting('app.token', true));
-- +goose StatementBegin
CREATE FUNCTION app.negocio_por_token(p_tipo text, p_token text) RETURNS uuid
LANGUAGE plpgsql SECURITY DEFINER SET search_path = public, app AS $$
DECLARE
	n uuid;
BEGIN
	PERFORM set_config('app.token', p_token, true);
	IF p_tipo = 'preferencias' THEN
		SELECT negocio_id INTO n FROM preferencias WHERE token = p_token;
	ELSIF p_tipo = 'resena' THEN
		SELECT negocio_id INTO n FROM resena_tokens WHERE token = p_token;
	ELSIF p_tipo = 'invitacion' THEN
		SELECT negocio_id INTO n FROM invitaciones WHERE token = p_token;
	END IF;
	PERFORM set_config('app.token', '', true);
	RETURN n;
END
$$;
-- +goose StatementEnd
REVOKE ALL ON FUNCTION app.negocio_por_token(text, text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION app.negocio_por_token(text, text) TO tengolibre_app;

-- Auditoría (AuditLog): quién hizo qué en el panel. clerk_id vacío: lo hizo el sistema (el worker).
-- La app sólo agrega: no puede editar ni borrar lo anotado (la purga va por app.purgar).
CREATE TABLE auditoria (
	id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
	negocio_id uuid NOT NULL REFERENCES negocios (id) ON DELETE CASCADE,
	clerk_id text NOT NULL DEFAULT '' CHECK (length(clerk_id) <= 80),
	accion text NOT NULL CHECK (length(accion) BETWEEN 1 AND 60),
	entidad text NOT NULL CHECK (length(entidad) BETWEEN 1 AND 40),
	entidad_id text NOT NULL DEFAULT '' CHECK (length(entidad_id) <= 80),
	detalle jsonb,
	creado_en timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX auditoria_negocio ON auditoria (negocio_id, creado_en DESC, id DESC);
CREATE INDEX auditoria_creado ON auditoria (creado_en);
SELECT app.aplicar_rls('auditoria');
GRANT SELECT, INSERT ON auditoria TO tengolibre_app;

-- Corridas de los trabajos del worker (JobRun). No es de ningún negocio: la política deja todo a la
-- app. ultima_en es a la vez el registro y el reclamo (ver trabajos.Planificador).
CREATE TABLE corridas (
	nombre text PRIMARY KEY CHECK (length(nombre) BETWEEN 1 AND 40),
	ultima_en timestamptz NOT NULL,
	termino_en timestamptz,
	ultimo_error text CHECK (length(ultimo_error) <= 2000),
	resultado jsonb,
	corridas int NOT NULL DEFAULT 0
);
ALTER TABLE corridas ENABLE ROW LEVEL SECURITY;
ALTER TABLE corridas FORCE ROW LEVEL SECURITY;
CREATE POLICY todas ON corridas TO tengolibre_app USING (true) WITH CHECK (true);
GRANT SELECT, INSERT, UPDATE ON corridas TO tengolibre_app;

-- La purga: borra, por tandas, lo que ya nadie va a leer, en todos los negocios.
CREATE POLICY purga ON avisos TO tengolibre_migrador USING (app.barrido() = 'purga');
CREATE POLICY purga ON notificaciones TO tengolibre_migrador USING (app.barrido() = 'purga');
CREATE POLICY purga ON auditoria TO tengolibre_migrador USING (app.barrido() = 'purga');
CREATE POLICY purga ON resena_tokens TO tengolibre_migrador USING (app.barrido() = 'purga');
CREATE POLICY purga ON invitaciones TO tengolibre_migrador USING (app.barrido() = 'purga');
CREATE POLICY purga ON espera TO tengolibre_migrador USING (app.barrido() = 'purga');
-- +goose StatementBegin
-- p_avisos: antes de cuándo se borran los avisos ya resueltos y la campanita; p_auditoria, la
-- auditoría; p_vencidos, los tokens de reseña, invitaciones y anotaciones de espera vencidos.
CREATE FUNCTION app.purgar(p_avisos timestamptz, p_auditoria timestamptz, p_vencidos timestamptz, p_tanda int)
RETURNS TABLE (tabla text, borradas bigint)
LANGUAGE plpgsql SECURITY DEFINER SET search_path = public, app AS $$
DECLARE
	n bigint;
BEGIN
	PERFORM set_config('app.barrido', 'purga', true);
	DELETE FROM avisos WHERE id IN (SELECT a.id FROM avisos a WHERE a.creado_en < p_avisos AND a.estado <> 'pendiente' LIMIT p_tanda);
	GET DIAGNOSTICS n = ROW_COUNT;
	tabla := 'avisos'; borradas := n; RETURN NEXT;
	DELETE FROM notificaciones WHERE id IN (SELECT x.id FROM notificaciones x WHERE x.creado_en < p_avisos LIMIT p_tanda);
	GET DIAGNOSTICS n = ROW_COUNT;
	tabla := 'notificaciones'; borradas := n; RETURN NEXT;
	DELETE FROM auditoria WHERE id IN (SELECT x.id FROM auditoria x WHERE x.creado_en < p_auditoria LIMIT p_tanda);
	GET DIAGNOSTICS n = ROW_COUNT;
	tabla := 'auditoria'; borradas := n; RETURN NEXT;
	DELETE FROM resena_tokens WHERE token IN (SELECT x.token FROM resena_tokens x WHERE x.vence_en < p_vencidos LIMIT p_tanda);
	GET DIAGNOSTICS n = ROW_COUNT;
	tabla := 'resena_tokens'; borradas := n; RETURN NEXT;
	DELETE FROM invitaciones WHERE id IN (SELECT x.id FROM invitaciones x WHERE x.vence_en < p_vencidos LIMIT p_tanda);
	GET DIAGNOSTICS n = ROW_COUNT;
	tabla := 'invitaciones'; borradas := n; RETURN NEXT;
	DELETE FROM espera WHERE id IN (SELECT x.id FROM espera x WHERE x.hasta < p_vencidos::date LIMIT p_tanda);
	GET DIAGNOSTICS n = ROW_COUNT;
	tabla := 'espera'; borradas := n; RETURN NEXT;
	PERFORM set_config('app.barrido', '', true);
END
$$;
-- +goose StatementEnd
REVOKE ALL ON FUNCTION app.purgar(timestamptz, timestamptz, timestamptz, int) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION app.purgar(timestamptz, timestamptz, timestamptz, int) TO tengolibre_app;

-- +goose Down
DROP FUNCTION app.purgar(timestamptz, timestamptz, timestamptz, int);
DROP TABLE corridas;
DROP TABLE auditoria;
DROP FUNCTION app.negocio_por_token(text, text);
DROP TABLE invitaciones;
DROP TABLE espera;
DROP TABLE resenas;
DROP TABLE resena_tokens;
DROP TABLE preferencias;
DROP TABLE notificaciones_vistas;
DROP TABLE notificaciones;
DROP FUNCTION app.avisos_pendientes(timestamptz, int);
DROP TABLE avisos;
DROP FUNCTION app.negocios_activos();
DROP POLICY barrido ON negocios;
ALTER TABLE ajustes DROP COLUMN resena_demora_horas, DROP COLUMN resumen_diario;
DROP FUNCTION app.barrido();
