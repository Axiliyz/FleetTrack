// Package handler содержит тесты HTTP-хендлеров для управления правилами алертов
package handler

import (
	"context"
	"errors"
	"fleettrack/internal/logger"
	"fleettrack/internal/middleware"
	"fleettrack/internal/model"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

// mockAlertRuleService имитирует AlertRuleService для тестирования HTTP-слоя
type mockAlertRuleService struct {
	createErr   error
	getErr      error
	listErr     error
	updateErr   error
	deleteErr   error
	rule        model.AlertRule
	rules       []model.AlertRule
	lastCreated model.AlertRule
	lastFilter  model.AlertRuleFilter
}

func (m *mockAlertRuleService) CreateRule(ctx context.Context, r model.AlertRule) (model.AlertRule, error) {
	if m.createErr != nil {
		return model.AlertRule{}, m.createErr
	}
	m.lastCreated = r
	r.ID = 10
	return r, nil
}

func (m *mockAlertRuleService) GetRuleByID(ctx context.Context, id int) (model.AlertRule, error) {
	if m.getErr != nil {
		return model.AlertRule{}, m.getErr
	}
	res := m.rule
	res.ID = id
	return res, nil
}

func (m *mockAlertRuleService) GetRulesList(ctx context.Context, filter model.AlertRuleFilter) ([]model.AlertRule, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	m.lastFilter = filter
	return m.rules, nil
}

func (m *mockAlertRuleService) UpdateRule(ctx context.Context, r model.AlertRule) (model.AlertRule, error) {
	if m.updateErr != nil {
		return model.AlertRule{}, m.updateErr
	}
	return r, nil
}

func (m *mockAlertRuleService) DeleteRuleByID(ctx context.Context, id int) (model.AlertRule, error) {
	if m.deleteErr != nil {
		return model.AlertRule{}, m.deleteErr
	}
	return m.rule, nil
}

// TestAlertRuleHandler_HandlePostRule проверяет обработку создания правила через POST /alert-rules
func TestAlertRuleHandler_HandlePostRule(t *testing.T) {
	tests := []struct {
		name           string
		body           string
		authCtx        *model.AuthContext
		serviceErr     error
		expectedStatus int
		checkOrgID     int
	}{
		{
			name:           "успешное создание диспетчером с автоподстановкой своей организации",
			body:           `{"organization_id": 999, "type": "SPEED_EXCEEDED", "name": "Speed", "threshold": 80, "severity": "HIGH"}`,
			authCtx:        &model.AuthContext{UserID: 1, Role: model.UserRoleDispatcher, OrganizationID: 5},
			serviceErr:     nil,
			expectedStatus: http.StatusOK,
			checkOrgID:     5, // non-admin не может переопределить свою организацию
		},
		{
			name:           "успешное создание администратором для указанной организации",
			body:           `{"organization_id": 999, "type": "SPEED_EXCEEDED", "name": "Speed", "threshold": 80, "severity": "HIGH"}`,
			authCtx:        &model.AuthContext{UserID: 1, Role: model.UserRoleAdmin, OrganizationID: 1},
			serviceErr:     nil,
			expectedStatus: http.StatusOK,
			checkOrgID:     999, // admin может создавать для любой организации
		},
		{
			name:           "ошибка отсутствия авторизации",
			body:           `{"type": "SPEED_EXCEEDED", "name": "Speed", "threshold": 80, "severity": "HIGH"}`,
			authCtx:        nil,
			serviceErr:     nil,
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "невалидный JSON в теле запроса",
			body:           `{invalid json`,
			authCtx:        &model.AuthContext{UserID: 1, Role: model.UserRoleDispatcher, OrganizationID: 1},
			serviceErr:     nil,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "ошибка валидации сервиса",
			body:           `{"type": "SPEED_EXCEEDED", "name": "", "threshold": 80, "severity": "HIGH"}`,
			authCtx:        &model.AuthContext{UserID: 1, Role: model.UserRoleDispatcher, OrganizationID: 1},
			serviceErr:     model.ErrInvalidName,
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockAlertRuleService{createErr: tt.serviceErr}
			log := logger.NewStdLogger(logger.DebugLevel)
			h := NewAlertRuleHandler(svc, log)

			r := chi.NewRouter()
			r.Post("/alert-rules", h.HandlePostRule)

			req := httptest.NewRequest("POST", "/alert-rules", strings.NewReader(tt.body))
			if tt.authCtx != nil {
				req = req.WithContext(middleware.ContextWithAuth(req.Context(), *tt.authCtx))
			}

			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("got status %d, want %d", rec.Code, tt.expectedStatus)
			}

			if tt.checkOrgID != 0 && svc.lastCreated.OrganizationID != tt.checkOrgID {
				t.Errorf("OrganizationID = %d, want %d", svc.lastCreated.OrganizationID, tt.checkOrgID)
			}
		})
	}
}

