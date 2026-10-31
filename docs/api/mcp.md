# Servidor MCP del negocio (asistente de IA)

Cada negocio puede conectar su asistente de IA (Claude, ChatGPT, Gemini u otro cliente MCP) a Tengo
Libre para consultar y operar la agenda en lenguaje natural. El modelo lo pone el cliente MCP del
negocio: a nosotros no nos cuesta IA. Código: `apps/api/internal/dominio/mcp`, migración `00016_mcp.sql`.
Es lo que usa la sección **Integraciones e IA** del panel («Tu asistente de IA»).

## Cómo se conecta

- URL del servidor: `https://api.tengolibre.com.ar/v1/mcp` (la devuelve también `GET /v1/panel/mcp`).
- Autenticación: la clave del panel como encabezado `Authorization: Bearer tlmcp_…`.
- Transporte: Streamable HTTP (spec MCP 2025-06-18; acepta también 2025-11-25 y 2025-03-26).

**Claude Code**

```bash
claude mcp add --transport http tengolibre https://api.tengolibre.com.ar/v1/mcp \
  --header "Authorization: Bearer tlmcp_XXXXXXXX"
```

**Claude Desktop** (`claude_desktop_config.json`), o cualquier cliente que acepte encabezados:

```json
{
  "mcpServers": {
    "tengolibre": {
      "type": "http",
      "url": "https://api.tengolibre.com.ar/v1/mcp",
      "headers": { "Authorization": "Bearer tlmcp_XXXXXXXX" }
    }
  }
}
```

En el panel conviene mostrar las dos cosas por separado: la URL (copiable) y la clave, que se ve una
sola vez al crearla («La clave va aparte y sólo se muestra una vez»).

## Protocolo

`POST /v1/mcp` con un mensaje JSON-RPC 2.0 (o un lote). Respuestas `application/json`, sin SSE ni
`Mcp-Session-Id` (el servidor no guarda estado entre pedidos).

| Método | Respuesta |
|---|---|
| `initialize` | `protocolVersion` (la del cliente si la conocemos; si no, `2025-06-18`), `capabilities.tools`, `serverInfo` (`tengolibre`, título con el nombre del negocio) e `instructions` en criollo (pedir confirmación antes de cambiar algo, formatos de fecha). Guarda el `clientInfo` en la clave. |
| `notifications/*` | `202` sin cuerpo. |
| `ping` | `{}` |
| `tools/list` | Sólo las herramientas que la clave tiene permitidas. |
| `tools/call` | `{content: [{type: "text", text: <JSON>}], structuredContent: {...}, isError}`. Un error del negocio (horario ocupado, sólo lectura…) vuelve con `isError: true` y `{error: "<frase>"}`. |
| otro | error `-32601`. |

- Sin clave, clave inventada o revocada: `401` con `WWW-Authenticate: Bearer`.
- Herramienta desconocida o sin permiso: error JSON-RPC `-32602` (y queda en el registro).
- `MCP-Protocol-Version` desconocida: `400`. `Origin` presente y ajeno: `403`. `GET`/`DELETE /v1/mcp`: `405`.
- El asistente opera como un encargado: mismas reglas que el panel (disponibilidad, extras, mails al
  cliente). Con la prueba vencida (`suscripcion_requerida`), las herramientas que escriben devuelven
  `isError` («Suscribite para seguir operando…») y las de lectura siguen andando.
- Fechas `AAAA-MM-DD` y horas `HH:MM` en la zona del negocio; las respuestas traen `inicio`/`fin` en
  RFC 3339 con la zona del negocio y `dia` en texto («lunes 2 de noviembre»).

## Permisos y herramientas

Cada clave tiene una lista de permisos (los interruptores de «Qué puede hacer»). Por defecto, al
crearla: `ver_agenda`, `mover_turnos`, `ver_clientes`.

| Permiso (`clave`) | Nombre | Herramientas |
|---|---|---|
| `ver_agenda` | Ver la agenda | `ver_agenda`, `buscar_horarios_libres`, `ver_lista_de_espera`, `ver_servicios_y_profesionales` |
| `mover_turnos` | Crear y mover turnos | `crear_turno`, `confirmar_turno`, `reprogramar_turno` |
| `cancelar_turnos` | Cancelar turnos | `cancelar_turno` |
| `ver_clientes` | Ver clientes | `buscar_clientes`, `ver_cliente` |
| `ver_numeros` | Ver números | `resumen_de_numeros` |

Sin `ver_clientes`, los turnos y la lista de espera muestran el nombre del cliente pero no su teléfono
ni su email.

| Herramienta | Argumentos | Qué hace |
|---|---|---|
| `ver_agenda` | `desde?`, `hasta?` (hasta 31 días; por defecto hoy), `profesionalId?`, `incluirCancelados?` | Turnos del rango. |
| `buscar_horarios_libres` | `servicioId`, `fecha`, `dias?` (1–7), `profesionalId?` | Horarios libres por día, con los profesionales que pueden. |
| `ver_lista_de_espera` | — | Los que esperan un horario (sin avisar y vigentes). |
| `ver_servicios_y_profesionales` | — | Servicios (duración, precio) y profesionales (servicios, horario) con sus ids. |
| `crear_turno` | `servicioId`, `fecha`, `hora`, `profesionalId?`, `clienteId?` o `cliente {nombre, telefono?, email?}`, `nota?` | Turno confirmado, como desde el panel (sin sobreturno). |
| `confirmar_turno` | `turnoId` | Pendiente → confirmado. |
| `reprogramar_turno` | `turnoId`, `fecha`, `hora`, `profesionalId?` | Mueve el turno a un horario libre. |
| `cancelar_turno` | `turnoId` | Cancela (avisa al cliente y a la lista de espera). |
| `buscar_clientes` | `texto` (2+ caracteres), `limite?` (hasta 50) | Por nombre, teléfono o email, con visitas y próximo turno. |
| `ver_cliente` | `clienteId` | Ficha: contacto, visitas, gastado, ausencias, penalidad y últimos 20 turnos. |
| `resumen_de_numeros` | `desde?`, `hasta?` (por defecto el mes en curso) | Turnos, atendidos, ausentes, cancelados, por venir, facturado y clientes nuevos. |

