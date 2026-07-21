// Package model contiene los modelos de la capa GraphQL. Order/OrderItem son modelos propios que
// llevan las claves foraneas (UserID, ProductID) para resolver esos campos con DataLoader; el resto
// de los modelos los genera gqlgen en models_gen.go.
package model

import (
	"fmt"
	"io"
	"strconv"
	"time"
)

// Order es el modelo GraphQL de una orden. Guarda UserID (no el usuario completo) para que el
// campo user se resuelva por su propio resolver (DataLoader), evitando el N+1.
type Order struct {
	ID        string
	UserID    string
	Items     []*OrderItem
	Total     float64
	Status    OrderStatus
	CreatedAt time.Time
}

// OrderItem guarda ProductID para resolver el campo product con DataLoader.
type OrderItem struct {
	ProductID string
	Quantity  int
	UnitPrice float64
}

// OrderStatus es el enum de estado de una orden.
type OrderStatus string

const (
	OrderStatusPending   OrderStatus = "PENDING"
	OrderStatusConfirmed OrderStatus = "CONFIRMED"
	OrderStatusCancelled OrderStatus = "CANCELLED"
)

// IsValid indica si el valor es un estado conocido.
func (e OrderStatus) IsValid() bool {
	switch e {
	case OrderStatusPending, OrderStatusConfirmed, OrderStatusCancelled:
		return true
	}
	return false
}

func (e OrderStatus) String() string { return string(e) }

// UnmarshalGQL decodifica el enum desde la entrada GraphQL.
func (e *OrderStatus) UnmarshalGQL(v any) error {
	s, ok := v.(string)
	if !ok {
		return fmt.Errorf("OrderStatus must be a string")
	}
	*e = OrderStatus(s)
	if !e.IsValid() {
		return fmt.Errorf("%q is not a valid OrderStatus", s)
	}
	return nil
}

// MarshalGQL codifica el enum a la salida GraphQL.
func (e OrderStatus) MarshalGQL(w io.Writer) {
	_, _ = io.WriteString(w, strconv.Quote(string(e)))
}
