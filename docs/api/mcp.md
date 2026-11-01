# Servidor MCP del negocio (asistente de IA)

Cada negocio puede conectar su asistente de IA (Claude, ChatGPT, Gemini u otro cliente MCP) a Tengo
Libre para consultar y operar la agenda en lenguaje natural. El modelo lo pone el cliente MCP del
negocio: a nosotros no nos cuesta IA. Código: `apps/api/internal/dominio/mcp`, migración `00016_mcp.sql`.
Es lo que usa la sección **Integraciones e IA** del panel («Tu asistente de IA»). El OAuth (para
claude.ai y ChatGPT) está en `oauth.go`, migración `00018_mcp_oauth.sql`, y la pantalla de permiso en
`apps/web/src/routes/conectar`.

## Cómo se conecta

- URL del servidor: `https://api.tengolibre.com.ar/v1/mcp` (la devuelve también `GET /v1/panel/mcp`).
- Autenticación, de dos maneras:
  - **OAuth 2.1** (claude.ai, ChatGPT y cualquier cliente que no deje cargar encabezados): se carga
    sólo la URL y el asistente pide permiso. Ver [Conectar con OAuth](#conectar-con-oauth-claudeai-chatgpt).
  - **Clave del panel** como encabezado `Authorization: Bearer tlmcp_…` (Claude Code, Claude Desktop).
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

**claude.ai** (web, escritorio y teléfono): Configuración → Conectores → «Agregar conector
personalizado», nombre «Tengo Libre» y URL `https://api.tengolibre.com.ar/v1/mcp`, sin nada más.
Al conectar, claude.ai abre Tengo Libre para dar permiso. **ChatGPT**: lo mismo en Conectores (modo
desarrollador) con esa URL y autenticación OAuth.

## Conectar con OAuth (claude.ai, ChatGPT)

Según la spec de autorización de MCP (2025-06-18). El authorization server es la misma API
(`https://api.tengolibre.com.ar`, el origen de `TENGOLIBRE_MCP_URL`); la pantalla de permiso vive
en el front (`TENGOLIBRE_SITIO` + `/conectar`).

1. El cliente llama a `/v1/mcp` sin token → `401` con
   `WWW-Authenticate: Bearer realm="TengoLibre", resource_metadata="https://api.tengolibre.com.ar/.well-known/oauth-protected-resource/v1/mcp"`
   (con un token vencido o inventado, además `error="invalid_token"`).
2. **Protected Resource Metadata** (RFC 9728), en `/.well-known/oauth-protected-resource/v1/mcp` y en
   `/.well-known/oauth-protected-resource`: `resource` = la URL del MCP, `authorization_servers` =
   `["https://api.tengolibre.com.ar"]`.
3. **Authorization Server Metadata** (RFC 8414) en `/.well-known/oauth-authorization-server`:
   `authorization_endpoint` `/v1/oauth/authorize`, `token_endpoint` `/v1/oauth/token`,
   `registration_endpoint` `/v1/oauth/register`, `revocation_endpoint` `/v1/oauth/revoke`, PKCE `S256`,
   grants `authorization_code` y `refresh_token`, `token_endpoint_auth_methods_supported: ["none"]`
   (clientes públicos) y `authorization_response_iss_parameter_supported`.
4. **Registro dinámico** (RFC 7591) `POST /v1/oauth/register` (JSON): `redirect_uris` (1 a 10, `https`
   o `http` a `localhost`/`127.0.0.1`/`[::1]`; sin fragmento), `client_name` (hasta 60; es el nombre que
   ve la persona y el que queda en el registro, p. ej. «Claude»). Responde `201` con `client_id`
   (`tlcli_…`) y `token_endpoint_auth_method: "none"`. Tope global: 300 registros por hora (`429`). Un
   cliente que nunca canjeó un código se borra a la semana.
5. **Autorización** `GET /v1/oauth/authorize` (el navegador): `response_type=code`, `client_id`,
   `redirect_uri` (exacto, de los registrados), `code_challenge` + `code_challenge_method=S256`
   (obligatorio), `state`, `resource` (RFC 8707: tiene que ser la URL del MCP; si falta, se asume).
   - Cliente desconocido o `redirect_uri` no registrado: no se vuelve al cliente; va a
     `/conectar?error=cliente_desconocido|redirect_invalido`.
   - Otro error (sin PKCE, `plain`, `response_type` ajeno, `resource` ajeno): vuelve al `redirect_uri`
     con `error` (`invalid_request`, `unsupported_response_type`, `invalid_target`), `state` e `iss`.
   - Bien: el pedido se guarda 30 minutos y el navegador va a `<sitio>/conectar?solicitud=<id>`.
6. **/conectar** (front): sin sesión pasa por `/ingresar` y vuelve. Muestra la app («Claude»), a dónde
   vuelve (`claude.ai`), los negocios de los que la persona es dueña o encargada y los interruptores de
   permisos (encendidos los de siempre). «Permitir» / «Cancelar» redirigen al `redirect_uri` con
   `code`, `state` e `iss` (o `error=access_denied`). La página no se deja enmarcar (`frame-ancestors 'none'`).
7. **Token** `POST /v1/oauth/token` (`application/x-www-form-urlencoded`):
   - `grant_type=authorization_code`, `code`, `client_id`, `code_verifier`, `redirect_uri` (si va,
     igual al del pedido), `resource` (si va, la URL del MCP). El código vive 2 minutos y se usa una
     vez: **reusarlo revoca la conexión** que salió de él.
   - `grant_type=refresh_token`, `refresh_token`, `client_id`. **Rota**: el refresco viejo y el acceso
     anterior dejan de servir.
   - Respuesta: `access_token` (`tlmcp_…`, 1 hora), `token_type: "Bearer"`, `expires_in: 3600`,
     `refresh_token` (`tlmcpr_…`, 60 días desde el último uso), `scope` (los permisos, separados por
     espacio). Errores RFC 6749: `invalid_request`, `invalid_client` (401, sin `client_id`),
     `invalid_grant` (código vencido/usado/de otro cliente, PKCE que no cierra, refresco usado),
     `invalid_target`, `unsupported_grant_type`.
8. **Revocación** (RFC 7009) `POST /v1/oauth/revoke` con `token` (acceso o refresco) y `client_id`:
   siempre `200`; se cae la conexión entera. Sólo conexiones OAuth (una clave del panel se revoca
   desde el panel).

Lo de OAuth (metadatos, registro, token, revocación) responde CORS `*`: no usa cookies.

Con el token de acceso, `/v1/mcp` funciona igual que con una clave: mismo negocio, mismos permisos
(los elegidos en /conectar, editables desde el panel), mismo registro, campanita («Lo pidió
«Claude»») y auditoría (`mcp.conexion_creada` con quien dio el permiso, `mcp.conexion_editada`,
`mcp.conexion_revocada`). Hasta 10 conexiones vigentes por negocio, aparte de las 10 claves.

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

- Sin clave, clave inventada, revocada o token vencido: `401` con `WWW-Authenticate: Bearer … resource_metadata="…"`.
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

Además trae `conexiones`: los asistentes conectados por OAuth (vigentes), como
`{ "id", "nombre": "Claude", "permisos": [...], "cliente": "claude-ai 0.1.0", "creadoEn", "usadoEn", "autorizadoPor": "user_…" }`.

`tokens`: las claves vigentes (las revocadas no; las conexiones OAuth tampoco, van en `conexiones`). `pista`: los últimos 4 caracteres de la clave.
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

### `PATCH /v1/panel/mcp/conexiones/{id}`

Cuerpo: `{"permisos": [...]}` (la lista entera de los encendidos). Valen al instante, también con el
token de acceso que ya tiene el asistente. Respuesta `200`: la `Conexion`. `404` si no existe o ya no está.

### `DELETE /v1/panel/mcp/conexiones/{id}`

Desconecta el asistente: el token de acceso y el de refresco dejan de andar al instante. `204`. Su
registro queda. (Para volver a conectarlo, la persona repite el permiso desde el asistente.)

### `GET /v1/panel/mcp/registro`

«Lo último que hizo la IA». Query: `token` (id), `soloCambios=1` (sólo lo que cambió algo), `antes`
(cursor RFC 3339: el `creadoEn` del último de la página anterior), `limite` (por defecto 50, hasta 200).

```json
{ "acciones": [{
  "id": "…", "tokenId": "…", "token": "Claude de Carla", "cliente": "claude-ai 0.1.0", "tipo": "clave",
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
- `mcp_registro` (RLS por negocio): una fila por `tools/call`. `tipo` en la respuesta: `clave` u
  `oauth` (en ese caso `token` es el nombre de la app, p. ej. «Claude»).
- Conexiones OAuth (`00018`): son filas de `mcp_tokens` con `oauth_cliente`; `hash` es el del token de
  acceso vigente (`acceso_vence`), y `refresco_hash`/`refresco_vence` el de refresco, que rota. El
  refresco se resuelve con `app.negocio_del_refresco(hash_hex)` (política `por_refresco`).
  `app.negocio_del_mcp` ya no resuelve un acceso vencido.
- `auth.oauth_clientes` (registro dinámico) y `auth.oauth_autorizaciones` (pedido → decisión → código,
  sólo hashes), globales como las sesiones.

# La IA de tus clientes (servidor MCP público)

El otro lado: que el asistente de IA **de cualquier persona** encuentre a los negocios que lo
prendieron y saque turno, sin clave. Es la tarjeta «La IA de tus clientes» de Integraciones e IA.
Código: `apps/api/internal/dominio/mcp/publico.go`, migración `00020_ia_publica_y_gestion.sql`.

## URL para cargar en los asistentes

`https://api.tengolibre.com.ar/v1/mcp/publico` — sin clave ni OAuth (la devuelve también
`GET /v1/panel/ia/publico` en `url`).

