package worker

import (
	"context"
	"fleettrack/internal/logger"
	"fmt"
	"sync"
	"time"
)

// ExpiredTokenDeleter удаляет истёкшие refresh-токены; реализует repository.RefreshTokenRepository
type ExpiredTokenDeleter interface {
	DeleteExpired(ctx context.Context, before time.Time) (int64, error)
}

// TokenCleanupWorker периодически удаляет истёкшие refresh-токены: каждый логин и каждая ротация
// добавляют строку, и без чистки таблица растёт бесконечно
type TokenCleanupWorker struct {
	repo     ExpiredTokenDeleter
	logger   logger.Logger
	interval time.Duration
	// keep - сколько хранить токен после истечения: по нему ещё видно повторное
	// использование, если украденный токен предъявят чуть позже
	keep time.Duration
	wg   sync.WaitGroup
}

// NewTokenCleanupWorker создаёт воркер чистки refresh-токенов
func NewTokenCleanupWorker(repo ExpiredTokenDeleter, l logger.Logger, interval, keep time.Duration) *TokenCleanupWorker {
	return &TokenCleanupWorker{
		repo:     repo,
		logger:   l,
		interval: interval,
		keep:     keep,
	}
}

// Start запускает чистку при старте и затем по таймеру в отдельной горутине
func (w *TokenCleanupWorker) Start(ctx context.Context) {
	w.wg.Add(1)
	go w.run(ctx)
}

// Stop ожидает завершения фоновой горутины
func (w *TokenCleanupWorker) Stop() {
	w.wg.Wait()
}

func (w *TokenCleanupWorker) run(ctx context.Context) {
	defer w.wg.Done()
	w.cleanup(ctx)

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.cleanup(ctx)
		}
	}
}

// cleanup удаляет токены, истёкшие раньше чем keep назад
func (w *TokenCleanupWorker) cleanup(ctx context.Context) {
	deleted, err := w.repo.DeleteExpired(ctx, time.Now().Add(-w.keep))
	if err != nil {
		w.logger.Error(fmt.Sprintf("failed to delete expired refresh tokens: %s", err))
		return
	}
	if deleted > 0 {
		w.logger.Info(fmt.Sprintf("deleted %d expired refresh tokens", deleted))
	}
}
