package graphql

import (
	"github.com/nicolas2601/go-graphql-orders-api/internal/usecase"
)

// Resolver es la raiz de resolvers de GraphQL. Solo cablea los casos de uso; no contiene logica
// de negocio (los resolvers delegan y mapean).
type Resolver struct {
	auth     *usecase.AuthUseCase
	users    *usecase.UserUseCase
	products *usecase.ProductUseCase
	orders   *usecase.OrderUseCase
}

// NewResolver construye la raiz de resolvers con los casos de uso inyectados.
func NewResolver(
	auth *usecase.AuthUseCase,
	users *usecase.UserUseCase,
	products *usecase.ProductUseCase,
	orders *usecase.OrderUseCase,
) *Resolver {
	return &Resolver{auth: auth, users: users, products: products, orders: orders}
}
