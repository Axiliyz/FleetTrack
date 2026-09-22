package handler

import (
	"context"
	"fleettrack/internal/logger"
	"fleettrack/internal/middleware"
	"fleettrack/internal/model"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

type mockTripService struct {
	returnError error
}

func (m *mockTripService) AssignTrip(ctx context.Context, driverID, vehicleID int, organizationID *int) (model.Trip, error) {
	if m.returnError != nil {
		return model.Trip{}, m.returnError
	}
	return model.Trip{ID: 1, DriverID: driverID, VehicleID: vehicleID, Status: model.TripStatusRunning}, nil
}

func (m *mockTripService) UpdateTrip(ctx context.Context, id int, upd model.Trip, organizationID *int) (model.Trip, error) {
	if m.returnError != nil {
		return model.Trip{}, m.returnError
	}
	return model.Trip{ID: id, Status: upd.Status}, nil
}

func (m *mockTripService) DeleteTrip(ctx context.Context, id int, organizationID *int) (model.Trip, error) {
	if m.returnError != nil {
		return model.Trip{}, m.returnError
	}
	return model.Trip{ID: id, Status: model.TripStatusCancelled}, nil
}

func (m *mockTripService) GetListTrips(ctx context.Context, filter model.TripFilter) ([]model.Trip, error) {
	if m.returnError != nil {
		return nil, m.returnError
	}
	return []model.Trip{}, nil
}

func (m *mockTripService) GetTripByID(ctx context.Context, id int, organizationID *int) (model.Trip, error) {
	if m.returnError != nil {
		return model.Trip{}, m.returnError
	}
	return model.Trip{ID: id}, nil
}

func TestHandleAssignTrip(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    string
		withAuth       bool
		serviceError   error
		expectedStatus int
	}{
		{name: "success", requestBody: `{"driver_id": 1, "vehicle_id": 1}`, withAuth: true, expectedStatus: http.StatusCreated},
		{name: "unauthorized", requestBody: `{"driver_id": 1, "vehicle_id": 1}`, withAuth: false, expectedStatus: http.StatusUnauthorized},
		{name: "invalid json", requestBody: `not-json`, withAuth: true, expectedStatus: http.StatusBadRequest},
		{name: "invalid driver id", requestBody: `{"driver_id": 0, "vehicle_id": 1}`, withAuth: true, serviceError: model.ErrInvalidDriverID, expectedStatus: http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &mockTripService{returnError: tt.serviceError}
			log := logger.NewStdLogger(logger.DebugLevel)
			h := NewTripHandler(service, log)

			r := chi.NewRouter()
			r.Post("/trips", h.HandleAssignTrip)

			request := httptest.NewRequest("POST", "/trips", strings.NewReader(tt.requestBody))
			if tt.withAuth {
				authCtx := model.AuthContext{Role: model.UserRoleAdmin, OrganizationID: 1}
				request = request.WithContext(middleware.ContextWithAuth(request.Context(), authCtx))
			}
			recorder := httptest.NewRecorder()
			r.ServeHTTP(recorder, request)

			if recorder.Code != tt.expectedStatus {
				t.Errorf("got status %d, expected %d", recorder.Code, tt.expectedStatus)
			}
		})
	}
}

func TestHandleUpdateTrip(t *testing.T) {
	tests := []struct {
		name           string
		urlID          string
		requestBody    string
		withAuth       bool
		serviceError   error
		expectedStatus int
	}{
		{name: "ok", urlID: "1", requestBody: `{"status": "SUCCEEDED"}`, withAuth: true, expectedStatus: http.StatusOK},
		{name: "unauthorized", urlID: "1", requestBody: `{"status": "SUCCEEDED"}`, withAuth: false, expectedStatus: http.StatusUnauthorized},
		{name: "invalid id", urlID: "hello", requestBody: `{}`, withAuth: true, expectedStatus: http.StatusBadRequest},
		{name: "invalid json", urlID: "1", requestBody: `not-json`, withAuth: true, expectedStatus: http.StatusBadRequest},
		{name: "already finished", urlID: "1", requestBody: `{"status": "RUNNING"}`, withAuth: true, serviceError: model.ErrTripAlreadyFinished, expectedStatus: http.StatusConflict},
		{name: "not found", urlID: "999", requestBody: `{"status": "SUCCEEDED"}`, withAuth: true, serviceError: model.ErrNotFound, expectedStatus: http.StatusNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &mockTripService{returnError: tt.serviceError}
			log := logger.NewStdLogger(logger.DebugLevel)
			h := NewTripHandler(service, log)

			r := chi.NewRouter()
			r.Patch("/trips/{id}", h.HandleUpdateTrip)

			request := httptest.NewRequest("PATCH", "/trips/"+tt.urlID, strings.NewReader(tt.requestBody))
			if tt.withAuth {
				authCtx := model.AuthContext{Role: model.UserRoleAdmin, OrganizationID: 1}
				request = request.WithContext(middleware.ContextWithAuth(request.Context(), authCtx))
			}
			recorder := httptest.NewRecorder()
			r.ServeHTTP(recorder, request)

			if recorder.Code != tt.expectedStatus {
				t.Errorf("got status %d, expected %d", recorder.Code, tt.expectedStatus)
			}
		})
	}
}

