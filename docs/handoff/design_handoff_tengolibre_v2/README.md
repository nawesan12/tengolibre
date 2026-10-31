# Handoff: Tengo Libre · diseño v2

## Qué es
Rediseño completo de Tengo Libre (turnos online para peluquerías, barberías, estética, uñas y consultorios): landing masiva, alta y onboarding, panel del negocio, web pública con reserva y seña, Mi turno, cuentas y admin. Repo de destino: `nawesan12/tengolibre`, app `apps/web` (SvelteKit).

## Sobre los archivos
Los `.dc.html` son **referencias de diseño hechas en HTML**: prototipos que muestran cómo se ve y se comporta cada pantalla. **No son código para copiar.** La tarea es recrearlos en `apps/web` con Svelte, sus rutas y sus componentes actuales. Los estilos están en línea en cada archivo; los valores exactos están en `tokens.css` y en este README.

Para abrirlos: servir la carpeta con cualquier servidor estático (`npx serve .`) y abrir `Índice v2.dc.html`. Necesitan `support.js` al lado y conexión para Lato y Material Symbols.

## Fidelidad
**Alta.** Colores, tipografía, espaciado, textos y movimiento son finales. Recrear al píxel.

## Reglas de diseño (obligatorias)
- **Un foco por pantalla.** La acción principal es una pastilla menta (en lo público, el color del negocio) y va al alcance del pulgar en el teléfono.
- **Lato sola, nunca monoespaciada.** Pesos 400 / 700 / 900. Horas, plata y códigos con `font-variant-numeric: tabular-nums`. Rótulos en MAYÚSCULAS, 900, `letter-spacing: .08em`.
- **Mínimos:** rótulos 11 px, texto corrido 13 px, campos en el teléfono 16 px. Títulos `text-wrap: balance`, párrafos `pretty`, sin palabra sola en el último renglón (`&nbsp;` antes de la última).
- **Espaciado** en múltiplos de 4: 4, 8, 12, 16, 24, 32, 48.
- **Contraste** ≥ 4,5:1 (3:1 sólo títulos grandes). La menta no lleva texto encima salvo petróleo hondo `#102b31`. Para texto menta sobre claro usar `#17805a`. **Ningún estado sólo con color:** siempre ícono + palabra.
- **Áreas táctiles:** 48 px en el teléfono, 36–40 px en escritorio.
- **Movimiento:** entrar 450 ms, tocar 150 ms, curva `cubic-bezier(.2,.7,.2,1)`. Todo gesto tiene un botón equivalente. Respetar `prefers-reduced-motion`.
- **Textos:** rioplatense con voseo, cortos, sin jerga. Plata «$ 8.000». Fechas «lunes 2 de noviembre, 10:30».

## Tokens
Ver `tokens.css` (listo para reemplazar las variables de `apps/web/src/app.css`).

- **Paleta:** petróleo `#183c44`, petróleo hondo `#102b31`, menta `#3ccc90`, menta texto `#17805a`, menta suave `#e3f6ec`, crema `#f4eee2`, crema página `#faf6ee`, papel `#fffdf9`, tinta `#0f2328`, sub `#4a5d61`, sub 2 `#33474b`, líneas `#ece3d1` / `#e2d6bf`, noche `#14221f`.
- **Radios:** pastilla 999 · tarjeta grande 36 · tarjeta 24 · fila/aviso 18 · chico 12.
- **Sombras:** tarjeta `0 14px 28px -22px rgba(15,35,40,.45)`; elevada `0 30px 50px -36px rgba(15,35,40,.8)`; luz arriba `inset 0 2px 0 rgba(255,255,255,.7)` (claro) / `inset 0 1px 0 rgba(255,255,255,.12)` (petróleo).
- **Texturas:** grano claro sobre petróleo (`--tl-grano`), grano oscuro sobre crema y menta (`--tl-grano-osc`), puntos de 1,2 px cada 18–20 px sobre crema. Brillo menta sobre petróleo: `radial-gradient(50% 60% at 50% 0%, rgba(60,204,144,.22), transparent 70%)`.

