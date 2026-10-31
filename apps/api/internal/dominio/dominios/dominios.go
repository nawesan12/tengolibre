// Package dominios: el dominio propio de un negocio (turnos.mibarberia.com o mibarberia.com) para
// su página. A diferencia de Jiku (que lo registraba en Vercel), acá lo sirve Caddy en el VPS: la
// dueña carga un TXT con un token (prueba que el dominio es suyo) y un CNAME al destino de
// TengoLibre (o, si es un dominio raíz, un A a la IP del VPS); la API lo verifica por DNS, y
// `tengolibrectl dominios-caddy` arma los bloques de Caddy de los verificados (el certificado lo
// saca Caddy con Let's Encrypt).
package dominios

import (
	"context"
	"crypto/rand"
	"errors"
	"net"
	"net/http"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/db"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/problema"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/sesion"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/web"
)

// Resolver de DNS. *net.Resolver lo cumple; los tests pasan uno falso.
type Resolver interface {
	LookupTXT(ctx context.Context, nombre string) ([]string, error)
	LookupCNAME(ctx context.Context, host string) (string, error)
	LookupHost(ctx context.Context, host string) ([]string, error)
}

// Servicio de dominios propios.
type Servicio struct {
	Pool   *db.Pool
	Sesion *sesion.Servicio
	// Resolver de DNS (nil: net.DefaultResolver).
	Resolver Resolver
	// Destino del CNAME (TENGOLIBRE_DOMINIOS_DESTINO) e IP del VPS para el A (TENGOLIBRE_VPS_IP).
	Destino, IP string
	// Propio: el dominio de TengoLibre, que no se le puede dar a un negocio (por defecto tengolibre.com.ar).
	Propio string
}

// Dominio y lo que falta para que ande.
type Dominio struct {
	Nombre       string     `json:"nombre"`
	Estado       string     `json:"estado"` // pendiente | verificado | error
	Detalle      string     `json:"detalle"`
	CreadoEn     time.Time  `json:"creadoEn"`
	ChequeadoEn  *time.Time `json:"chequeadoEn"`
	VerificadoEn *time.Time `json:"verificadoEn"`
	// Los registros a cargar en el DNS.
	Registros []Registro `json:"registros"`
}

// Registro de DNS a cargar.
type Registro struct {
	Tipo   string `json:"tipo"` // TXT | CNAME | A
	Nombre string `json:"nombre"`
	Valor  string `json:"valor"`
	// Para el dominio raíz se puede usar A en vez de CNAME (no todos los DNS aceptan CNAME en la raíz).
	Alternativa bool `json:"alternativa,omitempty"`
}

// Rutas del dominio.
func (s *Servicio) Rutas(mux *http.ServeMux) {
	mux.HandleFunc("GET /v1/panel/dominio", s.Sesion.Panel(sesion.Gestion, web.Envolver(s.leer)))
	mux.HandleFunc("PUT /v1/panel/dominio", s.Sesion.Panel([]string{sesion.Duenio}, web.Envolver(s.poner)))
	mux.HandleFunc("POST /v1/panel/dominio/verificar", s.Sesion.Panel(sesion.Gestion, web.Envolver(s.verificar)))
	mux.HandleFunc("DELETE /v1/panel/dominio", s.Sesion.Panel([]string{sesion.Duenio}, web.Envolver(s.quitar)))
	// Público: el front (o Caddy) pregunta de qué negocio es un host.
	mux.HandleFunc("GET /v1/host/{host}", web.Envolver(s.porHost))
}

var reDominio = regexp.MustCompile(`^([a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?\.)+[a-z]{2,63}$`)

// Normalizar deja el dominio como se guarda (sin esquema, ruta, puerto ni punto final, en
// minúsculas) o devuelve el problema. Un dominio con esquema o ruta nunca coincidiría con un host.
func Normalizar(entrada, propio string) (string, error) {
	d := strings.ToLower(strings.TrimSpace(entrada))
	d = strings.TrimPrefix(strings.TrimPrefix(d, "https://"), "http://")
	if i := strings.IndexAny(d, "/?#"); i >= 0 {
		d = d[:i]
	}
	if h, _, err := net.SplitHostPort(d); err == nil {
		d = h
	}
	d = strings.TrimSuffix(d, ".")
	if len(d) > 253 || !reDominio.MatchString(d) {
		return "", problema.Invalido("Ese dominio no parece válido. Ejemplo: turnos.mibarberia.com",
			problema.Campo{Ruta: "dominio", Mensaje: "formato inválido"})
	}
	if d == propio || strings.HasSuffix(d, "."+propio) {
		return "", problema.Invalido("Ese dominio no se puede usar.", problema.Campo{Ruta: "dominio", Mensaje: "es de TengoLibre"})
	}
	return d, nil
}

