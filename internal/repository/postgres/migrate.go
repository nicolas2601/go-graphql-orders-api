package postgres

import (
	"database/sql"
	"fmt"
	"log/slog"

	_ "github.com/jackc/pgx/v5/stdlib" // registra el driver "pgx" para database/sql (usado por goose)
	"github.com/pressly/goose/v3"

	"github.com/nicolas2601/go-graphql-orders-api/migrations"
)

// RunMigrations aplica las migraciones embebidas de goose de forma idempotente.
// Usa un *sql.DB con el driver stdlib de pgx solo para migrar; la app corre sobre pgxpool.
func RunMigrations(databaseURL string) error {
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return fmt.Errorf("open db for migrations: %w", err)
	}
	defer func() {
		if cerr := db.Close(); cerr != nil {
			slog.Warn("closing migration db connection", "error", cerr)
		}
	}()

	goose.SetBaseFS(migrations.FS)
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("set goose dialect: %w", err)
	}
	if err := goose.Up(db, "."); err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}
	return nil
}
