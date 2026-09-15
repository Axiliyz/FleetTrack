// Package worker содержит тесты фоновых воркеров приложения
package worker

import (
	"context"
	"fleettrack/internal/model"
	"testing"
	"time"
)

// mockRuleRepo реализует AlertRuleRepository для тестов оффлайн воркера
type mockRuleRepo struct {
	rules []model.AlertRule
	err   error
}

func (m *mockRuleRepo) Create(ctx context.Context, r *model.AlertRule) error {
	return nil
}

func (m *mockRuleRepo) GetByID(ctx context.Context, id int) (model.AlertRule, error) {
	return model.AlertRule{}, nil
}

func (m *mockRuleRepo) GetList(ctx context.Context, filter model.AlertRuleFilter) ([]model.AlertRule, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.rules, nil
}

func (m *mockRuleRepo) Update(ctx context.Context, upd model.AlertRule) (model.AlertRule, error) {
	return model.AlertRule{}, nil
}

func (m *mockRuleRepo) GetActiveRulesByOrg(ctx context.Context, orgID int) ([]model.AlertRule, error) {
	return m.rules, nil
}

func (m *mockRuleRepo) DeleteByID(ctx context.Context, id int) (model.AlertRule, error) {
	return model.AlertRule{}, nil
}

// mockOfflineAlertRepo реализует необходимые методы AlertRepository для тестов оффлайна
type mockOfflineAlertRepo struct {
	vehicles []model.OfflineVehicleInfo
	err      error
}

func (m *mockOfflineAlertRepo) GetActiveAlert(ctx context.Context, vehicleID, ruleID int) (model.Alert, error) {
	return model.Alert{}, model.ErrNotFound
}

func (m *mockOfflineAlertRepo) Create(ctx context.Context, a *model.Alert) error {
	return nil
}

func (m *mockOfflineAlertRepo) AcquireLock(ctx context.Context, vehicleID, ruleID int) error {
	return nil
}

func (m *mockOfflineAlertRepo) Resolve(ctx context.Context, id int, resolvedBy *int) (model.Alert, error) {
	return model.Alert{}, nil
}

func (m *mockOfflineAlertRepo) AcknowledgeAlert(ctx context.Context, alertID, userID int) (model.Alert, error) {
	return model.Alert{}, nil
}

func (m *mockOfflineAlertRepo) GetList(ctx context.Context, filter model.AlertFilter) ([]model.Alert, error) {
	return nil, nil
}

func (m *mockOfflineAlertRepo) FindOfflineVehicles(ctx context.Context, thresholdMinutes float64) ([]model.OfflineVehicleInfo, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.vehicles, nil
}

// TestNewOfflineWorker_Defaults проверяет инициализацию значений по умолчанию для интервала проверки
func TestNewOfflineWorker_Defaults(t *testing.T) {
	logger := &mockLogger{}
	w := NewOfflineWorker(nil, nil, nil, logger, 5*time.Second)

	if w.checkInterval != 1*time.Minute {
		t.Errorf("expected default checkInterval 1m, got %v", w.checkInterval)
	}

	wCustom := NewOfflineWorker(nil, nil, nil, logger, 10*time.Minute)
	if wCustom.checkInterval != 10*time.Minute {
		t.Errorf("expected custom checkInterval 10m, got %v", wCustom.checkInterval)
	}
}

// TestOfflineWorker_CheckOffline_Empty проверяет корректную работу при отсутствии оффлайн машин
func TestOfflineWorker_CheckOffline_Empty(t *testing.T) {
	ruleRepo := &mockRuleRepo{
		rules: []model.AlertRule{
			{
				ID:             1,
				OrganizationID: 1,
				Type:           model.AlertRuleDeviceOffline,
				Threshold:      15.0,
				Enabled:        true,
			},
		},
	}
	alertRepo := &mockOfflineAlertRepo{
		vehicles: nil,
	}
	logger := &mockLogger{}

	w := NewOfflineWorker(alertRepo, ruleRepo, nil, logger, time.Minute)
	ctx := context.Background()
	w.checkOffline(ctx)
}