func (s *Servicio) propio() string {
	if s.Propio == "" {
		return "tengolibre.com.ar"
	}
	return s.Propio
}

func (s *Servicio) resolver() Resolver {
	if s.Resolver == nil {
		return net.DefaultResolver
	}
	return s.Resolver
}

// NombreTXT es donde va el token: un subdominio propio, así no choca con otros TXT del dominio.
func NombreTXT(dominio string) string { return "_tengolibre." + dominio }

// ValorTXT es lo que tiene que decir el TXT.
func ValorTXT(token string) string { return "tengolibre-verificacion=" + token }

func (s *Servicio) registros(d *Dominio, token string) {
	d.Registros = []Registro{
		{Tipo: "TXT", Nombre: NombreTXT(d.Nombre), Valor: ValorTXT(token)},
		{Tipo: "CNAME", Nombre: d.Nombre, Valor: s.Destino},
	}
	// Un dominio de dos partes (o un .com.ar de tres) es la raíz: ahí puede ir un A.
	if raiz(d.Nombre) {
		d.Registros = append(d.Registros, Registro{Tipo: "A", Nombre: d.Nombre, Valor: s.IP, Alternativa: true})
	}
}

// raiz dice si el dominio parece la raíz de su zona (mibarberia.com, mibarberia.com.ar).
func raiz(d string) bool {
	partes := strings.Split(d, ".")
	if len(partes) == 2 {
		return true
	}
	segundo := map[string]bool{"com": true, "net": true, "org": true, "gob": true, "edu": true, "tur": true}
	return len(partes) == 3 && len(partes[2]) == 2 && segundo[partes[1]]
}

const columnas = "nombre, estado, detalle, creado_en, chequeado_en, verificado_en, token"

func leer(ctx context.Context, tx pgx.Tx) (Dominio, string, error) {
	var d Dominio
	var token string
	err := tx.QueryRow(ctx, "SELECT "+columnas+" FROM dominios WHERE negocio_id = app.negocio_actual()").Scan(
		&d.Nombre, &d.Estado, &d.Detalle, &d.CreadoEn, &d.ChequeadoEn, &d.VerificadoEn, &token)
	return d, token, err
}

func (s *Servicio) responder(w http.ResponseWriter, estado int, d *Dominio, token string) {
	if d == nil {
		web.JSON(w, estado, map[string]any{"dominio": nil, "destino": s.Destino, "ip": s.IP})
		return
	}
	s.registros(d, token)
	web.JSON(w, estado, map[string]any{"dominio": d, "destino": s.Destino, "ip": s.IP})
}

func (s *Servicio) leer(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	var d Dominio
	var token string
	err := s.Pool.EnNegocio(ctx, sesion.De(ctx).NegocioID, func(tx pgx.Tx) (err error) {
		d, token, err = leer(ctx, tx)
		return err
	})
	if errors.Is(err, pgx.ErrNoRows) {
		s.responder(w, http.StatusOK, nil, "")
		return nil
	}
	if err != nil {
		return err
	}
	s.responder(w, http.StatusOK, &d, token)
	return nil
}

func nuevoToken() string {
	const letras = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	for i := range b {
		b[i] = letras[int(b[i])%len(letras)]
	}
	return string(b)
}

// poner anota el dominio (o lo cambia por otro: el anterior deja de andar). Queda pendiente hasta
// verificarlo.
func (s *Servicio) poner(w http.ResponseWriter, r *http.Request) error {
	var in struct {
		Dominio string `json:"dominio"`
	}
	if err := web.Leer(w, r, &in); err != nil {
		return err
	}
	nombre, err := Normalizar(in.Dominio, s.propio())
	if err != nil {
		return err
	}
	ctx := r.Context()
	var d Dominio
	var token string
	err = s.Pool.EnNegocio(ctx, sesion.De(ctx).NegocioID, func(tx pgx.Tx) error {
		actual, tok, err := leer(ctx, tx)
		if err == nil && actual.Nombre == nombre {
			d, token = actual, tok // el mismo: no se pierde el token ya cargado en el DNS
			return nil
		}
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		if _, err := tx.Exec(ctx, "DELETE FROM dominios WHERE negocio_id = app.negocio_actual()"); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, "INSERT INTO dominios (negocio_id, nombre, token) VALUES (app.negocio_actual(), $1, $2)", nombre, nuevoToken()); err != nil {
			return err
		}
		d, token, err = leer(ctx, tx)
		return err
	})
	if err != nil {
		return err
	}
	s.responder(w, http.StatusOK, &d, token)
	return nil
}

