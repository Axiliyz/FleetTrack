// Package service содержит бизнес-логику приложения
package service

import (
	"context"
	"errors"
	"fleettrack/internal/database"
	"fleettrack/internal/logger"
	"fleettrack/internal/model"
	"fleettrack/internal/repository"
	"fleettrack/internal/repository/factory"
	"fleettrack/internal/transaction"
	"fmt"
	"sync"
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
	wg               sync.WaitGroup
}

// NewAlertService создаёт новый экземпляр AlertService с буферизированной очередью обработки
func NewAlertService(a repository.AlertRepository, r repository.AlertRuleRepository, c repository.NotificationChannelRepository, n repository.AlertNotificationRepository, tx transaction.TransactionManager, repoFactory factory.RepositoryFactory, l logger.Logger) *AlertService {
	return &AlertService{
		alertRepo:        a,
		ruleRepo:         r,
		channelRepo:      c,
		notificationRepo: n,
		txManager:        tx,
		repoFactory:      repoFactory,
		logger:           l,
		queue:            make(chan model.Telemetry, 10000),
	}
}

// EvaluateTelemetry проверяет точку телеметрии на соответствие активным правилам организации
func (s *AlertService) EvaluateTelemetry(ctx context.Context, t model.Telemetry) error {
	activeRules, err := s.ruleRepo.GetActiveRulesByOrg(ctx, t.OrganizationID)
	if err != nil {
		return err
	}
	for _, r := range activeRules {
		triggered, val, msg := checkRuleCondition(r, t)

		activeAlert, err := s.alertRepo.GetActiveAlert(ctx, t.VehicleID, r.ID)
		if err != nil && !errors.Is(err, model.ErrNotFound) {
			return err
		}
		hasActiveAlert := err == nil

		if triggered && !hasActiveAlert {
			if err := s.fireAlert(ctx, t, r, val, msg); err != nil {
				return err
			}
		}

		if !triggered && hasActiveAlert {
			_, err := s.alertRepo.Resolve(ctx, activeAlert.ID, nil)
			if err != nil {
				return err
			}
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
	return s.txManager.WithTx(ctx, func(tx database.DBTX) error {
		repos := s.repoFactory.New(tx)
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
}

// Enqueue добавляет точку телеметрии в неблокирующую очередь обработки алертов
func (s *AlertService) Enqueue(t model.Telemetry) {
	select {
	case s.queue <- t:
	default:
		s.logger.Warn(fmt.Sprintf("alert queue is full, dropping telemetry point for vehicle %d", t.VehicleID))
	}
}

// Start запускает указанное количество горутин-воркеров для разбора очереди
func (s *AlertService) Start(ctx context.Context, workers int) {
	for i := 0; i < workers; i++ {
		s.wg.Add(1)
		go s.worker(ctx)
	}
}

// worker обрабатывает входящие точки телеметрии из очереди
func (s *AlertService) worker(ctx context.Context) {
	defer s.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case t, ok := <-s.queue:
			if !ok {
				return
			}
			if err := s.EvaluateTelemetry(ctx, t); err != nil {
				s.logger.Error(fmt.Sprintf("failed to evaluate telemetry, err: %s, vehicle: %d", err, t.VehicleID))
			}
		}
	}
}

// Stop корректно останавливает воркеры и дожидается опустошения очереди
func (s *AlertService) Stop() {
	close(s.queue)
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

// UpdateRule валидирует и обновляет существующее правило алертов
func (s *AlertService) UpdateRule(ctx context.Context, r model.AlertRule) (model.AlertRule, error) {
	if err := validateAlertRule(r); err != nil {
		return model.AlertRule{}, err
	}
	return s.ruleRepo.Update(ctx, r)
}

// DeleteRuleByID удаляет правило алертов по его идентификатору
func (s *AlertService) DeleteRuleByID(ctx context.Context, id int) (model.AlertRule, error) {
	return s.ruleRepo.DeleteByID(ctx, id)
}
