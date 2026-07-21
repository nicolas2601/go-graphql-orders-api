package domain

import "errors"

// Errores tipados del dominio (sentinels). Representan resultados de negocio esperados y se
// mapean a codigos estables de GraphQL en la capa de delivery. Los errores inesperados no
// usan estos sentinels y se enmascaran como error interno.
var (
	// Autenticacion / usuarios.
	ErrUnauthenticated        = errors.New("unauthenticated")
	ErrInvalidCredentials     = errors.New("invalid credentials")
	ErrEmailAlreadyRegistered = errors.New("email already registered")
	ErrInvalidEmail           = errors.New("invalid email")
	ErrInvalidPassword        = errors.New("invalid password")
	ErrUserNotFound           = errors.New("user not found")

	// Catalogo de productos.
	ErrProductNotFound   = errors.New("product not found")
	ErrInvalidName       = errors.New("invalid name")
	ErrInvalidPrice      = errors.New("invalid price")
	ErrInvalidStock      = errors.New("invalid stock")
	ErrInsufficientStock = errors.New("insufficient stock")
	ErrInvalidFilter     = errors.New("invalid product filter")

	// Ordenes.
	ErrEmptyOrder      = errors.New("order must have at least one item")
	ErrInvalidQuantity = errors.New("invalid quantity")
	ErrOrderNotFound   = errors.New("order not found")
	ErrOrderNotOwned   = errors.New("order not owned by user")
	ErrOrderNotPending = errors.New("order is not pending")
)
