-- +goose Up
-- Dos propuestas del diseño v2:
--
-- 01 · Ofrecer el hueco solo. Cuando se libera un horario, se le ofrece a UNA persona de la lista de
-- espera por vez, en orden, con un link propio y un tiempo para tomarlo (ajustes.espera_minutos). Si
-- no lo toma o dice que no le sirve, pasa a la siguiente. La cola se arma al liberarse el horario
-- (oferta_intentos, una fila por persona) y el worker la hace avanzar cuando vence cada turno.
--
-- 06 · Pedir días libres. Un profesional con usuario pide desde/hasta y motivo; la dueña o encargada
-- lo aprueba (se crean los bloqueos de esos días y los turnos que caen ahí pasan a la lista para
-- reprogramar) o lo rechaza.

-- Ajustes de la lista de espera: cuánto tiene cada persona para tomar el lugar y si se ofrece sola
-- (si no, lo dispara recepción desde Turnos).
ALTER TABLE ajustes
	ADD COLUMN espera_minutos int NOT NULL DEFAULT 30 CHECK (espera_minutos BETWEEN 5 AND 720),
	ADD COLUMN espera_automatica boolean NOT NULL DEFAULT true;
GRANT UPDATE (espera_minutos, espera_automatica) ON ajustes TO tengolibre_app;

-- En qué momento del día le sirve a quien se anota (el «LE SIRVE · TARDE» del diseño). Mañana es antes
-- de las 13 en la hora del negocio.
ALTER TABLE espera
	ADD COLUMN momento text NOT NULL DEFAULT 'cualquiera' CHECK (momento IN ('cualquiera', 'manana', 'tarde'));
-- Quien dice «No me sirve» vuelve a esperar (avisado_en a NULL), y a quien se le cae la oferta también.
GRANT UPDATE (avisado_en) ON espera TO tengolibre_app;

-- Un horario liberado que se está ofreciendo. Copia servicio, profesional y precio del momento: el
-- link muestra lo que se ofreció aunque después cambie el catálogo.
--   ofreciendo: hay alguien con el link vigente (o el worker está por pasar al siguiente)
--   tomada: alguien de la lista lo reservó (turno_id)
--   cargada: recepción lo resolvió («Cargarlo yo»), con o sin turno
--   frenada: el panel dejó de ofrecerlo
--   agotada: nadie de la cola lo tomó
--   vencida: llegó la hora del turno sin que nadie lo tomara
--   caida: el horario se ocupó por otro lado (otro turno, un bloqueo)
CREATE TABLE ofertas (
	id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
	negocio_id uuid NOT NULL REFERENCES negocios (id) ON DELETE CASCADE,
	servicio_id uuid NOT NULL,
	servicio text NOT NULL CHECK (length(servicio) <= 80),
	profesional_id uuid NOT NULL,
	profesional text NOT NULL CHECK (length(profesional) <= 60),
	inicio timestamptz NOT NULL,
	fin timestamptz NOT NULL,
	precio bigint NOT NULL CHECK (precio >= 0),
	-- automatica: la lanzó el horario liberado; panel: «Ofrecer a la lista».
	origen text NOT NULL CHECK (origen IN ('automatica', 'panel')),
	estado text NOT NULL DEFAULT 'ofreciendo'
		CHECK (estado IN ('ofreciendo', 'tomada', 'cargada', 'frenada', 'agotada', 'vencida', 'caida')),
	turno_id uuid REFERENCES turnos (id) ON DELETE SET NULL,
	creado_en timestamptz NOT NULL DEFAULT now(),
	cerrada_en timestamptz,
	CHECK (fin > inicio),
	UNIQUE (negocio_id, id),
	FOREIGN KEY (negocio_id, servicio_id) REFERENCES servicios (negocio_id, id) ON DELETE CASCADE,
	FOREIGN KEY (negocio_id, profesional_id) REFERENCES profesionales (negocio_id, id) ON DELETE CASCADE
);
-- Un mismo horario de un profesional se ofrece una sola vez a la vez.
CREATE UNIQUE INDEX ofertas_activa ON ofertas (negocio_id, profesional_id, inicio) WHERE estado = 'ofreciendo';
CREATE INDEX ofertas_negocio ON ofertas (negocio_id, inicio);
SELECT app.aplicar_rls('ofertas');
GRANT SELECT, INSERT ON ofertas TO tengolibre_app;
GRANT UPDATE (estado, turno_id, cerrada_en) ON ofertas TO tengolibre_app;

