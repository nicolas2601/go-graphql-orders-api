package main

import (
	"context"
	"net/http"
	"testing"
	"time"
)

func TestRunFailsWithoutDatabaseURL(t *testing.T) {
	// getenv vacio -> DATABASE_URL ausente -> run debe abortar con error, sin colgarse.
	if err := run(context.Background(), func(string) string { return "" }); err == nil {
		t.Fatal("expected an error when DATABASE_URL is not set")
	}
}

func TestServeGracefulShutdown(t *testing.T) {
	srv := &http.Server{
		Addr:              "127.0.0.1:0",
		Handler:           http.NewServeMux(),
		ReadHeaderTimeout: readHeaderTimeout,
	}

	ctx, cancel := context.WithCancel(context.Background())
	result := make(chan error, 1)
	go func() { result <- serve(ctx, srv) }()

	time.Sleep(100 * time.Millisecond) // dar tiempo a que el servidor empiece a escuchar
	cancel()                           // simula SIGINT/SIGTERM

	select {
	case err := <-result:
		if err != nil {
			t.Fatalf("graceful shutdown returned an error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("serve did not return after context cancellation")
	}
}

func TestNewLoggerFallsBackOnInvalidLevel(t *testing.T) {
	if newLogger("not-a-level") == nil {
		t.Fatal("expected a logger even for an invalid level")
	}
}
