# Tengo Libre para Android: estado y cómo retomar

Objetivo (Nahuel, 5/11/2026): Android **IDÉNTICA** a la app de iPhone (`../ios`), y todas las apps de Grupo Vimo
lo más consistentes posible entre sí. Nada se publica en las tiendas hasta que esté. Plan completo:
`~/.claude/plans/hola-claude-tenemos-que-wiggly-toucan.md`.

## Reglas
- **Fuente de verdad: el código iOS**, no el handoff. Cada `.swift` tiene su `.kt` con el mismo nombre en la carpeta
  equivalente (`TengoLibre/Pantallas/Hoy/Hoy.swift` → `app/src/main/java/ar/com/tengolibre/app/pantallas/hoy/Hoy.kt`),
  con los mismos nombres de vistas, funciones, textos y comportamiento.
- **Mismos nombres de tema:** `C.fondo`, `Tipo.titulo`, `BotonMenta`, `Hueco`, `Rayas`, `Rotulo`, `Chip`… en
  `tema/Tema.kt`.
- **Íconos:** se usa el nombre del SF Symbol, igual que en iOS (`Icono("calendar.badge.plus")`). El equivalente de
  Material Symbols Rounded está en `herramientas/simbolos.txt`. Si se suma un ícono, se agrega ahí y se corre
  `herramientas/simbolos.sh`, que regenera las fuentes recortadas (unos 25 KB cada una) y `tema/Simbolos.kt`.
- Sin Material3: tema propio, como Caigo Android.
- **Estado observable:** `app/Estado.kt` usa `mutableStateOf`, igual que `@Observable` en iOS.
  - `estado.version` sube con cada cambio. Las pantallas recargan con `LaunchedEffect(estado.version)`, que es lo que
    en iOS es `.task(id: estado.version)`.
- **Compilación:** solo en GitHub. El código vive acá y `~/Trabajo/vimo/tengolibre-android/publicar.sh "<mensaje>"`
  lo copia al repo público `grupovimo/tengolibre-android`, commitea en los dos y empuja.
- **Respuestas simuladas compartidas:** `../pruebas/respuestas.json`.
  - Es la fuente única para iOS y Android, con marcadores `$iso(N)` y `$dia(N)`.
  - Hay copias en `app/src/debug/assets/pruebas/` y `app/src/test/resources/pruebas/`. Si se cambia el original, hay
    que copiarlo de nuevo a las dos.
  - Falta que iOS (`ModoPrueba.swift`, `PantallasTests`) lo lea también, en vez de sus respuestas en código.
- Sin APK: Nahuel prefirió esperar la prueba interna de Play. Se revisa con capturas.

## Hecho
- **Todo portado** (23 de 23 pantallas de iOS: lo comprueba `ParidadTest` con `paridad.txt`, que escribe
  `publicar.sh`). `app/Registro.kt` conecta Principal con las pantallas; ya no hay provisorios.
- **Piezas propias de Android** (lo que iOS da hecho): `tema/Hoja.kt` (la .sheet), `tema/Selectores.kt`
  (Confirmacion, MenuBoton, SelectorMenu = Picker, CalendarioMes, SelectorHora, RenglonFecha/RenglonHora = DatePicker),
  `tema/Graficos.kt` (Canvas), `nucleo/Consejos.kt` (TipKit), `SelectorDeColor` en MasWeb (ColorPicker).
- **Plataforma:**
  - `push/Push.kt`: FCM sin google-services.json; los avisos llegan sólo con datos y la app arma la notificación con
    las acciones de iOS (TL_RESERVA Confirmar/Ver turno, TL_CANCELO Avisar/Cargar, TL_HUECO Avisar, TL_RESENA
    Responder con texto). `AccionesPush` corre las acciones sin abrir la app.
  - `nucleo/Actividad.kt`: la Live Activity es una notificación en curso con cuenta regresiva y «Atendido» (Live
    Update en Android 16); la mueve el servidor por FCM (`tipo: actividad`).
  - `widgets/Widgets.kt`: «Tu día» con Glance, chico y mediano (en Android no hay widgets de pantalla bloqueada).
  - `res/xml/shortcuts.xml`: los atajos de Siri como atajos del ícono.
  - `nucleo/Compras.kt`: Play Billing con los mismos productos que la App Store; `obfuscatedAccountId` = negocio.
