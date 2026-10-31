# Confirmación y recordatorios por WhatsApp (Propuesta 02)

Paquetes: `internal/dominio/confirmaciones` (rutas, webhook, trabajos del worker) e
`internal/plataforma/whatsapp` (cliente de la Cloud API, firma del webhook, E.164). Migración `00014`.

## Cómo funciona

1. El día antes del turno, desde la hora que eligió el negocio (por defecto 10:00, hora del
   negocio) y hasta las 21, el worker encola la plantilla `tengolibre_confirmacion` para cada turno
   de mañana pendiente o confirmado con teléfono, salvo que el cliente ya haya confirmado solo
   (por WhatsApp, desde su link o pagando la seña) o que haya apagado los recordatorios en su link
   de preferencias. Sale desde el número de la plataforma.
2. El teléfono del cliente se normaliza a E.164 (Argentina por defecto: `11 15-2345-6789` →
   `+5491123456789`, `0351 155 123456` → `+5493515123456`). Si no se puede (falta la característica,
   por ejemplo), el mensaje queda `descartado` con el motivo, a la vista del panel.
3. El cliente toca un botón y Meta llama al webhook:
   - **Confirmo**: el turno pasa a `confirmado` con `confirmadoPor: "whatsapp"` y `confirmadoEn`
     (el panel lo ve en vivo por el SSE de siempre, `tipo: "turnos"`), suma la campanita
     `confirmacion` («Sofía Méndez confirmó por WhatsApp») y le contestamos «¡Listo! Queda
     confirmado. Si te surge algo, cambialo acá: https://tengolibre.com.ar/turno/{codigo}».
   - **Necesito cambiarlo**: el turno queda con `cambioPedidoEn` y para llamar
     (`llamarMotivo: "cambio"`), suma la campanita `cambio_pedido` («Nadia Paz pidió cambiar su
     turno», con el teléfono) y le contestamos con el link para que lo cambie ella.
   - Si el turno ya no está vigente (cancelado, pasado), le contestamos eso y el link de reserva.
4. Si no contesta en las horas del negocio (por defecto 4), el turno queda para llamar
   (`llamarMotivo: "sin_respuesta"`) y suma la campanita `sin_respuesta`.
5. Si el WhatsApp no le llega (número sin WhatsApp, o falló después de los reintentos), el turno
   queda para llamar (`llamarMotivo: "no_llego"`) y suma la campanita `aviso_fallido`.
6. Recordatorio por WhatsApp (opcional, además del mail): 24 horas antes, plantilla
   `tengolibre_recordatorio` con el botón «Ver mi turno». **Con la confirmación prendida no sale**:
   el mensaje de confirmación ya es el recordatorio.

El talón: `confirmadoPor` (`whatsapp` | `web` | `panel` | null) y `confirmadoEn` dicen «Confirmó por
WhatsApp · 10:02»; `cambioPedidoEn` dice «Pidió cambiarlo»; `llamarDesde`/`llamarMotivo` muestran
el botón «Llamar». Los cinco campos vienen en todo turno del panel (`GET /v1/panel/turnos`,
`GET /v1/panel/turnos/{id}`, etc.). El link del cliente (`GET /v1/t/{codigo}`) trae
`confirmadoPor`, `confirmadoEn` y `cambioPedidoEn`, pero `llamarDesde`/`llamarMotivo` siempre en null.

Reglas de `confirmadoPor` (trigger `app.cuidar_confirmacion`): al pasar a confirmado, si nadie dijo
de dónde, es `panel` (reserva desde el panel, `PATCH estado`); WhatsApp, el link y el pago de la
seña (`web`) lo dicen explícitamente. Al volver a pendiente o cancelarse se borra. Mover el turno
de horario borra `cambioPedidoEn` y lo de llamar; cancelarlo, atenderlo o marcarlo ausente borra lo
de llamar.

## Endpoints

Errores siempre `application/problem+json` con `code` estable. Panel: `Authorization` + `X-Negocio`.

### `GET /v1/panel/ajustes/whatsapp`
Roles: duenio, encargado, recepcion, profesional.

```json
{ "disponible": true, "confirmacion": true, "hora": "10:00", "sinRespuestaHoras": 4, "recordatorio": false }
```
`disponible`: la plataforma tiene WhatsApp configurado (si es false, el front muestra la opción
apagada y sin poder prenderla).

### `PUT /v1/panel/ajustes/whatsapp`
Roles: duenio, encargado. Cuerpo: lo mismo sin `disponible` (si viene, se ignora). `hora` vacía
vale 10:00. Responde 200 con los ajustes guardados. Queda en la auditoría (`ajustes.whatsapp`).