- **claude.ai**: Configuración → Conectores → «Agregar conector personalizado», nombre «Tengo Libre»
  y esa URL. **ChatGPT**: Conectores (modo desarrollador), esa URL y «Sin autenticación».
- **Claude Code**: `claude mcp add --transport http tengolibre-turnos https://api.tengolibre.com.ar/v1/mcp/publico`
- **Claude Desktop** y otros: `{"mcpServers": {"tengolibre": {"type": "http", "url": "https://api.tengolibre.com.ar/v1/mcp/publico"}}}`

Mismo transporte que el del negocio (POST JSON-RPC, respuesta JSON, sin sesión; `GET`/`DELETE` → 405).
Acepta cualquier `Origin`: no hay credenciales que cuidar.

## Herramientas

| Herramienta | Qué hace |
|---|---|
| `buscar_negocios` | `{texto?, rubro?, ciudad?, limite?}` (hasta 20). Sólo negocios activos, reservables y con la IA pública prendida. Devuelve `slug`, nombre, rubro, ciudad, dirección, servicios, `precioDesde` y `pagina`. |
| `ver_servicios_y_profesionales` | `{negocio}` (el slug). Servicios (duración, precio), profesionales (servicios y horario, sin contacto), dirección, política de cancelación y cuántos días se puede reservar. |
| `buscar_horarios_libres` | `{negocio, servicioId, fecha, dias?, profesionalId?}`, como el del negocio (mismas reglas que la página: anticipación, días de vista). |
| `reservar_turno` | `{negocio, servicioId, fecha, hora, profesionalId?, nombre, telefono?, email?, nota?}` (teléfono o email). |

