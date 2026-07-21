// Package config carga la configuracion de la aplicacion desde el entorno.
// El default de AppEnv es "production" (fail-safe): el playground y la introspection
// de GraphQL solo se exponen si se setea explicitamente APP_ENV=development.
package config

import (
	"strconv"
	"time"
)

const (
	envDevelopment = "development"

	defaultPort          = "8080"
	defaultAppEnv        = "production"
	defaultLogLevel      = "info"
	defaultAccessTTL     = 15 * time.Minute
	defaultRefreshTTL    = 168 * time.Hour // 7 dias
	defaultBcryptCost    = 10
	defaultAuthRateLimit = 10 // requests por minuto por IP en las mutations de auth
)

// Config agrupa la configuracion de ejecucion de la aplicacion.
type Config struct {
	Port              string
	AppEnv            string
	LogLevel          string
	GraphQLPlayground bool
	DatabaseURL       string
	JWTSecret         string
	JWTAccessTTL      time.Duration
	JWTRefreshTTL     time.Duration
	BcryptCost        int
	SeedProducts      bool
	AuthRateLimit     int
}

// IsDevelopment indica si la app corre en modo desarrollo (match exacto, fail-safe).
func (c Config) IsDevelopment() bool { return c.AppEnv == envDevelopment }

// Load construye la Config a partir de una funcion getenv inyectada (os.Getenv en produccion,
// un mapa en los tests). Los valores invalidos caen al default en vez de romper el arranque.
func Load(getenv func(string) string) Config {
	return Config{
		Port:              orDefault(getenv("PORT"), defaultPort),
		AppEnv:            orDefault(getenv("APP_ENV"), defaultAppEnv),
		LogLevel:          orDefault(getenv("LOG_LEVEL"), defaultLogLevel),
		GraphQLPlayground: boolOrDefault(getenv("GRAPHQL_PLAYGROUND"), true),
		DatabaseURL:       getenv("DATABASE_URL"),
		JWTSecret:         getenv("JWT_SECRET"),
		JWTAccessTTL:      durationOrDefault(getenv("JWT_ACCESS_TTL"), defaultAccessTTL),
		JWTRefreshTTL:     durationOrDefault(getenv("JWT_REFRESH_TTL"), defaultRefreshTTL),
		BcryptCost:        intOrDefault(getenv("BCRYPT_COST"), defaultBcryptCost),
		SeedProducts:      boolOrDefault(getenv("SEED_PRODUCTS"), true),
		AuthRateLimit:     intOrDefault(getenv("AUTH_RATE_LIMIT"), defaultAuthRateLimit),
	}
}

func orDefault(v, def string) string {
	if v == "" {
		return def
	}
	return v
}

func boolOrDefault(v string, def bool) bool {
	if v == "" {
		return def
	}
	parsed, err := strconv.ParseBool(v)
	if err != nil {
		return def
	}
	return parsed
}

func intOrDefault(v string, def int) int {
	if v == "" {
		return def
	}
	parsed, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return parsed
}

func durationOrDefault(v string, def time.Duration) time.Duration {
	if v == "" {
		return def
	}
	parsed, err := time.ParseDuration(v)
	if err != nil {
		return def
	}
	return parsed
}
