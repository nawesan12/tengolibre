# Estado

## 26/11/2026: sugerencias prendidas; el interruptor, sólo con plan pago

Pedido de Nahuel (26/11): «tienen que estar prendidas por defecto y que sólo se puedan sacar si tenés un plan pago».
- **Planes de Tengo Libre:** no hay plan gratis. Al crear el negocio corre una prueba de 14 días (`negocios.prueba_hasta`,
  rige Equipo); después, sin suscripción, el panel queda en sólo lectura (`Bloqueado`). Planes pagos: Inicial, Equipo y
  Centro (Mercado Pago, App Store o Google Play; `suscripcion.Acceso.ConSuscripcion`, que incluye la gracia de un cobro
  fallido). «Plan pago» = `ConSuscripcion`. La prueba no cuenta.
- **Interruptor «Sugerencias» (Configuración):** `GET /v1/panel/ajustes/sugerencias` responde 404 sin plan pago (web, iOS y
  Android ya esconden el renglón con 404) y `PUT` responde 403 `sugerencias_plan_pago`. En `Elegir`, `Apagadas` cuenta sólo
  con `Pago`: si dejó de pagar con el interruptor apagado, las vuelve a ver (el valor queda guardado y vuelve a regir si
  paga de nuevo).
- **Para todos los planes:** «Ya la uso / No me interesa» (para siempre) y «Ocultar por 30 días» en la tarjeta. Es el
  retiro gratuito del art. 27 de la Ley 25.326; no se saca. La política lo dice así.
- **14 días de gracia:** queda. En TL coincide con la prueba, así que en la práctica la ve quien ya paga (o una prueba
  extendida): primero que aprenda a usar la app y que no se sienta como un aviso en el arranque.
- **Tests:** `TestInterruptorSoloConPlanPago` (gratis no apaga, pago sí, «No me interesa» anda sin plan) y filas nuevas en
  `TestReglas`. Sin migración.
- **Encendido:** `SUGERENCIAS_ACTIVAS=true` en `/opt/tengolibre/api.conf` desde el 26/11/2026.
- **Desplegado el 26/11/2026 (e18b3da):** API (con `SUGERENCIAS_ACTIVAS=true` verificado en el proceso, `up -d --force-recreate api`) y web. Sin migración. iOS: sin cambios desde 2611260119, que sigue en todos los grupos (`testflight-ultima`). Android: corrida 45759102960 en verde, 143 en la prueba cerrada de Play.
- **AAIP:** pendiente de Nahuel: sumar la finalidad «recomendación de otros servicios de Grupo Vimo» a la base (texto exacto en `~/Trabajo/vimo/legales/aaip/pendientes.md`).

## 25/11/2026: sugerencias de la casa, fase 1 (rama `sugerencias`, migración 00039; sin desplegar)

