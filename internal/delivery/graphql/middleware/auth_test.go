package middleware_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nicolas2601/go-graphql-orders-api/internal/delivery/graphql/authctx"
	"github.com/nicolas2601/go-graphql-orders-api/internal/delivery/graphql/middleware"
)

// fakeTokens acepta solo el token "good"; cualquier otro es invalido.
type fakeTokens struct{}

func (fakeTokens) GenerateAccess(string) (string, error)  { return "", nil }
func (fakeTokens) GenerateRefresh(string) (string, error) { return "", nil }
func (fakeTokens) ParseRefresh(string) (string, error)    { return "", errors.New("n/a") }
func (fakeTokens) ParseAccess(token string) (string, error) {
	if token == "good" {
		return "user-1", nil
	}
	return "", errors.New("invalid token")
}

func TestAuthMiddleware(t *testing.T) {
	capture := func() (http.Handler, *string) {
		var got string
		h := middleware.Auth(fakeTokens{})(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
			if id, ok := authctx.UserID(r.Context()); ok {
				got = id
			}
		}))
		return h, &got
	}

	t.Run("valid token attaches the user to the context", func(t *testing.T) {
		h, got := capture()
		req := httptest.NewRequest(http.MethodPost, "/query", nil)
		req.Header.Set("Authorization", "Bearer good")
		h.ServeHTTP(httptest.NewRecorder(), req)
		if *got != "user-1" {
			t.Fatalf("expected user-1, got %q", *got)
		}
	})

	t.Run("invalid token does not attach a user (fails closed)", func(t *testing.T) {
		h, got := capture()
		req := httptest.NewRequest(http.MethodPost, "/query", nil)
		req.Header.Set("Authorization", "Bearer garbage.token.here")
		h.ServeHTTP(httptest.NewRecorder(), req)
		if *got != "" {
			t.Fatalf("an invalid token must not attach a user, got %q", *got)
		}
	})

	t.Run("missing or non-bearer header attaches no user", func(t *testing.T) {
		for _, header := range []string{"", "Basic abc", "good"} {
			h, got := capture()
			req := httptest.NewRequest(http.MethodPost, "/query", nil)
			if header != "" {
				req.Header.Set("Authorization", header)
			}
			h.ServeHTTP(httptest.NewRecorder(), req)
			if *got != "" {
				t.Fatalf("header %q should not attach a user, got %q", header, *got)
			}
		}
	})
}
