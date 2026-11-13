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

type mockOrgService struct {
	returnError error
	lastOrgID   int
}

func (m *mockOrgService) GetOrgList(ctx context.Context, organizationID int) ([]model.Org, error) {
	if m.returnError != nil {
		return nil, m.returnError
	}
	m.lastOrgID = organizationID
	return []model.Org{{ID: organizationID, Name: "Acme"}}, nil
}

// TestHandleGetListOrg проверяет, что пользователь получает только организацию из своего токена
func TestHandleGetListOrg(t *testing.T) {
	tests := []struct {
		name           string
		authCtx        *model.AuthContext
		serviceError   error
		expectedStatus int
	}{
		{
			name:           "своя организация",
			authCtx:        &model.AuthContext{UserID: 1, OrganizationID: 7, Role: model.UserRoleAnalytic},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "без токена",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "ошибка сервиса",
			authCtx:        &model.AuthContext{UserID: 1, OrganizationID: 7, Role: model.UserRoleAdmin},
			serviceError:   model.ErrServiceUnavailable,
			expectedStatus: http.StatusServiceUnavailable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &mockOrgService{returnError: tt.serviceError}
			handler := NewOrgHandler(service, logger.NewStdLogger(logger.ErrorLevel))

			r := chi.NewRouter()
			r.Get("/organizations", handler.HandleGetListOrg)

			request := httptest.NewRequest(http.MethodGet, "/organizations", nil)
			if tt.authCtx != nil {
				request = request.WithContext(middleware.ContextWithAuth(request.Context(), *tt.authCtx))
			}
			recorder := httptest.NewRecorder()
			r.ServeHTTP(recorder, request)

			if recorder.Code != tt.expectedStatus {
				t.Errorf("got status %d, expected %d", recorder.Code, tt.expectedStatus)
			}
			if tt.expectedStatus == http.StatusOK && service.lastOrgID != tt.authCtx.OrganizationID {
				t.Errorf("organization id = %d, want %d from token", service.lastOrgID, tt.authCtx.OrganizationID)
			}
		})
	}
}

// TestPostOrganizationsRemoved проверяет, что создать пустую организацию без админа больше нельзя:
// организация появляется только через /register
func TestPostOrganizationsRemoved(t *testing.T) {
	handler := NewOrgHandler(&mockOrgService{}, logger.NewStdLogger(logger.ErrorLevel))

	r := chi.NewRouter()
	r.Get("/organizations", handler.HandleGetListOrg)

	request := httptest.NewRequest(http.MethodPost, "/organizations", strings.NewReader(`{"name":"Acme"}`))
	recorder := httptest.NewRecorder()
	r.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusMethodNotAllowed {
		t.Errorf("got status %d, expected %d", recorder.Code, http.StatusMethodNotAllowed)
	}
}
