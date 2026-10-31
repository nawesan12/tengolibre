package cifrado_test

import (
	"crypto/rand"
	"encoding/base64"
	"testing"

	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/cifrado"
)

func TestCifrarYDescifrarPorNegocio(t *testing.T) {
	k := make([]byte, 32)
	_, _ = rand.Read(k)
	c, err := cifrado.Nuevo(base64.StdEncoding.EncodeToString(k))
	if err != nil {
		t.Fatal(err)
	}
	d, _ := c.Cifrar("negocio-a", []byte("APP_USR-secreto"))
	if p, err := c.Descifrar("negocio-a", d); err != nil || string(p) != "APP_USR-secreto" {
		t.Fatalf("ida y vuelta: %q %v", p, err)
	}
	if _, err := c.Descifrar("negocio-b", d); err == nil {
		t.Fatal("se descifró con la clave de otro negocio")
	}
	d[len(d)-1] ^= 1
	if _, err := c.Descifrar("negocio-a", d); err == nil {
		t.Fatal("aceptó un dato tocado")
	}
	if _, err := cifrado.Nuevo("corta"); err == nil {
		t.Fatal("aceptó una clave inválida")
	}
}