// TestAlertRuleHandler_HandleGetRuleByID проверяет получение правила по ID с сокрытием чужих данных
func TestAlertRuleHandler_HandleGetRuleByID(t *testing.T) {
	tests := []struct {
		name           string
		urlID          string
		authCtx        *model.AuthContext
		ruleOrgID      int
		serviceErr     error
		expectedStatus int
	}{
		{
			name:           "успешное получение правила своей организации",
			urlID:          "10",
			authCtx:        &model.AuthContext{UserID: 1, Role: model.UserRoleDispatcher, OrganizationID: 2},
			ruleOrgID:      2,
			serviceErr:     nil,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "правило чужой организации возвращает 404 чтобы скрыть существование",
			urlID:          "10",
			authCtx:        &model.AuthContext{UserID: 1, Role: model.UserRoleDispatcher, OrganizationID: 2},
			ruleOrgID:      3,
			serviceErr:     nil,
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "нечисловой идентификатор в URL",
			urlID:          "invalid-id",
			authCtx:        &model.AuthContext{UserID: 1, Role: model.UserRoleDispatcher, OrganizationID: 1},
			ruleOrgID:      1,
			serviceErr:     nil,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "запрос без токена авторизации",
			urlID:          "10",
			authCtx:        nil,
			ruleOrgID:      1,
			serviceErr:     nil,
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "правило не найдено в базе данных",
			urlID:          "99",
			authCtx:        &model.AuthContext{UserID: 1, Role: model.UserRoleDispatcher, OrganizationID: 1},
			ruleOrgID:      1,
			serviceErr:     model.ErrNotFound,
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockAlertRuleService{
				getErr: tt.serviceErr,
				rule: model.AlertRule{
					ID:             10,
					OrganizationID: tt.ruleOrgID,
					Name:           "Speeding",
					Type:           model.AlertRuleSpeedExceed,
					Threshold:      90,
					Severity:       model.SeverityLevelHigh,
				},
			}
			log := logger.NewStdLogger(logger.DebugLevel)
			h := NewAlertRuleHandler(svc, log)

			r := chi.NewRouter()
			r.Get("/alert-rules/{id}", h.HandleGetRuleByID)

			req := httptest.NewRequest("GET", "/alert-rules/"+tt.urlID, nil)
			if tt.authCtx != nil {
				req = req.WithContext(middleware.ContextWithAuth(req.Context(), *tt.authCtx))
			}

			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("got status %d, want %d", rec.Code, tt.expectedStatus)
			}
		})
	}
}

