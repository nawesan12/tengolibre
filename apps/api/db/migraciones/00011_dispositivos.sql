-- +goose Up
-- Push al teléfono: cada iPhone o Android donde alguien abrió sesión en la app registra su token
-- (APNs o FCM), atado a esa sesión: si la sesión se cierra o se borra, la fila se va con ella (ON
-- DELETE CASCADE); si vence, el worker no la elige. Es de la persona, no de un negocio: vive en auth,
-- como las sesiones, y lo que llega es la campanita de los negocios donde es miembro activo.
CREATE TABLE auth.dispositivos (
	id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
	clerk_id text NOT NULL CHECK (clerk_id ~ '^user_[A-Za-z0-9]{10,64}$'),
	sesion_hash bytea NOT NULL REFERENCES auth.sesiones (hash) ON DELETE CASCADE,
	plataforma text NOT NULL CHECK (plataforma IN ('ios', 'android')),
	-- APNs: hexadecimal (hoy 32 bytes; Apple avisa que puede crecer). FCM: el token de registro.
	token text NOT NULL UNIQUE CHECK (
		(plataforma = 'ios' AND token ~ '^[0-9a-f]{64,200}$')
		OR (plataforma = 'android' AND token ~ '^[A-Za-z0-9_:-]+$' AND length(token) BETWEEN 20 AND 4096)),
	-- Sólo APNs: el token de una app de Xcode es sandbox; TestFlight y App Store, producción.
	entorno text NOT NULL DEFAULT 'produccion' CHECK (entorno IN ('sandbox', 'produccion')),
	modelo text NOT NULL DEFAULT '' CHECK (length(modelo) <= 60),
	version_app text NOT NULL DEFAULT '' CHECK (length(version_app) <= 30),
	creado_en timestamptz NOT NULL DEFAULT now(),
	actualizado_en timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX dispositivos_clerk ON auth.dispositivos (clerk_id);
CREATE INDEX dispositivos_sesion ON auth.dispositivos (sesion_hash);
GRANT SELECT, INSERT, DELETE ON auth.dispositivos TO tengolibre_app;

-- Lo que falta mandar por push de la campanita. Lo que ya estaba antes de esta migración queda en
-- false (no se manda tarde); lo nuevo nace en true y el worker lo baja al tomarlo.
ALTER TABLE notificaciones ADD COLUMN push boolean NOT NULL DEFAULT false;
ALTER TABLE notificaciones ALTER COLUMN push SET DEFAULT true;
CREATE INDEX notificaciones_push ON notificaciones (negocio_id, creado_en) WHERE push;
GRANT UPDATE (push) ON notificaciones TO tengolibre_app;

-- +goose Down
DROP INDEX notificaciones_push;
ALTER TABLE notificaciones DROP COLUMN push;
DROP TABLE auth.dispositivos;
