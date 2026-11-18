package asistente_test

import (
	"strings"
	"testing"
	"time"

	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/asistente"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/avisos"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/catalogo"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/disponibilidad"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/espera"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/mcp"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/negocios"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/suscripcion"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/turnos"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/ia"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/prueba"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/servidor"
)

type P = prueba.Pedido

const (
	duenia = "user_duenia0001"
	profe  = "user_profe00001"
)

func en(s string) time.Time {
	t, err := time.ParseInLocation("2006-01-02 15:04", s, disponibilidad.ZonaPorDefecto)
	if err != nil {
		panic(err)
	}
	return t
}

// entorno: miércoles 18/11 a las 10:20. Corte de 30 min con Ana y Beto (lunes a sábado de 9 a 20);
// mañana Juana a las 15 con Ana y Pedro a las 16 con Beto (sin confirmar).
type entorno struct {
	*prueba.API
	t            *testing.T
	negocio      string
	servicio     catalogo.Prestacion
	ana, beto    catalogo.Profesional
	juana, pedro turnos.Turno
	falso        *ia.Falso
	asis         *asistente.Servicio
}

func nuevo(t *testing.T, conIA bool, topes ia.Topes) *entorno {
	e := &entorno{t: t}
	if conIA {
		e.falso = &ia.Falso{}
	}
	e.API = prueba.Nueva(t, en("2026-11-18 10:20"), func(a *prueba.API) []servidor.Dominio {
		app := a.Bases.App
		av := &avisos.Servicio{Pool: app, Sesion: a.Sesion, Ahora: a.Reloj()}
		sus := &suscripcion.Servicio{Pool: app, Sesion: a.Sesion, Ahora: a.Reloj()}
		a.Sesion.Escritura = sus.Escritura
		a.Sesion.SinChequeoDeEscritura = []string{asistente.SinChequeo}
		tur := &turnos.Servicio{Pool: app, Sesion: a.Sesion, Ahora: a.Reloj(), Avisar: av.Avisar}
		m := &mcp.Servicio{Pool: app, Sesion: a.Sesion, Turnos: tur, Ahora: a.Reloj()}
		esp := &espera.Servicio{Pool: app, Sesion: a.Sesion, Turnos: tur, Ahora: a.Reloj()}
		e.asis = &asistente.Servicio{Pool: app, Sesion: a.Sesion, MCP: m, Espera: esp, Ahora: a.Reloj(),
			Medidor: &ia.Medidor{Pool: app, Topes: topes, Ahora: a.Reloj()}, Cache: &ia.Cache{Ahora: a.Reloj()}}
		if conIA {
			e.asis.IA = e.falso
		}
		return []servidor.Dominio{&negocios.Servicio{Pool: app, Sesion: a.Sesion}, &catalogo.Servicio{Pool: app, Sesion: a.Sesion},
			tur, av, sus, esp, m, e.asis}
	})
	var n negocios.Negocio
	e.Esperar(201, P{Metodo: "POST", Ruta: "/v1/negocios", Quien: duenia, Cuerpo: map[string]string{"slug": "el-corte", "nombre": "El Corte"}}, &n)
	e.negocio = n.ID
	e.como(duenia, 201, "POST", "/v1/panel/servicios", map[string]any{"nombre": "Corte", "duracion": 30, "precio": 8000}, &e.servicio)
	var semana []disponibilidad.Franja
	for d := 1; d <= 6; d++ {
		semana = append(semana, disponibilidad.Franja{Dia: d, Desde: "09:00", Hasta: "20:00"})
	}
	e.como(duenia, 201, "POST", "/v1/panel/profesionales", map[string]any{"nombre": "Ana", "servicios": []string{e.servicio.ID},
		"horario": semana, "orden": 1}, &e.ana)
	e.como(duenia, 201, "POST", "/v1/panel/profesionales", map[string]any{"nombre": "Beto", "servicios": []string{e.servicio.ID},
		"horario": semana, "orden": 2}, &e.beto)
	e.juana = e.turno(e.ana.ID, "2026-11-19 15:00", "Juana Pérez", "1155550001")
	e.pedro = e.turno(e.beto.ID, "2026-11-19 16:00", "Pedro Gómez", "1155550002")
	e.Fila("UPDATE turnos SET estado = 'pendiente' WHERE id = $1", []any{e.pedro.ID})
	return e
}

func (e *entorno) como(quien string, estado int, metodo, ruta string, cuerpo, v any) prueba.Respuesta {
	e.t.Helper()
	return e.Esperar(estado, P{Metodo: metodo, Ruta: ruta, Quien: quien, Negocio: e.negocio, Cuerpo: cuerpo}, v)
}

