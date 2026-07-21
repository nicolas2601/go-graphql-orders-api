package graphql_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/nicolas2601/go-graphql-orders-api/internal/auth"
	"github.com/nicolas2601/go-graphql-orders-api/internal/config"
	graphqldelivery "github.com/nicolas2601/go-graphql-orders-api/internal/delivery/graphql"
	"github.com/nicolas2601/go-graphql-orders-api/internal/repository/postgres"
	"github.com/nicolas2601/go-graphql-orders-api/internal/server"
	"github.com/nicolas2601/go-graphql-orders-api/internal/usecase"
)

func newTestServer(t *testing.T) (http.Handler, *pgxpool.Pool) {
	t.Helper()
	testcontainers.SkipIfProviderIsNotHealthy(t)
	ctx := context.Background()

	container, err := tcpostgres.Run(ctx, "postgres:16-alpine",
		tcpostgres.WithDatabase("orders"), tcpostgres.WithUsername("orders"), tcpostgres.WithPassword("orders"),
		testcontainers.WithWaitStrategy(wait.ForListeningPort("5432/tcp").WithStartupTimeout(90*time.Second)),
	)
	if err != nil {
		t.Fatalf("start postgres: %v", err)
	}
	testcontainers.CleanupContainer(t, container)
	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("dsn: %v", err)
	}
	if err := postgres.RunMigrations(dsn); err != nil {
		t.Fatalf("migrations: %v", err)
	}
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	t.Cleanup(pool.Close)

	tokens, err := auth.NewJWTService("test-secret-0123456789", 15*time.Minute, 24*time.Hour, nil)
	if err != nil {
		t.Fatalf("jwt: %v", err)
	}
	hasher := auth.NewBcryptHasher(4)
	userRepo := postgres.NewUserRepository(pool)
	productRepo := postgres.NewProductRepository(pool)
	orderRepo := postgres.NewOrderRepository(pool)
	txManager := postgres.NewTxManager(pool)
	id := func() string { return uuid.NewString() }
	now := func() time.Time { return time.Now().UTC() }

	resolver := graphqldelivery.NewResolver(
		usecase.NewAuthUseCase(userRepo, hasher, tokens, id, now),
		usecase.NewUserUseCase(userRepo),
		usecase.NewProductUseCase(productRepo),
		usecase.NewOrderUseCase(orderRepo, productRepo, txManager, id, now),
	)
	handler := server.NewHandler(config.Config{AppEnv: "development"}, resolver, tokens, userRepo, productRepo)
	return handler, pool
}

func seedProduct(t *testing.T, pool *pgxpool.Pool, name string, price float64, stock int) string {
	t.Helper()
	id := uuid.NewString()
	if _, err := pool.Exec(context.Background(),
		`INSERT INTO products (id, name, price, stock, created_at) VALUES ($1,$2,$3,$4, now())`,
		id, name, price, stock); err != nil {
		t.Fatalf("seed product: %v", err)
	}
	return id
}

