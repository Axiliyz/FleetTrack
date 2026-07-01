package handler

import (
	"context"
	"fleettrack/internal/logger"
	"fleettrack/internal/model"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
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
	if m.returnError != nil {
		return []model.Telemetry{}, m.returnError
	}
	return []model.Telemetry{}, nil
}

func (m *mockTelemetryService) GetTelemetryByID(ctx context.Context, id int) (model.Telemetry, error) {
	if m.returnError != nil {
		return model.Telemetry{}, m.returnError
	}
	return model.Telemetry{}, nil
}

func (m *mockTelemetryService) GetTelemetryByVehicle(ctx context.Context, id int) ([]model.Telemetry, error) {
	if m.returnError != nil {
		return []model.Telemetry{}, m.returnError
	}
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
			method:         "PATCH",
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
			expectedStatus: http.StatusBadRequest,
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

			r := chi.NewRouter()
			r.Post("/telemetry", handler.HandleTelemetry)

			body := strings.NewReader(tt.requestBody)
			request := httptest.NewRequest(tt.method, "/telemetry", body)

			recorder := httptest.NewRecorder()
			r.ServeHTTP(recorder, request)
			if recorder.Code != tt.expectedStatus {
				t.Errorf("got status %d, expected %d", recorder.Code, tt.expectedStatus)
			}
		})
	}
}

func TestHandleGetTelemetryByID(t *testing.T) {
	tests := []struct {
		name           string
		urlID          string
		serviceError   error
		expectedStatus int
	}{
		{
			name:           "ok",
			urlID:          "31",
			serviceError:   nil,
			expectedStatus: 200,
		},
		{
			name:           "not found",
			urlID:          "7777777",
			serviceError:   model.ErrNotFound,
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "invalid id",
			urlID:          "hello",
			serviceError:   nil,
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &mockTelemetryService{returnError: tt.serviceError}
			logger := logger.NewStdLogger(logger.DebugLevel)
			handler := NewTelemetryHandler(service, logger)

			r := chi.NewRouter()
			r.Get("/telemetry/{id}", handler.HandleGetTelemetryByID)

			request := httptest.NewRequest("GET", "/telemetry/"+tt.urlID, nil)
			recorder := httptest.NewRecorder()
			r.ServeHTTP(recorder, request)

			if recorder.Code != tt.expectedStatus {
				t.Errorf("got %v expected %v", recorder.Code, tt.expectedStatus)
			}
		})
	}
}

func TestHandleGetTelemetryByVehicle(t *testing.T) {
	tests := []struct {
		name           string
		vehicleID      string
		serviceError   error
		expectedStatus int
	}{
		{
			name:           "ok",
			vehicleID:      "4",
			serviceError:   nil,
			expectedStatus: 200,
		},
		{
			name:           "not found",
			vehicleID:      "7777777",
			serviceError:   model.ErrNotFound,
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "invalid id",
			vehicleID:      "hello",
			serviceError:   nil,
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &mockTelemetryService{returnError: tt.serviceError}
			logger := logger.NewStdLogger(logger.DebugLevel)
			handler := NewTelemetryHandler(service, logger)

			r := chi.NewRouter()
			r.Get("/telemetry/vehicle/{id}", handler.HandlerGetTelemetryByVehicle)

			request := httptest.NewRequest("GET", "/telemetry/vehicle/"+tt.vehicleID, nil)
			recorder := httptest.NewRecorder()
			r.ServeHTTP(recorder, request)

			if recorder.Code != tt.expectedStatus {
				t.Errorf("got %v expected %v", recorder.Code, tt.expectedStatus)
			}
		})
	}
}
