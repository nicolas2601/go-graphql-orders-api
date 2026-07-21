package domain

import "context"

// ProductFilter son los criterios opcionales para listar productos. Un puntero nil significa
// "sin filtrar por ese campo".
type ProductFilter struct {
	Name     *string
	MinPrice *float64
	MaxPrice *float64
}

// UserRepository persiste usuarios. Vive en el dominio; las implementaciones (postgres, memoria)
// viven en la capa de repositorio.
type UserRepository interface {
	// Create persiste un usuario. Debe devolver ErrEmailAlreadyRegistered si el email ya existe.
	Create(ctx context.Context, user User) error
	// GetByEmail devuelve el usuario con ese email, o ErrUserNotFound si no existe.
	GetByEmail(ctx context.Context, email string) (User, error)
	// GetByID devuelve el usuario con ese id, o ErrUserNotFound si no existe.
	GetByID(ctx context.Context, id string) (User, error)
	// FindByIDs devuelve los usuarios cuyos ids esten en la lista (para el DataLoader).
	FindByIDs(ctx context.Context, ids []string) ([]User, error)
}

// ProductRepository persiste productos.
type ProductRepository interface {
	// List devuelve una pagina de productos que cumplen el filtro, junto con el total sin paginar.
	List(ctx context.Context, filter ProductFilter, page, pageSize int) ([]Product, int, error)
	// GetByID devuelve el producto con ese id, o ErrProductNotFound si no existe o el id es invalido.
	GetByID(ctx context.Context, id string) (Product, error)
	// FindByIDs devuelve los productos cuyos ids esten en la lista (para el DataLoader).
	FindByIDs(ctx context.Context, ids []string) ([]Product, error)
	// DecrementStock descuenta stock de forma atomica. Debe devolver ErrInsufficientStock si el
	// stock disponible es menor a la cantidad pedida, o ErrProductNotFound si el producto no existe.
	DecrementStock(ctx context.Context, id string, quantity int) error
	// IncrementStock restaura stock (por ejemplo, al cancelar una orden).
	IncrementStock(ctx context.Context, id string, quantity int) error
}

// OrderRepository persiste ordenes y sus items.
type OrderRepository interface {
	// Insert persiste una orden con sus items.
	Insert(ctx context.Context, order Order) error
	// GetByID devuelve la orden con ese id (con sus items), o ErrOrderNotFound si no existe.
	GetByID(ctx context.Context, id string) (Order, error)
	// ListByUser devuelve una pagina de ordenes de un usuario, junto con el total sin paginar.
	ListByUser(ctx context.Context, userID string, page, pageSize int) ([]Order, int, error)
	// UpdateStatus persiste el estado (y cancelled_at) de una orden.
	UpdateStatus(ctx context.Context, order Order) error
}

// TxManager ejecuta una funcion dentro de una transaccion. La implementacion propaga la
// transaccion por el contexto; el caso de uso nunca ve un *sql.Tx.
type TxManager interface {
	WithinTx(ctx context.Context, fn func(ctx context.Context) error) error
}

// TokenService emite y valida tokens JWT de acceso y de refresh.
type TokenService interface {
	GenerateAccess(userID string) (string, error)
	GenerateRefresh(userID string) (string, error)
	ParseAccess(token string) (userID string, err error)
	ParseRefresh(token string) (userID string, err error)
}

// PasswordHasher hashea y compara contrasenas.
type PasswordHasher interface {
	Hash(password string) (string, error)
	Compare(hash, password string) error
}
