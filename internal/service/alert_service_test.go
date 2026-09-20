// Package service содержит тесты бизнес-логики сервиса алертов
package service

import (
	"context"
	"errors"
	"fleettrack/internal/database"
	"fleettrack/internal/logger"
	"fleettrack/internal/model"
	"fleettrack/internal/repository"
	"fleettrack/internal/repository/factory"
	"sync"
	"testing"
	"time"
)

// mockAlertRuleRepo имитирует AlertRuleRepository для тестов
type mockAlertRuleRepo struct {
	createErr      error
	getErr         error
	listErr        error
	updateErr      error
	deleteErr      error
	activeRulesErr error
	rule           model.AlertRule
	rules          []model.AlertRule
	activeRules    []model.AlertRule
	lastDeletedID  int
	createdRules   []model.AlertRule
	updatedRules   []model.AlertRule
	mu             sync.Mutex
}

func (m *mockAlertRuleRepo) Create(ctx context.Context, r *model.AlertRule) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.createErr != nil {
		return m.createErr
	}
	r.ID = 1
	m.createdRules = append(m.createdRules, *r)
	return nil
}

func (m *mockAlertRuleRepo) GetByID(ctx context.Context, id int) (model.AlertRule, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.getErr != nil {
		return model.AlertRule{}, m.getErr
	}
	return m.rule, nil
}

func (m *mockAlertRuleRepo) GetList(ctx context.Context, filter model.AlertRuleFilter) ([]model.AlertRule, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.listErr != nil {
		return nil, m.listErr
	}
	return m.rules, nil
}

func (m *mockAlertRuleRepo) Update(ctx context.Context, upd model.AlertRule) (model.AlertRule, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.updateErr != nil {
		return model.AlertRule{}, m.updateErr
	}
	m.updatedRules = append(m.updatedRules, upd)
	return upd, nil
}

func (m *mockAlertRuleRepo) GetActiveRulesByOrg(ctx context.Context, orgID int) ([]model.AlertRule, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.activeRulesErr != nil {
		return nil, m.activeRulesErr
	}
	return m.activeRules, nil
}

func (m *mockAlertRuleRepo) DeleteByID(ctx context.Context, id int) (model.AlertRule, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.deleteErr != nil {
		return model.AlertRule{}, m.deleteErr
	}
	m.lastDeletedID = id
	return m.rule, nil
}

// mockAlertRepo имитирует AlertRepository для тестов
type mockAlertRepo struct {
	createErr     error
	getActiveErr  error
	resolveErr    error
	ackErr        error
	listErr       error
	activeAlert   model.Alert
	createdAlerts []model.Alert
	resolvedIDs   []int
	mu            sync.Mutex
}

func (m *mockAlertRepo) GetActiveAlert(ctx context.Context, vehicleID, ruleID int) (model.Alert, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.getActiveErr != nil {
		return model.Alert{}, m.getActiveErr
	}
	return m.activeAlert, nil
}

func (m *mockAlertRepo) Create(ctx context.Context, a *model.Alert) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.createErr != nil {
		return m.createErr
	}
	a.ID = 100
	m.createdAlerts = append(m.createdAlerts, *a)
	return nil
}

func (m *mockAlertRepo) Resolve(ctx context.Context, id int, resolvedBy *int) (model.Alert, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.resolveErr != nil {
		return model.Alert{}, m.resolveErr
	}
	m.resolvedIDs = append(m.resolvedIDs, id)
	return model.Alert{ID: id, Status: model.AlertStatusResolved}, nil
}

func (m *mockAlertRepo) AcknowledgeAlert(ctx context.Context, alertID, userID int) (model.Alert, error) {
	if m.ackErr != nil {
		return model.Alert{}, m.ackErr
	}
	return model.Alert{ID: alertID, Status: model.AlertStatusAcknowledged}, nil
}

func (m *mockAlertRepo) GetList(ctx context.Context, filter model.AlertFilter) ([]model.Alert, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	return nil, nil
}

func (m *mockAlertRepo) GetByID(ctx context.Context, id int) (model.Alert, error) {
	return model.Alert{ID: id}, nil
}

func (m *mockAlertRepo) FindOfflineVehicles(ctx context.Context, thresholdMinutes float64) ([]model.OfflineVehicleInfo, error) {
	return nil, nil
}

func (m *mockAlertRepo) AcquireLock(ctx context.Context, vehicleID, ruleID int) error {
	return nil
}
func (m *mockChannelRepo) GetUserIDByTelegramChatID(ctx context.Context, chatID int) (int, error) {
	return 1, nil
}

