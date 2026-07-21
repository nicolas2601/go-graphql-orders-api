package usecase

import (
	"context"
	"time"

	"github.com/nicolas2601/go-graphql-orders-api/internal/domain"
)

// OrderLine es una linea pedida por el cliente: producto y cantidad. El precio unitario NO lo pone
// el cliente; lo toma el caso de uso del producto al crear la orden (snapshot).
type OrderLine struct {
	ProductID string
	Quantity  int
}

// OrderUseCase orquesta la creacion, consulta, cancelacion y confirmacion de ordenes.
type OrderUseCase struct {
	orders   domain.OrderRepository
	products domain.ProductRepository
	tx       domain.TxManager
	newID    func() string
	now      func() time.Time
}

// NewOrderUseCase construye el caso de uso con sus dependencias inyectadas.
func NewOrderUseCase(
	orders domain.OrderRepository,
	products domain.ProductRepository,
	tx domain.TxManager,
	newID func() string,
	now func() time.Time,
) *OrderUseCase {
	if orders == nil || products == nil || tx == nil || newID == nil || now == nil {
		panic("usecase: order dependencies must not be nil")
	}
	return &OrderUseCase{orders: orders, products: products, tx: tx, newID: newID, now: now}
}

// Create crea una orden para el usuario: dentro de una transaccion valida y descuenta el stock de
// cada producto, toma el precio unitario del producto (snapshot) y persiste la orden. Fusiona las
// lineas repetidas del mismo producto. Si algo falla, la transaccion se revierte entera.
func (uc *OrderUseCase) Create(ctx context.Context, userID string, lines []OrderLine) (domain.Order, error) {
	lines = mergeLines(lines)
	if len(lines) == 0 {
		return domain.Order{}, domain.ErrEmptyOrder
	}

	var order domain.Order
	err := uc.tx.WithinTx(ctx, func(ctx context.Context) error {
		items := make([]domain.OrderItem, 0, len(lines))
		for _, line := range lines {
			if line.Quantity <= 0 {
				return domain.ErrInvalidQuantity
			}
			product, err := uc.products.GetByID(ctx, line.ProductID)
			if err != nil {
				return err
			}
			if err := uc.products.DecrementStock(ctx, line.ProductID, line.Quantity); err != nil {
				return err
			}
			items = append(items, domain.OrderItem{
				ProductID: line.ProductID,
				Quantity:  line.Quantity,
				UnitPrice: product.Price,
			})
		}
		created, err := domain.NewOrder(uc.newID(), userID, items, uc.now())
		if err != nil {
			return err
		}
		if err := uc.orders.Insert(ctx, created); err != nil {
			return err
		}
		order = created
		return nil
	})
	if err != nil {
		return domain.Order{}, err
	}
	return order, nil
}

// MyOrders devuelve una pagina de las ordenes del usuario.
func (uc *OrderUseCase) MyOrders(ctx context.Context, userID string, page, pageSize int) (Page[domain.Order], error) {
	page, pageSize = normalizePagination(page, pageSize)
	items, total, err := uc.orders.ListByUser(ctx, userID, page, pageSize)
	if err != nil {
		return Page[domain.Order]{}, err
	}
	return Page[domain.Order]{Items: items, Total: total, Page: page, PageSize: pageSize}, nil
}

// Get devuelve el detalle de una orden del usuario, o ErrOrderNotOwned si no le pertenece.
func (uc *OrderUseCase) Get(ctx context.Context, userID, orderID string) (domain.Order, error) {
	order, err := uc.orders.GetByID(ctx, orderID)
	if err != nil {
		return domain.Order{}, err
	}
	if err := assertOwner(order, userID); err != nil {
		return domain.Order{}, err
	}
	return order, nil
}

// Cancel cancela una orden PENDING del usuario restaurando el stock, dentro de una transaccion.
func (uc *OrderUseCase) Cancel(ctx context.Context, userID, orderID string) (domain.Order, error) {
	var order domain.Order
	err := uc.tx.WithinTx(ctx, func(ctx context.Context) error {
		found, err := uc.orders.GetByID(ctx, orderID)
		if err != nil {
			return err
		}
		if err := assertOwner(found, userID); err != nil {
			return err
		}
		if err := found.Cancel(uc.now()); err != nil {
			return err
		}
		for _, item := range found.Items {
			if err := uc.products.IncrementStock(ctx, item.ProductID, item.Quantity); err != nil {
				return err
			}
		}
		if err := uc.orders.UpdateStatus(ctx, found.ID, found.Status, found.CancelledAt); err != nil {
			return err
		}
		order = found
		return nil
	})
	if err != nil {
		return domain.Order{}, err
	}
	return order, nil
}

// Confirm confirma una orden PENDING del usuario.
func (uc *OrderUseCase) Confirm(ctx context.Context, userID, orderID string) (domain.Order, error) {
	var order domain.Order
	err := uc.tx.WithinTx(ctx, func(ctx context.Context) error {
		found, err := uc.orders.GetByID(ctx, orderID)
		if err != nil {
			return err
		}
		if err := assertOwner(found, userID); err != nil {
			return err
		}
		if err := found.Confirm(); err != nil {
			return err
		}
		if err := uc.orders.UpdateStatus(ctx, found.ID, found.Status, nil); err != nil {
			return err
		}
		order = found
		return nil
	})
	if err != nil {
		return domain.Order{}, err
	}
	return order, nil
}

func assertOwner(order domain.Order, userID string) error {
	if order.UserID != userID {
		return domain.ErrOrderNotOwned
	}
	return nil
}

// mergeLines fusiona las lineas del mismo producto sumando cantidades, preservando el orden de
// aparicion. Evita insertar dos filas con el mismo producto (la orden tiene una linea por producto).
func mergeLines(lines []OrderLine) []OrderLine {
	quantityByID := make(map[string]int, len(lines))
	order := make([]string, 0, len(lines))
	for _, line := range lines {
		if _, seen := quantityByID[line.ProductID]; !seen {
			order = append(order, line.ProductID)
		}
		quantityByID[line.ProductID] += line.Quantity
	}
	merged := make([]OrderLine, 0, len(order))
	for _, id := range order {
		merged = append(merged, OrderLine{ProductID: id, Quantity: quantityByID[id]})
	}
	return merged
}
