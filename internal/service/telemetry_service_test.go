package service

import (
	"context"
	"fleettrack/internal/logger"
	"fleettrack/internal/model"
	"testing"
)

type mockRepository struct{}

func (m *mockRepository) Save(ctx context.Context, t *model.Telemetry) error {
	return nil
}

func (m *mockRepository) GetList(ctx context.Context, limit int) ([]model.Telemetry, error) {
	return []model.Telemetry{}, nil
}

func (m *mockRepository) GetItemByID(ctx context.Context, id int) (model.Telemetry, error) {
	return model.Telemetry{}, nil
}

func (r *mockRepository) GetListByVehicle(ctx context.Context, id int) ([]model.Telemetry, error) {
	return []model.Telemetry{}, nil
}

func (m *mockRepository) DeleteItemByID(ctx context.Context, id int) (model.Telemetry, error) {
	return model.Telemetry{}, nil
}

func (r *mockRepository) DeleteListByVehicle(ctx context.Context, id int) ([]model.Telemetry, error) {
	return []model.Telemetry{}, nil
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
			name: "invalid vehicle id",
			telemetry: model.Telemetry{
				DeviceID:  1,
				VehicleID: -15,
				Lat:       55.75,
				Lon:       37.61,
				Fuel:      0.8,
			},
			wantErr: model.ErrInvalidVehicleID,
		},
		{
			name: "edge coords(lon=-180)",
			telemetry: model.Telemetry{
				DeviceID:  1,
				VehicleID: 1,
				Lat:       14.22,
				Lon:       -180,
				Fuel:      0.8,
			},
			wantErr: nil,
		},
		{
			name: "edge coords (lat=-90)",
			telemetry: model.Telemetry{
				DeviceID:  1,
				VehicleID: 1,
				Lat:       -90,
				Lon:       37.61,
				Fuel:      0.8,
			},
			wantErr: nil,
		},
		{
			name: "edge coords(lon=180)",
			telemetry: model.Telemetry{
				DeviceID:  1,
				VehicleID: 1,
				Lat:       14.22,
				Lon:       180,
				Fuel:      0.8,
			},
			wantErr: nil,
		},
		{
			name: "edge coords (lat=90)",
			telemetry: model.Telemetry{
				DeviceID:  1,
				VehicleID: 1,
				Lat:       90,
				Lon:       37.61,
				Fuel:      0.8,
			},
			wantErr: nil,
		},
		{
			name: "invalid coords (lat)",
			telemetry: model.Telemetry{
				DeviceID:  1,
				VehicleID: 1,
				Lat:       255.75,
				Lon:       37.61,
				Fuel:      0.8,
			},
			wantErr: model.ErrInvalidCoords,
		},
		{
			name: "invalid coords(lon)",
			telemetry: model.Telemetry{
				DeviceID:  1,
				VehicleID: 1,
				Lat:       75.75,
				Lon:       317.61,
				Fuel:      0.8,
			},
			wantErr: model.ErrInvalidCoords,
		},
		{
			name: "invalid fuel (> 1)",
			telemetry: model.Telemetry{
				DeviceID:  1,
				VehicleID: 1,
				Lat:       45.75,
				Lon:       17.61,
				Fuel:      1.2,
			},
			wantErr: model.ErrInvalidFuel,
		},
		{
			name: "invalid fuel (< 0)",
			telemetry: model.Telemetry{
				DeviceID:  1,
				VehicleID: 1,
				Lat:       45.75,
				Lon:       17.61,
				Fuel:      -0.14,
			},
			wantErr: model.ErrInvalidFuel,
		},
		{
			name: "edge fuel (= 0)",
			telemetry: model.Telemetry{
				DeviceID:  1,
				VehicleID: 1,
				Lat:       45.75,
				Lon:       17.61,
				Fuel:      0,
			},
			wantErr: nil,
		},
		{
			name: "edge fuel (= 1)",
			telemetry: model.Telemetry{
				DeviceID:  1,
				VehicleID: 1,
				Lat:       45.75,
				Lon:       17.61,
				Fuel:      1,
			},
			wantErr: nil,
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
