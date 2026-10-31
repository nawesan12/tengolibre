# Avisos push en el panel web (Propuesta 10)

Paquetes: `internal/dominio/push` (`web.go`, `preferencias.go`, `envio.go`) e
`internal/plataforma/webpush` (Web Push con VAPID, RFC 8291/8292, calcado del de Tu Local Web).
Migración `00014` (`auth.suscripciones_web`, `push_preferencias`).

## Cómo funciona

- Cada notificación nueva de la campanita la manda el worker (trabajo `push`, cada 10 s) a los
  miembros activos del negocio: al iPhone (APNs), a Android (FCM) y ahora también a cada navegador
  donde la persona activó los avisos del panel.
- Cada persona elige, en cada negocio, qué tipos le llegan. Vale para los tres canales. Mientras
  no elija, valen los de su rol:

| Rol | Le llega todo menos |
|---|---|
| duenio, encargado | nada |
| recepcion | `equipo` |
| profesional | `confirmacion`, `cambio_pedido`, `sin_respuesta`, `espera`, `ausente`, `equipo`, `aviso_fallido` |

- Se guarda lo **apagado**: un tipo nuevo de la campanita (de otra propuesta) le llega a todos
  hasta que lo apaguen.
- La suscripción del navegador se ata a la sesión propia con que entró: cerrar la sesión o que
  venza la borra. Si el servicio de push responde que ya no existe (404/410), el worker la borra.
- El permiso del navegador se pide desde la tarjeta «Enterate en el momento» de Inicio (nunca de
  entrada): el front llama a `Notification.requestPermission()` al tocar «Activar», se suscribe con
  `pushManager.subscribe({ userVisibleOnly: true, applicationServerKey: <clave> })` y manda
  `subscription.toJSON()` a la API.

Lo que recibe el service worker (cuerpo del push, JSON):
```json
{ "title": "Nueva reserva de Florencia", "body": "Lifting con Abril, el lunes 2 de noviembre a las 16:30.",
  "url": "/panel/turnos/<id>", "negocio": "<id del negocio>", "tipo": "reserva", "tag": "noti-<id>" }
```
`url`: `/panel/turnos/{id}` si la notificación es de un turno; si no, `/panel/notificaciones`. Si
`negocio` no es el elegido, el front cambia de negocio antes de abrirla. `tag`: dos avisos con el
mismo se ven como uno.

## Endpoints

Errores: `application/problem+json` con `code`. Las rutas `/v1/yo/…` piden la **sesión propia**
(`Authorization: Bearer tl_…`, la del front); con un token de Clerk suelto responden 403
`sin_permiso`.

### `GET /v1/push/vapid` (público)
```json
{ "clave": "BP4z9…", "activo": true }
```
`clave`: la pública VAPID en base64url (va como `applicationServerKey`). Vacía y `activo: false`
si el push web está apagado.

### `GET /v1/yo/suscripciones-web`
Si este navegador (esta sesión) ya está suscripto, para decidir si mostrar la tarjeta.
```json
{ "clave": "BP4z9…", "suscripta": false }
```

### `POST /v1/yo/suscripciones-web`
Cuerpo: `PushSubscription.toJSON()` tal cual, más `aparato` (opcional, cómo se ve: «Chrome · Mac»,
hasta 60).
```json
{ "endpoint": "https://fcm.googleapis.com/fcm/send/…", "expirationTime": null,
  "keys": { "p256dh": "…", "auth": "…" }, "aparato": "Chrome · Mac" }
```
También acepta `p256dh` y `auth` sueltos. 204. Si el endpoint ya estaba (otra sesión), se mueve a
esta. Errores: 400 `invalido` (no es de un servicio de push conocido —FCM, Apple, Mozilla,
Windows— o las claves no sirven); 409 `push_apagado` (sin VAPID); 401 sin sesión; 403 sin sesión
propia.

### `DELETE /v1/yo/suscripciones-web`
Da de baja las suscripciones de esta sesión. Cuerpo opcional `{ "endpoint": "…" }` para borrar sólo
esa. 204.

### `GET /v1/panel/push/preferencias`
Roles: duenio, encargado, recepcion, profesional (`X-Negocio`). De la persona en ese negocio.
```json
{ "rol": "recepcion", "porDefecto": true, "apagados": ["equipo"],
  "tipos": [ { "tipo": "reserva", "nombre": "Reservas nuevas", "activo": true },
             { "tipo": "equipo", "nombre": "Equipo", "activo": false } ] }
```
`tipos` es la lista para la pantalla, en orden: `reserva`, `cancelacion`, `reprogramacion`,
`confirmacion`, `cambio_pedido`, `sin_respuesta`, `resena`, `espera`, `ausente`, `equipo`,
`aviso_fallido`.

### `PUT /v1/panel/push/preferencias`
Mismos roles. Cuerpo: `{ "apagados": ["resena", "equipo"] }` (lo que no le llega) o
`{ "porDefecto": true }` (volver a lo de su rol). Responde 200 con las preferencias como el GET.
Auditoría `push.preferencias`. Errores: 400 `invalido` (falta `apagados`, un tipo mal escrito, más
de 40); 402 `suscripcion_requerida` con el panel en sólo lectura.

Los teléfonos siguen con `POST`/`DELETE /v1/yo/dispositivos` (sin cambios); las preferencias los
alcanzan igual.

## Configuración

| Variable | Qué es |
|---|---|
| `TENGOLIBRE_VAPID_PUBLICA` | Clave pública VAPID (base64url, 65 bytes). API y worker. |
| `TENGOLIBRE_VAPID_PRIVADA` | Clave privada VAPID (base64url, 32 bytes). Worker (la API la usa sólo para validar el par). |
| `TENGOLIBRE_VAPID_SUJETO` | Contacto para los servicios de push (por defecto `mailto:hola@tengolibre.com.ar`). |

Generarlas una vez con `go run ./cmd/vapid` (cambiarlas deja sin avisos a los navegadores ya
suscriptos hasta que vuelvan a activarlos). Sin claves, todo sigue andando y `GET /v1/push/vapid`
dice `activo: false`. En el iPhone, el push web sólo anda con el panel instalado en la pantalla de
inicio (iOS 16.4+).
