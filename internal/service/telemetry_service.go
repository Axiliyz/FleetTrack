package service

import (
	"context"
	"fleettrack/internal/model"
	"fleettrack/internal/repository"
	"time"
)

type TelemetryService struct {
	repository repository.TelemetryRepository
}

func NewTelemetryService(r repository.TelemetryRepository) *TelemetryService {
	return &TelemetryService{repository: r}
}

func (s *TelemetryService) ProcessTelemetry(ctx context.Context, t model.Telemetry) (model.Telemetry, error) {
	if t.DeviceID < 0 {
		return model.Telemetry{}, model.ErrInvalidDeviceID
	}

	if t.VehicleID < 0 {
		return model.Telemetry{}, model.ErrInvalidVehicleID
	}

	if t.Lat < -90 || t.Lat > 90 || t.Lon < -180 || t.Lon > 180 {
		return model.Telemetry{}, model.ErrInvalidCoords
	}

	// Если пришло без времени отправления ставим Now
	if t.DeviceTimestamp.IsZero() {
		t.DeviceTimestamp = time.Now()
	}

	t.ReceivedAt = time.Now()
	err := s.repository.Save(t)
	if err != nil {
		return model.Telemetry{}, err
	}

	return t, nil
}