`servicioId` y `profesionalId` aceptan el id o el nombre exacto. Una herramienta que falla responde
`isError: true` con `{"error": "frase"}` (la misma de la página: `Ese horario ya no está libre`,
`Por ausencias anteriores no podés reservar online…`).

`reservar_turno` reserva **como la página pública**: origen `ia` (en `turnos.origen` y en el panel),
estado `pendiente`, lo que se carga no es «de confianza» (`app.origen = 'web'`), una clienta
penalizada no reserva, y si el negocio cobra seña el turno queda `pendiente_pago` con el link de
Mercado Pago que abre la persona. Respuesta:

```json
{
  "negocio": "El Corte",
  "turno": { "codigo": "…", "dia": "lunes 2 de noviembre", "inicio": "2026-11-02T10:00:00-03:00",
             "servicio": "Corte", "profesional": "Ana", "precio": 8000, "estado": "pendiente_pago" },
  "link": "https://tengolibre.com.ar/turno/…",
  "pago": { "monto": 4000, "modo": "porcentaje", "url": "https://www.mercadopago.com.ar/…", "pagarAntesDe": "…" },
  "mensaje": "El turno queda guardado hasta que pague la seña con el link de Mercado Pago. …"
}
```

Sin seña no viene `pago`. Si Mercado Pago no respondió, `pago.url` es el link del turno (desde ahí se
pide de nuevo). Cupones y membresías no se usan por acá. Al negocio le llegan los avisos de siempre
de una reserva nueva.

## Topes por IP

La IP es `CF-Connecting-IP` (la API sólo escucha detrás de Caddy, que sólo acepta a Cloudflare).
- 60 pedidos por minuto: pasado eso, `429 demasiados_pedidos` (con `Retry-After: 60`).
- 5 reservas por hora (contando las que no pasaron; los datos incompletos no cuentan): la sexta
  responde `isError` con «Ya se hicieron muchas reservas desde esta conexión…».

En memoria (un solo proceso de API).

## Rutas del panel

Sólo **dueño y encargado**; el `PUT` respeta el sólo lectura.

### `GET /v1/panel/ia/publico`

```json
{
  "activo": true,
  "reservasMes": 3,
  "ultimas": [{ "turnoId": "…", "resumen": "reservó Corte para María el lunes 2 de noviembre a las 10:00", "creadoEn": "…" }],
  "url": "https://api.tengolibre.com.ar/v1/mcp/publico"
}
```

`reservasMes`: turnos con origen `ia` creados este mes (en la zona del negocio), sin los que vencieron
sin pagar la seña. `ultimas`: las 5 últimas reservas por la IA («Lo último que hizo la IA»; el front
las mezcla con el registro del asistente del negocio como «El asistente de un cliente …»).

### `PUT /v1/panel/ia/publico`

`{"activo": true}` → `204`. Prenderla o apagarla va a la auditoría (`ia_publica.prendida` /
`ia_publica.apagada`). Apagada, el negocio deja de aparecer y sus herramientas responden que no lo
encuentran. Por defecto está apagada.

## Base

- `negocios.ia_publica` (por defecto `false`).
- `turnos.origen` acepta `ia`; índice parcial `turnos_ia` para contar.
- `app.directorio_ia(rubro, ciudad, texto, limite)`: SECURITY DEFINER como `app.directorio`, sólo los
  que tienen `ia_publica`.
