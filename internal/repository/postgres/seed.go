package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/nicolas2601/go-graphql-orders-api/internal/domain"
)

// seedProducts son productos iniciales con ids fijos, para que el seed sea idempotente.
var seedProducts = []domain.Product{
	{ID: "11111111-1111-1111-1111-111111111111", Name: "Mechanical Keyboard", Price: 89.90, Stock: 50},
	{ID: "22222222-2222-2222-2222-222222222222", Name: "Wireless Mouse", Price: 39.90, Stock: 80},
	{ID: "33333333-3333-3333-3333-333333333333", Name: "4K Monitor", Price: 349.00, Stock: 20},
	{ID: "44444444-4444-4444-4444-444444444444", Name: "USB-C Hub", Price: 59.90, Stock: 40},
	{ID: "55555555-5555-5555-5555-555555555555", Name: "Noise-Cancelling Headphones", Price: 199.00, Stock: 30},
}

// SeedProducts inserta los productos iniciales de forma idempotente (no falla ni duplica si ya
// existen). Util para tener un catalogo demostrable apenas arranca la app.
func SeedProducts(ctx context.Context, pool *pgxpool.Pool) error {
	for _, p := range seedProducts {
		if _, err := pool.Exec(ctx,
			`INSERT INTO products (id, name, price, stock, created_at) VALUES ($1, $2, $3, $4, now())
			 ON CONFLICT (id) DO NOTHING`,
			p.ID, p.Name, p.Price, p.Stock); err != nil {
			return fmt.Errorf("seed product %q: %w", p.Name, err)
		}
	}
	return nil
}
