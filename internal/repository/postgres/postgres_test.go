package postgres_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/nicolas2601/go-graphql-orders-api/internal/domain"
	"github.com/nicolas2601/go-graphql-orders-api/internal/repository/postgres"
	"github.com/nicolas2601/go-graphql-orders-api/internal/usecase"
)

type harness struct {
	pool     *pgxpool.Pool
	users    *postgres.UserRepository
	products *postgres.ProductRepository
	orders   *postgres.OrderRepository
	tx       *postgres.TxManager
}

func (h *harness) reset(t *testing.T) {
	t.Helper()
	if _, err := h.pool.Exec(context.Background(), "TRUNCATE users, products, orders, order_items"); err != nil {
		t.Fatalf("truncate: %v", err)
	}
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	testcontainers.SkipIfProviderIsNotHealthy(t)
	ctx := context.Background()

	container, err := tcpostgres.Run(ctx, "postgres:16-alpine",
		tcpostgres.WithDatabase("orders"),
		tcpostgres.WithUsername("orders"),
		tcpostgres.WithPassword("orders"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).WithStartupTimeout(90*time.Second),
		),
	)
	if err != nil {
		t.Fatalf("start postgres: %v", err)
	}
	testcontainers.CleanupContainer(t, container)

	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("connection string: %v", err)
	}
	if err := postgres.RunMigrations(dsn); err != nil {
		t.Fatalf("migrations: %v", err)
	}
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	t.Cleanup(pool.Close)

	return &harness{
		pool:     pool,
		users:    postgres.NewUserRepository(pool),
		products: postgres.NewProductRepository(pool),
		orders:   postgres.NewOrderRepository(pool),
		tx:       postgres.NewTxManager(pool),
	}
}

func (h *harness) seedUser(t *testing.T) domain.User {
	t.Helper()
	u := domain.User{ID: uuid.NewString(), Email: "buyer@example.com", PasswordHash: "hash", CreatedAt: time.Now().UTC()}
	if err := h.users.Create(context.Background(), u); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	return u
}

func (h *harness) seedProduct(t *testing.T, name string, price float64, stock int) domain.Product {
	t.Helper()
	p := domain.Product{ID: uuid.NewString(), Name: name, Price: price, Stock: stock, CreatedAt: time.Now().UTC()}
	if _, err := h.pool.Exec(context.Background(),
		`INSERT INTO products (id, name, price, stock, created_at) VALUES ($1,$2,$3,$4,$5)`,
		p.ID, p.Name, p.Price, p.Stock, p.CreatedAt); err != nil {
		t.Fatalf("seed product: %v", err)
	}
	return p
}

func (h *harness) orderUseCase() *usecase.OrderUseCase {
	return usecase.NewOrderUseCase(h.orders, h.products, h.tx,
		func() string { return uuid.NewString() },
		func() time.Time { return time.Now().UTC() },
	)
}

func (h *harness) stockOf(t *testing.T, id string) int {
	t.Helper()
	var stock int
	if err := h.pool.QueryRow(context.Background(), `SELECT stock FROM products WHERE id = $1`, id).Scan(&stock); err != nil {
		t.Fatalf("read stock: %v", err)
	}
	return stock
}

