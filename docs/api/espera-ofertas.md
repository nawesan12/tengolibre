# Lista de espera: ofrecer el hueco solo (propuesta 01)

Cuando se libera un horario (un turno cancelado, o el horario viejo de uno reprogramado), en vez de
avisarle a varios a la vez se le ofrece a **una persona por vez**, en el orden en que se anotaron.
Esa persona recibe un mail «Se liberó lo que esperabas» con un link propio
(`https://tengolibre.com.ar/espera/{token}`) y un tiempo para tomarlo. Si no lo toma, dice que no le
sirve, o se le vence el tiempo, pasa a la siguiente. Código: `apps/api/internal/dominio/espera`
(`ofertas.go`), migración `00013`.

Convenciones de siempre: panel con `Authorization` + `X-Negocio`; errores `application/problem+json`
con `code` estable; horas en RFC 3339; fechas `AAAA-MM-DD` en la zona del negocio; precios en pesos
enteros. Tiempo real: cada cambio de una oferta llega por SSE con `tipo: "ofertas"`, y cada paso de
la cola (pasó a la siguiente, alguien dijo que no) con `tipo: "oferta_intentos"`: los dos invalidan
lo mismo (`panel:ofertas`). La lista de espera sigue con `tipo: "espera"`.

## Cómo funciona

- **Quién entra en la cola**: los de la lista que esperan ese servicio, con ese profesional o
  «cualquiera», que tienen ese día en su rango y les sirve ese momento (`momento`: `cualquiera`,
  `manana` = antes de las 13, `tarde`), que no estén ya avisados. Orden: el que se anotó primero.
  Hasta 20. La cola se arma una vez, al lanzarse la oferta.
- **Tiempo para tomarlo**: `ajustes.minutos` (por defecto 30); nunca vence después del inicio del turno.
- **Se ofrece sola** (`ajustes.automatica`, por defecto `true`): al liberarse el horario arranca sola.
  Con `false` no arranca: la campanita dice «Se liberó las 10:30 con Ana · 2 personas esperan…» y
  recepción la lanza con `POST /v1/panel/espera/ofertas`.
- **Al ofrecérsela** a alguien queda «avisado» (`avisadoEn`): sale de los que esperan.
  - Si dice «No me sirve» → vuelve a esperar otros horarios (`avisadoEn` a `null`).
  - Si no responde a tiempo → queda avisado (`resultado: "vencida"`, «no respondió»).
  - Si la oferta se frena, se cae o se carga a mano mientras la tenía → vuelve a esperar.
  - Si la toma → queda avisado (`resultado: "tomada"`).
- **El worker** (trabajo `ofertas`, cada 30 s) pasa a la siguiente cuando vence un link, y cierra las
  ofertas que llegaron a su hora.
- **Se cae sola** si el horario se ocupa por otro lado: un turno que se superpone (web, panel,
  reprogramación) o un bloqueo por días libres aprobados.
- **Tomar el lugar** reserva con las reglas de la página pública (origen `web`): penalidades y, si el
  negocio cobra seña, el turno queda `pendiente_pago` con el link de Mercado Pago. No aplica la
  anticipación mínima (el hueco se liberó a último momento a propósito). La campanita suma
  «Se llenó el hueco de las 09:00» (tipo `espera`) y el cliente recibe la confirmación de siempre.
- **Nadie lo tomó**: la oferta queda `agotada` y la campanita dice «Nadie de la lista tomó el hueco
  de las 09:00» (sigue libre en la agenda).

### Estados

Oferta (`estado`): `ofreciendo` · `tomada` (alguien de la lista lo reservó, `turnoId`) · `cargada`
(recepción lo resolvió) · `frenada` · `agotada` (se terminó la cola) · `vencida` (llegó la hora) ·
`caida` (se ocupó por otro lado).

Persona en la cola (`cola[].estado`): `en_cola` · `ofrecida` (tiene el link hasta `venceEn`) ·
`tomada` · `paso` («No me sirve») · `vencida` (no respondió) · `cancelada` (se cerró la oferta
mientras la tenía) · `salteada` (cuando le tocó ya no esperaba).

## Objetos

### Oferta (panel)

```json
{
  "id": "uuid",
  "estado": "ofreciendo",
  "origen": "automatica",            // o "panel"
  "servicioId": "uuid", "servicio": "Piernas completas",
  "profesionalId": "uuid", "profesional": "Mica",
  "inicio": "2026-11-02T15:30:00-03:00", "fin": "2026-11-02T16:15:00-03:00",
  "precio": 26000,
  "turnoId": null,                   // el turno que la llenó (tomada / cargada con esperaId)
  "creadoEn": "…", "cerradaEn": null,
  "actual": { …Intento } | null,     // «Ofreciendo a Martina G.» + cuenta regresiva con venceEn
  "sigue":  { …Intento } | null,     // «SIGUE: LOLA F.»; null = «ES LA ÚLTIMA»
  "cola": [ …Intento ]               // todos, en orden
}
```

### Intento

```json
{ "id": "uuid", "esperaId": "uuid" | null, "nombre": "Martina Gómez", "orden": 1, "estado": "ofrecida",
  "ofrecidaEn": "…" | null, "venceEn": "…" | null, "respondidaEn": "…" | null }
```

### Anotado (lista de espera, `GET /v1/panel/espera`) — campos nuevos

- `momento`: `cualquiera` | `manana` | `tarde` («LE SIRVE · TARDE»).
- `resultado`: lo que pasó la última vez que se le ofreció un hueco: `ofrecida`, `tomada`, `paso`,
  `vencida` («AVISADA · NO RESPONDIÓ»), `cancelada`; `null` si nunca. Sirve para las pestañas
  Esperando (`avisadoEn == null`) / Avisados / Vencidos (`?todos=1` trae también avisados y vencidos).

