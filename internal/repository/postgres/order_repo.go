package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/nicolas2601/go-graphql-orders-api/internal/domain"
)

// OrderRepository persiste ordenes y sus items en PostgreSQL.
type OrderRepository struct {
	pool *pgxpool.Pool
}

// NewOrderRepository crea el repositorio sobre el pool dado.
func NewOrderRepository(pool *pgxpool.Pool) *OrderRepository { return &OrderRepository{pool: pool} }

var _ domain.OrderRepository = (*OrderRepository)(nil)

const orderColumns = "id, user_id, total, status, created_at, cancelled_at"

// Insert persiste una orden con sus items. Pensado para correr dentro de WithinTx.
func (r *OrderRepository) Insert(ctx context.Context, order domain.Order) error {
	q := querier(ctx, r.pool)
	if _, err := q.Exec(ctx,
		`INSERT INTO orders (id, user_id, total, status, created_at, cancelled_at)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		order.ID, order.UserID, order.Total, order.Status, order.CreatedAt, order.CancelledAt); err != nil {
		return fmt.Errorf("insert order: %w", err)
	}
	for _, item := range order.Items {
		if _, err := q.Exec(ctx,
			`INSERT INTO order_items (order_id, product_id, quantity, unit_price)
			 VALUES ($1, $2, $3, $4)`,
			order.ID, item.ProductID, item.Quantity, item.UnitPrice); err != nil {
			return fmt.Errorf("insert order item: %w", err)
		}
	}
	return nil
}

// GetByID devuelve la orden con sus items, o ErrOrderNotFound si no existe. Dentro de una
// transaccion bloquea la fila (FOR UPDATE) para evitar lost updates en transiciones concurrentes.
func (r *OrderRepository) GetByID(ctx context.Context, id string) (domain.Order, error) {
	row := querier(ctx, r.pool).QueryRow(ctx,
		forUpdate(ctx, "SELECT "+orderColumns+" FROM orders WHERE id = $1"), id)
	order, err := scanOrder(row)
	if errors.Is(err, pgx.ErrNoRows) || isMalformedID(err) {
		return domain.Order{}, domain.ErrOrderNotFound
	}
	if err != nil {
		return domain.Order{}, fmt.Errorf("query order: %w", err)
	}

	items, err := r.itemsByOrderIDs(ctx, []string{id})
	if err != nil {
		return domain.Order{}, err
	}
	order.Items = items[id]
	return order, nil
}

// ListByUser devuelve una pagina de las ordenes del usuario, con sus items, y el total sin paginar.
func (r *OrderRepository) ListByUser(ctx context.Context, userID string, page, pageSize int) ([]domain.Order, int, error) {
	q := querier(ctx, r.pool)

	var total int
	if err := q.QueryRow(ctx, `SELECT count(*) FROM orders WHERE user_id = $1`, userID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count orders: %w", err)
	}

	rows, err := q.Query(ctx,
		`SELECT `+orderColumns+` FROM orders WHERE user_id = $1 ORDER BY created_at DESC, id LIMIT $2 OFFSET $3`,
		userID, pageSize, (page-1)*pageSize)
	if err != nil {
		return nil, 0, fmt.Errorf("query orders: %w", err)
	}
	defer rows.Close()

	orders := make([]domain.Order, 0)
	ids := make([]string, 0)
	for rows.Next() {
		order, err := scanOrder(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scan order: %w", err)
		}
		orders = append(orders, order)
		ids = append(ids, order.ID)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate orders: %w", err)
	}

	// Carga de items en una sola query (evita N+1 dentro del repositorio).
	itemsByOrder, err := r.itemsByOrderIDs(ctx, ids)
	if err != nil {
		return nil, 0, err
	}
	for i := range orders {
		orders[i].Items = itemsByOrder[orders[i].ID]
	}
	return orders, total, nil
}

// UpdateStatus persiste el estado y cancelled_at de una orden.
func (r *OrderRepository) UpdateStatus(ctx context.Context, orderID string, status domain.OrderStatus, cancelledAt *time.Time) error {
	tag, err := querier(ctx, r.pool).Exec(ctx,
		`UPDATE orders SET status = $2, cancelled_at = $3 WHERE id = $1`, orderID, status, cancelledAt)
	if isMalformedID(err) {
		return domain.ErrOrderNotFound
	}
	if err != nil {
		return fmt.Errorf("update order status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrOrderNotFound
	}
	return nil
}

// itemsByOrderIDs carga los items de varias ordenes agrupados por order_id.
func (r *OrderRepository) itemsByOrderIDs(ctx context.Context, orderIDs []string) (map[string][]domain.OrderItem, error) {
	out := make(map[string][]domain.OrderItem, len(orderIDs))
	if len(orderIDs) == 0 {
		return out, nil
	}
	rows, err := querier(ctx, r.pool).Query(ctx,
		`SELECT order_id, product_id, quantity, unit_price FROM order_items WHERE order_id = ANY($1::uuid[]) ORDER BY product_id`,
		orderIDs)
	if err != nil {
		return nil, fmt.Errorf("query order items: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var orderID string
		var item domain.OrderItem
		if err := rows.Scan(&orderID, &item.ProductID, &item.Quantity, &item.UnitPrice); err != nil {
			return nil, fmt.Errorf("scan order item: %w", err)
		}
		out[orderID] = append(out[orderID], item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate order items: %w", err)
	}
	return out, nil
}

func scanOrder(row rowScanner) (domain.Order, error) {
	var o domain.Order
	err := row.Scan(&o.ID, &o.UserID, &o.Total, &o.Status, &o.CreatedAt, &o.CancelledAt)
	return o, err
}
