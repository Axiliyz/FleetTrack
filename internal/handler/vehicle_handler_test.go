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

type mockVehicleService struct {
	returnError error
}

func (m *mockVehicleService) CreateVehicle(ctx context.Context, v model.Vehicle) (model.Vehicle, error) {
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
	return model.Vehicle{ID: id, OrganizationID: 1}, nil
}

func (m *mockVehicleService) DeleteVehicleByID(ctx context.Context, id int, organizationID *int) (model.Vehicle, error) {
	if m.returnError != nil {
		return model.Vehicle{}, m.returnError
	}
	return model.Vehicle{ID: id, Status: model.VehicleStatusDeleted}, nil
}

func (m *mockVehicleService) UpdateVehicleByID(ctx context.Context, id int, upd model.UpdateVehicle, organizationID *int) (model.Vehicle, error) {
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
			requestBody:    `{"vin": "1HGCM82633A123456", "number_plate": "A123BC77", "model": "Camry"}`,
			expectedStatus: http.StatusCreated,
		},
		{
			name:           "invalid json",
			requestBody:    `not-json`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid vin",
			serviceError:   model.ErrInvalidVIN,
			requestBody:    `{"vin": "SHORT", "number_plate": "A123BC77", "model": "Camry"}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "duplicate vin",
			serviceError:   model.ErrDuplicateVIN,
			requestBody:    `{"vin": "1HGCM82633A123456", "number_plate": "A123BC77", "model": "Camry"}`,
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
			authCtx := model.AuthContext{Role: model.UserRoleAdmin, OrganizationID: 1}
			request = request.WithContext(middleware.ContextWithAuth(request.Context(), authCtx))
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
			authCtx := model.AuthContext{Role: model.UserRoleAdmin, OrganizationID: 1}
			request = request.WithContext(middleware.ContextWithAuth(request.Context(), authCtx))
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
			authCtx := model.AuthContext{Role: model.UserRoleAdmin, OrganizationID: 1}
			request = request.WithContext(middleware.ContextWithAuth(request.Context(), authCtx))
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
			authCtx := model.AuthContext{Role: model.UserRoleAdmin, OrganizationID: 1}
			request = request.WithContext(middleware.ContextWithAuth(request.Context(), authCtx))
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
			authCtx := model.AuthContext{Role: model.UserRoleAdmin, OrganizationID: 1}
			request = request.WithContext(middleware.ContextWithAuth(request.Context(), authCtx))
			recorder := httptest.NewRecorder()
			r.ServeHTTP(recorder, request)

			if recorder.Code != tt.expectedStatus {
				t.Errorf("got %v expected %v", recorder.Code, tt.expectedStatus)
			}
		})
	}
}

// TestHandlePatchVehicle_RejectsUnknownFields проверяет, что машину нельзя перенести в другую организацию:
// поля organization_id в запросе нет, а неизвестное поле - 400, а не тихий пропуск
func TestHandlePatchVehicle_RejectsUnknownFields(t *testing.T) {
	tests := []struct {
		name           string
		body           string
		serviceError   error
		expectedStatus int
	}{
		{"смена номера", `{"number_plate": "A123BC77"}`, nil, http.StatusOK},
		{"попытка сменить организацию", `{"organization_id": 2}`, nil, http.StatusBadRequest},
		{"статус машины в рейсе", `{"status": "IDLE"}`, model.ErrVehicleOnTrip, http.StatusConflict},
		{"статус ON_TRIP вручную", `{"status": "ON_TRIP"}`, model.ErrInvalidStatus, http.StatusBadRequest},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewVehicleHandler(&mockVehicleService{returnError: tt.serviceError}, logger.NewStdLogger(logger.ErrorLevel))
			r := chi.NewRouter()
			r.Patch("/vehicles/{id}", handler.HandlePatchVehicle)

			request := httptest.NewRequest(http.MethodPatch, "/vehicles/1", strings.NewReader(tt.body))
			request = request.WithContext(middleware.ContextWithAuth(request.Context(), model.AuthContext{Role: model.UserRoleDispatcher, OrganizationID: 1}))
			recorder := httptest.NewRecorder()
			r.ServeHTTP(recorder, request)

			if recorder.Code != tt.expectedStatus {
				t.Errorf("status = %d, want %d", recorder.Code, tt.expectedStatus)
			}
		})
	}
}
