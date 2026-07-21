// Package server arma el handler HTTP: endpoint GraphQL con auth, DataLoaders, rate limiting y
// controles anti-DoS; health/readiness checks y, en desarrollo, el playground.
package server

import (
	"context"
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
	"github.com/nicolas2601/go-graphql-orders-api/internal/delivery/graphql/loaders"
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

// Deps agrupa las dependencias del handler HTTP.
type Deps struct {
	Config   config.Config
	Resolver *graphqldelivery.Resolver
	Tokens   domain.TokenService
	Users    domain.UserRepository
	Products domain.ProductRepository
	// Ready verifica que la app este lista (por ejemplo, ping a la base). Si es nil, /readyz da ok.
	Ready func(ctx context.Context) error
}

// NewHandler arma el handler HTTP. El endpoint GraphQL va detras de (de afuera hacia adentro):
// request-id, recover, limite de tamano de body, IP del cliente, auth y DataLoaders. La
// introspection y el playground solo se habilitan en desarrollo (fail-safe).
func NewHandler(deps Deps) http.Handler {
	schema := generated.NewExecutableSchema(generated.Config{Resolvers: deps.Resolver})

	gql := handler.New(schema)
	gql.AddTransport(transport.POST{})
	gql.Use(extension.FixedComplexityLimit(queryComplexityLimit))
	if deps.Config.IsDevelopment() {
		gql.Use(extension.Introspection{})
	}

	// MaxBytes como capa mas externa del endpoint: rechaza payloads abusivos antes de gastar CPU.
	endpoint := http.MaxBytesHandler(
		middleware.ClientIP(
			middleware.Auth(deps.Tokens)(
				loaders.Middleware(deps.Users, deps.Products)(gql))),
		maxRequestBytes)

	mux := http.NewServeMux()
	mux.Handle(graphQLPath, endpoint)
	mux.HandleFunc("/healthz", handleHealth)
	mux.HandleFunc("/readyz", readyHandler(deps.Ready))
	if deps.Config.IsDevelopment() && deps.Config.GraphQLPlayground {
		mux.Handle("/", playground.Handler("Orders API", graphQLPath))
	}

	// request-id y recover envuelven todo, para trazar y para que un panic no tumbe el proceso.
	return middleware.RequestID(middleware.Recover(mux))
}

func handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// readyHandler responde 200 si la app esta lista, o 503 si el check falla (readiness real).
func readyHandler(ready func(ctx context.Context) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if ready != nil {
			if err := ready(r.Context()); err != nil {
				writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "unavailable"})
				return
			}
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
	}
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		slog.Error("failed to write response", "error", err)
	}
}
