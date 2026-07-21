package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/nicolas2601/go-graphql-orders-api/internal/domain"
	"github.com/nicolas2601/go-graphql-orders-api/internal/usecase"
)

type fakeProductRepo struct {
	items       []domain.Product
	total       int
	listErr     error
	gotPage     int
	gotPageSize int
	byID        map[string]domain.Product
}

func (f *fakeProductRepo) List(_ context.Context, _ domain.ProductFilter, page, pageSize int) ([]domain.Product, int, error) {
	f.gotPage, f.gotPageSize = page, pageSize
	return f.items, f.total, f.listErr
}

func (f *fakeProductRepo) GetByID(_ context.Context, id string) (domain.Product, error) {
	p, ok := f.byID[id]
	if !ok {
		return domain.Product{}, domain.ErrProductNotFound
	}
	return p, nil
}

func (f *fakeProductRepo) FindByIDs(context.Context, []string) (map[string]domain.Product, error) {
	return nil, nil
}
func (f *fakeProductRepo) DecrementStock(context.Context, string, int) error { return nil }
func (f *fakeProductRepo) IncrementStock(context.Context, string, int) error { return nil }

func TestProductList(t *testing.T) {
	ctx := context.Background()

	t.Run("returns a page with items and total", func(t *testing.T) {
		repo := &fakeProductRepo{
			items: []domain.Product{{ID: "p1"}, {ID: "p2"}},
			total: 7,
		}
		uc := usecase.NewProductUseCase(repo)

		page, err := uc.List(ctx, domain.ProductFilter{}, 2, 20)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(page.Items) != 2 || page.Total != 7 || page.Page != 2 || page.PageSize != 20 {
			t.Fatalf("unexpected page: %+v", page)
		}
	})

	t.Run("clamps pagination parameters before hitting the repository", func(t *testing.T) {
		cases := []struct {
			name                   string
			page, pageSize         int
			wantPage, wantPageSize int
		}{
			{"page below one becomes one", 0, 20, 1, 20},
			{"negative page becomes one", -5, 20, 1, 20},
			{"zero page size uses default", 1, 0, 1, 20},
			{"page size above max is capped", 1, 1000, 1, 100},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				repo := &fakeProductRepo{}
				uc := usecase.NewProductUseCase(repo)
				page, err := uc.List(ctx, domain.ProductFilter{}, tc.page, tc.pageSize)
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if repo.gotPage != tc.wantPage || repo.gotPageSize != tc.wantPageSize {
					t.Fatalf("repo got page=%d size=%d, want page=%d size=%d",
						repo.gotPage, repo.gotPageSize, tc.wantPage, tc.wantPageSize)
				}
				if page.Page != tc.wantPage || page.PageSize != tc.wantPageSize {
					t.Fatalf("page reports page=%d size=%d, want %d/%d", page.Page, page.PageSize, tc.wantPage, tc.wantPageSize)
				}
			})
		}
	})

	t.Run("propagates repository error", func(t *testing.T) {
		sentinel := errors.New("db down")
		uc := usecase.NewProductUseCase(&fakeProductRepo{listErr: sentinel})
		if _, err := uc.List(ctx, domain.ProductFilter{}, 1, 20); !errors.Is(err, sentinel) {
			t.Fatalf("got %v, want sentinel", err)
		}
	})
}

func TestProductGet(t *testing.T) {
	ctx := context.Background()
	repo := &fakeProductRepo{byID: map[string]domain.Product{"p1": {ID: "p1", Name: "Keyboard"}}}
	uc := usecase.NewProductUseCase(repo)

	t.Run("existing product is returned", func(t *testing.T) {
		p, err := uc.Get(ctx, "p1")
		if err != nil || p.Name != "Keyboard" {
			t.Fatalf("got %+v, %v", p, err)
		}
	})

	t.Run("missing product returns ErrProductNotFound", func(t *testing.T) {
		if _, err := uc.Get(ctx, "ghost"); !errors.Is(err, domain.ErrProductNotFound) {
			t.Fatalf("got %v, want ErrProductNotFound", err)
		}
	})
}

func TestNewProductUseCasePanicsOnNilRepo(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic on nil repository")
		}
	}()
	usecase.NewProductUseCase(nil)
}
