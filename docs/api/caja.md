# Cobrar en el local y sumar productos (propuestas 07 y 08)

Al marcar un turno **Atendido** (`PATCH /v1/panel/turnos/{id}` con `estado: "atendido"`) el front
abre la caja: lo que resta cobrar, los productos que se llevó, el medio de pago, y registra el cobro.
También el catálogo de productos del negocio y el resumen de caja del día. Los reportes suman la
plata real cobrada.

Código: `apps/api/internal/dominio/caja`. Migración `00015_serie_y_caja.sql` (tablas `productos`,
`caja_cobros`, `caja_items`). Los cambios de `caja_cobros` salen por el SSE del panel con
`tipo: "caja_cobros"`.

Todas las rutas son del panel (`Authorization` + `X-Negocio`). Con la prueba vencida, las escrituras
responden 402 `suscripcion_requerida`. Montos en pesos enteros.

Roles:
- Agenda (`duenio`, `encargado`, `recepcion`, `profesional`): ver productos, ver la caja de un turno,
  cobrar y saldar.
- Gestión (`duenio`, `encargado`): crear, editar y borrar productos; anular un cobro.
- Caja (`duenio`, `encargado`, `recepcion`): resumen del día.

## Productos

```json
{ "id": "uuid", "nombre": "Sérum de pestañas", "precio": 9000, "stock": 4, "activo": true, "orden": 0 }
```

`stock: null`: no se lleva la cuenta. Con stock, cobrar lo descuenta y anular lo devuelve.

### GET /v1/panel/productos

`?activos=1` deja sólo los que se pueden vender. Respuesta 200: `{ "productos": [Producto] }`,
ordenados por `orden` y nombre.

### POST /v1/panel/productos

Cuerpo: `{ "nombre": "Sérum", "precio": 9000, "stock": 4, "activo": true, "orden": 0 }` (`nombre` de
2 a 80 y `precio` de 0 a 100.000.000 obligatorios; `stock` opcional, 0 a 1.000.000). 201 con el
producto. Errores: 400 `invalido`, 409 `duplicado` (ya hay uno con ese nombre).

### PATCH /v1/panel/productos/{id}

Lo que no viene no cambia. `"sinStock": true` deja de llevar la cuenta (no va junto con `stock`).
200 con el producto. Errores: 400 `invalido`, 404 `no_encontrado`, 409 `duplicado`.

### DELETE /v1/panel/productos/{id}

204. Lo ya vendido conserva nombre y precio. 404 `no_encontrado`.

## La caja de un turno

```json
{
  "turnoId": "uuid",
  "cliente": "Sofía Méndez",
  "servicio": "Lifting de pestañas",
  "estadoTurno": "atendido",
  "estado": "sin_cobrar",
  "medio": null,
  "cobradoEn": null,
  "lineas": [
    { "tipo": "servicio", "concepto": "Lifting de pestañas", "monto": 22000 },
    { "tipo": "sena", "concepto": "Seña pagada (Mercado Pago)", "monto": -6600 },
    { "tipo": "producto", "concepto": "Sérum de pestañas", "monto": 9000, "cantidad": 1, "productoId": "uuid" }
  ],
  "total": 24400,
  "medios": ["efectivo", "transferencia", "debito", "credito", "mercadopago", "despues"]
}
```

- `estado`: `sin_cobrar`, `debe` (se eligió «Después») o `cobrado`.
- `lineas[].tipo`: `servicio` (precio de lista: el del turno más el cupón), `cupon`, `membresia`
  (el turno lo cubre una membresía), `sena` (la seña aprobada en Mercado Pago, neta de devoluciones),
  `producto`, `descuento` (el que se puso en la caja). Lo que descuenta va en negativo.
- `total`: resta cobrar = máx(0, servicio − cupón − membresía − seña) + productos − descuento. Una vez
  cobrada (o debida), la cuenta queda guardada tal cual: si después cambian los precios, no cambia.

### GET /v1/panel/turnos/{id}/caja

200 con la cuenta (sin cobrar: la de hoy, sin productos). 404 `no_encontrado`.

