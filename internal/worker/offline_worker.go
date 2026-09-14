package worker

import (
	"context"
	"fleettrack/internal/logger"
	"fleettrack/internal/model"
	"fleettrack/internal/repository"
	"fleettrack/internal/service"
	"fmt"
	"sync"
	"time"
)

// OfflineWorker периодически проверяет автомобили с активными трекерами на отсутствие телеметрии
type OfflineWorker struct {
	alertRepo     repository.AlertRepository
	ruleRepo      repository.AlertRuleRepository
	alertService  *service.AlertService
	logger        logger.Logger
	checkInterval time.Duration
	wg            sync.WaitGroup
}

// NewOfflineWorker создаёт новый экземпляр воркера проверки оффлайн-устройств
func NewOfflineWorker(
	alertRepo repository.AlertRepository,
	ruleRepo repository.AlertRuleRepository,
	alertService *service.AlertService,
	l logger.Logger,
	interval time.Duration,
) *OfflineWorker {
	if interval < 10*time.Second {
		interval = 1 * time.Minute
	}
	return &OfflineWorker{
		alertRepo:     alertRepo,
		ruleRepo:      ruleRepo,
		alertService:  alertService,
		logger:        l,
		checkInterval: interval,
	}
}

// Start запускает периодический фоновый опрос в отдельной горутине
func (w *OfflineWorker) Start(ctx context.Context) {
	w.wg.Add(1)
	go w.run(ctx)
}

// Stop ожидает завершения фоновой горутины проверки
func (w *OfflineWorker) Stop() {
	w.wg.Wait()
}

// run запускает таймер периодической проверки
func (w *OfflineWorker) run(ctx context.Context) {
	defer w.wg.Done()

	ticker := time.NewTicker(w.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.checkOffline(ctx)
		}
	}
}

// checkOffline выполняет поиск замолчавших устройств по правилам DEVICE_OFFLINE и генерирует алерты
func (w *OfflineWorker) checkOffline(ctx context.Context) {
	enabled := true
	rules, err := w.ruleRepo.GetList(ctx, model.AlertRuleFilter{
		Enabled: &enabled,
	})
	if err != nil {
		w.logger.Error(fmt.Sprintf("failed to get alert rules for offline check: %s", err.Error()))
		return
	}
	for _, rule := range rules {
		if rule.Type != model.AlertRuleDeviceOffline {
			continue
		}
		offlineVehicles, err := w.alertRepo.FindOfflineVehicles(ctx, rule.Threshold)
		if err != nil {
			w.logger.Error(fmt.Sprintf("failed to find offline vehicles for rule %d: %s", rule.ID, err.Error()))
			continue
		}
		for _, vehicle := range offlineVehicles {
			if vehicle.OrganizationID != rule.OrganizationID {
				continue
			}
			if err := w.alertService.FireOfflineAlert(ctx, vehicle, rule); err != nil {
				w.logger.Error(fmt.Sprintf("failed to fire offline alert for vehicle %d: %s", vehicle.VehicleID, err.Error()))
			}
		}
	}
}
