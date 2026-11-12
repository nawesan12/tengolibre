package cuenta_test

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/cuenta"
	"github.com/nawesan12/tengolibre/apps/api/internal/dominio/negocios"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/db/pruebadb"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/prueba"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/servidor"
)

type P = prueba.Pedido

const (
	ana   = "user_cuentaana001"
	berta = "user_cuentaberta01"
	carla = "user_cuentacarla01"
)

// conToken manda un pedido con la sesión propia.
func conToken(a *prueba.API, metodo, ruta, token string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(metodo, ruta, strings.NewReader(""))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	a.Handler.ServeHTTP(rec, req)
	return rec
}

func ingresar(t *testing.T, a *prueba.API, quien string) string {
	t.Helper()
	var r struct {
		Token string `json:"token"`
	}
	a.Esperar(201, P{Metodo: "POST", Ruta: "/v1/auth/clerk", Cuerpo: map[string]string{"token": "dev:" + quien}}, &r)
	return r.Token
}

// admin corre SQL como superusuario (ve todo, sin RLS).
func admin(t *testing.T, a *prueba.API, fn func(conn *pgx.Conn)) {
	t.Helper()
	conn, err := pgx.Connect(context.Background(), a.Bases.AdminURL)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close(context.Background())
	fn(conn)
}

func uno[T any](t *testing.T, conn *pgx.Conn, sql string, args ...any) T {
	t.Helper()
	var v T
	if err := conn.QueryRow(context.Background(), sql, args...).Scan(&v); err != nil {
		t.Fatalf("%s: %v", sql, err)
	}
	return v
}

func exec(t *testing.T, conn *pgx.Conn, sql string, args ...any) {
	t.Helper()
	if _, err := conn.Exec(context.Background(), sql, args...); err != nil {
		t.Fatalf("%s: %v", sql, err)
	}
}

func armar(t *testing.T) (*prueba.API, *[]string) {
	cancelados := &[]string{}
	a := prueba.Nueva(t, time.Now(), func(a *prueba.API) []servidor.Dominio {
		return []servidor.Dominio{a.Sesion, &negocios.Servicio{Pool: a.Bases.App, Sesion: a.Sesion},
			&cuenta.Servicio{Pool: a.Bases.App, Sesion: a.Sesion, CancelarSuscripcion: func(_ context.Context, n string) error {
				*cancelados = append(*cancelados, n)
				return nil
			}}}
	})
	return a, cancelados
}

