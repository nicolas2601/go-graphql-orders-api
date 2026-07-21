package domain_test

import (
	"errors"
	"testing"
	"time"

	"github.com/nicolas2601/go-graphql-orders-api/internal/domain"
)

func TestNewUser(t *testing.T) {
	now := time.Date(2026, time.July, 21, 12, 0, 0, 0, time.UTC)

	t.Run("valid user", func(t *testing.T) {
		u, err := domain.NewUser("id-1", "nico@example.com", "hashed", now)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if u.Email != "nico@example.com" || u.PasswordHash != "hashed" {
			t.Fatalf("unexpected user: %+v", u)
		}
	})

	t.Run("invalid email returns ErrInvalidEmail", func(t *testing.T) {
		if _, err := domain.NewUser("id-1", "not-an-email", "hashed", now); !errors.Is(err, domain.ErrInvalidEmail) {
			t.Fatalf("got %v, want ErrInvalidEmail", err)
		}
	})
}

func TestValidateEmail(t *testing.T) {
	valid := []string{"a@b.co", "nico.perez@example.com", "user+tag@sub.domain.org"}
	invalid := []string{"", "plainstring", "@no-local.com", "no-at.com", "spaces @x.com"}

	for _, e := range valid {
		if err := domain.ValidateEmail(e); err != nil {
			t.Errorf("email %q should be valid, got %v", e, err)
		}
	}
	for _, e := range invalid {
		if err := domain.ValidateEmail(e); !errors.Is(err, domain.ErrInvalidEmail) {
			t.Errorf("email %q should be invalid", e)
		}
	}
}

func TestValidatePassword(t *testing.T) {
	t.Run("valid password", func(t *testing.T) {
		if err := domain.ValidatePassword("secret123"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
	t.Run("too short returns ErrInvalidPassword", func(t *testing.T) {
		if err := domain.ValidatePassword("abc123"); !errors.Is(err, domain.ErrInvalidPassword) {
			t.Fatalf("got %v, want ErrInvalidPassword", err)
		}
	})
	t.Run("no digit returns ErrInvalidPassword", func(t *testing.T) {
		if err := domain.ValidatePassword("onlyletters"); !errors.Is(err, domain.ErrInvalidPassword) {
			t.Fatalf("got %v, want ErrInvalidPassword", err)
		}
	})
}
