package turnos_test

import (
	"net/http"
	"strconv"
	"sync"
	"testing"
	"time"
)

// Varias personas reservan a la vez el mismo horario del mismo profesional: entra una sola. El
// profesional se bloquea (FOR UPDATE) adentro de la transacción de la reserva y la disponibilidad se
// recalcula ahí, así que la segunda ve el turno de la primera.
func TestReservasSimultaneas(t *testing.T) {
	a, _, servicio, ana, _ := armar(t)
	const n = 8
	var wg sync.WaitGroup
	estados := make(chan int, n)
	for i := range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r := a.Hacer(P{Metodo: "POST", Ruta: "/v1/p/el-corte/turnos", Cuerpo: map[string]any{
				"servicioId": servicio, "profesionalId": ana, "inicio": en("2026-11-02 09:30").Format(time.RFC3339),
				"cliente": map[string]string{"nombre": "Apurada", "telefono": "55" + strconv.Itoa(i)}}})
			estados <- r.Estado
		}()
	}
	wg.Wait()
	close(estados)
	ok, ocupado := 0, 0
	for e := range estados {
		switch e {
		case http.StatusCreated:
			ok++
		case http.StatusConflict:
			ocupado++
		default:
			t.Errorf("estado inesperado %d", e)
		}
	}
	if ok != 1 || ocupado != n-1 {
		t.Fatalf("entraron %d y rebotaron %d; quería 1 y %d", ok, ocupado, n-1)
	}
}
