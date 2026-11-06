# Cómo se clona una app de iPhone a Android (molde: Tengo Libre)

Regla de Nahuel (5/11/2026): Android **idéntica** a iOS, con las mismas funciones, y todas las apps de Grupo Vimo lo
más parecidas posible entre sí. Esta guía sale de hacer Tengo Libre (iOS 13.500 líneas → Android verde en CI el 6/11)
y es lo que hay que repetir con Tu Local Web, Corredor y la auditoría de Caigo.

## 1. Reglas que no se negocian
- **Fuente de verdad: el código iOS**, no el handoff ni la web. Si iOS hace algo raro, Android hace lo mismo.
- **Un `.kt` por cada `.swift`**, con el mismo nombre y en la carpeta equivalente en minúscula:
  `Pantallas/Hoy/Hoy.swift` → `pantallas/hoy/Hoy.kt`. Mismos nombres de vistas (funciones `@Composable`), de
  funciones, de modelos y **los mismos textos, letra por letra**. Así se audita par por par y los cambios futuros se
  aplican en paralelo.
- **Mismos nombres de tema:** `C.fondo`, `Tipo.lato(tam, Peso, numeros)`, `Radio`, `Movimiento`, `Haptico`,
  `BotonMenta`, `Rotulo`… en `tema/`.
- **Íconos por nombre de SF Symbol** (`Icono("calendar.badge.plus")`), con una fuente recortada de Material Symbols
  Rounded (contorno y relleno; `.fill` elige la rellena). El mapa está en `herramientas/simbolos.txt`. Si se suma uno,
  se corre `herramientas/simbolos.sh`, que necesita fonttools: `python3 -m venv /tmp/ftv && /tmp/ftv/bin/pip install
  fonttools` y después `PATH=/tmp/ftv/bin:$PATH sh herramientas/simbolos.sh`.
- **Se compila sólo en GitHub** (repo público `grupovimo/<app>-android`, gratis). El código vive en el monorepo de la
  app (`apps/android`) y `publicar.sh` lo copia, commitea en los dos lados y empuja. No se compila en la Mac.
- **Una sola compilación al final de cada tanda.** Se escribe todo, se publica, se corrige hasta verde.
- Sin APK (Nahuel prueba por Play, prueba interna). Sin Material3: tema propio. Sin Hilt ni Retrofit.

## 2. Stack (igual en todas)
AGP 8.13, Kotlin 2.2.21, Compose BOM 2025.10.01, minSdk 26, target/compile 36. OkHttp + kotlinx.serialization,
Coil 3, Tink (sesión cifrada), Glance (widgets), Play Billing 8, Firebase Messaging (sin google-services.json: se
inicia a mano con `tl.firebase.*`), zxing (QR), Robolectric 4.16 + Roborazzi para los tests de pantallas.
Copiar `settings.gradle.kts`, `build.gradle.kts`, `gradle/libs.versions.toml`, `app/build.gradle.kts` y el workflow
de Tengo Libre y cambiar el paquete y el prefijo de `local.properties`.

