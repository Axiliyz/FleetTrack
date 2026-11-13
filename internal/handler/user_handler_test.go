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

// mockUserService имитирует UserService и запоминает, с чем его вызвали
type mockUserService struct {
	returnError error
	lastCreated model.User
	lastActor   model.AuthContext
}

func (m *mockUserService) CreateUser(ctx context.Context, u model.User, password string) (model.User, error) {
	if m.returnError != nil {
		return model.User{}, m.returnError
	}
	m.lastCreated = u
	u.ID = 10
	return u, nil
}

func (m *mockUserService) GetUserByID(ctx context.Context, id int) (model.User, error) {
	if m.returnError != nil {
		return model.User{}, m.returnError
	}
	return model.User{ID: id, OrganizationID: 1, Email: "user@fleet.test"}, nil
}

func (m *mockUserService) GetUserByEmail(ctx context.Context, email string) (model.User, error) {
	return model.User{}, m.returnError
}

func (m *mockUserService) GetUsersList(ctx context.Context, filter model.UserFilter) ([]model.User, error) {
	return nil, m.returnError
}

func (m *mockUserService) DeleteUserByID(ctx context.Context, id int, actor model.AuthContext) (model.User, error) {
	m.lastActor = actor
	if m.returnError != nil {
		return model.User{}, m.returnError
	}
	return model.User{ID: id, OrganizationID: actor.OrganizationID}, nil
}

func withAuth(r *http.Request, authCtx model.AuthContext) *http.Request {
	return r.WithContext(middleware.ContextWithAuth(r.Context(), authCtx))
}

// TestHandleCreateUser_OrgFromTokenAndDriverLink проверяет, что пользователь создаётся
// в организации из токена, а driver_id из тела доезжает до сервиса
func TestHandleCreateUser_OrgFromTokenAndDriverLink(t *testing.T) {
	svc := &mockUserService{}
	h := NewUserHandler(svc, logger.NewStdLogger(logger.ErrorLevel))
	r := chi.NewRouter()
	r.Post("/users", h.HandleCreateUser)

	body := `{"organization_id": 999, "name": "Саня", "email": "sanya@fleet.test", "password": "x", "role": "DRIVER", "driver_id": 4}`
	req := withAuth(httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(body)),
		model.AuthContext{UserID: 1, OrganizationID: 3, Role: model.UserRoleAdmin})
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d", rec.Code)
	}
	if svc.lastCreated.OrganizationID != 3 {
		t.Errorf("organization = %d, want 3 from token", svc.lastCreated.OrganizationID)
	}
	if svc.lastCreated.DriverID == nil || *svc.lastCreated.DriverID != 4 {
		t.Errorf("driver_id = %v, want 4", svc.lastCreated.DriverID)
	}
}

// TestHandleGetUserByID_DriverSeesOnlySelf проверяет, что водитель не получает карточки коллег с почтами
func TestHandleGetUserByID_DriverSeesOnlySelf(t *testing.T) {
	driverID := 4
	tests := []struct {
		name           string
		path           string
		authCtx        model.AuthContext
		expectedStatus int
	}{
		{"водитель о себе", "/users/5", model.AuthContext{UserID: 5, OrganizationID: 1, Role: model.UserRoleDriver, DriverID: &driverID}, http.StatusOK},
		{"водитель о коллеге", "/users/6", model.AuthContext{UserID: 5, OrganizationID: 1, Role: model.UserRoleDriver, DriverID: &driverID}, http.StatusNotFound},
		{"диспетчер о коллеге", "/users/6", model.AuthContext{UserID: 5, OrganizationID: 1, Role: model.UserRoleDispatcher}, http.StatusOK},
		{"пользователь чужой организации", "/users/6", model.AuthContext{UserID: 5, OrganizationID: 2, Role: model.UserRoleAdmin}, http.StatusNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewUserHandler(&mockUserService{}, logger.NewStdLogger(logger.ErrorLevel))
			r := chi.NewRouter()
			r.Get("/users/{id}", h.HandleGetUserByID)

			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, withAuth(httptest.NewRequest(http.MethodGet, tt.path, nil), tt.authCtx))
			if rec.Code != tt.expectedStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.expectedStatus)
			}
		})
	}
}

// TestHandleDeleteUserByID проверяет, что сервис получает автора удаления, а его отказы
// (удалить себя, последнего админа) превращаются в 409
func TestHandleDeleteUserByID(t *testing.T) {
	tests := []struct {
		name           string
		serviceError   error
		expectedStatus int
	}{
		{"успех", nil, http.StatusOK},
		{"удалить себя", model.ErrDeleteSelf, http.StatusConflict},
		{"последний админ", model.ErrLastAdmin, http.StatusConflict},
		{"чужой пользователь", model.ErrNotFound, http.StatusNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockUserService{returnError: tt.serviceError}
			h := NewUserHandler(svc, logger.NewStdLogger(logger.ErrorLevel))
			r := chi.NewRouter()
			r.Delete("/users/{id}", h.HandleDeleteUserByID)

			actor := model.AuthContext{UserID: 1, OrganizationID: 3, Role: model.UserRoleAdmin}
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, withAuth(httptest.NewRequest(http.MethodDelete, "/users/2", nil), actor))

			if rec.Code != tt.expectedStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.expectedStatus)
			}
			if svc.lastActor != actor {
				t.Errorf("actor = %+v, want %+v", svc.lastActor, actor)
			}
		})
	}
}
