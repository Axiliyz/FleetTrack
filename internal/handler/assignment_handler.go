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

// AssignmentHandler обрабатывает HTTP запросы, связанные со связками устройство-автомобиль
type AssignmentHandler struct {
	assignmentService AssignmentService
	logger            logger.Logger
}

// AssignmentService определяет контракт бизнес-логики, необходимой AssignmentHandler
type AssignmentService interface {
	AssignDevice(ctx context.Context, deviceID, vehicleID int, organizationID int) error
	GetActiveAssignment(ctx context.Context, deviceID int) model.DeviceAssignment
}

// NewAssignmentHandler создаёт новый AssignmentHandler с переданными сервисом и логгером
func NewAssignmentHandler(as AssignmentService, l logger.Logger) *AssignmentHandler {
	return &AssignmentHandler{
		assignmentService: as,
		logger:            l,
	}
}

// HandlePostAssignment обрабатывает POST запрос на создание связки устройство-автомобиль
// @Summary Привязать трекер к автомобилю
// @Tags assignments
// @Accept json
// @Produce json
// @Param request body dto.CreateAssignmentRequest true "ID устройства и автомобиля"
// @Success 201 {object} dto.AssignmentResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 403 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse "устройство или автомобиль не найдены либо принадлежат другой организации"
// @Failure 409 {object} dto.ErrorResponse "устройство или автомобиль уже заняты"
// @Router /assignments [post]
// @Security BearerAuth
func (h *AssignmentHandler) HandlePostAssignment(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	var assignmentData dto.CreateAssignmentRequest

	err := json.NewDecoder(r.Body).Decode(&assignmentData)
	if err != nil {
		respondError(w, r, h.logger, model.ErrInvalidJSON)
		return
	}

	authCtx, ok := middleware.AuthFromContext(r.Context())
	if !ok {
		respondError(w, r, h.logger, model.ErrMissingToken)
		return
	}

	if err := h.assignmentService.AssignDevice(r.Context(), assignmentData.DeviceID, assignmentData.VehicleID, authCtx.OrganizationID); err != nil {
		respondError(w, r, h.logger, err)
		return
	}

	assignment := h.assignmentService.GetActiveAssignment(r.Context(), assignmentData.DeviceID)
	respondCreated(w, r, "device assigned", h.logger, dto.NewAssignmentResponse(assignment))
}
