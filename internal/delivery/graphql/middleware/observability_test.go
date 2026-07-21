package middleware_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nicolas2601/go-graphql-orders-api/internal/delivery/graphql/middleware"
)

func TestRequestIDGeneratesAndPropagates(t *testing.T) {
	t.Run("generates an id when absent and echoes it", func(t *testing.T) {
		var seen string
		h := middleware.RequestID(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
			seen = middleware.RequestIDFromContext(r.Context())
		}))
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
		if seen == "" || rec.Header().Get("X-Request-ID") != seen {
			t.Fatalf("request id not propagated: ctx=%q header=%q", seen, rec.Header().Get("X-Request-ID"))
		}
	})

	t.Run("keeps an incoming request id", func(t *testing.T) {
		h := middleware.RequestID(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {}))
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("X-Request-ID", "abc-123")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Header().Get("X-Request-ID") != "abc-123" {
			t.Fatalf("incoming id not kept: %q", rec.Header().Get("X-Request-ID"))
		}
	})
}

func TestClientIPExtractsFromRequest(t *testing.T) {
	t.Run("uses the last X-Forwarded-For hop (added by the trusted proxy), not a spoofable one", func(t *testing.T) {
		var ip string
		h := middleware.ClientIP(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
			ip = middleware.ClientIPFromContext(r.Context())
		}))
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		// El atacante manda "1.2.3.4" (spoof); el proxy confiable agrega la IP real al final.
		req.Header.Set("X-Forwarded-For", "1.2.3.4, 203.0.113.7")
		h.ServeHTTP(httptest.NewRecorder(), req)
		if ip != "203.0.113.7" {
			t.Fatalf("got %q, want 203.0.113.7 (the real client added by the proxy, not the spoofed first hop)", ip)
		}
	})

	t.Run("falls back to RemoteAddr host", func(t *testing.T) {
		var ip string
		h := middleware.ClientIP(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
			ip = middleware.ClientIPFromContext(r.Context())
		}))
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "192.0.2.5:54321"
		h.ServeHTTP(httptest.NewRecorder(), req)
		if ip != "192.0.2.5" {
			t.Fatalf("got %q, want 192.0.2.5", ip)
		}
	})
}

func TestRecoverCatchesPanic(t *testing.T) {
	h := middleware.Recover(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		panic("boom")
	}))
	rec := httptest.NewRecorder()
	// No debe propagar el panic; debe responder 500.
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/query", nil))
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("got %d, want 500", rec.Code)
	}
}

func TestContextHelpersReturnEmptyWhenAbsent(t *testing.T) {
	if middleware.RequestIDFromContext(context.Background()) != "" {
		t.Fatal("expected empty request id")
	}
	if middleware.ClientIPFromContext(context.Background()) != "" {
		t.Fatal("expected empty client ip")
	}
}
