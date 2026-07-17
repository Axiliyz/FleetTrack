package service

import (
	"context"
	"errors"
	"fleettrack/internal/database"
	"fleettrack/internal/model"
	"fleettrack/internal/repository"
	"fleettrack/internal/repository/factory"
	"fleettrack/internal/transaction"
)

// AssignmentService реализует бизнес-логику привязки устройств к автомобилям.
type AssignmentService struct {
	assignmentRepository repository.AssignmentRepository
	deviceRepository     repository.DeviceRepository
	vehicleRepository    repository.VehicleRepository
	txManager            transaction.TransactionManager
	repoFactory          factory.RepositoryFactory
}

// NewAssignmentService создаёт новый сервис связей устройств и автомобилей.
func NewAssignmentService(ar repository.AssignmentRepository, dr repository.DeviceRepository, vr repository.VehicleRepository, tm transaction.TransactionManager, rf factory.RepositoryFactory) *AssignmentService {
	return &AssignmentService{
		assignmentRepository: ar,
		deviceRepository:     dr,
		vehicleRepository:    vr,
		txManager:            tm,
		repoFactory:          rf,
	}
}

// GetActiveAssignment возвращает активную связь устройства с автомобилем.
// При ошибке возвращает нулевое значение model.DeviceAssignment.
func (s *AssignmentService) GetActiveAssignment(ctx context.Context, id int) model.DeviceAssignment {
	assign, err := s.assignmentRepository.GetActiveAssignment(ctx, id)
	if err != nil {
		return model.DeviceAssignment{}
	}
	return assign
}

// ValidateDevice проверяет, что устройство существует и свободно.
func (s *AssignmentService) ValidateDevice(ctx context.Context, device model.Device) error {

	if device.Status != model.DeviceStatusActive {
		return model.ErrDeviceIsBusy
	}
	return nil
}

// ValidateVehicle проверяет, что автомобиль существует и свободен.
func (s *AssignmentService) ValidateVehicle(ctx context.Context, vehicle model.Vehicle) error {
	if vehicle.Status != model.VehicleStatusIdle {
		return model.ErrVehicleIsBusy
	}
	return nil
}

// EndPreviousAssignment завершает активную связь устройства, если она есть.
// Отсутствие активной связи ошибкой не считается.
func (s *AssignmentService) EndPreviousAssignment(ctx context.Context, deviceID int, assignmentRepo repository.AssignmentRepository) error {
	_, err := assignmentRepo.GetActiveAssignment(ctx, deviceID)
	if err == nil {
		return assignmentRepo.EndAssignment(ctx, deviceID)
	}
	if !errors.Is(err, model.ErrNotFound) {
		return err
	}
	return nil
}

// AssignDevice соединяет весь пайплайн проверки и создания связи
func (s *AssignmentService) AssignDevice(ctx context.Context, deviceID int, vehicleID int) error {
	return s.txManager.WithTx(ctx, func(tx database.DBTX) error {
		repos := s.repoFactory.New(tx)

		device, err := repos.Device.GetByID(ctx, deviceID)
		if err != nil {
			return err
		}
		vehicle, err := repos.Vehicle.GetByID(ctx, vehicleID)
		if err != nil {
			return err
		}
		if err := s.ValidateDevice(ctx, device); err != nil {
			return err
		}
		if err := s.ValidateVehicle(ctx, vehicle); err != nil {
			return err
		}
		if err := s.EndPreviousAssignment(ctx, deviceID, repos.Assignment); err != nil {
			return err
		}
		return repos.Assignment.CreateAssignment(ctx, &model.DeviceAssignment{
			DeviceID:  deviceID,
			VehicleID: vehicleID,
		})
	})
}
