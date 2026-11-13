package handler

import (
	"context"
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

// HandleGetListOrg возвращает организацию текущего пользователя
// @Summary Получить свою организацию
// @Description Возвращает организацию текущего пользователя в виде списка из одного элемента
// @Tags organization
// @Produce json
// @Success 200 {object} dto.APIResponse{data=[]dto.OrgResponse}
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