-- La cola de una oferta: una fila por persona de la lista, en orden. El token es el link del mail.
--   en_cola: todavía no le tocó
--   ofrecida: tiene el link vigente hasta vence_en
--   tomada / paso («No me sirve») / vencida (no respondió a tiempo)
--   cancelada: la oferta se cerró mientras la tenía (frenada, cargada, caída)
--   salteada: cuando le tocó ya no estaba esperando (se borró, ya la avisaron por otro horario)
CREATE TABLE oferta_intentos (
	id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
	negocio_id uuid NOT NULL REFERENCES negocios (id) ON DELETE CASCADE,
	oferta_id uuid NOT NULL,
	espera_id uuid REFERENCES espera (id) ON DELETE SET NULL,
	-- Copiado de la lista: el panel sigue mostrando a quién se le ofreció aunque se borre de la lista.
	nombre text NOT NULL CHECK (length(nombre) BETWEEN 1 AND 80),
	orden int NOT NULL,
	token text NOT NULL UNIQUE CHECK (token ~ '^[A-Za-z0-9_-]{16,64}$'),
	estado text NOT NULL DEFAULT 'en_cola'
		CHECK (estado IN ('en_cola', 'ofrecida', 'tomada', 'paso', 'vencida', 'cancelada', 'salteada')),
	ofrecida_en timestamptz,
	vence_en timestamptz,
	respondida_en timestamptz,
	UNIQUE (oferta_id, orden),
	FOREIGN KEY (negocio_id, oferta_id) REFERENCES ofertas (negocio_id, id) ON DELETE CASCADE
);
CREATE INDEX oferta_intentos_espera ON oferta_intentos (espera_id);
SELECT app.aplicar_rls('oferta_intentos');
GRANT SELECT, INSERT ON oferta_intentos TO tengolibre_app;
GRANT UPDATE (estado, ofrecida_en, vence_en, respondida_en) ON oferta_intentos TO tengolibre_app;

CREATE POLICY por_token ON oferta_intentos FOR SELECT TO tengolibre_migrador
	USING (token = current_setting('app.oferta_token', true));
-- +goose StatementBegin
-- El link «Se liberó lo que esperabas»: del token al negocio, antes de fijarlo (como
-- app.negocio_del_calendario).
CREATE FUNCTION app.negocio_de_oferta(p_token text) RETURNS uuid
LANGUAGE plpgsql SECURITY DEFINER SET search_path = public, app AS $$
DECLARE
	n uuid;
BEGIN
	PERFORM set_config('app.oferta_token', p_token, true);
	SELECT negocio_id INTO n FROM oferta_intentos WHERE token = p_token;
	PERFORM set_config('app.oferta_token', '', true);
	RETURN n;
