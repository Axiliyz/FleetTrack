package middleware

import (
	"encoding/json"
	"errors"
	"fleettrack/internal/handler/dto"
	"fleettrack/internal/logger"
	"fleettrack/internal/model"
	"net/http"
	"net/http/httptest"
	"testing"
)

type fakeJWTParser struct {
	claims *model.JWTClaims
	err    error
}

func (f *fakeJWTParser) Parse(tokenString string) (*model.JWTClaims, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.claims, nil
}

func TestAuthMiddleware_MissingHeader(t *testing.T) {
	log := logger.NewStdLogger(logger.DebugLevel)
	called := false
	handler := AuthMiddleware(&fakeJWTParser{}, log)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if called {
		t.Error("next handler should not be called")
	}
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("got status %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestAuthMiddleware_MalformedHeader(t *testing.T) {
	log := logger.NewStdLogger(logger.DebugLevel)
	called := false
	handler := AuthMiddleware(&fakeJWTParser{}, log)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Basic xyz")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if called {
		t.Error("next handler should not be called")
	}
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("got status %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestAuthMiddleware_InvalidToken(t *testing.T) {
	log := logger.NewStdLogger(logger.DebugLevel)
	called := false
	handler := AuthMiddleware(&fakeJWTParser{err: errors.New("bad token")}, log)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer invalid")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if called {
		t.Error("next handler should not be called")
	}
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("got status %d, want %d", rec.Code, http.StatusUnauthorized)
	}

	var resp dto.ErrorResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}
	if resp.Status != "error" {
		t.Errorf("got status field %q, want %q", resp.Status, "error")
	}
}

func TestAuthMiddleware_ValidToken(t *testing.T) {
	log := logger.NewStdLogger(logger.DebugLevel)
	claims := &model.JWTClaims{UserID: 42, Role: model.UserRoleAnalytic}

	var gotAuthCtx model.AuthContext
	var gotOK bool
	handler := AuthMiddleware(&fakeJWTParser{claims: claims}, log)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuthCtx, gotOK = AuthFromContext(r.Context())
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if !gotOK {
		t.Fatal("expected AuthContext to be present in request context")
	}
	if gotAuthCtx.UserID != claims.UserID {
		t.Errorf("got UserID %d, want %d", gotAuthCtx.UserID, claims.UserID)
	}
	if gotAuthCtx.Role != claims.Role {
		t.Errorf("got Role %q, want %q", gotAuthCtx.Role, claims.Role)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("got status %d, want %d (handler wrote nothing, default is 200)", rec.Code, http.StatusOK)
	}
}
