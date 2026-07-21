package graphql

import (
	"context"

	"github.com/nicolas2601/go-graphql-orders-api/internal/delivery/graphql/middleware"
	"github.com/nicolas2601/go-graphql-orders-api/internal/delivery/graphql/ratelimit"
	"github.com/nicolas2601/go-graphql-orders-api/internal/usecase"
)

// Resolver es la raiz de resolvers de GraphQL. Solo cablea los casos de uso; no contiene logica
// de negocio (los resolvers delegan y mapean).
type Resolver struct {
	auth        *usecase.AuthUseCase
	users       *usecase.UserUseCase
	products    *usecase.ProductUseCase
	orders      *usecase.OrderUseCase
	authLimiter *ratelimit.Limiter
}

// NewResolver construye la raiz de resolvers con los casos de uso inyectados. authLimiter puede ser
// nil (sin rate limiting, util en tests).
func NewResolver(
	auth *usecase.AuthUseCase,
	users *usecase.UserUseCase,
	products *usecase.ProductUseCase,
	orders *usecase.OrderUseCase,
	authLimiter *ratelimit.Limiter,
) *Resolver {
	return &Resolver{auth: auth, users: users, products: products, orders: orders, authLimiter: authLimiter}
}

// allowAuth aplica el rate limiting por IP a las operaciones de autenticacion (login/register),
// para frenar fuerza bruta. Sin limiter configurado, no limita.
func (r *Resolver) allowAuth(ctx context.Context) error {
	if r.authLimiter == nil {
		return nil
	}
	if !r.authLimiter.Allow(middleware.ClientIPFromContext(ctx)) {
		return rateLimitedError()
	}
	return nil
}
