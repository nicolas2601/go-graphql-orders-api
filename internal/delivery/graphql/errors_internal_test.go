package graphql

import (
	"context"
	"errors"
	"testing"

	"github.com/vektah/gqlparser/v2/gqlerror"

	"github.com/nicolas2601/go-graphql-orders-api/internal/domain"
)

func codeOf(t *testing.T, err error) string {
	t.Helper()
	var gqlErr *gqlerror.Error
	if !errors.As(err, &gqlErr) {
		t.Fatalf("expected a gqlerror, got %T", err)
	}
	code, _ := gqlErr.Extensions["code"].(string)
	return code
}

func TestToGraphQLError(t *testing.T) {
	ctx := context.Background()

	t.Run("maps a known domain error to its code", func(t *testing.T) {
		err := toGraphQLError(ctx, domain.ErrInsufficientStock)
		if codeOf(t, err) != "INSUFFICIENT_STOCK" {
			t.Fatalf("got %q", codeOf(t, err))
		}
	})

	t.Run("maps a wrapped domain error via errors.Is", func(t *testing.T) {
		wrapped := errors.Join(errors.New("context"), domain.ErrOrderNotOwned)
		if codeOf(t, toGraphQLError(ctx, wrapped)) != "ORDER_NOT_OWNED" {
			t.Fatalf("wrapped error not unwrapped")
		}
	})

	t.Run("masks an unknown error as internal without leaking its message", func(t *testing.T) {
		err := toGraphQLError(ctx, errors.New("connection string secret leaked"))
		var gqlErr *gqlerror.Error
		_ = errors.As(err, &gqlErr)
		if codeOf(t, err) != "INTERNAL_ERROR" {
			t.Fatalf("got code %q", codeOf(t, err))
		}
		if gqlErr.Message != "internal server error" {
			t.Fatalf("internal error message leaked: %q", gqlErr.Message)
		}
	})
}
