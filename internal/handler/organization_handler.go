package handler

import (
	"context"
	"encoding/json"
	"fleettrack/internal/handler/dto"
	"fleettrack/internal/logger"
	"fleettrack/internal/middleware"
	"fleettrack/internal/model"
	"net/http"
)

// OrgHandler обрабатывает HTTP запросы, связанные с организациями
type OrgHandler struct {
	orgService OrgService
	logger     logger.Logger
}

// OrgService описывает бизнес-логику организаций, необходимую хендлеру
type OrgService interface {
	// CreateOrg валидирует и сохраняет новую организацию
	CreateOrg(ctx context.Context, o model.Org) (model.Org, error)
	// GetOrgList возвращает организацию с данным ID в виде списка из одного элемента
	GetOrgList(ctx context.Context, organizationID int) ([]model.Org, error)
}

// NewOrgHandler создаёт новый хендлер организаций
func NewOrgHandler(s OrgService, l logger.Logger) *OrgHandler {
	return &OrgHandler{
		orgService: s,
		logger:     l,
	}
}

// HandlePostOrg обрабатывает создание новой организации
// @Summary Создать организацию
// @Tags organization
// @Accept json
// @Produce json
// @Param request body dto.OrgRequest true "Название организации"
// @Success 201 {object} dto.OrgResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 403 {object} dto.ErrorResponse
// @Failure 409 {object} dto.ErrorResponse
// @Router /organizations [post]
// @Security BearerAuth
func (h *OrgHandler) HandlePostOrg(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	var orgData dto.OrgRequest
	err := json.NewDecoder(r.Body).Decode(&orgData)
	if err != nil {
		respondError(w, r, h.logger, model.ErrInvalidJSON)
		return
	}

	org, err := h.orgService.CreateOrg(r.Context(), orgData.ToDomainModel())
	if err != nil {
		respondError(w, r, h.logger, err)
		return
	}

	respondCreated(w, r, "organization created", h.logger, dto.NewOrgResponse(org))
}

// HandleGetListOrg возвращает список всех организаций
// @Summary Получить свою организацию
// @Description Возвращает организацию текущего пользователя в виде списка из одного элемента
// @Tags organization
// @Produce json
// @Success 200 {array} dto.OrgResponse
// @Failure 401 {object} dto.ErrorResponse
// @Router /organizations [get]
// @Security BearerAuth
func (h *OrgHandler) HandleGetListOrg(w http.ResponseWriter, r *http.Request) {
	authCtx, ok := middleware.AuthFromContext(r.Context())
	if !ok {
		respondError(w, r, h.logger, model.ErrMissingToken)
		return
	}

	orgs, err := h.orgService.GetOrgList(r.Context(), authCtx.OrganizationID)
	if err != nil {
		respondError(w, r, h.logger, err)
		return
	}

	responses := make([]dto.OrgResponse, 0, len(orgs))
	for _, o := range orgs {
		responses = append(responses, dto.NewOrgResponse(o))
	}

	respondSuccess(w, r, "organizations list", h.logger, responses)
}
