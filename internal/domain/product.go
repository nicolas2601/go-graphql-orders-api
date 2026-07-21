package domain

import "time"

// maxNameLength acota el nombre de un producto para evitar payloads abusivos.
const maxNameLength = 200

// Product es un producto del catalogo.
type Product struct {
	ID        string
	Name      string
	Price     float64
	Stock     int
	CreatedAt time.Time
}

// NewProduct construye un producto validando sus invariantes.
func NewProduct(id, name string, price float64, stock int, createdAt time.Time) (Product, error) {
	p := Product{ID: id, Name: name, Price: price, Stock: stock, CreatedAt: createdAt}
	if err := p.Validate(); err != nil {
		return Product{}, err
	}
	return p, nil
}

// Validate verifica las invariantes de un producto.
func (p Product) Validate() error {
	if p.Name == "" || len(p.Name) > maxNameLength {
		return ErrInvalidName
	}
	if p.Price <= 0 {
		return ErrInvalidPrice
	}
	if p.Stock < 0 {
		return ErrInvalidStock
	}
	return nil
}
