package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/nicolas2601/go-graphql-orders-api/internal/domain"
	"github.com/nicolas2601/go-graphql-orders-api/internal/usecase"
)

// --- Dobles de prueba ---

type stockProductRepo struct {
	products     map[string]domain.Product
	incrementErr error
}

func (r *stockProductRepo) GetByID(_ context.Context, id string) (domain.Product, error) {
	p, ok := r.products[id]
	if !ok {
		return domain.Product{}, domain.ErrProductNotFound
	}
	return p, nil
}

func (r *stockProductRepo) DecrementStock(_ context.Context, id string, qty int) error {
	p, ok := r.products[id]
	if !ok {
		return domain.ErrProductNotFound
	}
	if p.Stock < qty {
		return domain.ErrInsufficientStock
	}
	p.Stock -= qty
	r.products[id] = p
	return nil
}

func (r *stockProductRepo) IncrementStock(_ context.Context, id string, qty int) error {
	if r.incrementErr != nil {
		return r.incrementErr
	}
	p := r.products[id]
	p.Stock += qty
	r.products[id] = p
	return nil
}

func (r *stockProductRepo) List(context.Context, domain.ProductFilter, int, int) ([]domain.Product, int, error) {
	return nil, 0, nil
}
func (r *stockProductRepo) FindByIDs(context.Context, []string) (map[string]domain.Product, error) {
	return nil, nil
}

type fakeOrderRepo struct {
	orders    map[string]domain.Order
	insertErr error
	listErr   error
	updateErr error
}

func newFakeOrderRepo() *fakeOrderRepo { return &fakeOrderRepo{orders: map[string]domain.Order{}} }

func (r *fakeOrderRepo) Insert(_ context.Context, o domain.Order) error {
	if r.insertErr != nil {
		return r.insertErr
	}
	r.orders[o.ID] = o
	return nil
}

func (r *fakeOrderRepo) GetByID(_ context.Context, id string) (domain.Order, error) {
	o, ok := r.orders[id]
	if !ok {
		return domain.Order{}, domain.ErrOrderNotFound
	}
	return o, nil
}

func (r *fakeOrderRepo) ListByUser(_ context.Context, userID string, _, _ int) ([]domain.Order, int, error) {
	if r.listErr != nil {
		return nil, 0, r.listErr
	}
	out := []domain.Order{}
	for _, o := range r.orders {
		if o.UserID == userID {
			out = append(out, o)
		}
	}
	return out, len(out), nil
}

func (r *fakeOrderRepo) UpdateStatus(_ context.Context, id string, status domain.OrderStatus, cancelledAt *time.Time) error {
	if r.updateErr != nil {
		return r.updateErr
	}
	o := r.orders[id]
	o.Status = status
	o.CancelledAt = cancelledAt
	r.orders[id] = o
	return nil
}

// noopTx ejecuta la funcion sin transaccion real (la atomicidad real se prueba en la capa Postgres).
type noopTx struct{}

func (noopTx) WithinTx(ctx context.Context, fn func(context.Context) error) error { return fn(ctx) }

func newOrderUseCase(orders domain.OrderRepository, products domain.ProductRepository) *usecase.OrderUseCase {
	return usecase.NewOrderUseCase(orders, products, noopTx{},
		func() string { return "order-1" },
		func() time.Time { return time.Date(2026, time.July, 21, 12, 0, 0, 0, time.UTC) },
	)
}

func catalog() *stockProductRepo {
	return &stockProductRepo{products: map[string]domain.Product{
		"p1": {ID: "p1", Name: "Keyboard", Price: 10, Stock: 5},
		"p2": {ID: "p2", Name: "Mouse", Price: 5, Stock: 3},
	}}
}

const buyer = "user-1"

// --- Create ---