Errores: 400 `invalido` (`hora` fuera de 07:00–20:00 o mal escrita; `sinRespuestaHoras` fuera de
1–12); 409 `whatsapp_apagado` (prender algo sin WhatsApp en la plataforma); 402
`suscripcion_requerida` con el panel en sólo lectura.

### `GET /v1/panel/whatsapp/mensajes`
Roles: duenio, encargado, recepcion. Filtros: `turno` (id), `tipo` (`confirmacion`,
`recordatorio`, `respuesta`, separados por coma), `antes` (RFC 3339, el `creadoEn` del último de la
página). Los últimos 50, del más nuevo al más viejo.

```json
{ "mensajes": [ {
  "id": "…", "tipo": "confirmacion", "para": "+5491123456789", "turnoId": "…", "cliente": "Sofía Méndez",
  "estado": "enviado", "intentos": 1, "ultimoError": null,
  "enviadoEn": "…", "entregadoEn": "…", "leidoEn": "…",
  "respuesta": "confirmo", "respondidoEn": "…", "sinRespuestaEn": null, "creadoEn": "…" } ] }
```
`estado`: `pendiente` | `enviado` | `descartado` (con el motivo en `ultimoError`) | `fallido`.
`respuesta`: `confirmo` | `cambiar` | null.

### `GET /v1/panel/para-llamar`
Roles: los cuatro. Los turnos que recepción tiene que llamar (hasta 100, por horario).

```json
{ "turnos": [ { "turnoId": "…", "cliente": "Nadia Paz", "telefono": "0351155123456", "servicio": "Lifting",
  "profesional": "Abril", "inicio": "…", "estado": "pendiente", "motivo": "cambio", "desde": "…" } ] }
```
`motivo`: `cambio` | `sin_respuesta` | `no_llego`.

### `POST /v1/panel/turnos/{id}/llamado`
Roles: los cuatro. Ya lo llamaron: el turno sale de la lista. Sin cuerpo; 204. 404 si no existe.
Auditoría `turno.llamado`.

### `POST /v1/t/{codigo}/confirmar` (público, link del cliente)
El cliente confirma desde `/turno/{codigo}`. Sin cuerpo. Deja `confirmadoPor: "web"`, saca lo de
llamar y suma la campanita `confirmacion`. Si ya había confirmado él (web o WhatsApp), responde lo
mismo sin cambiar nada.

```json
{ "estado": "confirmado", "confirmadoPor": "web", "confirmadoEn": "…" }
```
Errores: 404 `no_encontrado`; 409 `estado` (cancelado, atendido o ya pasó).

### `GET /v1/webhooks/whatsapp` (Meta)
Verificación de la suscripción: con `hub.mode=subscribe` y `hub.verify_token` igual a
`TENGOLIBRE_WHATSAPP_VERIFICACION`, responde 200 `text/plain` con `hub.challenge`. Si no, 403.

### `POST /v1/webhooks/whatsapp` (Meta)
Firmado con `X-Hub-Signature-256` (HMAC-SHA256 del cuerpo con el secreto de la app). Procesa los
botones (`type: button` de las plantillas, `interactive.button_reply`) y los estados (`delivered`,
`read`, `failed`) del número de la plataforma; lo demás se ignora. Todo es idempotente (Meta repite
webhooks). 200 `{"ok":true}`; 401 `firma_invalida`; 400 si no es JSON; 404 sin
`TENGOLIBRE_WHATSAPP_SECRETO_APP`; 500 si falló la base (Meta reintenta).

## Trabajos del worker

| Nombre | Cada | Qué hace |
|---|---|---|
| `whatsapp_envios` | 5 s | Manda la cola (plantillas y respuestas). Reintenta lo reintentable (límites, Meta caído) con la espera de los mails, hasta 6 intentos; un número sin WhatsApp o una plantilla mal falla de una. |
| `whatsapp_confirmaciones` | 5 min | Encola la confirmación de mañana desde la hora del negocio hasta las 21. |
| `whatsapp_recordatorios` | 5 min | Encola el recordatorio de 24 h antes (si está prendido y la confirmación no). |
| `whatsapp_sin_respuesta` | 5 min | Marca para llamar lo que no contestó en N horas. |
| `whatsapp_purga` | 24 h | Borra los mensajes resueltos de más de 60 días. |

Sin `TENGOLIBRE_WHATSAPP_TOKEN` o `TENGOLIBRE_WHATSAPP_TELEFONO_ID`, el worker no suma estos
trabajos (lo dice el log) y la API responde `disponible: false`.

