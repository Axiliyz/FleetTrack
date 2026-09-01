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

type mockTripService struct {
	returnError error
}

func (m *mockTripService) AssignTrip(ctx context.Context, driverID, vehicleID int) (model.Trip, error) {
	if m.returnError != nil {
		return model.Trip{}, m.returnError
	}
	return model.Trip{ID: 1, DriverID: driverID, VehicleID: vehicleID, Status: model.TripStatusRunning}, nil
}

func (m *mockTripService) UpdateTrip(ctx context.Context, id int, upd model.Trip) (model.Trip, error) {
	if m.returnError != nil {
		return model.Trip{}, m.returnError
	}
	return model.Trip{ID: id, Status: upd.Status}, nil
}

func (m *mockTripService) DeleteTrip(ctx context.Context, id int) (model.Trip, error) {
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

func (m *mockTripService) GetTripByID(ctx context.Context, id int) (model.Trip, error) {
	return model.Trip{}, nil
}

func TestHandleAssignTrip(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    string
		serviceError   error
		expectedStatus int
	}{
		{name: "success", requestBody: `{"driver_id": 1, "vehicle_id": 1}`, expectedStatus: http.StatusOK},
		{name: "invalid json", requestBody: `not-json`, expectedStatus: http.StatusBadRequest},
		{name: "invalid driver id", requestBody: `{"driver_id": 0, "vehicle_id": 1}`, serviceError: model.ErrInvalidDriverID, expectedStatus: http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &mockTripService{returnError: tt.serviceError}
			log := logger.NewStdLogger(logger.DebugLevel)
			h := NewTripHandler(service, log)

			r := chi.NewRouter()
			r.Post("/trips", h.HandleAssignTrip)

			request := httptest.NewRequest("POST", "/trips", strings.NewReader(tt.requestBody))
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
		serviceError   error
		expectedStatus int
	}{
		{name: "ok", urlID: "1", requestBody: `{"status": "SUCCEEDED"}`, expectedStatus: http.StatusOK},
		{name: "invalid id", urlID: "hello", requestBody: `{}`, expectedStatus: http.StatusBadRequest},
		{name: "invalid json", urlID: "1", requestBody: `not-json`, expectedStatus: http.StatusBadRequest},
		{name: "already finished", urlID: "1", requestBody: `{"status": "RUNNING"}`, serviceError: model.ErrTripAlreadyFinished, expectedStatus: http.StatusConflict},
		{name: "not found", urlID: "999", requestBody: `{"status": "SUCCEEDED"}`, serviceError: model.ErrNotFound, expectedStatus: http.StatusNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &mockTripService{returnError: tt.serviceError}
			log := logger.NewStdLogger(logger.DebugLevel)
			h := NewTripHandler(service, log)

			r := chi.NewRouter()
			r.Patch("/trips/{id}", h.HandleUpdateTrip)

			request := httptest.NewRequest("PATCH", "/trips/"+tt.urlID, strings.NewReader(tt.requestBody))
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
		serviceError   error
		expectedStatus int
	}{
		{name: "ok", urlID: "1", expectedStatus: http.StatusOK},
		{name: "invalid id", urlID: "hello", expectedStatus: http.StatusBadRequest},
		{name: "already finished", urlID: "1", serviceError: model.ErrTripAlreadyFinished, expectedStatus: http.StatusConflict},
		{name: "not found", urlID: "999", serviceError: model.ErrNotFound, expectedStatus: http.StatusNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &mockTripService{returnError: tt.serviceError}
			log := logger.NewStdLogger(logger.DebugLevel)
			h := NewTripHandler(service, log)

			r := chi.NewRouter()
			r.Delete("/trips/{id}", h.HandleDeleteTrip)

			request := httptest.NewRequest("DELETE", "/trips/"+tt.urlID, nil)
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
		serviceError   error
		expectedStatus int
	}{
		{name: "ok", query: "", expectedStatus: http.StatusOK},
		{name: "invalid filter", query: "?driver_id=abc", expectedStatus: http.StatusBadRequest},
		{name: "service error", query: "", serviceError: model.ErrInvalidVehicleID, expectedStatus: http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &mockTripService{returnError: tt.serviceError}
			log := logger.NewStdLogger(logger.DebugLevel)
			h := NewTripHandler(service, log)

			r := chi.NewRouter()
			r.Get("/trips", h.HandleGetListTrips)

			request := httptest.NewRequest("GET", "/trips"+tt.query, nil)
			recorder := httptest.NewRecorder()
			r.ServeHTTP(recorder, request)

			if recorder.Code != tt.expectedStatus {
				t.Errorf("got status %d, expected %d", recorder.Code, tt.expectedStatus)
			}
		})
	}
}
