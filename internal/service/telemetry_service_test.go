package service

import (
	"context"
	"fleettrack/internal/logger"
	"fleettrack/internal/model"
	"testing"
	"time"
)

type mockRepository struct{}

func (m *mockRepository) Save(ctx context.Context, t *model.Telemetry) error {
	return nil
}

func (m *mockRepository) GetList(ctx context.Context, filter model.TelemetryFilter) ([]model.Telemetry, error) {
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

func intPtr(v int) *int             { return &v }
func float32Ptr(v float32) *float32 { return &v }
func float64Ptr(v float64) *float64 { return &v }
func timePtr(v time.Time) *time.Time { return &v }

func TestGetTelemetryList(t *testing.T) {
	from := timePtr(time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC))
	to := timePtr(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))

	tests := []struct {
		name    string
		filter  model.TelemetryFilter
		wantErr error
	}{
		{
			name:    "no filters",
			filter:  model.TelemetryFilter{Limit: 100},
			wantErr: nil,
		},
		{
			name:    "valid vehicle filter",
			filter:  model.TelemetryFilter{VehicleID: intPtr(1), Limit: 100},
			wantErr: nil,
		},
		{
			name:    "from after to",
			filter:  model.TelemetryFilter{From: from, To: to, Limit: 100},
			wantErr: model.ErrInvalidTimestamp,
		},
		{
			name:    "fuel min greater than fuel max",
			filter:  model.TelemetryFilter{FuelMin: float32Ptr(0.9), FuelMax: float32Ptr(0.1), Limit: 100},
			wantErr: model.ErrInvalidFuel,
		},
		{
			name:    "only fuel min set is valid",
			filter:  model.TelemetryFilter{FuelMin: float32Ptr(0.1), Limit: 100},
			wantErr: nil,
		},
		{
			name:    "lat min greater than lat max",
			filter:  model.TelemetryFilter{LatMin: float64Ptr(50), LatMax: float64Ptr(10), Limit: 100},
			wantErr: model.ErrInvalidCoords,
		},
		{
			name:    "only lat min set is valid",
			filter:  model.TelemetryFilter{LatMin: float64Ptr(10), Limit: 100},
			wantErr: nil,
		},
		{
			name:    "lon min greater than lon max",
			filter:  model.TelemetryFilter{LonMin: float64Ptr(50), LonMax: float64Ptr(10), Limit: 100},
			wantErr: model.ErrInvalidCoords,
		},
	}

	repo := &mockRepository{}
	logger := logger.NewStdLogger(logger.DebugLevel)
	service := NewTelemetryService(repo, logger)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := service.GetTelemetryList(context.Background(), tt.filter)
			if err != tt.wantErr {
				t.Errorf("got %v, want %v", err, tt.wantErr)
			}
		})
	}
}
