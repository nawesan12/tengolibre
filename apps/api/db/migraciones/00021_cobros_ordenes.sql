-- +goose Up
-- Las señas pasan de la API de Preferencias (Checkout Pro, que Mercado Pago discontinúa) a la API de
-- Orders. La orden (ORD…) y la transacción de pago (PAY…) se guardan para consultar y devolver; mp_pago
-- sigue siendo el id numérico del pago (en una orden viene en payments[].reference.id) y es lo que usan
-- los pagos viejos, creados con preferencias, que se devuelven por /v1/payments.
ALTER TABLE pagos ADD COLUMN mp_orden text, ADD COLUMN mp_transaccion text;
GRANT UPDATE (mp_orden, mp_transaccion) ON pagos TO tengolibre_app;
COMMENT ON COLUMN pagos.link IS 'checkout_url de la orden (o el init_point de una preferencia vieja): se reusa si el cliente vuelve a pedir el link.';

-- +goose Down
ALTER TABLE pagos DROP COLUMN mp_orden, DROP COLUMN mp_transaccion;
