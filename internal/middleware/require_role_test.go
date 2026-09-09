package middleware

import (
	"context"
	"encoding/json"
	"fleettrack/internal/handler/dto"
	"fleettrack/internal/logger"
	"fleettrack/internal/model"
	"net/http"
	"net/http/httptest"
	"testing"
)

func withAuthContext(r *http.Request, authCtx model.AuthContext) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), authContextKey, authCtx))
}

func TestRequireRole_Allowed(t *testing.T) {
	log := logger.NewStdLogger(logger.DebugLevel)
	called := false
	handler := RequireRole(log, model.UserRoleAdmin, model.UserRoleDispatcher)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))

	req := withAuthContext(httptest.NewRequest(http.MethodPost, "/", nil), model.AuthContext{UserID: 1, Role: model.UserRoleAdmin})
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if !called {
		t.Error("expected next handler to be called")
	}
	if rec.Code != http.StatusOK {
		t.Errorf("got status %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestRequireRole_Forbidden(t *testing.T) {
	log := logger.NewStdLogger(logger.DebugLevel)
	called := false
	handler := RequireRole(log, model.UserRoleAdmin)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))

	req := withAuthContext(httptest.NewRequest(http.MethodPost, "/", nil), model.AuthContext{UserID: 1, Role: model.UserRoleDriver})
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if called {
		t.Error("next handler should not be called")
	}
	if rec.Code != http.StatusForbidden {
		t.Errorf("got status %d, want %d", rec.Code, http.StatusForbidden)
	}

	var resp dto.ErrorResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}
	if resp.Message != "forbidden" {
		t.Errorf("got message %q, want %q", resp.Message, "forbidden")
	}
}

func TestRequireRole_NoAuthContext(t *testing.T) {
	log := logger.NewStdLogger(logger.DebugLevel)
	called := false
	handler := RequireRole(log, model.UserRoleAdmin)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if called {
		t.Error("next handler should not be called")
	}
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("got status %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}
