package authctx_test

import (
	"context"
	"testing"

	"github.com/nicolas2601/go-graphql-orders-api/internal/delivery/graphql/authctx"
)

func TestUserRoundTrip(t *testing.T) {
	ctx := authctx.WithUser(context.Background(), "user-1")
	id, ok := authctx.UserID(ctx)
	if !ok || id != "user-1" {
		t.Fatalf("got %q ok=%v, want user-1 true", id, ok)
	}
}

func TestUserAbsentOrEmpty(t *testing.T) {
	if _, ok := authctx.UserID(context.Background()); ok {
		t.Fatal("expected no user in an empty context")
	}
	// Un id vacio se trata como ausente.
	if _, ok := authctx.UserID(authctx.WithUser(context.Background(), "")); ok {
		t.Fatal("an empty user id must be treated as absent")
	}
}