func TestBajaYPurga(t *testing.T) {
	a, cancelados := armar(t)
	ctx := context.Background()
	// Ana es la única dueña de «sola»; dueña junto con Carla de «compartida»; encargada en «ajena»
	// (de Berta), donde además es profesional, escribió una nota y cobró en la caja.
	sola := pruebadb.CrearNegocio(t, a.Bases.App, "sola", ana)
	compartida := pruebadb.CrearNegocio(t, a.Bases.App, "compartida", ana)
	ajena := pruebadb.CrearNegocio(t, a.Bases.App, "ajena", berta)
	var miembroAjena string
	admin(t, a, func(c *pgx.Conn) {
		exec(t, c, "INSERT INTO miembros (negocio_id, clerk_id, email, nombre, rol) VALUES ($1, $2, 'carla@x.com', 'Carla', 'duenio')", compartida, carla)
		miembroAjena = uno[string](t, c, `INSERT INTO miembros (negocio_id, clerk_id, email, nombre, rol)
			VALUES ($1, $2, 'ana@x.com', 'Ana', 'encargado') RETURNING id::text`, ajena, ana)
		exec(t, c, `INSERT INTO profesionales (negocio_id, miembro_id, nombre, email, telefono, foto)
			VALUES ($1, $2, 'Ana P', 'ana@x.com', '1155550000', 'https://x/ana.jpg')`, ajena, miembroAjena)
		cli := uno[string](t, c, "INSERT INTO clientes (negocio_id, nombre) VALUES ($1, 'Clienta') RETURNING id::text", ajena)
		exec(t, c, "INSERT INTO notas_cliente (negocio_id, cliente_id, miembro_id, autor, texto) VALUES ($1, $2, $3, 'Ana', 'Le gusta corto')",
			ajena, cli, miembroAjena)
		exec(t, c, "INSERT INTO auditoria (negocio_id, clerk_id, accion, entidad) VALUES ($1, $2, 'algo', 'cosa')", ajena, ana)
		exec(t, c, "INSERT INTO push_preferencias (negocio_id, clerk_id) VALUES ($1, $2)", ajena, ana)
	})

	token := ingresar(t, a, ana)
	otraSesion := ingresar(t, a, ana)
	admin(t, a, func(c *pgx.Conn) {
		exec(t, c, `INSERT INTO auth.dispositivos (clerk_id, sesion_hash, plataforma, token)
			SELECT clerk_id, hash, 'android', 'token_de_fcm_de_prueba_123456' FROM auth.sesiones WHERE clerk_id = $1 LIMIT 1`, ana)
	})

	// Sin sesión, no.
	a.Esperar(401, P{Metodo: "DELETE", Ruta: "/v1/auth/cuenta"}, nil)
	if rec := conToken(a, "DELETE", "/v1/auth/cuenta", token); rec.Code != 204 {
		t.Fatalf("baja: %d %s", rec.Code, rec.Body)
	}
	if len(*cancelados) != 1 || (*cancelados)[0] != sola {
		t.Fatalf("suscripciones canceladas: %v (quería sólo %s)", *cancelados, sola)
	}

	// Las dos sesiones murieron; el ingreso responde cuenta_borrada, y un token de Clerk suelto también.
	for _, tk := range []string{token, otraSesion} {
		if rec := conToken(a, "GET", "/v1/yo", tk); rec.Code != 401 {
			t.Fatalf("sesión después de la baja: %d", rec.Code)
		}
	}
	r := a.Esperar(409, P{Metodo: "POST", Ruta: "/v1/auth/clerk", Cuerpo: map[string]string{"token": "dev:" + ana}}, nil)
	if !strings.Contains(string(r.Cuerpo), `"code":"cuenta_borrada"`) || !strings.Contains(string(r.Cuerpo), "hola@tengolibre.com.ar") {
		t.Fatalf("ingreso de una cuenta borrada: %s", r.Cuerpo)
	}
	a.Esperar(409, P{Metodo: "GET", Ruta: "/v1/yo", Quien: ana}, nil)
	// Las otras personas siguen como siempre.
	ingresar(t, a, berta)

	admin(t, a, func(c *pgx.Conn) {
		if uno[bool](t, c, "SELECT activo FROM negocios WHERE id = $1", sola) {
			t.Error("el negocio donde era la única dueña sigue activo")
		}
		if !uno[bool](t, c, "SELECT activo FROM negocios WHERE id = $1", compartida) || !uno[bool](t, c, "SELECT activo FROM negocios WHERE id = $1", ajena) {
			t.Error("se cerró un negocio que tenía otra dueña")
		}
		if n := uno[int](t, c, "SELECT count(*) FROM miembros WHERE clerk_id = $1 AND activo", ana); n != 0 {
			t.Errorf("membresías activas después de la baja: %d", n)
		}
		if n := uno[int](t, c, "SELECT count(*) FROM auth.dispositivos WHERE clerk_id = $1", ana); n != 0 {
			t.Errorf("dispositivos después de la baja: %d", n)
		}
		if e := uno[string](t, c, "SELECT email FROM auth.cuentas_borradas WHERE clerk_id = $1", ana); !strings.Contains(e, "@") {
			t.Errorf("la baja no guardó el email: %q", e)
		}
	})

	// Antes de los 30 días, la purga no toca nada.
	if n, err := cuenta.Purgar(ctx, a.Bases.App, time.Now().AddDate(0, 0, 29)); err != nil || n != 0 {
		t.Fatalf("purga antes de tiempo: %d %v", n, err)
	}
	if n, err := cuenta.Purgar(ctx, a.Bases.App, time.Now().AddDate(0, 0, 31)); err != nil || n != 1 {
		t.Fatalf("purga: %d %v", n, err)
	}
	admin(t, a, func(c *pgx.Conn) {
		if n := uno[int](t, c, "SELECT count(*) FROM miembros WHERE clerk_id = $1", ana); n != 0 {
			t.Errorf("membresías después de la purga: %d", n)
		}
		if n := uno[int](t, c, "SELECT count(*) FROM auditoria WHERE clerk_id = $1", ana); n != 0 {
			t.Errorf("auditoría con su clerk_id: %d", n)
		}
		if n := uno[int](t, c, "SELECT count(*) FROM push_preferencias WHERE clerk_id = $1", ana); n != 0 {
			t.Errorf("preferencias de push: %d", n)
		}
		if n := uno[int](t, c, "SELECT count(*) FROM notas_cliente WHERE autor = 'Ana' OR miembro_id IS NOT NULL"); n != 0 {
			t.Errorf("notas con autora: %d", n)
		}
		if n := uno[int](t, c, "SELECT count(*) FROM notas_cliente"); n != 1 {
			t.Errorf("la nota del negocio se borró: %d", n)
		}
		var nombre, email, tel, foto string
		var miembro *string
		if err := c.QueryRow(ctx, "SELECT nombre, email, telefono, foto, miembro_id::text FROM profesionales WHERE negocio_id = $1", ajena).
			Scan(&nombre, &email, &tel, &foto, &miembro); err != nil {
			t.Fatal(err)
		}
		if nombre != "Ana P" || email != "" || tel != "" || foto != "" || miembro != nil {
			t.Errorf("profesional después de la purga: %q %q %q %q %v", nombre, email, tel, foto, miembro)
		}
		var purgada *time.Time
		var e string
		if err := c.QueryRow(ctx, "SELECT purgada_en, email FROM auth.cuentas_borradas WHERE clerk_id = $1", ana).Scan(&purgada, &e); err != nil {
			t.Fatal(err)
		}
		if purgada == nil || e != "" {
			t.Errorf("la baja no quedó purgada: %v %q", purgada, e)
		}
	})
	// Purgada, vuelve a entrar como cuenta nueva (sin negocios).
	nuevo := ingresar(t, a, ana)
	if rec := conToken(a, "GET", "/v1/yo", nuevo); rec.Code != 200 || strings.Contains(rec.Body.String(), sola) {
		t.Fatalf("cuenta nueva: %d %s", rec.Code, rec.Body)
	}
	// Y la puede volver a borrar.
	if rec := conToken(a, "DELETE", "/v1/auth/cuenta", nuevo); rec.Code != 204 {
		t.Fatalf("segunda baja: %d %s", rec.Code, rec.Body)
	}
	a.Esperar(409, P{Metodo: "POST", Ruta: "/v1/auth/clerk", Cuerpo: map[string]string{"token": "dev:" + ana}}, nil)
}