Plan: `~/Trabajo/vimo/planes/promos-cruzadas-produccion-corregido.md`. Lámina: `docs/laminas/sugerencia.html` (+ `.png` y `capturas/`).
- **Qué hace:** Tengo Libre → Tu Local Web («Vendé tus productos online», sello «De Grupo Vimo»), al final de Hoy o del Inicio, sólo para la dueña o la encargada con rubro cargado. No se muestra con menos de 14 días, con el alta sin terminar, sin turnos, durante el recorrido, con la prueba vencida, con el interruptor apagado o cerrada. Destino: `tulocalweb.com.ar/empezar?utm_source=tengolibre&utm_medium=sugerencia&utm_content=tl-tlw`.
- **API:** `internal/dominio/sugerencias`. Expone `GET /v1/panel/sugerencia`, `POST …/{id}/cerrar` (30d|siempre) y su «Deshacer» `DELETE …/{id}/cerrar`, `POST …/{id}/vista`, `POST …/{id}/clic` y `GET/PUT /v1/panel/ajustes/sugerencias`. Hay un contador `sugerencias_metricas` agregado: el GET no cuenta; la vista se manda cuando la tarjeta quedó dibujada con su foto, y cada negocio suma a lo sumo una vista y un clic por sugerencia y por día (control en memoria, sin guardar el negocio). El alta guarda `negocios.origen` (utm).
- **Encender:** `SUGERENCIAS_ACTIVAS=true` en `/opt/tengolibre/api.conf` y `up -d api`. Por defecto está apagado y GET responde siempre `null`.
- **Imágenes:** `apps/web/src/lib/sugerencias/`, servidas por `/sugerencias/<archivo>` con caché de 30 días. No van en las apps.
- **Modo oscuro:** la web (panel, Configuración, Privacidad, /empezar) es sólo clara (`color-scheme: light` en `app.css`); la tarjeta en oscuro de la lámina la cumplen sólo iOS y Android. La paleta oscura de la web (los valores de `PaletaOscura`: #0B181B, #14272C, #F4EEE2, #A7B6B5, #2A434A, #5FDCAA) va en una tanda aparte, para todo el panel y no sólo la tarjeta.
- **Lámina aprobada por Nahuel el 26/11/2026** («sí, están buenas»). Integrada a main y desplegada apagada ese día.
- **Desplegado el 26/11/2026 (cd09bf3):** API, worker y web, migración 00039. Copia previa: `/var/backups/tengolibre/antes-sugerencias-202611260416.sql.gz`. `SUGERENCIAS_ACTIVAS` no está en `api.conf`. iOS build 2611260119 en todos los grupos; Android corrida 45754394397 en verde (el `.aab` no se subió a Play).
- **Rubros (26/11/2026):** la regla TL → TLW usa la lista explícita `RubrosTLW` (`reglas.go`): todos los de `miweb.Rubros` menos `salud` (a un consultorio no se le ofrece «Vendé tus productos online»). Entrenamiento (gimnasios) y «otro» siguen. Un rubro nuevo en miweb no entra solo: se suma a mano.
- **Política:** sale publicada con el deploy del 26/11 («Actualizada el 26 de noviembre de 2026»), porque el origen utm (`tl-origen`, `negocios.origen`) ya se guarda aunque las sugerencias estén apagadas.
- **Pendiente antes de encender (`SUGERENCIAS_ACTIVAS=true`):** declarar ante la AAIP, en el expediente de la base de Tengo Libre, la finalidad de publicidad propia (recomendar otras apps de Grupo Vimo según rubro y uso, con derecho de retiro, art. 27 de la Ley 25.326) y corregir `legales/aaip/pendientes.md`, que hoy la encaja en la finalidad 1; lo mismo para Tu Local Web → Caigo. Es declaración jurada: la confirma Nahuel. Además, antes de encender:
  - Poner la fecha real del encendido en la política (`apps/web/src/routes/privacidad/+page.svelte`, «actualizado») y desplegarla en el mismo deploy que prende la variable.
  - Del lado de TLW → Caigo: una landing de Caigo para dueños sin cuenta (hoy `lugares.caigo.com.ar` cae en ingresar).

## 19/11/2026: suscripciones nuevas cargadas en App Store Connect y Google Play

Scripts (idempotentes): `~/Trabajo/vimo/credenciales-apple/tengolibre-scripts/asc_suscripciones.py tengolibre` y
`~/Trabajo/vimo/tiendas/play/suscripciones.py tengolibre`. Sin prueba gratis en la tienda.

| productId | Play (sólo AR) | App Store (base USD, 175 territorios) |
|---|---|---|
| `ar.com.tengolibre.inicial.mensual` | $ 6.900, activo (antes 7.000) | US$ 4,49 (antes 4,99), READY_TO_SUBMIT, nivel 3 |
| `ar.com.tengolibre.inicial.anual` | $ 69.000, activo | US$ 43,99, READY_TO_SUBMIT, nivel 3 |
| `ar.com.tengolibre.equipo.mensual` | $ 14.900, activo | US$ 9,49, READY_TO_SUBMIT, nivel 2 |
| `ar.com.tengolibre.equipo.anual` | $ 149.000, activo | US$ 94,99, READY_TO_SUBMIT, nivel 2 |
| `ar.com.tengolibre.centro.mensual` | $ 29.900, activo | US$ 18,99, READY_TO_SUBMIT, nivel 1 |
| `ar.com.tengolibre.centro.anual` | $ 299.000, activo | US$ 189,99, READY_TO_SUBMIT, nivel 1 |
| `ar.com.tengolibre.completo.mensual` | plan base desactivado (no se vende) | sin territorios (no se vende) |

- Play: plan base `mensual`/`anual`, regiones 2026/01 (con 2022/02 Argentina cobra en USD y la API rechaza ARS).
- Apple: la captura de revisión vieja fallaba por tamaño (1178 px); se reescaló a 1179×2556. Sigue siendo la pantalla
  «Potenciá tu negocio»: conviene cambiarla por una de Más › Suscripción.
- **Falta (Apple):** las suscripciones nuevas se mandan a revisión junto con la próxima versión de la app (la primera
  vez van con un binario); el acuerdo de apps pagas sigue sin banco ni W-8BEN (lo carga Nahuel). La URL de
  notificaciones de producción está puesta; la de sandbox sigue dando 409 hasta que el acuerdo quede activo.

## 19/11/2026: planes y precios nuevos DESPLEGADOS (API y web 58fd6d7, migración 00038; iOS build 2611191839; Android 139)

- **Planes:** Inicial $ 6.900, Equipo $ 14.900 (destacado), Centro $ 29.900 (+ $ 1.900 por profesional arriba de 25); prueba de 14 días; «completo» pasó a «equipo».
- **Integración:** `planes-precios` entró a main por avance rápido.
- **Copia previa:** `/var/backups/tengolibre/antes-planes-202611192133.sql.gz`.
- **Despliegue:** el de «Desplegar» (abajo), con el worker parado durante `migrar`; la web con `bun run build` + rsync. `goose_db_version` en 38, `/salud/listo` «listo» y la home muestra los precios nuevos.
- **IA:** `TENGOLIBRE_IA_TOPE_DIARIO` no está fijado en `/opt/tengolibre/api.conf`: rige el 100 por defecto del código.
- **iOS:** `testflight-local.sh … tengolibre` (build 2611191839) y `ultima.py`. El espejo grupovimo/tengolibre-ios no se actualizó (iOS se compila desde el monorepo).
- **Android:** espejo grupovimo/tengolibre-android desde la copia limpia (`[skip ci]`), workflow «Android» corrida 39 y `subir_aab.py`: 139 en la prueba cerrada, mandado a revisión.
- **Suscripciones en las tiendas:** cargadas el mismo 19/11 (sección de arriba).

## 19/11/2026: todo en producción (API y web con migraciones hasta la 00037; apps subidas el 18-19/11)

Las entradas del 18/11 de abajo que dicen «sin desplegar» ya están desplegadas:
- comité,
- asistente con Claude,
- link protagonista,
- app sólo para negocios.

Producción tiene la migración 00037, iOS está en TestFlight (todos los grupos) y Android en la prueba cerrada de Play.

- **Lucas (socio):** su reserva no aparecía porque nunca creó su negocio. Se resolvió así:
  - la app es sólo para quien da turnos;
  - los clientes reservan desde la web;
  - «Tus turnos» en /mi-cuenta muestra los turnos reservados con el mail de la sesión;
  - los avisos al negocio también salen cuando reserva alguien con sesión iniciada.
- **Meta:**
  - El píxel de Tengo Libre (1693501452118578) está creado y el dominio verificado.
  - Falta poner el código del píxel en la web (hoy sólo TLW lo tiene, en `/l/` y `meta.ts`) y generar el token CAPI.
- **AAIP:** Meta y Anthropic agregados al expediente EX-2026-110277456 (IF-2026-111936374).

## 18/11/2026: cierre de la revisión del comité (rama comite-revision, sin desplegar; migraciones 00034–00037)

Todos los hallazgos del comité aplicados en API, web, iPhone y Android (detalle en los commits de la rama).
| Migración | Qué |
|---|---|
| 00034 | «Tus turnos» seguro: `turnos.email_reserva` (el mail de ESA reserva web) y `app.turnos_de_persona` sólo por ese mail o el de confianza de la ficha; primero los negocios con algo, sin pendiente_pago vencidos. La web ya no completa el mail de una ficha existente. |
| 00035 | `ajustes.ia_activa` (la IA la prende la dueña: hasta entonces sólo el intérprete), propuestas que vacían los argumentos, `notificaciones.profesional_id` (campanita y push por rol), links anteriores retenidos mientras el negocio esté activo (los últimos 5), acento por paleta (adiós al cian), purga de propuestas (30 d), MCP (180 d) y sesiones vencidas, `suprimir_cliente` limpia lo de la IA, calendarios que se cortan al bajar a alguien (trigger). |
| 00036 | Índice `turnos_abiertos_fin` (CONCURRENTLY, sin transacción). |
| 00037 | `turnos.profesional_elegida` (reprogramar con «Cualquiera» reasigna), la profesional del alta con la dueña, push sólo por los negocios con algo pendiente. |
Antes de desplegar:
- Migrar con el worker parado (`docker compose stop worker`), ahora con `lock_timeout` de 5 s y tres reintentos.
- `GOMEMLIMIT` ya está en `infra/vps/compose.yml` (api 100 MiB, worker 48 MiB): hace falta `docker compose up -d api worker`.
- No poner `ANTHROPIC_API_KEY` hasta modificar la inscripción en la AAIP. Con la clave puesta, cada dueña decide (asistente › «Prender la IA»).
- Decidido y documentado: la API sigue mandando los push `tipo=cliente` (las builds publicadas todavía los usan);
  sacarlos cuando las nuevas reemplacen a las viejas.
- Nuevo: «Pedir seña» de un turno cargado a mano (`POST /v1/panel/turnos/{id}/sena`, web y apps) y «Anotar a
  alguien» en la lista de espera (`POST /v1/panel/espera`, web; en las apps, pendiente).
- Compilar las apps en GitHub (íconos de Android regenerados con `herramientas/simbolos.sh`).

## 18/11/2026: asistente con IA (rama asistente-claude, sin desplegar; migración 00031)

Asistente en Inicio (web) y en Hoy (iPhone y Android). Primero un intérprete propio (agenda de un día, quién viene,
horarios libres, mover / cancelar / confirmar); lo demás va a Claude Haiku 5.5 con las herramientas del MCP y el rol de
quien pregunta. Las acciones vuelven como propuesta con «Confirmar». Contrato y costos: `docs/api/asistente.md`.
Para prender la IA: `ANTHROPIC_API_KEY` en `/opt/tengolibre/api.conf` (sin clave, sólo el intérprete). Topes:
cupo por mes y por día de cada plan (Inicial 100 y 5, Equipo 400 y 20, Centro 1.500 y 75, en `planes.go`),
`TENGOLIBRE_IA_TOPE_DIARIO` (100, techo diario que ningún plan pasa), `TENGOLIBRE_IA_PRESUPUESTO_USD` (20).
Falta: probar con la clave real (dos preguntas seguidas → `cache_leido` > 0 en el log `ia: respuesta`), compilar las
apps en GitHub, modificar el registro en la AAIP (Anthropic como proveedor) y el comando `tengolibrectl ia-uso`.

## 18/11/2026: el link, protagonista (rama link-protagonista, migración 00032; sin desplegar)

El link se puede cambiar sin romper los viejos: `PATCH /v1/panel/negocio/link` (dueño/encargado), el anterior queda en
`slugs_anteriores` y `app.negocio_publico` lo sigue resolviendo (toda la API pública anda con el viejo; la web redirige
con 301 en `/[slug]` y `/[slug]/reservar`). Retenido 90 días para otros negocios; 5 cambios cada 30 días; auditoría
`negocio.link_cambiado`. `GET /v1/slugs/{slug}` reemplaza el chequeo por 404 del alta. Web: TuLink con Copiar,
WhatsApp, Instagram, QR y «Cambiar link»; primero en Inicio con menos de 20 turnos en 30 días; «Tu link» en la barra;
primera sección de Mi web; grande y editable en «El nombre» y primero en «¡Listo!». iPhone y Android: TarjetaLink en
Hoy (misma regla), Mi web y el alta; CambiarLink. Falta compilar las apps.

## 18/11/2026: la app es sólo para negocios (rama app-solo-negocios; sin compilar ni desplegar)

Decisión de Nahuel: el socio eligió «Reservo turnos», nunca creó su negocio y reservó en la página de otro. Fuera, en
iPhone y Android: «¿Cómo vas a usar Tengo Libre?» (`ElegirModo`, fase `eligiendoModo`), el modo clienta
(`Pantallas/Cliente`, `PrincipalCliente`, fase `cliente`, `turnoClienteAbierto`, ruta `turno-cliente`, «Usar como
clienta» en Más, canal «Tus turnos» de Android). Sin negocio se va directo al alta, con «¿Querés reservar un turno?
Hacelo desde la página del negocio.» (abre tengolibre.com.ar/explorar en el navegador). El `modo`/`misTurnos`
guardado se borra al abrir y los turnos que seguía el teléfono se dejan de seguir al registrar el token.
`/turno/*` sale de `enlaces.json` y del intent-filter: el AASA/assetlinks cambia recién al desplegar la web.
Pendiente: compilar en GitHub; desplegar la web (AASA); fichas de tienda en `~/Trabajo/vimo/tiendas`
(«Y también para tus clientas…» en `asc/fichas.py`, `play/fichas.py`, `play/tengolibre.json`); la API sigue con
`/v1/t/{codigo}/dispositivo` y los push `tipo=cliente` (ya no los usa ninguna app).

## 18/11/2026: revisión integral (API y web desplegadas, 8bd33bf; sin migraciones)

Cuatro revisores (API, web, iPhone, Android) y verificación de cada hallazgo. Arreglado: ver el commit 8bd33bf.
Límite de cuerpo: 30 MB en web.conf y en Caddy (copias `web.conf.antes-30m` y `/root/tengolibre.caddy.antes-30m`).

Pendiente (verificado o probable, sin arreglar todavía):
- Web: una pestaña vieja guarda en el negocio elegido en otra pestaña (el negocio sale de la cookie, no del formulario).
- API: devolver una seña con doble clic descuadra `devuelto` (leer el pago con FOR UPDATE dentro de la transacción);
  un cobro aprobado de MP reactiva una suscripción cancelada o reemplazada (`ConciliarCobro`); notificaciones de
  Apple fuera de orden pisan el estado (comparar signedDate); el premio de un referido sobrevive si se cancela el
  turno referido; `/v1/p/{slug}/dias` sin tope; PATCH de servicio sin FOR UPDATE; fotos aceptan http:// y cualquier dominio.
- iPhone: acciones del push con la app cerrada (sin `.foreground`, `estado` nil) y arranque en frío con otro negocio
  (igual en Android, MainActivity.atender); `withTaskGroup` en Cliente.swift (pasar a secuencial); SSE no detecta el 401;
  subidas de fotos sueltas si se cierra la hoja (archivos huérfanos).

## 17/11/2026: fotos de los servicios — API y web desplegadas (14b2ef5, migración 00030)

Hasta 5 fotos por servicio (`servicios.fotos`); la primera es la portada y queda también en `foto` (los clientes que
mandan sólo `foto` cambian la portada sin perder las otras). Panel web: subir varias, elegir portada y sacar. Página
pública: miniatura en la fila del servicio. Reserva: tira con todas las fotos del servicio elegido. iPhone y Android:
grilla en Editar servicio (mantener apretada → portada o sacar). Pruebas: `TestFotosDelServicio` (API) y la e2e
«fotos de un servicio» (escritorio y teléfono). Copia previa: `/var/backups/tengolibre/antes-202611171831.sql.gz`.
Pendiente: TestFlight. El código iOS compila (el archivo de la corrida 44154299565 salió bien), pero el export falló con
«Error Downloading App Information» (Apple) y después GitHub no asignó una Mac. Relanzar `gh workflow run testflight -R
grupovimo/tengolibre-ios` con el token de grupovimo. Android en verde (aaf3f3c en el espejo).

## 16/11/2026: QA de punta a punta — desplegado (API y web 94e6cb7, migración 00029; iOS 86fba08, Android 3b2ab92)

Revisión de API, web e iPhone/Android, con negocios QA en producción (ya borrados, igual que sus usuarios de Clerk).
- Compras de prueba de Apple/Google ya no activan planes: para la revisión de Apple, `TENGOLIBRE_APPLE_SANDBOX=1` en api.conf (y `up -d`) mientras dure; `TENGOLIBRE_GOOGLE_PRUEBAS` igual para Play.
- Reserva pública y link del turno: sólo lo tipeado o enmascarado (antes, con el teléfono de un cliente se veía su email confiable). La lista de espera no pisa el email de un socio.
- Referidos: códigos de 8, premio al confirmar y se devuelve al cancelar (00029); topes por IP en reservas, espera y cupones (en memoria: un solo proceso de API).
- Rol profesional sólo con sus turnos; claves MCP revocadas al bajar de rol; pago tardío revisa disponibilidad.
- Web: Enter en «Tus datos» reserva; sin 500 en empezar ni en horarios; el socio en un dominio propio ingresa en tengolibre.com.ar.
- Apps: «Confirmar que voy», días libres y reporte mensual fallaban siempre; push tras volver a entrar; sin conexión al abrir; compras atadas a su negocio.
Pendiente: `clienteId` en el turno suelto del panel (la API lo ignora: `json:"-"`).

## 12/11/2026: datos personales (AAIP) — desplegado (migraciones hasta la 00028)

Lo de `~/Trabajo/vimo/legales/aaip/pendientes.md` (1.4 y 3.4). Migraciones nuevas, ya aplicadas en producción (visto el 15/11):
| Migración | Qué |
|---|---|
| 00026 | Corregir el teléfono (`GRANT UPDATE (telefono)`) y suprimir un cliente: `clientes.suprimido_en` y `app.suprimir_cliente(id, ahora)` |
| 00027 | Purga de negocios cerrados: `negocios.cerrado_en` (trigger), `auth.cobros_archivados`, `app.negocios_para_purgar` y `app.purgar_negocios` |
| 00028 | Días libres: el motivo «salud» pasa a «personal» (pedidos, bloqueos, campanita y auditoría) |

- Clientes: `PATCH /v1/panel/clientes/{id}` acepta `telefono` (único en el negocio: 409 `cliente_existente`).
  `DELETE /v1/panel/clientes/{id}` (recepción para arriba) responde `{"resultado": "borrado" | "anonimizado"}`: sin turnos
  se borra; con turnos pasados queda «Cliente borrado» sin teléfono, email ni nacimiento, los turnos sin nota y con otro
  código (Mi turno deja de andar), y se van notas, etiquetas, penalidades, membresías, referidos, reseñas, lista de espera
  y los mails/WhatsApp pendientes. Con turnos por venir, 409 `cliente_con_turnos`. Los suprimidos no se listan ni se buscan.
  En la web, el lápiz de la ficha corrige nombre, teléfono y email y abajo está «Borrar sus datos»; en las apps, en Editar.
- Negocios cerrados: a los 90 días del cierre el worker borra el negocio y todo lo suyo (en cascada) y sus imágenes, y
  copia los cobros (señas aprobadas, caja y suscripciones cobradas) a `auth.cobros_archivados` sin datos de nadie. Un
  negocio con una suscripción autorizada o pausada queda retenido. **Apagada**: se prende con `TENGOLIBRE_PURGA_NEGOCIOS=1`
  en `/opt/tengolibre/api.conf` (y `docker compose up -d worker`, que además monta `./archivos`). Antes, mirar qué borraría:
  `tengolibrectl purga-negocios --listar` (como el migrador). Los cerrados antes de 00027 cuentan desde la migración.
- Reserva: aviso del art. 6 antes de pedir los datos (`src/lib/publico/AvisoDatos.svelte`, también en la lista de espera).
- Política y términos: responsable, casilla privacidad@, dónde están los datos y transferencias, datos y proveedores que
  faltaban, plazos de borrado, negocio responsable y Tengo Libre encargado.
- iOS: `project.yml` sólo pide guardar en Fotos (el QR); el `Info.plist` de Tu Local Web se reemplazó por el de Tengo Libre.
  Android: el manifest saca `AD_ID` si una librería lo suma.

## 31/10/2026 (noche): diseño v2 en toda la web y todas las funciones del diseño

En producción desde el 1/11 a la 1:20 (versión 81af42c, migraciones hasta 00020, con copia previa
`/var/backups/tengolibre/antes-202611010121.sql.gz`). Suite e2e completa: 70/70.

Toda la web se rehízo con el handoff v2 (`docs/handoff/design_handoff_tengolibre_v2/`, fiel al
píxel): tokens, Lato y Material Symbols en `apps/web/src/app.css` y `app.html`, la marca en
`static/marca/` y las firmas en `src/lib/ui/` (Talon, Hueco, Icono, Logo). Por área:
- Landing, alta y onboarding de 8 pasos (`src/lib/landing`, `src/lib/alta`, `/empezar`, cartel A5 con QR).
- Web pública con 6 plantillas × estilo × colores (`src/lib/publico/PaginaNegocio.svelte`), reserva en
  una pantalla, Mi turno como talón (repetir, confirmar, .ics), reseña, `/espera/{token}`.
- Panel: armazón (sidebar con «El día», buscador en vivo, campanita, pestañas en el teléfono), Inicio,
  Turnos (WhatsApp, oferta del hueco, caja, para reprogramar), Calendario, Lista de espera, Clientes
  (recordatorios), y toda la gestión: Mi web, Plantillas (vista previa en vivo por iframe `?vista=1`),
  Integraciones e IA, servicios y productos, equipo y días libres, pagos, membresías, cupones,
  reseñas, referidos, reportes (horas muertas, reporte mensual imprimible), configuración, suscripción.
- Cuenta: ingreso (equipo y socio), `/conectar` (OAuth del MCP), invitación, Mi cuenta, preferencias
  de mails, Explorar, admin.

Funciones nuevas de la API (Nahuel decidió construir todas las del diseño; contratos en `docs/api/`):
| Migración | Qué |
|---|---|
| 00012 | Estilo publicado de la web (plantilla, portada, fondo, títulos, forma; luego secciones) |
| 00013 | Ofrecer el hueco solo a la lista de espera (con vencimiento) y pedir días libres |
| 00014 | Confirmación y recordatorio por WhatsApp (Cloud API de Meta) y push web (VAPID) con preferencias |
| 00015 | Turno fijo (series), volver a reservar desde Mi turno, caja y productos |
| 00016 | Servidor MCP del negocio (claves, permisos, registro de acciones) |
| 00017 | Horas muertas con cupón por franja, cumpleaños y «hace mucho que no venís», reporte mensual |
| 00018 | OAuth 2.1 del MCP (registro dinámico, PKCE) para claude.ai y ChatGPT |
| 00019 | Datos de cuenta, explorar (puntaje, próximo libre) y admin; preferencias de mails separadas |
| 00020 | IA de los clientes (MCP público `/v1/mcp/publico`), reportes y suscripción que faltaban |

Lo que necesita configuración externa para prenderse (sin eso queda apagado, sin romper nada):
WhatsApp (`TENGOLIBRE_WHATSAPP_*`, cuenta de Meta y dos plantillas: ver `docs/api/whatsapp.md`),
push web (`TENGOLIBRE_VAPID_*`, se generan con `go run ./cmd/vapid`).

Mercado Pago, configurado el 1/11: app «TengoLibre» 6161498246914243 en la cuenta de Nahuel
(Checkout Pro, OAuth con PKCE). Vuelta del OAuth en
`https://tengolibre.com.ar/panel/mercadopago/vuelta` y webhook de producción en
`https://api.tengolibre.com.ar/v1/webhooks/mercadopago` (pagos, órdenes, planes y suscripciones, vinculación).
`TENGOLIBRE_MP_*` está en `/opt/tengolibre/api.conf` y en CREDENCIALES.md.

Señas con la API de Orders (31/10): Mercado Pago discontinúa la API de Preferencias, así que la seña
se cobra con Checkout Pro vía Orders (`internal/integraciones/mercadopago/ordenes.go`):
`POST /v1/orders` con el token del negocio (`type: online`, `processing_mode: manual`, `total_amount`
e importes en texto con dos decimales, `external_reference` = referencia del pago, `expiration_time`
`PT{n}M` hasta `vence_en`, `config.online` con las URLs de vuelta, `max_installments: 1`,
`X-Idempotency-Key`); la respuesta trae `checkout_url`, que va a `pagos.link`, y el id de la orden a
`pagos.mp_orden` (migración 00021; `mp_transaccion` es la PAY… y `mp_pago` el id numérico de
`payments[].reference.id`). El webhook procesa los dos tópicos: `order` (consulta `GET /v1/orders/{id}`
con el token del negocio y aplica la transacción relevante) y `payment` (pagos de preferencias
viejas, mientras existan). La devolución va por `POST /v1/orders/{id}/refund` (total sin cuerpo,
parcial con `transactions[{id, amount}]`) si el pago tiene `mp_orden`, y por `/v1/payments/{id}/refunds`
si no. Una orden cuyo intento fue rechazado no admite otro (`retries.allowed: false` de fábrica): el
siguiente pedido de link crea una orden nueva. Para que Mercado Pago avise las órdenes hay que tener
tildado el evento «Order (Mercado Pago)» (tópico `order`) en Webhooks del panel, además de «Pagos
(legacy)»: ya están los dos (1/11). En producción desde el 1/11 (versión 3eb0fe8, migración 00021,
copia previa `antes-202611010154.sql.gz`). Contra la API real se comprobó que una orden con
`expiration_time: PT15M` se crea y devuelve `checkout_url`. Falta probar un pago real de punta a
punta (redirect de vuelta y webhook `order`) cuando un negocio conecte su cuenta.

## 31/10/2026 (tarde): front publicado

https://tengolibre.com.ar sirve el front (SvelteKit, `apps/web`, contenedor `tengolibre-web-1` en
127.0.0.1:3400) con todas las pantallas de Jiku: página pública y reserva en pasos, link del turno,
reseñas, invitaciones, explorar, panel completo (turnos, calendario en vivo, clientes, equipo y
horarios, mi web, pagos, membresías, cupones, referidos, reportes, configuración, suscripción),
sucursales, mi cuenta y admin. Estilo neutro provisorio: la UI final sale del handoff.
Ingreso: Google y Apple por Clerk producción, canjeado por una sesión propia de la API (30 días).
Tiempo real: SSE de la API (`/v1/panel/eventos`, LISTEN/NOTIFY) reenviado por `/panel/eventos`.
Con la prueba vencida, todo el panel muestra arriba que está en sólo lectura (`GET /v1/panel/acceso`,
que puede leer cualquier rol); al dueño, además, los últimos 3 días de prueba.
Membresías en la reserva pública: el socio ingresa (Clerk) desde «Tus datos» y tilda «Usar mi
membresía». La API lo identifica por su email verificado contra el email «de confianza» del cliente
(`clientes.email_confiable`: lo cargó el negocio, nunca lo tipeado en la web; ver
`app.cuidar_email`, migración 00009), así nadie usa una membresía ajena escribiendo un teléfono.
Google Calendar (y Apple/Outlook): link iCal secreto por negocio o por profesional
(`calendario`, `/panel/calendario/suscribirse`, migración 00010). El front sirve `/cal/{token}.ics`
reenviando `GET /v1/cal/…`; un profesional con usuario vinculado maneja sólo el suyo.

