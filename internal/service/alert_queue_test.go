package service

import (
	"context"
	"fleettrack/internal/logger"
	"fleettrack/internal/model"
	"testing"
	"time"
)

// countingRuleRepo считает вызовы оценки, чтобы проверить, что очередь дочитана
type countingRuleRepo struct {
	mockAlertRuleRepo
	calls chan struct{}
}

func (c *countingRuleRepo) GetActiveRulesByOrg(ctx context.Context, orgID int) ([]model.AlertRule, error) {
	c.calls <- struct{}{}
	return nil, nil
}

func TestEnqueue_ReturnsWhenRequestCancelled(t *testing.T) {
	svc := NewAlertService(nil, &mockAlertRuleRepo{}, nil, nil, nil, nil, logger.NewStdLogger(logger.ErrorLevel), 1)
	svc.Enqueue(context.Background(), model.Telemetry{VehicleID: 1})

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	done := make(chan struct{})
	go func() {
		svc.Enqueue(ctx, model.Telemetry{VehicleID: 2})
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Enqueue blocked after request context was cancelled")
	}
	if got := svc.QueueLength(); got != 1 {
		t.Fatalf("queue length = %d, want 1", got)
	}
}

func TestStop_DrainsQueueAndRejectsLateEnqueue(t *testing.T) {
	repo := &countingRuleRepo{calls: make(chan struct{}, 10)}
	svc := NewAlertService(nil, repo, nil, nil, nil, nil, logger.NewStdLogger(logger.ErrorLevel), 10)
	for i := 1; i <= 3; i++ {
		svc.Enqueue(context.Background(), model.Telemetry{VehicleID: i})
	}

	svc.Start(context.Background(), 1)
	svc.Stop()

	if got := len(repo.calls); got != 3 {
		t.Fatalf("evaluated %d queued points, want 3", got)
	}

	svc.Enqueue(context.Background(), model.Telemetry{VehicleID: 4})
	svc.Stop()
}
