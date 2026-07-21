package graphql

import (
	"context"
	"errors"
	"log/slog"

	"github.com/vektah/gqlparser/v2/gqlerror"

	"github.com/nicolas2601/go-graphql-orders-api/internal/domain"
)

// codeInternal es el codigo para errores inesperados: no se filtra el detalle al cliente.
const codeInternal = "INTERNAL_ERROR"

// rateLimitedError es el error que se devuelve cuando se supera el limite de tasa de auth.
func rateLimitedError() error {
	return &gqlerror.Error{
		Message:    "too many requests",
		Extensions: map[string]any{"code": "RATE_LIMITED"},
	}
}

// domainErrorCodes mapea los errores tipados del dominio a su codigo estable de GraphQL.
var domainErrorCodes = []struct {
	err  error
	code string
}{
	{domain.ErrUnauthenticated, "UNAUTHENTICATED"},
	{domain.ErrInvalidCredentials, "INVALID_CREDENTIALS"},
	{domain.ErrEmailAlreadyRegistered, "EMAIL_ALREADY_REGISTERED"},
	{domain.ErrInvalidEmail, "INVALID_EMAIL"},
	{domain.ErrInvalidPassword, "INVALID_PASSWORD"},
	{domain.ErrUserNotFound, "USER_NOT_FOUND"},
	{domain.ErrProductNotFound, "PRODUCT_NOT_FOUND"},
	{domain.ErrInvalidName, "INVALID_NAME"},
	{domain.ErrInvalidPrice, "INVALID_PRICE"},
	{domain.ErrInvalidStock, "INVALID_STOCK"},
	{domain.ErrInsufficientStock, "INSUFFICIENT_STOCK"},
	{domain.ErrInvalidFilter, "INVALID_FILTER"},
	{domain.ErrEmptyOrder, "EMPTY_ORDER"},
	{domain.ErrInvalidQuantity, "INVALID_QUANTITY"},
	{domain.ErrOrderNotFound, "ORDER_NOT_FOUND"},
	{domain.ErrOrderNotOwned, "ORDER_NOT_OWNED"},
	{domain.ErrOrderNotPending, "ORDER_NOT_PENDING"},
}

// toGraphQLError traduce un error del dominio a un error GraphQL con extensions.code. Los errores
// no reconocidos se enmascaran como error interno (sin filtrar detalle) y se loguean.
func toGraphQLError(ctx context.Context, err error) error {
	for _, mapping := range domainErrorCodes {
		if errors.Is(err, mapping.err) {
			return &gqlerror.Error{
				Message:    err.Error(),
				Extensions: map[string]any{"code": mapping.code},
			}
		}
	}
	slog.ErrorContext(ctx, "unhandled internal error", "error", err)
	return &gqlerror.Error{
		Message:    "internal server error",
		Extensions: map[string]any{"code": codeInternal},
	}
}
