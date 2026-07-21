// Command orders-api es el punto de entrada de la API de ordenes.
// Por ahora carga la configuracion y el logger; la composicion completa (base de datos,
// migraciones, seed y servidor GraphQL) se cablea en las siguientes features.
package main

import (
	"log/slog"
	"os"

	"github.com/nicolas2601/go-graphql-orders-api/internal/config"
)

func main() {
	cfg := config.Load(os.Getenv)
	slog.SetDefault(newLogger(cfg.LogLevel))
	slog.Info("orders-api bootstrap", "env", cfg.AppEnv, "port", cfg.Port)
}

// newLogger arma un logger JSON estructurado con el nivel dado (info por defecto).
func newLogger(level string) *slog.Logger {
	var lvl slog.Level
	if err := lvl.UnmarshalText([]byte(level)); err != nil {
		lvl = slog.LevelInfo
	}
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: lvl}))
}
