package config

import "testing"

func TestEntornoInvalidoNoArranca(t *testing.T) {
	t.Setenv("TENGOLIBRE_ENTORNO", "prod")
	if _, err := Leer(); err == nil {
		t.Fatal("un entorno desconocido tiene que fallar")
	}
}

func TestDesarrolloTieneBaseLocal(t *testing.T) {
	t.Setenv("TENGOLIBRE_ENTORNO", "desarrollo")
	c, err := Leer()
	if err != nil {
		t.Fatal(err)
	}
	if err := c.Exigir("BaseApp", "BaseMigrador", "BaseAdmin"); err != nil {
		t.Fatal(err)
	}
}

func TestProduccionExigeLaBase(t *testing.T) {
	t.Setenv("TENGOLIBRE_ENTORNO", "produccion")
	t.Setenv("TENGOLIBRE_BASE_DE_DATOS", "")
	c, err := Leer()
	if err != nil {
		t.Fatal(err)
	}
	if err := c.Exigir("BaseApp"); err == nil {
		t.Fatal("sin base en producción tiene que fallar")
	}
}

// Las compras de prueba de las tiendas se aceptan en desarrollo y no en producción, salvo que se
// prenda la variable (la revisión de Apple compra en sandbox).
func TestComprasDePruebaPorEntorno(t *testing.T) {
	t.Setenv("TENGOLIBRE_ENTORNO", "desarrollo")
	if c, _ := Leer(); !c.AppleSandbox || !c.GooglePruebas {
		t.Fatal("en desarrollo van prendidas")
	}
	t.Setenv("TENGOLIBRE_ENTORNO", "produccion")
	if c, _ := Leer(); c.AppleSandbox || c.GooglePruebas {
		t.Fatal("en producción van apagadas")
	}
	t.Setenv("TENGOLIBRE_APPLE_SANDBOX", "1")
	if c, _ := Leer(); !c.AppleSandbox || c.GooglePruebas {
		t.Fatal("la variable prende sólo la de Apple")
	}
}