Push al teléfono (`push`, migración 00011): la app registra su token en `POST /v1/yo/dispositivos`
(atado a la sesión) y el worker manda cada notificación nueva de la campanita a los miembros activos,
por APNs (iPhone, clave «Vimo APNs») y FCM (Android, cuenta `fcm-tengolibre`). Las claves viven sólo
en el VPS, en `/opt/tengolibre/claves/` (montada en el worker); `vimo-copia` las guarda.
Asistente de IA por MCP (`mcp`, migración 00016): el negocio crea una clave en el panel y conecta su
Claude/ChatGPT a `https://api.tengolibre.com.ar/v1/mcp`, con permisos por clave y registro de acciones.
Detalle para el front en `docs/api/mcp.md`.
Diseño: el handoff v2 está en `docs/handoff/design_handoff_tengolibre_v2/` (README con las reglas).

Desplegar el front: `cd apps/web && bun run build`, `rsync -az --delete build/ root@187.41.102.190:/opt/tengolibre/web/build/`
y `docker compose restart web` en /opt/tengolibre. Config en `/opt/tengolibre/web.conf`.

## 31/10/2026: backend de Jiku completo en Go, en producción

La API (`apps/api`) cubre todo lo de Jiku y está desplegada en https://api.tengolibre.com.ar
(versión en `/v1/version`). Falta el front: se arma con el handoff del rediseño.

