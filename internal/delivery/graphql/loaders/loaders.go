// Package loaders provee DataLoaders por request para resolver Order.user y OrderItem.product sin
// caer en el problema N+1: las cargas individuales se agrupan (batch) en una sola consulta.
package loaders

import (
	"context"
	"errors"
	"net/http"

	"github.com/vikstrous/dataloadgen"

	"github.com/nicolas2601/go-graphql-orders-api/internal/domain"
)

type ctxKey struct{}

var errNoLoaders = errors.New("loaders: not present in context")

// Loaders agrupa los DataLoaders de una request. Se crean frescos por request para no compartir
// cache entre usuarios distintos (evita fugas de datos).
type Loaders struct {
	users    *dataloadgen.Loader[string, domain.User]
	products *dataloadgen.Loader[string, domain.Product]
}

func newLoaders(users domain.UserRepository, products domain.ProductRepository) *Loaders {
	return &Loaders{
		users: dataloadgen.NewMappedLoader(func(ctx context.Context, ids []string) (map[string]domain.User, error) {
			return users.FindByIDs(ctx, ids)
		}),
		products: dataloadgen.NewMappedLoader(func(ctx context.Context, ids []string) (map[string]domain.Product, error) {
			return products.FindByIDs(ctx, ids)
		}),
	}
}

// Middleware inyecta un juego de loaders fresco en el contexto de cada request.
func Middleware(users domain.UserRepository, products domain.ProductRepository) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := context.WithValue(r.Context(), ctxKey{}, newLoaders(users, products))
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func from(ctx context.Context) (*Loaders, error) {
	l, ok := ctx.Value(ctxKey{}).(*Loaders)
	if !ok {
		return nil, errNoLoaders
	}
	return l, nil
}

// LoadUser carga un usuario por id de forma batcheada. NO es un limite de autorizacion: solo hace
// fetch por id. La autorizacion (que la orden sea del caller) es responsabilidad de la capa de
// resolvers, que ya la aplica antes de que un UserID ajeno pueda llegar hasta aca.
func LoadUser(ctx context.Context, id string) (domain.User, error) {
	l, err := from(ctx)
	if err != nil {
		return domain.User{}, err
	}
	user, err := l.users.Load(ctx, id)
	if errors.Is(err, dataloadgen.ErrNotFound) {
		return domain.User{}, domain.ErrUserNotFound
	}
	return user, err
}

// LoadProduct carga un producto por id de forma batcheada. Tampoco autoriza (ver LoadUser).
func LoadProduct(ctx context.Context, id string) (domain.Product, error) {
	l, err := from(ctx)
	if err != nil {
		return domain.Product{}, err
	}
	product, err := l.products.Load(ctx, id)
	if errors.Is(err, dataloadgen.ErrNotFound) {
		return domain.Product{}, domain.ErrProductNotFound
	}
	return product, err
}
