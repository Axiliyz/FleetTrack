package handler

import (
	"context"
	"fleettrack/internal/handler/dto"
	"fleettrack/internal/logger"
	"fleettrack/internal/middleware"
	"fleettrack/internal/model"
	"net/http"
)

// FleetHandler отдаёт сводное состояние парка для карты диспетчера
type FleetHandler struct {
	fleetService FleetService
	logger       logger.Logger
}

// FleetService определяет контракт бизнес-логики для FleetHandler
type FleetService interface {
	Snapshot(ctx context.Context, actor model.AuthContext) (model.FleetSnapshot, error)
}

// NewFleetHandler создаёт новый экземпляр хендлера парка
func NewFleetHandler(fs FleetService, l logger.Logger) *FleetHandler {
	return &FleetHandler{fleetService: fs, logger: l}
}

// HandleGetSnapshot возвращает снимок парка организации текущего пользователя
// @Summary Снимок парка для карты
// @Description Машины, водители, рейсы смены, последняя точка по каждой машине, алерты смены и правила одним ответом.
// @Description Окно смены — последние 24 часа; рейсы в пути и незакрытые алерты попадают в ответ независимо от возраста.
// @Tags fleet
// @Produce json
// @Success 200 {object} dto.APIResponse{data=dto.FleetSnapshotResponse}
// @Failure 401 {object} dto.ErrorResponse
// @Failure 403 {object} dto.ErrorResponse
// @Router /fleet/snapshot [get]
// @Security BearerAuth
func (h *FleetHandler) HandleGetSnapshot(w http.ResponseWriter, r *http.Request) {
	authCtx, ok := middleware.AuthFromContext(r.Context())
	if !ok {
		respondError(w, r, h.logger, model.ErrMissingToken)
		return
	}

	snap, err := h.fleetService.Snapshot(r.Context(), authCtx)
	if err != nil {
		respondError(w, r, h.logger, err)
		return
	}

	respondSuccess(w, r, "fleet snapshot", h.logger, dto.NewFleetSnapshotResponse(snap))
}
