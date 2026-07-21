// Package server arma el handler HTTP: endpoint GraphQL con auth y controles anti-DoS, health check
// y, en desarrollo, el playground.
package server

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/99designs/gqlgen/graphql/playground"

	"github.com/nicolas2601/go-graphql-orders-api/internal/config"
	graphqldelivery "github.com/nicolas2601/go-graphql-orders-api/internal/delivery/graphql"
	"github.com/nicolas2601/go-graphql-orders-api/internal/delivery/graphql/generated"
	"github.com/nicolas2601/go-graphql-orders-api/internal/delivery/graphql/middleware"
	"github.com/nicolas2601/go-graphql-orders-api/internal/domain"
)

const (
	graphQLPath = "/query"
	// maxRequestBytes limita el tamano del body para evitar payloads abusivos.
	maxRequestBytes = 1 << 20 // 1 MiB
	// queryComplexityLimit acota la complejidad de una operacion GraphQL (anti-DoS).
	queryComplexityLimit = 200
)

// NewHandler arma el handler HTTP cableando los resolvers de GraphQL. La introspection y el
// playground solo se habilitan en modo desarrollo (fail-safe). El endpoint GraphQL va detras del
// middleware de auth y de un limite de tamano de body.
func NewHandler(cfg config.Config, resolver *graphqldelivery.Resolver, tokens domain.TokenService) http.Handler {
	schema := generated.NewExecutableSchema(generated.Config{Resolvers: resolver})

	gql := handler.New(schema)
	gql.AddTransport(transport.POST{})
	gql.Use(extension.FixedComplexityLimit(queryComplexityLimit))
	if cfg.IsDevelopment() {
		gql.Use(extension.Introspection{})
	}

	mux := http.NewServeMux()
	mux.Handle(graphQLPath, middleware.Auth(tokens)(http.MaxBytesHandler(gql, maxRequestBytes)))
	mux.HandleFunc("/healthz", handleHealth)
	if cfg.IsDevelopment() && cfg.GraphQLPlayground {
		mux.Handle("/", playground.Handler("Orders API", graphQLPath))
	}
	return mux
}

func handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]string{"status": "ok"}); err != nil {
		slog.Error("failed to write health response", "error", err)
	}
}
