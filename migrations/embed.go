// Package migrations embebe las migraciones SQL de goose en el binario.
package migrations

import "embed"

// FS contiene los archivos de migracion, para correrlos con goose desde el binario.
//
//go:embed *.sql
var FS embed.FS
