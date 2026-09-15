package worker

import (
	"context"
	"fleettrack/internal/logger"
	"fleettrack/internal/model"
	"fleettrack/internal/notifier"
	"fleettrack/internal/repository"
	"fmt"
	"sync"
	"time"
)

// NotificationWorker периодически вычитывает пачки задач из Transactional Outbox и отправляет их адресатам
type NotificationWorker struct {
	repo         repository.AlertNotificationRepository
	dispatcher   *notifier.Dispatcher
	logger       logger.Logger
	pollInterval time.Duration
	batchSize    int
	wg           sync.WaitGroup
}

// NewNotificationWorker создаёт новый экземпляр фонового воркера отправки уведомлений
func NewNotificationWorker(r repository.AlertNotificationRepository, dispatcher *notifier.Dispatcher, l logger.Logger, pi time.Duration, bs int) *NotificationWorker {
	if dispatcher == nil {
		panic("notification worker: dispatcher is required")
	}
	if r == nil {
		panic("notification worker: repository is required")
	}

	if pi < 1 {
		pi = 2 * time.Second
	}
	if bs < 1 {
		bs = 50
	}
	return &NotificationWorker{
		repo:         r,
		dispatcher:   dispatcher,
		logger:       l,
		pollInterval: pi,
		batchSize:    bs,
	}
}

// Start запускает фоновый опрос очереди в отдельной горутине
func (w *NotificationWorker) Start(ctx context.Context) {
	w.wg.Add(1)
	go w.run(ctx)
}

// Stop ожидает завершения работы горутины воркера
func (w *NotificationWorker) Stop() {
	w.wg.Wait()
}

func (w *NotificationWorker) run(ctx context.Context) {
	defer w.wg.Done()

	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.processBatch(ctx)
		}
	}
}

func (w *NotificationWorker) processBatch(ctx context.Context) {
	tasks, err := w.repo.FetchPending(ctx, w.batchSize)
	if err != nil {
		w.logger.Error(fmt.Sprintf("failed to fetch pending notifications: %s", err.Error()))
		return
	}

	for _, task := range tasks {
		w.processTask(ctx, task)
	}
}

// processTask отправляет одно уведомление и фиксирует статус в базе данных
func (w *NotificationWorker) processTask(ctx context.Context, task model.NotificationTask) {
	err := w.dispatcher.Dispatch(ctx, task.Channel, task.Alert)
	if err != nil {
		w.logger.Warn(fmt.Sprintf("failed to send notification %d: %s", task.ID, err.Error()))
		nextRetry := calculateNextRetry(task.Attempts)
		if markErr := w.repo.MarkFailed(ctx, task.ID, err.Error(), nextRetry); markErr != nil {
			w.logger.Error(fmt.Sprintf("failed to mark notification %d failed: %s", task.ID, markErr.Error()))
		}
		return
	}

	if err := w.repo.MarkSent(ctx, task.ID); err != nil {
		w.logger.Error(fmt.Sprintf("failed to mark notification %d sent: %s", task.ID, err.Error()))
	}
}

// calculateNextRetry рассчитывает задержку перед следующей попыткой по экспоненциальному закону
func calculateNextRetry(attempts int) *time.Time {
	var delay time.Duration
	switch attempts {
	case 0:
		delay = 10 * time.Second
	case 1:
		delay = 30 * time.Second
	case 2:
		delay = 1 * time.Minute
	default:
		return nil
	}

	t := time.Now().Add(delay)
	return &t
}
