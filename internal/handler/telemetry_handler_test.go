package handler

import (
	"context"
	"encoding/json"
	"fleettrack/internal/logger"
	"fleettrack/internal/middleware"
	"fleettrack/internal/model"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

type mockTelemetryService struct {
	returnError     error
	lastDriverScope *int
}

func float32Ptr(v float32) *float32 { return &v }

func (m *mockTelemetryService) ProcessTelemetry(ctx context.Context, t model.Telemetry) (model.Telemetry, error) {
	if m.returnError != nil {
		return model.Telemetry{}, m.returnError
	}
	return model.Telemetry{
		TelemetryID: 123,
		VehicleID:   1,
		DeviceID:    12,
		Lat:         44.4,
		Lon:         44.4,
		Fuel:        float32Ptr(0.5),
	}, nil
}

func (m *mockTelemetryService) GetTelemetryList(ctx context.Context, filter model.TelemetryFilter) ([]model.Telemetry, error) {
	if m.returnError != nil {
		return []model.Telemetry{}, m.returnError
	}
	return []model.Telemetry{}, nil
}

func (m *mockTelemetryService) GetTelemetryByID(ctx context.Context, id int, driverID *int) (model.Telemetry, error) {
	m.lastDriverScope = driverID
	if m.returnError != nil {
		return model.Telemetry{}, m.returnError
	}
	return model.Telemetry{OrganizationID: 1}, nil
}

func (m *mockTelemetryService) GetTelemetryByVehicle(ctx context.Context, id int, organizationID, driverID *int) ([]model.Telemetry, error) {
	m.lastDriverScope = driverID
	if m.returnError != nil {
		return []model.Telemetry{}, m.returnError
	}
	return []model.Telemetry{}, nil
}

func (m *mockTelemetryService) DeleteTelemetryByID(ctx context.Context, id int, organizationID *int) (model.Telemetry, error) {
	if m.returnError != nil {
		return model.Telemetry{}, m.returnError
	}
	return model.Telemetry{}, nil
}

func (m *mockTelemetryService) DeleteTelemetryByVehicle(ctx context.Context, id int, organizationID *int) (int64, error) {
	if m.returnError != nil {
		return 0, m.returnError
	}
	return 1500000, nil
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
			expectedStatus: http.StatusCreated,
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
		{
			// без координат точка раньше сохранялась как (0, 0) - Гвинейский залив
			name:           "missing coords",
			method:         "POST",
			serviceError:   nil,
			requestBody:    `{"device_id": 1, "vehicle_id": 1, "fuel": 0.8}`,
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
			authCtx := model.AuthContext{Role: model.UserRoleDriver, OrganizationID: 1}
			request = request.WithContext(middleware.ContextWithAuth(request.Context(), authCtx))

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

func TestHandleDeleteTelemetryByID(t *testing.T) {
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
			expectedStatus: http.StatusOK,
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
			r.Delete("/telemetry/{id}", handler.HandleDeleteTelemetryByID)

			request := httptest.NewRequest("DELETE", "/telemetry/"+tt.urlID, nil)
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

func TestHandleDeleteTelemetryByVehicleID(t *testing.T) {
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
			expectedStatus: http.StatusOK,
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
			r.Delete("/telemetry/vehicle/{id}", handler.HandleDeleteTelemetryByVehicleID)

			request := httptest.NewRequest("DELETE", "/telemetry/vehicle/"+tt.vehicleID, nil)
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
			r.Get("/telemetry/vehicle/{id}", handler.HandleGetTelemetryByVehicle)

			request := httptest.NewRequest("GET", "/telemetry/vehicle/"+tt.vehicleID, nil)
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

// TestHandleTelemetry_ResponseHasCoordinates проверяет, что ответ телеметрии несёт координаты и топливо для карты
func TestHandleTelemetry_ResponseHasCoordinates(t *testing.T) {
	h := NewTelemetryHandler(&mockTelemetryService{}, logger.NewStdLogger(logger.DebugLevel))
	req := httptest.NewRequest(http.MethodPost, "/telemetry",
		strings.NewReader(`{"device_id": 12, "vehicle_id": 1, "lat": 44.4, "lon": 44.4, "fuel": 0.5}`))
	req = req.WithContext(middleware.ContextWithAuth(req.Context(), model.AuthContext{Role: model.UserRoleDriver, OrganizationID: 1}))
	rec := httptest.NewRecorder()
	h.HandleTelemetry(rec, req)

	var body struct {
		Data map[string]any `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v, body: %s", err, rec.Body.String())
	}
	if body.Data["lat"] != 44.4 || body.Data["lon"] != 44.4 || body.Data["fuel"] != 0.5 {
		t.Errorf("data = %v, want lat/lon 44.4 and fuel 0.5", body.Data)
	}
}

// TestTelemetryReadByID_DriverScope проверяет, что выборки по ID ограничиваются водителем:
// DRIVER видит только точки своих рейсов, остальные роли - без ограничения
func TestTelemetryReadByID_DriverScope(t *testing.T) {
	driverID := 9
	tests := []struct {
		name           string
		path           string
		authCtx        model.AuthContext
		expectedStatus int
		wantScope      *int
	}{
		{"точка: водитель", "/telemetry/5", model.AuthContext{Role: model.UserRoleDriver, OrganizationID: 1, DriverID: &driverID}, http.StatusOK, &driverID},
		{"точка: диспетчер", "/telemetry/5", model.AuthContext{Role: model.UserRoleDispatcher, OrganizationID: 1}, http.StatusOK, nil},
		{"точка: водитель без привязки", "/telemetry/5", model.AuthContext{Role: model.UserRoleDriver, OrganizationID: 1}, http.StatusForbidden, nil},
		{"машина: водитель", "/telemetry/vehicles/3", model.AuthContext{Role: model.UserRoleDriver, OrganizationID: 1, DriverID: &driverID}, http.StatusOK, &driverID},
		{"машина: аналитик", "/telemetry/vehicles/3", model.AuthContext{Role: model.UserRoleAnalytic, OrganizationID: 1}, http.StatusOK, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &mockTelemetryService{}
			handler := NewTelemetryHandler(service, logger.NewStdLogger(logger.ErrorLevel))

			r := chi.NewRouter()
			r.Get("/telemetry/vehicles/{id}", handler.HandleGetTelemetryByVehicle)
			r.Get("/telemetry/{id}", handler.HandleGetTelemetryByID)

			request := httptest.NewRequest(http.MethodGet, tt.path, nil)
			request = request.WithContext(middleware.ContextWithAuth(request.Context(), tt.authCtx))
			recorder := httptest.NewRecorder()
			r.ServeHTTP(recorder, request)

			if recorder.Code != tt.expectedStatus {
				t.Fatalf("status = %d, want %d", recorder.Code, tt.expectedStatus)
			}
			if tt.expectedStatus != http.StatusOK {
				return
			}
			if (tt.wantScope == nil) != (service.lastDriverScope == nil) ||
				(tt.wantScope != nil && *tt.wantScope != *service.lastDriverScope) {
				t.Errorf("driver scope = %v, want %v", service.lastDriverScope, tt.wantScope)
			}
		})
	}
}

// TestHandleDeleteTelemetryByVehicleID_ReturnsCount проверяет, что массовое удаление отдаёт число строк, а не сами строки
func TestHandleDeleteTelemetryByVehicleID_ReturnsCount(t *testing.T) {
	handler := NewTelemetryHandler(&mockTelemetryService{}, logger.NewStdLogger(logger.ErrorLevel))
	r := chi.NewRouter()
	r.Delete("/telemetry/vehicles/{id}", handler.HandleDeleteTelemetryByVehicleID)

	request := httptest.NewRequest(http.MethodDelete, "/telemetry/vehicles/3", nil)
	request = request.WithContext(middleware.ContextWithAuth(request.Context(), model.AuthContext{Role: model.UserRoleAdmin, OrganizationID: 1}))
	recorder := httptest.NewRecorder()
	r.ServeHTTP(recorder, request)

	var resp struct {
		Data struct {
			Deleted int64 `json:"deleted"`
		} `json:"data"`
	}
	if err := json.NewDecoder(recorder.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if recorder.Code != http.StatusOK || resp.Data.Deleted != 1500000 {
		t.Errorf("status = %d, deleted = %d", recorder.Code, resp.Data.Deleted)
	}
}
