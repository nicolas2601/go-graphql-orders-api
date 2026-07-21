package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/nicolas2601/go-graphql-orders-api/internal/domain"
)

// UserRepository persiste usuarios en PostgreSQL.
type UserRepository struct {
	pool *pgxpool.Pool
}

// NewUserRepository crea el repositorio sobre el pool dado.
func NewUserRepository(pool *pgxpool.Pool) *UserRepository { return &UserRepository{pool: pool} }

var _ domain.UserRepository = (*UserRepository)(nil)

const userColumns = "id, email, password_hash, created_at"

// Create inserta un usuario. Devuelve ErrEmailAlreadyRegistered si el email ya existe.
func (r *UserRepository) Create(ctx context.Context, user domain.User) error {
	_, err := querier(ctx, r.pool).Exec(ctx,
		`INSERT INTO users (id, email, password_hash, created_at) VALUES ($1, $2, $3, $4)`,
		user.ID, user.Email, user.PasswordHash, user.CreatedAt)
	if isUniqueViolation(err) {
		return domain.ErrEmailAlreadyRegistered
	}
	if err != nil {
		return fmt.Errorf("insert user: %w", err)
	}
	return nil
}

// GetByEmail devuelve el usuario con ese email, o ErrUserNotFound si no existe.
func (r *UserRepository) GetByEmail(ctx context.Context, email string) (domain.User, error) {
	row := querier(ctx, r.pool).QueryRow(ctx,
		`SELECT `+userColumns+` FROM users WHERE email = $1`, email)
	return scanUser(row)
}

// GetByID devuelve el usuario con ese id, o ErrUserNotFound si no existe (o el id es invalido).
func (r *UserRepository) GetByID(ctx context.Context, id string) (domain.User, error) {
	row := querier(ctx, r.pool).QueryRow(ctx,
		`SELECT `+userColumns+` FROM users WHERE id = $1`, id)
	user, err := scanUser(row)
	if isMalformedID(err) {
		return domain.User{}, domain.ErrUserNotFound
	}
	return user, err
}

// FindByIDs devuelve un mapa id -> usuario para los ids dados (para el DataLoader).
func (r *UserRepository) FindByIDs(ctx context.Context, ids []string) (map[string]domain.User, error) {
	rows, err := querier(ctx, r.pool).Query(ctx,
		`SELECT `+userColumns+` FROM users WHERE id = ANY($1::uuid[])`, ids)
	if err != nil {
		return nil, fmt.Errorf("query users: %w", err)
	}
	defer rows.Close()

	out := make(map[string]domain.User, len(ids))
	for rows.Next() {
		user, err := scanUserRow(rows)
		if err != nil {
			return nil, fmt.Errorf("scan user: %w", err)
		}
		out[user.ID] = user
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate users: %w", err)
	}
	return out, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanUser(row pgx.Row) (domain.User, error) {
	user, err := scanUserRow(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.User{}, domain.ErrUserNotFound
	}
	if err != nil {
		return domain.User{}, fmt.Errorf("scan user: %w", err)
	}
	return user, nil
}

func scanUserRow(row rowScanner) (domain.User, error) {
	var u domain.User
	err := row.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.CreatedAt)
	return u, err
}
