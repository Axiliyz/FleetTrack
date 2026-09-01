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
	"time"
)

// TelemetryService обрабатывает и валидирует телеметрию
type TelemetryService struct {
	repository    repository.TelemetryRepository
	logger        logger.Logger
	txManager     transaction.TransactionManager
	repoFactory   factory.RepositoryFactory
	motionService MotionService
}

// NewTelemetryService создаёт новый сервис с заданным репозиторием и логгером
func NewTelemetryService(r repository.TelemetryRepository, logger logger.Logger, tx transaction.TransactionManager, rf factory.RepositoryFactory, ms MotionService) *TelemetryService {
	return &TelemetryService{
		repository:    r,
		logger:        logger,
		txManager:     tx,
		repoFactory:   rf,
		motionService: ms,
	}
}

// validateTelemetry проверяет входные данные телеметрии.
//
// Проверяет:
// - DeviceID >= 0
// - VehicleID >= 0
// - Lat в диапазоне [-90, 90]
// - Lon в диапазоне [-180, 180]
// - Fuel в диапазоне [0, 1]
func validateTelemetry(t model.Telemetry) error {
	if t.DeviceID < 0 {
		return model.ErrInvalidDeviceID
	}
	if t.VehicleID < 0 {
		return model.ErrInvalidVehicleID
	}
	if t.Lat < -90 || t.Lat > 90 || t.Lon < -180 || t.Lon > 180 {
		return model.ErrInvalidCoords
	}
	if t.Fuel < 0.0 || t.Fuel > 1.0 {
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

// ProcessTelemetry валидирует телеметрию и сохраняет в репозиторий.
// Возвращает сохраненную телеметрию или ошибку валидации.
//
// Если DeviceTimestamp не указан - устанавливает текущее время.
// ReceivedAt всегда ставится в текущее время
func (s *TelemetryService) ProcessTelemetry(ctx context.Context, t model.Telemetry) (model.Telemetry, error) {
	if err := validateTelemetry(t); err != nil {
		return model.Telemetry{}, err
	}

	// Если пришло без времени отправления = Now
	if t.DeviceTimestamp.IsZero() {
		t.DeviceTimestamp = time.Now()
	}
	t.ReceivedAt = time.Now()

	err := s.txManager.WithTx(ctx, func(tx database.DBTX) error {
		repos := s.repoFactory.New(tx)

		trip, err := resolveActiveTrip(ctx, repos, t.VehicleID)
		if err != nil {
			return err
		}
		t.TripID = trip.ID

		last, err := resolveLastTelemetry(ctx, repos, t.VehicleID)
		if err != nil {
			return err
		}

		if err := s.applyMotion(ctx, repos, last, trip, &t); err != nil {
			return err
		}

		orgID, err := resolveVehicleOrg(ctx, repos, t.VehicleID)
		if err != nil {
			return err
		}
		t.OrganizationID = orgID

		return repos.Telemetry.Save(ctx, &t)
	})
	if err != nil {
		return model.Telemetry{}, err
	}

	message := fmt.Sprintf(
		"data stored: ID: %d Device: %d Vehicle: %d Lat: %f Lon: %f Fuel: %f",
		t.TelemetryID,
		t.DeviceID,
		t.VehicleID,
		t.Lat,
		t.Lon,
		t.Fuel,
	)
	s.logger.Info(message)
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
func (s *TelemetryService) GetTelemetryByVehicle(ctx context.Context, id int) ([]model.Telemetry, error) {
	res, err := s.repository.GetListByVehicle(ctx, id)
	if err != nil {
		return nil, err
	}
	message := fmt.Sprintf("Got telemetry for vehicle %d", id)
	s.logger.Info(message)
	return res, nil
}

// DeleteTelemetryByID используется в DELETE /telemetry/{id}
// Удаляет запись по её ID, либо возвращает ошибку
func (s *TelemetryService) DeleteTelemetryByID(ctx context.Context, id int) (model.Telemetry, error) {
	res, err := s.repository.DeleteItemByID(ctx, id)
	if err != nil {
		return model.Telemetry{}, err
	}
	message := fmt.Sprintf("Telemetry with id %d was deleted", id)
	s.logger.Info(message)
	return res, nil
}

// DeleteTelemetryByVehicle используется в DELETE /telemetry/vehicle/{id}
// Удаляет срез записей по машине по её ID, либо возвращает ошибку
func (s *TelemetryService) DeleteTelemetryByVehicle(ctx context.Context, id int) ([]model.Telemetry, error) {
	res, err := s.repository.DeleteListByVehicle(ctx, id)
	if err != nil {
		return nil, err
	}
	message := fmt.Sprintf("Telemetry for vehicle %d was deleted", id)
	s.logger.Info(message)
	return res, nil
}