### POST /v1/panel/turnos/{id}/caja

Registra el cobro. Cuerpo:

```json
{
  "productos": [{ "productoId": "uuid", "cantidad": 1 }],
  "descuento": 0,
  "medio": "efectivo"
}
```

- `medio`: `efectivo`, `transferencia`, `debito`, `credito`, `mercadopago` o `despues` (queda
  debiendo: `estado: "debe"`, `cobradoEn: null`).
- `productos`: hasta 30 líneas, `cantidad` de 1 a 99; el mismo producto dos veces se suma. El
  precio sale del catálogo, nunca del cuerpo.
- `descuento`: opcional, no más que lo que resta cobrar.
- Queda en la auditoría como `caja.cobrado` o `caja.debe`.

201 con la cuenta. Errores:
- 400 `invalido`: medio, cantidades, descuento negativo o mayor al total.
- 404 `no_encontrado`: el turno, o un producto que no existe o está inactivo.
- 409 `no_atendido`: el turno no está atendido.
- 409 `sin_stock`: no alcanza el stock de un producto (`detail` dice cuál).
- 409 `ya_cobrado`: ya tiene cobro; 409 `debe`: quedó debiendo (usar `/saldar`).

### POST /v1/panel/turnos/{id}/caja/saldar

Cobra lo que quedó debiendo. Cuerpo: `{ "medio": "transferencia" }` (cualquiera menos `despues`).
Entra en la caja del día en que se paga. Auditoría `caja.saldado`. 200 con la cuenta.
Errores: 400 `invalido`, 409 `sin_cobrar` (no pasó por la caja), 409 `ya_cobrado`.

### DELETE /v1/panel/turnos/{id}/caja

Anula el cobro (o la deuda) que se cargó mal: devuelve el stock y la caja vuelve a `sin_cobrar`.
Auditoría `caja.anulado`. 200 con la cuenta. 404 `no_encontrado` si no había cobro.

## GET /v1/panel/caja

Resumen de caja de un día: `?fecha=AAAA-MM-DD` (por defecto hoy, en la zona del negocio).

```json
{
  "fecha": "2026-11-02",
  "porMedio": { "efectivo": 13000, "transferencia": 7000, "debito": 0, "credito": 0, "mercadopago": 0 },
  "caja": 20000,
  "senas": 4000,
  "total": 24000,
  "productos": 9000,
  "cobros": 2,
  "debe": 0,
  "movimientos": [
    { "turnoId": "uuid", "cliente": "Sofía Méndez", "servicio": "Lifting", "medio": "efectivo", "total": 13000,
      "productos": 9000, "cobradoEn": "2026-11-02T15:00:00Z" }
  ]
}
```

- `porMedio` (siempre con los cinco medios) y `caja`: lo cobrado en el local ese día (por
  `cobradoEn`; lo saldado cuenta el día que se pagó).
- `senas`: las señas online aprobadas ese día, netas de devoluciones (por la fecha del pago, que se
  aprueba dentro de los 15 minutos que guarda el horario).
- `total` = `caja` + `senas`. `productos`: cuánto de la caja fue de productos.
- `debe`: lo que quedó debiendo de los turnos de ese día (por la fecha del turno).

Errores: 400 `invalido` (fecha).

## Reportes (`GET /v1/panel/reportes` y `GET /v1/panel/estadisticas`)

Sin cambiar lo que ya estaba (`ingresos` sigue siendo lo facturado: el precio de los turnos
atendidos), se suman campos:

- `resumen.cobrado`: la plata real que entró en el período = `cobradoSenas` + `cobradoCaja`.
- `resumen.cobradoSenas`, `resumen.cobradoCaja`.
- `resumen.cobradoPorMedio`: la caja por medio (`efectivo`, `transferencia`, `debito`, `credito`,
  `mercadopago`).
- `resumen.productos`: cuánto de la caja fue de productos.
- `resumen.debe`: lo que quedó debiendo de los turnos del período.
- `porDia[].cobrado` (y por lo tanto `ultimos7Dias[].cobrado` del tablero): señas y caja que entraron
  ese día.
