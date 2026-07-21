// Package postgres implementa los ports del dominio (repositorios y TxManager) sobre PostgreSQL
// con pgx. La transaccion se propaga por el contexto: el caso de uso nunca ve un pgx.Tx.
package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/nicolas2601/go-graphql-orders-api/internal/domain"
)

// Querier abstrae las operaciones comunes de *pgxpool.Pool y pgx.Tx, para que los repositorios
// funcionen igual dentro o fuera de una transaccion.
type Querier interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// txKey es la clave (privada) bajo la cual viaja la transaccion en el contexto.
type txKey struct{}

// TxManager implementa domain.TxManager con pgx.
type TxManager struct {
	pool *pgxpool.Pool
}

// NewTxManager crea un TxManager sobre el pool dado.
func NewTxManager(pool *pgxpool.Pool) *TxManager { return &TxManager{pool: pool} }

var _ domain.TxManager = (*TxManager)(nil)

// WithinTx abre una transaccion, la inyecta en el contexto y ejecuta fn. Hace commit si fn
// devuelve nil, o rollback si devuelve error. El caso de uso solo ve esta interfaz.
func (m *TxManager) WithinTx(ctx context.Context, fn func(context.Context) error) error {
	tx, err := m.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	if err := fn(context.WithValue(ctx, txKey{}, tx)); err != nil {
		_ = tx.Rollback(ctx)
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}
	return nil
}

// querier devuelve la transaccion del contexto si la hay, o el pool en caso contrario.
func querier(ctx context.Context, pool *pgxpool.Pool) Querier {
	if tx, ok := ctx.Value(txKey{}).(pgx.Tx); ok {
		return tx
	}
	return pool
}

// inTx indica si el contexto lleva una transaccion. Los repositorios lo usan para decidir si
// bloquear la fila (SELECT ... FOR UPDATE) al leer dentro de una transaccion.
func inTx(ctx context.Context) bool {
	_, ok := ctx.Value(txKey{}).(pgx.Tx)
	return ok
}

// forUpdate agrega la clausula de bloqueo de fila cuando se lee dentro de una transaccion.
func forUpdate(ctx context.Context, query string) string {
	if inTx(ctx) {
		return query + " FOR UPDATE"
	}
	return query
}
