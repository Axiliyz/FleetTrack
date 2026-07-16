package service

import (
	"context"
	"fleettrack/internal/logger"
)

type AssignmentService struct {
	assignmentRepo    AssignmentRepository
	deviceRepository  DeviceRepository
	vehicleRepository VehicleRepository
	logger            logger.Logger
}

func NewAssignmentService(a AssignmentRepository, d DeviceRepository, v VehicleRepository, l logger.Logger) *AssignmentService {
	return &AssignmentService{
		assignmentRepo:    a,
		deviceRepository:  d,
		vehicleRepository: v,
		logger:            l,
	}
}

func (s *AssignmentService) AssignDevice(ctx context.Context, deviceID int, vehicleID int) error {

}
