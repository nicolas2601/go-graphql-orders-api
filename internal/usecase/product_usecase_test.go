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
	gotFilter   domain.ProductFilter
	gotPage     int
	gotPageSize int
	byID        map[string]domain.Product
}

func (f *fakeProductRepo) List(_ context.Context, filter domain.ProductFilter, page, pageSize int) ([]domain.Product, int, error) {
	f.gotFilter = filter
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

	t.Run("returns a page with items and total and passes the filter through", func(t *testing.T) {
		repo := &fakeProductRepo{
			items: []domain.Product{{ID: "p1"}, {ID: "p2"}},
			total: 7,
		}
		uc := usecase.NewProductUseCase(repo)

		name := "keyboard"
		page, err := uc.List(ctx, domain.ProductFilter{Name: &name}, 2, 20)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(page.Items) != 2 || page.Total != 7 || page.Page != 2 || page.PageSize != 20 {
			t.Fatalf("unexpected page: %+v", page)
		}
		if repo.gotFilter.Name == nil || *repo.gotFilter.Name != "keyboard" {
			t.Fatalf("filter was not passed through intact: %+v", repo.gotFilter)
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
			{"page size at max stays", 1, 100, 1, 100},
			{"page size just above max is capped", 1, 101, 1, 100},
			{"page size far above max is capped", 1, 1000, 1, 100},
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

	t.Run("invalid filter returns ErrInvalidFilter and does not hit the repository", func(t *testing.T) {
		neg := -1.0
		lo, hi := 100.0, 10.0
		cases := map[string]domain.ProductFilter{
			"negative min":  {MinPrice: &neg},
			"negative max":  {MaxPrice: &neg},
			"min above max": {MinPrice: &lo, MaxPrice: &hi},
		}
		for name, filter := range cases {
			t.Run(name, func(t *testing.T) {
				repo := &fakeProductRepo{}
				uc := usecase.NewProductUseCase(repo)
				if _, err := uc.List(ctx, filter, 1, 20); !errors.Is(err, domain.ErrInvalidFilter) {
					t.Fatalf("got %v, want ErrInvalidFilter", err)
				}
				if repo.gotPageSize != 0 {
					t.Fatal("repository should not be called on an invalid filter")
				}
			})
		}
	})
}

func TestPageHelpers(t *testing.T) {
	t.Run("total pages rounds up", func(t *testing.T) {
		p := usecase.Page[int]{Total: 21, Page: 1, PageSize: 10}
		if p.TotalPages() != 3 {
			t.Fatalf("got %d, want 3", p.TotalPages())
		}
	})
	t.Run("zero page size yields zero pages", func(t *testing.T) {
		p := usecase.Page[int]{Total: 5, PageSize: 0}
		if p.TotalPages() != 0 {
			t.Fatalf("got %d, want 0", p.TotalPages())
		}
	})
	t.Run("has next page when not on the last page", func(t *testing.T) {
		if !(usecase.Page[int]{Total: 21, Page: 1, PageSize: 10}).HasNextPage() {
			t.Fatal("page 1 of 3 should have a next page")
		}
	})
	t.Run("no next page on the last page", func(t *testing.T) {
		if (usecase.Page[int]{Total: 21, Page: 3, PageSize: 10}).HasNextPage() {
			t.Fatal("last page should not have a next page")
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