## 3. Equivalencias SwiftUI → Compose (ya resueltas; reusar)
| iOS | Android (Tengo Libre) |
|---|---|
| `@Observable Estado` + `.environment` | `class Estado` con `mutableStateOf` + `LocalEstado` |
| `.task(id: estado.version)` | `LaunchedEffect(estado.version) { cargar() }` |
| `try? await api.get` | `api.opcional<T>(ruta)` (en `nucleo/Api.kt`) |
| `NavigationStack` + `navigationDestination` | `Navegacion` con una pila por pestaña + `app/Registro.kt` que mapea cada destino a su pantalla |
| `TabView` (las pestañas siguen vivas) | se componen todas las visitadas y sólo se ubica la elegida (`mostrarSi` en Principal) |
| `.sheet` / detents | `tema/Hoja.kt` (`Hoja(visible, alCerrar, alturas)`) en su propia ventana |
| `confirmationDialog` | `Confirmacion` (corre la acción **antes** de cerrar) |
| `Menu` / `contextMenu` | `MenuBoton` / `MenuContextual` |
| `Picker` (menú) | `SelectorMenu` |
| `Picker(.segmented)` | `Segmentado(sobrePetroleo = false)` |
| `DatePicker` fecha / hora | `RenglonFecha` (con `CalendarioMes`) / `RenglonHora` (`SelectorHora`) |
| `Toggle` | `Interruptor` (con título) o un interruptor suelto de 51×31 si `.labelsHidden()` |
| `Form`/`Section` | `Bloque(titulo, pie, filas…)` + `Renglon`, `RenglonCampo`, `RenglonPesos`, `Numerito` |
| `.refreshable` | `Refrescable({ cargar() }) { Column(…verticalScroll…) }` (`tema/Refrescar.kt`) |
| TipKit | `nucleo/Consejos.kt` (`VistaConsejo`, `ConConsejo`, uno por hora) |
| Swift Charts | `tema/Graficos.kt` (Canvas) |
| `ColorPicker` | `SelectorDeColor` |
| `PhotosPicker` | `PickVisualMedia` + achicar a 2000 px JPEG |
| Contactos | `rememberContactos` (no trae email) |
| `ShareLink` / `UIPasteboard` | `estado.compartirTexto` / `ClipboardManager` |
| Keychain | `Llavero` (Tink + Android Keystore) |
| APNs + categorías con acciones | FCM **sólo datos** + `Push.mostrar` arma las acciones; `AccionesPush` las corre sin abrir la app |
| Live Activity / Dynamic Island | notificación en curso con cuenta regresiva y «Atendido», pedida como Live Update (Android 16) |
| WidgetKit | Glance (chico y mediano; en Android no hay pantalla bloqueada) |
| App Intents / Siri | `res/xml/shortcuts.xml` (no responden sin abrir la app) |
| StoreKit 2 | Play Billing (`nucleo/Compras.kt`), mismos ids de producto, `obfuscatedAccountId` = id del negocio |
| Universal Links | App Links (`autoVerify`) + `assetlinks.json` en la web (con el SHA-256 de Play) |
| Sign in with Apple nativo | Clerk por Custom Tabs (Google y Apple), vuelta `ar.com.<app>.app://callback` |

## 4. Orden de trabajo
1. Base: Gradle, CI, tema, íconos, núcleo (Api, Formato, Dia, Modelos, EnVivo, Clerk, Estado). Compilar esta base
   sola (es lo único que conviene compilar antes).
2. Pantallas en el orden de iOS: ingreso → principal → la pestaña de inicio → el resto → «Más» → modo cliente →
   enlaces → `Registro.kt`.
3. Plataforma: push, actividad, widgets, atajos, compras, `MainActivity`.
4. Tests: `PantallasTest` con **la misma lista que el `PantallasTests.swift`** (mismos nombres de captura) y
   `ParidadTest` (lee `paridad.txt`, que escribe `publicar.sh`: cada `.swift` de Pantallas tiene su `.kt`).
5. API: lo que pide Android (ver §6). `pg_dump` antes de migrar en producción.
6. Publicar, compilar, corregir hasta verde.
7. **Auditoría de fidelidad**: repartir los pares `.swift`/`.kt` en 4 revisiones paralelas (cada una con archivos
   distintos), comparando texto por texto, acción por acción, rol por rol, endpoint por endpoint. Los cambios a piezas
   compartidas los hace uno solo al final. Compilar otra vez.

## 5. Trampas que ya mordieron (no repetir)
- **Extensiones compartidas en el núcleo.** `Api.opcional` estaba en un paquete de pantallas y rompió 300 líneas en
  todas las demás. Todo lo que usan varias pantallas va en `nucleo/` o `tema/`.
- `LocalFontScale` no existe en Compose: `LocalDensity.current.fontScale`.
- `Confirmacion` cerraba (y limpiaba el elegido) antes de correr la acción: «Sacar de la lista» no hacía nada. Ya
  está corregido en la pieza; en cualquier diálogo, capturar el valor antes de abrirlo.
