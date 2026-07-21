// Command orders-api es el punto de entrada: carga la configuracion, conecta a PostgreSQL, corre
// las migraciones y el seed, cablea todas las dependencias y sirve la API GraphQL con apagado ordenado.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/nicolas2601/go-graphql-orders-api/internal/auth"
	"github.com/nicolas2601/go-graphql-orders-api/internal/config"
	graphqldelivery "github.com/nicolas2601/go-graphql-orders-api/internal/delivery/graphql"
	"github.com/nicolas2601/go-graphql-orders-api/internal/delivery/graphql/ratelimit"
	"github.com/nicolas2601/go-graphql-orders-api/internal/repository/postgres"
	"github.com/nicolas2601/go-graphql-orders-api/internal/server"
	"github.com/nicolas2601/go-graphql-orders-api/internal/usecase"
)

const (
	readHeaderTimeout = 5 * time.Second
	readTimeout       = 15 * time.Second
	writeTimeout      = 15 * time.Second
	idleTimeout       = 60 * time.Second
	shutdownTimeout   = 10 * time.Second
)

func main() {
	if err := run(context.Background(), os.Getenv); err != nil {
		slog.Error("orders-api failed", "error", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, getenv func(string) string) error {
	cfg := config.Load(getenv)
	slog.SetDefault(newLogger(cfg.LogLevel))

	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	if cfg.DatabaseURL == "" {
		return errors.New("DATABASE_URL is required")
	}
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("connect to database: %w", err)
	}
	defer pool.Close()

	if err := postgres.RunMigrations(cfg.DatabaseURL); err != nil {
		return err
	}
	if cfg.SeedProducts {
		if err := postgres.SeedProducts(ctx, pool); err != nil {
			return err
		}
	}

	handler, err := buildHandler(cfg, pool)
	if err != nil {
		return err
	}

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           handler,
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
	}
	return serve(ctx, srv)
}

// buildHandler cablea infraestructura, casos de uso y resolvers, y arma el handler HTTP.
func buildHandler(cfg config.Config, pool *pgxpool.Pool) (http.Handler, error) {
	tokens, err := auth.NewJWTService(cfg.JWTSecret, cfg.JWTAccessTTL, cfg.JWTRefreshTTL, time.Now)
	if err != nil {
		return nil, fmt.Errorf("token service: %w", err)
	}
	hasher := auth.NewBcryptHasher(cfg.BcryptCost)

	users := postgres.NewUserRepository(pool)
	products := postgres.NewProductRepository(pool)
	orders := postgres.NewOrderRepository(pool)
	tx := postgres.NewTxManager(pool)

	newID := uuid.NewString
	now := func() time.Time { return time.Now().UTC() }

	resolver := graphqldelivery.NewResolver(
		usecase.NewAuthUseCase(users, hasher, tokens, newID, now),
		usecase.NewUserUseCase(users),
		usecase.NewProductUseCase(products),
		usecase.NewOrderUseCase(orders, products, tx, newID, now),
		ratelimit.New(cfg.AuthRateLimit),
	)

	return server.NewHandler(server.Deps{
		Config:   cfg,
		Resolver: resolver,
		Tokens:   tokens,
		Users:    users,
		Products: products,
		Ready:    pool.Ping,
	}), nil
}

// serve arranca el servidor y hace apagado ordenado cuando se cancela el contexto (SIGINT/SIGTERM).
func serve(ctx context.Context, srv *http.Server) error {
	errCh := make(chan error, 1)
	go func() {
		slog.Info("orders-api listening", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		slog.Info("shutting down")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	}
}

func newLogger(level string) *slog.Logger {
	var lvl slog.Level
	if err := lvl.UnmarshalText([]byte(level)); err != nil {
		lvl = slog.LevelInfo
	}
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: lvl}))
}
