package service

import (
	"context"
	"fleettrack/internal/logger"
	"fleettrack/internal/model"
	"testing"
)

type mockRepository struct{}

func (m *mockRepository) Save(ctx context.Context, t model.Telemetry) error {
	return nil
}

func TestProcessTelemetry(t *testing.T) {
	tests := []struct {
		name      string
		telemetry model.Telemetry
		wantErr   error
	}{
		{
			name: "valid",
			telemetry: model.Telemetry{
				DeviceID:  1,
				VehicleID: 1,
				Lat:       55.75,
				Lon:       37.61,
				Fuel:      0.8,
			},
			wantErr: nil,
		},
		{
			name: "invalid device id",
			telemetry: model.Telemetry{
				DeviceID:  -1,
				VehicleID: 1,
				Lat:       55.75,
				Lon:       37.61,
				Fuel:      0.8,
			},
			wantErr: model.ErrInvalidDeviceID,
		},
		{
			name: "invalid coords",
			telemetry: model.Telemetry{
				DeviceID:  1,
				VehicleID: 1,
				Lat:       155.75,
				Lon:       37.61,
				Fuel:      0.8,
			},
			wantErr: model.ErrInvalidCoords,
		},
	}

	repo := &mockRepository{}
	logger := logger.NewStdLogger(logger.DebugLevel)
	service := NewTelemetryService(repo, logger)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := service.ProcessTelemetry(context.Background(), tt.telemetry)
			if err != tt.wantErr {
				t.Errorf("got %v, want %v", err, tt.wantErr)
			}
		})
	}
}