### Dominios (`internal/dominio`)
| Paquete | Qué hace |
|---|---|
| `negocios` | Alta (con Clerk), datos, ajustes de reserva, equipo con roles |
| `catalogo` | Categorías, servicios, profesionales, horario semanal, bloqueos, excepciones |
| `disponibilidad` | Motor puro de horarios libres (availability-engine.ts) |
| `turnos` | Página pública, reserva, link del cliente (ver/cancelar/reprogramar), agenda del panel |
| `clientes`, `reportes` | CRM, etiquetas, notas, penalidades por ausencias, búsqueda, estadísticas, CSV |
| `cobros`, `promos`, `membresias`, `suscripcion` | Mercado Pago (señas, webhook, devoluciones), cupones, referidos, membresías, suscripción a TengoLibre y panel en sólo lectura con la prueba vencida |
| `avisos`, `resenas`, `espera`, `invitaciones` | Cola de mails y campanita, reseñas, lista de espera, invitaciones al equipo |
| `miweb`, `dominios`, `directorio`, `grupos`, `arranque`, `admin` | Web del negocio y galería, dominio propio, explorar, sucursales, onboarding, admin de la plataforma |

Plataforma (`internal/plataforma`): `db` (RLS por negocio), `sesion` (Clerk + X-Negocio + roles +
sólo lectura), `web`, `problema`, `archivos`, `correo`, `auditoria`, `trabajos`, `cifrado`, `clerk`.

