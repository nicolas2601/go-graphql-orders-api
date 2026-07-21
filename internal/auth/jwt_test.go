package auth_test

import (
	"testing"
	"time"

	"github.com/nicolas2601/go-graphql-orders-api/internal/auth"
)

func fixedClock(t time.Time) func() time.Time { return func() time.Time { return t } }

func TestJWTService(t *testing.T) {
	base := time.Date(2026, time.July, 21, 12, 0, 0, 0, time.UTC)
	svc := auth.NewJWTService("test-secret", 15*time.Minute, 24*time.Hour, fixedClock(base))

	t.Run("access token round-trips the user id", func(t *testing.T) {
		token, err := svc.GenerateAccess("user-1")
		if err != nil {
			t.Fatalf("generate: %v", err)
		}
		got, err := svc.ParseAccess(token)
		if err != nil {
			t.Fatalf("parse: %v", err)
		}
		if got != "user-1" {
			t.Fatalf("got %q, want user-1", got)
		}
	})

	t.Run("access token is rejected when parsed as refresh", func(t *testing.T) {
		token, _ := svc.GenerateAccess("user-1")
		if _, err := svc.ParseRefresh(token); err == nil {
			t.Fatal("an access token must not be accepted as a refresh token")
		}
	})

	t.Run("refresh token round-trips", func(t *testing.T) {
		token, _ := svc.GenerateRefresh("user-1")
		got, err := svc.ParseRefresh(token)
		if err != nil || got != "user-1" {
			t.Fatalf("got %q, err %v", got, err)
		}
	})

	t.Run("expired token is rejected", func(t *testing.T) {
		token, _ := svc.GenerateAccess("user-1")
		// un verificador cuyo reloj esta despues de la expiracion del access (15m)
		expired := auth.NewJWTService("test-secret", 15*time.Minute, 24*time.Hour, fixedClock(base.Add(time.Hour)))
		if _, err := expired.ParseAccess(token); err == nil {
			t.Fatal("expired token should be rejected")
		}
	})

	t.Run("token signed with another secret is rejected", func(t *testing.T) {
		token, _ := svc.GenerateAccess("user-1")
		other := auth.NewJWTService("different-secret", 15*time.Minute, 24*time.Hour, fixedClock(base))
		if _, err := other.ParseAccess(token); err == nil {
			t.Fatal("token with wrong signature should be rejected")
		}
	})
}
