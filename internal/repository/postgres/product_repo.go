package postgres

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/nicolas2601/go-graphql-orders-api/internal/domain"
)

// ProductRepository persiste productos en PostgreSQL.
type ProductRepository struct {
	pool *pgxpool.Pool
}

// NewProductRepository crea el repositorio sobre el pool dado.
func NewProductRepository(pool *pgxpool.Pool) *ProductRepository {
	return &ProductRepository{pool: pool}
}

var _ domain.ProductRepository = (*ProductRepository)(nil)

const productColumns = "id, name, price, stock, created_at"

// List devuelve una pagina de productos que cumplen el filtro y el total sin paginar.
func (r *ProductRepository) List(ctx context.Context, filter domain.ProductFilter, page, pageSize int) ([]domain.Product, int, error) {
	where, args := buildProductFilter(filter)
	q := querier(ctx, r.pool)

	var total int
	if err := q.QueryRow(ctx, "SELECT count(*) FROM products"+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count products: %w", err)
	}

	limitArg := "$" + strconv.Itoa(len(args)+1)
	offsetArg := "$" + strconv.Itoa(len(args)+2)
	args = append(args, pageSize, (page-1)*pageSize)
	rows, err := q.Query(ctx,
		"SELECT "+productColumns+" FROM products"+where+" ORDER BY name, id LIMIT "+limitArg+" OFFSET "+offsetArg, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("query products: %w", err)
	}
	defer rows.Close()

	products := make([]domain.Product, 0)
	for rows.Next() {
		p, err := scanProductRow(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scan product: %w", err)
		}
		products = append(products, p)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate products: %w", err)
	}
	return products, total, nil
}

// buildProductFilter arma la clausula WHERE con placeholders (los valores van parametrizados).
func buildProductFilter(filter domain.ProductFilter) (string, []any) {
	conds := make([]string, 0, 3)
	args := make([]any, 0, 3)
	add := func(cond string, value any) {
		args = append(args, value)
		conds = append(conds, fmt.Sprintf(cond, len(args)))
	}
	if filter.Name != nil {
		add("name ILIKE '%%' || $%d || '%%'", *filter.Name)
	}
	if filter.MinPrice != nil {
		add("price >= $%d", *filter.MinPrice)
	}
	if filter.MaxPrice != nil {
		add("price <= $%d", *filter.MaxPrice)
	}
	if len(conds) == 0 {
		return "", args
	}
	return " WHERE " + strings.Join(conds, " AND "), args
}

// GetByID devuelve un producto por id, o ErrProductNotFound si no existe o el id es invalido.
// Dentro de una transaccion bloquea la fila (FOR UPDATE) para consistencia con el descuento de stock.
func (r *ProductRepository) GetByID(ctx context.Context, id string) (domain.Product, error) {
	row := querier(ctx, r.pool).QueryRow(ctx,
		forUpdate(ctx, "SELECT "+productColumns+" FROM products WHERE id = $1"), id)
	p, err := scanProductRow(row)
	if errors.Is(err, pgx.ErrNoRows) || isMalformedID(err) {
		return domain.Product{}, domain.ErrProductNotFound
	}
	if err != nil {
		return domain.Product{}, fmt.Errorf("query product: %w", err)
	}
	return p, nil
}

// FindByIDs devuelve un mapa id -> producto para los ids dados (para el DataLoader).
func (r *ProductRepository) FindByIDs(ctx context.Context, ids []string) (map[string]domain.Product, error) {
	rows, err := querier(ctx, r.pool).Query(ctx,
		"SELECT "+productColumns+" FROM products WHERE id = ANY($1::uuid[])", ids)
	if err != nil {
		return nil, fmt.Errorf("query products: %w", err)
	}
	defer rows.Close()

	out := make(map[string]domain.Product, len(ids))
	for rows.Next() {
		p, err := scanProductRow(rows)
		if err != nil {
			return nil, fmt.Errorf("scan product: %w", err)
		}
		out[p.ID] = p
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate products: %w", err)
	}
	return out, nil
}

// DecrementStock descuenta stock de forma atomica. La condicion stock >= qty en el mismo UPDATE
// evita la sobreventa por concurrencia (sin lecturas previas ni locks explicitos).
func (r *ProductRepository) DecrementStock(ctx context.Context, id string, quantity int) error {
	tag, err := querier(ctx, r.pool).Exec(ctx,
		`UPDATE products SET stock = stock - $2 WHERE id = $1 AND stock >= $2`, id, quantity)
	if isMalformedID(err) {
		return domain.ErrProductNotFound
	}
	if err != nil {
		return fmt.Errorf("decrement stock: %w", err)
	}
	if tag.RowsAffected() == 0 {
		// 0 filas: o el producto no existe, o no tiene stock suficiente. Se distingue.
		exists, err := r.exists(ctx, id)
		if err != nil {
			return err
		}
		if !exists {
			return domain.ErrProductNotFound
		}
		return domain.ErrInsufficientStock
	}
	return nil
}

// IncrementStock restaura stock (por ejemplo, al cancelar una orden).
func (r *ProductRepository) IncrementStock(ctx context.Context, id string, quantity int) error {
	_, err := querier(ctx, r.pool).Exec(ctx,
		`UPDATE products SET stock = stock + $2 WHERE id = $1`, id, quantity)
	if err != nil {
		return fmt.Errorf("increment stock: %w", err)
	}
	return nil
}

func (r *ProductRepository) exists(ctx context.Context, id string) (bool, error) {
	var exists bool
	err := querier(ctx, r.pool).QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM products WHERE id = $1)`, id).Scan(&exists)
	if isMalformedID(err) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("check product exists: %w", err)
	}
	return exists, nil
}

func scanProductRow(row rowScanner) (domain.Product, error) {
	var p domain.Product
	err := row.Scan(&p.ID, &p.Name, &p.Price, &p.Stock, &p.CreatedAt)
	return p, err
}
