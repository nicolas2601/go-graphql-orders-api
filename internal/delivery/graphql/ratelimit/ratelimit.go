// Package ratelimit provee un limitador de tasa por clave (por ejemplo, por IP) para frenar abuso
// como fuerza bruta en login/register.
package ratelimit

import (
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// Limiter mantiene un token bucket independiente por clave.
type Limiter struct {
	mu      sync.Mutex
	buckets map[string]*rate.Limiter
	limit   rate.Limit
	burst   int
}

// New crea un limiter que permite perMinute eventos por minuto por clave (con un burst igual).
// Nota: los buckets se guardan por clave sin expiracion; para un servicio de larga vida con muchas
// IPs distintas, conviene evolucionar a una cache con TTL. Suficiente para el alcance de la prueba.
func New(perMinute int) *Limiter {
	if perMinute < 1 {
		perMinute = 1
	}
	return &Limiter{
		buckets: make(map[string]*rate.Limiter),
		limit:   rate.Every(time.Minute / time.Duration(perMinute)),
		burst:   perMinute,
	}
}

// Allow indica si el evento para la clave dada esta permitido segun su tasa.
func (l *Limiter) Allow(key string) bool {
	l.mu.Lock()
	bucket, ok := l.buckets[key]
	if !ok {
		bucket = rate.NewLimiter(l.limit, l.burst)
		l.buckets[key] = bucket
	}
	l.mu.Unlock()
	return bucket.Allow()
}
