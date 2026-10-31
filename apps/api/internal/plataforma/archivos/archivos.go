// Package archivos guarda las imágenes que sube un negocio (logo, portada, galería, fotos de
// servicios y profesionales) en el disco del VPS, una carpeta por negocio, y las sirve sin sesión
// con caché larga: el nombre es aleatorio y nunca se reutiliza, así que una URL no cambia de
// contenido. En el VPS no hay Cloudinary ni R2 (ver infra/vps/compose.yml: ./archivos:/datos/archivos).
package archivos

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"image"
	_ "image/jpeg" // DecodeConfig de JPEG
	_ "image/png"  // DecodeConfig de PNG
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/jackc/pgx/v5"
	_ "golang.org/x/image/webp" // DecodeConfig de WebP

	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/db"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/problema"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/sesion"
	"github.com/nawesan12/tengolibre/apps/api/internal/plataforma/web"
)

// Topes. Caddy corta el cuerpo en 10 MB; acá, cada imagen hasta 5 MB y 8000 px de lado (una más
// grande es una bomba de descompresión, no una foto de una peluquería).
const (
	MaxBytes      = 5 << 20
	MaxLado       = 8000
	MaxPorNegocio = 500
)

// Tipos aceptados, con su extensión.
var tipos = map[string]string{"image/jpeg": "jpg", "image/png": "png", "image/webp": "webp"}

// Archivo subido.
type Archivo struct {
	ID       string    `json:"id"`
	Nombre   string    `json:"nombre"`
	Tipo     string    `json:"tipo"`
	Bytes    int       `json:"bytes"`
	Ancho    int       `json:"ancho"`
	Alto     int       `json:"alto"`
	CreadoEn time.Time `json:"creadoEn"`
	URL      string    `json:"url"`
}

// Servicio de archivos.
type Servicio struct {
	Pool   *db.Pool
	Sesion *sesion.Servicio
	// Carpeta raíz (TENGOLIBRE_ARCHIVOS) y base pública de las URLs (TENGOLIBRE_ARCHIVOS_URL).
	Carpeta, URLBase string
}

var (
	reID     = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
	reNombre = regexp.MustCompile(`^[0-9a-f]{32}\.(jpg|png|webp)$`)
)

// URL pública de un archivo de un negocio.
func (s *Servicio) URL(negocio, nombre string) string {
	return s.URLBase + "/" + negocio + "/" + nombre
}

// Rutas del dominio.
func (s *Servicio) Rutas(mux *http.ServeMux) {
	g := func(h web.Manejador) http.HandlerFunc { return s.Sesion.Panel(sesion.Gestion, web.Envolver(h)) }
	// Subir: multipart con la imagen en el campo "archivo".
	mux.HandleFunc("POST /v1/panel/archivos", g(s.subir))
	mux.HandleFunc("GET /v1/panel/archivos", g(s.listar))
	mux.HandleFunc("DELETE /v1/panel/archivos/{id}", g(s.borrar))
	// Público: lo que ve la web del negocio.
	mux.HandleFunc("GET /archivos/{negocio}/{nombre}", s.servir)
}

// Detectar mira los primeros bytes (nunca el nombre ni el Content-Type que manda el navegador) y
// devuelve el tipo y las medidas si es una imagen aceptada que además se puede abrir.
func Detectar(cuerpo []byte) (tipo string, ancho, alto int, err error) {
	tipo = http.DetectContentType(cuerpo)
	if _, ok := tipos[tipo]; !ok {
		return "", 0, 0, problema.NoProcesable("tipo_no_aceptado", "Sólo se aceptan imágenes JPG, PNG o WebP.")
	}
	cfg, formato, err := image.DecodeConfig(bytes.NewReader(cuerpo))
	if err != nil || "image/"+formato != tipo {
		return "", 0, 0, problema.NoProcesable("imagen_rota", "La imagen no se pudo abrir.")
	}
	if cfg.Width <= 0 || cfg.Height <= 0 || cfg.Width > MaxLado || cfg.Height > MaxLado {
		return "", 0, 0, problema.NoProcesable("imagen_enorme", fmt.Sprintf("La imagen pasa de %d píxeles de lado.", MaxLado))
	}
	return tipo, cfg.Width, cfg.Height, nil
}

func muyGrande() error {
	return problema.NoProcesable("muy_grande", fmt.Sprintf("La imagen pasa de %d MB.", MaxBytes>>20))
}

