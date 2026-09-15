package worker

import (
	"context"
	"errors"
	"fleettrack/internal/logger"
	"fmt"
	"sync"
	"testing"
	"time"
)

type mockPartitionRepo struct {
	mu         sync.Mutex
	partitions []string
	returnErr  error
}

func (m *mockPartitionRepo) CreatePartition(ctx context.Context, parentTable, partitionName string, from, to time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.returnErr != nil {
		return m.returnErr
	}
	m.partitions = append(m.partitions, fmt.Sprintf("%s:%s", parentTable, partitionName))
	return nil
}

// TestPartitionWorker_CreatePartitions проверяет корректное создание партиций для таблиц telemetry и alerts
func TestPartitionWorker_CreatePartitions(t *testing.T) {
	repo := &mockPartitionRepo{}
	log := logger.NewStdLogger(logger.DebugLevel)
	w := NewPartitionWorker(repo, log, time.Hour, 3)

	w.createPartitions(context.Background())

	// 2 таблицы * (3 месяца вперед + текущий месяц = 4) = 8 партиций
	expectedCount := 2 * (3 + 1)
	if len(repo.partitions) != expectedCount {
		t.Errorf("expected %d partitions, got %d", expectedCount, len(repo.partitions))
	}
}

// TestPartitionWorker_RepoError проверяет обработку ошибки репозитория
func TestPartitionWorker_RepoError(t *testing.T) {
	repo := &mockPartitionRepo{returnErr: errors.New("db error")}
	log := logger.NewStdLogger(logger.DebugLevel)
	w := NewPartitionWorker(repo, log, time.Hour, 1)

	// Не должно паниковать при ошибке БД
	w.createPartitions(context.Background())
}

// TestPartitionWorker_StartStop проверяет запуск и остановку воркера
func TestPartitionWorker_StartStop(t *testing.T) {
	repo := &mockPartitionRepo{}
	log := logger.NewStdLogger(logger.DebugLevel)
	w := NewPartitionWorker(repo, log, 10*time.Millisecond, 1)

	ctx, cancel := context.WithCancel(context.Background())
	w.Start(ctx)

	time.Sleep(50 * time.Millisecond)
	cancel()
	w.Stop()

	if len(repo.partitions) == 0 {
		t.Errorf("expected at least 1 partition created on startup")
	}
}