## Las cuatro firmas de la marca
1. **Talón** = un turno. Tarjeta con un troquel vertical (borde `2px dashed #e2d6bf`) a 88–92 px del borde izquierdo y dos medialunas de 9 px del color del fondo arriba y abajo de ese troquel. A la izquierda la hora (900, tabular); a la derecha cliente/servicio y estado con ícono. Estados: **nuevo** (fondo menta, texto `#102b31`), **en curso** (contorno petróleo 2 px), **atendido** («picado», opacidad .55), **pendiente** (borde punteado), **cancelado** (tachado). Clase `.tl-talon`.
2. **Hueco** = tiempo libre. Rayado a −45°: `rgba(60,204,144,.4) 0 2px, rgba(255,253,249,.9) 2px 10px`, tamaño 28–34 px, que corre despacio (`tl-raya`, 2,4–3,5 s lineal). En la web pública el rayado usa el color del negocio. Siempre con texto («Libre 45 min»). Clase `.tl-hueco`.
3. **El día** (sidebar del panel): una fila por profesional con su franja del día; lo ocupado en petróleo, lo libre rayado; la hora actual en una pestaña menta.
4. **Pestaña** = navegación. El ítem activo del menú se funde con la página como una solapa. La misma forma es la del logo: en el hero de la landing, una pastilla menta encastrada arriba al centro de la caja crema (sombra `0 0 0 14px #183c44` para "recortar" el fondo).

## Pantallas → rutas
| Archivo | Rutas en `apps/web` |
|---|---|
| Landing y alta v3 | `src/lib/landing/*`, `/`, `/empezar` |
| Onboarding v2 | `/empezar/*` (7 pasos) |
| Panel v2 (Inicio, Turnos, Calendario, Lista de espera, Clientes) | `src/routes/panel/+layout.svelte`, `panel/{turnos,calendario,lista-espera,clientes}` |
| Gestion secciones (+ Panel v2 - Gestión, lienzo) | `panel/{mi-web,plantillas,integraciones,servicios,equipo,pagos,membresias,cupones,resenas,referidos,reportes,configuracion,suscripcion,sucursales,notificaciones}` |
| Reserva v2 | `/[slug]`, `/[slug]/reservar`, `/turno/[codigo]` |
| Cuenta y accesos v2 | `/ingresar`, `/invitacion`, `/mi-cuenta`, `/explorar`, `/resena`, `/admin` |
| Pantallas que faltaban v2 | buscador, selector de negocio, etiquetas, penalidades, socias, bloqueos, Mercado Pago, dominio, auditoría, avisos |
| Estados y teléfono v2 | errores, vacíos, sin conexión, gestión en el teléfono |
| Reporte mensual v2 | mail/PDF mensual |
| OG v2 | imágenes Open Graph |
| Sistema de diseño v2 | referencia de tokens y componentes |
| Propuestas / Propuestas integradas | funciones nuevas (no implementar sin decidir) |

## Pantallas clave

