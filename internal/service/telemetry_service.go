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
	"time"
)

// TelemetryBatchConfig задаёт параметры пакетной записи телеметрии
type TelemetryBatchConfig struct {
	BufferSize int
	MaxSize    int
	MaxWait    time.Duration
}

// TelemetryService обрабатывает и валидирует телеметрию
type TelemetryService struct {
	repository    repository.TelemetryRepository
	logger        logger.Logger
	txManager     transaction.TransactionManager
	repoFactory   factory.RepositoryFactory
	motionService MotionService
	alertService  *AlertService
	batcher       *batcher[model.Telemetry, model.Telemetry]
	batcherDone   chan struct{}
}

// NewTelemetryService создаёт сервис телеметрии. Запись идёт пачками, поэтому
// перед использованием нужно вызвать StartBatcher.
func NewTelemetryService(r repository.TelemetryRepository, logger logger.Logger, tx transaction.TransactionManager, rf factory.RepositoryFactory, ms MotionService, as *AlertService, batchCfg TelemetryBatchConfig) *TelemetryService {
	s := &TelemetryService{
		repository:    r,
		logger:        logger,
		txManager:     tx,
		repoFactory:   rf,
		motionService: ms,
		alertService:  as,
		batcherDone:   make(chan struct{}),
	}
	s.batcher = newBatcher[model.Telemetry, model.Telemetry](batchCfg.BufferSize, batchCfg.MaxSize, batchCfg.MaxWait, s.flushBatch)
	return s
}

// StartBatcher запускает пакетную запись телеметрии. По отмене ctx батчер дописывает
// накопленные записи и завершается; дождаться этого можно через WaitBatcher.
func (s *TelemetryService) StartBatcher(ctx context.Context) {
	go func() {
		defer close(s.batcherDone)
		s.batcher.Run(ctx)
	}()
}

// WaitBatcher блокируется, пока батчер, запущенный через StartBatcher, не завершится.
func (s *TelemetryService) WaitBatcher() {
	<-s.batcherDone
}

// validateTelemetry проверяет входные данные телеметрии.
//
// Проверяет:
// - DeviceID >= 1
// - VehicleID >= 1
// - Lat в диапазоне [-90, 90]
// - Lon в диапазоне [-180, 180]
// - Fuel в диапазоне [0, 100]
func validateTelemetry(t model.Telemetry) error {
	if t.DeviceID < 1 {
		return model.ErrInvalidDeviceID
	}
	if t.VehicleID < 1 {
		return model.ErrInvalidVehicleID
	}
	if t.Lat < -90 || t.Lat > 90 || t.Lon < -180 || t.Lon > 180 {
		return model.ErrInvalidCoords
	}
	if t.Fuel != nil && (*t.Fuel < 0.0 || *t.Fuel > 100.0) {
		return model.ErrInvalidFuel
	}
	return nil
}

// resolveActiveTrip находит активный (RUNNING) рейс машины.
// Возвращает model.ErrNoActiveTrip, если такого рейса нет.
func resolveActiveTrip(ctx context.Context, repos factory.Repositories, vehicleID int) (model.Trip, error) {
	running := model.TripStatusRunning
	filter := &model.TripFilter{VehicleID: &vehicleID, Status: &running, Limit: 1}
	trips, err := repos.Trip.GetListTrips(ctx, filter)
	if err != nil {
		return model.Trip{}, err
	}
	if len(trips) == 0 {
		return model.Trip{}, model.ErrNoActiveTrip
	}
	return trips[0], nil
}

// resolveVehicleOrg позволяет получить OrgID по машине
// Возвращает ID организации или ошибку
func resolveVehicleOrg(ctx context.Context, repos factory.Repositories, vehicleID int) (int, error) {
	vehicle, err := repos.Vehicle.GetByID(ctx, vehicleID)
	if err != nil {
		return 0, err
	}
	return vehicle.OrganizationID, nil
}

