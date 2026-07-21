package domain_test

import (
	"errors"
	"testing"
	"time"

	"github.com/nicolas2601/go-graphql-orders-api/internal/domain"
)

func sampleItems() []domain.OrderItem {
	return []domain.OrderItem{
		{ProductID: "p1", Quantity: 2, UnitPrice: 10.0},
		{ProductID: "p2", Quantity: 1, UnitPrice: 5.5},
	}
}

func TestNewOrder(t *testing.T) {
	now := time.Date(2026, time.July, 21, 12, 0, 0, 0, time.UTC)

	t.Run("computes total and starts pending", func(t *testing.T) {
		o, err := domain.NewOrder("o1", "u1", sampleItems(), now)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if o.Total != 25.5 { // 2*10 + 1*5.5
			t.Fatalf("total: got %v, want 25.5", o.Total)
		}
		if o.Status != domain.StatusPending {
			t.Fatalf("status: got %v, want PENDING", o.Status)
		}
		if o.UserID != "u1" || len(o.Items) != 2 {
			t.Fatalf("unexpected order: %+v", o)
		}
	})

	t.Run("empty order returns ErrEmptyOrder", func(t *testing.T) {
		if _, err := domain.NewOrder("o1", "u1", nil, now); !errors.Is(err, domain.ErrEmptyOrder) {
			t.Fatalf("got %v, want ErrEmptyOrder", err)
		}
	})

	t.Run("non-positive quantity returns ErrInvalidQuantity", func(t *testing.T) {
		items := []domain.OrderItem{{ProductID: "p1", Quantity: 0, UnitPrice: 10}}
		if _, err := domain.NewOrder("o1", "u1", items, now); !errors.Is(err, domain.ErrInvalidQuantity) {
			t.Fatalf("got %v, want ErrInvalidQuantity", err)
		}
	})

	t.Run("negative unit price returns ErrInvalidPrice", func(t *testing.T) {
		items := []domain.OrderItem{{ProductID: "p1", Quantity: 1, UnitPrice: -1}}
		if _, err := domain.NewOrder("o1", "u1", items, now); !errors.Is(err, domain.ErrInvalidPrice) {
			t.Fatalf("got %v, want ErrInvalidPrice", err)
		}
	})
}

func TestOrderConfirm(t *testing.T) {
	now := time.Date(2026, time.July, 21, 12, 0, 0, 0, time.UTC)

	t.Run("pending order can be confirmed", func(t *testing.T) {
		o, _ := domain.NewOrder("o1", "u1", sampleItems(), now)
		if err := o.Confirm(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if o.Status != domain.StatusConfirmed {
			t.Fatalf("status: got %v, want CONFIRMED", o.Status)
		}
	})

	t.Run("non-pending order cannot be confirmed", func(t *testing.T) {
		o, _ := domain.NewOrder("o1", "u1", sampleItems(), now)
		_ = o.Cancel(now)
		if err := o.Confirm(); !errors.Is(err, domain.ErrOrderNotPending) {
			t.Fatalf("got %v, want ErrOrderNotPending", err)
		}
	})
}

func TestOrderCancel(t *testing.T) {
	now := time.Date(2026, time.July, 21, 12, 0, 0, 0, time.UTC)
	later := now.Add(time.Hour)

	t.Run("pending order can be cancelled and records the timestamp", func(t *testing.T) {
		o, _ := domain.NewOrder("o1", "u1", sampleItems(), now)
		if err := o.Cancel(later); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if o.Status != domain.StatusCancelled {
			t.Fatalf("status: got %v, want CANCELLED", o.Status)
		}
		if o.CancelledAt == nil || !o.CancelledAt.Equal(later) {
			t.Fatalf("cancelledAt not recorded: %v", o.CancelledAt)
		}
	})

	t.Run("already cancelled order cannot be cancelled again", func(t *testing.T) {
		o, _ := domain.NewOrder("o1", "u1", sampleItems(), now)
		_ = o.Cancel(later)
		if err := o.Cancel(later); !errors.Is(err, domain.ErrOrderNotPending) {
			t.Fatalf("got %v, want ErrOrderNotPending", err)
		}
	})
}
