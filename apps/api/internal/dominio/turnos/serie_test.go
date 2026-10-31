package turnos_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/turnos"
)

func TestTurnoFijo(t *testing.T) {
	a, negocio, servicio, ana, _ := armar(t)
	panel := func(estado int, metodo, ruta string, cuerpo, v any) []byte {
		t.Helper()
		return a.Esperar(estado, P{Metodo: metodo, Ruta: ruta, Quien: duenia, Negocio: negocio, Cuerpo: cuerpo}, v).Cuerpo
	}
	// El 16/11 Ana ya tiene un turno a las 10; el 30/11 no atiende (bloqueo del día entero).
	reservar(a, 201, servicio, ana, "2026-11-16 10:00", "9999")
	panel(201, "POST", "/v1/panel/profesionales/"+ana+"/bloqueos", map[string]any{"fecha": "2026-11-30", "motivo": "Vacaciones"}, nil)

	// Cada 2 semanas, 4 turnos: el 14/12 cae fuera de los 30 días de la web, pero el panel lo ve.
	pedido := map[string]any{"servicioId": servicio, "profesionalId": ana, "inicio": en("2026-11-02 10:00").Format(time.RFC3339),
		"cadaSemanas": 2, "cantidad": 4}
	var pv turnos.Previa
	panel(200, "POST", "/v1/panel/series/previa", pedido, &pv)
	if len(pv.Fechas) != 4 || pv.Libres != 2 || pv.Chocan != 2 || pv.Profesional != "Ana" {
		t.Fatalf("previa: %+v", pv)
	}
	f := pv.Fechas
	if f[0].Fecha != "2026-11-02" || f[0].Estado != "libre" || f[0].Alternativa != nil || f[3].Fecha != "2026-12-14" || f[3].Estado != "libre" {
		t.Fatalf("fechas libres: %+v", f)
	}
	// Ocupado a las 10: lo más cerca es 9:30 o 10:30, y gana la más tarde.
	if f[1].Estado != "choca" || f[1].Motivo != turnos.MotivoOcupado || f[1].Alternativa == nil || f[1].Alternativa.Hora != "10:30" {
		t.Fatalf("16/11: %+v %+v", f[1], f[1].Alternativa)
	}
	if f[2].Estado != "choca" || f[2].Motivo != turnos.MotivoBloqueado || f[2].Alternativa != nil {
		t.Fatalf("30/11: %+v", f[2])
	}
	// Hasta una fecha da lo mismo.
	delete(pedido, "cantidad")
	pedido["hasta"] = "2026-12-14"
	panel(200, "POST", "/v1/panel/series/previa", pedido, &pv)
	if len(pv.Fechas) != 4 {
		t.Fatalf("hasta: %d fechas", len(pv.Fechas))
	}
	pedido["cantidad"] = 3
	panel(400, "POST", "/v1/panel/series/previa", pedido, nil)
	delete(pedido, "hasta")
	pedido["cadaSemanas"] = 5
	panel(400, "POST", "/v1/panel/series/previa", pedido, nil)

	// Se reservan las que se aceptaron: el 16/11 con la alternativa y el 30/11 afuera.
	var serie turnos.Serie
	panel(201, "POST", "/v1/panel/series", map[string]any{"servicioId": servicio, "profesionalId": ana, "cadaSemanas": 2,
		"cliente": map[string]string{"nombre": "Tomás", "telefono": "1155"},
		"turnos":  []map[string]any{{"inicio": f[0].Inicio}, {"inicio": f[1].Alternativa.Inicio}, {"inicio": f[3].Inicio}}}, &serie)
	if len(serie.Turnos) != 3 || serie.CadaSemanas != 2 || serie.Turnos[1].Inicio.Equal(f[1].Inicio) {
		t.Fatalf("serie: %+v", serie)
	}
	for _, x := range serie.Turnos {
		if x.SerieID == nil || *x.SerieID != serie.ID || x.Estado != "confirmado" || x.Cliente.ID != serie.ClienteID {
			t.Fatalf("turno de la serie: %+v", x)
		}
	}

	// Atómica: si una choca, no entra ninguna.
	var antes, despues struct{ Turnos []turnos.Turno }
	panel(200, "GET", "/v1/panel/turnos?desde=2026-11-01&hasta=2026-12-31", nil, &antes)
	cuerpo := panel(409, "POST", "/v1/panel/series", map[string]any{"servicioId": servicio, "profesionalId": ana, "cadaSemanas": 1,
		"clienteId": serie.ClienteID, "turnos": []map[string]any{{"inicio": en("2026-11-09 09:00")}, {"inicio": en("2026-11-16 10:30")}}}, nil)
	var prob struct {
		Code   string `json:"code"`
		Issues []struct {
			Path string `json:"path"`
		} `json:"issues"`
	}
	_ = json.Unmarshal(cuerpo, &prob)
	if prob.Code != "horario_ocupado" || len(prob.Issues) != 1 || prob.Issues[0].Path != "turnos[1].inicio" {
		t.Fatalf("choque: %s", cuerpo)
	}
	panel(200, "GET", "/v1/panel/turnos?desde=2026-11-01&hasta=2026-12-31", nil, &despues)
	if len(despues.Turnos) != len(antes.Turnos) {
		t.Fatalf("la serie que chocó dejó turnos: %d → %d", len(antes.Turnos), len(despues.Turnos))
	}
	// Desordenados, no.
	panel(400, "POST", "/v1/panel/series", map[string]any{"servicioId": servicio, "profesionalId": ana, "cadaSemanas": 1,
		"clienteId": serie.ClienteID, "turnos": []map[string]any{{"inicio": en("2026-11-16 09:00")}, {"inicio": en("2026-11-09 09:00")}}}, nil)

	// «Este y los siguientes» desde el segundo: quedan el primero y los dos cancelados.
	var out struct {
		Cancelados []string     `json:"cancelados"`
		Serie      turnos.Serie `json:"serie"`
	}
	panel(200, "POST", "/v1/panel/turnos/"+serie.Turnos[1].ID+"/cancelar-siguientes", nil, &out)
	if len(out.Cancelados) != 2 || out.Serie.Turnos[0].Estado != "confirmado" || out.Serie.Turnos[1].Estado != "cancelado" ||
		out.Serie.Turnos[2].Estado != "cancelado" {
		t.Fatalf("cancelar siguientes: %+v", out)
	}
	panel(200, "GET", "/v1/panel/series/"+serie.ID, nil, &serie)
	if len(serie.Turnos) != 3 || serie.Turnos[2].Estado != "cancelado" {
		t.Fatalf("serie después: %+v", serie)
	}
	// El horario del 16/11 10:30 se liberó.
	var l libresResp
	a.Esperar(200, P{Metodo: "GET", Ruta: "/v1/p/el-corte/libres?servicio=" + servicio + "&profesional=" + ana + "&fecha=2026-11-16"}, &l)
	libre := false
	for _, h := range l.Horarios {
		libre = libre || h.Hora == "10:30"
	}
	if !libre {
		t.Fatalf("las 10:30 del 16/11 tendrían que estar libres: %+v", l.Horarios)
	}
	// Un turno suelto no tiene siguientes.
	suelto := reservar(a, 201, servicio, ana, "2026-11-09 10:00", "7777")
	panel(409, "POST", "/v1/panel/turnos/"+suelto.ID+"/cancelar-siguientes", nil, nil)
	panel(404, "GET", "/v1/panel/series/00000000-0000-0000-0000-000000000000", nil, nil)
}
