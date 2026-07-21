package domain

import "time"

// OrderStatus es el estado de una orden.
type OrderStatus string

const (
	StatusPending   OrderStatus = "PENDING"
	StatusConfirmed OrderStatus = "CONFIRMED"
	StatusCancelled OrderStatus = "CANCELLED"
)

// OrderItem es una linea de una orden. UnitPrice es un snapshot del precio del producto al momento
// de crear la orden: si el producto cambia de precio despues, la orden historica no se altera.
type OrderItem struct {
	ProductID string
	Quantity  int
	UnitPrice float64
}

// Order es una orden de compra de un usuario.
type Order struct {
	ID          string
	UserID      string
	Items       []OrderItem
	Total       float64
	Status      OrderStatus
	CreatedAt   time.Time
	CancelledAt *time.Time
}

// NewOrder construye una orden validando sus invariantes y calculando el total. Queda en PENDING.
func NewOrder(id, userID string, items []OrderItem, createdAt time.Time) (Order, error) {
	if len(items) == 0 {
		return Order{}, ErrEmptyOrder
	}
	var total float64
	for _, item := range items {
		if item.Quantity <= 0 {
			return Order{}, ErrInvalidQuantity
		}
		if item.UnitPrice < 0 {
			return Order{}, ErrInvalidPrice
		}
		total += float64(item.Quantity) * item.UnitPrice
	}
	return Order{
		ID:        id,
		UserID:    userID,
		Items:     items,
		Total:     total,
		Status:    StatusPending,
		CreatedAt: createdAt,
	}, nil
}

// Confirm pasa la orden de PENDING a CONFIRMED. Devuelve ErrOrderNotPending en otro estado.
func (o *Order) Confirm() error {
	if o.Status != StatusPending {
		return ErrOrderNotPending
	}
	o.Status = StatusConfirmed
	return nil
}

// Cancel pasa la orden de PENDING a CANCELLED y registra el momento. Devuelve ErrOrderNotPending
// en otro estado. La restauracion de stock la orquesta el caso de uso, no el dominio.
func (o *Order) Cancel(now time.Time) error {
	if o.Status != StatusPending {
		return ErrOrderNotPending
	}
	o.Status = StatusCancelled
	o.CancelledAt = &now
	return nil
}
