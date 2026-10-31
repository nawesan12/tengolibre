# Recordatorios automáticos (Clientes · propuesta 12)

El saludo de **cumpleaños** y **«hace mucho que no venís»**. Cada uno se prende por negocio, con un
texto corto propio y un cupón opcional; el panel muestra el mail tal cual llega y a quiénes les toca
esta semana. Código: `apps/api/internal/dominio/recordatorios` (rutas, trabajo del worker y armado
del mail, enganchado al `avisos.Enviador` con `Armadores`). Migración `00017` (tabla `recordatorios`,
`clientes.nacimiento`, tipos de aviso `cumpleanios` y `extraniamos`).

Rutas del panel: `Authorization: Bearer …` y `X-Negocio: <id>`. Errores en `application/problem+json`
con `code`.

## Reglas

- **Cumpleaños** (`cumpleanios`): le llega el día de su cumple, desde las 10 (hora del negocio), si
  tiene fecha de nacimiento, email, vino al menos una vez (un turno atendido) y no se dio de baja de
  los mails. Uno por año. El 29/2, el 28 en los años no bisiestos.
- **Hace mucho que no venís** (`extraniamos`): le llega cuando pasan `dias` (por defecto 60, de 14 a
  365) desde su último turno atendido y no tiene otro reservado (pendiente, confirmado o esperando el
  pago). Desde las 10; si el worker estuvo parado, recupera hasta 6 días atrás. Uno por «último
  turno»: si vuelve y después deja de venir, le llega otro.
- El trabajo del worker es `recordatorios_automaticos` (cada 30 minutos). El mail se arma al
  mandarlo: si el negocio apagó el recordatorio, el cliente se dio de baja o (en «hace mucho») ya
  reservó, queda `descartado` con el motivo en el registro de avisos (`GET /v1/panel/avisos`).
- El cupón va sólo si sigue vigente al mandarse (activo, en fecha y con usos); si no, el mail sale
  sin cupón. El botón «Reservar turno» lleva a `/{slug}?cupon=CODIGO`.
- Todo mail lleva el link de baja del cliente (`/mi-cuenta/avisos/{token}`) al pie y en
  `List-Unsubscribe`.

## GET /v1/panel/recordatorios

Roles: `duenio`, `encargado`, `recepcion`.

```json
{
  "recordatorios": [
    {
      "tipo": "cumpleanios",
      "titulo": "Cumpleaños",
      "activo": true,
      "texto": "En Lumi te regalamos un 20% en lo que quieras este mes. Reservá cuando quieras.",
      "cuponId": "…",
      "cupon": { "id": "…", "codigo": "CUMPLE20", "descuento": "20%", "vigente": true },
      "dias": 60,
      "regla": "Le llega el día de su cumple, a las 10, si vino al menos una vez.",
      "vistaPrevia": {
        "para": "sofia@ejemplo.com",
        "asunto": "¡Feliz cumple, Sofía!",
        "html": "<!doctype html>…",
        "texto": "Lumi Estética\n\n¡Feliz cumple, Sofía!\n\n…"
      },
      "semana": [
        { "clienteId": "…", "nombre": "Sofía Méndez", "email": "sofia@ejemplo.com",
          "fecha": "2026-11-03", "detalle": "Cumple el martes 3 de noviembre (36)", "enviado": false }
      ],
      "semanaTexto": "Esta semana: 4 cumpleaños"
    },
    {
      "tipo": "extraniamos",
      "titulo": "«Hace mucho que no venís»",
      "activo": false,
      "texto": "Cuando quieras volver, tenemos lugar. Reservás en un minuto desde acá.",
      "cuponId": null,
      "cupon": null,
      "dias": 60,
      "regla": "Le llega cuando pasan 60 días desde su último turno atendido y no tiene otro reservado.",
      "vistaPrevia": { "para": "…", "asunto": "Te extrañamos, Paula", "html": "…", "texto": "…" },
      "semana": [
        { "clienteId": "…", "nombre": "Paula Gómez", "email": "paula@ejemplo.com",
          "fecha": "2026-11-01", "detalle": "Último turno el miércoles 2 de septiembre: Drenaje", "enviado": true }
      ],
      "semanaTexto": "Esta semana: 9 clientes"
    }
  ]
}
```

- Siempre vienen los dos, en ese orden; sin configurar, `activo: false` y el texto por defecto.
- `semana`: hoy y los 6 días que siguen (`fecha` es el día que le toca). `enviado`: ya está en la
  cola de avisos (salió o sale en un momento). Lista aunque el recordatorio esté apagado: es lo que
  saldría.
- `vistaPrevia`: el mail tal cual llega (mismo armado que el del worker), con el primero de la
  semana o, si no hay nadie, con un ejemplo («Sofía»). `html` es el documento entero, para un
  `<iframe srcdoc>`; el link de baja de la vista previa es de ejemplo.
- `cupon.descuento`: `"20%"` o `"$ 2.000"`.

## PUT /v1/panel/recordatorios/{tipo}

Roles: `duenio`, `encargado`. `{tipo}`: `cumpleanios` o `extraniamos`. Lo que no viene no cambia.

```json
{
  "activo": true,
  "texto": "En Lumi te regalamos un 20% en lo que quieras este mes.",
  "cuponId": "…",
  "dias": 60
}
```

- `texto`: hasta 300 letras; `""` vuelve al texto por defecto.
- `cuponId`: un cupón del negocio; `""` lo saca.
- `dias`: sólo cuenta en `extraniamos` (14–365).
- Queda en la auditoría como `recordatorio.<tipo>`.

Respuesta `200`: el recordatorio como en el GET (con `vistaPrevia` y `semana`).

Errores: `404` tipo desconocido; `400 invalido` (texto largo, cupón inexistente, días fuera de
rango); `402 suscripcion_requerida` con el panel en sólo lectura.

## GET /v1/panel/recordatorios/{tipo}/vista-previa

Roles: `duenio`, `encargado`, `recepcion`. La vista previa con un borrador, sin guardar nada (para
mostrarla mientras se escribe). Query opcional: `texto`, `cuponId` (`""` = sin cupón), `dias`.

```json
{ "para": "sofia@ejemplo.com", "asunto": "¡Feliz cumple, Sofía!", "html": "<!doctype html>…", "texto": "…" }
```

Errores: los mismos `400`/`404` del PUT.

## Fecha de nacimiento del cliente

- `GET /v1/panel/clientes`, `GET /v1/panel/clientes/{id}`: cada cliente suma `"nacimiento": "1990-11-03"`
  (o `null`).
- `POST /v1/panel/clientes` y `PATCH /v1/panel/clientes/{id}` (roles `duenio`, `encargado`,
  `recepcion`) aceptan `"nacimiento": "AAAA-MM-DD"`; en el PATCH, `""` la borra.
- Reserva pública `POST /v1/p/{slug}/turnos`: `cliente.nacimiento` opcional (`"AAAA-MM-DD"`). Se
  guarda sólo si el cliente todavía no tenía (nunca pisa la que cargó el negocio).
- Validación: desde 1900 y no en el futuro; si no, `400 invalido` en `nacimiento` (o
  `cliente.nacimiento`).