- `Interruptor` ocupa todo el ancho: dentro de un `Renglon` tapa el título. Para el Toggle sin texto, un interruptor
  suelto.
- Un bloque deshabilitado (`.disabled`) no se logra sólo con alpha: hay que tragarse los toques
  (`pointerInput` que consume todo).
- Clases `@Serializable` locales dentro de funciones: **sí** compilan (se usan mucho).
- **Tests de pantallas:** avanzar el reloj de Compose no espera las respuestas (llegan por el hilo de OkHttp). Hay que
  alternar `Thread.sleep` + `advanceTimeBy` + `waitForIdle` (`asentar` en `PantallasTest`), y prender
  `ModoPrueba.activo` para que Play Billing no se cuelgue.
- `ApiDePrueba` acepta `extra` para pisar respuestas por test (p. ej. el día vacío).
- `respuestas.json` (`apps/pruebas/`) es compartido con iOS: editarlo **sin reformatear** (insertar la línea a mano) y
  copiarlo a `app/src/debug/assets/pruebas/` y `app/src/test/resources/pruebas/`.
- Llamar por teléfono: `ACTION_DIAL` (abre el marcador; llamar directo pide permiso).
- El contador del ícono no se puede fijar a mano en Android (sale de las notificaciones).
- Glance no carga Lato; el widget usa la fuente del sistema.
- Los widgets de Android se actualizan cada 30 min como mínimo, más cuando la app avisa (`Widgets.actualizar`).
- En zsh `${!var}` no existe: para cargar secretos con `gh secret set` pasar cada valor explícito (un script que
  imprime `CLAVE=valor` y un `while read`).
- Las capturas de iOS del CI salen negras: no comparar con capturas de iOS (Nahuel revisa a mano en TestFlight).

## 6. Lo que la API tiene que darle a Android (hecho en Tengo Libre, commit 1fa132d)
- **FCM sólo datos** (`fcm.Envio{SoloDatos: true}`): sin bloque `notification`, con
  `{titulo, cuerpo, ruta, negocio, tipo, categoria}`. Si no, Android muestra el aviso solo y sin acciones.
- `plataforma` (`ios`|`android`) en las tablas de tokens que antes eran sólo de APNs (actividad del turno, teléfonos de
  la clienta), con el token FCM sin pasar a minúsculas.
- Actividad en curso por FCM: `{tipo: actividad, turno, fin (RFC3339), siguiente, terminar: "1"|"0", cliente, servicio}`.
- Google Play: `POST /v1/panel/suscripcion/google` `{producto, token}` (verifica con la Play Developer API) y el
  webhook de RTDN `POST /v1/webhooks/google-play?clave=…`. Config `<APP>_PLAY_CUENTA` y `<APP>_PLAY_CLAVE_WEBHOOK`
  (vacías = apagado). Mercado Pago responde 409 `suscripcion_google` si la vigente es de Play.

## 7. Cuentas y secretos
- Firebase común `grupo-vimo` (Spark): las 4 apps Android ya están registradas; `~/Trabajo/vimo/firebase/google-services.json`.
- Llave de subida por app: `~/Trabajo/vimo/credenciales-android/<app>-subida.{jks,clave}` (keytool RSA 4096).
- Secretos del repo espejo (el workflow los escribe en `local.properties`): `TL_FIREBASE_APP_ID`, `TL_FIREBASE_API_KEY`,
  `TL_FIREBASE_PROYECTO`, `TL_FIREBASE_REMITENTE`, `TL_FIRMA_JKS` (base64), `TL_FIRMA_CLAVE` (cambiar el prefijo por app).
- El paquete firmado para Play sale con «Run workflow» (artifact `play`).
- Cuando exista la cuenta de Play: crear la app, los productos de suscripción con los mismos ids que la App Store,
  el tópico RTDN, la cuenta de servicio de la Play Developer API, el SHA-256 de la firma de Play en `assetlinks.json`
  y el cliente OAuth Android de Google.
