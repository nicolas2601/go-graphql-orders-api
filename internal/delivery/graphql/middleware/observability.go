package middleware

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"runtime/debug"
	"strings"

	"github.com/google/uuid"
)

type (
	requestIDKey struct{}
	clientIPKey  struct{}
)

const requestIDHeader = "X-Request-ID"

// RequestID asigna (o propaga) un id de correlacion por request: lo pone en el contexto y en el
// header de respuesta, para poder trazar cada request en los logs.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get(requestIDHeader)
		if id == "" {
			id = uuid.NewString()
		}
		w.Header().Set(requestIDHeader, id)
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), requestIDKey{}, id)))
	})
}

// RequestIDFromContext devuelve el id de correlacion de la request, si esta presente.
func RequestIDFromContext(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKey{}).(string)
	return id
}

// ClientIP extrae la IP del cliente (respetando X-Forwarded-For) y la deja en el contexto, para el
// rate limiting por IP.
func ClientIP(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), clientIPKey{}, clientIP(r))))
	})
}

// ClientIPFromContext devuelve la IP del cliente si el middleware ClientIP la seteo.
func ClientIPFromContext(ctx context.Context) string {
	ip, _ := ctx.Value(clientIPKey{}).(string)
	return ip
}

func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return strings.TrimSpace(strings.Split(xff, ",")[0])
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// Recover atrapa un panic en la cadena HTTP: lo loguea con su stack (y el request-id del contexto)
// y devuelve un error generico, en vez de tumbar el proceso o dejar la conexion colgada.
func Recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if p := recover(); p != nil {
				slog.ErrorContext(r.Context(), "panic recovered",
					"panic", p, "request_id", RequestIDFromContext(r.Context()), "stack", string(debug.Stack()))
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(`{"errors":[{"message":"internal server error"}]}`))
			}
		}()
		next.ServeHTTP(w, r)
	})
}
