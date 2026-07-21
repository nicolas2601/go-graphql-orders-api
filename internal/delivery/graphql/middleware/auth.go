// Package middleware provee middlewares HTTP para la capa GraphQL.
package middleware

import (
	"net/http"
	"strings"

	"github.com/nicolas2601/go-graphql-orders-api/internal/delivery/graphql/authctx"
	"github.com/nicolas2601/go-graphql-orders-api/internal/domain"
)

// Auth adjunta la identidad del usuario al contexto si el request trae un access token valido en
// el header Authorization: Bearer <token>. No rechaza requests sin token: la autorizacion la
// deciden los resolvers (asi register/login siguen siendo publicos).
func Auth(tokens domain.TokenService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			if token := bearerToken(r.Header.Get("Authorization")); token != "" {
				if userID, err := tokens.ParseAccess(token); err == nil {
					ctx = authctx.WithUser(ctx, userID)
				}
			}
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func bearerToken(header string) string {
	const prefix = "Bearer "
	if strings.HasPrefix(header, prefix) {
		return strings.TrimSpace(strings.TrimPrefix(header, prefix))
	}
	return ""
}
