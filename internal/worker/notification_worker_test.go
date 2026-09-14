// Package worker содержит тесты фоновых воркеров приложения
package worker

import (
	"context"
	"errors"
	"fleettrack/internal/model"
	"fleettrack/internal/notifier"
	"sync"
	"testing"
	"time"
)

// mockRepo имитирует репозиторий alert_notifications для тестирования воркера
type mockRepo struct {
	mu           sync.Mutex
	tasks        []model.NotificationTask
	fetchErr     error
	sentIDs      []int
	failedIDs    []int
	failedErrors []string
}

func (m *mockRepo) CreateBatch(ctx context.Context, nots []model.AlertNotification) error {
	return nil
}

func (m *mockRepo) FetchPending(ctx context.Context, batchSize int) ([]model.NotificationTask, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.fetchErr != nil {
		return nil, m.fetchErr
	}
	res := m.tasks
	m.tasks = nil
	return res, nil
}

func (m *mockRepo) MarkSent(ctx context.Context, id int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sentIDs = append(m.sentIDs, id)
	return nil
}

func (m *mockRepo) MarkFailed(ctx context.Context, id int, errText string, nextRetryAt *time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.failedIDs = append(m.failedIDs, id)
	m.failedErrors = append(m.failedErrors, errText)
	return nil
}

// mockSender реализует отправку для диспетчера в тестах
type mockSender struct {
	err error
}

func (s *mockSender) Send(ctx context.Context, ch model.UserNotificationChannel, alert model.Alert) error {
	return s.err
}

// mockLogger реализует интерфейс logger.Logger для тестов
type mockLogger struct{}

func (m *mockLogger) Debug(msg string) {}
func (m *mockLogger) Info(msg string)  {}
func (m *mockLogger) Warn(msg string)  {}
func (m *mockLogger) Error(msg string) {}

// TestCalculateNextRetry проверяет экспоненциальный расчёт задержки повторных попыток
func TestCalculateNextRetry(t *testing.T) {
	now := time.Now()

	// Первая попытка (0) — задержка 10 секунд
	retry0 := calculateNextRetry(0)
	if retry0 == nil {
		t.Fatal("expected non-nil retry for attempt 0")
	}
	diff0 := retry0.Sub(now)
	if diff0 < 9*time.Second || diff0 > 11*time.Second {
		t.Errorf("expected ~10s delay, got %v", diff0)
	}

	// Вторая попытка (1) — задержка 30 секунд
	retry1 := calculateNextRetry(1)
	if retry1 == nil {
		t.Fatal("expected non-nil retry for attempt 1")
	}
	diff1 := retry1.Sub(now)
	if diff1 < 29*time.Second || diff1 > 31*time.Second {
		t.Errorf("expected ~30s delay, got %v", diff1)
	}

	// Третья попытка (2) — задержка 1 минута
	retry2 := calculateNextRetry(2)
	if retry2 == nil {
		t.Fatal("expected non-nil retry for attempt 2")
	}
	diff2 := retry2.Sub(now)
	if diff2 < 59*time.Second || diff2 > 61*time.Second {
		t.Errorf("expected ~60s delay, got %v", diff2)
	}

	// Четвёртая попытка (3) — попытки исчерпаны
	retry3 := calculateNextRetry(3)
	if retry3 != nil {
		t.Errorf("expected nil for attempt 3, got %v", retry3)
	}
}

// TestNewNotificationWorker_Defaults проверяет установку значений по умолчанию для интервала и размера пачки
func TestNewNotificationWorker_Defaults(t *testing.T) {
	repo := &mockRepo{}
	logger := &mockLogger{}
	dispatcher := notifier.NewDispatcher(&mockSender{}, &mockSender{}, logger)

	w := NewNotificationWorker(repo, dispatcher, logger, 0, 0)
	if w.pollInterval != 2*time.Second {
		t.Errorf("expected default pollInterval 2s, got %v", w.pollInterval)
	}
	if w.batchSize != 50 {
		t.Errorf("expected default batchSize 50, got %d", w.batchSize)
	}
}

// TestNotificationWorker_ProcessBatch_Success проверяет успешную обработку задачи воркером
func TestNotificationWorker_ProcessBatch_Success(t *testing.T) {
	repo := &mockRepo{
		tasks: []model.NotificationTask{
			{
				ID:       101,
				Attempts: 0,
				Channel:  model.UserNotificationChannel{Type: model.NotificationChannelTg},
				Alert:    model.Alert{ID: 1},
			},
		},
	}
	logger := &mockLogger{}
	dispatcher := notifier.NewDispatcher(&mockSender{}, &mockSender{}, logger)
	w := NewNotificationWorker(repo, dispatcher, logger, time.Second, 10)

	ctx := context.Background()
	w.processBatch(ctx)

	if len(repo.sentIDs) != 1 || repo.sentIDs[0] != 101 {
		t.Errorf("expected task 101 marked as sent, got: %v", repo.sentIDs)
	}
	if len(repo.failedIDs) != 0 {
		t.Errorf("expected 0 failed tasks, got: %v", repo.failedIDs)
	}
}

// TestNotificationWorker_ProcessBatch_Failed проверяет обработку ошибки отправки и фиксацию сбоя
func TestNotificationWorker_ProcessBatch_Failed(t *testing.T) {
	sendErr := errors.New("network timeout")
	repo := &mockRepo{
		tasks: []model.NotificationTask{
			{
				ID:       102,
				Attempts: 1,
				Channel:  model.UserNotificationChannel{Type: model.NotificationChannelTg},
				Alert:    model.Alert{ID: 2},
			},
		},
	}
	logger := &mockLogger{}
	dispatcher := notifier.NewDispatcher(&mockSender{err: sendErr}, &mockSender{}, logger)
	w := NewNotificationWorker(repo, dispatcher, logger, time.Second, 10)

	ctx := context.Background()
	w.processBatch(ctx)

	if len(repo.sentIDs) != 0 {
		t.Errorf("expected 0 sent tasks, got: %v", repo.sentIDs)
	}
	if len(repo.failedIDs) != 1 || repo.failedIDs[0] != 102 {
		t.Errorf("expected task 102 marked as failed, got: %v", repo.failedIDs)
	}
	if len(repo.failedErrors) != 1 || repo.failedErrors[0] != sendErr.Error() {
		t.Errorf("expected error %q, got: %v", sendErr.Error(), repo.failedErrors)
	}
}
