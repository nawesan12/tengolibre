package suscripcion

// El botón de arrepentimiento (Res. SC 424/2020): lo contratado en línea se puede revocar dentro de
// los 10 días, sin entrar a la cuenta. El pedido queda guardado con un código para seguirlo; lo
// atiende la plataforma (cancela y devuelve lo cobrado).

import (
	"crypto/rand"
	"net/http"
	"net/mail"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"

	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/problema"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/tope"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/web"
)

const letrasCodigo = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"

func codigoArrepentimiento() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	for i := range b {
		b[i] = letrasCodigo[int(b[i])%len(letrasCodigo)]
	}
	return string(b)
}

var topeArrepentimiento tope.Contador

// arrepentirse: POST /v1/arrepentimiento {"email", "negocio", "detalle"} → {"codigo"}.
func (s *Servicio) arrepentirse(w http.ResponseWriter, r *http.Request) error {
	var in struct {
		Email   string `json:"email"`
		Negocio string `json:"negocio"`
		Detalle string `json:"detalle"`
	}
	if err := web.Leer(w, r, &in); err != nil {
		return err
	}
	if err := topeArrepentimiento.PermitirPublico(r, "arrepentimiento", "", tope.Publico{PorIP: 5, PorNegocio: 200, Ventana: time.Hour}, s.ahora()); err != nil {
		return err
	}
	in.Email, in.Negocio, in.Detalle = strings.TrimSpace(in.Email), strings.TrimSpace(in.Negocio), strings.TrimSpace(in.Detalle)
	if a, err := mail.ParseAddress(in.Email); err != nil || a.Address != in.Email || len(in.Email) > 254 {
		return problema.Invalido("Revisá el mail.", problema.Campo{Ruta: "email", Mensaje: "inválido"})
	}
	if utf8.RuneCountInString(in.Negocio) > 120 || utf8.RuneCountInString(in.Detalle) > 1000 {
		return problema.Invalido("El texto es muy largo.")
	}
	codigo := codigoArrepentimiento()
	err := s.Pool.EnTx(r.Context(), func(tx pgx.Tx) error {
		_, err := tx.Exec(r.Context(), `INSERT INTO auth.arrepentimientos (codigo, email, negocio, detalle, creado_en) VALUES ($1, $2, $3, $4, $5)`,
			codigo, in.Email, in.Negocio, in.Detalle, s.ahora())
		return err
	})
	if err != nil {
		return err
	}
	s.log().Warn("suscripcion: pedido de arrepentimiento (revisar en auth.arrepentimientos)", "codigo", codigo)
	web.JSON(w, http.StatusCreated, map[string]string{"codigo": codigo})
	return nil
}
