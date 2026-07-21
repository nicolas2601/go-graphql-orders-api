package loaders

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/nicolas2601/go-graphql-orders-api/internal/domain"
)

// spyUserRepo cuenta las llamadas a FindByIDs del loader de usuarios.
type spyUserRepo struct {
	mu    sync.Mutex
	calls int
	users map[string]domain.User
}

func (s *spyUserRepo) FindByIDs(_ context.Context, ids []string) (map[string]domain.User, error) {
	s.mu.Lock()
	s.calls++
	s.mu.Unlock()
	out := make(map[string]domain.User, len(ids))
	for _, id := range ids {
		if u, ok := s.users[id]; ok {
			out[id] = u
		}
	}
	return out, nil
}
func (s *spyUserRepo) Create(context.Context, domain.User) error { return nil }
func (s *spyUserRepo) GetByEmail(context.Context, string) (domain.User, error) {
	return domain.User{}, nil
}
func (s *spyUserRepo) GetByID(context.Context, string) (domain.User, error) {
	return domain.User{}, nil
}

// spyProductRepo cuenta las llamadas a FindByIDs para verificar el batching.
type spyProductRepo struct {
	mu       sync.Mutex
	calls    int
	fetchErr error
	products map[string]domain.Product
}

func (s *spyProductRepo) FindByIDs(_ context.Context, ids []string) (map[string]domain.Product, error) {
	s.mu.Lock()
	s.calls++
	s.mu.Unlock()
	if s.fetchErr != nil {
		return nil, s.fetchErr
	}
	out := make(map[string]domain.Product, len(ids))
	for _, id := range ids {
		if p, ok := s.products[id]; ok {
			out[id] = p
		}
	}
	return out, nil
}

func (s *spyProductRepo) List(context.Context, domain.ProductFilter, int, int) ([]domain.Product, int, error) {
	return nil, 0, nil
}
func (s *spyProductRepo) GetByID(context.Context, string) (domain.Product, error) {
	return domain.Product{}, nil
}
func (s *spyProductRepo) DecrementStock(context.Context, string, int) error { return nil }
func (s *spyProductRepo) IncrementStock(context.Context, string, int) error { return nil }

func TestProductLoaderBatchesConcurrentLoads(t *testing.T) {
	spy := &spyProductRepo{products: map[string]domain.Product{
		"a": {ID: "a", Name: "A"}, "b": {ID: "b", Name: "B"}, "c": {ID: "c", Name: "C"},
	}}
	ctx := context.WithValue(context.Background(), ctxKey{}, newLoaders(nil, spy))

	ids := []string{"a", "b", "c"}
	results := make([]domain.Product, len(ids))
	errs := make([]error, len(ids))
	var wg sync.WaitGroup
	wg.Add(len(ids))
	for i, id := range ids {
		go func(i int, id string) {
			defer wg.Done()
			results[i], errs[i] = LoadProduct(ctx, id)
		}(i, id)
	}
	wg.Wait()

	if spy.calls != 1 {
		t.Fatalf("expected the 3 concurrent loads to batch into 1 call, got %d", spy.calls)
	}
	for i, id := range ids {
		if errs[i] != nil || results[i].ID != id {
			t.Fatalf("load %q: got %+v err %v", id, results[i], errs[i])
		}
	}
}

func TestProductLoaderMissingKeyReturnsDomainNotFound(t *testing.T) {
	spy := &spyProductRepo{products: map[string]domain.Product{}}
	ctx := context.WithValue(context.Background(), ctxKey{}, newLoaders(nil, spy))

	// Un id ausente del map se traduce al error tipado del dominio (no al sentinel de la libreria),
	// para que el mapeo a extensions.code siga devolviendo PRODUCT_NOT_FOUND.
	if _, err := LoadProduct(ctx, "ghost"); !errors.Is(err, domain.ErrProductNotFound) {
		t.Fatalf("got %v, want ErrProductNotFound", err)
	}
}

func TestProductLoaderFetchErrorPropagatesToAllLoads(t *testing.T) {
	dbDown := errors.New("db down")
	spy := &spyProductRepo{fetchErr: dbDown}
	ctx := context.WithValue(context.Background(), ctxKey{}, newLoaders(nil, spy))

	ids := []string{"a", "b", "c"}
	errs := make([]error, len(ids))
	var wg sync.WaitGroup
	wg.Add(len(ids))
	for i, id := range ids {
		go func(i int, id string) {
			defer wg.Done()
			_, errs[i] = LoadProduct(ctx, id)
		}(i, id)
	}
	wg.Wait()

	for i, err := range errs {
		if !errors.Is(err, dbDown) {
			t.Fatalf("load %d: got %v, want the fetch error propagated", i, err)
		}
	}
}

func TestUserLoaderBatchesConcurrentLoads(t *testing.T) {
	spy := &spyUserRepo{users: map[string]domain.User{
		"u1": {ID: "u1", Email: "a@x.com"}, "u2": {ID: "u2", Email: "b@x.com"},
	}}
	ctx := context.WithValue(context.Background(), ctxKey{}, newLoaders(spy, nil))

	ids := []string{"u1", "u2", "u1"} // el id repetido tambien se coalesce
	var wg sync.WaitGroup
	wg.Add(len(ids))
	for _, id := range ids {
		go func(id string) {
			defer wg.Done()
			_, _ = LoadUser(ctx, id)
		}(id)
	}
	wg.Wait()

	if spy.calls != 1 {
		t.Fatalf("expected concurrent user loads to batch into 1 call, got %d", spy.calls)
	}
}

func TestUserLoaderMissingKeyReturnsDomainNotFound(t *testing.T) {
	spy := &spyUserRepo{users: map[string]domain.User{}}
	ctx := context.WithValue(context.Background(), ctxKey{}, newLoaders(spy, nil))
	if _, err := LoadUser(ctx, "ghost"); !errors.Is(err, domain.ErrUserNotFound) {
		t.Fatalf("got %v, want ErrUserNotFound", err)
	}
}

func TestLoadersRequireContext(t *testing.T) {
	if _, err := LoadProduct(context.Background(), "a"); err == nil {
		t.Fatal("expected an error when loaders are not in the context")
	}
	if _, err := LoadUser(context.Background(), "a"); err == nil {
		t.Fatal("expected an error when loaders are not in the context")
	}
}

func TestMiddlewareInjectsLoaders(t *testing.T) {
	spy := &spyProductRepo{products: map[string]domain.Product{"a": {ID: "a"}}}
	var loaded domain.Product
	var loadErr error

	handler := Middleware(nil, spy)(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		loaded, loadErr = LoadProduct(r.Context(), "a")
	}))
	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/query", nil))

	if loadErr != nil || loaded.ID != "a" {
		t.Fatalf("middleware did not wire loaders: %+v %v", loaded, loadErr)
	}
}