func (e *entorno) turno(prof, cuando, nombre, tel string) turnos.Turno {
	var x turnos.Turno
	e.como(duenia, 201, "POST", "/v1/panel/turnos", map[string]any{"servicioId": e.servicio.ID, "profesionalId": prof,
		"inicio": en(cuando).Format(time.RFC3339), "cliente": map[string]string{"nombre": nombre, "telefono": tel}}, &x)
	return x
}

func (e *entorno) preguntar(quien, pregunta string) asistente.Respuesta {
	e.t.Helper()
	var r asistente.Respuesta
	e.como(quien, 200, "POST", "/v1/panel/asistente", map[string]any{"pregunta": pregunta}, &r)
	return r
}

func (e *entorno) ver(id string) turnos.Turno {
	var t turnos.Turno
	e.como(duenia, 200, "GET", "/v1/panel/turnos/"+id, nil, &t)
	return t
}

// Sin IA: el intérprete contesta, propone y la propuesta se confirma o se descarta.
func TestMotorSinIA(t *testing.T) {
	e := nuevo(t, false, ia.Topes{})
	r := e.preguntar(duenia, "¿Qué tengo mañana?")
	if r.Origen != "motor" || !strings.Contains(r.Respuesta, "Juana Pérez") || !strings.Contains(r.Respuesta, "Pedro Gómez") ||
		!strings.Contains(r.Respuesta, "2 turnos") {
		t.Fatalf("agenda: %+v", r)
	}
	r = e.preguntar(duenia, "¿quién viene mañana a las 4?")
	if !strings.Contains(r.Respuesta, "Pedro Gómez") || strings.Contains(r.Respuesta, "Juana") {
		t.Fatalf("quién: %+v", r)
	}
	r = e.preguntar(duenia, "¿Hay lugar mañana a la tarde?")
	if !strings.Contains(r.Respuesta, "hay lugar para Corte") || !strings.Contains(r.Respuesta, "15:30") {
		t.Fatalf("libres: %+v", r)
	}

	// Mover: nada cambia hasta confirmar.
	r = e.preguntar(duenia, "Pasá a Juana al viernes a las 5")
	if len(r.Propuestas) != 1 || r.Propuestas[0].Accion != "reprogramar_turno" || !strings.Contains(r.Propuestas[0].Resumen, "viernes 20") ||
		!strings.Contains(r.Propuestas[0].Resumen, "17:00") {
		t.Fatalf("mover: %+v", r)
	}
	if !e.ver(e.juana.ID).Inicio.Equal(en("2026-11-19 15:00")) {
		t.Fatal("el turno se movió sin confirmar")
	}
	id := r.Propuestas[0].ID
	// Otra persona del equipo no confirma lo que pidió otra.
	e.como(duenia, 201, "POST", "/v1/panel/equipo", map[string]string{"clerkId": profe, "rol": "profesional"}, nil)
	e.como(profe, 404, "POST", "/v1/panel/asistente/confirmar", map[string]string{"propuestaId": id}, nil)
	var hecho map[string]any
	e.como(duenia, 200, "POST", "/v1/panel/asistente/confirmar", map[string]string{"propuestaId": id}, &hecho)
	if !strings.HasPrefix(hecho["respuesta"].(string), "Listo. Movimos el turno de Juana Pérez") {
		t.Fatalf("confirmar: %v", hecho)
	}
	if !e.ver(e.juana.ID).Inicio.Equal(en("2026-11-20 17:00")) {
		t.Fatal("confirmado y no se movió")
	}
	e.como(duenia, 409, "POST", "/v1/panel/asistente/confirmar", map[string]string{"propuestaId": id}, nil)

	// Cancelar y descartar: queda como estaba.
	r = e.preguntar(duenia, "cancelá el de las 4 de mañana")
	if len(r.Propuestas) != 1 || r.Propuestas[0].Accion != "cancelar_turno" || !strings.Contains(r.Propuestas[0].Resumen, "Pedro Gómez") {
		t.Fatalf("cancelar: %+v", r)
	}
	e.como(duenia, 200, "POST", "/v1/panel/asistente/descartar", map[string]string{"propuestaId": r.Propuestas[0].ID}, nil)
	if e.ver(e.pedro.ID).Estado != "pendiente" {
		t.Fatal("descartado y se canceló")
	}

	// Confirmar a Pedro.
	r = e.preguntar(duenia, "Confirmá a Pedro")
	if len(r.Propuestas) != 1 || r.Propuestas[0].Accion != "confirmar_turno" {
		t.Fatalf("confirmar a Pedro: %+v", r)
	}
	e.como(duenia, 200, "POST", "/v1/panel/asistente/confirmar", map[string]string{"propuestaId": r.Propuestas[0].ID}, nil)
	if e.ver(e.pedro.ID).Estado != "confirmado" {
		t.Fatal("no se confirmó")
	}

	// Un horario ocupado no se propone: se explica y se ofrecen los más cercanos.
	e.turno(e.beto.ID, "2026-11-20 17:00", "Lu", "1155550003")
	r = e.preguntar(duenia, "pasá a Pedro al viernes a las 5")
	if len(r.Propuestas) != 0 || !strings.Contains(r.Respuesta, "no está libre") || !strings.Contains(r.Respuesta, "17:30") {
		t.Fatalf("ocupado: %+v", r)
	}

	// Una propuesta vencida no se confirma.
	r = e.preguntar(duenia, "cancelá a Pedro")
	e.Ahora = e.Ahora.Add(asistente.VidaPropuesta + time.Minute)
	e.como(duenia, 409, "POST", "/v1/panel/asistente/confirmar", map[string]string{"propuestaId": r.Propuestas[0].ID}, nil)

	// Sin IA, lo que no entiende lo dice.
	r = e.preguntar(duenia, "¿cuánto facturé este mes?")
	if r.Origen != "motor" || !strings.Contains(r.Respuesta, "todavía no lo sé") {
		t.Fatalf("sin IA: %+v", r)
	}
}

