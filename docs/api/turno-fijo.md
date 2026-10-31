# Turno fijo (propuesta 03)

Dentro de **Nuevo turno** del panel: se elige cada cuánto se repite (cada 1, 2, 3 o 4 semanas) y
hasta cuándo (una fecha o una cantidad). Primero se pide la vista previa; después se reservan las
fechas que el usuario aceptó (con las alternativas elegidas), todas juntas o ninguna.

Código: `apps/api/internal/dominio/turnos/serie.go`. Migración `00015_serie_y_caja.sql` (tabla
`series`, columna `turnos.serie_id`).

Todas las rutas son del panel: `Authorization` + `X-Negocio`, roles de agenda (`duenio`,
`encargado`, `recepcion`, `profesional`). Con la prueba vencida, las escrituras responden 402
`suscripcion_requerida` como el resto del panel.

Las reglas de disponibilidad son las de siempre (horario, bloqueos, margen, anticipación, otros
turnos). Lo único distinto: la vista (`dias_vista`, 30 por defecto) se estira hasta la última fecha
de la serie, porque el panel reserva más allá de lo que ve la web. Topes: 52 turnos y un año entre
el primero y el último.

Todo turno (`Turno` del panel, del link del cliente, de la agenda) trae ahora `serieId`
(`string | null`).

## POST /v1/panel/series/previa

Cuerpo:

```json
{
  "servicioId": "uuid",
  "profesionalId": "uuid",
  "inicio": "2026-11-03T18:00:00-03:00",
  "cadaSemanas": 2,
  "hasta": "2026-12-29",
  "cantidad": 5
}
```

- `profesionalId` es obligatorio (la serie es con un profesional).
- `cadaSemanas`: 1 a 4.
- Va `hasta` (AAAA-MM-DD, inclusive) **o** `cantidad` (2 a 52, contando el primero), no los dos.
- Las fechas son `inicio` + k × `cadaSemanas` semanas, a la misma hora del reloj del negocio.

Respuesta 200:

```json
{
  "cadaSemanas": 2,
  "servicioId": "uuid", "servicio": "Corte y barba",
  "profesionalId": "uuid", "profesional": "Leo",
  "libres": 3,
  "chocan": 2,
  "fechas": [
    { "inicio": "2026-11-03T21:00:00Z", "fecha": "2026-11-03", "hora": "18:00", "estado": "libre", "alternativa": null },
    { "inicio": "2026-12-01T21:00:00Z", "fecha": "2026-12-01", "hora": "18:00", "estado": "choca",
      "motivo": "bloqueado", "alternativa": null },
    { "inicio": "2026-12-29T21:00:00Z", "fecha": "2026-12-29", "hora": "18:00", "estado": "choca",
      "motivo": "ocupado", "alternativa": { "inicio": "2026-12-29T21:30:00Z", "hora": "18:30" } }
  ]
}
```

- `estado`: `libre` o `choca`.
- `motivo` (sólo si choca): `ocupado` (otro turno del profesional), `bloqueado` (bloqueo,
  vacaciones o un horario bloqueado), `no_atiende` (fuera de su horario ese día), `pasado` (ya pasó o
  no llega a la anticipación mínima).
- `alternativa` (sólo si choca): el horario libre más cercano **de ese mismo día** con el mismo
  profesional (si empatan, el más tarde), o `null` si ese día no le queda nada.

Errores: 400 `invalido` (cadaSemanas, hasta/cantidad, fechas fuera de rango, falta el profesional;
`issues[].path` dice cuál), 404 `no_encontrado` (servicio o profesional inexistente, inactivo o que no
hace el servicio).

## POST /v1/panel/series

Reserva la serie de forma atómica. Cuerpo:

```json
{
  "servicioId": "uuid",
  "profesionalId": "uuid",
  "cadaSemanas": 2,
  "cliente": { "nombre": "Tomás Aguirre", "telefono": "1155556666", "email": "" },
  "clienteId": "uuid",
  "nota": "",
  "turnos": [
    { "inicio": "2026-11-03T21:00:00Z" },
    { "inicio": "2026-12-29T21:30:00Z" }
  ]
}
```

- `turnos`: los inicios que se aceptaron (de 1 a 52), en orden y sin repetir; para una fecha que
  chocaba va el `inicio` de su alternativa. Las que se descartaron no van.
- El cliente va como `cliente` (se busca por teléfono o se crea, como en `POST /v1/panel/turnos`) o
  como `clienteId` de uno que ya está en el negocio.
- Cada turno pasa por la misma reserva que `POST /v1/panel/turnos` (estado `confirmado`, extras de
  membresía incluidos) y queda con `serieId`.
- Avisos: un solo mail de reserva, el del primer turno (los recordatorios de cada turno llegan como
  siempre). Queda en la auditoría como `serie.creada`.

Respuesta 201:

```json
{
  "id": "uuid",
  "cadaSemanas": 2,
  "clienteId": "uuid",
  "creadoEn": "2026-11-01T13:00:00Z",
  "turnos": [ { "id": "uuid", "codigo": "…", "serieId": "uuid", "estado": "confirmado", "...": "Turno" } ]
}
```

Errores:
- 400 `invalido`: cadaSemanas, cantidad de turnos, orden, más de un año, nota larga, datos del
  cliente.
- 404 `no_encontrado`: servicio, profesional o `clienteId`.
- 409 `horario_ocupado`: alguno ya no está libre. No se reserva **ninguno**. `issues[0].path` dice
  cuál (`turnos[2].inicio`) y `detail` la fecha y hora: conviene volver a pedir la vista previa.

## GET /v1/panel/series/{id}

La serie con todos sus turnos (los cancelados también), en orden. Respuesta 200 como la de arriba.
404 `no_encontrado`.

## POST /v1/panel/turnos/{id}/cancelar-siguientes

«Este y los siguientes»: cancela el turno `{id}` y los de su serie que vienen después y siguen
activos (`pendiente_pago`, `pendiente`, `confirmado`). Los atendidos, ausentes y ya cancelados no se
tocan. Sin cuerpo. Cada cancelación corre los extras de siempre (devuelve la visita de la membresía,
anula el pago pendiente) y avisa como una cancelación del panel (mail al cliente, lista de espera,
auditoría).

Respuesta 200:

```json
{ "cancelados": ["uuid", "uuid"], "serie": { "id": "uuid", "cadaSemanas": 2, "clienteId": "uuid", "creadoEn": "…", "turnos": [] } }
```

Errores: 404 `no_encontrado` (turno), 409 `sin_serie` (el turno no es parte de un turno fijo).
