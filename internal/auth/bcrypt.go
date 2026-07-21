// Package auth provee las implementaciones de infraestructura de los ports de autenticacion
// del dominio: PasswordHasher (bcrypt) y TokenService (JWT).
package auth

import (
	"golang.org/x/crypto/bcrypt"

	"github.com/nicolas2601/go-graphql-orders-api/internal/domain"
)

// BcryptHasher implementa domain.PasswordHasher con bcrypt.
type BcryptHasher struct {
	cost int
}

// NewBcryptHasher crea un hasher con el cost dado; si esta fuera de rango usa el cost por defecto.
func NewBcryptHasher(cost int) BcryptHasher {
	if cost < bcrypt.MinCost || cost > bcrypt.MaxCost {
		cost = bcrypt.DefaultCost
	}
	return BcryptHasher{cost: cost}
}

var _ domain.PasswordHasher = BcryptHasher{}

// Hash devuelve el hash bcrypt de la contrasena.
func (h BcryptHasher) Hash(password string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(password), h.cost)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// Compare verifica una contrasena contra su hash. Devuelve error si no coinciden.
func (h BcryptHasher) Compare(hash, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}
