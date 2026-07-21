package domain_test

import (
	"errors"
	"testing"
	"time"

	"github.com/nicolas2601/go-graphql-orders-api/internal/domain"
)

func TestNewProduct(t *testing.T) {
	now := time.Date(2026, time.July, 21, 12, 0, 0, 0, time.UTC)

	t.Run("valid product", func(t *testing.T) {
		p, err := domain.NewProduct("id-1", "Keyboard", 49.90, 10, now)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if p.ID != "id-1" || p.Name != "Keyboard" || p.Price != 49.90 || p.Stock != 10 {
			t.Fatalf("unexpected product: %+v", p)
		}
	})

	t.Run("empty name returns ErrInvalidName", func(t *testing.T) {
		if _, err := domain.NewProduct("id-1", "", 49.90, 10, now); !errors.Is(err, domain.ErrInvalidName) {
			t.Fatalf("got %v, want ErrInvalidName", err)
		}
	})

	t.Run("non-positive price returns ErrInvalidPrice", func(t *testing.T) {
		if _, err := domain.NewProduct("id-1", "Keyboard", 0, 10, now); !errors.Is(err, domain.ErrInvalidPrice) {
			t.Fatalf("got %v, want ErrInvalidPrice", err)
		}
	})

	t.Run("negative stock returns ErrInvalidStock", func(t *testing.T) {
		if _, err := domain.NewProduct("id-1", "Keyboard", 49.90, -1, now); !errors.Is(err, domain.ErrInvalidStock) {
			t.Fatalf("got %v, want ErrInvalidStock", err)
		}
	})
}