`servicioId` y `profesionalId` aceptan el id o el nombre exacto (sin distinguir mayúsculas).

Lo que cambia algo (crear, confirmar, reprogramar, cancelar) además:
- va a la **campanita**: «El asistente cargó un turno», «… confirmó…», «… movió…», «… canceló…»,
  con el texto del turno y «Lo pidió «<nombre de la clave>»» (tipos `reserva`, `reprogramacion`,
  `cancelacion`);
- va a la **auditoría** con `clerkId` = `mcp:<id de la clave>` y acción `asistente.turno_creado`,
  `asistente.turno_confirmado`, `asistente.turno_reprogramado` o `asistente.turno_cancelado` (más el
  `turno.*` de siempre de los avisos).

## Rutas del panel

Todas con la sesión del panel (`Authorization` + `X-Negocio`), sólo **dueño y encargado**. Las que
escriben respetan el sólo lectura de la prueba vencida (`402 suscripcion_requerida`).

### `GET /v1/panel/mcp`

```json
{
  "url": "https://api.tengolibre.com.ar/v1/mcp",
  "tokens": [{ "id": "…", "nombre": "Claude de Carla", "pista": "x9Qa", "permisos": ["ver_agenda", "mover_turnos", "ver_clientes"],
               "cliente": "claude-ai 0.1.0", "creadoEn": "…", "usadoEn": "…" }],
  "permisos": [{ "clave": "ver_agenda", "nombre": "Ver la agenda", "descripcion": "Turnos, horarios libres, …",
                 "herramientas": ["ver_agenda", "buscar_horarios_libres", "ver_lista_de_espera", "ver_servicios_y_profesionales"] }]
}
```

`tokens`: las claves vigentes (las revocadas no). `pista`: los últimos 4 caracteres de la clave.
`cliente`: el `clientInfo` del último `initialize` (vacío si nunca se conectó; sirve para «Conectado»).
`usadoEn`: el último pedido (se actualiza a lo sumo una vez por minuto).

### `POST /v1/panel/mcp/tokens`

Cuerpo: `{"nombre": "Claude de Carla", "permisos": ["ver_agenda", …]}` (`permisos` opcional: por
defecto los tres de arriba; `[]` = ninguno). `nombre` de 1 a 60. Hasta 10 claves vigentes por negocio
(`409 tope`).

Respuesta `201`: `{"token": Token, "clave": "tlmcp_…", "url": "https://api.tengolibre.com.ar/v1/mcp"}`.
**`clave` sólo viene acá**: se guarda hasheada y no se puede volver a ver (para otra, crear una nueva).

### `PATCH /v1/panel/mcp/tokens/{id}`

Cuerpo: `{"nombre"?: "…", "permisos"?: [...]}` — `permisos` es la lista entera de los encendidos (un
interruptor = mandar la lista con o sin esa clave). Respuesta `200`: el `Token`. `404` si no existe o
está revocada. Un permiso desconocido: `400`.

### `DELETE /v1/panel/mcp/tokens/{id}`

Revoca la clave: deja de andar al instante (`401` en el servidor MCP). `204`. Su registro queda.

### `GET /v1/panel/mcp/registro`

«Lo último que hizo la IA». Query: `token` (id), `soloCambios=1` (sólo lo que cambió algo), `antes`
(cursor RFC 3339: el `creadoEn` del último de la página anterior), `limite` (por defecto 50, hasta 200).

```json
{ "acciones": [{
  "id": "…", "tokenId": "…", "token": "Claude de Carla", "cliente": "claude-ai 0.1.0",
  "herramienta": "crear_turno", "titulo": "Crear un turno",
  "argumentos": { "servicioId": "…", "fecha": "2026-11-02", "hora": "09:00", "cliente": { "nombre": "María Gómez", "telefono": "(dado)" } },
  "ok": true, "escribe": true,
  "resumen": "cargó Corte para María Gómez con Ana, el lunes 2 de noviembre a las 09:00.",
  "error": "", "creadoEn": "…"
}] }
```

Lo más nuevo primero. Los argumentos van resumidos: teléfonos, emails y notas como `"(dado)"`, textos
cortados a 80. `resumen` es una frase lista para mostrar después del nombre de la clave («Claude
cargó…»); en un error, `ok: false` y `error` con lo que se le contestó al asistente. Tiempo real:
cada acción nueva avisa por el SSE del panel con tipo `mcp_registro`.

## Base

- `mcp_tokens` (RLS por negocio): `hash` (sha256 de la clave), `pista`, `permisos text[]`, `cliente`,
  `usado_en`, `revocado_en`. La clave se resuelve antes de fijar el negocio con
  `app.negocio_del_mcp(hash_hex)` (SECURITY DEFINER, política `por_hash` acotada por el setting
  `app.mcp_hash`, como `app.negocio_del_calendario`).
- `mcp_registro` (RLS por negocio): una fila por `tools/call`.
