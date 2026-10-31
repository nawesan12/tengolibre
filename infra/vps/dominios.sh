#!/bin/sh
# Regenera /etc/caddy/sitios/tengolibre-dominios.caddy (dominios propios verificados de los negocios)
# desde la base y recarga Caddy. Calcado de `scripts/desplegar-vps.sh dominios` de Corredor.
#
#   infra/vps/dominios.sh          desde una copia del repo (entra al VPS por ssh)
#
# Se valida el Caddyfile entero antes de recargar: si no valida, vuelve el archivo anterior (queda en
# /root/tengolibre-dominios.caddy.anterior) y Caddy sigue sirviendo lo de siempre. Vuelta atrás a
# mano: cp /root/tengolibre-dominios.caddy.anterior /etc/caddy/sitios/tengolibre-dominios.caddy &&
# systemctl reload caddy (o borrar el archivo si era el primero).
# Toca Caddy, que comparten Corredor y Tu Local Web: avisar a esas sesiones antes.
set -eu
VPS=root@187.41.102.190
ssh "$VPS" 'set -e; cd /opt/tengolibre; S=/etc/caddy/sitios
	docker compose run --rm --no-deps -T api /app/tengolibrectl dominios-caddy > /root/tengolibre-dominios.caddy.nuevo
	[ -f $S/tengolibre-dominios.caddy ] && cp $S/tengolibre-dominios.caddy /root/tengolibre-dominios.caddy.anterior || true
	mv /root/tengolibre-dominios.caddy.nuevo $S/tengolibre-dominios.caddy
	if caddy validate --config /etc/caddy/Caddyfile --adapter caddyfile >/dev/null 2>&1; then systemctl reload caddy; echo "dominios propios recargados"
	else echo "tengolibre-dominios.caddy no valida: se deja el anterior"; if [ -f /root/tengolibre-dominios.caddy.anterior ]; then cp /root/tengolibre-dominios.caddy.anterior $S/tengolibre-dominios.caddy; else rm $S/tengolibre-dominios.caddy; fi; exit 1; fi'
