package repository

import (
	"fleettrack/internal/logger"
	"fleettrack/internal/model"
	"fmt"
)

type TelemetryRepository interface {
	Save(t model.Telemetry) error
}

type MemoryTelemetryRepository struct {
	history []model.Telemetry
	current map[int]model.Telemetry
	logger  logger.Logger
}

func (r *MemoryTelemetryRepository) Save(t model.Telemetry) error {
	r.history = append(r.history, t)
	r.current[t.VehicleID] = t

	message := fmt.Sprintf(
		"Telemetry stored:\nID: %d, Vehicle: %d, Device: %d, Time: %v",
		t.TelemetryID,
		t.VehicleID,
		t.DeviceID,
		t.ReceivedAt.Format("02.01.2006 15:04:05"),
	)
	r.logger.Info(message)
	return nil
}

func NewMemoryTelemetryRepository(logger logger.Logger) *MemoryTelemetryRepository {
	return &MemoryTelemetryRepository{
		history: make([]model.Telemetry, 0),
		current: make(map[int]model.Telemetry),
		logger:  logger,
	}
}
