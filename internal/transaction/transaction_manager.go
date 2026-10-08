// Package transaction предоставляет менеджер для выполнения операций в рамках БД-транзакции.
package transaction

import (
	"context"
	"fleettrack/internal/database"
	"fmt"
	"time"

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

// Observer получает длительность фаз транзакции: "begin" и "commit".
type Observer func(phase string, d time.Duration)

// Option настраивает PostgresTransactionManager.
type Option func(*PostgresTransactionManager)

// WithObserver подключает наблюдателя за длительностью фаз транзакции.
func WithObserver(o Observer) Option {
	return func(m *PostgresTransactionManager) { m.observe = o }
}

// PostgresTransactionManager реализует TransactionManager поверх pgxpool.Pool.
type PostgresTransactionManager struct {
	pool    beginner
	observe Observer
}

// NewPostgresTransactionManager создаёт новый TransactionManager с заданным пулом соединений.
func NewPostgresTransactionManager(pool *pgxpool.Pool, opts ...Option) *PostgresTransactionManager {
	m := &PostgresTransactionManager{pool: pool}
	for _, opt := range opts {
		opt(m)
	}
	return m
}

func (m *PostgresTransactionManager) record(phase string, start time.Time) {
	if m.observe != nil {
		m.observe(phase, time.Since(start))
	}
}

// WithTx открывает транзакцию, выполняет fn и коммитит её при успехе.
// При ошибке или панике транзакция откатывается.
func (m *PostgresTransactionManager) WithTx(ctx context.Context, fn func(tx database.DBTX) error) error {
	beginStart := time.Now()
	tx, err := m.pool.Begin(ctx)
	m.record("begin", beginStart)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}

	defer func() {
		_ = tx.Rollback(ctx)
	}()

	if err = fn(tx); err != nil {
		return err
	}

	commitStart := time.Now()
	err = tx.Commit(ctx)
	m.record("commit", commitStart)
	return err
}
