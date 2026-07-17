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

type mockDeviceService struct {
	returnError error
}

func (m *mockDeviceService) ProcessDevice(ctx context.Context, d model.Device) (model.Device, error) {
	if m.returnError != nil {
		return model.Device{}, m.returnError
	}
	d.ID = 1
	return d, nil
}

func (m *mockDeviceService) GetDeviceByID(ctx context.Context, id int) (model.Device, error) {
	if m.returnError != nil {
		return model.Device{}, m.returnError
	}
	return model.Device{ID: id}, nil
}

func (m *mockDeviceService) DeleteDevice(ctx context.Context, id int) (model.Device, error) {
	if m.returnError != nil {
		return model.Device{}, m.returnError
	}
	return model.Device{ID: id, Status: model.DeviceStatusInactive}, nil
}

func TestHandlePostDevice(t *testing.T) {
	tests := []struct {
		name           string
		serviceError   error
		requestBody    string
		expectedStatus int
	}{
		{
			name:           "success",
			requestBody:    `{"serial_number": "DEV-001"}`,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "invalid json",
			requestBody:    `not-json`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid serial number",
			serviceError:   model.ErrInvalidSerialNumber,
			requestBody:    `{"serial_number": ""}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "duplicate serial number",
			serviceError:   model.ErrDuplicateSerialNumber,
			requestBody:    `{"serial_number": "DEV-001"}`,
			expectedStatus: http.StatusConflict,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &mockDeviceService{returnError: tt.serviceError}
			log := logger.NewStdLogger(logger.DebugLevel)
			h := NewDeviceHandler(service, log)

			r := chi.NewRouter()
			r.Post("/devices", h.HandlePostDevice)

			request := httptest.NewRequest("POST", "/devices", strings.NewReader(tt.requestBody))
			recorder := httptest.NewRecorder()
			r.ServeHTTP(recorder, request)

			if recorder.Code != tt.expectedStatus {
				t.Errorf("got status %d, expected %d", recorder.Code, tt.expectedStatus)
			}
		})
	}
}

func TestHandleGetDeviceByID(t *testing.T) {
	tests := []struct {
		name           string
		urlID          string
		serviceError   error
		expectedStatus int
	}{
		{name: "ok", urlID: "31", expectedStatus: http.StatusOK},
		{name: "not found", urlID: "7777777", serviceError: model.ErrNotFound, expectedStatus: http.StatusNotFound},
		{name: "invalid id", urlID: "hello", expectedStatus: http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &mockDeviceService{returnError: tt.serviceError}
			log := logger.NewStdLogger(logger.DebugLevel)
			h := NewDeviceHandler(service, log)

			r := chi.NewRouter()
			r.Get("/devices/{id}", h.HandleGetDeviceByID)

			request := httptest.NewRequest("GET", "/devices/"+tt.urlID, nil)
			recorder := httptest.NewRecorder()
			r.ServeHTTP(recorder, request)

			if recorder.Code != tt.expectedStatus {
				t.Errorf("got %v expected %v", recorder.Code, tt.expectedStatus)
			}
		})
	}
}

func TestHandleDeleteDeviceByID(t *testing.T) {
	tests := []struct {
		name           string
		urlID          string
		serviceError   error
		expectedStatus int
	}{
		{name: "ok", urlID: "31", expectedStatus: http.StatusOK},
		{name: "not found", urlID: "7777777", serviceError: model.ErrNotFound, expectedStatus: http.StatusNotFound},
		{name: "invalid id", urlID: "hello", expectedStatus: http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &mockDeviceService{returnError: tt.serviceError}
			log := logger.NewStdLogger(logger.DebugLevel)
			h := NewDeviceHandler(service, log)

			r := chi.NewRouter()
			r.Delete("/devices/{id}", h.HandleDeleteDeviceByID)

			request := httptest.NewRequest("DELETE", "/devices/"+tt.urlID, nil)
			recorder := httptest.NewRecorder()
			r.ServeHTTP(recorder, request)

			if recorder.Code != tt.expectedStatus {
				t.Errorf("got %v expected %v", recorder.Code, tt.expectedStatus)
			}
		})
	}
}