func TestRecuperarCuenta(t *testing.T) {
	a, _ := armar(t)
	sola := pruebadb.CrearNegocio(t, a.Bases.App, "recupero", berta)
	token := ingresar(t, a, berta)
	if rec := conToken(a, "DELETE", "/v1/auth/cuenta", token); rec.Code != 204 {
		t.Fatalf("baja: %d %s", rec.Code, rec.Body)
	}
	// La app no puede recuperarla: sólo el migrador (tengolibrectl).
	err := a.Bases.App.EnTx(context.Background(), func(tx pgx.Tx) error {
		_, err := tx.Exec(context.Background(), "SELECT app.recuperar_cuenta($1)", berta)
		return err
	})
	if err == nil {
		t.Fatal("la app pudo recuperar una cuenta")
	}
	admin(t, a, func(c *pgx.Conn) {
		email := uno[string](t, c, "SELECT email FROM auth.cuentas_borradas WHERE clerk_id = $1", berta)
		if id := uno[*string](t, c, "SELECT app.recuperar_cuenta($1)", strings.ToUpper(email)); id == nil || *id != berta {
			t.Fatalf("recuperar por email: %v", id)
		}
		if !uno[bool](t, c, "SELECT activo FROM negocios WHERE id = $1", sola) {
			t.Error("el negocio no volvió")
		}
		if id := uno[*string](t, c, "SELECT app.recuperar_cuenta($1)", berta); id != nil {
			t.Errorf("recuperó dos veces: %v", *id)
		}
	})
	// Vuelve a entrar con su negocio.
	tk := ingresar(t, a, berta)
	if rec := conToken(a, "GET", "/v1/yo", tk); rec.Code != 200 || !strings.Contains(rec.Body.String(), sola) {
		t.Fatalf("después de recuperar: %d %s", rec.Code, rec.Body)
	}
}
