package graphql

import (
	"context"
	"testing"

	"github.com/nicolas2601/go-graphql-orders-api/internal/delivery/graphql/ratelimit"
)

func TestAllowAuth(t *testing.T) {
	t.Run("nil limiter always allows", func(t *testing.T) {
		r := &Resolver{}
		if err := r.allowAuth(context.Background()); err != nil {
			t.Fatalf("a nil limiter should allow: %v", err)
		}
	})

	t.Run("blocks once the per-key budget is exhausted", func(t *testing.T) {
		r := &Resolver{authLimiter: ratelimit.New(1)}
		if err := r.allowAuth(context.Background()); err != nil {
			t.Fatalf("first attempt should pass: %v", err)
		}
		if err := r.allowAuth(context.Background()); err == nil {
			t.Fatal("second attempt should be rate limited")
		}
	})
}
