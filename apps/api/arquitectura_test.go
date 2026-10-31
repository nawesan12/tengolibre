package arquitectura_test

import (
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// El pool de Postgres sólo se usa desde internal/plataforma/db: el resto pasa por EnNegocio, que es
// lo que fija el negocio. Un import de pgxpool en otro lado es una puerta para saltearse eso.
func TestPgxpoolSoloEnDB(t *testing.T) {
	permitido := filepath.Join("internal", "plataforma", "db")
	err := filepath.WalkDir(".", func(ruta string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(ruta, ".go") {
			return err
		}
		f, err := parser.ParseFile(token.NewFileSet(), ruta, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		for _, imp := range f.Imports {
			p, _ := strconv.Unquote(imp.Path.Value)
			if p == "github.com/jackc/pgx/v5/pgxpool" && !strings.HasPrefix(filepath.Dir(ruta), permitido) {
				t.Errorf("%s importa pgxpool: usá db.Pool.EnNegocio", ruta)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