### Landing (`Landing y alta v3`)
Página para el público masivo. Ancho máximo 1320, laterales `clamp(20px,4vw,48px)`, secciones separadas `clamp(64px,9vw,120px)`.
- **Nav** (84 px, sobre petróleo con grano): logotipo negativo 28 px · Cómo funciona · Precios · «Bajá la app» (ícono menta) · Ingresar · pastilla menta «Empezar gratis» 48 px.
- **Hero:** caja crema con grano y puntos, radio `clamp(32px,4.4vw,64px)`, con la **pestaña** menta encastrada arriba al centro (`clamp(220px,26vw,320px)` × `clamp(88px,9vw,116px)`): rótulo «7 DÍAS GRATIS · SIN TARJETA» + «Empezar ahora →». H1 «Llená cada hueco.» `clamp(58px,11vw,176px)`, 900, lh .86, ls −0.06em; «hueco» subrayado con rayado animado. Una línea: «Tus clientes reservan y pagan la seña solos. Vos atendés.» Botones App Store / Google Play (56 px, `#0b1214`). Tira de talones en marquesina (48 s, lineal, máscara de fundido a los costados). Debajo, fuera de la caja: «FUNCIONA CON» + logos en cuadros blancos de 38 px.
- **Troquel** entre hero y contenido: franja de 28 px mitad petróleo/mitad papel, medialunas a los costados y línea punteada.
- **Cómo funciona:** tres tarjetas 01/02/03 (crema con puntos / petróleo / menta), número 88 px en contorno (`-webkit-text-stroke: 2px`), cada una con un objeto real: QR + link, talón con seña pagada, hueco → WhatsApp «Ana lo tomó en 6 minutos». CTA «Crear mi página» (petróleo).
- **La app:** bloque `#102b31` con brillo menta: «Tu agenda, en el bolsillo.», tres puntos con ícono, tiendas + QR; teléfono de 300 px inclinado −5° con tres avisos apilados y el ícono de la app flotando (`tl-flota` 6 s).
- **Calculadora** «¿Cuánto te cuestan los huecos?»: dos sliders (turnos perdidos 1–15, precio $ 4.000–40.000) → total mensual = n × precio × 4,3, en menta sobre petróleo, con «Recuperalos».
- **Bento** «Chau libreta, Excel y mil audios.»: plantillas (6 miniaturas), asistente IA (chat), recordatorios (WhatsApp), socias y cupones (talón «3 de 8»), reporte mensual (link). CTA «Probalo 7 días gratis».
- **Rubros:** tarjetas con foto 260 px de alto y degradé petróleo, cada una con «Empezar →».
- **Precios:** dos talones verticales (Inicial $ 7.000 crema / Completo $ 15.000 petróleo «RECOMENDADO»), troquel horizontal, lista con `check_circle`, «Empezar gratis».
- **Preguntas frecuentes:** `<details>` de 68 px, ícono `add_circle`.
- **Cierre:** bloque menta con puntos y pestaña petróleo con el símbolo: «Tu dirección todavía está libre.» + campo `tengolibre.com.ar/____` que valida al instante (libre / tomada con sugerencias).
- **Pie** `#102b31` con grano: «¿Tenés un hueco? Llenalo.» + CTA + tiendas; 4 columnas (Producto, Rubros, Ayuda, Empresa) + Seguinos; fila legal; logotipo gigante al 7 % de fondo.
- **Barra flotante** al pasar 760 px de scroll: pastilla `rgba(16,43,49,.92)` con blur, «7 días gratis, sin tarjeta» + «Empezar».
- Las insignias de las tiendas y el QR son dibujos: reemplazar por los oficiales y un QR real.

### Onboarding (`Onboarding v2`)
Siete pasos: rubro (tarjetas con foto; al elegir, tilde y avanza solo) → nombre + dirección web → servicios sugeridos por rubro → quién atiende y horarios → seña → color y plantilla → final con link y QR. Progreso: anillo alrededor del ícono + «Servicios · paso 3 de 7». Encabezado de paso: número grande en contorno, pastilla petróleo con ícono y nombre, consejo corto. Contenido sobre hoja papel que entra desde la derecha. Escritorio: a la derecha, «Tu página, en vivo» en un teléfono inclinado sobre petróleo, con avisos flotantes al completar pasos. Los pasos opcionales tienen «Lo completo después».

