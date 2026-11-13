package service

import (
	"context"
	"errors"
	"fleettrack/internal/database"
	"fleettrack/internal/logger"
	"fleettrack/internal/model"
	"fleettrack/internal/repository"
	"fleettrack/internal/repository/factory"
	"fleettrack/internal/stream"
	"fleettrack/internal/transaction"
	"fleettrack/internal/validator"
	"fmt"
)

// TripService работает с рейсами и ведёт статус машины: пока рейс идёт, машина в ON_TRIP
type TripService struct {
	eventSink
	repository  repository.TripRepository
	logger      logger.Logger
	txManager   transaction.TransactionManager
	repoFactory factory.RepositoryFactory
}

// NewTripService - конструктор для сервиса рейсов
func NewTripService(r repository.TripRepository, l logger.Logger, tx transaction.TransactionManager, rf factory.RepositoryFactory) *TripService {
	return &TripService{
		repository:  r,
		logger:      l,
		txManager:   tx,
		repoFactory: rf,
	}
}

// AssignTrip создаёт рейс водителя на машине и переводит машину IDLE -> ON_TRIP в той же транзакции.
// Машина, которая не простаивает (уже в рейсе или на обслуживании), - model.ErrVehicleIsBusy:
// условие на статус стоит в UPDATE, поэтому два рейса одну машину не займут
func (s *TripService) AssignTrip(ctx context.Context, driverID, vehicleID int, organizationID *int) (model.Trip, error) {
	if driverID <= 0 {
		return model.Trip{}, model.ErrInvalidDriverID
	}
	if vehicleID <= 0 {
		return model.Trip{}, model.ErrInvalidVehicleID
	}

	t := model.Trip{
		DriverID:  driverID,
		VehicleID: vehicleID,
		Status:    model.TripStatusRunning,
	}
	if err := validator.ValidateTrip(t.Status); err != nil {
		return model.Trip{}, err
	}
	var vehicle model.Vehicle
	err := s.txManager.WithTx(ctx, func(tx database.DBTX) error {
		repos := s.repoFactory.New(tx)
		var err error
		vehicle, err = repos.Vehicle.UpdateStatus(ctx, vehicleID, model.VehicleStatusIdle, model.VehicleStatusOnTrip, organizationID)
		if errors.Is(err, model.ErrNotFound) {
			return explainBusyVehicle(ctx, repos, vehicleID, organizationID)
		}
		if err != nil {
			return err
		}
		return repos.Trip.CreateTrip(ctx, &t, organizationID)
	})
	if err != nil {
		return model.Trip{}, err
	}
	s.logger.Info(fmt.Sprintf("Created trip: driver %d, vehicle %d", t.DriverID, t.VehicleID))
	s.publishTrip(organizationID, t)
	s.publish(vehicle.OrganizationID, stream.EventVehicle, vehicle)
	return t, nil
}

// explainBusyVehicle выбирает ошибку, когда машину не удалось перевести в ON_TRIP:
// своя машина есть, но не простаивает - model.ErrVehicleIsBusy, иначе model.ErrNotFound
func explainBusyVehicle(ctx context.Context, repos factory.Repositories, vehicleID int, organizationID *int) error {
	v, err := repos.Vehicle.GetByID(ctx, vehicleID)
	if err != nil {
		return err
	}
	if organizationID != nil && v.OrganizationID != *organizationID {
		return model.ErrNotFound
	}
	return model.ErrVehicleIsBusy
}

// UpdateTrip обновляет статус рейса по ID. Завершающий статус (SUCCEEDED, CANCELLED)
// в той же транзакции возвращает машину из ON_TRIP в IDLE
func (s *TripService) UpdateTrip(ctx context.Context, id int, upd model.Trip, organizationID *int) (model.Trip, error) {
	if id <= 0 {
		return model.Trip{}, model.ErrInvalidTripID
	}
	if err := validator.ValidateTrip(upd.Status); err != nil {
		return model.Trip{}, err
	}
	return s.changeTripStatus(ctx, id, upd.Status, organizationID)
}