### Para el front
- Sesión: `Authorization: Bearer <token de Clerk>`. En el panel, además `X-Negocio: <id>` (los negocios
  de la persona salen de `GET /v1/yo`). En desarrollo sin Clerk vale `Bearer dev:user_xxxxxxxxxx`.
- Errores: `application/problem+json` con `code` estable (p. ej. `horario_ocupado`,
  `cliente_penalizado`, `suscripcion_requerida` = 402).
- Rutas: públicas en `/v1/p/{slug}/…`, `/v1/t/{codigo}/…` (link del cliente), `/v1/explorar`,
  `/v1/resenas/{token}`, `/v1/invitaciones/{token}`, `/v1/preferencias/{token}`, `/v1/host/{host}`;
  panel en `/v1/panel/…`; sucursales en `/v1/grupos`; admin en `/v1/admin/…`. El detalle de cada una
  está en el `Rutas` de cada paquete y en sus tests de punta a punta.
- Links que mandan los mails: `/turno/{codigo}`, `/{slug}`, `/resena/{token}`,
  `/invitacion/{token}`, `/mi-cuenta/avisos/{token}`, `/panel?fecha=`.
- Vuelta de Clerk: `https://tengolibre.com.ar/ingresar/clerk` (y `ar.com.tengolibre.app://callback`
  en la app nativa), ya cargadas en Clerk producción. Vuelta de Mercado Pago:
  `/panel/mercadopago/vuelta`, que manda `code` y `state` a `POST /v1/panel/mercadopago/vuelta`.

