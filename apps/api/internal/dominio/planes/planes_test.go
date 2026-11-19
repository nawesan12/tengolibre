package planes

import (
	"bytes"
	"encoding/json"
	"flag"
	"os"
	"testing"
)

var actualizar = flag.Bool("actualizar", false, "reescribe el catálogo de la web con el de planes.go")

// La landing (y lo que la web muestra sin pedirle a la API) lee apps/web/src/lib/planes-catalogo.json:
// tiene que decir exactamente lo mismo que planes.go. go test ./internal/dominio/planes -actualizar
// lo regenera.
func TestCatalogoDeLaWebIgualAPlanes(t *testing.T) {
	type plan struct {
		Clave       string   `json:"clave"`
		Nombre      string   `json:"nombre"`
		Precio      int64    `json:"precio"`
		PrecioAnual int64    `json:"precioAnual"`
		Destacado   bool     `json:"destacado"`
		Bajada      string   `json:"bajada"`
		Incluye     []string `json:"incluye"`
	}
	type catalogo struct {
		DiasDePrueba int    `json:"diasDePrueba"`
		MesesAnual   int    `json:"mesesAnual"`
		Planes       []plan `json:"planes"`
	}
	c := catalogo{DiasDePrueba: DiasDePrueba, MesesAnual: MesesAnual}
	for _, p := range Planes {
		c.Planes = append(c.Planes, plan{p.Clave, p.Nombre, p.Precio, p.PrecioAnual, p.Destacado, p.Bajada, p.Incluye})
	}
	quiero, _ := json.MarshalIndent(c, "", "\t")
	quiero = append(quiero, '\n')
	const ruta = "../../../../web/src/lib/planes-catalogo.json"
	if *actualizar {
		if err := os.WriteFile(ruta, quiero, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	hay, err := os.ReadFile(ruta)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(hay, quiero) {
		t.Fatalf("%s no coincide con planes.go: correr go test ./internal/dominio/planes -actualizar", ruta)
	}
}
