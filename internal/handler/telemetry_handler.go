// Package handler содержит приём данных из внешнего мира
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

// TelemetryHandler передаёт данные в сервис и логирует
type TelemetryHandler struct {
	telemetryService TelemetryService
	logger           logger.Logger
}

// TelemetryService определяет контракт обработки телеметрии
type TelemetryService interface {
	ProcessTelemetry(ctx context.Context, t model.Telemetry) (model.Telemetry, error)
	GetTelemetryList(ctx context.Context, filter model.TelemetryFilter) ([]model.Telemetry, error)
	GetTelemetryByID(ctx context.Context, id int, driverID *int) (model.Telemetry, error)
	GetTelemetryByVehicle(ctx context.Context, id int, organizationID, driverID *int) ([]model.Telemetry, error)
	DeleteTelemetryByID(ctx context.Context, id int, organizationID *int) (model.Telemetry, error)
	DeleteTelemetryByVehicle(ctx context.Context, id int, organizationID *int) (int64, error)
}

// NewTelemetryHandler создаёт новый хэндлер с заданным сервисом и логгером
func NewTelemetryHandler(service TelemetryService, logger logger.Logger) *TelemetryHandler {
	return &TelemetryHandler{
		telemetryService: service,
		logger:           logger,
	}
}

// HandleTelemetry принимает входящий JSON
// @Summary Принять точку телеметрии от GPS-трекера
// @Tags telemetry
// @Accept json
// @Produce json
// @Param request body dto.TelemetryRequest true "Точка телеметрии"
// @Success 201 {object} dto.APIResponse{data=dto.TelemetryResponse}
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse "автомобиль не найден или принадлежит другой организации"
// @Router /telemetry [post]
// @Security BearerAuth
func (h *TelemetryHandler) HandleTelemetry(w http.ResponseWriter, r *http.Request) {
	var telemetryData dto.TelemetryRequest

	err := json.NewDecoder(r.Body).Decode(&telemetryData)
	if err != nil {
		respondError(w, r, h.logger, model.ErrInvalidJSON)
		return
	}

	authCtx, ok := middleware.AuthFromContext(r.Context())
	if !ok {
		respondError(w, r, h.logger, model.ErrMissingToken)
		return
	}

	telemetry, err := telemetryData.ToDomainModel()
	if err != nil {
		respondError(w, r, h.logger, err)
		return
	}
	telemetry.OrganizationID = authCtx.OrganizationID
	savedTelemetry, err := h.telemetryService.ProcessTelemetry(
		r.Context(),
		telemetry,
	)

	if err != nil {
		respondError(w, r, h.logger, err)
		return
	}

	telemetryResponse := dto.NewTelemetryResponse(savedTelemetry)

	respondCreated(w, r, "telemetry stored", h.logger, telemetryResponse)
}

// HandleGetListTelemetry возвращает список телеметрии
// @Summary Список телеметрии с фильтрами
// @Tags telemetry
// @Produce json
// @Param vehicle_id query int false "ID автомобиля"
// @Param device_id query int false "ID трекера"
// @Param trip_id query int false "ID рейса"
// @Param driver_id query int false "ID водителя (для роли DRIVER игнорируется и подставляется свой)"
// @Param lat_min query number false "Минимальная широта"
// @Param lat_max query number false "Максимальная широта"
// @Param lon_min query number false "Минимальная долгота"
// @Param lon_max query number false "Максимальная долгота"
// @Param fuel_min query number false "Минимальный уровень топлива"
// @Param fuel_max query number false "Максимальный уровень топлива"
// @Param from query string false "От даты (RFC3339)"
// @Param to query string false "До даты (RFC3339)"
// @Param limit query int false "Лимит записей (по умолчанию 100, максимум 500)"
// @Param offset query int false "Смещение"
// @Success 200 {object} dto.APIResponse{data=[]dto.TelemetryResponse}
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 403 {object} dto.ErrorResponse "роль DRIVER без привязанного водителя"
// @Router /telemetry [get]
// @Security BearerAuth
func (h *TelemetryHandler) HandleGetListTelemetry(w http.ResponseWriter, r *http.Request) {
	filter, err := dto.ParseTelemetryFilter(r.URL.Query())
	if err != nil {
		respondError(w, r, h.logger, err)
		return
	}

	authCtx, ok := middleware.AuthFromContext(r.Context())
	if !ok {
		respondError(w, r, h.logger, model.ErrMissingToken)
		return
	}
	filter.OrganizationID = scopeOrganizationID(authCtx)
	filter.DriverID, err = scopeDriverID(authCtx, filter.DriverID)
	if err != nil {
		respondError(w, r, h.logger, err)
		return
	}

	telemetries, err := h.telemetryService.GetTelemetryList(r.Context(), filter)
	if err != nil {
		respondError(w, r, h.logger, err)
		return
	}

	responses := dto.NewTelemetryListResponse(telemetries)

	respondSuccess(w, r, "telemetry list", h.logger, responses)
}

