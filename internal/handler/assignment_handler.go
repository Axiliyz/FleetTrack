package handler

import (
	"context"
	"encoding/json"
	"fleettrack/internal/handler/dto"
	"fleettrack/internal/logger"
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
	AssignDevice(ctx context.Context, deviceID, vehicleID int) error
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
func (h *AssignmentHandler) HandlePostAssignment(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	var assignmentData dto.CreateAssignmentRequest

	err := json.NewDecoder(r.Body).Decode(&assignmentData)
	if err != nil {
		respondError(w, r, h.logger, model.ErrInvalidJSON)
		return
	}

	if err := h.assignmentService.AssignDevice(r.Context(), assignmentData.DeviceID, assignmentData.VehicleID); err != nil {
		respondError(w, r, h.logger, err)
		return
	}

	assignment := h.assignmentService.GetActiveAssignment(r.Context(), assignmentData.DeviceID)
	respondSuccess(w, r, "device assigned", h.logger, dto.NewAssignmentResponse(assignment))
}
