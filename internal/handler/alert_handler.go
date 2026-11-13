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

// AlertHandler обрабатывает HTTP-запросы к сработавшим алертам
type AlertHandler struct {
	alertService AlertService
	logger       logger.Logger
}

// AlertService определяет контракт бизнес-логики для AlertHandler
type AlertService interface {
	GetAlertsList(ctx context.Context, filter model.AlertFilter) ([]model.Alert, error)
	AcknowledgeAlert(ctx context.Context, alertID int, actor model.AuthContext) (model.Alert, error)
}

// NewAlertHandler создаёт новый экземпляр хендлера алертов
func NewAlertHandler(as AlertService, l logger.Logger) *AlertHandler {
	return &AlertHandler{
		alertService: as,
		logger:       l,
	}
}

// HandleGetAlertsList возвращает алерты организации текущего пользователя, новые первыми
// @Summary Список алертов организации
// @Tags alerts
// @Produce json
// @Param status query string false "Статус алерта" Enums(FIRED, ACKNOWLEDGED, RESOLVED)
// @Param severity query string false "Критичность" Enums(LOW, MEDIUM, HIGH, CRITICAL)
// @Param vehicle_id query int false "ID машины"
// @Param created_from query string false "Алерты начиная с момента (RFC3339)" format(date-time)
// @Param limit query int false "Максимум записей" default(100) minimum(1) maximum(500)
// @Param offset query int false "Сколько записей пропустить" default(0) minimum(0)
// @Success 200 {object} dto.APIResponse{data=[]dto.AlertResponse}
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Router /alerts [get]
// @Security BearerAuth
func (h *AlertHandler) HandleGetAlertsList(w http.ResponseWriter, r *http.Request) {
	authCtx, ok := middleware.AuthFromContext(r.Context())
	if !ok {
		respondError(w, r, h.logger, model.ErrMissingToken)
		return
	}

	filter, err := dto.ParseAlertFilter(r.URL.Query())
	if err != nil {
		respondError(w, r, h.logger, err)
		return
	}
	filter.OrganizationID = scopeOrganizationID(authCtx)

	alerts, err := h.alertService.GetAlertsList(r.Context(), filter)
	if err != nil {
		respondError(w, r, h.logger, err)
		return
	}

	respondSuccess(w, r, "got alerts", h.logger, dto.NewAlertListResponse(alerts))
}

// HandlePatchAlert переводит алерт в ACKNOWLEDGED от имени текущего пользователя
// @Summary Принять алерт в работу
// @Tags alerts
// @Accept json
// @Produce json
// @Param id path int true "ID алерта"
// @Param request body dto.UpdateAlertRequest true "Новый статус, поддерживается только ACKNOWLEDGED"
// @Success 200 {object} dto.APIResponse{data=dto.AlertResponse}
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 403 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse "алерта нет или он принадлежит другой организации"
// @Failure 409 {object} dto.ErrorResponse "алерт уже принят или закрыт"
// @Router /alerts/{id} [patch]
// @Security BearerAuth
func (h *AlertHandler) HandlePatchAlert(w http.ResponseWriter, r *http.Request) {
	authCtx, ok := middleware.AuthFromContext(r.Context())
	if !ok {
		respondError(w, r, h.logger, model.ErrMissingToken)
		return
	}

	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil || id <= 0 {
		respondError(w, r, h.logger, model.ErrInvalidAlertID)
		return
	}

	var req dto.UpdateAlertRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, r, h.logger, model.ErrInvalidJSON)
		return
	}
	if err := req.Validate(); err != nil {
		respondError(w, r, h.logger, err)
		return
	}

	alert, err := h.alertService.AcknowledgeAlert(r.Context(), id, authCtx)
	if err != nil {
		respondError(w, r, h.logger, err)
		return
	}

	respondSuccess(w, r, "alert acknowledged", h.logger, dto.NewAlertResponse(alert))
}
