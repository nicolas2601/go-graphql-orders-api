package domain

import (
	"net/mail"
	"strings"
	"time"
	"unicode"
)

const (
	minPasswordLength = 8
	// maxPasswordLength es el limite de bcrypt (72 bytes): pasado ese punto trunca/error.
	// Se valida en el dominio para no filtrar un error de infraestructura ni gastar CPU de mas.
	maxPasswordLength = 72
)

// User es un usuario de la plataforma. PasswordHash guarda el hash de la contrasena,
// nunca la contrasena en claro, y no se expone en la capa de delivery.
type User struct {
	ID           string
	Email        string
	PasswordHash string
	CreatedAt    time.Time
}

// NewUser construye un usuario normalizando y validando el email. El email se guarda normalizado
// para que la unicidad sea insensible a mayusculas y espacios.
func NewUser(id, email, passwordHash string, createdAt time.Time) (User, error) {
	email = NormalizeEmail(email)
	if err := ValidateEmail(email); err != nil {
		return User{}, err
	}
	return User{ID: id, Email: email, PasswordHash: passwordHash, CreatedAt: createdAt}, nil
}

// NormalizeEmail lleva el email a una forma canonica (minusculas, sin espacios en los bordes).
// El caso de uso lo usa tambien para las busquedas de login, de modo que "A@B.com" y "a@b.com"
// se traten como el mismo usuario.
func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// ValidateEmail verifica que el email tenga un formato valido.
func ValidateEmail(email string) error {
	addr, err := mail.ParseAddress(email)
	if err != nil || addr.Address != email {
		return ErrInvalidEmail
	}
	return nil
}

// ValidatePassword aplica la politica de contrasena: entre 8 y 72 caracteres y al menos un numero.
func ValidatePassword(password string) error {
	if len(password) < minPasswordLength || len(password) > maxPasswordLength {
		return ErrInvalidPassword
	}
	for _, r := range password {
		if unicode.IsDigit(r) {
			return nil
		}
	}
	return ErrInvalidPassword
}
