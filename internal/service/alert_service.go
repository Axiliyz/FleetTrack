// Package service содержит бизнес-логику приложения
package service

import (
	"context"
	"errors"
	"fleettrack/internal/database"
	"fleettrack/internal/logger"
	"fleettrack/internal/metrics"
	"fleettrack/internal/model"
	"fleettrack/internal/repository"
	"fleettrack/internal/repository/factory"
	"fleettrack/internal/transaction"
	"fmt"
	"sync"
	"time"
)

// AlertService управляет жизненным циклом алертов, оценкой телеметрии и фоновой очередью
type AlertService struct {
	alertRepo        repository.AlertRepository
	ruleRepo         repository.AlertRuleRepository
	channelRepo      repository.NotificationChannelRepository
	notificationRepo repository.AlertNotificationRepository
	txManager        transaction.TransactionManager
	repoFactory      factory.RepositoryFactory
	logger           logger.Logger
	queue            chan model.Telemetry
	queueMu          sync.RWMutex
	queueClosed      bool
	wg               sync.WaitGroup
}

// NewAlertService создаёт новый экземпляр AlertService с буферизированной очередью обработки
func NewAlertService(a repository.AlertRepository, r repository.AlertRuleRepository, c repository.NotificationChannelRepository, n repository.AlertNotificationRepository, tx transaction.TransactionManager, repoFactory factory.RepositoryFactory, l logger.Logger, queueSize int) *AlertService {
	return &AlertService{
		alertRepo:        a,
		ruleRepo:         r,
		channelRepo:      c,
		notificationRepo: n,
		txManager:        tx,
		repoFactory:      repoFactory,
		logger:           l,
		queue:            make(chan model.Telemetry, queueSize),
	}
}

// EvaluateTelemetry проверяет точку телеметрии на соответствие активным правилам организации
func (s *AlertService) EvaluateTelemetry(ctx context.Context, t model.Telemetry) error {
	evalStart := time.Now()
	defer func() { metrics.RecordTelemetryStage("alert_eval_total", time.Since(evalStart).Seconds()) }()
	stageStart := time.Now()
	activeRules, err := s.ruleRepo.GetActiveRulesByOrg(ctx, t.OrganizationID)
	metrics.RecordTelemetryStage("alert_eval_rules", time.Since(stageStart).Seconds())
	if err != nil {
		return err
	}
	for _, r := range activeRules {
		triggered, val, msg := checkRuleCondition(r, t)

		stageStart = time.Now()
		activeAlert, err := s.alertRepo.GetActiveAlert(ctx, t.VehicleID, r.ID)
		metrics.RecordTelemetryStage("alert_eval_active", time.Since(stageStart).Seconds())
		if err != nil && !errors.Is(err, model.ErrNotFound) {
			return err
		}
		hasActiveAlert := err == nil

		if triggered && !hasActiveAlert {
			stageStart = time.Now()
			if err := s.fireAlert(ctx, t, r, val, msg); err != nil {
				return err
			}
			metrics.RecordTelemetryStage("alert_eval_fire", time.Since(stageStart).Seconds())
		}

		if !triggered && hasActiveAlert {
			_, err := s.alertRepo.Resolve(ctx, activeAlert.ID, nil)
			if err != nil {
				return err
			}
			s.logger.Info(fmt.Sprintf("alert %d resolved automatically for vehicle %d", activeAlert.ID, t.VehicleID))
		}
	}
	return nil
}

// checkRuleCondition проверяет выполнение условия правила для точки телеметрии
func checkRuleCondition(r model.AlertRule, t model.Telemetry) (triggered bool, val float64, msg string) {
	switch r.Type {
	case model.AlertRuleSpeedExceed:
		if t.SpeedKmh >= r.Threshold {
			return true, t.SpeedKmh, fmt.Sprintf("Превышена скорость: %.1f км/ч (порог %.1f)", t.SpeedKmh, r.Threshold)
		}
		return false, t.SpeedKmh, ""
	case model.AlertRuleLowFuel:
		if t.Fuel != nil && float64(*t.Fuel) <= r.Threshold {
			return true, float64(*t.Fuel), fmt.Sprintf("Низкий уровень топлива: %.1f%% (порог %.1f%%)", *t.Fuel, r.Threshold)
		}
		if t.Fuel != nil {
			return false, float64(*t.Fuel), ""
		}
		return false, 0, ""
	}
	return false, 0, ""
}

// fireAlert атомарно в транзакции сохраняет алерт и формирует задачи в Outbox
func (s *AlertService) fireAlert(ctx context.Context, t model.Telemetry, r model.AlertRule, val float64, msg string) error {
	alert := model.Alert{
		OrganizationID: t.OrganizationID,
		VehicleID:      t.VehicleID,
		RuleID:         r.ID,
		Type:           r.Type,
		Severity:       r.Severity,
		Status:         model.AlertStatusFired,
		Message:        msg,
		Value:          &val,
	}

	channels, err := s.channelRepo.GetChannelsForAlert(ctx, t.OrganizationID, r.Severity)
	if err != nil {
		return err
	}
	err = s.txManager.WithTx(ctx, func(tx database.DBTX) error {
		repos := s.repoFactory.New(tx)
		err := repos.Alert.AcquireLock(ctx, t.VehicleID, r.ID)
		if err != nil {
			return err
		}
		_, err = repos.Alert.GetActiveAlert(ctx, t.VehicleID, r.ID)
		if err != nil && !errors.Is(err, model.ErrNotFound) {
			return err
		}
		if err == nil {
			return nil
		}
		if err := repos.Alert.Create(ctx, &alert); err != nil {
			return err
		}

		if len(channels) == 0 {
			return nil
		}

		notifications := make([]model.AlertNotification, 0, len(channels))
		for _, ch := range channels {
			notifications = append(notifications, model.AlertNotification{
				AlertID:   alert.ID,
				ChannelID: ch.ID,
				Status:    model.NotificationStatusPending,
			})
		}

		return repos.AlertNotification.CreateBatch(ctx, notifications)
	})
	if err != nil {
		return err
	}
	s.logger.Info(fmt.Sprintf("alert fired: ID=%d, vehicle=%d, rule=%d, type=%s, severity=%s", alert.ID, alert.VehicleID, alert.RuleID, alert.Type, alert.Severity))
	return nil
}

