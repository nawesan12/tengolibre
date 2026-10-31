# Pedir días libres (propuesta 06)

Un profesional con usuario vinculado pide días libres desde su panel (desde/hasta y motivo) y ve
cuántos turnos tiene esos días. La dueña o encargada lo ve en la campanita y en la lista de pedidos:
**aprobar** crea los bloqueos del profesional esos días y pasa sus turnos a la lista «para
reprogramar»; **rechazar** lo cierra. Recepción mueve esos turnos y los marca resueltos. Código:
`apps/api/internal/dominio/diaslibres`, migración `00013`.

Convenciones: panel con `Authorization` + `X-Negocio`; errores `application/problem+json` con `code`;
fechas `AAAA-MM-DD` en la zona del negocio. Tiempo real por SSE: `tipo: "dias_libres"` (pedidos) y
`tipo: "reprogramar"` (la lista para mover). La campanita usa el tipo `equipo`.

«Turnos activos» = `pendiente_pago`, `pendiente` o `confirmado`.

## Objetos

### Pedido

```json
{
  "id": "uuid",
  "profesionalId": "uuid", "profesional": "Abril",
  "desde": "2026-11-14", "hasta": "2026-11-18",
  "motivo": "vacaciones",          // vacaciones | tramite | salud | otro
  "nota": "",
  "estado": "pendiente",           // pendiente | aprobado | rechazado | cancelado
  "turnos": 3,                     // los que tenía esos días al pedirlo
  "afectados": 3,                  // los activos que hay ahora esos días («3 turnos para mover»)
  "respuesta": "",                 // lo que escribió quien respondió
  "respondidoEn": null,
  "creadoEn": "…"
}
```

### TurnoAfectado

```json
{ "id": "uuid", "codigo": "…", "inicio": "…", "fin": "…", "servicio": "Lifting", "estado": "confirmado",
  "cliente": "Sofía Méndez", "telefono": "1123456789" }
```

### Reprogramar (un turno de la lista para mover)

```json
{
  "id": "uuid", "pedidoId": "uuid", "motivo": "vacaciones",
  "resueltoEn": null, "resueltoPor": null,   // clerk id de quien lo marcó, o "automatico"
  "creadoEn": "…",
  "turno": { "id", "codigo", "inicio", "fin", "servicio", "profesionalId", "profesional", "estado",
             "cliente": { "nombre", "telefono", "email" } }
}
```

Un turno de la lista se resuelve **solo** (`resueltoPor: "automatico"`) si se cancela o se mueve de
horario o de profesional.

## Endpoints

### `GET /v1/panel/dias-libres/turnos?desde=&hasta=[&profesional=]`

«Ya tenés 3 turnos esos días». Roles: todos los del panel. El profesional ve los suyos (ignora
`profesional`); los demás pasan `profesional=` (sin él, el propio si tienen uno vinculado). `hasta`
opcional (= `desde`).
200 `{ "cantidad": 3, "turnos": [TurnoAfectado] }`. 400 `invalido` (fechas mal, desde antes de hoy,
más de 61 días); 409 `sin_profesional`.

### `POST /v1/panel/dias-libres` — «Enviar a Carla»

Roles: cualquiera del panel con un profesional vinculado a su usuario (en la práctica, rol profesional).
```json
{ "desde": "2026-11-14", "hasta": "2026-11-18", "motivo": "vacaciones", "nota": "opcional, hasta 300" }
```
`hasta` opcional (un solo día). 201 `Pedido`. Suma a la campanita «Abril pide días libres · Del sábado
14 de noviembre al miércoles 18 de noviembre · vacaciones · 3 turnos para mover.».
Errores: 400 `invalido` (motivo, fechas, nota larga); 409 `sin_profesional` (el usuario no está
vinculado a un profesional), `superpuesto` (ya tiene un pedido pendiente o aprobado que se cruza).

### `GET /v1/panel/dias-libres[?estado=pendiente,aprobado]`

Del más nuevo al más viejo (hasta 200). El profesional ve sólo los suyos; el resto, todos.
200 `{ "pedidos": [Pedido] }`.

### `GET /v1/panel/dias-libres/{id}`

200 `Pedido` + `"turnosAfectados": [TurnoAfectado]`. 404 si no existe o (profesional) no es suyo.

### `POST /v1/panel/dias-libres/{id}/aprobar` — «Aprobar y ver turnos»

Roles: dueño/a y encargado/a. Cuerpo opcional `{ "respuesta": "¡Disfrutá!" }` (hasta 300).
Hace: un bloqueo de día entero por cada día (motivo «Días libres (vacaciones)»; salta los días que ya
estaban bloqueados enteros), los turnos activos de esos días a la lista para reprogramar, y deja de
ofrecer a la lista de espera los huecos de ese profesional esos días. Campanita: «Aprobados los días
libres de Abril · … 3 turnos pasan a la lista para reprogramar.».
200 `{ "pedido": Pedido, "reprogramar": [Reprogramar] }`.
Errores: 403 (otros roles); 404; 409 `ya_respondido`.

### `POST /v1/panel/dias-libres/{id}/rechazar` — «Rechazar»

Roles: dueño/a y encargado/a. Cuerpo opcional `{ "respuesta": "…" }`. 200 `Pedido` (`rechazado`).
Campanita: «No se aprobaron los días libres de Abril». 403, 404, 409 `ya_respondido`.

### `POST /v1/panel/dias-libres/{id}/cancelar`

Quien lo pidió (sólo los suyos) o dueño/a / encargado/a, mientras está pendiente. Recepción: 403.
200 `Pedido` (`cancelado`). 404, 409 `ya_respondido`.

### `GET /v1/panel/reprogramar[?todos=1][&pedido={id}]`

La lista «para reprogramar», ordenada por hora del turno. Roles: dueño/a, encargado/a, recepción.
Sin `todos`: sólo los que faltan. 200 `{ "turnos": [Reprogramar] }`.
Para moverlos se usa lo de siempre (`POST /v1/panel/turnos/{id}/reprogramar`, o cancelar con
`PATCH /v1/panel/turnos/{id}`): salen solos de la lista.

### `POST /v1/panel/reprogramar/{id}/resolver`

Recepción lo marca resuelto a mano (lo habló con el cliente, lo dejó así). Sin cuerpo. Idempotente.
200 `Reprogramar`. 404 si no existe.
