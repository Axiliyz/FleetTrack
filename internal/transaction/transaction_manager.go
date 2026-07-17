// Package transaction предоставляет менеджер для выполнения операций в рамках БД-транзакции.
package transaction

import (
	"context"
	"fleettrack/internal/database"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TransactionManager выполняет переданную функцию в рамках БД-транзакции.
type TransactionManager interface {
	WithTx(ctx context.Context, fn func(tx database.DBTX) error) error
}

// beginner абстрагирует запуск транзакции пулом. Через этот интерфейс
// в тестах подменяется *pgxpool.Pool, чтобы не поднимать реальную БД.
type beginner interface {
	Begin(ctx context.Context) (pgx.Tx, error)
}

// PostgresTransactionManager реализует TransactionManager поверх pgxpool.Pool.
type PostgresTransactionManager struct {
	pool beginner
}

// NewPostgresTransactionManager создаёт новый TransactionManager с заданным пулом соединений.
func NewPostgresTransactionManager(pool *pgxpool.Pool) *PostgresTransactionManager {
	return &PostgresTransactionManager{
		pool: pool,
	}
}

// WithTx открывает транзакцию, выполняет fn и коммитит её при успехе.
// При ошибке или панике транзакция откатывается.
func (m *PostgresTransactionManager) WithTx(ctx context.Context, fn func(tx database.DBTX) error) error {
	tx, err := m.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}

	defer tx.Rollback(ctx)

	if err = fn(tx); err != nil {
		return err
	}

	if err = tx.Commit(ctx); err != nil {
		return err
	}

	return nil
}
