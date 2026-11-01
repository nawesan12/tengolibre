# Estado

## 31/10/2026 (noche): diseño v2 en toda la web y todas las funciones del diseño

Toda la web se rehízo con el handoff v2 (`docs/handoff/design_handoff_tengolibre_v2/`, fiel al
píxel): tokens, Lato y Material Symbols en `apps/web/src/app.css` y `app.html`, la marca en
`static/marca/` y las firmas en `src/lib/ui/` (Talon, Hueco, Icono, Logo). Por área:
- Landing, alta y onboarding de 8 pasos (`src/lib/landing`, `src/lib/alta`, `/empezar`, cartel A5 con QR).
- Web pública con 6 plantillas × estilo × colores (`src/lib/publico/PaginaNegocio.svelte`), reserva en
  una pantalla, Mi turno como talón (repetir, confirmar, .ics), reseña, `/espera/{token}`.
- Panel: armazón (sidebar con «El día», buscador en vivo, campanita, pestañas en el teléfono), Inicio,
  Turnos (WhatsApp, oferta del hueco, caja, para reprogramar), Calendario, Lista de espera, Clientes
  (recordatorios), y toda la gestión: Mi web, Plantillas (vista previa en vivo por iframe `?vista=1`),
  Integraciones e IA, servicios y productos, equipo y días libres, pagos, membresías, cupones,
  reseñas, referidos, reportes (horas muertas, reporte mensual imprimible), configuración, suscripción.
- Cuenta: ingreso (equipo y socio), `/conectar` (OAuth del MCP), invitación, Mi cuenta, preferencias
  de mails, Explorar, admin.

Funciones nuevas de la API (Nahuel decidió construir todas las del diseño; contratos en `docs/api/`):
| Migración | Qué |
|---|---|
| 00012 | Estilo publicado de la web (plantilla, portada, fondo, títulos, forma; luego secciones) |
| 00013 | Ofrecer el hueco solo a la lista de espera (con vencimiento) y pedir días libres |
| 00014 | Confirmación y recordatorio por WhatsApp (Cloud API de Meta) y push web (VAPID) con preferencias |
| 00015 | Turno fijo (series), volver a reservar desde Mi turno, caja y productos |
| 00016 | Servidor MCP del negocio (claves, permisos, registro de acciones) |
| 00017 | Horas muertas con cupón por franja, cumpleaños y «hace mucho que no venís», reporte mensual |
| 00018 | OAuth 2.1 del MCP (registro dinámico, PKCE) para claude.ai y ChatGPT |
| 00019 | Datos de cuenta, explorar (puntaje, próximo libre) y admin; preferencias de mails separadas |
| 00020 | IA de los clientes (MCP público `/v1/mcp/publico`), reportes y suscripción que faltaban |

Lo que necesita configuración externa para prenderse (sin eso queda apagado, sin romper nada):
WhatsApp (`TENGOLIBRE_WHATSAPP_*`, cuenta de Meta y dos plantillas: ver `docs/api/whatsapp.md`),
push web (`TENGOLIBRE_VAPID_*`, se generan con `go run ./cmd/vapid`), Mercado Pago (`TENGOLIBRE_MP_*`).

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

Push al teléfono (`push`, migración 00011): la app registra su token en `POST /v1/yo/dispositivos`
(atado a la sesión) y el worker manda cada notificación nueva de la campanita a los miembros activos,
por APNs (iPhone, clave «Vimo APNs») y FCM (Android, cuenta `fcm-tengolibre`). Las claves viven sólo
en el VPS, en `/opt/tengolibre/claves/` (montada en el worker); `vimo-copia` las guarda.
Asistente de IA por MCP (`mcp`, migración 00016): el negocio crea una clave en el panel y conecta su
Claude/ChatGPT a `https://api.tengolibre.com.ar/v1/mcp`, con permisos por clave y registro de acciones.
Detalle para el front en `docs/api/mcp.md`.
Diseño: el handoff v2 está en `docs/handoff/design_handoff_tengolibre_v2/` (README con las reglas).

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
1. App de iPhone (handoff en ~/Trabajo/vimo/tengolibreiphone.zip; App Store Connect 6829332747).
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
