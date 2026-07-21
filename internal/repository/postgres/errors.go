package postgres

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
)

// Codigos SQLSTATE de PostgreSQL relevantes.
const (
	uniqueViolationCode       = "23505" // violacion de unique/primary key
	invalidTextRepresentation = "22P02" // ej. un id que no es un UUID valido
)

// isUniqueViolation indica si el error es una violacion de restriccion unique.
func isUniqueViolation(err error) bool {
	return hasPgCode(err, uniqueViolationCode)
}

// isMalformedID indica si el error es por un id con formato invalido para una columna UUID.
// Se trata como "no encontrado" (igual que un id inexistente), no como error interno.
func isMalformedID(err error) bool {
	return hasPgCode(err, invalidTextRepresentation)
}

func hasPgCode(err error, code string) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == code
}