// TestAlertRuleHandler_HandleGetRulesList проверяет получение списка правил с изоляцией по организации
func TestAlertRuleHandler_HandleGetRulesList(t *testing.T) {
	tests := []struct {
		name           string
		authCtx        *model.AuthContext
		serviceErr     error
		expectedStatus int
		wantFilterOrg  *int
	}{
		{
			name:           "список для диспетчера ограничен его организацией",
			authCtx:        &model.AuthContext{UserID: 1, Role: model.UserRoleDispatcher, OrganizationID: 4},
			serviceErr:     nil,
			expectedStatus: http.StatusOK,
			wantFilterOrg:  intPtr(4),
		},
		{
			name:           "список для администратора также ограничен его организацией",
			authCtx:        &model.AuthContext{UserID: 1, Role: model.UserRoleAdmin, OrganizationID: 1},
			serviceErr:     nil,
			expectedStatus: http.StatusOK,
			wantFilterOrg:  intPtr(1),
		},
		{
			name:           "запрос без токена возвращает 401",
			authCtx:        nil,
			serviceErr:     nil,
			expectedStatus: http.StatusUnauthorized,
			wantFilterOrg:  nil,
		},
		{
			name:           "внутренняя ошибка сервиса",
			authCtx:        &model.AuthContext{UserID: 1, Role: model.UserRoleDispatcher, OrganizationID: 1},
			serviceErr:     errors.New("db error"),
			expectedStatus: http.StatusInternalServerError,
			wantFilterOrg:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockAlertRuleService{
				listErr: tt.serviceErr,
				rules:   []model.AlertRule{{ID: 1, OrganizationID: 4}},
			}
			log := logger.NewStdLogger(logger.DebugLevel)
			h := NewAlertRuleHandler(svc, log)

			r := chi.NewRouter()
			r.Get("/alert-rules", h.HandleGetRulesList)

			req := httptest.NewRequest("GET", "/alert-rules", nil)
			if tt.authCtx != nil {
				req = req.WithContext(middleware.ContextWithAuth(req.Context(), *tt.authCtx))
			}

			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("got status %d, want %d", rec.Code, tt.expectedStatus)
			}

			if tt.wantFilterOrg != nil {
				if svc.lastFilter.OrganizationID == nil || *svc.lastFilter.OrganizationID != *tt.wantFilterOrg {
					t.Errorf("filter OrganizationID = %v, want %v", svc.lastFilter.OrganizationID, *tt.wantFilterOrg)
				}
			}
		})
	}
}

// TestAlertRuleHandler_HandleDeleteRuleByID проверяет удаление правила с авторизацией и проверкой организации
func TestAlertRuleHandler_HandleDeleteRuleByID(t *testing.T) {
	tests := []struct {
		name           string
		urlID          string
		authCtx        *model.AuthContext
		ruleOrgID      int
		serviceGetErr  error
		serviceDelErr  error
		expectedStatus int
	}{
		{
			name:           "успешное удаление своего правила",
			urlID:          "15",
			authCtx:        &model.AuthContext{UserID: 1, Role: model.UserRoleDispatcher, OrganizationID: 2},
			ruleOrgID:      2,
			serviceGetErr:  nil,
			serviceDelErr:  nil,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "попытка удаления правила чужой организации возвращает 404",
			urlID:          "15",
			authCtx:        &model.AuthContext{UserID: 1, Role: model.UserRoleDispatcher, OrganizationID: 2},
			ruleOrgID:      3,
			serviceGetErr:  nil,
			serviceDelErr:  nil,
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "правило для удаления не найдено",
			urlID:          "99",
			authCtx:        &model.AuthContext{UserID: 1, Role: model.UserRoleDispatcher, OrganizationID: 2},
			ruleOrgID:      2,
			serviceGetErr:  model.ErrNotFound,
			serviceDelErr:  nil,
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "невалидный ID в строке запроса",
			urlID:          "bad",
			authCtx:        &model.AuthContext{UserID: 1, Role: model.UserRoleDispatcher, OrganizationID: 2},
			ruleOrgID:      2,
			serviceGetErr:  nil,
			serviceDelErr:  nil,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "отсутствует авторизация при удалении",
			urlID:          "15",
			authCtx:        nil,
			ruleOrgID:      2,
			serviceGetErr:  nil,
			serviceDelErr:  nil,
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockAlertRuleService{
				getErr:    tt.serviceGetErr,
				deleteErr: tt.serviceDelErr,
				rule: model.AlertRule{
					ID:             15,
					OrganizationID: tt.ruleOrgID,
				},
			}
			log := logger.NewStdLogger(logger.DebugLevel)
			h := NewAlertRuleHandler(svc, log)

			r := chi.NewRouter()
			r.Delete("/alert-rules/{id}", h.HandleDeleteRuleByID)

			req := httptest.NewRequest("DELETE", "/alert-rules/"+tt.urlID, nil)
			if tt.authCtx != nil {
				req = req.WithContext(middleware.ContextWithAuth(req.Context(), *tt.authCtx))
			}

			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("got status %d, want %d", rec.Code, tt.expectedStatus)
			}
		})
	}
}

func intPtr(i int) *int {
	return &i
}
