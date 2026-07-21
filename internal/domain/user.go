package domain

import (
	"net/mail"
	"time"
	"unicode"
)

const minPasswordLength = 8

// User es un usuario de la plataforma. PasswordHash guarda el hash de la contrasena,
// nunca la contrasena en claro, y no se expone en la capa de delivery.
type User struct {
	ID           string
	Email        string
	PasswordHash string
	CreatedAt    time.Time
}

// NewUser construye un usuario validando el formato del email.
func NewUser(id, email, passwordHash string, createdAt time.Time) (User, error) {
	if err := ValidateEmail(email); err != nil {
		return User{}, err
	}
	return User{ID: id, Email: email, PasswordHash: passwordHash, CreatedAt: createdAt}, nil
}

// ValidateEmail verifica que el email tenga un formato valido.
func ValidateEmail(email string) error {
	addr, err := mail.ParseAddress(email)
	if err != nil || addr.Address != email {
		return ErrInvalidEmail
	}
	return nil
}

// ValidatePassword aplica la politica de contrasena: al menos 8 caracteres y al menos un numero.
func ValidatePassword(password string) error {
	if len(password) < minPasswordLength {
		return ErrInvalidPassword
	}
	for _, r := range password {
		if unicode.IsDigit(r) {
			return nil
		}
	}
	return ErrInvalidPassword
}