func (s *Servicio) subir(w http.ResponseWriter, r *http.Request) error {
	// Un margen para el resto del multipart; lo que pase el tope corta la lectura.
	r.Body = http.MaxBytesReader(w, r.Body, MaxBytes+64<<10)
	f, _, err := r.FormFile("archivo")
	if err != nil {
		var grande *http.MaxBytesError
		if errors.As(err, &grande) {
			return muyGrande()
		}
		return problema.Invalido("Mandá la imagen en el campo «archivo» (multipart/form-data).")
	}
	defer f.Close()
	cuerpo, err := io.ReadAll(io.LimitReader(f, MaxBytes+1))
	if err != nil {
		var grande *http.MaxBytesError
		if errors.As(err, &grande) {
			return muyGrande()
		}
		return err
	}
	if len(cuerpo) > MaxBytes {
		return muyGrande()
	}
	if len(cuerpo) == 0 {
		return problema.Invalido("El archivo está vacío.")
	}
	tipo, ancho, alto, err := Detectar(cuerpo)
	if err != nil {
		return err
	}
	al := make([]byte, 16)
	if _, err := rand.Read(al); err != nil {
		return err
	}
	a := Archivo{Nombre: hex.EncodeToString(al) + "." + tipos[tipo], Tipo: tipo, Bytes: len(cuerpo), Ancho: ancho, Alto: alto}
	ctx := r.Context()
	negocio := sesion.De(ctx).NegocioID
	err = s.Pool.EnNegocio(ctx, negocio, func(tx pgx.Tx) error {
		var n int
		if err := tx.QueryRow(ctx, "SELECT count(*) FROM archivos").Scan(&n); err != nil {
			return err
		}
		if n >= MaxPorNegocio {
			return problema.Conflicto("demasiados_archivos", fmt.Sprintf("Llegaste al tope de %d imágenes: borrá alguna que no uses.", MaxPorNegocio))
		}
		if err := tx.QueryRow(ctx, `INSERT INTO archivos (negocio_id, nombre, tipo, bytes, ancho, alto)
			VALUES (app.negocio_actual(), $1, $2, $3, $4, $5) RETURNING id::text, creado_en`,
			a.Nombre, a.Tipo, a.Bytes, a.Ancho, a.Alto).Scan(&a.ID, &a.CreadoEn); err != nil {
			return err
		}
		// Al disco antes del commit: si falla la escritura, no queda una ficha sin archivo.
		return s.guardar(negocio, a.Nombre, cuerpo)
	})
	if err != nil {
		return err
	}
	a.URL = s.URL(negocio, a.Nombre)
	web.JSON(w, http.StatusCreated, a)
	return nil
}

// guardar escribe en un temporal de la misma carpeta y lo renombra: nunca se sirve medio archivo.
func (s *Servicio) guardar(negocio, nombre string, cuerpo []byte) error {
	dir := filepath.Join(s.Carpeta, negocio)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".subiendo-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(cuerpo); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Chmod(0o644); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), filepath.Join(dir, nombre))
}

func (s *Servicio) listar(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	negocio := sesion.De(ctx).NegocioID
	out := []Archivo{}
	err := s.Pool.EnNegocio(ctx, negocio, func(tx pgx.Tx) error {
		filas, err := tx.Query(ctx, "SELECT id::text, nombre, tipo, bytes, ancho, alto, creado_en FROM archivos ORDER BY creado_en DESC")
		if err != nil {
			return err
		}
		defer filas.Close()
		for filas.Next() {
			var a Archivo
			if err := filas.Scan(&a.ID, &a.Nombre, &a.Tipo, &a.Bytes, &a.Ancho, &a.Alto, &a.CreadoEn); err != nil {
				return err
			}
			a.URL = s.URL(negocio, a.Nombre)
			out = append(out, a)
		}
		return filas.Err()
	})
	if err != nil {
		return err
	}
	web.JSON(w, http.StatusOK, map[string]any{"archivos": out})
	return nil
}

func (s *Servicio) borrar(w http.ResponseWriter, r *http.Request) error {
	id := r.PathValue("id")
	if !reID.MatchString(id) {
		return problema.NoEncontrado("No existe o no es de este negocio.")
	}
	ctx := r.Context()
	negocio := sesion.De(ctx).NegocioID
	var nombre string
	err := s.Pool.EnNegocio(ctx, negocio, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, "DELETE FROM archivos WHERE id = $1 RETURNING nombre", id).Scan(&nombre)
	})
	if err != nil {
		return err
	}
	// Después del commit: si el disco falla, queda un huérfano en la carpeta, nunca una ficha rota.
	if err := os.Remove(filepath.Join(s.Carpeta, negocio, nombre)); err != nil && !errors.Is(err, os.ErrNotExist) {
		slog.Warn("archivos: no se pudo borrar del disco", "negocio", negocio, "nombre", nombre, "error", err)
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}

// servir manda el archivo tal cual del disco. Negocio y nombre se validan contra patrones fijos: no
// hay forma de armar una ruta fuera de la carpeta.
func (s *Servicio) servir(w http.ResponseWriter, r *http.Request) {
	negocio, nombre := r.PathValue("negocio"), r.PathValue("nombre")
	if !reID.MatchString(negocio) || !reNombre.MatchString(nombre) {
		problema.Escribir(w, problema.NoEncontrado("No existe."))
		return
	}
	f, err := os.Open(filepath.Join(s.Carpeta, negocio, nombre))
	if err != nil {
		problema.Escribir(w, problema.NoEncontrado("No existe."))
		return
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || info.IsDir() {
		problema.Escribir(w, problema.NoEncontrado("No existe."))
		return
	}
	h := w.Header()
	h.Set("Content-Type", map[string]string{"jpg": "image/jpeg", "png": "image/png", "webp": "image/webp"}[filepath.Ext(nombre)[1:]])
	h.Set("Cache-Control", "public, max-age=31536000, immutable")
	h.Set("X-Content-Type-Options", "nosniff")
	h.Set("Content-Security-Policy", "default-src 'none'; sandbox")
	http.ServeContent(w, r, "", info.ModTime(), f)
}
