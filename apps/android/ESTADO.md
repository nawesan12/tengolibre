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

## Hecho (sin compilar todavía: Nahuel pidió una sola compilación al final)
- **F0:** Gradle, CI, tema (`tema/Tema.kt`), íconos, núcleo y `Estado`. Ya compiló y pasó el CI.
- **Piezas propias de Android** (lo que iOS da hecho):
  - `tema/Hoja.kt`: la hoja (.sheet) en su propia ventana, con alturas media y grande y arrastre.
  - `tema/Selectores.kt`: Confirmacion, MenuBoton, CalendarioMes y SelectorHora.
  - `tema/Graficos.kt`: los 6 gráficos con Canvas, Pulso, .entrada(i), NumeroAnimado y pesosCortos.
  - `nucleo/Consejos.kt`: lo de TipKit a mano, con VistaConsejo y ConConsejo.
- **Pantallas portadas:**
  - Ingreso (las 6 de entrada) y Principal (Navegacion, BarraPestanas, BotonTurnoFlotante, Cabecera,
    SelectorNegocio, Cuerpo, Destinos y Provisoria).
  - Hoy (Hoy, Talon, DatosDia, PlataHoy, Guia, Momentos) y Agenda.
  - Turno (DetalleTurno con Cobrar y Reprogramar; NuevoTurno con PreviaFijo y Contactos).
  - Mas a medias:
    - Hechos: piezas Bloque…, DestinoMas y EstadoNegocio con Potencia y BarraAvance.
    - Faltan en Mas.kt: Mas, PantallaMas, HojaEdicion, PrimerosPasos y Cuenta.
- `Estado` tiene `compartirTexto` y `abrirLink`, que conecta MainActivity.

## Falta portar (en este orden), y después la única compilación
1. `espera/ListaEspera.kt`.
2. `clientes/Clientes.kt`, con `DestinoClientes.ficha(id)`, `PlanMembresia`, `Socio`, `EtiquetaConUso` y lo que
   use la ficha (GraficoVisitas, SugerenciaMembresia).
3. `resenas/Resenas.kt`.
4. `avisos/Avisos.kt` + Compartir: QR con zxing, WhatsApp, historias de Instagram
   (`com.instagram.share.ADD_TO_STORY`).
5. `buscar/Buscar.kt` con CampoBuscar.
6. `mas/Mas.kt`, lo que falta: Mas, PantallaMas, HojaEdicion, PrimerosPasos y Cuenta (con «Usar como clienta» y
   salir).
7. `mas/MasEquipo.kt`, `mas/MasPlata.kt` (ConexionMP, ConfigCobro, Cupon, ResumenCaja, NuevaSocia,
   SuscripcionVista con Play Billing) y `mas/MasWeb.kt` (NegocioDatos, FotoGaleria; selector de fotos del
   sistema).
8. `cliente/Cliente.kt` (modo clienta) e `ingreso/Enlaces.kt` (TurnoClienteCargar, InvitacionVista; los abre Raiz).
9. **Registrar todo en `principal/Destinos`** desde un `app/Registro.kt`:
   - Las raíces de cada pestaña y los destinos de DestinoMas y DestinoClientes.
   - Las hojas: DetalleTurno, NuevoTurno, Compartir, Buscar, Avisos y Recorrido.
   - Borrar `pantallas/Muestra.kt` y los provisorios.
10. **Plataforma:**
    - `push/Push.kt`: FCM con acciones por tipo, como el ServicioPush de Caigo.
    - `nucleo/Actividad.kt` (ActividadTurno.sincronizar): notificación en curso.
    - Widgets Glance, `res/xml/shortcuts.xml` y Compras (Play Billing).
    - Fijar en MainActivity `EstadoNegocio.hayAvisos` y `EstadoNegocio.hayWidget`.
11. **PantallasTest:** la misma lista que iOS, más la prueba de paridad .swift / .kt.
12. **API:** datos de FCM, notificación en curso y `/v1/panel/suscripcion/google`. Antes de migrar, `pg_dump`.
13. Recién ahí: `publicar.sh`, compilar una vez, corregir hasta verde y comparar con `herramientas/comparar.sh`.

## Cuentas externas pendientes
- **Firebase `grupo-vimo`:** registrar la app Android `ar.com.tengolibre.app`. Sus valores van a `local.properties`
  (`tl.firebase.*`) y a los secretos del repo.
- **Google:** cliente OAuth Android con el SHA-1 de la llave de subida.
- **Llave de subida:** `~/Trabajo/vimo/credenciales-android/tengolibre-subida.jks`.
- **Clerk:** comprobar que la vuelta `ar.com.tengolibre.app://callback` esté permitida para Android (en iOS ya
  funciona con el mismo esquema).
