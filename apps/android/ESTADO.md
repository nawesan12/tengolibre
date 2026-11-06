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
- **F0 (base):**
  - Gradle con las mismas versiones que Caigo, más Glance, Robolectric y Roborazzi.
  - CI `android.yml`: compila, corre los tests y sube el artifact «capturas».
  - Tema completo clonado de `Tema.swift` (colores claro y oscuro, Lato, espacios, radios, movimiento, hápticos) y
    todas sus piezas, con los íconos.
  - Núcleo: `Formato`, `Fecha`, `Dia`, `Modelos`, `Api`, `Guardado`, `Llavero`, `EnVivo`, `Web`, `Clerk`,
    `Preferencias`, `ModoPrueba`.
  - `Estado` con `Ruta`, fases, roles, ingreso por Clerk (Google y Apple, los dos por Custom Tab), toasts con
    «Deshacer» y push (las funciones de registro; falta el servicio FCM).
  - `pantallas/Muestra.kt` es provisoria: todas las piezas juntas. Se compara su captura con iOS y se borra cuando
    estén las pantallas reales.

## Sigue (en orden)
1. **Raíz y navegación:**
   - `app/TengoLibreApp.kt` (Raiz con las fases), `MainActivity` con la vuelta de Clerk
     (`ar.com.tengolibre.app://callback`) y los links.
   - `pantallas/principal/Principal.kt`: barra propia, una pila por pestaña, hojas y botón flotante.
2. **F1:** `ingreso/Ingreso.kt`, `hoy/{Hoy,Talon,DatosDia,PlataHoy,Guia,Momentos}.kt`, `agenda/Agenda.kt`,
   `turno/{DetalleTurno,NuevoTurno}.kt`.
3. **F2:** espera, clientes, reseñas, avisos y compartir, buscar.
4. **F3:** Más y sus 27 secciones (`MasEquipo`, `MasPlata`, `MasWeb`, `EstadoNegocio` completo) y los gráficos de
   `Graficos.swift` con Canvas.
5. **F4:** modo clienta (`cliente/Cliente.kt`).
6. **F5:** push FCM con acciones, widgets Glance, notificación en curso, `shortcuts.xml`, App Links, consejos y
   Play Billing.
7. **F6, API:** datos FCM por tipo, actualización de la notificación en curso, `/v1/panel/suscripcion/google`.
   Antes de cualquier migración, `pg_dump`.
8. **Verificación de cada fase:**
   - `PantallasTest`: la misma lista que `TengoLibreTests/PantallasTests.swift`, en claro y oscuro, a 393×852 dp.
   - Prueba de paridad: lista de `.swift` contra lista de `.kt`.
   - Hoja comparativa iOS | Android (`herramientas/comparar.sh`, falta escribirla), que se publica como Artifact.

## Cuentas externas pendientes
- **Firebase `grupo-vimo`:** registrar la app Android `ar.com.tengolibre.app`. Sus valores van a `local.properties`
  (`tl.firebase.*`) y a los secretos del repo.
- **Google:** cliente OAuth Android con el SHA-1 de la llave de subida.
- **Llave de subida:** `~/Trabajo/vimo/credenciales-android/tengolibre-subida.jks`.
- **Clerk:** comprobar que la vuelta `ar.com.tengolibre.app://callback` esté permitida para Android (en iOS ya
  funciona con el mismo esquema).
