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
	// _, ok := ctx.Value(middleware.RequestIDKey).(string)

	// if !ok {
	// 	requestID = "unknown"
	// }

	if t.DeviceID < 0 || t.VehicleID < 0 {
		return model.Telemetry{}, model.ErrInvalidID
	}

	if t.Lat < -90 || t.Lat > 90 || t.Lon < -180 || t.Lon > 180 {
		return model.Telemetry{}, model.ErrInvalidCoords
	}

	// Если пришло без времени отправления - ставим Now
	if t.DeviceTimestamp.IsZero() {
		t.DeviceTimestamp = time.Now()
	}

	t.ReceivedAt = time.Now()
	err := s.repository.Save(t)
	if err != nil {
		return model.Telemetry{}, err
	}

	return t, nil

	// return model.TelemetryResponse{
	// 	Status:      "accepted",
	// 	Message:     "Telemetry saved successfully",
	// 	RequestID:   requestID,
	// 	TelemetryID: t.TelemetryID,
	// 	VehicleID:   t.VehicleID,
	// 	DeviceID:    t.DeviceID,
	// 	ReceivedAt:  t.ReceivedAt,
	// }
}
