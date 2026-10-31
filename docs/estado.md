# Estado

## 31/10/2026 (tarde): front publicado

https://tengolibre.com.ar sirve el front (SvelteKit, `apps/web`, contenedor `tengolibre-web-1` en
127.0.0.1:3400) con todas las pantallas de Jiku: página pública y reserva en pasos, link del turno,
reseñas, invitaciones, explorar, panel completo (turnos, calendario en vivo, clientes, equipo y
horarios, mi web, pagos, membresías, cupones, referidos, reportes, configuración, suscripción),
sucursales, mi cuenta y admin. Estilo neutro provisorio: la UI final sale del handoff.
Ingreso: Google y Apple por Clerk producción, canjeado por una sesión propia de la API (30 días).
Tiempo real: SSE de la API (`/v1/panel/eventos`, LISTEN/NOTIFY) reenviado por `/panel/eventos`.
Con la prueba vencida, todo el panel muestra arriba que está en sólo lectura (`GET /v1/panel/acceso`,
que puede leer cualquier rol); al dueño, además, los últimos 3 días de prueba.
Membresías en la reserva pública: el socio ingresa (Clerk) desde «Tus datos» y tilda «Usar mi
membresía». La API lo identifica por su email verificado contra el email «de confianza» del cliente
(`clientes.email_confiable`: lo cargó el negocio, nunca lo tipeado en la web; ver
`app.cuidar_email`, migración 00009), así nadie usa una membresía ajena escribiendo un teléfono.
Google Calendar (y Apple/Outlook): link iCal secreto por negocio o por profesional
(`calendario`, `/panel/calendario/suscribirse`, migración 00010). El front sirve `/cal/{token}.ics`
reenviando `GET /v1/cal/…`; un profesional con usuario vinculado maneja sólo el suyo.

Desplegar el front: `cd apps/web && bun run build`, `rsync -az --delete build/ root@187.41.102.190:/opt/tengolibre/web/build/`
y `docker compose restart web` en /opt/tengolibre. Config en `/opt/tengolibre/web.conf`.

## 31/10/2026: backend de Jiku completo en Go, en producción

La API (`apps/api`) cubre todo lo de Jiku y está desplegada en https://api.tengolibre.com.ar
(versión en `/v1/version`). Falta el front: se arma con el handoff del rediseño.

### Dominios (`internal/dominio`)
| Paquete | Qué hace |
|---|---|
| `negocios` | Alta (con Clerk), datos, ajustes de reserva, equipo con roles |
| `catalogo` | Categorías, servicios, profesionales, horario semanal, bloqueos, excepciones |
| `disponibilidad` | Motor puro de horarios libres (availability-engine.ts) |
| `turnos` | Página pública, reserva, link del cliente (ver/cancelar/reprogramar), agenda del panel |
| `clientes`, `reportes` | CRM, etiquetas, notas, penalidades por ausencias, búsqueda, estadísticas, CSV |
| `cobros`, `promos`, `membresias`, `suscripcion` | Mercado Pago (señas, webhook, devoluciones), cupones, referidos, membresías, suscripción a TengoLibre y panel en sólo lectura con la prueba vencida |
| `avisos`, `resenas`, `espera`, `invitaciones` | Cola de mails y campanita, reseñas, lista de espera, invitaciones al equipo |
| `miweb`, `dominios`, `directorio`, `grupos`, `arranque`, `admin` | Web del negocio y galería, dominio propio, explorar, sucursales, onboarding, admin de la plataforma |

Plataforma (`internal/plataforma`): `db` (RLS por negocio), `sesion` (Clerk + X-Negocio + roles +
sólo lectura), `web`, `problema`, `archivos`, `correo`, `auditoria`, `trabajos`, `cifrado`, `clerk`.

### Para el front
- Sesión: `Authorization: Bearer <token de Clerk>`. En el panel, además `X-Negocio: <id>` (los negocios
  de la persona salen de `GET /v1/yo`). En desarrollo sin Clerk vale `Bearer dev:user_xxxxxxxxxx`.
- Errores: `application/problem+json` con `code` estable (p. ej. `horario_ocupado`,
  `cliente_penalizado`, `suscripcion_requerida` = 402).
- Rutas: públicas en `/v1/p/{slug}/…`, `/v1/t/{codigo}/…` (link del cliente), `/v1/explorar`,
  `/v1/resenas/{token}`, `/v1/invitaciones/{token}`, `/v1/preferencias/{token}`, `/v1/host/{host}`;
  panel en `/v1/panel/…`; sucursales en `/v1/grupos`; admin en `/v1/admin/…`. El detalle de cada una
  está en el `Rutas` de cada paquete y en sus tests de punta a punta.
- Links que mandan los mails: `/turno/{codigo}`, `/{slug}`, `/resena/{token}`,
  `/invitacion/{token}`, `/mi-cuenta/avisos/{token}`, `/panel?fecha=`.
- Vuelta de Clerk: `https://tengolibre.com.ar/ingresar/clerk` (y `ar.com.tengolibre.app://callback`
  en la app nativa), ya cargadas en Clerk producción. Vuelta de Mercado Pago:
  `/panel/mercadopago/vuelta`, que manda `code` y `state` a `POST /v1/panel/mercadopago/vuelta`.

### En el VPS
- `/opt/tengolibre`: compose `tengolibre` (postgres 256m, api 128m en 127.0.0.1:4200, worker 64m),
  `api.conf` (600), `pg.secret`, `archivos/`, `proximamente/`. Web reservada en 3400.
- Caddy: `/etc/caddy/sitios/tengolibre.caddy`; certificado de origen `/etc/caddy/certs/tengolibre.*`.
- Correo: `sistema@tengolibre.com.ar` (manda como hola@) por mail.tulocalweb.com.ar:465.
- Copia diaria: `vimo-copia` (la lleva la sesión de Tu Local Web) guarda base, conf y archivos.
- Registro común de todas las apps: `/opt/VPS.md`.

### Pendiente
1. UI final del front: aplicar el handoff del rediseño (fiel al handoff) sobre las pantallas que ya andan.
2. Credenciales de Mercado Pago de TengoLibre: App ID y secreto (marketplace con PKCE), clave de
   webhooks y token de la cuenta de la plataforma (`TENGOLIBRE_MP_*`). Sin eso, cobrar y suscribirse
   responden que falta configurarlo.
3. Admin: listo por email (`TENGOLIBRE_ADMINS_EMAILS` con nsantillandev@gmail.com y nahuel@grupovimo.com.ar); `TENGOLIBRE_ADMINS` (IDs) queda vacío.
4. Dominios propios: el DNS `dominios.tengolibre.com.ar` ya existe (sin proxy). Falta correr
   `infra/vps/dominios.sh` después de la primera verificación (coordinar con las otras apps, que
   comparten Caddy) y ponerle un timer.

## Desplegar
```bash
cd apps/api && V=$(git rev-parse --short HEAD)
for c in api worker tengolibrectl; do CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "-X main.version=$V" -o bin/$c ./cmd/$c; done
ssh root@187.41.102.190 'docker exec tengolibre-postgres-1 pg_dumpall -U postgres | gzip > /var/backups/tengolibre/antes-$(date +%Y%m%d%H%M).sql.gz'
for c in api worker tengolibrectl; do scp bin/$c root@187.41.102.190:/opt/tengolibre/bin/$c.nuevo; done
ssh root@187.41.102.190 'cd /opt/tengolibre && for c in api worker tengolibrectl; do mv bin/$c.nuevo bin/$c; done &&
  docker compose run --rm --no-deps -T api /app/tengolibrectl migrar && docker compose restart api worker'
```