### Panel (`Panel v2`)
Sidebar petróleo con grano (pestaña activa), barra superior con buscador (se achica), «Ver mi página», notificaciones. Roles: dueño, recepción, profesional (ve sólo lo suyo). Menos de 1180 px: el botón de demo queda sólo con ícono. Teléfono: barra inferior de pestañas y acción principal abajo.
- **Inicio:** saludo, talones del día, huecos con «Avisar a la lista».
- **Calendario:** Día (una columna por profesional, fondo rayado = libre, tocar abre carga rápida a esa hora, línea menta = ahora) y Semana (barritas por profesional, horas libres por día). Panel «Verlo en tu calendario» con links iCal por negocio/profesional.
- **Lista de espera:** Esperando / Avisados / Vencidos; cada persona es un talón con días y momento; si hay un hueco que le sirve, «HOY 15:30 · MICA» + Avisar/Quitar.
- **Clientes:** buscador, etiquetas, ficha lateral con números, penalidad, membresía como talón, notas e historial.
- **Plantillas (en vivo):** grilla 2×3 de plantillas (Carta, Vidriera, Sobria, Galería, Editorial, Bloques) + «Ajustá el estilo» (Portada: Con foto/Sólo color · Fondo: Crema/Blanco/Noche · Títulos: Grandes/Medianos/Livianos · Botones: Redondos/Suaves/Rectos). A la derecha, sticky, un teléfono de 376×720 con la web real del negocio. Estado: `publicado` y `borrador`; si difieren, aviso «Cambios sin publicar» con el resumen, «Descartar» y «Publicar»; al publicar, «Publicada. Tus clientes ya la ven.» (4 s).
- **Integraciones e IA:** servidor MCP del negocio con permisos por interruptor, ejemplo de conversación y registro de acciones; Google Calendar, Mercado Pago, WhatsApp, Apple/Outlook, Instagram.
- **Suscripción:** talón de la prueba con los 7 días, comparación Inicial/Completo con ícono + palabra en cada celda, cobro con Mercado Pago, estados activa / cobro fallido / cancelada.

### Web pública y reserva (`Reserva v2`)
Seis plantillas × ajustes de estilo × colores del negocio (primario y acento; el texto sobre el primario se elige blanco o tinta según contraste). En Noche, superficies y textos se invierten y el color de marca se aclara 60 % hacia blanco. Reserva en una pantalla: servicio → profesional (o «Cualquiera») → día → hora → datos → seña con Mercado Pago. Barra fija abajo en el teléfono con «PRÓXIMO LUGAR» y «Reservar». Mi turno como talón con mover/cancelar y reseña.
Props del componente: `plantilla`, `portada`, `fondo`, `titulos`, `forma`, `primario`, `acento`, y para incrustarlo en el panel `ancho` (fuerza el ancho de diseño) e `incrustado` (la barra inferior pasa de `fixed` a `sticky`).

## Estado necesario (resumen)
- Panel: `rol`, `vista`, `turnoSeleccionado`, `filtros`, `modoCalendario (dia|semana)`, `profesionalVisible` (teléfono), `mostrarCancelados`.
- Plantillas: `publicado {plantilla, portada, fondo, titulos, forma}` (del backend) y `borrador` (local); publicar = PATCH del negocio.
- Onboarding: `paso`, respuestas por paso, persistencia para retomar (falta un acceso desde Inicio para retomarlo).
- Landing: `slug` con verificación de disponibilidad (debounce 300 ms), sliders de la calculadora, `scrollY` para la barra flotante.

## Propuestas que necesitan API nueva
Ofrecer el hueco sólo al primero de la lista (con vencimiento), confirmación por WhatsApp, turno fijo, servidor MCP por negocio, links iCal por profesional, borrador/publicado de la web pública.

## Recursos
- `marca/`: logotipos (horizontal, negativo, apilado), símbolo (principal, una tinta blanco/petróleo), íconos de app, favicon.
- `apps/web/static/fotos/`: manicura, peluquera-con-clienta, recepcion, barbero-cortando (del repo).
- `apps/web/static/logos/`: Apple, Google, Mercado Pago, WhatsApp. `marca-icono.png`: ícono de la app.
- Íconos: Material Symbols Rounded (Google Fonts).
- Faltan fotos de Masajes, Consultorio y Otro (hoy rayado con «Foto de…»).

## Capturas
En `capturas/` (ancho de 910 px; el grano no sale en las capturas, sí en los archivos):
- `01–05-landing.jpg`: hero, cómo funciona, la app, precios y pie.
- `01–04-panel.jpg`: Inicio, Calendario, Lista de espera, Clientes. `05-panel.jpg`: Plantillas en vivo.
- `01–03-reserva.jpg`: web pública y reserva.
- `01–02-onboarding.jpg`: paso de rubro y siguiente.
- `01–02-sistema.jpg`: sistema de diseño.
- `cuenta.jpg`: ingresar.

## Archivos
`Índice v2.dc.html` enlaza todo. `support.js` y `doc-page.js` sólo sirven para abrir los prototipos; no van al producto.
