package service

import (
	"context"
	"fleettrack/internal/logger"
	"fleettrack/internal/model"
	"fmt"
	"time"
)

type TelemetryRepository interface {
	Save(ctx context.Context, t model.Telemetry) error
}

type TelemetryService struct {
	repository TelemetryRepository
	logger     logger.Logger
}

func NewTelemetryService(r TelemetryRepository, logger logger.Logger) *TelemetryService {
	return &TelemetryService{
		repository: r,
		logger:     logger,
	}
}

func (s *TelemetryService) ProcessTelemetry(ctx context.Context, t model.Telemetry) (model.Telemetry, error) {
	if t.DeviceID < 0 {
		s.logger.Error(model.ErrInvalidDeviceID.Error())
		return model.Telemetry{}, model.ErrInvalidDeviceID
	}

	if t.VehicleID < 0 {
		s.logger.Error(model.ErrInvalidVehicleID.Error())
		return model.Telemetry{}, model.ErrInvalidVehicleID
	}

	if t.Lat < -90 || t.Lat > 90 || t.Lon < -180 || t.Lon > 180 {
		s.logger.Error(model.ErrInvalidCoords.Error())
		return model.Telemetry{}, model.ErrInvalidCoords
	}

	// Если пришло без времени отправления ставим Now
	if t.DeviceTimestamp.IsZero() {
		t.DeviceTimestamp = time.Now()
	}

	t.ReceivedAt = time.Now()
	err := s.repository.Save(ctx, t)
	if err != nil {
		s.logger.Error(err.Error())
		return model.Telemetry{}, err
	}

	message := fmt.Sprintf(
		"data stored: ID: %d Device: %d Vehicle: %d Lat: %f Lon: %f Fuel: %f",
		t.TelemetryID,
		t.DeviceID,
		t.VehicleID,
		t.Lat,
		t.Lon,
		t.Fuel,
	)
	s.logger.Info(message)
	return t, nil
}
