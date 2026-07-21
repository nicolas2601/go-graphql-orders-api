package usecase

import (
	"context"

	"github.com/nicolas2601/go-graphql-orders-api/internal/domain"
)

// ProductUseCase expone la lectura del catalogo de productos. Los productos son de solo lectura
// para los usuarios (no hay operaciones de escritura en la API); se cargan por seed.
type ProductUseCase struct {
	products domain.ProductRepository
}

// NewProductUseCase construye el caso de uso con el repositorio inyectado.
func NewProductUseCase(products domain.ProductRepository) *ProductUseCase {
	if products == nil {
		panic("usecase: product repository must not be nil")
	}
	return &ProductUseCase{products: products}
}

// List devuelve una pagina de productos que cumplen el filtro. Acota la paginacion antes de
// consultar el repositorio.
func (uc *ProductUseCase) List(ctx context.Context, filter domain.ProductFilter, page, pageSize int) (Page[domain.Product], error) {
	if err := validateFilter(filter); err != nil {
		return Page[domain.Product]{}, err
	}
	page, pageSize = normalizePagination(page, pageSize)
	items, total, err := uc.products.List(ctx, filter, page, pageSize)
	if err != nil {
		return Page[domain.Product]{}, err
	}
	return Page[domain.Product]{Items: items, Total: total, Page: page, PageSize: pageSize}, nil
}

// validateFilter rechaza un filtro incoherente (precios negativos o rango invertido) en la capa de
// aplicacion, para no dejar a cada repositorio decidir que hacer con un rango sin sentido.
func validateFilter(f domain.ProductFilter) error {
	if f.MinPrice != nil && *f.MinPrice < 0 {
		return domain.ErrInvalidFilter
	}
	if f.MaxPrice != nil && *f.MaxPrice < 0 {
		return domain.ErrInvalidFilter
	}
	if f.MinPrice != nil && f.MaxPrice != nil && *f.MinPrice > *f.MaxPrice {
		return domain.ErrInvalidFilter
	}
	return nil
}

// Get devuelve un producto por id, o ErrProductNotFound si no existe.
func (uc *ProductUseCase) Get(ctx context.Context, id string) (domain.Product, error) {
	return uc.products.GetByID(ctx, id)
}