// Chequear mira el DNS: el TXT con el token y que el dominio apunte a TengoLibre (CNAME al destino,
// o A a la IP del VPS). Devuelve "" si está todo bien o qué falta, en criollo.
func Chequear(ctx context.Context, res Resolver, dominio, token, destino, ip string) string {
	txts, err := res.LookupTXT(ctx, NombreTXT(dominio))
	if err != nil || !slices.Contains(txts, ValorTXT(token)) {
		return "Falta el registro TXT " + NombreTXT(dominio) + " con el valor " + ValorTXT(token) + " (puede tardar en propagarse)."
	}
	destino = strings.TrimSuffix(strings.ToLower(destino), ".")
	if cname, err := res.LookupCNAME(ctx, dominio); err == nil && strings.TrimSuffix(strings.ToLower(cname), ".") == destino {
		return ""
	}
	if ips, err := res.LookupHost(ctx, dominio); err == nil && slices.Contains(ips, ip) {
		return ""
	}
	return "El dominio todavía no apunta a TengoLibre: cargá un CNAME a " + destino + " (o, si es el dominio raíz, un A a " + ip + ")."
}

func (s *Servicio) verificar(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	negocio := sesion.De(ctx).NegocioID
	var d Dominio
	var token string
	err := s.Pool.EnNegocio(ctx, negocio, func(tx pgx.Tx) (err error) {
		d, token, err = leer(ctx, tx)
		return err
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return problema.NoEncontrado("Todavía no cargaste un dominio.")
	}
	if err != nil {
		return err
	}
	// El DNS fuera de la transacción: puede tardar segundos.
	dctx, cancelar := context.WithTimeout(ctx, 10*time.Second)
	falta := Chequear(dctx, s.resolver(), d.Nombre, token, s.Destino, s.IP)
	cancelar()
	err = s.Pool.EnNegocio(ctx, negocio, func(tx pgx.Tx) error {
		estado := "verificado"
		if falta != "" {
			estado = "error"
		}
		_, err := tx.Exec(ctx, `UPDATE dominios SET estado = $1, detalle = $2, chequeado_en = now(),
			verificado_en = CASE WHEN $1 = 'verificado' THEN coalesce(verificado_en, now()) ELSE NULL END
			WHERE negocio_id = app.negocio_actual() AND nombre = $3`, estado, falta, d.Nombre)
		var pg *pgconn.PgError
		if errors.As(err, &pg) && pg.Code == "23505" {
			return problema.Conflicto("dominio_ocupado", "Ese dominio ya está conectado a otro negocio.")
		}
		if err != nil {
			return err
		}
		d, token, err = leer(ctx, tx)
		return err
	})
	if err != nil {
		return err
	}
	s.responder(w, http.StatusOK, &d, token)
	return nil
}

func (s *Servicio) quitar(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	err := s.Pool.EnNegocio(ctx, sesion.De(ctx).NegocioID, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, "DELETE FROM dominios WHERE negocio_id = app.negocio_actual()")
		return err
	})
	if err != nil {
		return err
	}
	s.responder(w, http.StatusOK, nil, "")
	return nil
}

// SlugPorHost devuelve el slug del negocio de un host (dominio propio verificado de un negocio
// activo; www.<dominio> es el mismo), o "" si no es de nadie.
func SlugPorHost(ctx context.Context, pool *db.Pool, host string) (string, error) {
	host = strings.TrimSuffix(strings.ToLower(strings.TrimSpace(host)), ".")
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	if !reDominio.MatchString(host) {
		return "", nil
	}
	var slug *string
	err := pool.EnTx(ctx, func(tx pgx.Tx) error {
		if err := tx.QueryRow(ctx, "SELECT app.negocio_por_host($1)", host).Scan(&slug); err != nil {
			return err
		}
		if sin, ok := strings.CutPrefix(host, "www."); slug == nil && ok {
			return tx.QueryRow(ctx, "SELECT app.negocio_por_host($1)", sin).Scan(&slug)
		}
		return nil
	})
	if err != nil || slug == nil {
		return "", err
	}
	return *slug, nil
}

func (s *Servicio) porHost(w http.ResponseWriter, r *http.Request) error {
	slug, err := SlugPorHost(r.Context(), s.Pool, r.PathValue("host"))
	if err != nil {
		return err
	}
	if slug == "" {
		return problema.NoEncontrado("Ese dominio no es de ningún negocio.")
	}
	web.JSON(w, http.StatusOK, map[string]string{"slug": slug})
	return nil
}