// mockChannelRepo имитирует NotificationChannelRepository для тестов
type mockChannelRepo struct {
	channelsErr error
	channels    []model.UserNotificationChannel
}

func (m *mockChannelRepo) Create(ctx context.Context, ch *model.UserNotificationChannel) error {
	return nil
}

func (m *mockChannelRepo) GetByUserID(ctx context.Context, userID int) ([]model.UserNotificationChannel, error) {
	return nil, nil
}

func (m *mockChannelRepo) GetChannelsForAlert(ctx context.Context, orgID int, severity model.SeverityLevel) ([]model.UserNotificationChannel, error) {
	if m.channelsErr != nil {
		return nil, m.channelsErr
	}
	return m.channels, nil
}

func (m *mockChannelRepo) Delete(ctx context.Context, id int) error {
	return nil
}

// mockNotificationRepo имитирует AlertNotificationRepository для тестов
type mockNotificationRepo struct {
	createBatchErr error
	createdBatches [][]model.AlertNotification
	mu             sync.Mutex
}

func (m *mockNotificationRepo) CreateBatch(ctx context.Context, nots []model.AlertNotification) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.createBatchErr != nil {
		return m.createBatchErr
	}
	m.createdBatches = append(m.createdBatches, nots)
	return nil
}

func (m *mockNotificationRepo) FetchPending(ctx context.Context, batchSize int) ([]model.NotificationTask, error) {
	return nil, nil
}

func (m *mockNotificationRepo) MarkSent(ctx context.Context, id int) error {
	return nil
}

func (m *mockNotificationRepo) MarkFailed(ctx context.Context, id int, errMsg string, nextRetry *time.Time) error {
	return nil
}

// alertTestTxManager имитирует транзакционный менеджер для тестов
type alertTestTxManager struct {
	err error
}

func (m *alertTestTxManager) WithTx(ctx context.Context, fn func(tx database.DBTX) error) error {
	if m.err != nil {
		return m.err
	}
	return fn(nil)
}

// alertTestRepoFactory имитирует фабрику репозиториев
type alertTestRepoFactory struct {
	alertRepo        repository.AlertRepository
	notificationRepo repository.AlertNotificationRepository
}

func (f *alertTestRepoFactory) New(tx database.DBTX) factory.Repositories {
	return factory.Repositories{
		Alert:             f.alertRepo,
		AlertNotification: f.notificationRepo,
	}
}

// TestCheckRuleCondition проверяет логику вычисления условий для различных типов правил
func TestCheckRuleCondition(t *testing.T) {
	fuelLow := float32(10.0)
	fuelHigh := float32(50.0)

	tests := []struct {
		name          string
		rule          model.AlertRule
		telemetry     model.Telemetry
		wantTriggered bool
		wantVal       float64
	}{
		{
			name:          "скорость превысила порог",
			rule:          model.AlertRule{Type: model.AlertRuleSpeedExceed, Threshold: 80},
			telemetry:     model.Telemetry{SpeedKmh: 95.5},
			wantTriggered: true,
			wantVal:       95.5,
		},
		{
			name:          "скорость равна порогу",
			rule:          model.AlertRule{Type: model.AlertRuleSpeedExceed, Threshold: 80},
			telemetry:     model.Telemetry{SpeedKmh: 80.0},
			wantTriggered: true,
			wantVal:       80.0,
		},
		{
			name:          "скорость ниже порога",
			rule:          model.AlertRule{Type: model.AlertRuleSpeedExceed, Threshold: 80},
			telemetry:     model.Telemetry{SpeedKmh: 75.0},
			wantTriggered: false,
			wantVal:       75.0,
		},
		{
			name:          "топливо ниже порога",
			rule:          model.AlertRule{Type: model.AlertRuleLowFuel, Threshold: 15},
			telemetry:     model.Telemetry{Fuel: &fuelLow},
			wantTriggered: true,
			wantVal:       10.0,
		},
		{
			name:          "топливо выше порога",
			rule:          model.AlertRule{Type: model.AlertRuleLowFuel, Threshold: 15},
			telemetry:     model.Telemetry{Fuel: &fuelHigh},
			wantTriggered: false,
			wantVal:       50.0,
		},
		{
			name:          "топливо nil не триггерит правило",
			rule:          model.AlertRule{Type: model.AlertRuleLowFuel, Threshold: 15},
			telemetry:     model.Telemetry{Fuel: nil},
			wantTriggered: false,
			wantVal:       0,
		},
		{
			name:          "неизвестный тип правила",
			rule:          model.AlertRule{Type: "UNKNOWN", Threshold: 50},
			telemetry:     model.Telemetry{SpeedKmh: 100},
			wantTriggered: false,
			wantVal:       0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			triggered, val, msg := checkRuleCondition(tt.rule, tt.telemetry)
			if triggered != tt.wantTriggered {
				t.Errorf("got triggered=%v, want %v", triggered, tt.wantTriggered)
			}
			if val != tt.wantVal {
				t.Errorf("got val=%f, want %f", val, tt.wantVal)
			}
			if triggered && msg == "" {
				t.Errorf("ожидалось непустое сообщение алерта")
			}
		})
	}
}