// Enqueue ставит точку телеметрии в очередь оценки алертов. Если очередь заполнена,
// ждёт свободного места, но не дольше, чем живёт ctx запроса. Точки, не попавшие
// в очередь (отмена ctx или остановка сервиса), учитываются в метрике и логируются.
func (s *AlertService) Enqueue(ctx context.Context, t model.Telemetry) {
	s.queueMu.RLock()
	defer s.queueMu.RUnlock()
	if s.queueClosed {
		s.dropFromQueue(t, "alert service stopped")
		return
	}
	select {
	case s.queue <- t:
	case <-ctx.Done():
		s.dropFromQueue(t, ctx.Err().Error())
	}
}

func (s *AlertService) dropFromQueue(t model.Telemetry, reason string) {
	metrics.RecordAlertQueueDropped()
	s.logger.Warn(fmt.Sprintf("telemetry %d of vehicle %d not queued for alert evaluation: %s", t.TelemetryID, t.VehicleID, reason))
}

// QueueLength возвращает число точек, ожидающих оценки алертов.
func (s *AlertService) QueueLength() int {
	return len(s.queue)
}

// Start запускает воркеры оценки алертов. ctx используется для запросов к БД и
// должен оставаться живым до завершения Stop, иначе очередь не будет дочитана.
func (s *AlertService) Start(ctx context.Context, workers int) {
	for i := 0; i < workers; i++ {
		s.wg.Add(1)
		go s.worker(ctx)
	}
}

// worker разбирает очередь до её закрытия в Stop
func (s *AlertService) worker(ctx context.Context) {
	defer s.wg.Done()
	for t := range s.queue {
		if err := s.EvaluateTelemetry(ctx, t); err != nil {
			s.logger.Error(fmt.Sprintf("failed to evaluate telemetry, err: %s, vehicle: %d", err, t.VehicleID))
		}
	}
}

// Stop закрывает очередь для новых точек, дожидается, пока воркеры обработают
// уже поставленные, и возвращает управление. Повторный вызов безопасен.
func (s *AlertService) Stop() {
	s.queueMu.Lock()
	if !s.queueClosed {
		s.queueClosed = true
		close(s.queue)
	}
	s.queueMu.Unlock()
	s.wg.Wait()
}

// validateAlertRule проверяет корректность полей конфигурации правила
func validateAlertRule(r model.AlertRule) error {
	if r.OrganizationID < 1 {
		return model.ErrInvalidOrganizationID
	}
	if r.Name == "" {
		return model.ErrInvalidName
	}
	if r.Threshold < 0 {
		return model.ErrInvalidThreshold
	}

	return nil
}

// CreateRule валидирует и сохраняет новое правило алертов
func (s *AlertService) CreateRule(ctx context.Context, r model.AlertRule) (model.AlertRule, error) {
	if err := validateAlertRule(r); err != nil {
		return model.AlertRule{}, err
	}

	err := s.ruleRepo.Create(ctx, &r)
	if err != nil {
		return model.AlertRule{}, err
	}
	message := fmt.Sprintf(
		`rule created: 
	ID: %d 
	OrgID: %d 
	type: %s 
	name: %s 
	threshold: %f 
	severity: %s 
	enabled: %t`,
		r.ID, r.OrganizationID, r.Type, r.Name, r.Threshold, r.Severity, r.Enabled)
	s.logger.Info(message)
	return r, nil
}

// GetRuleByID возвращает правило алертов по его ID
func (s *AlertService) GetRuleByID(ctx context.Context, id int) (model.AlertRule, error) {
	return s.ruleRepo.GetByID(ctx, id)
}

// GetRulesList возвращает список правил алертов по переданному фильтру
func (s *AlertService) GetRulesList(ctx context.Context, filter model.AlertRuleFilter) ([]model.AlertRule, error) {
	return s.ruleRepo.GetList(ctx, filter)
}

// DeleteRuleByID удаляет правило алертов по его идентификатору
func (s *AlertService) DeleteRuleByID(ctx context.Context, id int) (model.AlertRule, error) {
	return s.ruleRepo.DeleteByID(ctx, id)
}

// AcknowledgeAlert переводит статус алерта в ACKNOWLEDGED
func (s *AlertService) AcknowledgeAlert(ctx context.Context, alertID, userID int) (model.Alert, error) {
	alert, err := s.alertRepo.AcknowledgeAlert(ctx, alertID, userID)
	if err != nil {
		return model.Alert{}, err
	}
	s.logger.Info(fmt.Sprintf("alert %d acknowledged by user %d", alertID, userID))
	return alert, nil
}

// FireOfflineAlert создаёт алерт о потере связи с устройством, если такого открытого алерта ещё нет
func (s *AlertService) FireOfflineAlert(ctx context.Context, info model.OfflineVehicleInfo, r model.AlertRule) error {
	message := fmt.Sprintf("Устройство ID %d не выходит на связь %.0f мин (порог %.0f мин)",
		info.DeviceID, info.MinutesOffline, r.Threshold)
	val := info.MinutesOffline

	return s.fireAlert(ctx, model.Telemetry{
		OrganizationID: info.OrganizationID,
		VehicleID:      info.VehicleID,
	}, r, val, message)
}