## Página pública

### `POST /v1/p/{slug}/espera` (sin sesión) — cambia

Suma `momento` (opcional, por defecto `cualquiera`). 400 `invalido` si no es uno de los tres.
Respuesta 201: `{ "servicio", "desde", "hasta", "momento" }`.

## Link de la persona (sin sesión; el token es el secreto)

Pantalla «Se liberó lo que esperabas» en el front: `/espera/{token}`.

### `GET /v1/espera/{token}`

200:
```json
{
  "estado": "ofrecida",          // ofrecida | tomada | paso | vencida | cerrada
  "nombre": "Martina",           // primer nombre
  "negocio": { …Negocio público (nombre, slug, dirección, logo…) },
  "servicio": "Piernas completas", "profesional": "Mica",
  "inicio": "…", "fin": "…", "precio": 26000,
  "venceEn": "…",                // para la cuenta regresiva («Es tuyo si lo tomás en los próximos 23 min»)
  "turno": "codigo" | null       // con estado tomada: el link del turno (/turno/{codigo})
}
```
`cerrada`: se ocupó, la frenaron o la resolvió recepción. 404 `no_encontrado`: token inexistente.

### `POST /v1/espera/{token}/tomar` — «Tomar el lugar»

Sin cuerpo: usa nombre, teléfono y email con que se anotó. 201: el `Turno` como lo devuelve
`POST /v1/p/{slug}/turnos` (con `pago: {monto, modo, url}` si hay seña; si `pago.url` viene vacío, se
pide de nuevo con `POST /v1/t/{codigo}/pagar`). Redirigir a `pago.url` o a `/turno/{codigo}`.

Errores (409): `oferta_vencida` (pasó su tiempo), `oferta_cerrada` (ya no está: la tomó, la pasó, se
cayó, la frenaron), `horario_ocupado`, `cliente_penalizado`. 404 token inexistente.

### `POST /v1/espera/{token}/pasar` — «No me sirve, que pase»

Sin cuerpo. 200: la `OfertaPublica` con `estado: "paso"`. El lugar pasa enseguida a la siguiente.
409 `oferta_vencida` / `oferta_cerrada` como arriba.

## Panel

Roles: dueño/a, encargado/a y recepción (los mismos de la lista de espera). Profesional: 403.

### `GET /v1/panel/espera/ofertas`

Sin parámetros: las que se están ofreciendo (para pintar «Ofreciendo a…» en Turnos).
`?fecha=AAAA-MM-DD`: todas las de ese día, en cualquier estado. 200 `{ "ofertas": [Oferta] }`.
400 si la fecha está mal.

### `GET /v1/panel/espera/ofertas/{id}`

200 `Oferta`. 404 si no existe en el negocio.

### `GET /v1/panel/espera/candidatos?servicio=&profesional=&inicio=`

A quiénes se les ofrecería ese hueco, en orden (el «Ofrecer a la lista · N» y la cola antes de
lanzarla). `inicio` en RFC 3339 (url-encodear el `+`). 200 `{ "espera": [Anotado] }`. 400 si falta algo.

### `POST /v1/panel/espera/ofertas` — «Ofrecer a la lista»

```json
{ "servicioId": "uuid", "profesionalId": "uuid", "inicio": "2026-11-02T15:30:00-03:00" }
```
201 `Oferta` (`origen: "panel"`, ya ofrecida a la primera). Errores: 400 `invalido` (faltan datos,
horario pasado); 404 servicio o profesional; 409 `horario_ocupado`, `ya_ofreciendo` (ese horario ya
tiene una oferta activa), `sin_espera` (nadie de la lista busca ese horario).

### `POST /v1/panel/espera/ofertas/{id}/frenar` — «Frenar»

Sin cuerpo. 200 `Oferta` (`frenada`). Quien tenía el link vuelve a esperar.
409 `oferta_cerrada` si ya no se estaba ofreciendo; 404.

### `POST /v1/panel/espera/ofertas/{id}/cargar` — «Cargarlo yo»

Cuerpo opcional:
- Sin cuerpo (o `{}`): deja de ofrecerlo (`cargada`) y devuelve el hueco; el front abre «Nuevo turno»
  con `servicioId`, `profesionalId` e `inicio` de la oferta y reserva con `POST /v1/panel/turnos`.
- `{ "esperaId": "uuid" }`: se lo da a esa persona de la lista: reserva el turno a su nombre como
  turno del panel (`confirmado`, sin seña) y la marca avisada.

200 `{ "oferta": Oferta, "turno": Turno | null }`. 409 `oferta_cerrada`, `horario_ocupado`; 404 oferta
o `esperaId`.

### `GET /v1/panel/ajustes/espera` · `PUT /v1/panel/ajustes/espera`

Configuración › Lista de espera: «tiempo para tomarlo» y «se ofrece sola».
```json
{ "minutos": 30, "automatica": true }
```
GET: dueño/a, encargado/a, recepción. PUT: dueño/a y encargado/a; responde lo guardado. 400 `invalido`
si `minutos` no está entre 5 y 720.

## Mail

Tipo de aviso `espera` (con `datos.intento`). Asunto «Se liberó lo que esperabas — {negocio}»: servicio,
profesional, fecha, hora, precio, dónde, «Es tuyo si lo tomás antes de las HH:MM», botón «Tomar el
lugar» al link. Si cuando sale el mail la oferta ya no está vigente, se descarta.