// changeTripStatus меняет статус рейса и, если рейс закончился, освобождает машину
func (s *TripService) changeTripStatus(ctx context.Context, id int, status model.TripStatus, organizationID *int) (model.Trip, error) {
	var (
		t        model.Trip
		vehicle  model.Vehicle
		released bool
	)
	err := s.txManager.WithTx(ctx, func(tx database.DBTX) error {
		repos := s.repoFactory.New(tx)
		var err error
		if t, err = repos.Trip.UpdateTrip(ctx, model.Trip{ID: id, Status: status}, organizationID); err != nil {
			return err
		}
		if !isTripFinished(t.Status) {
			return nil
		}
		vehicle, err = repos.Vehicle.UpdateStatus(ctx, t.VehicleID, model.VehicleStatusOnTrip, model.VehicleStatusIdle, organizationID)
		if errors.Is(err, model.ErrNotFound) {
			// машину успели удалить или поставить на обслуживание - освобождать нечего
			return nil
		}
		released = err == nil
		return err
	})
	if err != nil {
		return model.Trip{}, err
	}
	s.logger.Info(fmt.Sprintf("Updated trip %d: status %s", t.ID, t.Status))
	s.publishTrip(organizationID, t)
	if released {
		s.publish(vehicle.OrganizationID, stream.EventVehicle, vehicle)
	}
	return t, nil
}

// isTripFinished - статус завершает рейс, и машина больше не занята
func isTripFinished(status model.TripStatus) bool {
	return status == model.TripStatusSucceeded || status == model.TripStatusCancelled
}

// GetListTrips возвращает список рейсов с фильтрацией
func (s *TripService) GetListTrips(ctx context.Context, filter model.TripFilter) ([]model.Trip, error) {
	if filter.StartedFrom != nil && filter.StartedTo != nil && filter.StartedFrom.After(*filter.StartedTo) {
		return nil, model.ErrInvalidTimestamp
	}
	if filter.DriverID != nil && *filter.DriverID <= 0 {
		return nil, model.ErrInvalidDriverID
	}
	if filter.VehicleID != nil && *filter.VehicleID <= 0 {
		return nil, model.ErrInvalidVehicleID
	}
	if filter.MinDistance != nil && *filter.MinDistance <= 0 {
		return nil, model.ErrInvalidDistance
	}
	if filter.MaxDistance != nil && *filter.MaxDistance <= 0 {
		return nil, model.ErrInvalidDistance
	}
	if filter.MaxDistance != nil && filter.MinDistance != nil && *filter.MaxDistance < *filter.MinDistance {
		return nil, model.ErrInvalidDistance
	}
	if filter.MinAvgSpeed != nil && *filter.MinAvgSpeed <= 0 {
		return nil, model.ErrInvalidSpeed
	}
	if filter.MaxAvgSpeed != nil && *filter.MaxAvgSpeed <= 0 {
		return nil, model.ErrInvalidSpeed
	}
	if filter.MinAvgSpeed != nil && filter.MaxAvgSpeed != nil && *filter.MinAvgSpeed > *filter.MaxAvgSpeed {
		return nil, model.ErrInvalidSpeed
	}
	if filter.MinMaxSpeed != nil && *filter.MinMaxSpeed <= 0 {
		return nil, model.ErrInvalidSpeed
	}
	if filter.MaxMaxSpeed != nil && *filter.MaxMaxSpeed <= 0 {
		return nil, model.ErrInvalidSpeed
	}
	if filter.MinMaxSpeed != nil && filter.MaxMaxSpeed != nil && *filter.MinMaxSpeed > *filter.MaxMaxSpeed {
		return nil, model.ErrInvalidSpeed
	}

	trips, err := s.repository.GetListTrips(ctx, &filter)
	if err != nil {
		return nil, err
	}
	s.logger.Debug("got trips list")
	return trips, nil
}

// DeleteTrip отменяет рейс по ID (ставит статус CANCELLED)
func (s *TripService) DeleteTrip(ctx context.Context, id int, organizationID *int) (model.Trip, error) {
	if id <= 0 {
		return model.Trip{}, model.ErrInvalidTripID
	}

	return s.changeTripStatus(ctx, id, model.TripStatusCancelled, organizationID)
}

// GetTripByID возвращает рейс по его ID
func (s *TripService) GetTripByID(ctx context.Context, id int, organizationID *int) (model.Trip, error) {
	if id <= 0 {
		return model.Trip{}, model.ErrInvalidTripID
	}
	res, err := s.repository.GetByID(ctx, id, organizationID)
	if err != nil {
		return model.Trip{}, err
	}
	s.logger.Debug(fmt.Sprintf("got trip with id %d", id))
	return res, nil
}

// publishTrip отправляет рейс на карту. У рейса нет своей организации, её знает вызывающий:
// хендлер всегда передаёт организацию из токена
func (s *TripService) publishTrip(organizationID *int, t model.Trip) {
	if organizationID != nil {
		s.publish(*organizationID, stream.EventTrip, t)
	}
}
