package service

import (
	"context"
	"fleettrack/internal/logger"
	"fleettrack/internal/model"
	"fleettrack/internal/repository"
	"fleettrack/internal/validator"
	"fmt"
)

// TripService работает с рейсами
type TripService struct {
	repository repository.TripRepository
	logger     logger.Logger
}

// NewTripService - конструктор для сервиса рейсов
func NewTripService(r repository.TripRepository, l logger.Logger) *TripService {
	return &TripService{
		repository: r,
		logger:     l,
	}
}

// AssignTrip создаёт связку водитель-авто
func (s *TripService) AssignTrip(ctx context.Context, driverID, vehicleID int) (model.Trip, error) {
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
	err := s.repository.CreateTrip(ctx, &t)
	if err != nil {
		return model.Trip{}, err
	}
	s.logger.Info(fmt.Sprintf("Created trip: driver %d, vehicle %d", t.DriverID, t.VehicleID))
	return t, nil
}

// UpdateTrip обновляет статус рейса по ID
func (s *TripService) UpdateTrip(ctx context.Context, id int, upd model.Trip) (model.Trip, error) {
	if id <= 0 {
		return model.Trip{}, model.ErrInvalidTripID
	}
	if err := validator.ValidateTrip(upd.Status); err != nil {
		return model.Trip{}, err
	}

	t, err := s.repository.UpdateTrip(ctx, model.Trip{ID: id, Status: upd.Status})
	if err != nil {
		return model.Trip{}, err
	}
	s.logger.Info(fmt.Sprintf("Updated trip %d: status %s", t.ID, t.Status))
	return t, nil
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
	s.logger.Info("Got filtered trips list")
	return trips, nil
}

// DeleteTrip отменяет рейс по ID (ставит статус CANCELLED)
func (s *TripService) DeleteTrip(ctx context.Context, id int) (model.Trip, error) {
	if id <= 0 {
		return model.Trip{}, model.ErrInvalidTripID
	}

	t, err := s.repository.DeleteTrip(ctx, id)
	if err != nil {
		return model.Trip{}, err
	}
	s.logger.Info(fmt.Sprintf("Cancelled trip %d", t.ID))
	return t, nil
}

// GetTripByID возвращает рейс по его ID
func (s *TripService) GetTripByID(ctx context.Context, id int) (model.Trip, error) {
	if id <= 0 {
		return model.Trip{}, model.ErrInvalidTripID
	}
	res, err := s.repository.GetByID(ctx, id)
	if err != nil {
		return model.Trip{}, err
	}
	message := fmt.Sprintf("Got trip with id %d", id)
	s.logger.Info(message)
	return res, nil
}