func TestPostgresIntegration(t *testing.T) {
	h := newHarness(t)
	ctx := context.Background()

	t.Run("user repository", func(t *testing.T) {
		h.reset(t)
		u := domain.User{ID: uuid.NewString(), Email: "nico@example.com", PasswordHash: "hash", CreatedAt: time.Now().UTC()}
		if err := h.users.Create(ctx, u); err != nil {
			t.Fatalf("create: %v", err)
		}
		if err := h.users.Create(ctx, domain.User{ID: uuid.NewString(), Email: "nico@example.com", PasswordHash: "x", CreatedAt: time.Now().UTC()}); !errors.Is(err, domain.ErrEmailAlreadyRegistered) {
			t.Fatalf("duplicate email: got %v", err)
		}
		got, err := h.users.GetByEmail(ctx, "nico@example.com")
		if err != nil || got.ID != u.ID {
			t.Fatalf("get by email: %+v %v", got, err)
		}
		if _, err := h.users.GetByID(ctx, "not-a-uuid"); !errors.Is(err, domain.ErrUserNotFound) {
			t.Fatalf("malformed id should be not found: %v", err)
		}
		byIDs, err := h.users.FindByIDs(ctx, []string{u.ID})
		if err != nil || byIDs[u.ID].Email != "nico@example.com" {
			t.Fatalf("find by ids: %+v %v", byIDs, err)
		}
	})

	t.Run("product repository list, filter and stock", func(t *testing.T) {
		h.reset(t)
		h.seedProduct(t, "Keyboard", 50, 10)
		h.seedProduct(t, "Mouse", 20, 10)
		h.seedProduct(t, "Monitor", 300, 10)

		max := 100.0
		items, total, err := h.products.List(ctx, domain.ProductFilter{MaxPrice: &max}, 1, 20)
		if err != nil || total != 2 || len(items) != 2 {
			t.Fatalf("filter by price: total=%d items=%d err=%v", total, len(items), err)
		}

		name := "mou"
		items, total, err = h.products.List(ctx, domain.ProductFilter{Name: &name}, 1, 20)
		if err != nil || total != 1 || items[0].Name != "Mouse" {
			t.Fatalf("filter by name: %+v %d %v", items, total, err)
		}

		mouse := items[0]
		if err := h.products.DecrementStock(ctx, mouse.ID, 4); err != nil {
			t.Fatalf("decrement: %v", err)
		}
		if h.stockOf(t, mouse.ID) != 6 {
			t.Fatalf("stock not decremented: %d", h.stockOf(t, mouse.ID))
		}
		if err := h.products.DecrementStock(ctx, mouse.ID, 999); !errors.Is(err, domain.ErrInsufficientStock) {
			t.Fatalf("insufficient: got %v", err)
		}
		if err := h.products.DecrementStock(ctx, uuid.NewString(), 1); !errors.Is(err, domain.ErrProductNotFound) {
			t.Fatalf("missing product: got %v", err)
		}
		if err := h.products.IncrementStock(ctx, mouse.ID, 4); err != nil || h.stockOf(t, mouse.ID) != 10 {
			t.Fatalf("increment: %v stock=%d", err, h.stockOf(t, mouse.ID))
		}

		byIDs, err := h.products.FindByIDs(ctx, []string{mouse.ID})
		if err != nil || byIDs[mouse.ID].Name != "Mouse" {
			t.Fatalf("find by ids: %+v %v", byIDs, err)
		}
	})

	t.Run("repository not-found paths", func(t *testing.T) {
		h.reset(t)
		if _, err := h.users.GetByEmail(ctx, "ghost@example.com"); !errors.Is(err, domain.ErrUserNotFound) {
			t.Fatalf("user not found: got %v", err)
		}
		if _, err := h.orders.GetByID(ctx, uuid.NewString()); !errors.Is(err, domain.ErrOrderNotFound) {
			t.Fatalf("order not found: got %v", err)
		}
		if _, err := h.orders.GetByID(ctx, "not-a-uuid"); !errors.Is(err, domain.ErrOrderNotFound) {
			t.Fatalf("order malformed id: got %v", err)
		}
		if err := h.orders.UpdateStatus(ctx, uuid.NewString(), domain.StatusConfirmed, nil); !errors.Is(err, domain.ErrOrderNotFound) {
			t.Fatalf("update missing order: got %v", err)
		}
	})

	t.Run("order use case creates, persists items and deducts stock", func(t *testing.T) {
		h.reset(t)
		buyer := h.seedUser(t)
		p1 := h.seedProduct(t, "Keyboard", 50, 10)
		p2 := h.seedProduct(t, "Mouse", 20, 10)
		uc := h.orderUseCase()

		order, err := uc.Create(ctx, buyer.ID, []usecase.OrderLine{
			{ProductID: p1.ID, Quantity: 2},
			{ProductID: p2.ID, Quantity: 3},
		})
		if err != nil {
			t.Fatalf("create order: %v", err)
		}
		if order.Total != 160 { // 2*50 + 3*20
			t.Fatalf("total: got %v want 160", order.Total)
		}
		if h.stockOf(t, p1.ID) != 8 || h.stockOf(t, p2.ID) != 7 {
			t.Fatalf("stock not deducted: p1=%d p2=%d", h.stockOf(t, p1.ID), h.stockOf(t, p2.ID))
		}

		got, err := uc.Get(ctx, buyer.ID, order.ID)
		if err != nil || len(got.Items) != 2 {
			t.Fatalf("get order: %+v %v", got, err)
		}

		page, err := uc.MyOrders(ctx, buyer.ID, 1, 20)
		if err != nil || page.Total != 1 {
			t.Fatalf("my orders: %+v %v", page, err)
		}

		if _, err := uc.Cancel(ctx, buyer.ID, order.ID); err != nil {
			t.Fatalf("cancel: %v", err)
		}
		if h.stockOf(t, p1.ID) != 10 || h.stockOf(t, p2.ID) != 10 {
			t.Fatalf("stock not restored: p1=%d p2=%d", h.stockOf(t, p1.ID), h.stockOf(t, p2.ID))
		}
	})

	t.Run("create rolls back entirely when a later product lacks stock", func(t *testing.T) {
		h.reset(t)
		buyer := h.seedUser(t)
		p1 := h.seedProduct(t, "Keyboard", 50, 10) // suficiente
		p2 := h.seedProduct(t, "Mouse", 20, 1)     // insuficiente para qty 5
		uc := h.orderUseCase()

		_, err := uc.Create(ctx, buyer.ID, []usecase.OrderLine{
			{ProductID: p1.ID, Quantity: 2},
			{ProductID: p2.ID, Quantity: 5},
		})
		if !errors.Is(err, domain.ErrInsufficientStock) {
			t.Fatalf("got %v, want ErrInsufficientStock", err)
		}
		// La transaccion debe revertir TODO: el stock de p1 no se descuenta y no queda ninguna orden.
		if h.stockOf(t, p1.ID) != 10 {
			t.Fatalf("rollback failed: p1 stock is %d, want 10", h.stockOf(t, p1.ID))
		}
		var orderCount int
		if err := h.pool.QueryRow(ctx, `SELECT count(*) FROM orders`).Scan(&orderCount); err != nil {
			t.Fatal(err)
		}
		if orderCount != 0 {
			t.Fatalf("rollback failed: %d orders persisted", orderCount)
		}
	})

	t.Run("concurrent orders never oversell", func(t *testing.T) {
		h.reset(t)
		buyer := h.seedUser(t)
		p := h.seedProduct(t, "Limited", 10, 5) // solo 5 unidades
		uc := h.orderUseCase()

		const attempts = 20
		var (
			wg        sync.WaitGroup
			mu        sync.Mutex
			succeeded int
		)
		wg.Add(attempts)
		for i := 0; i < attempts; i++ {
			go func() {
				defer wg.Done()
				_, err := uc.Create(ctx, buyer.ID, []usecase.OrderLine{{ProductID: p.ID, Quantity: 1}})
				if err == nil {
					mu.Lock()
					succeeded++
					mu.Unlock()
				}
			}()
		}
		wg.Wait()

		if succeeded != 5 {
			t.Fatalf("expected exactly 5 successful orders, got %d", succeeded)
		}
		if h.stockOf(t, p.ID) != 0 {
			t.Fatalf("oversold: final stock is %d, want 0", h.stockOf(t, p.ID))
		}
	})

	t.Run("txmanager rolls back on error", func(t *testing.T) {
		h.reset(t)
		p := h.seedProduct(t, "Widget", 10, 10)
		wantErr := errors.New("boom")

		err := h.tx.WithinTx(ctx, func(ctx context.Context) error {
			if err := h.products.DecrementStock(ctx, p.ID, 3); err != nil {
				return err
			}
			return wantErr // fuerza el rollback
		})
		if !errors.Is(err, wantErr) {
			t.Fatalf("got %v, want boom", err)
		}
		if h.stockOf(t, p.ID) != 10 {
			t.Fatalf("rollback failed: stock is %d, want 10", h.stockOf(t, p.ID))
		}
	})

	t.Run("panic inside a transaction rolls back and does not leak the connection", func(t *testing.T) {
		h.reset(t)
		p := h.seedProduct(t, "Widget", 10, 10)

		func() {
			defer func() { _ = recover() }() // capturamos el panic re-lanzado
			_ = h.tx.WithinTx(ctx, func(ctx context.Context) error {
				_ = h.products.DecrementStock(ctx, p.ID, 3)
				panic("boom in fn")
			})
		}()

		if h.stockOf(t, p.ID) != 10 {
			t.Fatalf("panic did not roll back: stock is %d, want 10", h.stockOf(t, p.ID))
		}
		// La conexion no debe haberse perdido: el pool sigue respondiendo.
		if _, err := h.products.GetByID(ctx, p.ID); err != nil {
			t.Fatalf("connection leaked after panic: pool unusable: %v", err)
		}
	})
}
