package handler

import (
	"context"
	"fleettrack/internal/logger"
	"fleettrack/internal/model"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type mockTelemetryService struct {
	returnError error
}

func (m *mockTelemetryService) ProcessTelemetry(ctx context.Context, t model.Telemetry) (model.Telemetry, error) {
	if m.returnError != nil {
		return model.Telemetry{}, m.returnError
	} else {
		return model.Telemetry{
			TelemetryID: 123,
			VehicleID:   1,
			DeviceID:    12,
			Lat:         44.4,
			Lon:         44.4,
			Fuel:        0.5,
		}, nil
	}
}

func (m *mockTelemetryService) GetTelemetryList(ctx context.Context, limit int) ([]model.Telemetry, error) {
	return []model.Telemetry{}, nil
}

func (m *mockTelemetryService) GetTelemetryByID(ctx context.Context, id int) (model.Telemetry, error) {
	return model.Telemetry{}, nil
}

func (m *mockTelemetryService) GetTelemetryByVehicle(ctx context.Context, id int) ([]model.Telemetry, error) {
	return []model.Telemetry{}, nil
}

func TestHandleTelemetry(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		serviceError   error
		requestBody    string
		expectedStatus int
	}{
		{
			name:           "success",
			method:         "POST",
			serviceError:   nil,
			requestBody:    `{"device_id": 1, "vehicle_id": 1, "lat": 55.75, "lon": 37.61, "fuel": 0.8}`,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "wrong method",
			method:         "DELETE",
			serviceError:   nil,
			requestBody:    `{"device_id": 1, "vehicle_id": 1, "lat": 55.75, "lon": 37.61, "fuel": 0.8}`,
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name:           "error in fuel",
			method:         "POST",
			serviceError:   model.ErrInvalidFuel,
			requestBody:    `{"device_id": 1, "vehicle_id": 1, "lat": 55.75, "lon": 37.61, "fuel": 1.8}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "error in device id",
			method:         "POST",
			serviceError:   model.ErrInvalidDeviceID,
			requestBody:    `{"device_id": -1, "vehicle_id": 1, "lat": 55.75, "lon": 37.61, "fuel": 0.68}`,
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "bad coords",
			method:         "POST",
			serviceError:   model.ErrInvalidCoords,
			requestBody:    `{"device_id": -1, "vehicle_id": 1, "lat": 255.75, "lon": 37.61, "fuel": 0.68}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid json",
			method:         "POST",
			serviceError:   nil,
			requestBody:    ``,
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &mockTelemetryService{returnError: tt.serviceError}
			logger := logger.NewStdLogger(logger.DebugLevel)
			handler := NewTelemetryHandler(service, logger)

			body := strings.NewReader(tt.requestBody)
			request := httptest.NewRequest(tt.method, "/telemetry", body)

			recorder := httptest.NewRecorder()
			handler.HandleTelemetry(recorder, request)
			if recorder.Code != tt.expectedStatus {
				t.Errorf("got status %d, expected %d", recorder.Code, tt.expectedStatus)
			}
		})
	}
}