// Un profesional ve y toca sólo su agenda, también por el asistente.
func TestRolProfesional(t *testing.T) {
	e := nuevo(t, false, ia.Topes{})
	var m negocios.Miembro
	e.como(duenia, 201, "POST", "/v1/panel/equipo", map[string]string{"clerkId": profe, "rol": "profesional"}, &m)
	e.como(duenia, 200, "PATCH", "/v1/panel/profesionales/"+e.ana.ID, map[string]any{"miembroId": m.ID}, nil)
	r := e.preguntar(profe, "¿Qué tengo mañana?")
	if !strings.Contains(r.Respuesta, "Juana") || strings.Contains(r.Respuesta, "Pedro") {
		t.Fatalf("agenda del profesional: %+v", r)
	}
	r = e.preguntar(profe, "cancelá a Pedro")
	if len(r.Propuestas) != 0 || !strings.Contains(r.Respuesta, "No encontré") {
		t.Fatalf("tocó un turno ajeno: %+v", r)
	}
	r = e.preguntar(profe, "Pasá a Juana al viernes a las 10")
	if len(r.Propuestas) != 1 {
		t.Fatalf("no pudo mover su turno: %+v", r)
	}
	e.como(profe, 200, "POST", "/v1/panel/asistente/confirmar", map[string]string{"propuestaId": r.Propuestas[0].ID}, nil)
	if !e.ver(e.juana.ID).Inicio.Equal(en("2026-11-20 10:00")) {
		t.Fatal("no se movió")
	}
}

