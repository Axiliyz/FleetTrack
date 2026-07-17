package transaction

import (
	"context"
	"errors"
	"fleettrack/internal/database"
	"testing"

	"github.com/jackc/pgx/v5"
)

// mockTx встраивает pgx.Tx, чтобы не реализовывать весь интерфейс -
// нужны только Commit и Rollback, остальные методы в тестах не вызываются.
type mockTx struct {
	pgx.Tx
	commitErr   error
	rollbackErr error
	committed   bool
	rolledBack  bool
}

func (m *mockTx) Commit(ctx context.Context) error {
	m.committed = true
	return m.commitErr
}

func (m *mockTx) Rollback(ctx context.Context) error {
	m.rolledBack = true
	return m.rollbackErr
}

type mockBeginner struct {
	tx       *mockTx
	beginErr error
}

func (m *mockBeginner) Begin(ctx context.Context) (pgx.Tx, error) {
	if m.beginErr != nil {
		return nil, m.beginErr
	}
	return m.tx, nil
}

func TestWithTx_Success(t *testing.T) {
	tx := &mockTx{}
	m := &PostgresTransactionManager{pool: &mockBeginner{tx: tx}}

	called := false
	err := m.WithTx(context.Background(), func(_ database.DBTX) error {
		called = true
		return nil
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Error("fn was not called")
	}
	if !tx.committed {
		t.Error("expected transaction to be committed")
	}
	if !tx.rolledBack {
		t.Error("expected deferred Rollback to be called even after commit")
	}
}

func TestWithTx_FnError_RollsBack(t *testing.T) {
	tx := &mockTx{}
	m := &PostgresTransactionManager{pool: &mockBeginner{tx: tx}}

	fnErr := errors.New("validation failed")
	err := m.WithTx(context.Background(), func(_ database.DBTX) error {
		return fnErr
	})

	if !errors.Is(err, fnErr) {
		t.Errorf("got %v, want %v", err, fnErr)
	}
	if tx.committed {
		t.Error("transaction should not be committed when fn fails")
	}
	if !tx.rolledBack {
		t.Error("expected transaction to be rolled back")
	}
}

func TestWithTx_BeginError(t *testing.T) {
	beginErr := errors.New("connection refused")
	m := &PostgresTransactionManager{pool: &mockBeginner{beginErr: beginErr}}

	called := false
	err := m.WithTx(context.Background(), func(_ database.DBTX) error {
		called = true
		return nil
	})

	if !errors.Is(err, beginErr) {
		t.Errorf("got %v, want wrapped %v", err, beginErr)
	}
	if called {
		t.Error("fn should not be called when Begin fails")
	}
}

func TestWithTx_CommitError(t *testing.T) {
	commitErr := errors.New("commit failed")
	tx := &mockTx{commitErr: commitErr}
	m := &PostgresTransactionManager{pool: &mockBeginner{tx: tx}}

	err := m.WithTx(context.Background(), func(_ database.DBTX) error {
		return nil
	})

	if !errors.Is(err, commitErr) {
		t.Errorf("got %v, want %v", err, commitErr)
	}
	if !tx.rolledBack {
		t.Error("expected deferred Rollback to still be called after failed commit")
	}
}