END
$$;
-- +goose StatementEnd
REVOKE ALL ON FUNCTION app.negocio_de_oferta(text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION app.negocio_de_oferta(text) TO tengolibre_app;

-- Pedidos de días libres de un profesional. turnos: cuántos tenía esos días al pedirlo.
CREATE TABLE dias_libres (
	id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
	negocio_id uuid NOT NULL REFERENCES negocios (id) ON DELETE CASCADE,
	profesional_id uuid NOT NULL,
	-- La persona (Clerk) que lo pidió y la que lo respondió.
	pidio text NOT NULL CHECK (length(pidio) <= 80),
	desde date NOT NULL,
	hasta date NOT NULL,
	motivo text NOT NULL CHECK (motivo IN ('vacaciones', 'tramite', 'salud', 'otro')),
	nota text NOT NULL DEFAULT '' CHECK (length(nota) <= 300),
	estado text NOT NULL DEFAULT 'pendiente' CHECK (estado IN ('pendiente', 'aprobado', 'rechazado', 'cancelado')),
	turnos int NOT NULL DEFAULT 0,
	respondio text CHECK (length(respondio) <= 80),
	respuesta text NOT NULL DEFAULT '' CHECK (length(respuesta) <= 300),
	respondido_en timestamptz,
	creado_en timestamptz NOT NULL DEFAULT now(),
	CHECK (hasta >= desde AND hasta - desde <= 60),
	UNIQUE (negocio_id, id),
	FOREIGN KEY (negocio_id, profesional_id) REFERENCES profesionales (negocio_id, id) ON DELETE CASCADE
);
CREATE INDEX dias_libres_negocio ON dias_libres (negocio_id, creado_en DESC);
CREATE INDEX dias_libres_profesional ON dias_libres (profesional_id, desde);
SELECT app.aplicar_rls('dias_libres');
GRANT SELECT, INSERT ON dias_libres TO tengolibre_app;
GRANT UPDATE (estado, respondio, respuesta, respondido_en) ON dias_libres TO tengolibre_app;

-- Turnos que quedaron dentro de unos días libres aprobados: recepción los mueve y los marca
-- resueltos (o se resuelven solos si el turno se cancela o cambia de horario o profesional).
CREATE TABLE reprogramar (
	id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
	negocio_id uuid NOT NULL REFERENCES negocios (id) ON DELETE CASCADE,
	turno_id uuid NOT NULL REFERENCES turnos (id) ON DELETE CASCADE,
	dia_libre_id uuid NOT NULL,
	resuelto_en timestamptz,
	-- Clerk de quien lo resolvió; 'automatico' si se resolvió solo.
	resuelto_por text CHECK (length(resuelto_por) <= 80),
	creado_en timestamptz NOT NULL DEFAULT now(),
	UNIQUE (turno_id, dia_libre_id),
	FOREIGN KEY (negocio_id, dia_libre_id) REFERENCES dias_libres (negocio_id, id) ON DELETE CASCADE
);
CREATE INDEX reprogramar_pendientes ON reprogramar (negocio_id, creado_en) WHERE resuelto_en IS NULL;
SELECT app.aplicar_rls('reprogramar');
GRANT SELECT, INSERT ON reprogramar TO tengolibre_app;
GRANT UPDATE (resuelto_en, resuelto_por) ON reprogramar TO tengolibre_app;

-- +goose StatementBegin
-- Cuando un turno ocupa un horario (por la web, el panel o una reprogramación), la oferta de ese
-- horario se cae: a quien tenía el link se le cancela y vuelve a esperar. Y si un turno que estaba
-- para reprogramar se cancela o se mueve, sale solo de esa lista. Corre con el negocio fijado (RLS).
CREATE FUNCTION app.turno_ocupa() RETURNS trigger
LANGUAGE plpgsql AS $$
DECLARE
	caidas uuid[];
BEGIN
	IF NEW.estado <> 'cancelado' AND NEW.profesional_id IS NOT NULL THEN
		WITH c AS (
			UPDATE ofertas SET estado = 'caida', cerrada_en = now()
			WHERE negocio_id = NEW.negocio_id AND estado = 'ofreciendo' AND profesional_id = NEW.profesional_id
				AND inicio < NEW.fin AND fin > NEW.inicio
			RETURNING id)
		SELECT array_agg(id) INTO caidas FROM c;
		IF caidas IS NOT NULL THEN
			WITH i AS (
				UPDATE oferta_intentos SET estado = 'cancelada'
				WHERE oferta_id = ANY (caidas) AND estado = 'ofrecida'
				RETURNING espera_id)
			UPDATE espera SET avisado_en = NULL WHERE id IN (SELECT espera_id FROM i);
		END IF;
	END IF;
	IF TG_OP = 'UPDATE' AND (NEW.estado = 'cancelado' OR NEW.inicio IS DISTINCT FROM OLD.inicio
			OR NEW.profesional_id IS DISTINCT FROM OLD.profesional_id) THEN
		UPDATE reprogramar SET resuelto_en = now(), resuelto_por = 'automatico'
		WHERE turno_id = NEW.id AND resuelto_en IS NULL;
	END IF;
	RETURN NULL;
END
$$;
-- +goose StatementEnd
CREATE TRIGGER ocupa AFTER INSERT OR UPDATE OF inicio, fin, profesional_id, estado ON turnos
	FOR EACH ROW EXECUTE FUNCTION app.turno_ocupa();

-- En vivo: el panel ve aparecer la oferta, la cuenta regresiva que pasa a la siguiente, los pedidos de
-- días libres y la lista para reprogramar.
CREATE TRIGGER tiempo_real AFTER INSERT OR UPDATE OR DELETE ON ofertas FOR EACH ROW EXECUTE FUNCTION app.avisar_cambio();
CREATE TRIGGER tiempo_real AFTER INSERT OR UPDATE OR DELETE ON oferta_intentos FOR EACH ROW EXECUTE FUNCTION app.avisar_cambio();
CREATE TRIGGER tiempo_real AFTER INSERT OR UPDATE OR DELETE ON dias_libres FOR EACH ROW EXECUTE FUNCTION app.avisar_cambio();
CREATE TRIGGER tiempo_real AFTER INSERT OR UPDATE OR DELETE ON reprogramar FOR EACH ROW EXECUTE FUNCTION app.avisar_cambio();

-- +goose Down
DROP TRIGGER tiempo_real ON reprogramar;
DROP TRIGGER tiempo_real ON dias_libres;
DROP TRIGGER tiempo_real ON oferta_intentos;
DROP TRIGGER tiempo_real ON ofertas;
DROP TRIGGER ocupa ON turnos;
DROP FUNCTION app.turno_ocupa();
DROP TABLE reprogramar;
DROP TABLE dias_libres;
DROP FUNCTION app.negocio_de_oferta(text);
DROP TABLE oferta_intentos;
DROP TABLE ofertas;
ALTER TABLE espera DROP COLUMN momento;
ALTER TABLE ajustes DROP COLUMN espera_minutos, DROP COLUMN espera_automatica;