## Configuración

| Variable | Qué es |
|---|---|
| `TENGOLIBRE_WHATSAPP_TOKEN` | Token permanente de un usuario del sistema de la cuenta comercial (permisos `whatsapp_business_messaging` y `whatsapp_business_management`). API y worker. |
| `TENGOLIBRE_WHATSAPP_TELEFONO_ID` | Phone number ID del número de TengoLibre (no el número). API y worker. |
| `TENGOLIBRE_WHATSAPP_SECRETO_APP` | App secret de la app de Meta (Configuración › Básica): valida la firma del webhook. API. |
| `TENGOLIBRE_WHATSAPP_VERIFICACION` | Un texto al azar que se carga como «Token de verificación» al suscribir el webhook. API. |
| `TENGOLIBRE_WHATSAPP_VERSION` | Versión de la Graph API (por defecto `v23.0`). |

## Lo que hay que hacer en Meta

1. **App** de tipo Empresa en developers.facebook.com con el producto WhatsApp, atada a la cuenta
   comercial de Grupo Vimo / TengoLibre, y verificación del negocio aprobada (sin eso, el límite es
   de 250 conversaciones por día y sólo a números de prueba).
2. **Número**: dar de alta el número de TengoLibre en la cuenta de WhatsApp Business, con nombre
   visible «TengoLibre» aprobado. Copiar su *Phone number ID* a `TENGOLIBRE_WHATSAPP_TELEFONO_ID`.
3. **Token**: Business Settings › Usuarios del sistema › crear uno (admin), asignarle la app y la
   cuenta de WhatsApp, y generar un token sin vencimiento con `whatsapp_business_messaging` y
   `whatsapp_business_management` → `TENGOLIBRE_WHATSAPP_TOKEN`.
4. **Webhook**: en la app, WhatsApp › Configuración › Webhook: URL
   `https://api.tengolibre.com.ar/v1/webhooks/whatsapp`, token de verificación =
   `TENGOLIBRE_WHATSAPP_VERIFICACION`; suscribir el campo **messages**. El App secret va a
   `TENGOLIBRE_WHATSAPP_SECRETO_APP`.
5. **Plantillas** (WhatsApp Manager › Plantillas de mensajes), las dos en categoría **Utilidad**:

### `tengolibre_confirmacion`, idioma `es_AR` (Español (ARG))

Cuerpo:
```
Hola {{1}} 👋 Te esperamos mañana {{2}} a las {{3}} en {{4}} para tu {{5}} con {{6}}. ¿Venís?
```
Ejemplos para la revisión: `Sofía`, `lunes 2`, `10:30`, `Lumi Estética`, `lifting`, `Abril`.

Botones (respuesta rápida, en este orden): **Confirmo** · **Necesito cambiarlo**. El payload de
cada uno lo pone la API al mandar (`confirmo:<token>` / `cambiar:<token>`): no hace falta cargarlo.

Parámetros: 1 nombre del cliente, 2 día («lunes 2»), 3 hora, 4 negocio, 5 servicio (en minúscula),
6 profesional («el equipo» si no hay).

### `tengolibre_recordatorio`, idioma `es_AR`

Cuerpo:
```
Hola {{1}}, te recordamos tu turno en {{2}}: {{3}}, el {{4}} a las {{5}}.
```
Ejemplos: `Sofía`, `Lumi Estética`, `Lifting de pestañas`, `lunes 2 de noviembre`, `10:30`.

Botón de link (**Visitar sitio web**, URL dinámica): texto **Ver mi turno**, URL
`https://tengolibre.com.ar/turno/{{1}}` (ejemplo: `https://tengolibre.com.ar/turno/7K2QabcdEFGH1234`).
La API completa `{{1}}` con el código del turno.

Si se cambia un nombre o el orden de los parámetros, hay que cambiarlo también en
`internal/dominio/confirmaciones/textos.go`.

Las respuestas a los botones («¡Listo! Queda confirmado…», «Dale…», «Ese turno ya no está
vigente…») son textos libres: van dentro de las 24 horas que abre el botón y no necesitan plantilla.

## Pendiente

- Número propio por negocio (que el chat diga «Lumi Estética · Cuenta de empresa», como en el
  diseño): necesita el registro integrado (Embedded Signup) de Meta y plantillas por cuenta. Hoy
  sale todo desde el número de TengoLibre, con el nombre del negocio en el texto.
- Respuestas escritas a mano («sí», «dale»): hoy sólo cuentan los botones.