// TestAlertService_CreateRule проверяет создание и валидацию правил алертов
func TestAlertService_CreateRule(t *testing.T) {
	log := logger.NewStdLogger(logger.DebugLevel)
	ruleRepo := &mockAlertRuleRepo{}
	svc := NewAlertService(nil, ruleRepo, nil, nil, nil, nil, log)

	tests := []struct {
		name    string
		rule    model.AlertRule
		wantErr error
	}{
		{
			name: "успешное создание правила превышения скорости",
			rule: model.AlertRule{
				OrganizationID: 1,
				Name:           "Превышение 90",
				Type:           model.AlertRuleSpeedExceed,
				Threshold:      90,
				Severity:       model.SeverityLevelHigh,
			},
			wantErr: nil,
		},
		{
			name: "успешное создание правила низкого топлива",
			rule: model.AlertRule{
				OrganizationID: 1,
				Name:           "Низкое топливо",
				Type:           model.AlertRuleLowFuel,
				Threshold:      10,
				Severity:       model.SeverityLevelMedium,
			},
			wantErr: nil,
		},
		{
			name: "невалидный ID организации",
			rule: model.AlertRule{
				OrganizationID: 0,
				Name:           "Тест",
				Type:           model.AlertRuleSpeedExceed,
				Threshold:      80,
				Severity:       model.SeverityLevelLow,
			},
			wantErr: model.ErrInvalidOrganizationID,
		},
		{
			name: "пустое имя правила",
			rule: model.AlertRule{
				OrganizationID: 1,
				Name:           "",
				Type:           model.AlertRuleSpeedExceed,
				Threshold:      80,
				Severity:       model.SeverityLevelLow,
			},
			wantErr: model.ErrInvalidName,
		},
		{
			name: "отрицательный порог правила",
			rule: model.AlertRule{
				OrganizationID: 1,
				Name:           "Тест",
				Type:           model.AlertRuleSpeedExceed,
				Threshold:      -5,
				Severity:       model.SeverityLevelLow,
			},
			wantErr: model.ErrInvalidThreshold,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.CreateRule(context.Background(), tt.rule)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("got error %v, want %v", err, tt.wantErr)
			}
		})
	}

	// Проверка ошибки репозитория
	ruleRepoErr := &mockAlertRuleRepo{createErr: errors.New("db error")}
	svcErr := NewAlertService(nil, ruleRepoErr, nil, nil, nil, nil, log)
	validRule := model.AlertRule{OrganizationID: 1, Name: "Правило", Type: model.AlertRuleSpeedExceed, Threshold: 60, Severity: model.SeverityLevelLow}
	_, err := svcErr.CreateRule(context.Background(), validRule)
	if err == nil {
		t.Errorf("ожидалась ошибка базы данных при создании правила")
	}
}

