package ratelimit_test

import (
	"testing"

	"github.com/nicolas2601/go-graphql-orders-api/internal/delivery/graphql/ratelimit"
)

func TestLimiterAllowsUpToBurstThenBlocks(t *testing.T) {
	l := ratelimit.New(3) // burst 3 por clave

	for i := 0; i < 3; i++ {
		if !l.Allow("1.2.3.4") {
			t.Fatalf("request %d should be allowed", i+1)
		}
	}
	if l.Allow("1.2.3.4") {
		t.Fatal("the 4th request should be blocked")
	}
	// Otra clave (IP) tiene su propio bucket.
	if !l.Allow("5.6.7.8") {
		t.Fatal("a different key should have its own budget")
	}
}

func TestLimiterHandlesNonPositiveRate(t *testing.T) {
	l := ratelimit.New(0) // se normaliza a 1
	if !l.Allow("k") {
		t.Fatal("first request should be allowed")
	}
	if l.Allow("k") {
		t.Fatal("second request should be blocked with rate 1")
	}
}
