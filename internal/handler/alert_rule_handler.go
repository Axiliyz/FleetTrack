package handler

import (
	"context"
	"encoding/json"
	"fleettrack/internal/handler/dto"
	"fleettrack/internal/logger"
	"fleettrack/internal/middleware"
	"fleettrack/internal/model"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

// AlertRuleHandler обрабатывает HTTP-запросы управления правилами алертов
type AlertRuleHandler struct {
	alertRuleService AlertRuleService
	logger           logger.Logger
}

// AlertRuleService определяет контракт бизнес-логики для AlertRuleHandler
type AlertRuleService interface {
	CreateRule(ctx context.Context, r model.AlertRule) (model.AlertRule, error)
	GetRuleByID(ctx context.Context, id int) (model.AlertRule, error)
	GetRulesList(ctx context.Context, filter model.AlertRuleFilter) ([]model.AlertRule, error)
	UpdateRule(ctx context.Context, r model.AlertRule) (model.AlertRule, error)
	DeleteRuleByID(ctx context.Context, id int) (model.AlertRule, error)
}

// NewAlertRuleHandler создаёт новый экземпляр хендлера правил алертов
func NewAlertRuleHandler(ars AlertRuleService, l logger.Logger) *AlertRuleHandler {
	return &AlertRuleHandler{
		alertRuleService: ars,
		logger:           l,
	}
}

// HandlePostRule обрабатывает создание нового правила алертов
// @Summary Создать правило алерта
// @Tags alert-rules
// @Accept json
// @Produce json
// @Param request body dto.CreateAlertRuleRequest true "Данные правила"
// @Success 201 {object} dto.AlertRuleResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 403 {object} dto.ErrorResponse
// @Router /alert-rules [post]
// @Security BearerAuth
func (h *AlertRuleHandler) HandlePostRule(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	var ruleData dto.CreateAlertRuleRequest
	if err := json.NewDecoder(r.Body).Decode(&ruleData); err != nil {
		respondError(w, r, h.logger, model.ErrInvalidJSON)
		return
	}

	authCtx, ok := middleware.AuthFromContext(r.Context())
	if !ok {
		respondError(w, r, h.logger, model.ErrMissingToken)
		return
	}
	if authCtx.Role != model.UserRoleAdmin {
		ruleData.OrganizationID = authCtx.OrganizationID
	}
	newRule := ruleData.ToDomainModel()
	rule, err := h.alertRuleService.CreateRule(r.Context(), newRule)
	if err != nil {
		respondError(w, r, h.logger, err)
		return
	}

	var response = dto.NewAlertRuleResponse(rule)

	respondCreated(w, r, "rule created", h.logger, response)
}

// HandleGetRuleByID возвращает правило алертов по его идентификатору
// @Summary Получить правило алерта по ID
// @Tags alert-rules
// @Produce json
// @Param id path int true "ID правила"
// @Success 200 {object} dto.AlertRuleResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Router /alert-rules/{id} [get]
// @Security BearerAuth
func (h *AlertRuleHandler) HandleGetRuleByID(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		respondError(w, r, h.logger, model.ErrInvalidRuleID)
		return
	}

	authCtx, ok := middleware.AuthFromContext(r.Context())
	if !ok {
		respondError(w, r, h.logger, model.ErrMissingToken)
		return
	}

	rule, err := h.alertRuleService.GetRuleByID(r.Context(), id)
	if err != nil {
		respondError(w, r, h.logger, err)
		return
	}

	if err := requireOwnOrg(authCtx, rule.OrganizationID); err != nil {
		respondError(w, r, h.logger, err)
		return
	}

	respondSuccess(w, r, "rule found", h.logger, dto.NewAlertRuleResponse(rule))
}

// HandleDeleteRuleByID удаляет правило алертов по его идентификатору с проверкой прав доступа
// @Summary Удалить правило алерта
// @Tags alert-rules
// @Produce json
// @Param id path int true "ID правила"
// @Success 200 {object} dto.AlertRuleResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 403 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Router /alert-rules/{id} [delete]
// @Security BearerAuth
func (h *AlertRuleHandler) HandleDeleteRuleByID(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		respondError(w, r, h.logger, model.ErrInvalidRuleID)
		return
	}

	authCtx, ok := middleware.AuthFromContext(r.Context())
	if !ok {
		respondError(w, r, h.logger, model.ErrMissingToken)
		return
	}

	rule, err := h.alertRuleService.GetRuleByID(r.Context(), id)
	if err != nil {
		respondError(w, r, h.logger, err)
		return
	}

	if err := requireOwnOrg(authCtx, rule.OrganizationID); err != nil {
		respondError(w, r, h.logger, err)
		return
	}

	deletedRule, err := h.alertRuleService.DeleteRuleByID(r.Context(), id)
	if err != nil {
		respondError(w, r, h.logger, err)
		return
	}

	respondSuccess(w, r, "rule deleted", h.logger, dto.NewAlertRuleResponse(deletedRule))
}

// HandleGetRulesList возвращает список правил алертов организации текущего пользователя
// @Summary Список правил алертов организации
// @Tags alert-rules
// @Produce json
// @Success 200 {array} dto.AlertRuleResponse
// @Failure 401 {object} dto.ErrorResponse
// @Router /alert-rules [get]
// @Security BearerAuth
func (h *AlertRuleHandler) HandleGetRulesList(w http.ResponseWriter, r *http.Request) {
	authCtx, ok := middleware.AuthFromContext(r.Context())
	if !ok {
		respondError(w, r, h.logger, model.ErrMissingToken)
		return
	}

	filter := model.AlertRuleFilter{
		OrganizationID: scopeOrganizationID(authCtx),
	}

	rules, err := h.alertRuleService.GetRulesList(r.Context(), filter)
	if err != nil {
		respondError(w, r, h.logger, err)
		return
	}

	respondSuccess(w, r, "got rules", h.logger, dto.NewAlertRuleListResponse(rules))
}
