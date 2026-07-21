// Package authctx transporta la identidad del usuario autenticado por el contexto. El middleware
// de auth la setea a partir del token; los resolvers la leen para autorizar por dueno.
package authctx

import "context"

type userKey struct{}

// WithUser devuelve un contexto que lleva el id del usuario autenticado.
func WithUser(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userKey{}, userID)
}

// UserID devuelve el id del usuario autenticado y si estaba presente.
func UserID(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(userKey{}).(string)
	return id, ok && id != ""
}
