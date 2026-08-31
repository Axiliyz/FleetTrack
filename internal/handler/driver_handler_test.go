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

type mockDriverService struct {
	returnError error
}

func (m *mockDriverService) CreateDriver(ctx context.Context, d model.Driver) (model.Driver, error) {
	if m.returnError != nil {
		return model.Driver{}, m.returnError
	}
	d.ID = 1
	return d, nil
}

func (m *mockDriverService) GetDriverByID(ctx context.Context, id int) (model.Driver, error) {
	if m.returnError != nil {
		return model.Driver{}, m.returnError
	}
	return model.Driver{ID: id}, nil
}

func (m *mockDriverService) GetDriverList(ctx context.Context, filter model.DriverFilter) ([]model.Driver, error) {
	if m.returnError != nil {
		return nil, m.returnError
	}
	return []model.Driver{}, nil
}

func (m *mockDriverService) DeleteDriverByID(ctx context.Context, id int) (model.Driver, error) {
	if m.returnError != nil {
		return model.Driver{}, m.returnError
	}
	return model.Driver{ID: id}, nil
}

func (m *mockDriverService) UpdateDriverByID(ctx context.Context, id int, upd model.UpdateDriver) (model.Driver, error) {
	if m.returnError != nil {
		return model.Driver{}, m.returnError
	}
	return model.Driver{ID: id}, nil
}

func TestHandlePostDriver(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    string
		serviceError   error
		expectedStatus int
	}{
		{name: "success", requestBody: `{"organization_id": 1, "name": "Ivan Petrov"}`, expectedStatus: http.StatusOK},
		{name: "invalid json", requestBody: `not-json`, expectedStatus: http.StatusBadRequest},
		{name: "invalid name", requestBody: `{"organization_id": 1, "name": ""}`, serviceError: model.ErrInvalidDriverName, expectedStatus: http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &mockDriverService{returnError: tt.serviceError}
			log := logger.NewStdLogger(logger.DebugLevel)
			h := NewDriverHandler(service, log)

			r := chi.NewRouter()
			r.Post("/drivers", h.HandlePostDriver)

			request := httptest.NewRequest("POST", "/drivers", strings.NewReader(tt.requestBody))
			recorder := httptest.NewRecorder()
			r.ServeHTTP(recorder, request)

			if recorder.Code != tt.expectedStatus {
				t.Errorf("got status %d, expected %d", recorder.Code, tt.expectedStatus)
			}
		})
	}
}

func TestHandleGetDriverByID(t *testing.T) {
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
			service := &mockDriverService{returnError: tt.serviceError}
			log := logger.NewStdLogger(logger.DebugLevel)
			h := NewDriverHandler(service, log)

			r := chi.NewRouter()
			r.Get("/drivers/{id}", h.HandleGetDriverByID)

			request := httptest.NewRequest("GET", "/drivers/"+tt.urlID, nil)
			recorder := httptest.NewRecorder()
			r.ServeHTTP(recorder, request)

			if recorder.Code != tt.expectedStatus {
				t.Errorf("got %v expected %v", recorder.Code, tt.expectedStatus)
			}
		})
	}
}

func TestHandleDeleteDriver(t *testing.T) {
	tests := []struct {
		name           string
		urlID          string
		serviceError   error
		expectedStatus int
	}{
		{name: "ok", urlID: "31", expectedStatus: http.StatusOK},
		{name: "has active trips", urlID: "1", serviceError: model.ErrDriverHasActiveTrips, expectedStatus: http.StatusConflict},
		{name: "not found", urlID: "7777777", serviceError: model.ErrNotFound, expectedStatus: http.StatusNotFound},
		{name: "invalid id", urlID: "hello", expectedStatus: http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &mockDriverService{returnError: tt.serviceError}
			log := logger.NewStdLogger(logger.DebugLevel)
			h := NewDriverHandler(service, log)

			r := chi.NewRouter()
			r.Delete("/drivers/{id}", h.HandleDeleteDriver)

			request := httptest.NewRequest("DELETE", "/drivers/"+tt.urlID, nil)
			recorder := httptest.NewRecorder()
			r.ServeHTTP(recorder, request)

			if recorder.Code != tt.expectedStatus {
				t.Errorf("got %v expected %v", recorder.Code, tt.expectedStatus)
			}
		})
	}
}

func TestHandleGetListDriver(t *testing.T) {
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
			service := &mockDriverService{returnError: tt.serviceError}
			log := logger.NewStdLogger(logger.DebugLevel)
			h := NewDriverHandler(service, log)

			r := chi.NewRouter()
			r.Get("/drivers", h.HandleGetListDriver)

			request := httptest.NewRequest("GET", "/drivers"+tt.query, nil)
			recorder := httptest.NewRecorder()
			r.ServeHTTP(recorder, request)

			if recorder.Code != tt.expectedStatus {
				t.Errorf("got %v expected %v", recorder.Code, tt.expectedStatus)
			}
		})
	}
}

func TestHandlePatchDriver(t *testing.T) {
	tests := []struct {
		name           string
		urlID          string
		requestBody    string
		serviceError   error
		expectedStatus int
	}{
		{name: "ok", urlID: "31", requestBody: `{"name": "New Name"}`, expectedStatus: http.StatusOK},
		{name: "invalid id", urlID: "hello", requestBody: `{}`, expectedStatus: http.StatusBadRequest},
		{name: "invalid json", urlID: "31", requestBody: `not-json`, expectedStatus: http.StatusBadRequest},
		{name: "not found", urlID: "999", requestBody: `{}`, serviceError: model.ErrNotFound, expectedStatus: http.StatusNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &mockDriverService{returnError: tt.serviceError}
			log := logger.NewStdLogger(logger.DebugLevel)
			h := NewDriverHandler(service, log)

			r := chi.NewRouter()
			r.Patch("/drivers/{id}", h.HandlePatchDriver)

			request := httptest.NewRequest("PATCH", "/drivers/"+tt.urlID, strings.NewReader(tt.requestBody))
			recorder := httptest.NewRecorder()
			r.ServeHTTP(recorder, request)

			if recorder.Code != tt.expectedStatus {
				t.Errorf("got %v expected %v", recorder.Code, tt.expectedStatus)
			}
		})
	}
}
