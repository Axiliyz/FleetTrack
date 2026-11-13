package worker

import (
	"context"
	"errors"
	"fleettrack/internal/logger"
	"sync"
	"testing"
	"time"
)

type fakeTokenDeleter struct {
	mu      sync.Mutex
	befores []time.Time
	err     error
}

func (f *fakeTokenDeleter) DeleteExpired(_ context.Context, before time.Time) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.befores = append(f.befores, before)
	return 3, f.err
}

// TestTokenCleanupWorker_DeletesWithGrace проверяет, что чистка идёт сразу при старте
// и не трогает токены, истёкшие недавно: по ним ещё ловится повторное использование
func TestTokenCleanupWorker_DeletesWithGrace(t *testing.T) {
	repo := &fakeTokenDeleter{}
	w := NewTokenCleanupWorker(repo, logger.NewStdLogger(logger.ErrorLevel), time.Hour, 24*time.Hour)

	ctx, cancel := context.WithCancel(context.Background())
	w.Start(ctx)
	cancel()
	w.Stop()

	repo.mu.Lock()
	defer repo.mu.Unlock()
	if len(repo.befores) != 1 {
		t.Fatalf("DeleteExpired calls = %d, want 1 at start", len(repo.befores))
	}
	if age := time.Since(repo.befores[0]); age < 23*time.Hour {
		t.Errorf("before = now-%s, токены младше суток после истечения должны остаться", age)
	}
}

// TestTokenCleanupWorker_ErrorDoesNotStop проверяет, что ошибка базы не роняет воркер
func TestTokenCleanupWorker_ErrorDoesNotStop(t *testing.T) {
	repo := &fakeTokenDeleter{err: errors.New("db down")}
	w := NewTokenCleanupWorker(repo, logger.NewStdLogger(logger.ErrorLevel), 10*time.Millisecond, time.Hour)

	ctx, cancel := context.WithCancel(context.Background())
	w.Start(ctx)
	time.Sleep(35 * time.Millisecond)
	cancel()
	w.Stop()

	repo.mu.Lock()
	defer repo.mu.Unlock()
	if len(repo.befores) < 2 {
		t.Errorf("DeleteExpired calls = %d, воркер должен продолжать после ошибки", len(repo.befores))
	}
}
