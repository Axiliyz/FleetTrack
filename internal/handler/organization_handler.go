package handler

import (
	"context"
	"encoding/json"
	"fleettrack/internal/handler/dto"
	"fleettrack/internal/logger"
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
	// GetOrgList возвращает список всех организаций
	GetOrgList(ctx context.Context) ([]model.Org, error)
}

// NewOrgHandler создаёт новый хендлер организаций
func NewOrgHandler(s OrgService, l logger.Logger) *OrgHandler {
	return &OrgHandler{
		orgService: s,
		logger:     l,
	}
}

// HandlePostOrg обрабатывает создание новой организации
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

	respondSuccess(w, r, "organization created", h.logger, dto.NewOrgResponse(org))
}

// HandleGetListOrg возвращает список всех организаций
func (h *OrgHandler) HandleGetListOrg(w http.ResponseWriter, r *http.Request) {
	orgs, err := h.orgService.GetOrgList(r.Context())
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
