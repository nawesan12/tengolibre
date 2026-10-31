# Horas muertas (Reportes · propuesta 11)

La semana como mapa de calor: qué parte de la capacidad de cada hora quedó sin usar en las últimas
8 semanas, la franja peor y un cupón sugerido para llenarla. Código:
`apps/api/internal/dominio/reportes/horas.go`; las restricciones de cupón, en
`internal/dominio/promos/franja.go`. Migración `00017`.

Todas las rutas son del panel: `Authorization: Bearer …` y `X-Negocio: <id>`. Errores en
`application/problem+json` con `code` (`invalido` = 400, `duplicado`/`sin_franja` = 409,
`suscripcion_requerida` = 402 con el panel en sólo lectura).

## Cómo se calcula

- Ventana: las 8 semanas enteras anteriores a hoy (hoy no entra), en la zona del negocio.
- **Capacidad** de una celda (día de la semana × hora, p. ej. martes 14 = de 14 a 15): los minutos
  del horario semanal de cada profesional activo que caen en esa hora, sumados en las 8 semanas,
  menos los días bloqueados enteros y lo que tapan los bloqueos parciales.
- **Ocupado**: los minutos de esa capacidad tapados por turnos no cancelados (un `pendiente_pago`
  vencido no cuenta). Un sobreturno fuera de horario no suma.
- **Vacío** = (capacidad − ocupado) / capacidad, en porcentaje con un decimal. «Vacío el 72% de las
  veces» es eso, redondeado.
- **Peor franja**: las dos horas seguidas del mismo día con más vacío (con al menos 120 minutos de
  capacidad entre las dos); si ningún día tiene dos horas seguidas, una sola.
- **Sugerencia** (sólo si la peor franja queda vacía el 40% o más): 20% de descuento si el vacío es
  ≥ 80%, 15% si es ≥ 50%, 10% si no; para clientes cuyo primer turno atendido fue hace más de 60 días.

## GET /v1/panel/reportes/horas-muertas

Roles: `duenio`, `encargado`.

```json
{
  "desde": "2026-09-06",
  "hasta": "2026-10-31",
  "semanas": 8,
  "mapa": {
    "dias": [1, 2, 3, 4, 5, 6],
    "horas": [10, 11, 12, 13, 14, 15, 16, 17, 18, 19],
    "celdas": [
      { "dia": 2, "hora": 14, "capacidadMinutos": 480, "ocupadoMinutos": 60,
        "ocupacion": 12.5, "vacio": 87.5, "estado": "suele_quedar_vacio" }
    ],
    "capacidadMinutos": 2880,
    "ocupadoMinutos": 1980
  },
  "peor": {
    "dia": 2, "horaDesde": 14, "horaHasta": 16, "vacio": 72.4, "libreMinutos": 695,
    "texto": "Martes de 14 a 16: vacío el 72% de las veces"
  },
  "sugerencia": {
    "porcentaje": 15, "dias": [2], "horaDesde": "14:00", "horaHasta": "16:00",
    "antiguedadDias": 60, "destinatarios": 46, "codigo": "MARTES14",
    "texto": "Probá un 15% sólo en esa franja, para clientes que vienen hace más de 2 meses."
  }
}
```

- `dias`: de lunes a domingo (0 = domingo), sólo los que tienen horario. `horas`: de la primera a la
  última con horario. `celdas`: una por día × hora, día por día.
- `estado`: `suele_quedar_vacio` (vacío ≥ 50%), `ocupado`, o `cerrado` (sin horario esa hora:
  `capacidadMinutos` 0 y `ocupacion`/`vacio` en `null`). En el diseño: rayado, petróleo y «cerrado».
- `peor` y `sugerencia` pueden venir en `null` (sin horarios, o nada queda tan vacío).
- `destinatarios`: clientes con email, sin baja de mails, que vienen hace más de `antiguedadDias`.

Errores: 403 si el rol no es de gestión.

## POST /v1/panel/reportes/horas-muertas/cupon

Roles: `duenio`, `encargado`. Crea el cupón de la sugerencia, listo («Crear cupón»). El cuerpo es
opcional; cada campo pisa la sugerencia:

```json
{
  "codigo": "MARTES14",
  "porcentaje": 15,
  "dias": [2],
  "horaDesde": "14:00",
  "horaHasta": "16:00",
  "antiguedadDias": 60,
  "vigenciaDias": 30
}
```

- Sin `codigo`, se usa el sugerido y, si ya existe, `MARTES14-2`, `-3`…
- Vigencia por defecto: 30 días desde ahora. Tipo siempre `porcentaje`.
- Queda en la auditoría como `cupon.horas_muertas`.

Respuesta `201`: el cupón, como en `GET /v1/panel/cupones` (ver abajo).

Errores: `409 sin_franja` si no hay sugerencia y no vinieron `dias`, `horaDesde` y `horaHasta`;
`409 duplicado` si el `codigo` pedido ya existe; `400 invalido` (porcentaje fuera de 1–100, franja
que termina antes de empezar, días fuera de 0–6, vigencia fuera de 1–365).

## Cupones restringidos (promos)

Todo cupón (`GET/POST /v1/panel/cupones`, `PATCH /v1/panel/cupones/{id}`) suma cuatro campos
opcionales y un texto armado:

```json
{
  "id": "…", "codigo": "MARTES14", "tipo": "porcentaje", "valor": 15,
  "dias": [2],
  "horaDesde": "14:00",
  "horaHasta": "16:00",
  "antiguedadDias": 60,
  "condiciones": "Vale los martes de 14 a 16 h, para clientes que vienen hace más de 2 meses."
}
```

- `dias`: días de la semana en que vale (0 = domingo); `[]` = todos.
- `horaDesde`/`horaHasta`: el turno tiene que **empezar** en `[horaDesde, horaHasta)`, hora local.
  Van juntas o ninguna.
- `antiguedadDias` (1–3650): el primer turno atendido del cliente fue hace al menos esos días.
- En `PATCH`, `"dias": []`, `"horaDesde": ""`/`"horaHasta": ""` y `"antiguedadDias": 0` sacan cada
  restricción. Un cupón sin restricciones vale como antes; `condiciones` viene `""`.

Al reservar (`POST /v1/p/{slug}/turnos` con `cupon`), un cupón fuera de su franja o para un cliente
sin la antigüedad responde `422 cupon_invalido` («Este cupón vale sólo los martes de 14 a 16 h.»,
«Este cupón es para clientes que vienen hace más de 2 meses.») y la reserva no se hace.

`POST /v1/p/{slug}/cupones/validar` acepta además `"inicio"` (RFC 3339, opcional): si viene, chequea
la franja. La respuesta suma `condiciones` (la antigüedad se chequea recién al reservar, cuando se
sabe quién es el cliente).
