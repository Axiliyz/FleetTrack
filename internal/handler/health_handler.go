package handler

import (
	"context"
	"fleettrack/internal/logger"
	"net/http"
)

// HealthHandler обрабатывает HTTP запросы, связанные со здоровьем приложения
type HealthHandler struct {
	healthService HealthService
	logger        logger.Logger
}

// HealthService определяет контракт бизнес-логики, необходимой HealthHandler
type HealthService interface {
	// CheckHealth выполняет liveness-проверку процесса
	CheckHealth(ctx context.Context) error
	// CheckReadiness проверяет готовность приложения принимать трафик
	CheckReadiness(ctx context.Context) error
}

// NewHealthHandler создаёт новый HealthHandler с переданными сервисом и логгером
func NewHealthHandler(s HealthService, l logger.Logger) *HealthHandler {
	return &HealthHandler{
		healthService: s,
		logger:        l,
	}
}

// HandleHealthCheck обрабатывает GET запрос на liveness-проверку
// @Summary Проверить, что процесс жив (liveness)
// @Description Публичный эндпоинт, не обращается к БД и внешним сервисам
// @Tags health
// @Produce json
// @Success 200 {object} dto.APIResponse
// @Failure 503 {object} dto.ErrorResponse
// @Router /health [get]
func (h *HealthHandler) HandleHealthCheck(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	if err := h.healthService.CheckHealth(r.Context()); err != nil {
		respondError(w, r, h.logger, err)
		return
	}
	respondSuccess(w, r, "OK", h.logger, nil)
}

// HandleReadinessCheck обрабатывает GET запрос на readiness-проверку
// @Summary Проверить готовность принимать трафик (readiness)
// @Description Публичный эндпоинт, проверяет доступность PostgreSQL
// @Tags health
// @Produce json
// @Success 200 {object} dto.APIResponse
// @Failure 503 {object} dto.ErrorResponse
// @Router /readyz [get]
func (h *HealthHandler) HandleReadinessCheck(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	if err := h.healthService.CheckReadiness(r.Context()); err != nil {
		respondError(w, r, h.logger, err)
		return
	}
	respondSuccess(w, r, "OK", h.logger, nil)
}
