// Package service содержит бизнес-логику приложения
package service

import (
	"context"
	"fleettrack/internal/logger"
	"fleettrack/internal/model"
	"fmt"
	"time"
)

// TelemetryRepository определяет контракт сохранения телеметрии
type TelemetryRepository interface {
	// Save сохраняет телеметрию в хранилище
	// Возвращает ошибку если сохранение не удалось
	Save(ctx context.Context, t *model.Telemetry) error
	GetList(ctx context.Context, limit int) ([]model.Telemetry, error)
	GetItemByID(ctx context.Context, id int) (model.Telemetry, error)
	GetListByVehicle(ctx context.Context, id int) ([]model.Telemetry, error)
}

// TelemetryService обрабатывает и валидирует телеметрию
type TelemetryService struct {
	repository TelemetryRepository
	logger     logger.Logger
}

// NewTelemetryService создаёт новый сервис с заданным репозиторием и логгером
func NewTelemetryService(r TelemetryRepository, logger logger.Logger) *TelemetryService {
	return &TelemetryService{
		repository: r,
		logger:     logger,
	}
}

// ProcessTelemetry валидирует телеметрию и сохраняет в репозиторий.
// Возвращает сохраненную телеметрию или ошибку валидации.
//
// Проверяет:
// - DeviceID >= 0
// - VehicleID >= 0
// - Lat в диапазоне [-90, 90]
// - Lon в диапазоне [-180, 180]
// - Fuel в диапазоне [0, 1]
//
// Если DeviceTimestamp не указан - устанавливает текущее время.
// ReceivedAt всегда ставится в текущее время
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

	if t.Fuel < 0.0 || t.Fuel > 1.0 {
		return model.Telemetry{}, model.ErrInvalidFuel
	}

	// Если пришло без времени отправления ставим Now
	if t.DeviceTimestamp.IsZero() {
		t.DeviceTimestamp = time.Now()
	}

	t.ReceivedAt = time.Now()
	err := s.repository.Save(ctx, &t)
	if err != nil {
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

func (s *TelemetryService) GetTelemetryList(ctx context.Context, limit int) ([]model.Telemetry, error) {
	return s.repository.GetList(ctx, limit)
}

func (s *TelemetryService) GetTelemetryByID(ctx context.Context, id int) (model.Telemetry, error) {
	return s.repository.GetItemByID(ctx, id)
}

func (s *TelemetryService) GetTelemetryByVehicle(ctx context.Context, id int) ([]model.Telemetry, error) {
	return s.repository.GetListByVehicle(ctx, id)
}