### En el VPS
- `/opt/tengolibre`: compose `tengolibre` (postgres 256m, api 128m en 127.0.0.1:4200, worker 64m),
  `api.conf` (600), `pg.secret`, `archivos/`, `proximamente/`. Web reservada en 3400.
- Caddy: `/etc/caddy/sitios/tengolibre.caddy`; certificado de origen `/etc/caddy/certs/tengolibre.*`.
- Correo: `sistema@tengolibre.com.ar` (manda como hola@) por mail.tulocalweb.com.ar:465.
- Copia diaria: `vimo-copia` (la lleva la sesión de Tu Local Web) guarda base, conf y archivos.
- Registro común de todas las apps: `/opt/VPS.md`.

### Pendiente
- **Suscripciones de Apple (5/11):** el acuerdo de apps pagas está firmado, pero en «Información del usuario pendiente». Faltan la cuenta bancaria y el W-8BEN, que Apple habilita después de procesar la entidad legal. Cuando quede activo, correr `asc_suscripciones.py` (credenciales-apple/tengolibre-scripts) para la URL de sandbox y los planes, que hoy están en MISSING_METADATA. TestFlight externo: grupo «Socios» con link público en las cuatro apps (`asc_socios.py`).
1. App de iPhone (handoff en ~/Trabajo/vimo/tengolibreiphone.zip; App Store Connect 6829332747).
2. Mercado Pago: tildar «Order (Mercado Pago)» en el webhook, desplegar y probar una seña real (ver arriba). Cuando no queden señas con preferencias, sacar `LeerPago`/`Devolver` y el tópico `payment`.
3. Admin: listo por email (`TENGOLIBRE_ADMINS_EMAILS` con nsantillandev@gmail.com y nahuel@grupovimo.com.ar); `TENGOLIBRE_ADMINS` (IDs) queda vacío.
4. Dominios propios: el DNS `dominios.tengolibre.com.ar` ya existe (sin proxy). Falta correr
   `infra/vps/dominios.sh` después de la primera verificación (coordinar con las otras apps, que
   comparten Caddy) y ponerle un timer.

## Desplegar
```bash
cd apps/api && V=$(git rev-parse --short HEAD)
for c in api worker tengolibrectl; do CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "-X main.version=$V" -o bin/$c ./cmd/$c; done
ssh root@187.41.102.190 'docker exec tengolibre-postgres-1 pg_dumpall -U postgres | gzip > /var/backups/tengolibre/antes-$(date +%Y%m%d%H%M).sql.gz'
for c in api worker tengolibrectl; do scp bin/$c root@187.41.102.190:/opt/tengolibre/bin/$c.nuevo; done
ssh root@187.41.102.190 'cd /opt/tengolibre && for c in api worker tengolibrectl; do mv bin/$c.nuevo bin/$c; done &&
  docker compose stop worker && docker compose run --rm --no-deps -T api /app/tengolibrectl migrar && docker compose up -d api worker'
```