func TestHandleDeleteTrip(t *testing.T) {
	tests := []struct {
		name           string
		urlID          string
		withAuth       bool
		serviceError   error
		expectedStatus int
	}{
		{name: "ok", urlID: "1", withAuth: true, expectedStatus: http.StatusOK},
		{name: "unauthorized", urlID: "1", withAuth: false, expectedStatus: http.StatusUnauthorized},
		{name: "invalid id", urlID: "hello", withAuth: true, expectedStatus: http.StatusBadRequest},
		{name: "already finished", urlID: "1", withAuth: true, serviceError: model.ErrTripAlreadyFinished, expectedStatus: http.StatusConflict},
		{name: "not found", urlID: "999", withAuth: true, serviceError: model.ErrNotFound, expectedStatus: http.StatusNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &mockTripService{returnError: tt.serviceError}
			log := logger.NewStdLogger(logger.DebugLevel)
			h := NewTripHandler(service, log)

			r := chi.NewRouter()
			r.Delete("/trips/{id}", h.HandleDeleteTrip)

			request := httptest.NewRequest("DELETE", "/trips/"+tt.urlID, nil)
			if tt.withAuth {
				authCtx := model.AuthContext{Role: model.UserRoleAdmin, OrganizationID: 1}
				request = request.WithContext(middleware.ContextWithAuth(request.Context(), authCtx))
			}
			recorder := httptest.NewRecorder()
			r.ServeHTTP(recorder, request)

			if recorder.Code != tt.expectedStatus {
				t.Errorf("got status %d, expected %d", recorder.Code, tt.expectedStatus)
			}
		})
	}
}

func TestHandleGetListTrips(t *testing.T) {
	tests := []struct {
		name           string
		query          string
		withAuth       bool
		serviceError   error
		expectedStatus int
	}{
		{name: "ok", query: "", withAuth: true, expectedStatus: http.StatusOK},
		{name: "unauthorized", query: "", withAuth: false, expectedStatus: http.StatusUnauthorized},
		{name: "invalid filter", query: "?driver_id=abc", withAuth: true, expectedStatus: http.StatusBadRequest},
		{name: "service error", query: "", withAuth: true, serviceError: model.ErrInvalidVehicleID, expectedStatus: http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &mockTripService{returnError: tt.serviceError}
			log := logger.NewStdLogger(logger.DebugLevel)
			h := NewTripHandler(service, log)

			r := chi.NewRouter()
			r.Get("/trips", h.HandleGetListTrips)

			request := httptest.NewRequest("GET", "/trips"+tt.query, nil)
			if tt.withAuth {
				authCtx := model.AuthContext{Role: model.UserRoleAdmin, OrganizationID: 1}
				request = request.WithContext(middleware.ContextWithAuth(request.Context(), authCtx))
			}
			recorder := httptest.NewRecorder()
			r.ServeHTTP(recorder, request)

			if recorder.Code != tt.expectedStatus {
				t.Errorf("got status %d, expected %d", recorder.Code, tt.expectedStatus)
			}
		})
	}
}

func TestHandleGetTripByID(t *testing.T) {
	tests := []struct {
		name           string
		urlID          string
		withAuth       bool
		serviceError   error
		expectedStatus int
	}{
		{name: "ok", urlID: "1", withAuth: true, expectedStatus: http.StatusOK},
		{name: "unauthorized", urlID: "1", withAuth: false, expectedStatus: http.StatusUnauthorized},
		{name: "invalid id", urlID: "hello", withAuth: true, expectedStatus: http.StatusBadRequest},
		{name: "not found", urlID: "999", withAuth: true, serviceError: model.ErrNotFound, expectedStatus: http.StatusNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &mockTripService{returnError: tt.serviceError}
			log := logger.NewStdLogger(logger.DebugLevel)
			h := NewTripHandler(service, log)

			r := chi.NewRouter()
			r.Get("/trips/{id}", h.HandleGetTripByID)

			request := httptest.NewRequest("GET", "/trips/"+tt.urlID, nil)
			if tt.withAuth {
				authCtx := model.AuthContext{Role: model.UserRoleAdmin, OrganizationID: 1}
				request = request.WithContext(middleware.ContextWithAuth(request.Context(), authCtx))
			}
			recorder := httptest.NewRecorder()
			r.ServeHTTP(recorder, request)

			if recorder.Code != tt.expectedStatus {
				t.Errorf("got status %d, expected %d", recorder.Code, tt.expectedStatus)
			}
		})
	}
}
