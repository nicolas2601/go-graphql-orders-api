package auth_test

import (
	"testing"

	"github.com/nicolas2601/go-graphql-orders-api/internal/auth"
)

func TestBcryptHasher(t *testing.T) {
	h := auth.NewBcryptHasher(4) // cost bajo para tests rapidos

	t.Run("hash then compare succeeds", func(t *testing.T) {
		hash, err := h.Hash("secret123")
		if err != nil {
			t.Fatalf("hash: %v", err)
		}
		if hash == "secret123" {
			t.Fatal("hash must not equal the plaintext")
		}
		if err := h.Compare(hash, "secret123"); err != nil {
			t.Fatalf("compare should succeed: %v", err)
		}
	})

	t.Run("compare with wrong password fails", func(t *testing.T) {
		hash, _ := h.Hash("secret123")
		if err := h.Compare(hash, "wrongpass1"); err == nil {
			t.Fatal("compare with wrong password should fail")
		}
	})

	t.Run("invalid cost falls back to default", func(t *testing.T) {
		hh := auth.NewBcryptHasher(999) // fuera de rango
		if _, err := hh.Hash("secret123"); err != nil {
			t.Fatalf("hash with fallback cost should work: %v", err)
		}
	})
}