- **API** (commit 1fa132d): FCM sólo datos con `tipo`/`categoria`, notificación en curso y avisos de la clienta por
  FCM (`plataforma: android`), `POST /v1/panel/suscripcion/google` y el webhook de RTDN. Migración `00023_android.sql`
  **sin aplicar en producción** (pg_dump antes).
- **Tests:** `PantallasTest` con la misma lista que iOS (27 secciones, tres roles, claro y oscuro) y `ParidadTest`.
- **Llave de subida:** `~/Trabajo/vimo/credenciales-android/tengolibre-subida.{jks,clave}` (SHA-1
  25:07:D0:0A:56:1E:08:0B:03:9E:34:1B:D7:91:C1:EA:53:96:24:0C).

- **Auditoría de fidelidad (6/11):** cuatro revisiones par por par (`.swift` contra `.kt`). Corregidos: cupones (el
  código no se veía), «Sacar de la lista» (no hacía nada; `Confirmacion` ahora corre la acción antes de cerrar),
  bloque de cobro deshabilitado sin Mercado Pago, Plantillas › Reintentar, Guardar el QR (sólo el QR), Mover › Picker
  de profesional, errores del ingreso con Clerk debajo de los botones, Custom Tab cerrada sin trabar el botón, en vivo
  que se corta al parar, consejos uno por hora, botones deshabilitados enteros al 45 %, barra de estado petróleo,
  botón Turno con pantallas apiladas, widget con hora en dos cifras, atajo «Mi día», animaciones de Hoy y acciones
  de accesibilidad. Sumado: tirar para refrescar en las 9 pantallas que lo tienen en iOS y pestañas que siguen vivas
  (TabView). CI verde (run 42154463468).
- **Producción (6/11):** API 3c953a1 desplegada con la migración 00023 (copia previa
  `/var/backups/tengolibre/antes-202611060254.sql.gz`). Secretos de Firebase y firma cargados en el repo espejo.

## Diferencias de plataforma que quedan (aceptadas)
Llamar abre el marcador; el contador del ícono no se fija a mano; el widget se actualiza cada 30 min y sin Lato; no hay
widgets de pantalla bloqueada; los atajos abren la app; el consejo de la cápsula es tarjeta y no globo con flecha;
Contactos no trae el email.

## Sin probar en un teléfono (6/11)
La app nunca corrió en un aparato real (sin APK; sólo tests y capturas del CI). Con la prueba interna de Play hay que
confirmar: el ingreso con Clerk volviendo a la app, que lleguen los avisos con sus botones (y el turno en curso) y la
compra con Google Play.

## Falta (cuando esté la cuenta de Play)
1. Crear la app en Play, subir el paquete («Run workflow» → artifact `play`) a prueba interna.
2. Productos de suscripción con los mismos ids que la App Store; tópico RTDN → `https://api.tengolibre.com.ar/v1/webhooks/google-play?clave=…`;
   cuenta de servicio de la Play Developer API → `TENGOLIBRE_PLAY_CUENTA` y `TENGOLIBRE_PLAY_CLAVE_WEBHOOK` en
   `/opt/tengolibre/api.conf` y reiniciar api y worker.
3. SHA-256 de la firma de Play en `https://tengolibre.com.ar/.well-known/assetlinks.json` (App Links) y cliente OAuth
   Android de Google con ese SHA-1.
4. Clerk: comprobar que la vuelta `ar.com.tengolibre.app://callback` ande en Android (es la misma que iOS).
5. iOS: que lea `../pruebas/respuestas.json` en vez de sus respuestas en código.

Para clonar las otras apps: `GUIA-CLON-IOS.md`.
