// Package db embebe las migraciones del esquema (goose, SQL plano).
package db

import "embed"

// Migraciones son los archivos de db/migraciones, embebidos en el binario.
//
//go:embed migraciones/*.sql
var Migraciones embed.FS
