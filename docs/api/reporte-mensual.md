# Reporte mensual

El resumen de un mes comparado con el anterior, con la estructura de «Reporte mensual v2.dc.html»:
portada (titular, bajada y cuatro números), agenda (mapa de calor, horas libres, huecos llenados,
sugerencia), plata y servicios, equipo y clientes (con reseñas y «Para noviembre»). El front lo
muestra como página imprimible/PDF; el primer día de cada mes sale un mail que lo resume y lleva ahí.

Código: `apps/api/internal/dominio/reportes/mensual.go` (datos y ajustes) y `reportes/correo.go`
(trabajo del worker y mail). Migración `00017` (`ajustes.reporte_mensual`,
`miembros.reporte_mensual`, tipo de aviso `mensual`).

Rutas del panel: `Authorization: Bearer …` y `X-Negocio: <id>`. Errores en `application/problem+json`.

## GET /v1/panel/reportes/mensual?mes=AAAA-MM

Roles: `duenio`, `encargado`. Sin `mes`: el mes pasado (en la zona del negocio). Se puede pedir el mes
en curso (sale con `completo: false`). Montos en pesos enteros; porcentajes con un decimal.

```json
{
  "negocio": "Lumi Estética",
  "slug": "lumi",
  "mes": "2026-10",
  "anterior": "2026-09",
  "nombre": "octubre 2026",
  "nombreAnterior": "septiembre",
  "nombreSiguiente": "noviembre",
  "desde": "2026-10-01",
  "hasta": "2026-10-31",
  "periodo": "Del 1 al 31 de octubre de 2026 · comparado con septiembre",
  "completo": true,
  "titulo": "Tu mejor mes del año.",
  "bajada": "Facturaste un 12% más que en septiembre y casi nadie faltó. Quedan lugares libres los martes a la tarde: más abajo te contamos cómo llenarlos.",
  "indicadores": [
    { "clave": "facturado", "titulo": "FACTURADO", "valor": 2184000, "anterior": 1950000,
      "valorTexto": "$ 2.184.000", "texto": "12% más que septiembre", "tendencia": "sube", "bueno": true },
    { "clave": "turnos", "titulo": "TURNOS", "valor": 312, "anterior": 284,
      "valorTexto": "312", "texto": "28 más que septiembre", "tendencia": "sube", "bueno": true },
    { "clave": "ocupacion", "titulo": "OCUPACIÓN", "valor": 78.2, "anterior": 72.1,
      "valorTexto": "78%", "texto": "6 puntos más", "tendencia": "sube", "bueno": true },
    { "clave": "ausencias", "titulo": "AUSENCIAS", "valor": 3.1, "anterior": 9,
      "valorTexto": "3%", "texto": "Bajó del 9%", "tendencia": "baja", "bueno": true }
  ],
  "resumen": { "turnos": 312, "atendidos": 300, "ingresos": 2184000, "ticketPromedio": 7280, "clientes": 200,
               "tasaAusencias": 3.1, "ocupacion": 78.2 },
  "resumenAnterior": { "…": "lo mismo, del mes anterior" },
  "agenda": {
    "ocupacion": 78.2,
    "titulo": "Ocupaste el 78% de tus horas libres.",
    "mapa": { "dias": [1, 2, 3, 4, 5, 6], "horas": [10, 11, 12], "celdas": [
      { "dia": 1, "hora": 10, "capacidadMinutos": 480, "ocupadoMinutos": 456, "ocupacion": 95, "vacio": 5, "estado": "lleno" }
    ], "capacidadMinutos": 9000, "ocupadoMinutos": 7040 },
    "horasLibres": 38,
    "franjaLibre": { "dias": [2, 3], "horaDesde": 14, "horaHasta": 16, "texto": "Martes y miércoles, 14 a 16 h",
                     "detalle": "Ahí está la mitad de lo libre del mes.", "momento": "los martes y miércoles a la tarde" },
    "huecosLlenados": 21,
    "sugerencia": { "titulo": "Para llenar los martes y miércoles a la tarde",
                    "texto": "Probá un cupón de 15% sólo para martes y miércoles, 14 a 16 h, y mandáselo a los 46 clientes que no vienen hace más de 60 días. Con la mitad de esos turnos llenarías unas 19 horas más por mes." }
  },
  "plata": {
    "titulo": "Entraron $ 2.184.000, y $ 412.000 fueron señas cobradas.",
    "ingresos": 2184000, "senas": 412000, "turnosConSena": 264,
    "membresias": 360000, "sociosActivos": 18,
    "ticketPromedio": 7000, "ticketAnterior": 6550, "ticketTexto": "Subió $ 450",
    "semanas": [ { "etiqueta": "1–7 oct", "ingresos": 448000 }, { "etiqueta": "8–14", "ingresos": 512000 } ],
    "servicios": [ { "nombre": "Lifting de pestañas", "turnos": 74, "facturado": 1628000, "parte": 34.1 },
                   { "nombre": "Otros", "turnos": 37, "facturado": 382000, "parte": 8 } ],
    "nota": "Montos finales, con descuentos aplicados. Sin la comisión de Mercado Pago."
  },
  "equipo": [ { "id": "…", "nombre": "Mica", "ocupacion": 86, "turnos": 124, "resenas": 12, "puntaje": 4.9 } ],
  "clientes": { "nuevos": 58, "recurrentes": 142, "volvieron": 71, "nuevas": 29, "porReferido": 9,
                "texto": "58 clientes nuevos: 9 llegaron por referidos." },
  "resenas": { "cantidad": 27, "promedio": 4.9,
               "destacada": { "comentario": "Me atendieron a horario y el lifting quedó divino.", "nombre": "Carla",
                              "profesional": "Mica", "puntaje": 5 } },
  "destacados": [
    { "titulo": "Mejor día", "texto": "El viernes 23 fue tu mejor día: $ 180.000 en 14 turnos." },
    { "titulo": "Lo más pedido", "texto": "Perfilado de cejas: 88 turnos." },
    { "titulo": "Del equipo", "texto": "Mica atendió 124 turnos, con 86% de ocupación." },
    { "titulo": "Reseñas", "texto": "4,9 ★ en 27 reseñas nuevas." }
  ],
  "siguiente": [
    { "numero": "01", "titulo": "Llená los martes y miércoles a la tarde",
      "texto": "Cupón de 15% para 14 a 16 h y aviso a los clientes que no vienen hace 60 días." },
    { "numero": "02", "titulo": "Sumá horarios donde hay espera",
      "texto": "Quedan 9 personas en lista de espera: un día más de atención los convierte en turnos." }
  ],
  "pie": "Hecho con Tengo Libre · tengolibre.com.ar/lumi"
}
```

