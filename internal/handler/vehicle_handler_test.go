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

type mockVehicleService struct {
	returnError error
}

func (m *mockVehicleService) ProcessVehicle(ctx context.Context, v model.Vehicle) (model.Vehicle, error) {
	if m.returnError != nil {
		return model.Vehicle{}, m.returnError
	}
	v.ID = 1
	return v, nil
}

func (m *mockVehicleService) GetVehicleList(ctx context.Context, filter model.VehicleFilter) ([]model.Vehicle, error) {
	if m.returnError != nil {
		return nil, m.returnError
	}
	return []model.Vehicle{}, nil
}

func (m *mockVehicleService) GetVehicleByID(ctx context.Context, id int) (model.Vehicle, error) {
	if m.returnError != nil {
		return model.Vehicle{}, m.returnError
	}
	return model.Vehicle{ID: id}, nil
}

func (m *mockVehicleService) DeleteVehicleByID(ctx context.Context, id int) (model.Vehicle, error) {
	if m.returnError != nil {
		return model.Vehicle{}, m.returnError
	}
	return model.Vehicle{ID: id, Status: model.VehicleStatusDeleted}, nil
}

func (m *mockVehicleService) UpdateVehicleByID(ctx context.Context, id int, upd model.UpdateVehicle) (model.Vehicle, error) {
	if m.returnError != nil {
		return model.Vehicle{}, m.returnError
	}
	return model.Vehicle{ID: id}, nil
}

func TestHandlePostVehicle(t *testing.T) {
	tests := []struct {
		name           string
		serviceError   error
		requestBody    string
		expectedStatus int
	}{
		{
			name:           "success",
			requestBody:    `{"organization_id": 1, "vin": "1HGCM82633A123456", "number_plate": "A123BC77", "model": "Camry"}`,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "invalid json",
			requestBody:    `not-json`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid vin",
			serviceError:   model.ErrInvalidVIN,
			requestBody:    `{"organization_id": 1, "vin": "SHORT", "number_plate": "A123BC77", "model": "Camry"}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "duplicate vin",
			serviceError:   model.ErrDuplicateVIN,
			requestBody:    `{"organization_id": 1, "vin": "1HGCM82633A123456", "number_plate": "A123BC77", "model": "Camry"}`,
			expectedStatus: http.StatusConflict,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &mockVehicleService{returnError: tt.serviceError}
			log := logger.NewStdLogger(logger.DebugLevel)
			h := NewVehicleHandler(service, log)

			r := chi.NewRouter()
			r.Post("/vehicles", h.HandlePostVehicle)

			request := httptest.NewRequest("POST", "/vehicles", strings.NewReader(tt.requestBody))
			recorder := httptest.NewRecorder()
			r.ServeHTTP(recorder, request)

			if recorder.Code != tt.expectedStatus {
				t.Errorf("got status %d, expected %d", recorder.Code, tt.expectedStatus)
			}
		})
	}
}

func TestHandleGetVehicleByID(t *testing.T) {
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
			service := &mockVehicleService{returnError: tt.serviceError}
			log := logger.NewStdLogger(logger.DebugLevel)
			h := NewVehicleHandler(service, log)

			r := chi.NewRouter()
			r.Get("/vehicles/{id}", h.HandleGetVehicleByID)

			request := httptest.NewRequest("GET", "/vehicles/"+tt.urlID, nil)
			recorder := httptest.NewRecorder()
			r.ServeHTTP(recorder, request)

			if recorder.Code != tt.expectedStatus {
				t.Errorf("got %v expected %v", recorder.Code, tt.expectedStatus)
			}
		})
	}
}

func TestHandleDeleteVehicle(t *testing.T) {
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
			service := &mockVehicleService{returnError: tt.serviceError}
			log := logger.NewStdLogger(logger.DebugLevel)
			h := NewVehicleHandler(service, log)

			r := chi.NewRouter()
			r.Delete("/vehicles/{id}", h.HandleDeleteVehicle)

			request := httptest.NewRequest("DELETE", "/vehicles/"+tt.urlID, nil)
			recorder := httptest.NewRecorder()
			r.ServeHTTP(recorder, request)

			if recorder.Code != tt.expectedStatus {
				t.Errorf("got %v expected %v", recorder.Code, tt.expectedStatus)
			}
		})
	}
}

func TestHandleGetListVehicle(t *testing.T) {
	tests := []struct {
		name           string
		query          string
		serviceError   error
		expectedStatus int
	}{
		{name: "ok", query: "", expectedStatus: http.StatusOK},
		{name: "invalid filter", query: "?organization_id=abc", expectedStatus: http.StatusBadRequest},
		{name: "service error", query: "", serviceError: model.ErrInvalidOrganizationID, expectedStatus: http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &mockVehicleService{returnError: tt.serviceError}
			log := logger.NewStdLogger(logger.DebugLevel)
			h := NewVehicleHandler(service, log)

			r := chi.NewRouter()
			r.Get("/vehicles", h.HandleGetListVehicle)

			request := httptest.NewRequest("GET", "/vehicles"+tt.query, nil)
			recorder := httptest.NewRecorder()
			r.ServeHTTP(recorder, request)

			if recorder.Code != tt.expectedStatus {
				t.Errorf("got %v expected %v", recorder.Code, tt.expectedStatus)
			}
		})
	}
}

func TestHandlePatchVehicle(t *testing.T) {
	tests := []struct {
		name           string
		urlID          string
		requestBody    string
		serviceError   error
		expectedStatus int
	}{
		{name: "ok", urlID: "31", requestBody: `{"number_plate": "B999XY77"}`, expectedStatus: http.StatusOK},
		{name: "invalid id", urlID: "hello", requestBody: `{}`, expectedStatus: http.StatusBadRequest},
		{name: "invalid json", urlID: "31", requestBody: `not-json`, expectedStatus: http.StatusBadRequest},
		{name: "not found", urlID: "999", requestBody: `{}`, serviceError: model.ErrNotFound, expectedStatus: http.StatusNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &mockVehicleService{returnError: tt.serviceError}
			log := logger.NewStdLogger(logger.DebugLevel)
			h := NewVehicleHandler(service, log)

			r := chi.NewRouter()
			r.Patch("/vehicles/{id}", h.HandlePatchVehicle)

			request := httptest.NewRequest("PATCH", "/vehicles/"+tt.urlID, strings.NewReader(tt.requestBody))
			recorder := httptest.NewRecorder()
			r.ServeHTTP(recorder, request)

			if recorder.Code != tt.expectedStatus {
				t.Errorf("got %v expected %v", recorder.Code, tt.expectedStatus)
			}
		})
	}
}
