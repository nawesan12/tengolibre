-- +goose Up
-- Android: la notificación en curso del turno y los avisos de la clienta también salen por FCM (el
-- token de un teléfono Android no es hex), y la suscripción se puede pagar con Google Play.

ALTER TABLE turno_actividades
	ADD COLUMN plataforma text NOT NULL DEFAULT 'ios' CHECK (plataforma IN ('ios', 'android')),
	DROP CONSTRAINT turno_actividades_token_check,
	ADD CONSTRAINT turno_actividades_token_check CHECK (
		(plataforma = 'ios' AND token ~ '^[0-9a-f]{64,400}$') OR (plataforma = 'android' AND token ~ '^[A-Za-z0-9_:-]{20,}$' AND length(token) <= 4096));

ALTER TABLE turno_dispositivos
	ADD COLUMN plataforma text NOT NULL DEFAULT 'ios' CHECK (plataforma IN ('ios', 'android')),
	DROP CONSTRAINT turno_dispositivos_token_check,
	ADD CONSTRAINT turno_dispositivos_token_check CHECK (
		(plataforma = 'ios' AND token ~ '^[0-9a-f]{64,200}$') OR (plataforma = 'android' AND token ~ '^[A-Za-z0-9_:-]{20,}$' AND length(token) <= 4096));

-- Suscripción por Google Play: se identifica por el token de compra (Play manda uno nuevo si se
-- cambia de plan; el viejo queda en linkedPurchaseToken).
ALTER TABLE suscripciones
	DROP CONSTRAINT suscripciones_origen_check,
	ADD CONSTRAINT suscripciones_origen_check CHECK (origen IN ('mercadopago', 'apple', 'google')),
	ADD COLUMN google_token text UNIQUE,
	ADD COLUMN google_producto text CHECK (length(google_producto) <= 120);
GRANT UPDATE (google_token, google_producto) ON suscripciones TO tengolibre_app;

-- +goose Down
ALTER TABLE suscripciones DROP COLUMN google_token, DROP COLUMN google_producto,
	DROP CONSTRAINT suscripciones_origen_check,
	ADD CONSTRAINT suscripciones_origen_check CHECK (origen IN ('mercadopago', 'apple'));
DELETE FROM turno_dispositivos WHERE plataforma = 'android';
DELETE FROM turno_actividades WHERE plataforma = 'android';
ALTER TABLE turno_dispositivos DROP CONSTRAINT turno_dispositivos_token_check, DROP COLUMN plataforma,
	ADD CONSTRAINT turno_dispositivos_token_check CHECK (token ~ '^[0-9a-f]{64,200}$');
ALTER TABLE turno_actividades DROP CONSTRAINT turno_actividades_token_check, DROP COLUMN plataforma,
	ADD CONSTRAINT turno_actividades_token_check CHECK (token ~ '^[0-9a-f]{64,400}$');