// HandleGetTelemetryByID возвращает запись телеметрии по ID
// @Summary Получить точку телеметрии по ID
// @Tags telemetry
// @Produce json
// @Param id path int true "ID записи телеметрии"
// @Success 200 {object} dto.APIResponse{data=dto.TelemetryResponse}
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Router /telemetry/{id} [get]
// @Security BearerAuth
func (h *TelemetryHandler) HandleGetTelemetryByID(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		respondError(w, r, h.logger, model.ErrInvalidTelemetryID)
		return
	}

	authCtx, ok := middleware.AuthFromContext(r.Context())
	if !ok {
		respondError(w, r, h.logger, model.ErrMissingToken)
		return
	}

	// водитель видит только точки своих рейсов
	driverScope, err := scopeDriverFilter(authCtx)
	if err != nil {
		respondError(w, r, h.logger, err)
		return
	}

	telemetry, err := h.telemetryService.GetTelemetryByID(r.Context(), id, driverScope)
	if err != nil {
		respondError(w, r, h.logger, err)
		return
	}
	if err := requireOwnOrg(authCtx, telemetry.OrganizationID); err != nil {
		respondError(w, r, h.logger, err)
		return
	}

	telemetryResponse := dto.NewTelemetryResponse(telemetry)

	respondSuccess(w, r, "telemetry found", h.logger, telemetryResponse)
}

// HandleGetTelemetryByVehicle возвращает последние записи телеметрии по ID машины;
// водителю - только точки его рейсов
// @Summary Последние 100 точек автомобиля
// @Tags telemetry
// @Produce json
// @Param id path int true "ID автомобиля"
// @Success 200 {object} dto.APIResponse{data=[]dto.TelemetryResponse}
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Router /telemetry/vehicles/{id} [get]
// @Security BearerAuth
func (h *TelemetryHandler) HandleGetTelemetryByVehicle(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		respondError(w, r, h.logger, model.ErrInvalidVehicleID)
		return
	}

	authCtx, ok := middleware.AuthFromContext(r.Context())
	if !ok {
		respondError(w, r, h.logger, model.ErrMissingToken)
		return
	}

	orgScope := scopeOrganizationID(authCtx)
	driverScope, err := scopeDriverFilter(authCtx)
	if err != nil {
		respondError(w, r, h.logger, err)
		return
	}
	telemetries, err := h.telemetryService.GetTelemetryByVehicle(r.Context(), id, orgScope, driverScope)
	if err != nil {
		respondError(w, r, h.logger, err)
		return
	}

	responses := dto.NewTelemetryListResponse(telemetries)

	respondSuccess(w, r, "telemetry list", h.logger, responses)
}

// HandleDeleteTelemetryByID удаляет телеметрию по её ID
// @Summary Удалить точку телеметрии
// @Tags telemetry
// @Produce json
// @Param id path int true "ID записи телеметрии"
// @Success 200 {object} dto.APIResponse{data=dto.TelemetryResponse}
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 403 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Router /telemetry/{id} [delete]
// @Security BearerAuth
func (h *TelemetryHandler) HandleDeleteTelemetryByID(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		respondError(w, r, h.logger, model.ErrInvalidTelemetryID)
		return
	}
	authCtx, ok := middleware.AuthFromContext(r.Context())
	if !ok {
		respondError(w, r, h.logger, model.ErrMissingToken)
		return
	}
	orgScope := scopeOrganizationID(authCtx)

	deleted, err := h.telemetryService.DeleteTelemetryByID(r.Context(), id, orgScope)
	if err != nil {
		respondError(w, r, h.logger, err)
		return
	}

	telemetryResponse := dto.NewTelemetryResponse(deleted)

	respondSuccess(w, r, "telemetry deleted", h.logger, telemetryResponse)
}

// HandleDeleteTelemetryByVehicleID удаляет телеметрию по машине по её ID.
// В ответе только число удалённых точек: самих точек за месяцы могут быть миллионы
// @Summary Удалить всю телеметрию автомобиля
// @Tags telemetry
// @Produce json
// @Param id path int true "ID автомобиля"
// @Success 200 {object} dto.APIResponse{data=dto.DeletedCountResponse}
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 403 {object} dto.ErrorResponse
// @Router /telemetry/vehicles/{id} [delete]
// @Security BearerAuth
func (h *TelemetryHandler) HandleDeleteTelemetryByVehicleID(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		respondError(w, r, h.logger, model.ErrInvalidVehicleID)
		return
	}

	authCtx, ok := middleware.AuthFromContext(r.Context())
	if !ok {
		respondError(w, r, h.logger, model.ErrMissingToken)
		return
	}
	orgScope := scopeOrganizationID(authCtx)

	deleted, err := h.telemetryService.DeleteTelemetryByVehicle(r.Context(), id, orgScope)
	if err != nil {
		respondError(w, r, h.logger, err)
		return
	}

	respondSuccess(w, r, "vehicle telemetry deleted", h.logger, dto.DeletedCountResponse{Deleted: deleted})
}