// TestAlertService_RuleCRUD проверяет операции получения списка, обновления и удаления правил
func TestAlertService_RuleCRUD(t *testing.T) {
	log := logger.NewStdLogger(logger.DebugLevel)
	ruleRepo := &mockAlertRuleRepo{
		rule: model.AlertRule{
			ID:             5,
			OrganizationID: 1,
			Name:           "Порог скорости",
			Type:           model.AlertRuleSpeedExceed,
			Threshold:      100,
			Severity:       model.SeverityLevelCritical,
		},
		rules: []model.AlertRule{
			{ID: 5, OrganizationID: 1},
		},
	}
	svc := NewAlertService(nil, ruleRepo, nil, nil, nil, nil, log)

	// GetRuleByID
	rule, err := svc.GetRuleByID(context.Background(), 5)
	if err != nil || rule.ID != 5 {
		t.Fatalf("ошибка GetRuleByID: %v", err)
	}

	// GetRulesList
	rules, err := svc.GetRulesList(context.Background(), model.AlertRuleFilter{})
	if err != nil || len(rules) != 1 {
		t.Fatalf("ошибка GetRulesList: %v", err)
	}

	// UpdateRule
	upd := model.AlertRule{
		ID:             5,
		OrganizationID: 1,
		Name:           "Обновлённое имя",
		Type:           model.AlertRuleSpeedExceed,
		Threshold:      110,
		Severity:       model.SeverityLevelCritical,
	}
	updated, err := svc.UpdateRule(context.Background(), upd)
	if err != nil || updated.Threshold != 110 {
		t.Fatalf("ошибка UpdateRule: %v", err)
	}

	// UpdateRule с невалидным именем
	updInvalid := upd
	updInvalid.Name = ""
	_, err = svc.UpdateRule(context.Background(), updInvalid)
	if !errors.Is(err, model.ErrInvalidName) {
		t.Errorf("ожидалась ошибка ErrInvalidName при обновлении, получено %v", err)
	}

	// DeleteRuleByID
	deleted, err := svc.DeleteRuleByID(context.Background(), 5)
	if err != nil || deleted.ID != 5 {
		t.Fatalf("ошибка DeleteRuleByID: %v", err)
	}
}

// TestAlertService_EvaluateTelemetry_FireAlert проверяет создание нового алерта и Outbox-задач при нарушении порога
func TestAlertService_EvaluateTelemetry_FireAlert(t *testing.T) {
	log := logger.NewStdLogger(logger.DebugLevel)
	ruleRepo := &mockAlertRuleRepo{
		activeRules: []model.AlertRule{
			{
				ID:             1,
				OrganizationID: 1,
				Type:           model.AlertRuleSpeedExceed,
				Threshold:      80,
				Severity:       model.SeverityLevelHigh,
			},
		},
	}
	alertRepo := &mockAlertRepo{
		getActiveErr: model.ErrNotFound, // активного алерта ещё нет
	}
	channelRepo := &mockChannelRepo{
		channels: []model.UserNotificationChannel{
			{ID: 10, UserID: 100, Type: model.NotificationChannelTg},
			{ID: 11, UserID: 101, Type: model.NotificationChannelEmail},
		},
	}
	notificationRepo := &mockNotificationRepo{}
	txManager := &alertTestTxManager{}
	repoFactory := &alertTestRepoFactory{
		alertRepo:        alertRepo,
		notificationRepo: notificationRepo,
	}

	svc := NewAlertService(alertRepo, ruleRepo, channelRepo, notificationRepo, txManager, repoFactory, log)

	point := model.Telemetry{
		OrganizationID: 1,
		VehicleID:      42,
		SpeedKmh:       95,
	}

	err := svc.EvaluateTelemetry(context.Background(), point)
	if err != nil {
		t.Fatalf("неожиданная ошибка EvaluateTelemetry: %v", err)
	}

	if len(alertRepo.createdAlerts) != 1 {
		t.Fatalf("ожидался 1 созданный алерт, создано %d", len(alertRepo.createdAlerts))
	}
	created := alertRepo.createdAlerts[0]
	if created.VehicleID != 42 || created.Status != model.AlertStatusFired || created.Severity != model.SeverityLevelHigh {
		t.Errorf("некорректные параметры созданного алерта: %+v", created)
	}

	if len(notificationRepo.createdBatches) != 1 {
		t.Fatalf("ожидалась 1 пачка нотификаций в Outbox, получено %d", len(notificationRepo.createdBatches))
	}
	batch := notificationRepo.createdBatches[0]
	if len(batch) != 2 {
		t.Errorf("ожидалось 2 нотификации для каналов, получено %d", len(batch))
	}
	if batch[0].Status != model.NotificationStatusPending {
		t.Errorf("статус нотификации = %s, ожидался PENDING", batch[0].Status)
	}
}

