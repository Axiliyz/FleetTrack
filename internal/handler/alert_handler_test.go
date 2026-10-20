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

// mockAlertService имитирует AlertService для тестирования HTTP-слоя
type mockAlertService struct {
	ackErr     error
	listErr    error
	alerts     []model.Alert
	lastActor  model.AuthContext
	lastID     int
	lastFilter model.AlertFilter
}

func (m *mockAlertService) GetAlertsList(ctx context.Context, filter model.AlertFilter) ([]model.Alert, error) {
	m.lastFilter = filter
	if m.listErr != nil {
		return nil, m.listErr
	}
	return m.alerts, nil
}

func (m *mockAlertService) AcknowledgeAlert(ctx context.Context, alertID int, actor model.AuthContext) (model.Alert, error) {
	m.lastID = alertID
	m.lastActor = actor
	if m.ackErr != nil {
		return model.Alert{}, m.ackErr
	}
	return model.Alert{ID: alertID, OrganizationID: actor.OrganizationID, Status: model.AlertStatusAcknowledged}, nil
}

// TestAlertHandler_HandlePatchAlert проверяет перевод алерта в ACKNOWLEDGED через PATCH /alerts/{id}
func TestAlertHandler_HandlePatchAlert(t *testing.T) {
	dispatcher := &model.AuthContext{UserID: 7, OrganizationID: 5, Role: model.UserRoleDispatcher}

	tests := []struct {
		name           string
		path           string
		body           string
		authCtx        *model.AuthContext
		serviceErr     error
		expectedStatus int
	}{
		{name: "диспетчер принимает алерт", path: "/alerts/42", body: `{"status":"ACKNOWLEDGED"}`, authCtx: dispatcher, expectedStatus: http.StatusOK},
		{name: "нет токена", path: "/alerts/42", body: `{"status":"ACKNOWLEDGED"}`, expectedStatus: http.StatusUnauthorized},
		{name: "нечисловой id", path: "/alerts/abc", body: `{"status":"ACKNOWLEDGED"}`, authCtx: dispatcher, expectedStatus: http.StatusBadRequest},
		{name: "отрицательный id", path: "/alerts/-1", body: `{"status":"ACKNOWLEDGED"}`, authCtx: dispatcher, expectedStatus: http.StatusBadRequest},
		{name: "невалидный JSON", path: "/alerts/42", body: `{oops`, authCtx: dispatcher, expectedStatus: http.StatusBadRequest},
		{name: "закрывать вручную нельзя", path: "/alerts/42", body: `{"status":"RESOLVED"}`, authCtx: dispatcher, expectedStatus: http.StatusBadRequest},
		{name: "алерта нет или он чужой", path: "/alerts/42", body: `{"status":"ACKNOWLEDGED"}`, authCtx: dispatcher, serviceErr: model.ErrNotFound, expectedStatus: http.StatusNotFound},
		{name: "уже принят соседом", path: "/alerts/42", body: `{"status":"ACKNOWLEDGED"}`, authCtx: dispatcher, serviceErr: model.ErrAlertNotFired, expectedStatus: http.StatusConflict},
		{name: "роль без прав", path: "/alerts/42", body: `{"status":"ACKNOWLEDGED"}`, authCtx: dispatcher, serviceErr: model.ErrForbidden, expectedStatus: http.StatusForbidden},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockAlertService{ackErr: tt.serviceErr}
			h := NewAlertHandler(svc, logger.NewStdLogger(logger.DebugLevel))

			r := chi.NewRouter()
			r.Patch("/alerts/{id}", h.HandlePatchAlert)

			req := httptest.NewRequest(http.MethodPatch, tt.path, strings.NewReader(tt.body))
			if tt.authCtx != nil {
				req = req.WithContext(middleware.ContextWithAuth(req.Context(), *tt.authCtx))
			}
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Fatalf("got status %d, want %d, body: %s", rec.Code, tt.expectedStatus, rec.Body.String())
			}
			if tt.expectedStatus == http.StatusOK {
				if svc.lastID != 42 || svc.lastActor != *tt.authCtx {
					t.Errorf("service got id=%d actor=%+v", svc.lastID, svc.lastActor)
				}
			}
		})
	}
}

// TestAlertHandler_HandleGetAlertsList проверяет фильтры и изоляцию организации в GET /alerts
func TestAlertHandler_HandleGetAlertsList(t *testing.T) {
	analytic := &model.AuthContext{UserID: 3, OrganizationID: 5, Role: model.UserRoleAnalytic}

	tests := []struct {
		name           string
		query          string
		authCtx        *model.AuthContext
		expectedStatus int
	}{
		{name: "список своей организации", query: "?status=FIRED&severity=HIGH&vehicle_id=3&limit=20", authCtx: analytic, expectedStatus: http.StatusOK},
		{name: "чужая организация в query игнорируется", query: "?organization_id=999", authCtx: analytic, expectedStatus: http.StatusOK},
		{name: "нет токена", expectedStatus: http.StatusUnauthorized},
		{name: "неизвестный статус", query: "?status=OPEN", authCtx: analytic, expectedStatus: http.StatusBadRequest},
		{name: "неизвестная критичность", query: "?severity=URGENT", authCtx: analytic, expectedStatus: http.StatusBadRequest},
		{name: "кривой vehicle_id", query: "?vehicle_id=x", authCtx: analytic, expectedStatus: http.StatusBadRequest},
		{name: "нулевой limit", query: "?limit=0", authCtx: analytic, expectedStatus: http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockAlertService{}
			h := NewAlertHandler(svc, logger.NewStdLogger(logger.DebugLevel))

			r := chi.NewRouter()
			r.Get("/alerts", h.HandleGetAlertsList)

			req := httptest.NewRequest(http.MethodGet, "/alerts"+tt.query, nil)
			if tt.authCtx != nil {
				req = req.WithContext(middleware.ContextWithAuth(req.Context(), *tt.authCtx))
			}
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Fatalf("got status %d, want %d, body: %s", rec.Code, tt.expectedStatus, rec.Body.String())
			}
			if tt.expectedStatus != http.StatusOK {
				return
			}
			if svc.lastFilter.OrganizationID == nil || *svc.lastFilter.OrganizationID != tt.authCtx.OrganizationID {
				t.Errorf("OrganizationID = %v, want %d", svc.lastFilter.OrganizationID, tt.authCtx.OrganizationID)
			}
			if svc.lastFilter.Limit == 0 {
				t.Error("limit must always be set, otherwise the list is unbounded")
			}
		})
	}
}

// TestAlertHandler_EmptyListIsArray проверяет, что пустой список уходит как [], а не null — фронт итерирует data
func TestAlertHandler_EmptyListIsArray(t *testing.T) {
	h := NewAlertHandler(&mockAlertService{}, logger.NewStdLogger(logger.DebugLevel))
	req := httptest.NewRequest(http.MethodGet, "/alerts", nil)
	req = req.WithContext(middleware.ContextWithAuth(req.Context(), model.AuthContext{UserID: 1, OrganizationID: 1, Role: model.UserRoleAdmin}))
	rec := httptest.NewRecorder()
	h.HandleGetAlertsList(rec, req)

	var body struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if string(body.Data) != "[]" {
		t.Errorf("data = %s, want []", body.Data)
	}
}
