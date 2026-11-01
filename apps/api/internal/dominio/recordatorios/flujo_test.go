package recordatorios_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/avisos"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/catalogo"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/clientes"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/disponibilidad"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/negocios"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/promos"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/recordatorios"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/turnos"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/correo"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/prueba"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/servidor"
)

const (
	duenia = "user_duenia0001"
	sitio  = "https://tengolibre.com.ar"
)

type P = prueba.Pedido

func en(s string) time.Time {
	t, err := time.ParseInLocation("2006-01-02 15:04", s, disponibilidad.ZonaPorDefecto)
	if err != nil {
		panic(err)
	}
	return t
}

func TestCumpleaniosYHaceMuchoQueNoVenis(t *testing.T) {
	var rec *recordatorios.Servicio
	a := prueba.Nueva(t, en("2026-08-30 10:00"), func(a *prueba.API) []servidor.Dominio {
		app := a.Bases.App
		rec = &recordatorios.Servicio{Pool: app, Sesion: a.Sesion, Sitio: sitio, Ahora: a.Reloj()}
		return []servidor.Dominio{
			&negocios.Servicio{Pool: app, Sesion: a.Sesion},
			&catalogo.Servicio{Pool: app, Sesion: a.Sesion},
			&turnos.Servicio{Pool: app, Sesion: a.Sesion, Ahora: a.Reloj()},
			&clientes.Servicio{Pool: app, Sesion: a.Sesion, Ahora: a.Reloj()},
			&promos.Servicio{Pool: app, Sesion: a.Sesion, Ahora: a.Reloj()},
			&avisos.Servicio{Pool: app, Sesion: a.Sesion, Ahora: a.Reloj()},
			rec,
		}
	})
	var n negocios.Negocio
	a.Esperar(201, P{Metodo: "POST", Ruta: "/v1/negocios", Quien: duenia, Cuerpo: map[string]string{"slug": "el-corte", "nombre": "El Corte"}}, &n)
	panel := func(estado int, metodo, ruta string, cuerpo, v any) prueba.Respuesta {
		t.Helper()
		return a.Esperar(estado, P{Metodo: metodo, Ruta: ruta, Quien: duenia, Negocio: n.ID, Cuerpo: cuerpo}, v)
	}
	var s catalogo.Prestacion
	panel(201, "POST", "/v1/panel/servicios", map[string]any{"nombre": "Corte", "duracion": 30, "precio": 8000}, &s)
	var horario []disponibilidad.Franja
	for d := range 7 {
		horario = append(horario, disponibilidad.Franja{Dia: d, Desde: "09:00", Hasta: "18:00"})
	}
	var ana catalogo.Profesional
	panel(201, "POST", "/v1/panel/profesionales", map[string]any{"nombre": "Ana", "servicios": []string{s.ID}, "horario": horario, "orden": 1}, &ana)
	reservar := func(inicio, nombre, telefono, nacimiento string) turnos.Turno {
		t.Helper()
		var x turnos.Turno
		a.Esperar(201, P{Metodo: "POST", Ruta: "/v1/p/el-corte/turnos", Cuerpo: map[string]any{
			"servicioId": s.ID, "profesionalId": ana.ID, "inicio": en(inicio).Format(time.RFC3339),
			"cliente": map[string]string{"nombre": nombre, "telefono": telefono, "email": strings.ToLower(strings.Fields(nombre)[0]) + "@ejemplo.com",
				"nacimiento": nacimiento}}}, &x)
		return x
	}
	atendido := func(inicio, nombre, telefono, nacimiento string) turnos.Turno {
		t.Helper()
		a.Ahora = en(inicio).AddDate(0, 0, -1)
		x := reservar(inicio, nombre, telefono, nacimiento)
		panel(200, "PATCH", "/v1/panel/turnos/"+x.ID, map[string]any{"estado": "atendido"}, nil)
		return x
	}
	// Paula y Tomás vinieron el 2/9 (60 días antes del 1/11); Tomás ya tiene otro turno. Sofía dejó su
	// cumple al reservar; Mara lo tiene cargado desde el panel pero se dio de baja; Lucía nunca vino.
	atendido("2026-09-02 10:00", "Paula Gómez", "1111", "")
	atendido("2026-09-02 11:00", "Tomás Ruiz", "2222", "")
	sofia := atendido("2026-09-20 10:00", "Sofía Méndez", "3333", "1990-11-03")
	mara := atendido("2026-09-21 10:00", "Mara Paz", "4444", "")
	a.Ahora = en("2026-10-20 10:00")
	reservar("2026-11-10 10:00", "Tomás Ruiz", "2222", "")
	// Una reserva nueva no pisa el cumple que ya estaba.
	reservar("2026-10-25 10:00", "Sofía Méndez", "3333", "2000-01-01")
	var ficha clientes.Ficha
	panel(200, "GET", "/v1/panel/clientes/"+sofia.Cliente.ID, nil, &ficha)
	if ficha.Nacimiento == nil || *ficha.Nacimiento != "1990-11-03" {
		t.Fatalf("nacimiento de Sofía: %v", ficha.Nacimiento)
	}
	var c clientes.Cliente
	panel(200, "PATCH", "/v1/panel/clientes/"+mara.Cliente.ID, map[string]any{"nacimiento": "1988-11-05"}, &c)
	if c.Nacimiento == nil || *c.Nacimiento != "1988-11-05" {
		t.Fatalf("nacimiento de Mara: %v", c.Nacimiento)
	}
	panel(400, "PATCH", "/v1/panel/clientes/"+mara.Cliente.ID, map[string]any{"nacimiento": "2030-01-01"}, nil)
	panel(400, "PATCH", "/v1/panel/clientes/"+mara.Cliente.ID, map[string]any{"nacimiento": "5/11/1988"}, nil)
	a.Esperar(400, P{Metodo: "POST", Ruta: "/v1/p/el-corte/turnos", Cuerpo: map[string]any{"servicioId": s.ID, "profesionalId": ana.ID,
		"inicio": en("2026-10-26 10:00").Format(time.RFC3339), "cliente": map[string]string{"nombre": "X", "telefono": "9999", "nacimiento": "ayer"}}}, nil)
	panel(201, "POST", "/v1/panel/clientes", map[string]any{"nombre": "Lucía", "email": "lucia@ejemplo.com", "nacimiento": "1995-11-04"}, &c)
	if c.Nacimiento == nil || *c.Nacimiento != "1995-11-04" {
		t.Fatalf("alta con nacimiento: %+v", c)
	}
	ctx := context.Background()
	var token string
	if err := a.Bases.App.EnNegocio(ctx, n.ID, func(tx pgx.Tx) error {
		p, err := avisos.PreferenciasDe(ctx, tx, mara.Cliente.ID)
		token = p.Token
		return err
	}); err != nil {
		t.Fatal(err)
	}
	a.Esperar(200, P{Metodo: "PUT", Ruta: "/v1/preferencias/" + token, Cuerpo: map[string]bool{"novedades": false}}, nil)

	// Domingo 1/11 a las 9: los dos apagados; esta semana cumple Sofía y le toca a Paula.
	a.Ahora = en("2026-11-01 09:00")
	type lista struct {
		Recordatorios []recordatorios.RecordatorioPanel `json:"recordatorios"`
	}
	var l lista
	panel(200, "GET", "/v1/panel/recordatorios", nil, &l)
	if len(l.Recordatorios) != 2 {
		t.Fatalf("recordatorios: %+v", l)
	}
	cu, ex := l.Recordatorios[0], l.Recordatorios[1]
	if cu.Tipo != "cumpleanios" || cu.Activo || len(cu.Semana) != 1 || cu.Semana[0].Nombre != "Sofía Méndez" || cu.Semana[0].Fecha != "2026-11-03" ||
		cu.Semana[0].Detalle != "Cumple el martes 3 de noviembre (36)" || cu.SemanaTexto != "Esta semana: 1 cumpleaños" ||
		cu.VistaPrevia.Asunto != "¡Feliz cumple, Sofía!" || !strings.Contains(cu.VistaPrevia.HTML, "Que tengas un muy lindo día") {
		t.Fatalf("cumpleaños: %+v", cu)
	}
	if ex.Tipo != "extraniamos" || ex.Dias != 60 || len(ex.Semana) != 1 || ex.Semana[0].Nombre != "Paula Gómez" || ex.Semana[0].Fecha != "2026-11-01" ||
		ex.VistaPrevia.Asunto != "Te extrañamos, Paula" || !strings.Contains(ex.VistaPrevia.Texto, "Hace 9 semanas de tu último corte.") ||
		ex.Regla != "Le llega cuando pasan 60 días desde su último turno atendido y no tiene otro reservado." {
		t.Fatalf("hace mucho: %+v", ex)
	}

	// Se prenden, con texto propio y cupón en el de cumpleaños.
	hasta := en("2026-12-31 23:00")
	var cupon promos.Cupon
	panel(201, "POST", "/v1/panel/cupones", map[string]any{"codigo": "cumple20", "tipo": "porcentaje", "valor": 20, "hasta": hasta}, &cupon)
	var rp recordatorios.RecordatorioPanel
	panel(200, "PUT", "/v1/panel/recordatorios/cumpleanios", map[string]any{"activo": true,
		"texto": "En El Corte te regalamos un 20% en lo que quieras este mes.", "cuponId": cupon.ID}, &rp)
	if !rp.Activo || rp.Cupon == nil || rp.Cupon.Codigo != "CUMPLE20" || rp.Cupon.Descuento != "20%" ||
		!strings.Contains(rp.VistaPrevia.Texto, "Código: CUMPLE20") || !strings.Contains(rp.VistaPrevia.Texto, "te regalamos un 20%") {
		t.Fatalf("cumpleaños prendido: %+v", rp)
	}
	panel(200, "PUT", "/v1/panel/recordatorios/extraniamos", map[string]any{"activo": true, "dias": 60}, &rp)
	panel(404, "PUT", "/v1/panel/recordatorios/aniversario", map[string]any{"activo": true}, nil)
	panel(400, "PUT", "/v1/panel/recordatorios/cumpleanios", map[string]any{"texto": strings.Repeat("a", 301)}, nil)
	panel(400, "PUT", "/v1/panel/recordatorios/cumpleanios", map[string]any{"cuponId": "00000000-0000-0000-0000-000000000000"}, nil)
	panel(400, "PUT", "/v1/panel/recordatorios/extraniamos", map[string]any{"dias": 5}, nil)
	var vp recordatorios.VistaPrevia
	panel(200, "GET", "/v1/panel/recordatorios/extraniamos/vista-previa?texto=Volv%C3%A9+cuando+quieras&cuponId=", nil, &vp)
	if !strings.Contains(vp.Texto, "Volvé cuando quieras") || strings.Contains(vp.Texto, "Código") {
		t.Fatalf("vista previa: %+v", vp)
	}

	// El trabajo: antes de las 10 no encola; a las 10, a Paula (una sola vez).
	mails := &correo.Memoria{}
	env := &avisos.Enviador{Pool: a.Bases.App, Correo: mails, Sitio: sitio, Armadores: recordatorios.Armadores()}
	encolar := func() int {
		t.Helper()
		r, err := rec.Encolar(ctx, a.Ahora)
		if err != nil {
			t.Fatal(err)
		}
		return r.(map[string]int)["encolados"]
	}
	if k := encolar(); k != 0 {
		t.Fatalf("a las 9 encoló %d", k)
	}
	a.Ahora = en("2026-11-01 10:00")
	if k := encolar(); k != 1 {
		t.Fatalf("a las 10 encoló %d", k)
	}
	if k := encolar(); k != 0 {
		t.Fatalf("la segunda pasada encoló %d", k)
	}
	if r, err := env.Pasada(ctx, a.Ahora); err != nil || r.Enviados != 1 {
		t.Fatalf("envío: %+v %v", r, err)
	}
	m := mails.Para("paula@ejemplo.com")
	if len(m) != 1 || m[0].Asunto != "Te extrañamos, Paula" || !strings.Contains(m[0].Texto, "Hace 9 semanas de tu último corte.") ||
		!strings.Contains(m[0].Texto, sitio+"/mi-cuenta/avisos/") || m[0].Baja == "" {
		t.Fatalf("mail a Paula: %+v", m)
	}

	// Martes 3: el cumple de Sofía, con el cupón.
	a.Ahora = en("2026-11-03 10:30")
	if k := encolar(); k != 1 {
		t.Fatalf("el martes encoló %d", k)
	}
	if r, err := env.Pasada(ctx, a.Ahora); err != nil || r.Enviados != 1 {
		t.Fatalf("envío del cumple: %+v %v", r, err)
	}
	m = mails.Para("sofía@ejemplo.com")
	if len(m) != 1 || m[0].Asunto != "¡Feliz cumple, Sofía!" || !strings.Contains(m[0].Texto, "Código: CUMPLE20") ||
		!strings.Contains(m[0].Texto, sitio+"/el-corte?cupon=CUMPLE20") {
		t.Fatalf("mail de cumple: %+v", m)
	}
	panel(200, "GET", "/v1/panel/recordatorios", nil, &l)
	if len(l.Recordatorios[0].Semana) == 0 || !l.Recordatorios[0].Semana[0].Enviado {
		t.Fatalf("el de Sofía no figura enviado: %+v", l.Recordatorios[0].Semana)
	}
	if mails.Cantidad() != 2 {
		t.Fatalf("salieron %d mails", mails.Cantidad())
	}
}
