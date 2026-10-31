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
