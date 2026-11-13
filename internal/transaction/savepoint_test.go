package transaction

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// recordingDBTX запоминает выполненные команды и может сломать одну из них
type recordingDBTX struct {
	commands []string
	failOn   string
}

func (r *recordingDBTX) Exec(_ context.Context, sql string, _ ...any) (pgconn.CommandTag, error) {
	r.commands = append(r.commands, sql)
	if sql == r.failOn {
		return pgconn.CommandTag{}, errors.New("connection lost")
	}
	return pgconn.CommandTag{}, nil
}

func (r *recordingDBTX) Query(context.Context, string, ...any) (pgx.Rows, error) { return nil, nil }
func (r *recordingDBTX) QueryRow(context.Context, string, ...any) pgx.Row        { return nil }

func TestWithSavepoint_ReleaseOnSuccess(t *testing.T) {
	tx := &recordingDBTX{}
	fnErr, txErr := WithSavepoint(context.Background(), tx, func() error { return nil })
	if fnErr != nil || txErr != nil {
		t.Fatalf("fnErr=%v txErr=%v", fnErr, txErr)
	}
	want := []string{"SAVEPOINT " + savepointName, "RELEASE SAVEPOINT " + savepointName}
	if len(tx.commands) != 2 || tx.commands[0] != want[0] || tx.commands[1] != want[1] {
		t.Errorf("commands = %v, want %v", tx.commands, want)
	}
}

func TestWithSavepoint_RollbackOnlyItem(t *testing.T) {
	tx := &recordingDBTX{}
	itemErr := errors.New("device not assigned")
	fnErr, txErr := WithSavepoint(context.Background(), tx, func() error { return itemErr })
	if !errors.Is(fnErr, itemErr) {
		t.Errorf("fnErr = %v, want %v", fnErr, itemErr)
	}
	if txErr != nil {
		t.Errorf("txErr = %v, транзакция должна остаться живой", txErr)
	}
	if last := tx.commands[len(tx.commands)-1]; last != "ROLLBACK TO SAVEPOINT "+savepointName {
		t.Errorf("last command = %q", last)
	}
}

func TestWithSavepoint_BrokenTransaction(t *testing.T) {
	tx := &recordingDBTX{failOn: "SAVEPOINT " + savepointName}
	called := false
	_, txErr := WithSavepoint(context.Background(), tx, func() error { called = true; return nil })
	if txErr == nil {
		t.Fatal("ожидалась ошибка транзакции")
	}
	if called {
		t.Error("fn не должна выполняться без точки сохранения")
	}
}
