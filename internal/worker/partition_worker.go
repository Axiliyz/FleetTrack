package worker

import (
	"context"
	"fleettrack/internal/logger"
	"fleettrack/internal/repository"
	"fmt"
	"sync"
	"time"
)

// PartitionWorker отвечает за периодическое создание партиций таблиц в БД
type PartitionWorker struct {
	repo      repository.PartitionRepository
	logger    logger.Logger
	intervals time.Duration
	months    int
	wg        sync.WaitGroup
}

// NewPartitionWorker создаёт новый экземпляр воркера партиций
func NewPartitionWorker(repo repository.PartitionRepository, l logger.Logger, interval time.Duration, months int) *PartitionWorker {
	return &PartitionWorker{
		repo:      repo,
		logger:    l,
		intervals: interval,
		months:    months,
	}
}

// Start запускает периодическое создание партиций в отдельной горутине
func (w *PartitionWorker) Start(ctx context.Context) {
	w.wg.Add(1)
	go w.run(ctx)
}

// Stop ожидает корректного завершения фоновой горутины
func (w *PartitionWorker) Stop() {
	w.wg.Wait()
}

// run выполняет создание партиций при старте и по таймеру
func (w *PartitionWorker) run(ctx context.Context) {
	defer w.wg.Done()
	w.createPartitions(ctx)
	ticker := time.NewTicker(w.intervals)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.createPartitions(ctx)
		}
	}
}

// createPartitions создаёт партиции для telemetry и alerts на указанное количество месяцев вперёд
func (w *PartitionWorker) createPartitions(ctx context.Context) {
	now := time.Now().UTC()
	currentYear := now.Year()
	currentMonth := now.Month()

	tables := []string{"telemetry", "alerts"}

	for i := 0; i <= w.months; i++ {
		from := time.Date(currentYear, currentMonth+time.Month(i), 1, 0, 0, 0, 0, time.UTC)
		to := time.Date(currentYear, currentMonth+time.Month(i+1), 1, 0, 0, 0, 0, time.UTC)

		for _, table := range tables {
			partitionName := fmt.Sprintf("%s_%d_%02d", table, from.Year(), from.Month())
			if err := w.repo.CreatePartition(ctx, table, partitionName, from, to); err != nil {
				w.logger.Error(fmt.Sprintf("failed to create partition %s: %s", partitionName, err.Error()))
				continue
			}
			w.logger.Info(fmt.Sprintf("Partition ensured: %s", partitionName))
		}
	}
}