// TestAlertService_EvaluateTelemetry_Deduplication проверяет, что при повторном нарушении дублирующий алерт не создаётся
func TestAlertService_EvaluateTelemetry_Deduplication(t *testing.T) {
	log := logger.NewStdLogger(logger.DebugLevel)
	ruleRepo := &mockAlertRuleRepo{
		activeRules: []model.AlertRule{
			{
				ID:             1,
				OrganizationID: 1,
				Type:           model.AlertRuleSpeedExceed,
				Threshold:      80,
				Severity:       model.SeverityLevelHigh,
			},
		},
	}
	alertRepo := &mockAlertRepo{
		activeAlert: model.Alert{ID: 99, Status: model.AlertStatusFired}, // активный алерт уже есть в базе
	}
	channelRepo := &mockChannelRepo{}
	notificationRepo := &mockNotificationRepo{}
	txManager := &alertTestTxManager{}
	repoFactory := &alertTestRepoFactory{
		alertRepo:        alertRepo,
		notificationRepo: notificationRepo,
	}

	svc := NewAlertService(alertRepo, ruleRepo, channelRepo, notificationRepo, txManager, repoFactory, log)

	point := model.Telemetry{
		OrganizationID: 1,
		VehicleID:      42,
		SpeedKmh:       110,
	}

	err := svc.EvaluateTelemetry(context.Background(), point)
	if err != nil {
		t.Fatalf("неожиданная ошибка EvaluateTelemetry: %v", err)
	}

	if len(alertRepo.createdAlerts) != 0 {
		t.Errorf("алерт не должен создаваться повторно при наличии активного алерта")
	}
	if len(alertRepo.resolvedIDs) != 0 {
		t.Errorf("алерт не должен закрываться, пока скорость превышена")
	}
}

// TestAlertService_EvaluateTelemetry_AutoResolve проверяет автозакрытие активного алерта при нормализации показателей
func TestAlertService_EvaluateTelemetry_AutoResolve(t *testing.T) {
	log := logger.NewStdLogger(logger.DebugLevel)
	ruleRepo := &mockAlertRuleRepo{
		activeRules: []model.AlertRule{
			{
				ID:             1,
				OrganizationID: 1,
				Type:           model.AlertRuleSpeedExceed,
				Threshold:      80,
				Severity:       model.SeverityLevelHigh,
			},
		},
	}
	alertRepo := &mockAlertRepo{
		activeAlert: model.Alert{ID: 77, Status: model.AlertStatusFired}, // активный алерт существует
	}
	channelRepo := &mockChannelRepo{}
	notificationRepo := &mockNotificationRepo{}
	txManager := &alertTestTxManager{}
	repoFactory := &alertTestRepoFactory{
		alertRepo:        alertRepo,
		notificationRepo: notificationRepo,
	}

	svc := NewAlertService(alertRepo, ruleRepo, channelRepo, notificationRepo, txManager, repoFactory, log)

	// Скорость снизилась до 60 км/ч (ниже порога 80)
	point := model.Telemetry{
		OrganizationID: 1,
		VehicleID:      42,
		SpeedKmh:       60,
	}

	err := svc.EvaluateTelemetry(context.Background(), point)
	if err != nil {
		t.Fatalf("неожиданная ошибка EvaluateTelemetry: %v", err)
	}

	if len(alertRepo.createdAlerts) != 0 {
		t.Errorf("новый алерт не должен создаваться")
	}
	if len(alertRepo.resolvedIDs) != 1 || alertRepo.resolvedIDs[0] != 77 {
		t.Errorf("ожидалось закрытие алерта 77, закрыты: %v", alertRepo.resolvedIDs)
	}
}

// TestAlertService_WorkerQueue проверяет асинхронную обработку точек телеметрии через воркер-пул
func TestAlertService_WorkerQueue(t *testing.T) {
	log := logger.NewStdLogger(logger.DebugLevel)
	ruleRepo := &mockAlertRuleRepo{
		activeRules: []model.AlertRule{
			{
				ID:             1,
				OrganizationID: 1,
				Type:           model.AlertRuleSpeedExceed,
				Threshold:      80,
				Severity:       model.SeverityLevelHigh,
			},
		},
	}
	alertRepo := &mockAlertRepo{
		getActiveErr: model.ErrNotFound,
	}
	channelRepo := &mockChannelRepo{}
	notificationRepo := &mockNotificationRepo{}
	txManager := &alertTestTxManager{}
	repoFactory := &alertTestRepoFactory{
		alertRepo:        alertRepo,
		notificationRepo: notificationRepo,
	}

	svc := NewAlertService(alertRepo, ruleRepo, channelRepo, notificationRepo, txManager, repoFactory, log)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	svc.Start(ctx, 2)

	// Отправляем точку с превышением скорости в очередь
	svc.Enqueue(context.Background(), model.Telemetry{
		OrganizationID: 1,
		VehicleID:      10,
		SpeedKmh:       120,
	})

	// Ожидаем обработки воркером
	var processed bool
	for i := 0; i < 50; i++ {
		alertRepo.mu.Lock()
		count := len(alertRepo.createdAlerts)
		alertRepo.mu.Unlock()
		if count > 0 {
			processed = true
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	if !processed {
		t.Fatalf("воркер не обработал телеметрию из очереди вовремя")
	}

	svc.Stop()
}