// Con IA (cliente falso): el modelo lee la agenda, propone mover y contesta; la propuesta no se
// hace sola; el uso queda medido; lo que no propone nada queda en el caché.
func TestConIA(t *testing.T) {
	e := nuevo(t, true, ia.Topes{CupoMensual: 100, TopeDiario: 100, PresupuestoMicro: 1_000_000})
	e.falso.Respuestas = []string{
		ia.RespuestaHerramienta("tu_1", "ver_agenda", map[string]any{"desde": "2026-11-19"}),
		ia.RespuestaHerramienta("tu_2", "reprogramar_turno", map[string]any{"turnoId": e.juana.ID, "fecha": "2026-11-21", "hora": "11:00"}),
		ia.RespuestaTexto("Te propuse pasar a Juana al sábado a las 11. Tocá «Confirmar»."),
	}
	r := e.preguntar(duenia, "a la de las 3 de mañana pasala al sábado temprano, a las 11")
	if r.Origen != "ia" || len(r.Propuestas) != 1 || r.Propuestas[0].Accion != "reprogramar_turno" ||
		!strings.Contains(r.Propuestas[0].Resumen, "sábado 21") {
		t.Fatalf("con IA: %+v", r)
	}
	if !e.ver(e.juana.ID).Inicio.Equal(en("2026-11-19 15:00")) {
		t.Fatal("la IA movió el turno sin confirmación")
	}
	// El modelo recibió las herramientas del rol y los datos del negocio con el breakpoint.
	p := e.falso.Recibidos[0]
	if len(p.Tools) < 10 || p.Model != ia.Modelo || !strings.Contains(p.Messages[0].Content[0].OfText.Text, "El Corte") {
		t.Fatalf("pedido al modelo: %d herramientas", len(p.Tools))
	}
	var respuestas, pedidos int
	e.Fila("SELECT respuestas, pedidos FROM ia_uso WHERE negocio_id = $1", []any{e.negocio}, &respuestas, &pedidos)
	if respuestas != 1 || pedidos != 3 {
		t.Fatalf("uso: %d respuestas, %d pedidos", respuestas, pedidos)
	}

	// Caché de respuestas: la misma pregunta (sin propuestas) no vuelve al modelo.
	e.falso.Respuestas = []string{ia.RespuestaTexto("Este mes facturaste $ 0.")}
	if r := e.preguntar(duenia, "¿Cuánto facturé este mes?"); r.Origen != "ia" {
		t.Fatalf("%+v", r)
	}
	antes := e.falso.Llamadas()
	if r := e.preguntar(duenia, "cuanto facture este mes"); r.Origen != "cache" || r.Respuesta != "Este mes facturaste $ 0." {
		t.Fatalf("caché: %+v", r)
	}
	if e.falso.Llamadas() != antes {
		t.Fatal("la pregunta repetida fue al modelo")
	}

	// Dos preguntas seguidas mandan el mismo prefijo (herramientas, system y datos).
	if string(ia.Prefijo(e.falso.Recibidos[0])) != string(ia.Prefijo(e.falso.Recibidos[len(e.falso.Recibidos)-1])) {
		t.Fatal("el prefijo cambió entre dos preguntas del mismo negocio")
	}

	// El intérprete sigue primero: lo que entiende no llega al modelo.
	if r := e.preguntar(duenia, "¿qué tengo mañana?"); r.Origen != "motor" || e.falso.Llamadas() != antes {
		t.Fatalf("fue a la IA: %+v", r)
	}
}

// Topes: el cupo del mes, el del día y el presupuesto de la plataforma.
func TestTopes(t *testing.T) {
	e := nuevo(t, true, ia.Topes{CupoMensual: 2, TopeDiario: 1, PresupuestoMicro: 5000})
	e.falso.Respuestas = []string{ia.RespuestaTexto("Uno."), ia.RespuestaTexto("Dos."), ia.RespuestaTexto("Tres.")}
	if r := e.preguntar(duenia, "contame algo del negocio"); r.Origen != "ia" {
		t.Fatalf("%+v", r)
	}
	r := e.preguntar(duenia, "contame otra cosa")
	if r.Tope == nil || r.Tope.Motivo != "dia" || e.falso.Llamadas() != 1 {
		t.Fatalf("tope diario: %+v", r)
	}
	// Al otro día pasa, y después se acaba el mes.
	e.Ahora = e.Ahora.Add(24 * time.Hour)
	if r := e.preguntar(duenia, "contame otra cosa"); r.Origen != "ia" {
		t.Fatalf("%+v", r)
	}
	e.Ahora = e.Ahora.Add(24 * time.Hour)
	if r := e.preguntar(duenia, "y otra más"); r.Tope == nil || r.Tope.Motivo != "mes" {
		t.Fatalf("tope del mes: %+v", r)
	}
	var estado struct {
		IA   bool      `json:"ia"`
		Cupo ia.Estado `json:"cupo"`
	}
	e.como(duenia, 200, "GET", "/v1/panel/asistente", nil, &estado)
	if !estado.IA || estado.Cupo.UsadasMes != 2 || estado.Cupo.CupoMensual != 2 {
		t.Fatalf("estado: %+v", estado)
	}

	// El presupuesto de toda la plataforma: otro negocio gastó todo.
	e2 := nuevo(t, true, ia.Topes{PresupuestoMicro: 5000})
	e2.Fila("INSERT INTO ia_uso (negocio_id, dia, ruta, costo_micro) VALUES ($1, '2026-11-02', 'asistente', 6000)", []any{e2.negocio})
	var otro negocios.Negocio
	e2.Esperar(201, P{Metodo: "POST", Ruta: "/v1/negocios", Quien: "user_otra000001", Cuerpo: map[string]string{"slug": "otro", "nombre": "Otro"}}, &otro)
	var rr asistente.Respuesta
	e2.Esperar(200, P{Metodo: "POST", Ruta: "/v1/panel/asistente", Quien: "user_otra000001", Negocio: otro.ID,
		Cuerpo: map[string]string{"pregunta": "contame algo"}}, &rr)
	if rr.Tope == nil || rr.Tope.Motivo != "plataforma" || e2.falso.Llamadas() != 0 {
		t.Fatalf("presupuesto global: %+v", rr)
	}
}
