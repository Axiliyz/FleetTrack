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
	logger               logger.Logger
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
		logger:               l,
	}
}

func (s *AssignmentService) GetActiveAssignment(ctx context.Context, id int) model.DeviceAssignment {
	assign, err := s.assignmentRepository.GetActiveAssignment(ctx, id)
	if err != nil {
		return model.DeviceAssignment{}
	}
	return assign
}

func (s *AssignmentService) CreateAssignment(ctx context.Context) {

}

func (s *AssignmentService) AssignDevice(ctx context.Context, deviceID int, vehicleID int) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		s.logger.Error(err.Error())
		return err
	}
	defer tx.Rollback(ctx)

	deviceRepo := postgres.NewPostgresDeviceRepository(tx)
	assignmentRepo := postgres.NewPostgresAssignmentRepository(tx)
	vehicleRepo := postgres.NewPostgresVehicleRepository(tx)
	// get device
	device, err := deviceRepo.GetByID(ctx, deviceID)

	// validate device
	if err != nil {
		s.logger.Error(err.Error())
		return err
	}
	if device.Status != model.DeviceStatusActive {
		s.logger.Error(model.ErrDeviceIsBusy.Error())
		return model.ErrDeviceIsBusy
	}
	// get vehicle
	vehicle, err := vehicleRepo.GetByID(ctx, vehicleID)
	// validate vehicle
	if err != nil {
		s.logger.Error(err.Error())
		return err
	}
	if vehicle.Status != model.VehicleStatusIdle {
		s.logger.Error(model.ErrVehicleIsBusy.Error())
		return model.ErrVehicleIsBusy
	}
	// check assignment
	_, err = assignmentRepo.GetActiveAssignment(ctx, deviceID)

	if err != nil && !errors.Is(err, model.ErrNotFound) {
		s.logger.Error(err.Error())
		return err
	}
	// create assignment
	newAssignment := model.DeviceAssignment{
		DeviceID:  deviceID,
		VehicleID: vehicleID,
	}
	switch {
	case errors.Is(err, model.ErrNotFound):
		if err := assignmentRepo.CreateAssignment(ctx, &newAssignment); err != nil {
			return err
		}
	case err != nil:
		s.logger.Error(err.Error())
		return err
	default:
		err := assignmentRepo.EndAssignment(ctx, deviceID)
		if err != nil {
			return err
		}
		if err := assignmentRepo.CreateAssignment(ctx, &newAssignment); err != nil {
			return err
		}
		if err != nil {
			return err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	return nil
}
