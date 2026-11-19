# Asistente del panel (con IA)

El asistente que vive en el panel y en las apps: «¿qué tengo mañana?», «pasá a Juana al jueves a las 5».
Diseño común de Grupo Vimo: `~/Trabajo/vimo/docs-ia/asistente-ia.md`. Código: `apps/api/internal/dominio/asistente`
(intérprete, motor, propuestas) y `apps/api/internal/plataforma/ia` (cliente de Claude, caché, bucle, topes).
Migración `00031_asistente_ia.sql` (`ia_uso`, `asistente_propuestas`).

1. **Primero el intérprete propio** (`interprete.go`, sin IA ni costo): agenda de un día, quién viene a una hora,
   horarios libres (con franja: mañana / tarde / noche), mover, cancelar y confirmar un turno. Entiende fechas
   relativas (hoy, mañana, pasado mañana, «el jueves», «el jueves que viene», «el 20», «20/11», «2 de diciembre»),
   horas («a las 5» es 17 h si el negocio atiende a esa hora y no a las 5; «17hs», «5 y media», «9 de la noche») y los
   nombres de profesionales, servicios y clientes. Si queda una palabra que no entiende, no adivina.
2. **Si no entiende, Claude Haiku 5.5** con las herramientas del MCP del negocio (mismas operaciones, sin HTTP) y el
   rol de quien pregunta: un profesional sólo ve y toca su agenda; recepción no ve los números.
3. **Nada que cambie la agenda se hace directo**: vuelve como **propuesta** y la persona toca «Confirmar».

Roles: cualquiera del equipo (dueño, encargado, recepción, profesional). Preguntar anda con el panel en sólo
lectura; confirmar una propuesta, no (402 como el resto).

## Rutas

`GET /v1/panel/asistente`

```json
{ "ia": true, "sugerencias": ["¿Qué tengo mañana?", "…"],
  "cupo": { "cupoMensual": 150, "usadasMes": 12, "usadasHoy": 3, "topeDiario": 30 } }
```

`ia: false` cuando la API no tiene clave de Anthropic (contesta sólo el intérprete).

`POST /v1/panel/asistente`: `{"pregunta": "…" (1–500), "historial": [{"rol": "usuario" | "asistente", "texto": "…"}]}`
(se usan los últimos 8 turnos, recortados a 600 caracteres).

```json
{ "respuesta": "Mañana hay 2 turnos:\n• 15:00 Juana Pérez — Corte con Ana\n• …",
  "propuestas": [ { "id": "uuid", "accion": "reprogramar_turno",
                    "resumen": "Mover el turno de Juana Pérez (Corte) del jueves 19 de noviembre a las 15:00 al viernes 20 de noviembre a las 17:00. Le avisamos al cliente.",
                    "venceEn": "2026-11-18T10:35:00-03:00" } ],
  "origen": "motor",
  "tope": { "motivo": "mes", "mensaje": "Ya usaste las 150 respuestas con IA de este mes. …" } }
```

- `origen`: `motor` (intérprete, sin IA), `ia` o `cache` (la misma pregunta hace menos de 10 minutos).
- `tope` sólo cuando no hay más IA (motivo `mes`, `dia` o `plataforma`): mostrar `mensaje` tal cual; el intérprete
  sigue andando.
- `accion`: `crear_turno`, `confirmar_turno`, `reprogramar_turno`, `cancelar_turno` u `ofrecer_a_la_lista`.

`POST /v1/panel/asistente/confirmar` `{"propuestaId": "…"}` → `{"respuesta": "Listo. Movimos el turno de …", "resultado": {…}}`.
Sólo la persona que preguntó, dentro de los 15 minutos y una sola vez. Errores (problem+json): 404 (no existe o es de
otra persona), 409 `propuesta_resuelta`, `propuesta_vencida`, `horario_ocupado` u otro de la agenda, 402 sólo lectura.

`POST /v1/panel/asistente/descartar` `{"propuestaId": "…"}` → `{"respuesta": "Listo, no hago nada."}`.

## Costo y topes

- El cupo de cada negocio sale de su plan (`planes.CupoIA`): por mes y por día, Inicial 100 y 5, Equipo 400 y
  20, Centro 1.500 y 75. Variables: `ANTHROPIC_API_KEY`, `TENGOLIBRE_IA_TOPE_DIARIO` (100: techo diario de
  seguridad que ningún plan pasa), `TENGOLIBRE_IA_PRESUPUESTO_USD` (parte de Tengo Libre del presupuesto de IA del
  mes, 20) y `TENGOLIBRE_IA_CUPO_MENSUAL` (150: sólo el valor por defecto si no hay plan, en pruebas). 0 apaga ese
  tope. Ojo: con 20 USD y unos USD 0,006 por respuesta alcanzan ~3.300 respuestas al mes para toda la plataforma;
  dos negocios Centro que usen todo su cupo lo agotan (decisión de presupuesto pendiente con Nahuel).
- Sólo cuenta lo que llega al modelo. `ia_uso` guarda por negocio, día y ruta las respuestas, las llamadas, los tokens
  (entrada, caché leído y escrito, salida) y el costo en micro-dólares.
- Caché de prompts: herramientas (por nombre) → system fijo (breakpoint) → datos del negocio con la hora redondeada a
  15 minutos (breakpoint) → conversación (caché automático). El log `ia: respuesta` muestra `cache_leido`: si da 0 en
  preguntas seguidas del mismo negocio, algo invalida el prefijo.
