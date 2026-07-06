// Package service содержит бизнес-логику приложения
package service

import (
	"context"
	"fleettrack/internal/logger"
	"fleettrack/internal/model"
	"fleettrack/internal/validator"
	"fmt"
)

// VehicleRepository определяет контракт хранения автомобилей в системе
type VehicleRepository interface {
	// Create создаёт новый автомобиль
	// Возвращает ошибку если не удалось
	Create(ctx context.Context, v *model.Vehicle) error
	// GetByID возвращает данные по автомобилю по его ID
	// Или ошибку, если не нашлось
	GetByID(ctx context.Context, id int) (model.Vehicle, error)
	// GetList возвращает срез автомобилей
	// Или ошибку
	GetList(ctx context.Context, filter model.VehicleFilter) ([]model.Vehicle, error)
	// Delete удаляет машину по ID
	// Возвращает удалённую машину, и ошибку, если не удалось
	Delete(ctx context.Context, id int) (model.Vehicle, error)
	// Update обновляет некоторые поля по авто
	Update(ctx context.Context, id int, upd model.UpdateVehicle) (model.Vehicle, error)
}

// VehicleService нужен для обработки автомобилей
type VehicleService struct {
	repository VehicleRepository
	logger     logger.Logger
}

// NewVehicleSesrvice создаёт новый сервис с репозиторием и логгером
func NewVehicleService(r VehicleRepository, l logger.Logger) *VehicleService {
	return &VehicleService{
		repository: r,
		logger:     l,
	}
}

// ProcessVehicle добавляет в БД новую машину
func (s *VehicleService) ProcessVehicle(ctx context.Context, v model.Vehicle) (model.Vehicle, error) {
	if err := validator.ValidateVehicle(v); err != nil {
		return model.Vehicle{}, err
	}
	err := s.repository.Create(ctx, &v)
	if err != nil {
		return model.Vehicle{}, err
	}

	s.logger.Info(fmt.Sprintf("Vehicle %d created", v.ID))

	return v, nil
}

// GetVehicleList возвращает список машин
func (s *VehicleService) GetVehicleList(ctx context.Context, filter model.VehicleFilter) ([]model.Vehicle, error) {
	if filter.OrganizationID != nil && *filter.OrganizationID < 1 {
		return nil, model.ErrInvalidOrganizationID
	}
	if filter.VIN != nil && len(*filter.VIN) != 17 {
		return nil, model.ErrInvalidVIN
	}
	if filter.NumberPlate != nil && (len(*filter.NumberPlate) < 8 ||
		len(*filter.NumberPlate) > 9) {
		return nil, model.ErrInvalidNumberPlate
	}
	if filter.Status != nil && !model.IsStatusValid(*filter.Status) {
		return nil, model.ErrInvalidStatus
	}
	if filter.CreatedFrom != nil && filter.CreatedTo != nil &&
		filter.CreatedTo.Before(*filter.CreatedFrom) {
		return nil, model.ErrInvalidTimestamp
	}

	vehicles, err := s.repository.GetList(ctx, filter)
	if err != nil {
		return nil, err
	}
	return vehicles, nil
}

// GetVehicleByID возвращает машину по её ID
func (s *VehicleService) GetVehicleByID(ctx context.Context, id int) (model.Vehicle, error) {
	v, err := s.repository.GetByID(ctx, id)
	if err != nil {
		return model.Vehicle{}, err
	}
	message := fmt.Sprintf("Got vehicle with id %d", id)
	s.logger.Info(message)
	return v, nil
}

// DeleteVehicleByID удаляет машину по её ID
func (s *VehicleService) DeleteVehicleByID(ctx context.Context, id int) (model.Vehicle, error) {
	v, err := s.repository.Delete(ctx, id)
	if err != nil {
		return model.Vehicle{}, err
	}
	s.logger.Info(fmt.Sprintf("Vehicle %d deleted", v.ID))
	return v, nil
}

// UpdateVehicleByID обновляет некоторые данные авто по ID
// Можно поменять: organization_id, number_plate, status
func (s *VehicleService) UpdateVehicleByID(ctx context.Context, id int, upd model.UpdateVehicle) (model.Vehicle, error) {
	v, err := s.repository.Update(ctx, id, upd)
	if err != nil {
		return model.Vehicle{}, err
	}
	s.logger.Info(fmt.Sprintf("Vehicle %d was updated", v.ID))
	return v, nil
}
