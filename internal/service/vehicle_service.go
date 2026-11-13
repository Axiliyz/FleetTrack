package service

import (
	"context"
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

// VehicleService нужен для обработки автомобилей
type VehicleService struct {
	eventSink
	repository  repository.VehicleRepository
	logger      logger.Logger
	txManager   transaction.TransactionManager
	repoFactory factory.RepositoryFactory
}

// NewVehicleService создаёт новый сервис с репозиторием и логгером
func NewVehicleService(r repository.VehicleRepository, l logger.Logger, tx transaction.TransactionManager, rf factory.RepositoryFactory) *VehicleService {
	return &VehicleService{
		repository:  r,
		logger:      l,
		txManager:   tx,
		repoFactory: rf,
	}
}

// CreateVehicle добавляет в БД новую машину
func (s *VehicleService) CreateVehicle(ctx context.Context, v model.Vehicle) (model.Vehicle, error) {
	if err := validator.ValidateVehicle(v); err != nil {
		return model.Vehicle{}, err
	}
	err := s.repository.Create(ctx, &v)
	if err != nil {
		return model.Vehicle{}, err
	}

	s.logger.Info(fmt.Sprintf("Vehicle %d created", v.ID))
	s.publish(v.OrganizationID, stream.EventVehicle, v)

	return v, nil
}

// GetVehicleList возвращает список машин
func (s *VehicleService) GetVehicleList(ctx context.Context, filter model.VehicleFilter) ([]model.Vehicle, error) {
	if filter.OrganizationID != nil && *filter.OrganizationID < 1 {
		return nil, model.ErrInvalidOrganizationID
	}
	if filter.VIN != nil && !validator.IsVINValid(*filter.VIN) {
		return nil, model.ErrInvalidVIN
	}
	if filter.NumberPlate != nil && !validator.IsNumberPlateValid(*filter.NumberPlate) {
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
	s.logger.Debug(fmt.Sprintf("got vehicle with id %d", id))
	return v, nil
}

// DeleteVehicleByID мягко удаляет машину и в той же транзакции убирает всё, что от неё осталось бы висеть:
// закрывает привязку трекера (иначе машина вечно «не выходит на связь») и её открытые алерты.
// Машину в рейсе удалить нельзя - model.ErrVehicleIsBusy
func (s *VehicleService) DeleteVehicleByID(ctx context.Context, id int, organizationID *int) (model.Vehicle, error) {
	var (
		v        model.Vehicle
		resolved []model.Alert
	)
	err := s.txManager.WithTx(ctx, func(tx database.DBTX) error {
		repos := s.repoFactory.New(tx)
		var err error
		if v, err = repos.Vehicle.Delete(ctx, id, organizationID); err != nil {
			return err
		}
		if err = repos.Assignment.EndAssignmentByVehicle(ctx, id); err != nil {
			return err
		}
		resolved, err = repos.Alert.ResolveByVehicle(ctx, id)
		return err
	})
	if err != nil {
		return model.Vehicle{}, err
	}
	s.logger.Info(fmt.Sprintf("Vehicle %d deleted, %d alerts resolved", v.ID, len(resolved)))
	// со статусом DELETED: карта по нему убирает машину
	s.publish(v.OrganizationID, stream.EventVehicle, v)
	for _, a := range resolved {
		s.publish(a.OrganizationID, stream.EventAlert, a)
	}
	return v, nil
}

// validateVehicleUpdate проверяет то, что можно поменять в машине вручную.
// Вручную ставятся только IDLE и IN_SERVICE: ON_TRIP ставит и снимает рейс,
// DELETED - только DELETE /vehicles/{id}
func validateVehicleUpdate(upd model.UpdateVehicle) error {
	if upd.NumberPlate != nil && !validator.IsNumberPlateValid(*upd.NumberPlate) {
		return model.ErrInvalidNumberPlate
	}
	if upd.Status == nil {
		return nil
	}
	switch *upd.Status {
	case model.VehicleStatusIdle, model.VehicleStatusInService:
		return nil
	default:
		return model.ErrInvalidStatus
	}
}

// UpdateVehicleByID обновляет некоторые данные авто по ID
// Можно поменять: number_plate, status (IDLE или IN_SERVICE, если машина не в рейсе)
func (s *VehicleService) UpdateVehicleByID(ctx context.Context, id int, upd model.UpdateVehicle, organizationID *int) (model.Vehicle, error) {
	if err := validateVehicleUpdate(upd); err != nil {
		return model.Vehicle{}, err
	}
	v, err := s.repository.Update(ctx, id, upd, organizationID)
	if err != nil {
		return model.Vehicle{}, err
	}
	s.logger.Info(fmt.Sprintf("Vehicle %d was updated", v.ID))
	s.publish(v.OrganizationID, stream.EventVehicle, v)
	return v, nil
}
