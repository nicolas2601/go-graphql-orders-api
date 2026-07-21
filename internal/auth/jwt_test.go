package auth_test

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/nicolas2601/go-graphql-orders-api/internal/auth"
)

const testSecret = "test-secret-0123456789" // >= 16 bytes

func fixedClock(t time.Time) func() time.Time { return func() time.Time { return t } }

func mustService(t *testing.T, secret string, clock func() time.Time) *auth.JWTService {
	t.Helper()
	svc, err := auth.NewJWTService(secret, 15*time.Minute, 24*time.Hour, clock)
	if err != nil {
		t.Fatalf("NewJWTService: %v", err)
	}
	return svc
}

func TestNewJWTServiceRejectsWeakSecret(t *testing.T) {
	for _, secret := range []string{"", "short"} {
		if _, err := auth.NewJWTService(secret, time.Minute, time.Hour, nil); err == nil {
			t.Fatalf("secret %q should be rejected", secret)
		}
	}
}

func TestJWTService(t *testing.T) {
	base := time.Date(2026, time.July, 21, 12, 0, 0, 0, time.UTC)
	svc := mustService(t, testSecret, fixedClock(base))

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
		expired := mustService(t, testSecret, fixedClock(base.Add(time.Hour)))
		if _, err := expired.ParseAccess(token); err == nil {
			t.Fatal("expired token should be rejected")
		}
	})

	t.Run("token signed with another secret is rejected", func(t *testing.T) {
		token, _ := svc.GenerateAccess("user-1")
		other := mustService(t, "different-secret-0123456789", fixedClock(base))
		if _, err := other.ParseAccess(token); err == nil {
			t.Fatal("token with wrong signature should be rejected")
		}
	})

	// Regresion de seguridad: un token con alg "none" (o cualquier metodo distinto de HS256) debe
	// rechazarse, para evitar el ataque clasico de confusion de algoritmo.
	t.Run("token with alg none is rejected", func(t *testing.T) {
		unsigned := jwt.NewWithClaims(jwt.SigningMethodNone, jwt.MapClaims{
			"sub": "user-1",
			"typ": "access",
			"exp": base.Add(time.Hour).Unix(),
		})
		token, err := unsigned.SignedString(jwt.UnsafeAllowNoneSignatureType)
		if err != nil {
			t.Fatalf("sign none: %v", err)
		}
		if _, err := svc.ParseAccess(token); err == nil {
			t.Fatal("alg=none token must be rejected")
		}
	})
}
