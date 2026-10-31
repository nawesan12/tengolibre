package clerk_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/clerk"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/clerk/clerkfalso"
)

var ctx = context.Background()

func nuevo(t *testing.T) (*clerk.Cliente, *clerkfalso.Servidor) {
	f := clerkfalso.Nuevo(t)
	c := &clerk.Cliente{Secreto: clerkfalso.Secreto, Emisor: clerkfalso.Emisor,
		Origenes: []string{"https://tulocalweb.com.ar"}, API: f.URL, HTTP: f.Client()}
	return c, f
}

func TestEmisorDeLaClavePublicable(t *testing.T) {
	for pk, want := range map[string]string{
		"pk_live_Y2xlcmsuZ3J1cG92aW1vLmNvbS5hciQ":                    "https://clerk.grupovimo.com.ar",
		"pk_test_cHJlY2lzZS1zd2FuLTgwMjIuY2xlcmsuYWNjb3VudHMuZGV2JA": "https://precise-swan-8022.clerk.accounts.dev",
	} {
		if got, err := clerk.EmisorDe(pk); err != nil || got != want {
			t.Errorf("%s: %q %v", pk, got, err)
		}
	}
	for _, mal := range []string{"", "sk_live_abc", "pk_live_", "pk_prod_Y2xlcmsuZ3J1cG92aW1vLmNvbS5hciQ"} {
		if _, err := clerk.EmisorDe(mal); err == nil {
			t.Errorf("%q: sin error", mal)
		}
	}
}

func TestVerificar(t *testing.T) {
	c, f := nuevo(t)
	bueno := f.Token(t, clerkfalso.Reclamos{Sub: "user_1", Azp: "https://tulocalweb.com.ar"})
	if sub, err := c.Verificar(ctx, bueno); err != nil || sub != "user_1" {
		t.Fatalf("%q %v", sub, err)
	}
	// La app de iPhone no tiene origen: sin azp, vale.
	if sub, err := c.Verificar(ctx, f.Token(t, clerkfalso.Reclamos{Sub: "user_2"})); err != nil || sub != "user_2" {
		t.Fatalf("sin azp: %q %v", sub, err)
	}
	vencido := time.Now().Add(-time.Minute)
	casos := map[string]string{
		"otro emisor":     f.Token(t, clerkfalso.Reclamos{Sub: "user_1", Iss: "https://otra.clerk.accounts.dev"}),
		"vencido":         f.Token(t, clerkfalso.Reclamos{Sub: "user_1", Exp: vencido}),
		"origen ajeno":    f.Token(t, clerkfalso.Reclamos{Sub: "user_1", Azp: "https://malo.example"}),
		"sub raro":        f.Token(t, clerkfalso.Reclamos{Sub: "../users"}),
		"firma de otra":   f.TokenAjeno(t, clerkfalso.Reclamos{Sub: "user_1"}),
		"kid desconocido": f.TokenKid(t, "otro", clerkfalso.Reclamos{Sub: "user_1"}),
		"basura":          "a.b.c",
		"alg none":        "eyJhbGciOiJub25lIiwia2lkIjoiazEifQ." + strings.Split(bueno, ".")[1] + ".",
	}
	for nombre, tok := range casos {
		if _, err := c.Verificar(ctx, tok); !errors.Is(err, clerk.ErrToken) {
			t.Errorf("%s: err = %v", nombre, err)
		}
	}
	// El JWKS se bajó una sola vez (el kid desconocido no lo vuelve a pedir dentro del minuto).
	if f.PedidosJWKS() != 1 {
		t.Fatalf("jwks pedido %d veces", f.PedidosJWKS())
	}
}

func TestUsuarioTomaElEmailVerificado(t *testing.T) {
	c, f := nuevo(t)
	f.Usuarios["user_1"] = clerkfalso.Usuario{Nombre: "Lucía", Apellido: "Fernández", Email: "lucia@gmail.com",
		Verificado: true, Foto: "https://img.clerk.com/abc"}
	f.Usuarios["user_2"] = clerkfalso.Usuario{Nombre: "Ana", Email: "ana@gmail.com"}
	u, err := c.Usuario(ctx, "user_1")
	if err != nil || u.Email != "lucia@gmail.com" || u.Nombre != "Lucía" || u.Foto != "https://img.clerk.com/abc" {
		t.Fatalf("%+v %v", u, err)
	}
	if u, err := c.Usuario(ctx, "user_2"); err != nil || u.Email != "" {
		t.Fatalf("sin verificar: %+v %v", u, err)
	}
	if _, err := c.Usuario(ctx, "user_nadie"); err == nil {
		t.Fatal("inexistente sin error")
	}
	if !f.SoloConSecreto() {
		t.Fatal("algún pedido a la Backend API sin la clave secreta")
	}
}

func TestDesdeLasClaves(t *testing.T) {
	c, err := clerk.Nuevo("sk_test_x", "pk_test_cHJlY2lzZS1zd2FuLTgwMjIuY2xlcmsuYWNjb3VudHMuZGV2JA", nil)
	if err != nil || c.Emisor != "https://precise-swan-8022.clerk.accounts.dev" {
		t.Fatalf("%+v %v", c, err)
	}
	if _, err := clerk.Nuevo("pk_test_x", "pk_test_cHJlY2lzZS1zd2FuLTgwMjIuY2xlcmsuYWNjb3VudHMuZGV2JA", nil); err == nil {
		t.Fatal("secreta mal formada sin error")
	}
}
