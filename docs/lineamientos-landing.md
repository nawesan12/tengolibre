# Lineamientos de la landing (los de Corredor, pasados a pedido de Nahuel el 31/10/2026)

Referencia en código: repo crm-inmobiliaria, rama `corredor` (worktree ~/Trabajo/vimo/.wt/corredor):
`web/src/lib/Portada.svelte`, `web/src/lib/portada/{Panel,Telefono}.svelte`, `web/src/lib/aparecer.ts`,
`web/static/logos/FUENTES.md`. En vivo: https://corredor.com.ar. Copiar y adaptar con la marca de TengoLibre.

## Lo que pidió Nahuel, en orden de importancia
1. Poco texto: por bloque, un título corto y una línea; a lo sumo tres puntos de 2 o 3 palabras. Sin subtítulo en el hero.
2. Sólo los colores del logo (fondo y acento del ícono) para todo: hero, botones, íconos y cierre. Nada genérico.
3. Pocos bordes y pocas cajas: sin marcos en fotos ni tarjetas con borde para listar funciones. Jerarquía por tamaño, peso, color y espacio.
4. Logos reales de las integraciones (WhatsApp, Google y Apple de Wikimedia; Mercado Pago del sitio oficial) y badges oficiales de App Store y Google Play sin modificar, con «Próximamente» como texto aparte.
5. Fotos reales de Unsplash, de gente (para TengoLibre: barberos, peluqueras, estética). Bajadas de images.unsplash.com/photo-<id>, comprimidas con sips a ~1600 px y 150 a 200 KB. Si cambia una foto, cambia el nombre del archivo (Cloudflare cachea).
6. Título del hero en dos renglones justos, con la parte de acento en `white-space: nowrap`. Nunca una palabra sola en el último renglón (`text-wrap: balance` en títulos, `pretty` en párrafos).
7. Celular: los botones del hero centrados (los 2 llamados a lo ancho y los badges abajo). Una barra fija con el llamado principal abajo, que aparece recién cuando los botones del hero salen de pantalla. Probar a 375, 390 y 430 px.
8. Una sección con números grandes lleva fondo propio (crema cálido de punta a punta, números en el color oscuro de la marca).

## Estructura que funcionó
- Hero con fondo de la marca de punta a punta: pastilla con un punto que late, título enorme (700, letter-spacing -0.05em), llamado principal en el acento con flecha, secundario «fantasma» (blanco al 8% con borde interno sutil) y los badges. A la derecha, foto con esquinas de 32 px, sombra grande y 1 o 2 tarjetitas de interfaz flotando («Nuevo turno», «Confirmado»).
- Cinta de integraciones que se desliza sola: fichas blancas con logo real y nombre, con máscara de degradé a los costados.
- «El sistema»: mockup del producto en HTML/CSS (no captura): ventana de navegador, menú lateral de la marca, números con mini gráficos y tarjetas con foto; un elemento animado en loop y un teléfono con la página pública superpuesto; trama de puntos de fondo que se esfuma.
- Tres bloques alternados foto/texto: «01 —», título, una línea, tres checks y una pieza de interfaz sobre la foto que cuente lo que hace. Detrás de cada foto, una forma de color suave desplazada, sin borde.
- Funciones: grilla de íconos grandes en cuadraditos de color suave, sin texto descriptivo; al pasar el mouse se invierten al color de la marca.
- Pasos: mini escenas de la app en vez de números en círculos, flechas entre pasos y el número grande abajo.
- Cierre de punta a punta en el color oscuro, a dos columnas: título con la segunda mitad en el acento, llamado, secundario («Hablá con nosotros») y badges; del otro lado el teléfono inclinado -4° con dos notificaciones flotando y un halo del acento.
- Pie con columnas (Producto, Grupo Vimo, Contacto) y una línea final con © y «Hecho en Argentina».

## Textura y detalles
- Grano: ruido SVG (feTurbulence) en un ::after con opacity .16 y mix-blend-mode overlay sobre los fondos de color.
- La forma del logo gigante y casi transparente (opacidad .04) de fondo en el hero y el cierre.
- Subrayado del acento como a mano alzada (SVG en background) debajo de lo importante del título.
- Botones: sombra del color del propio botón, translateY(-2px) al pasar y una flecha que se corre 3 px.
- Ritmo vertical: una sola variable para el espacio entre secciones, clamp(104px, 12vw, 168px).
- Fuente Geist e íconos Material Symbols Rounded rellenos.

## Movimiento
- Al hacer scroll cada bloque aparece subiendo 32 px con fade, una sola vez (IntersectionObserver, ver aparecer.ts).
- Tarjetas flotantes y teléfono flotan suave (5 a 7 s); la cinta corre en 32 s.
- Todo se apaga con prefers-reduced-motion.

## Cómo verificar
- Capturas de página completa a 1440 y 375 con un script de Playwright que primero recorre la página (si no, lo que entra con el scroll queda invisible).
- Revisar superposiciones y desbordes horizontales en el celular.