// resolveLastTelemetry находит предыдущую точку телеметрии машины.
// Если точки ещё не было - возвращает (nil, nil), это не ошибка.
func resolveLastTelemetry(ctx context.Context, repos factory.Repositories, vehicleID int) (*model.Telemetry, error) {
	found, err := repos.Telemetry.GetLastByVehicle(ctx, vehicleID)
	if err != nil {
		if errors.Is(err, model.ErrNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &found, nil
}

// applyMotion считает пройденное расстояние и скорость по предыдущей точке
// и прибавляет расстояние к рейсу. Если предыдущей точки не было - ничего не делает.
func (s *TelemetryService) applyMotion(ctx context.Context, repos factory.Repositories, last *model.Telemetry, trip model.Trip, t *model.Telemetry) error {
	if last == nil {
		return nil
	}
	motion, err := s.motionService.Calculate(last, *t)
	if err != nil {
		return err
	}
	t.DistanceKm = motion.DistanceKm
	t.SpeedKmh = motion.SpeedKmh
	_, err = repos.Trip.UpdateTripStats(ctx, trip.ID, t.DistanceKm, t.SpeedKmh)
	return err
}

// ProcessTelemetry валидирует телеметрию и сохраняет её в составе пачки.
// Возвращает сохранённую телеметрию или ошибку.
//
// Если DeviceTimestamp не указан - устанавливает текущее время.
// ReceivedAt всегда ставится в текущее время.
func (s *TelemetryService) ProcessTelemetry(ctx context.Context, t model.Telemetry) (model.Telemetry, error) {
	serviceStart := time.Now()
	defer func() { metrics.RecordTelemetryStage("service_total", time.Since(serviceStart).Seconds()) }()

	if err := validateTelemetry(t); err != nil {
		return model.Telemetry{}, err
	}

	if t.DeviceTimestamp.IsZero() {
		t.DeviceTimestamp = time.Now()
	}
	t.ReceivedAt = time.Now()

	saved, err := s.batcher.Submit(ctx, t)
	if err != nil {
		return model.Telemetry{}, err
	}

	if s.alertService != nil {
		enqueueStart := time.Now()
		s.alertService.Enqueue(ctx, saved)
		metrics.RecordTelemetryStage("alert_enqueue", time.Since(enqueueStart).Seconds())
	}

	s.logger.Debug(fmt.Sprintf("telemetry stored: id=%d device=%d vehicle=%d", saved.TelemetryID, saved.DeviceID, saved.VehicleID))
	return saved, nil
}

// flushBatch пишет пачку одной транзакцией. Каждая запись выполняется под своим
// SAVEPOINT: её ошибка откатывает только её изменения и возвращается только её
// отправителю. Ошибка всей пачки возвращается, лишь если сломалась сама транзакция.
func (s *TelemetryService) flushBatch(ctx context.Context, batch []model.Telemetry) ([]itemResult[model.Telemetry], error) {
	results := make([]itemResult[model.Telemetry], len(batch))
	err := s.txManager.WithTx(ctx, func(tx database.DBTX) error {
		repos := s.repoFactory.New(tx)
		for i, t := range batch {
			if _, err := tx.Exec(ctx, "SAVEPOINT telemetry_item"); err != nil {
				return err
			}
			saved, itemErr := s.saveInTx(ctx, repos, t)
			if itemErr != nil {
				if _, err := tx.Exec(ctx, "ROLLBACK TO SAVEPOINT telemetry_item"); err != nil {
					return err
				}
				results[i] = itemResult[model.Telemetry]{err: itemErr}
				continue
			}
			if _, err := tx.Exec(ctx, "RELEASE SAVEPOINT telemetry_item"); err != nil {
				return err
			}
			results[i] = itemResult[model.Telemetry]{value: saved}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return results, nil
}

// saveInTx проверяет право записи и сохраняет одну точку телеметрии в рамках транзакции.
// Проверки идут до любых изменений: машина должна принадлежать организации отправителя,
// а устройство - быть назначено этой машине.
func (s *TelemetryService) saveInTx(ctx context.Context, repos factory.Repositories, t model.Telemetry) (model.Telemetry, error) {
	stageStart := time.Now()
	orgID, err := resolveVehicleOrg(ctx, repos, t.VehicleID)
	metrics.RecordTelemetryStage("vehicle_org", time.Since(stageStart).Seconds())
	if err != nil {
		return model.Telemetry{}, err
	}
	if t.OrganizationID != 0 && t.OrganizationID != orgID {
		return model.Telemetry{}, model.ErrNotFound
	}
	t.OrganizationID = orgID

	assignment, err := repos.Assignment.GetActiveAssignment(ctx, t.DeviceID)
	if err != nil && !errors.Is(err, model.ErrNotFound) {
		return model.Telemetry{}, err
	}
	if err != nil || assignment.VehicleID != t.VehicleID {
		return model.Telemetry{}, model.ErrDeviceNotAssigned
	}

	stageStart = time.Now()
	trip, err := resolveActiveTrip(ctx, repos, t.VehicleID)
	metrics.RecordTelemetryStage("active_trip", time.Since(stageStart).Seconds())
	if err != nil && !errors.Is(err, model.ErrNoActiveTrip) {
		return model.Telemetry{}, err
	}

	stageStart = time.Now()
	last, err := resolveLastTelemetry(ctx, repos, t.VehicleID)
	metrics.RecordTelemetryStage("last_telemetry", time.Since(stageStart).Seconds())
	if err != nil {
		return model.Telemetry{}, err
	}

	if trip.ID != 0 {
		t.TripID = trip.ID
		if err := s.applyMotion(ctx, repos, last, trip, &t); err != nil {
			s.logger.Warn(fmt.Sprintf("failed to apply motion: %s", err.Error()))
		}
	} else if last != nil && s.motionService != nil {
		if motion, err := s.motionService.Calculate(last, t); err == nil && motion != nil {
			t.DistanceKm = motion.DistanceKm
			t.SpeedKmh = motion.SpeedKmh
		}
	}

	stageStart = time.Now()
	err = repos.Telemetry.Save(ctx, &t)
	metrics.RecordTelemetryStage("save", time.Since(stageStart).Seconds())
	if err != nil {
		return model.Telemetry{}, err
	}

	stageStart = time.Now()
	err = repos.Vehicle.UpdateLastTelemetryAt(ctx, t.VehicleID, t.ReceivedAt)
	metrics.RecordTelemetryStage("update_vehicle", time.Since(stageStart).Seconds())
	if err != nil {
		return model.Telemetry{}, err
	}
	return t, nil
}

// GetTelemetryList используется в GET /telemetry
// Возвращает срез всех телеметрий(с возможностью фильтрации), либо ошибку
func (s *TelemetryService) GetTelemetryList(ctx context.Context, filter model.TelemetryFilter) ([]model.Telemetry, error) {
	if filter.From != nil && filter.To != nil && filter.From.After(*filter.To) {
		return nil, model.ErrInvalidTimestamp
	}
	if filter.FuelMin != nil && filter.FuelMax != nil &&
		*filter.FuelMin > *filter.FuelMax {
		return nil, model.ErrInvalidFuel
	}
	if filter.LatMin != nil && filter.LatMax != nil &&
		*filter.LatMin > *filter.LatMax {
		return nil, model.ErrInvalidCoords
	}

	if filter.LonMin != nil && filter.LonMax != nil &&
		*filter.LonMin > *filter.LonMax {
		return nil, model.ErrInvalidCoords
	}

	if filter.DeviceID != nil && *filter.DeviceID < 0 {
		return nil, model.ErrInvalidDeviceID
	}

	if filter.VehicleID != nil && *filter.VehicleID < 0 {
		return nil, model.ErrInvalidVehicleID
	}

	if filter.DriverID != nil && *filter.DriverID < 0 {
		return nil, model.ErrInvalidDriverID
	}

	if filter.TripID != nil && *filter.TripID < 0 {
		return nil, model.ErrInvalidTripID
	}

	if filter.OrganizationID != nil && *filter.OrganizationID < 0 {
		return nil, model.ErrInvalidOrganizationID
	}

	res, err := s.repository.GetList(ctx, filter)
	if err != nil {
		return nil, err
	}
	s.logger.Info("Got all the filtered telemetry")
	return res, nil
}

// GetTelemetryByID используется в GET /telemetry/{id}
// Возвращает запись по её ID, либо ошибку
func (s *TelemetryService) GetTelemetryByID(ctx context.Context, id int) (model.Telemetry, error) {
	res, err := s.repository.GetItemByID(ctx, id)
	if err != nil {
		return model.Telemetry{}, err
	}
	message := fmt.Sprintf("Got telemetry with id %d", id)
	s.logger.Info(message)
	return res, nil
}

// GetTelemetryByVehicle используется в GET /telemetry/vehicle/{id}
// Возвращает срез записей по машине по её ID, либо ошибку
func (s *TelemetryService) GetTelemetryByVehicle(ctx context.Context, id int, organizationID *int) ([]model.Telemetry, error) {
	res, err := s.repository.GetListByVehicle(ctx, id, organizationID)
	if err != nil {
		return nil, err
	}
	message := fmt.Sprintf("Got telemetry for vehicle %d", id)
	s.logger.Info(message)
	return res, nil
}

// DeleteTelemetryByID используется в DELETE /telemetry/{id}
// Удаляет запись по её ID, либо возвращает ошибку
func (s *TelemetryService) DeleteTelemetryByID(ctx context.Context, id int, organizationID *int) (model.Telemetry, error) {
	res, err := s.repository.DeleteItemByID(ctx, id, organizationID)
	if err != nil {
		return model.Telemetry{}, err
	}
	message := fmt.Sprintf("Telemetry with id %d was deleted", id)
	s.logger.Info(message)
	return res, nil
}

// DeleteTelemetryByVehicle используется в DELETE /telemetry/vehicle/{id}
// Удаляет срез записей по машине по её ID, либо возвращает ошибку
func (s *TelemetryService) DeleteTelemetryByVehicle(ctx context.Context, id int, organizationID *int) ([]model.Telemetry, error) {
	res, err := s.repository.DeleteListByVehicle(ctx, id, organizationID)
	if err != nil {
		return nil, err
	}
	message := fmt.Sprintf("Telemetry for vehicle %d was deleted", id)
	s.logger.Info(message)
	return res, nil
}
