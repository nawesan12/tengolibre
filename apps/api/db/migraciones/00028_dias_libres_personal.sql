-- +goose Up
-- El motivo «salud» de los días libres era un dato de salud del equipo que la agenda no necesita: pasa
-- a «personal», neutro. Se convierten los pedidos, los bloqueos que dejó su aprobación («Días libres
-- (salud)»), la campanita («… · salud») y la auditoría.
ALTER TABLE dias_libres DROP CONSTRAINT dias_libres_motivo_check;
ALTER TABLE dias_libres NO FORCE ROW LEVEL SECURITY;
ALTER TABLE bloqueos NO FORCE ROW LEVEL SECURITY;
ALTER TABLE notificaciones NO FORCE ROW LEVEL SECURITY;
ALTER TABLE auditoria NO FORCE ROW LEVEL SECURITY;
UPDATE dias_libres SET motivo = 'personal' WHERE motivo = 'salud';
UPDATE bloqueos SET motivo = 'Días libres (personal)' WHERE motivo = 'Días libres (salud)';
UPDATE notificaciones SET texto = replace(texto, ' · salud', ' · personal') WHERE tipo = 'equipo' AND texto LIKE '% · salud%';
UPDATE auditoria SET detalle = jsonb_set(detalle, '{motivo}', '"personal"')
	WHERE entidad = 'dias_libres' AND detalle ->> 'motivo' = 'salud';
ALTER TABLE auditoria FORCE ROW LEVEL SECURITY;
ALTER TABLE notificaciones FORCE ROW LEVEL SECURITY;
ALTER TABLE bloqueos FORCE ROW LEVEL SECURITY;
ALTER TABLE dias_libres FORCE ROW LEVEL SECURITY;
ALTER TABLE dias_libres ADD CONSTRAINT dias_libres_motivo_check CHECK (motivo IN ('vacaciones', 'tramite', 'personal', 'otro'));

-- +goose Down
-- No vuelve a «salud»: los que eran salud no se distinguen de los personales.
ALTER TABLE dias_libres DROP CONSTRAINT dias_libres_motivo_check;
ALTER TABLE dias_libres ADD CONSTRAINT dias_libres_motivo_check CHECK (motivo IN ('vacaciones', 'tramite', 'salud', 'personal', 'otro'));