func TestOrderCreate(t *testing.T) {
	ctx := context.Background()

	t.Run("creates a pending order, snapshots price and deducts stock", func(t *testing.T) {
		products := catalog()
		orders := newFakeOrderRepo()
		uc := newOrderUseCase(orders, products)

		order, err := uc.Create(ctx, buyer, []usecase.OrderLine{
			{ProductID: "p1", Quantity: 2},
			{ProductID: "p2", Quantity: 1},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if order.Status != domain.StatusPending || order.UserID != buyer {
			t.Fatalf("unexpected order: %+v", order)
		}
		if order.Total != 25 { // 2*10 + 1*5
			t.Fatalf("total: got %v, want 25", order.Total)
		}
		if products.products["p1"].Stock != 3 || products.products["p2"].Stock != 2 {
			t.Fatalf("stock not deducted: p1=%d p2=%d", products.products["p1"].Stock, products.products["p2"].Stock)
		}
		if _, ok := orders.orders["order-1"]; !ok {
			t.Fatal("order was not persisted")
		}
	})

	t.Run("merges duplicate product lines", func(t *testing.T) {
		products := catalog()
		uc := newOrderUseCase(newFakeOrderRepo(), products)

		order, err := uc.Create(ctx, buyer, []usecase.OrderLine{
			{ProductID: "p1", Quantity: 2},
			{ProductID: "p1", Quantity: 1},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(order.Items) != 1 || order.Items[0].Quantity != 3 {
			t.Fatalf("lines not merged: %+v", order.Items)
		}
		if products.products["p1"].Stock != 2 { // 5 - 3
			t.Fatalf("stock: got %d, want 2", products.products["p1"].Stock)
		}
	})

	t.Run("insufficient stock aborts without persisting", func(t *testing.T) {
		products := catalog()
		orders := newFakeOrderRepo()
		uc := newOrderUseCase(orders, products)

		_, err := uc.Create(ctx, buyer, []usecase.OrderLine{{ProductID: "p2", Quantity: 99}})
		if !errors.Is(err, domain.ErrInsufficientStock) {
			t.Fatalf("got %v, want ErrInsufficientStock", err)
		}
		if len(orders.orders) != 0 {
			t.Fatal("no order should have been persisted")
		}
	})

	t.Run("unknown product returns ErrProductNotFound", func(t *testing.T) {
		uc := newOrderUseCase(newFakeOrderRepo(), catalog())
		if _, err := uc.Create(ctx, buyer, []usecase.OrderLine{{ProductID: "ghost", Quantity: 1}}); !errors.Is(err, domain.ErrProductNotFound) {
			t.Fatalf("got %v, want ErrProductNotFound", err)
		}
	})

	t.Run("empty order returns ErrEmptyOrder", func(t *testing.T) {
		uc := newOrderUseCase(newFakeOrderRepo(), catalog())
		if _, err := uc.Create(ctx, buyer, nil); !errors.Is(err, domain.ErrEmptyOrder) {
			t.Fatalf("got %v, want ErrEmptyOrder", err)
		}
	})

	t.Run("non-positive quantity returns ErrInvalidQuantity", func(t *testing.T) {
		uc := newOrderUseCase(newFakeOrderRepo(), catalog())
		if _, err := uc.Create(ctx, buyer, []usecase.OrderLine{{ProductID: "p1", Quantity: 0}}); !errors.Is(err, domain.ErrInvalidQuantity) {
			t.Fatalf("got %v, want ErrInvalidQuantity", err)
		}
	})

	t.Run("repository insert error propagates", func(t *testing.T) {
		orders := newFakeOrderRepo()
		orders.insertErr = errors.New("db down")
		uc := newOrderUseCase(orders, catalog())
		if _, err := uc.Create(ctx, buyer, []usecase.OrderLine{{ProductID: "p1", Quantity: 1}}); err == nil {
			t.Fatal("expected insert error to propagate")
		}
	})

	t.Run("product with invalid price is rejected by the domain", func(t *testing.T) {
		products := &stockProductRepo{products: map[string]domain.Product{"bad": {ID: "bad", Price: 0, Stock: 5}}}
		uc := newOrderUseCase(newFakeOrderRepo(), products)
		if _, err := uc.Create(ctx, buyer, []usecase.OrderLine{{ProductID: "bad", Quantity: 1}}); !errors.Is(err, domain.ErrInvalidPrice) {
			t.Fatalf("got %v, want ErrInvalidPrice", err)
		}
	})
}

// --- MyOrders / Get ---

func seedOrder(t *testing.T, orders *fakeOrderRepo, products *stockProductRepo, owner string) domain.Order {
	t.Helper()
	uc := usecase.NewOrderUseCase(orders, products, noopTx{},
		func() string { return "order-1" },
		func() time.Time { return time.Date(2026, time.July, 21, 12, 0, 0, 0, time.UTC) },
	)
	o, err := uc.Create(context.Background(), owner, []usecase.OrderLine{{ProductID: "p1", Quantity: 1}})
	if err != nil {
		t.Fatalf("seed order: %v", err)
	}
	return o
}

func TestOrderMyOrders(t *testing.T) {
	orders := newFakeOrderRepo()
	_ = seedOrder(t, orders, catalog(), buyer)
	uc := newOrderUseCase(orders, catalog())

	page, err := uc.MyOrders(context.Background(), buyer, 1, 20)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if page.Total != 1 || len(page.Items) != 1 || page.Page != 1 {
		t.Fatalf("unexpected page: %+v", page)
	}
}

func TestOrderMyOrdersListError(t *testing.T) {
	orders := newFakeOrderRepo()
	orders.listErr = errors.New("db down")
	uc := newOrderUseCase(orders, catalog())
	if _, err := uc.MyOrders(context.Background(), buyer, 1, 20); err == nil {
		t.Fatal("expected list error to propagate")
	}
}

func TestOrderGet(t *testing.T) {
	ctx := context.Background()
	orders := newFakeOrderRepo()
	_ = seedOrder(t, orders, catalog(), buyer)
	uc := newOrderUseCase(orders, catalog())

	t.Run("owner reads the order", func(t *testing.T) {
		o, err := uc.Get(ctx, buyer, "order-1")
		if err != nil || o.ID != "order-1" {
			t.Fatalf("got %+v, %v", o, err)
		}
	})

	t.Run("non-owner is rejected", func(t *testing.T) {
		if _, err := uc.Get(ctx, "intruder", "order-1"); !errors.Is(err, domain.ErrOrderNotOwned) {
			t.Fatalf("got %v, want ErrOrderNotOwned", err)
		}
	})

	t.Run("missing order returns ErrOrderNotFound", func(t *testing.T) {
		if _, err := uc.Get(ctx, buyer, "ghost"); !errors.Is(err, domain.ErrOrderNotFound) {
			t.Fatalf("got %v, want ErrOrderNotFound", err)
		}
	})
}

// --- Cancel / Confirm ---

func TestOrderCancel(t *testing.T) {
	ctx := context.Background()

	t.Run("owner cancels a pending order and restores stock", func(t *testing.T) {
		products := catalog()
		orders := newFakeOrderRepo()
		_ = seedOrder(t, orders, products, buyer) // descuenta 1 de p1 (stock 5 -> 4)

		uc := newOrderUseCase(orders, products)
		o, err := uc.Cancel(ctx, buyer, "order-1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if o.Status != domain.StatusCancelled || o.CancelledAt == nil {
			t.Fatalf("order not cancelled: %+v", o)
		}
		if products.products["p1"].Stock != 5 { // restaurado
			t.Fatalf("stock not restored: got %d, want 5", products.products["p1"].Stock)
		}
	})

	t.Run("non-owner cannot cancel", func(t *testing.T) {
		orders := newFakeOrderRepo()
		_ = seedOrder(t, orders, catalog(), buyer)
		uc := newOrderUseCase(orders, catalog())
		if _, err := uc.Cancel(ctx, "intruder", "order-1"); !errors.Is(err, domain.ErrOrderNotOwned) {
			t.Fatalf("got %v, want ErrOrderNotOwned", err)
		}
	})

	t.Run("cannot cancel a non-pending order", func(t *testing.T) {
		orders := newFakeOrderRepo()
		_ = seedOrder(t, orders, catalog(), buyer)
		uc := newOrderUseCase(orders, catalog())
		_, _ = uc.Cancel(ctx, buyer, "order-1") // primera cancelacion
		if _, err := uc.Cancel(ctx, buyer, "order-1"); !errors.Is(err, domain.ErrOrderNotPending) {
			t.Fatalf("got %v, want ErrOrderNotPending", err)
		}
	})

	t.Run("missing order returns ErrOrderNotFound", func(t *testing.T) {
		uc := newOrderUseCase(newFakeOrderRepo(), catalog())
		if _, err := uc.Cancel(ctx, buyer, "ghost"); !errors.Is(err, domain.ErrOrderNotFound) {
			t.Fatalf("got %v, want ErrOrderNotFound", err)
		}
	})

	t.Run("stock restore error propagates", func(t *testing.T) {
		products := catalog()
		orders := newFakeOrderRepo()
		_ = seedOrder(t, orders, products, buyer)
		products.incrementErr = errors.New("db down")
		uc := newOrderUseCase(orders, products)
		if _, err := uc.Cancel(ctx, buyer, "order-1"); err == nil {
			t.Fatal("expected stock restore error to propagate")
		}
	})

	t.Run("update status error propagates", func(t *testing.T) {
		orders := newFakeOrderRepo()
		_ = seedOrder(t, orders, catalog(), buyer)
		orders.updateErr = errors.New("db down")
		uc := newOrderUseCase(orders, catalog())
		if _, err := uc.Cancel(ctx, buyer, "order-1"); err == nil {
			t.Fatal("expected update error to propagate")
		}
	})
}

func TestOrderConfirm(t *testing.T) {
	ctx := context.Background()

	t.Run("owner confirms a pending order", func(t *testing.T) {
		orders := newFakeOrderRepo()
		_ = seedOrder(t, orders, catalog(), buyer)
		uc := newOrderUseCase(orders, catalog())
		o, err := uc.Confirm(ctx, buyer, "order-1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if o.Status != domain.StatusConfirmed {
			t.Fatalf("order not confirmed: %+v", o)
		}
	})

	t.Run("non-owner cannot confirm", func(t *testing.T) {
		orders := newFakeOrderRepo()
		_ = seedOrder(t, orders, catalog(), buyer)
		uc := newOrderUseCase(orders, catalog())
		if _, err := uc.Confirm(ctx, "intruder", "order-1"); !errors.Is(err, domain.ErrOrderNotOwned) {
			t.Fatalf("got %v, want ErrOrderNotOwned", err)
		}
	})

	t.Run("cannot confirm a cancelled order", func(t *testing.T) {
		orders := newFakeOrderRepo()
		_ = seedOrder(t, orders, catalog(), buyer)
		uc := newOrderUseCase(orders, catalog())
		_, _ = uc.Cancel(ctx, buyer, "order-1")
		if _, err := uc.Confirm(ctx, buyer, "order-1"); !errors.Is(err, domain.ErrOrderNotPending) {
			t.Fatalf("got %v, want ErrOrderNotPending", err)
		}
	})

	t.Run("missing order returns ErrOrderNotFound", func(t *testing.T) {
		uc := newOrderUseCase(newFakeOrderRepo(), catalog())
		if _, err := uc.Confirm(ctx, buyer, "ghost"); !errors.Is(err, domain.ErrOrderNotFound) {
			t.Fatalf("got %v, want ErrOrderNotFound", err)
		}
	})

	t.Run("update status error propagates", func(t *testing.T) {
		orders := newFakeOrderRepo()
		_ = seedOrder(t, orders, catalog(), buyer)
		orders.updateErr = errors.New("db down")
		uc := newOrderUseCase(orders, catalog())
		if _, err := uc.Confirm(ctx, buyer, "order-1"); err == nil {
			t.Fatal("expected update error to propagate")
		}
	})
}

func TestNewOrderUseCasePanicsOnNilDependency(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic when a dependency is nil")
		}
	}()
	usecase.NewOrderUseCase(nil, nil, nil, nil, nil)
}