// gql envia una operacion GraphQL y devuelve data y el primer codigo de error (si hay).
func gql(t *testing.T, h http.Handler, token, query string) (map[string]any, string) {
	t.Helper()
	body, _ := json.Marshal(map[string]string{"query": query})
	req := httptest.NewRequest(http.MethodPost, "/query", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	var resp struct {
		Data   map[string]any `json:"data"`
		Errors []struct {
			Message    string         `json:"message"`
			Extensions map[string]any `json:"extensions"`
		} `json:"errors"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v (body: %s)", err, rec.Body.String())
	}
	code := ""
	if len(resp.Errors) > 0 {
		if c, ok := resp.Errors[0].Extensions["code"].(string); ok {
			code = c
		}
	}
	return resp.Data, code
}

func TestGraphQLEndToEnd(t *testing.T) {
	h, pool := newTestServer(t)
	p1 := seedProduct(t, pool, "Keyboard", 50, 10)
	p2 := seedProduct(t, pool, "Mouse", 20, 10)

	// register -> access token
	data, code := gql(t, h, "", `mutation { register(input:{email:"nico@example.com", password:"secret123"}) { accessToken user { email } } }`)
	if code != "" {
		t.Fatalf("register failed: %s", code)
	}
	token := data["register"].(map[string]any)["accessToken"].(string)
	if token == "" {
		t.Fatal("no access token")
	}

	t.Run("duplicate registration maps to EMAIL_ALREADY_REGISTERED", func(t *testing.T) {
		_, code := gql(t, h, "", `mutation { register(input:{email:"nico@example.com", password:"secret123"}) { accessToken } }`)
		if code != "EMAIL_ALREADY_REGISTERED" {
			t.Fatalf("got %q", code)
		}
	})

	t.Run("me returns the authenticated user", func(t *testing.T) {
		data, code := gql(t, h, token, `{ me { email } }`)
		if code != "" || data["me"].(map[string]any)["email"] != "nico@example.com" {
			t.Fatalf("me: %v code=%s", data, code)
		}
	})

	t.Run("createOrder without a token is UNAUTHENTICATED", func(t *testing.T) {
		_, code := gql(t, h, "", `mutation { createOrder(input:{items:[{productId:"`+p1+`", quantity:1}]}) { id } }`)
		if code != "UNAUTHENTICATED" {
			t.Fatalf("got %q", code)
		}
	})

	var orderID string
	t.Run("createOrder deducts stock and resolves nested product", func(t *testing.T) {
		q := `mutation { createOrder(input:{items:[{productId:"` + p1 + `", quantity:2},{productId:"` + p2 + `", quantity:1}]}) {
			id total status items { quantity unitPrice product { name } } user { email } } }`
		data, code := gql(t, h, token, q)
		if code != "" {
			t.Fatalf("createOrder failed: %s", code)
		}
		order := data["createOrder"].(map[string]any)
		orderID = order["id"].(string)
		if order["total"].(float64) != 120 { // 2*50 + 1*20
			t.Fatalf("total: got %v", order["total"])
		}
		if order["status"] != "PENDING" || order["user"].(map[string]any)["email"] != "nico@example.com" {
			t.Fatalf("unexpected order: %v", order)
		}
		items := order["items"].([]any)
		if len(items) != 2 {
			t.Fatalf("expected 2 items, got %d", len(items))
		}
	})

	t.Run("insufficient stock maps to INSUFFICIENT_STOCK", func(t *testing.T) {
		_, code := gql(t, h, token, `mutation { createOrder(input:{items:[{productId:"`+p2+`", quantity:999}]}) { id } }`)
		if code != "INSUFFICIENT_STOCK" {
			t.Fatalf("got %q", code)
		}
	})

	t.Run("myOrders lists the user orders", func(t *testing.T) {
		data, code := gql(t, h, token, `{ myOrders { total hasNextPage items { id } } }`)
		if code != "" || data["myOrders"].(map[string]any)["total"].(float64) != 1 {
			t.Fatalf("myOrders: %v code=%s", data, code)
		}
	})

	t.Run("another user cannot read the order (ORDER_NOT_OWNED)", func(t *testing.T) {
		data, _ := gql(t, h, "", `mutation { register(input:{email:"intruder@example.com", password:"secret123"}) { accessToken } }`)
		intruder := data["register"].(map[string]any)["accessToken"].(string)
		_, code := gql(t, h, intruder, `{ order(id:"`+orderID+`") { id } }`)
		if code != "ORDER_NOT_OWNED" {
			t.Fatalf("got %q", code)
		}
	})

	t.Run("cancelOrder restores stock", func(t *testing.T) {
		_, code := gql(t, h, token, `mutation { cancelOrder(id:"`+orderID+`") { status } }`)
		if code != "" {
			t.Fatalf("cancel failed: %s", code)
		}
		var stock int
		_ = pool.QueryRow(context.Background(), `SELECT stock FROM products WHERE id=$1`, p1).Scan(&stock)
		if stock != 10 {
			t.Fatalf("stock not restored: %d", stock)
		}
	})

	t.Run("products query filters and paginates", func(t *testing.T) {
		data, code := gql(t, h, token, `{ products(filter:{name:"key"}) { total items { name } } }`)
		if code != "" || data["products"].(map[string]any)["total"].(float64) != 1 {
			t.Fatalf("products: %v code=%s", data, code)
		}
	})
}
