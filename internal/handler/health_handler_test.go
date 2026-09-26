package handler

import (
	"context"
	"errors"
	"fleettrack/internal/logger"
	"fleettrack/internal/model"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type mockHealthService struct {
	healthErr    error
	readinessErr error
}

func (m *mockHealthService) CheckHealth(ctx context.Context) error {
	return m.healthErr
}

func (m *mockHealthService) CheckReadiness(ctx context.Context) error {
	return m.readinessErr
}

func TestHealthHandler(t *testing.T) {
	notReady := fmt.Errorf("%w: ping postgres: %w", model.ErrServiceUnavailable, errors.New("connection refused"))

	tests := []struct {
		name           string
		service        *mockHealthService
		path           string
		expectedStatus int
	}{
		{
			name:           "health ok",
			service:        &mockHealthService{},
			path:           "/health",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "health failed",
			service:        &mockHealthService{healthErr: model.ErrServiceUnavailable},
			path:           "/health",
			expectedStatus: http.StatusServiceUnavailable,
		},
		{
			name:           "readiness ok",
			service:        &mockHealthService{},
			path:           "/readyz",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "readiness failed",
			service:        &mockHealthService{readinessErr: notReady},
			path:           "/readyz",
			expectedStatus: http.StatusServiceUnavailable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewHealthHandler(tt.service, logger.NewStdLogger(logger.DebugLevel))

			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			rec := httptest.NewRecorder()

			if tt.path == "/health" {
				h.HandleHealthCheck(rec, req)
			} else {
				h.HandleReadinessCheck(rec, req)
			}

			if rec.Code != tt.expectedStatus {
				t.Fatalf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}
			if strings.Contains(rec.Body.String(), "ping postgres") {
				t.Errorf("response leaks internal error: %s", rec.Body.String())
			}
		})
	}
}
