package sesion

import (
	"context"
	"errors"
	"strings"

	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/clerk"
)

// Falso acepta tokens "dev:<clerk id>". Sólo para desarrollo local y tests: en producción la API
// no lo monta (ver cmd/api).
type Falso struct{}

func (Falso) Verificar(_ context.Context, token string) (string, error) {
	id, ok := strings.CutPrefix(token, "dev:")
	if !ok || !strings.HasPrefix(id, "user_") {
		return "", clerk.ErrToken
	}
	return id, nil
}

func (Falso) Usuario(_ context.Context, id string) (clerk.Usuario, error) {
	if !strings.HasPrefix(id, "user_") {
		return clerk.Usuario{}, errors.New("sesion: persona desconocida")
	}
	return clerk.Usuario{ID: id, Email: strings.TrimPrefix(id, "user_") + "@prueba.tengolibre.com.ar", Nombre: "Prueba"}, nil
}
