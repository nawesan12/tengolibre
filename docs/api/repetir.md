# Volver a reservar en un toque (propuesta 04)

En **Mi turno** ya atendido (`/turno/{codigo}`), el próximo turno viene armado: mismo servicio, mismo
profesional, a las N semanas y el horario más parecido, con tres fechas cercanas más. Se confirma sin
pasar por los pasos de la reserva: los datos del cliente ya los tiene el negocio.

Código: `apps/api/internal/dominio/turnos/repetir.go`. El monto de la seña lo estima cobros
(`cobros.Servicio.Estimar`, interfaz `turnos.Estimador`).

Rutas públicas, sin sesión: el código del link es el secreto (como el resto de `/v1/t/{codigo}`).

## GET /v1/t/{codigo}/repetir

Respuesta 200:

```json
{
  "semanas": 4,
  "habitual": false,
  "servicio": { "id": "uuid", "nombre": "Lifting de pestañas", "precio": 22000, "duracion": 60 },
  "profesional": { "id": "uuid", "nombre": "Abril" },
  "sugerencia": { "inicio": "2026-11-30T13:30:00Z", "fecha": "2026-11-30", "hora": "10:30", "mismoHorario": true },
  "alternativas": [
    { "inicio": "2026-11-23T13:30:00Z", "fecha": "2026-11-23", "hora": "10:30", "mismoHorario": true },
    { "inicio": "2026-12-07T14:00:00Z", "fecha": "2026-12-07", "hora": "11:00", "mismoHorario": false }
  ],
  "pago": { "monto": 6600, "modo": "porcentaje" }
}
```

- `semanas`: la frecuencia habitual del cliente, la mediana entre sus turnos atendidos en el negocio
  (redondeada a semanas, de 1 a 12; dos turnos a menos de 4 días no cuentan). Sin historia, 4 y
  `habitual: false`.
- El objetivo es el turno atendido + `semanas`, a la misma hora (si ya pasó, la primera semana
  siguiente que no pasó). Los días se miran en este orden: ese día, el mismo día de la semana una
  semana antes y después, dos semanas antes y después, y los días de alrededor (±1 a ±6). En cada día
  se toma el horario libre más cercano a la hora original (si empatan, el más tarde) con el mismo
  profesional y las reglas de la web (vista, anticipación, margen).
- `sugerencia`: el primero de esos días que tiene lugar; `null` si no hay nada.
- `alternativas`: hasta 3 días más de esa lista, ordenados por fecha (puede haber menos).
- `mismoHorario`: misma hora que el turno atendido.
- `pago`: lo que se cobraría por adelantado sobre el precio actual del servicio (seña o total, con
  la configuración y la cuenta de Mercado Pago del negocio, igual que la reserva pública); `null` si
  no se cobra.

Errores:
- 404 `no_encontrado`: no existe el código.
- 409 `no_atendido`: el turno no está atendido.
- 409 `no_disponible`: el servicio o el profesional ya no existen, están inactivos o el profesional
  dejó de hacer el servicio (mostrar «sacá un turno nuevo desde la página»).

## POST /v1/t/{codigo}/repetir

Cuerpo: `{ "inicio": "2026-11-30T13:30:00Z" }` (el de la sugerencia o el de una alternativa; también
sirve otro horario libre del mismo profesional).

Reserva como la página pública (`POST /v1/p/{slug}/turnos`): mismo cliente, servicio y profesional,
origen `web`, y las mismas reglas (penalidad por ausencias, disponibilidad recalculada con el
profesional bloqueado). Si el negocio cobra, queda `pendiente_pago` por 15 minutos y trae el link de
Checkout Pro; si Mercado Pago no respondió, `pago.url` viene vacío y se pide con
`POST /v1/t/{codigo-nuevo}/pagar`. Avisa como cualquier reserva (mail y campanita).

Respuesta 201: el `Turno` nuevo (sin `cliente.id`), con su propio `codigo` para el link:

```json
{ "id": "uuid", "codigo": "…", "estado": "pendiente_pago", "venceEn": "…", "pago": { "monto": 6600, "modo": "porcentaje", "url": "https://…" }, "...": "Turno" }
```

Errores:
- 400 `invalido`: falta `inicio`.
- 404 `no_encontrado`: no existe el código.
- 409 `no_atendido`, `no_disponible`: como arriba.
- 409 `horario_ocupado`: ese horario ya no está libre.
- 409 `cliente_penalizado`: el cliente tiene una penalidad vigente.