Cómo sale cada cosa:

- Facturado, turnos, ocupación, ausencias, ticket, clientes nuevos/recurrentes, por servicio y por
  profesional: lo mismo que `GET /v1/panel/reportes` para ese mes (facturado = precio de los turnos
  atendidos; ocupación = minutos reservados sobre minutos de horario; ausencias = ausentes sobre
  atendidos + ausentes). `valor`/`anterior` pueden ser `null` (sin horarios o sin turnos cerrados).
- `titulo`: «Tu mejor mes del año.» si facturó más que cada uno de los 11 meses anteriores; si no,
  «Un mes mejor que…», «Un mes más tranquilo que…», «Un mes parejo con…», o «Un mes sin turnos.».
- `agenda.mapa`: como en Horas muertas (`docs/api/horas-muertas.md`) pero del mes, con `estado`
  `lleno` (≥ 85% ocupado), `a_medias` (≥ 50%), `libre` o `cerrado` (los tres colores y el gris del
  diseño). `franjaLibre`: la peor franja de dos horas y los otros días con esas mismas horas más
  vacías que llenas; `null` si nada quedó vacío la mitad o más. `huecosLlenados`: avisos de la lista
  de espera del mes que terminaron en turno del mismo servicio y email en menos de 2 horas.
- `plata.senas`: pagos aprobados de Mercado Pago creados en el mes, menos lo devuelto. `membresias`:
  el precio de los planes de las membresías activas creadas en el mes; `sociosActivos`: vigentes al
  cierre.
- `servicios`: los 4 que más facturaron y «Otros» si hay más de 5. `parte` sobre el facturado.
- `equipo`: por turnos, con las reseñas del mes por nombre de profesional.
- `siguiente`: hasta 3 pasos (franja libre, lista de espera, seña si las ausencias pasan el 8%,
  reseñas si hay menos de una cada 10 atendidos, compartir el link).
- El diseño también muestra de dónde llegaron las clientas nuevas (Instagram): la API no registra
  el origen, sólo los referidos (`porReferido`).

Errores: `400 invalido` si `mes` no es `AAAA-MM` o todavía no empezó; 403 sin rol de gestión.

## GET /v1/panel/reportes/mensual/ajustes

Roles: `duenio`, `encargado`.

```json
{
  "activo": true,
  "encargados": [ { "id": "…", "nombre": "Carla", "email": "carla@lumi.com", "recibe": false } ]
}
```

`activo` (por defecto `true`): el negocio recibe el reporte; le llega siempre a la dueña. Cada
encargado activo elige si lo recibe (`recibe`, por defecto `false`).

## PUT /v1/panel/reportes/mensual/ajustes

Roles: `duenio`, `encargado`. Lo que no viene no cambia.

```json
{ "activo": true, "encargados": [ { "id": "…", "recibe": true } ] }
```

Respuesta `200`: como el GET. La dueña cambia a cualquier encargado; un encargado sólo a sí mismo.
Queda en la auditoría como `ajustes.reporte_mensual`.

Errores: `404` si un `id` no es un encargado del negocio (o es otro encargado y quien pide no es la
dueña); `402 suscripcion_requerida` con el panel en sólo lectura.

## El mail

- Trabajo del worker `reporte_mensual` (cada hora): el día 1, desde las 8 del negocio, encola un
  aviso `mensual` por persona (dueñas activas y encargados con `recibe`), con clave
  `mensual:AAAA-MM:<miembro>`: sale una vez aunque corra todo el día.
- Se arma al mandarlo, con los datos de ese momento. Se descarta si el negocio lo apagó, si la persona
  ya no es del equipo o dejó de pedirlo, o si el mes no tuvo turnos.
- Contenido (plantilla común de avisos): encabezado «Reporte mensual · Lumi Estética · octubre
  2026», asunto «Tu octubre 2026 en Lumi Estética», el titular y la bajada, una fila por número
  («Facturado: $ 2.184.000 · 12% más que septiembre», …, «Clientes», «Huecos»), lo mejor del mes y
  los pasos para el mes que viene, y el botón «Ver el reporte completo» a
  `https://tengolibre.com.ar/panel/reportes/mensual?mes=2026-10`: la página imprimible que arma el
  front con el GET de arriba.
