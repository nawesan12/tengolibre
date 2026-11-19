package push

import "testing"

// Sin lista de espera en el plan, la cancelación no ofrece «Avisar a la lista».
func TestCategoriaSinLista(t *testing.T) {
	if c := Categoria(Aviso{Tipo: "cancelacion"}); c != "TL_CANCELO" {
		t.Fatal(c)
	}
	if c := Categoria(Aviso{Tipo: "cancelacion", SinLista: true}); c != "TL_CANCELO_SIN_LISTA" {
		t.Fatal(c)
	}
	if c := Categoria(Aviso{Tipo: "espera", SinLista: true}); c != "" {
		t.Fatal(c)
	}
	if d := DatosFCM(Aviso{Tipo: "cancelacion", SinLista: true}); d["categoria"] != "TL_CANCELO_SIN_LISTA" {
		t.Fatal(d)
	}
}
