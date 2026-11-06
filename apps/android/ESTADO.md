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

## Falta
1. Compilar (una vez), corregir hasta verde y comparar con `herramientas/comparar.sh` (hace falta una corrida nueva
   de iOS con capturas).
2. Desplegar la API con la migración 00023 (pg_dump antes) y cargar en el VPS `TENGOLIBRE_PLAY_CUENTA` y
   `TENGOLIBRE_PLAY_CLAVE_WEBHOOK` (Google Play) cuando exista la app en Play.
3. Secretos del repo `grupovimo/tengolibre-android` (el CI ya los lee): `TL_FIREBASE_APP_ID`, `TL_FIREBASE_API_KEY`,
   `TL_FIREBASE_PROYECTO`, `TL_FIREBASE_REMITENTE` (de `~/Trabajo/vimo/firebase/google-services.json`),
   `TL_FIRMA_JKS` (base64 de la llave) y `TL_FIRMA_CLAVE`. El paquete para Play sale con «Run workflow».
4. Cuando exista la app en Play: el SHA-256 de la firma de Play en `assetlinks.json` de tengolibre.com.ar (App
   Links), el cliente OAuth Android de Google con ese SHA-1 y los productos de suscripción con los mismos ids.
5. Clerk: comprobar que la vuelta `ar.com.tengolibre.app://callback` esté permitida para Android.
6. iOS: que lea `../pruebas/respuestas.json` en vez de sus respuestas en código.
