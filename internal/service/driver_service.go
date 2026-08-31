package service

import (
	"context"
	"fleettrack/internal/logger"
	"fleettrack/internal/model"
	"fleettrack/internal/repository"
	"fleettrack/internal/validator"
	"fmt"
	"strings"
)

// DriverService нужен для обработки водителей
type DriverService struct {
	repository repository.DriverRepository
	logger     logger.Logger
}

// NewDriverService создаёт новый сервис с репозиторием и логгером
func NewDriverService(r repository.DriverRepository, l logger.Logger) *DriverService {
	return &DriverService{
		repository: r,
		logger:     l,
	}
}

// CreateDriver валидирует и сохраняет нового водителя
func (s *DriverService) CreateDriver(ctx context.Context, d model.Driver) (model.Driver, error) {
	if err := validator.ValidateDriver(d); err != nil {
		return model.Driver{}, err
	}

	if err := s.repository.Create(ctx, &d); err != nil {
		return model.Driver{}, err
	}

	s.logger.Info(fmt.Sprintf("Driver %d created", d.ID))
	return d, nil
}

// GetDriverByID возвращает водителя по его ID
func (s *DriverService) GetDriverByID(ctx context.Context, id int) (model.Driver, error) {
	d, err := s.repository.GetByID(ctx, id)
	if err != nil {
		return model.Driver{}, err
	}
	s.logger.Info(fmt.Sprintf("Got driver with id %d", id))
	return d, nil
}

// GetDriverList возвращает список водителей по фильтру
func (s *DriverService) GetDriverList(ctx context.Context, filter model.DriverFilter) ([]model.Driver, error) {
	if filter.OrganizationID != nil && *filter.OrganizationID < 1 {
		return nil, model.ErrInvalidOrganizationID
	}
	if filter.CreatedFrom != nil && filter.CreatedTo != nil &&
		filter.CreatedTo.Before(*filter.CreatedFrom) {
		return nil, model.ErrInvalidTimestamp
	}

	drivers, err := s.repository.GetList(ctx, filter)
	if err != nil {
		return nil, err
	}
	return drivers, nil
}

// DeleteDriverByID удаляет водителя по его ID
func (s *DriverService) DeleteDriverByID(ctx context.Context, id int) (model.Driver, error) {
	d, err := s.repository.Delete(ctx, id)
	if err != nil {
		return model.Driver{}, err
	}
	s.logger.Info(fmt.Sprintf("Driver %d deleted", d.ID))
	return d, nil
}

// UpdateDriverByID обновляет некоторые данные водителя по ID
// Можно поменять: organization_id, name
func (s *DriverService) UpdateDriverByID(ctx context.Context, id int, upd model.UpdateDriver) (model.Driver, error) {
	if upd.OrganizationID != nil && *upd.OrganizationID < 1 {
		return model.Driver{}, model.ErrInvalidOrganizationID
	}
	if upd.Name != nil && (strings.TrimSpace(*upd.Name) == "" || len(*upd.Name) > 35) {
		return model.Driver{}, model.ErrInvalidDriverName
	}

	d, err := s.repository.Update(ctx, id, upd)
	if err != nil {
		return model.Driver{}, err
	}
	s.logger.Info(fmt.Sprintf("Driver %d was updated", d.ID))
	return d, nil
}
