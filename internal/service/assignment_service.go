package service

import (
	"context"
	"errors"
	"fleettrack/internal/logger"
	"fleettrack/internal/model"
	"fleettrack/internal/repository/postgres"

	"github.com/jackc/pgx/v5/pgxpool"
)

type AssignmentService struct {
	assignmentRepository AssignmentRepository
	deviceRepository     DeviceRepository
	vehicleRepository    VehicleRepository
	pool                 *pgxpool.Pool
}

type DeviceRepository interface {
	GetByID(ctx context.Context, deviceID int) (model.Device, error)
}

type AssignmentRepository interface {
	GetActiveAssignment(ctx context.Context, deviceID int) (model.DeviceAssignment, error)
	CreateAssignment(ctx context.Context, assignment *model.DeviceAssignment) error
	EndAssignment(ctx context.Context, deviceID int) error
}

func NewAssignmentService(a AssignmentRepository, d DeviceRepository, v VehicleRepository, p *pgxpool.Pool, l logger.Logger) *AssignmentService {
	return &AssignmentService{
		assignmentRepository: a,
		deviceRepository:     d,
		vehicleRepository:    v,
		pool:                 p,
	}
}

func (s *AssignmentService) GetActiveAssignment(ctx context.Context, id int) model.DeviceAssignment {
	assign, err := s.assignmentRepository.GetActiveAssignment(ctx, id)
	if err != nil {
		return model.DeviceAssignment{}
	}
	return assign
}

// ValidateDevice проверяет, что устройство существует и свободно.
func (s *AssignmentService) ValidateDevice(ctx context.Context, deviceID int, deviceRepo DeviceRepository) error {
	device, err := deviceRepo.GetByID(ctx, deviceID)
	if err != nil {
		return err
	}
	if device.Status != model.DeviceStatusActive {
		return model.ErrDeviceIsBusy
	}
	return nil
}

// ValidateVehicle проверяет, что автомобиль существует и свободен.
func (s *AssignmentService) ValidateVehicle(ctx context.Context, vehicleID int, vehicleRepo VehicleRepository) error {
	vehicle, err := vehicleRepo.GetByID(ctx, vehicleID)
	if err != nil {
		return err
	}
	if vehicle.Status != model.VehicleStatusIdle {
		return model.ErrVehicleIsBusy
	}
	return nil
}

// EndPreviousAssignment завершает активную связь устройства, если она есть.
// Отсутствие активной связи ошибкой не считается.
func (s *AssignmentService) EndPreviousAssignment(ctx context.Context, deviceID int, assignmentRepo AssignmentRepository) error {
	_, err := assignmentRepo.GetActiveAssignment(ctx, deviceID)
	if err == nil {
		return assignmentRepo.EndAssignment(ctx, deviceID)
	}
	if !errors.Is(err, model.ErrNotFound) {
		return err
	}
	return nil
}

// CreateAssignment создаёт запись о связи устройства и автомобиля в БД.
func (s *AssignmentService) CreateAssignment(ctx context.Context, deviceID int, vehicleID int, assignmentRepo AssignmentRepository) error {
	newAssignment := model.DeviceAssignment{
		DeviceID:  deviceID,
		VehicleID: vehicleID,
	}
	return assignmentRepo.CreateAssignment(ctx, &newAssignment)
}

// AssignDevice соединяет весь пайплайн проверки и создания связи
func (s *AssignmentService) AssignDevice(ctx context.Context, deviceID int, vehicleID int) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Временное решение: создание tx-bound репозиториев собрано в одном месте,
	// позже этим займётся TransactionManager.
	assignmentRepo := postgres.NewPostgresAssignmentRepository(tx)
	deviceRepo := postgres.NewPostgresDeviceRepository(tx)
	vehicleRepo := postgres.NewPostgresVehicleRepository(tx)

	if err := s.ValidateDevice(ctx, deviceID, deviceRepo); err != nil {
		return err
	}
	if err := s.ValidateVehicle(ctx, vehicleID, vehicleRepo); err != nil {
		return err
	}
	if err := s.EndPreviousAssignment(ctx, deviceID, assignmentRepo); err != nil {
		return err
	}
	if err := s.CreateAssignment(ctx, deviceID, vehicleID, assignmentRepo); err != nil {
		return err
	}

	return tx.Commit(ctx)
}
