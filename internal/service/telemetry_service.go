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
	"fleettrack/internal/stream"
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

// AlertEnqueuer - очередь оценки алертов, в которую сервис телеметрии кладёт сохранённые точки.
// Интерфейс, а не *AlertService: телеметрии нужна только очередь, и тесты подставляют свою
type AlertEnqueuer interface {
	Enqueue(ctx context.Context, t model.Telemetry)
}

// savedTelemetry — результат записи одной точки: сама точка и рейс, если его статистика обновилась
type savedTelemetry struct {
	telemetry model.Telemetry
	trip      *model.Trip
}

// TelemetryService обрабатывает и валидирует телеметрию
type TelemetryService struct {
	eventSink
	repository    repository.TelemetryRepository
	logger        logger.Logger
	txManager     transaction.TransactionManager
	repoFactory   factory.RepositoryFactory
	motionService MotionService
	alertService  AlertEnqueuer
	batcher       *batcher[model.Telemetry, savedTelemetry]
	batcherDone   chan struct{}
	maxClockAhead time.Duration
}

// NewTelemetryService создаёт сервис телеметрии. Запись идёт пачками, поэтому
// перед использованием нужно вызвать StartBatcher.
// as может быть nil - тогда точки в очередь алертов не попадают.
func NewTelemetryService(r repository.TelemetryRepository, logger logger.Logger, tx transaction.TransactionManager, rf factory.RepositoryFactory, ms MotionService, as AlertEnqueuer, batchCfg TelemetryBatchConfig) *TelemetryService {
	s := &TelemetryService{
		repository:    r,
		logger:        logger,
		txManager:     tx,
		repoFactory:   rf,
		motionService: ms,
		alertService:  as,
		batcherDone:   make(chan struct{}),
		maxClockAhead: DefaultMaxClockAhead,
	}
	s.batcher = newBatcher[model.Telemetry, savedTelemetry](batchCfg.BufferSize, batchCfg.MaxSize, batchCfg.MaxWait, s.flushBatch)
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

// DefaultMaxClockAhead — насколько часы терминала могут убегать вперёд.
// Точка «из будущего» становится последней для GetLastByVehicle, и пока её время не наступит,
// у всех следующих точек время меньше: скорость и пробег нулевые, SPEED_EXCEEDED молчит.
// Поэтому допуск - минуты, на рассинхрон часов, а не дни
const DefaultMaxClockAhead = 2 * time.Minute

// SetMaxClockAhead меняет допуск часов терминала; d <= 0 оставляет DefaultMaxClockAhead.
// Вызывается один раз при сборке приложения
func (s *TelemetryService) SetMaxClockAhead(d time.Duration) {
	if d > 0 {
		s.maxClockAhead = d
	}
}

// validateTelemetry проверяет входные данные телеметрии.
//
// Проверяет:
// - DeviceID >= 1
// - VehicleID >= 1
// - Lat в диапазоне [-90, 90]
// - Lon в диапазоне [-180, 180]
// - Fuel в диапазоне [0, 100]
// - DeviceTimestamp не дальше maxClockAhead в будущем
func validateTelemetry(t model.Telemetry, maxClockAhead time.Duration) error {
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
	if !t.DeviceTimestamp.IsZero() && t.DeviceTimestamp.After(time.Now().Add(maxClockAhead)) {
		return model.ErrInvalidTimestamp
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

// applyMotion считает пройденное расстояние и скорость по предыдущей точке,
// прибавляет расстояние к рейсу и возвращает рейс с новой статистикой.
// Если предыдущей точки не было или отрезок оказался скачком GPS - статистику рейса
// не трогает и возвращает nil.
func (s *TelemetryService) applyMotion(ctx context.Context, repos factory.Repositories, last *model.Telemetry, trip model.Trip, t *model.Telemetry) (*model.Trip, error) {
	if last == nil {
		return nil, nil
	}
	motion, err := s.motionService.Calculate(last, *t)
	if err != nil {
		return nil, err
	}
	if motion.Outlier {
		s.logOutlier(*t)
		return nil, nil
	}
	t.DistanceKm = motion.DistanceKm
	t.SpeedKmh = motion.SpeedKmh
	updated, err := repos.Trip.UpdateTripStats(ctx, trip.ID, t.DistanceKm, t.SpeedKmh)
	if err != nil {
		return nil, err
	}
	return &updated, nil
}

// logOutlier пишет в лог скачок GPS: сама точка сохраняется, но в пробег и скорость не идёт
func (s *TelemetryService) logOutlier(t model.Telemetry) {
	metrics.RecordTelemetryOutlier()
	s.logger.Warn(fmt.Sprintf("gps outlier ignored: vehicle=%d device=%d speed above %.0f km/h",
		t.VehicleID, t.DeviceID, MaxPlausibleSpeedKmh))
}

// ProcessTelemetry валидирует телеметрию и сохраняет её в составе пачки.
// Возвращает сохранённую телеметрию или ошибку.
//
// Если DeviceTimestamp не указан - устанавливает текущее время.
// ReceivedAt всегда ставится в текущее время.
func (s *TelemetryService) ProcessTelemetry(ctx context.Context, t model.Telemetry) (model.Telemetry, error) {
	serviceStart := time.Now()
	defer func() { metrics.RecordTelemetryStage("service_total", time.Since(serviceStart).Seconds()) }()

	if err := validateTelemetry(t, s.maxClockAhead); err != nil {
		return model.Telemetry{}, err
	}

	if t.DeviceTimestamp.IsZero() {
		t.DeviceTimestamp = time.Now()
	}
	t.ReceivedAt = time.Now()

	result, err := s.batcher.Submit(ctx, t)
	if err != nil {
		return model.Telemetry{}, err
	}
	saved := result.telemetry

	// Submit возвращается после коммита пачки — теперь точку можно показывать на карте
	s.publish(saved.OrganizationID, stream.EventTelemetry, saved)
	if result.trip != nil {
		s.publish(saved.OrganizationID, stream.EventTrip, *result.trip)
	}

	if s.alertService != nil {
		enqueueStart := time.Now()
		s.alertService.Enqueue(ctx, saved)
		metrics.RecordTelemetryStage("alert_enqueue", time.Since(enqueueStart).Seconds())
	}

	s.logger.Debug(fmt.Sprintf("telemetry stored: id=%d device=%d vehicle=%d", saved.TelemetryID, saved.DeviceID, saved.VehicleID))
	return saved, nil
}

// flushBatch пишет пачку одной транзакцией. Каждая запись выполняется под своей
// точкой сохранения: её ошибка откатывает только её изменения и возвращается только её
// отправителю. Ошибка всей пачки возвращается, лишь если сломалась сама транзакция.
func (s *TelemetryService) flushBatch(ctx context.Context, batch []model.Telemetry) ([]itemResult[savedTelemetry], error) {
	results := make([]itemResult[savedTelemetry], len(batch))
	err := s.txManager.WithTx(ctx, func(tx database.DBTX) error {
		repos := s.repoFactory.New(tx)
		for i, t := range batch {
			var saved savedTelemetry
			itemErr, txErr := transaction.WithSavepoint(ctx, tx, func() error {
				var err error
				saved, err = s.saveInTx(ctx, repos, t)
				return err
			})
			if txErr != nil {
				return txErr
			}
			results[i] = itemResult[savedTelemetry]{value: saved, err: itemErr}
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
func (s *TelemetryService) saveInTx(ctx context.Context, repos factory.Repositories, t model.Telemetry) (savedTelemetry, error) {
	stageStart := time.Now()
	orgID, err := resolveVehicleOrg(ctx, repos, t.VehicleID)
	metrics.RecordTelemetryStage("vehicle_org", time.Since(stageStart).Seconds())
	if err != nil {
		return savedTelemetry{}, err
	}
	if t.OrganizationID != 0 && t.OrganizationID != orgID {
		return savedTelemetry{}, model.ErrNotFound
	}
	t.OrganizationID = orgID

	assignment, err := repos.Assignment.GetActiveAssignment(ctx, t.DeviceID)
	if err != nil && !errors.Is(err, model.ErrNotFound) {
		return savedTelemetry{}, err
	}
	if err != nil || assignment.VehicleID != t.VehicleID {
		return savedTelemetry{}, model.ErrDeviceNotAssigned
	}

	stageStart = time.Now()
	trip, err := resolveActiveTrip(ctx, repos, t.VehicleID)
	metrics.RecordTelemetryStage("active_trip", time.Since(stageStart).Seconds())
	if err != nil && !errors.Is(err, model.ErrNoActiveTrip) {
		return savedTelemetry{}, err
	}

	stageStart = time.Now()
	last, err := resolveLastTelemetry(ctx, repos, t.VehicleID)
	metrics.RecordTelemetryStage("last_telemetry", time.Since(stageStart).Seconds())
	if err != nil {
		return savedTelemetry{}, err
	}

	var updatedTrip *model.Trip
	if trip.ID != 0 {
		t.TripID = trip.ID
		updatedTrip, err = s.applyMotion(ctx, repos, last, trip, &t)
		if err != nil {
			s.logger.Warn(fmt.Sprintf("failed to apply motion: %s", err.Error()))
		}
	} else if last != nil && s.motionService != nil {
		if motion, err := s.motionService.Calculate(last, t); err == nil && motion != nil {
			if motion.Outlier {
				s.logOutlier(t)
			}
			t.DistanceKm = motion.DistanceKm
			t.SpeedKmh = motion.SpeedKmh
		}
	}

	stageStart = time.Now()
	err = repos.Telemetry.Save(ctx, &t)
	metrics.RecordTelemetryStage("save", time.Since(stageStart).Seconds())
	if err != nil {
		return savedTelemetry{}, err
	}

	stageStart = time.Now()
	err = repos.Vehicle.UpdateLastTelemetryAt(ctx, t.VehicleID, t.ReceivedAt)
	metrics.RecordTelemetryStage("update_vehicle", time.Since(stageStart).Seconds())
	if err != nil {
		return savedTelemetry{}, err
	}
	return savedTelemetry{telemetry: t, trip: updatedTrip}, nil
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
	s.logger.Debug("got filtered telemetry")
	return res, nil
}

// GetTelemetryByID используется в GET /telemetry/{id}.
// driverID != nil оставляет только точки рейсов этого водителя
// Возвращает запись по её ID, либо ошибку
func (s *TelemetryService) GetTelemetryByID(ctx context.Context, id int, driverID *int) (model.Telemetry, error) {
	res, err := s.repository.GetItemByID(ctx, id, driverID)
	if err != nil {
		return model.Telemetry{}, err
	}
	s.logger.Debug(fmt.Sprintf("got telemetry with id %d", id))
	return res, nil
}

// GetTelemetryByVehicle используется в GET /telemetry/vehicles/{id}.
// driverID != nil оставляет только точки рейсов этого водителя
// Возвращает срез записей по машине по её ID, либо ошибку
func (s *TelemetryService) GetTelemetryByVehicle(ctx context.Context, id int, organizationID, driverID *int) ([]model.Telemetry, error) {
	res, err := s.repository.GetListByVehicle(ctx, id, organizationID, driverID)
	if err != nil {
		return nil, err
	}
	s.logger.Debug(fmt.Sprintf("got telemetry for vehicle %d", id))
	return res, nil
}

// DeleteTelemetryByID используется в DELETE /telemetry/{id}
// Удаляет запись по её ID, либо возвращает ошибку
func (s *TelemetryService) DeleteTelemetryByID(ctx context.Context, id int, organizationID *int) (model.Telemetry, error) {
	res, err := s.repository.DeleteItemByID(ctx, id, organizationID)
	if err != nil {
		return model.Telemetry{}, err
	}
	s.logger.Info(fmt.Sprintf("telemetry with id %d was deleted", id))
	return res, nil
}

// DeleteTelemetryByVehicle используется в DELETE /telemetry/vehicles/{id}
// Удаляет все записи машины и возвращает их число, либо ошибку
func (s *TelemetryService) DeleteTelemetryByVehicle(ctx context.Context, id int, organizationID *int) (int64, error) {
	deleted, err := s.repository.DeleteListByVehicle(ctx, id, organizationID)
	if err != nil {
		return 0, err
	}
	s.logger.Info(fmt.Sprintf("telemetry for vehicle %d was deleted: %d rows", id, deleted))
	return deleted, nil
}
