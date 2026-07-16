package service

import (
	"context"
	"fleettrack/internal/model"
)

type DeviceRepository interface {
	GetByID(ctx context.Context, id int) (model.Device, error)
}
type AssignmentRepository interface {
	GetByID(ctx context.Context, id int) (model.DeviceAssignment, error)
}
