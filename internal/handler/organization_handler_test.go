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

type mockOrgService struct {
	returnError error
}

func (m *mockOrgService) CreateOrg(ctx context.Context, o model.Org) (model.Org, error) {
	if m.returnError != nil {
		return model.Org{}, m.returnError
	}
	return model.Org{
		ID:   1,
		Name: o.Name,
	}, nil
}

func TestHandlePostOrg(t *testing.T) {
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
			requestBody:    `{"name": "Acme"}`,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "wrong method",
			method:         "PATCH",
			serviceError:   nil,
			requestBody:    `{"name": "Acme"}`,
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name:           "invalid name",
			method:         "POST",
			serviceError:   model.ErrInvalidOrgName,
			requestBody:    `{"name": ""}`,
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
			service := &mockOrgService{returnError: tt.serviceError}
			logger := logger.NewStdLogger(logger.DebugLevel)
			handler := NewOrgHandler(service, logger)

			r := chi.NewRouter()
			r.Post("/organizations", handler.HandlePostOrg)

			body := strings.NewReader(tt.requestBody)
			request := httptest.NewRequest(tt.method, "/organizations", body)

			recorder := httptest.NewRecorder()
			r.ServeHTTP(recorder, request)
			if recorder.Code != tt.expectedStatus {
				t.Errorf("got status %d, expected %d", recorder.Code, tt.expectedStatus)
			}
		})
	}
}
