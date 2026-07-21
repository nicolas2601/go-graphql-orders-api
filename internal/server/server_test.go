package server_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/nicolas2601/go-graphql-orders-api/internal/auth"
	"github.com/nicolas2601/go-graphql-orders-api/internal/config"
	graphqldelivery "github.com/nicolas2601/go-graphql-orders-api/internal/delivery/graphql"
	"github.com/nicolas2601/go-graphql-orders-api/internal/domain"
	"github.com/nicolas2601/go-graphql-orders-api/internal/server"
)

func newHandler(t *testing.T, cfg config.Config) http.Handler {
	t.Helper()
	tokens, err := auth.NewJWTService("test-secret-0123456789", 15*time.Minute, 24*time.Hour, nil)
	if err != nil {
		t.Fatalf("jwt: %v", err)
	}
	// Estos tests no invocan resolvers (introspection, playground, limites), asi que los casos de
	// uso pueden ser nil: las queries se rechazan antes de resolver o no tocan resolvers.
	var _ domain.TokenService = tokens
	resolver := graphqldelivery.NewResolver(nil, nil, nil, nil)
	return server.NewHandler(cfg, resolver, tokens)
}

func graphQLRequest(t *testing.T, query string) *http.Request {
	t.Helper()
	body, err := json.Marshal(map[string]string{"query": query})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/query", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	return req
}

func TestHealthz(t *testing.T) {
	rec := httptest.NewRecorder()
	newHandler(t, config.Config{AppEnv: "development"}).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "ok") {
		t.Fatalf("healthz: %d %s", rec.Code, rec.Body.String())
	}
}

func TestPlaygroundDisabledInProduction(t *testing.T) {
	rec := httptest.NewRecorder()
	newHandler(t, config.Config{AppEnv: "production", GraphQLPlayground: true}).
		ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("playground must be off in production, got %d", rec.Code)
	}
}

func TestIntrospectionGatedByEnvironment(t *testing.T) {
	const introspection = "{ __schema { queryType { name } } }"

	t.Run("enabled in development", func(t *testing.T) {
		rec := httptest.NewRecorder()
		newHandler(t, config.Config{AppEnv: "development"}).ServeHTTP(rec, graphQLRequest(t, introspection))
		if !strings.Contains(rec.Body.String(), "queryType") {
			t.Fatalf("introspection should work in development: %s", rec.Body.String())
		}
	})

	t.Run("disabled in production", func(t *testing.T) {
		rec := httptest.NewRecorder()
		newHandler(t, config.Config{AppEnv: "production"}).ServeHTTP(rec, graphQLRequest(t, introspection))
		body := rec.Body.String()
		if !strings.Contains(body, "errors") || strings.Contains(body, "queryType") {
			t.Fatalf("introspection should be rejected in production: %s", body)
		}
	})
}

func TestComplexityLimitRejectsExpensiveQuery(t *testing.T) {
	var sb strings.Builder
	sb.WriteString("query {")
	for i := 0; i < 150; i++ {
		fmt.Fprintf(&sb, " a%d: products { total }", i)
	}
	sb.WriteString(" }")

	rec := httptest.NewRecorder()
	newHandler(t, config.Config{AppEnv: "development"}).ServeHTTP(rec, graphQLRequest(t, sb.String()))
	if !strings.Contains(strings.ToLower(rec.Body.String()), "complexity") {
		t.Fatalf("expected a complexity error: %s", rec.Body.String())
	}
}

func TestRequestBodyLimitRejectsHugePayload(t *testing.T) {
	huge := `{"query":"` + strings.Repeat("#", (1<<20)+1024) + `"}`
	req := httptest.NewRequest(http.MethodPost, "/query", strings.NewReader(huge))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	newHandler(t, config.Config{AppEnv: "development"}).ServeHTTP(rec, req)
	body := strings.ToLower(rec.Body.String())
	if rec.Code == http.StatusOK && !strings.Contains(body, "error") {
		t.Fatalf("oversized body should be rejected, got %d: %s", rec.Code, rec.Body.String())
	}
}
