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

// TripHandler обрабатывает HTTP запросы, связанные с рейсами
type TripHandler struct {
	tripService TripService
	logger      logger.Logger
}

// TripService определяет контракт бизнес-логики, необходимой TripHandler
type TripService interface {
	AssignTrip(ctx context.Context, driverID int, vehicleID int, organizationID *int) (model.Trip, error)
	GetTripByID(ctx context.Context, id int, organizationID *int) (model.Trip, error)
	UpdateTrip(ctx context.Context, id int, upd model.Trip, organizationID *int) (model.Trip, error)
	DeleteTrip(ctx context.Context, id int, organizationID *int) (model.Trip, error)
	GetListTrips(ctx context.Context, filter model.TripFilter) ([]model.Trip, error)
}

// NewTripHandler создаёт новый TripHandler с переданными сервисом и логгером
func NewTripHandler(s TripService, l logger.Logger) *TripHandler {
	return &TripHandler{
		tripService: s,
		logger:      l,
	}
}

// HandleAssignTrip работает с POST запросом создания связи для рейса
// @Summary Открыть новый рейс
// @Tags trips
// @Accept json
// @Produce json
// @Param request body dto.TripRequest true "ID водителя и автомобиля"
// @Success 201 {object} model.Trip
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 403 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse "водитель или автомобиль не найдены либо принадлежат другой организации"
// @Router /trips [post]
// @Security BearerAuth
func (h *TripHandler) HandleAssignTrip(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	authCtx, ok := middleware.AuthFromContext(r.Context())
	if !ok {
		respondError(w, r, h.logger, model.ErrMissingToken)
		return
	}

	var TripData dto.TripRequest
	err := json.NewDecoder(r.Body).Decode(&TripData)
	if err != nil {
		respondError(w, r, h.logger, model.ErrInvalidJSON)
		return
	}
	trip := TripData.ToDomain()
	orgScope := scopeOrganizationID(authCtx)
	savedTrip, err := h.tripService.AssignTrip(r.Context(), trip.DriverID, trip.VehicleID, orgScope)
	if err != nil {
		respondError(w, r, h.logger, err)
		return
	}

	respondCreated(w, r, "trip created", h.logger, savedTrip)
}

// HandleUpdateTrip обрабатывает PATCH запрос на изменение статуса рейса
// @Summary Изменить статус рейса
// @Tags trips
// @Accept json
// @Produce json
// @Param id path int true "ID рейса"
// @Param request body dto.UpdateTripRequest true "Новый статус (COMPLETED, CANCELLED)"
// @Success 200 {object} model.Trip
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 403 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 409 {object} dto.ErrorResponse "рейс уже завершён"
// @Router /trips/{id} [patch]
// @Security BearerAuth
func (h *TripHandler) HandleUpdateTrip(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	authCtx, ok := middleware.AuthFromContext(r.Context())
	if !ok {
		respondError(w, r, h.logger, model.ErrMissingToken)
		return
	}

	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, r, h.logger, model.ErrInvalidTripID)
		return
	}

	var updateData dto.UpdateTripRequest
	if err := json.NewDecoder(r.Body).Decode(&updateData); err != nil {
		respondError(w, r, h.logger, model.ErrInvalidJSON)
		return
	}

	orgScope := scopeOrganizationID(authCtx)
	updatedTrip, err := h.tripService.UpdateTrip(r.Context(), id, updateData.ToDomain(), orgScope)
	if err != nil {
		respondError(w, r, h.logger, err)
		return
	}

	respondSuccess(w, r, "trip updated", h.logger, updatedTrip)
}

// HandleDeleteTrip обрабатывает DELETE запрос на отмену рейса
// @Summary Отменить рейс
// @Tags trips
// @Produce json
// @Param id path int true "ID рейса"
// @Success 200 {object} model.Trip
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 403 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 409 {object} dto.ErrorResponse "рейс уже завершён"
// @Router /trips/{id} [delete]
// @Security BearerAuth
func (h *TripHandler) HandleDeleteTrip(w http.ResponseWriter, r *http.Request) {
	authCtx, ok := middleware.AuthFromContext(r.Context())
	if !ok {
		respondError(w, r, h.logger, model.ErrMissingToken)
		return
	}

	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, r, h.logger, model.ErrInvalidTripID)
		return
	}

	orgScope := scopeOrganizationID(authCtx)
	deletedTrip, err := h.tripService.DeleteTrip(r.Context(), id, orgScope)
	if err != nil {
		respondError(w, r, h.logger, err)
		return
	}

	respondSuccess(w, r, "trip cancelled", h.logger, deletedTrip)
}

// HandleGetListTrips возвращает список рейсов с фильтрами
// @Summary Список рейсов с фильтрами
// @Tags trips
// @Produce json
// @Param driver_id query int false "ID водителя (для роли DRIVER игнорируется и подставляется свой)"
// @Param vehicle_id query int false "ID автомобиля"
// @Param status query string false "Статус рейса"
// @Param started_from query string false "От даты начала (RFC3339)"
// @Param started_to query string false "До даты начала (RFC3339)"
// @Param min_distance query number false "Минимальная дистанция, км"
// @Param max_distance query number false "Максимальная дистанция, км"
// @Param min_avg_speed query number false "Минимальная средняя скорость"
// @Param max_avg_speed query number false "Максимальная средняя скорость"
// @Param min_max_speed query number false "Минимальная макс. скорость"
// @Param max_max_speed query number false "Максимальная макс. скорость"
// @Param limit query int false "Лимит записей (по умолчанию 100, максимум 500)"
// @Param offset query int false "Смещение"
// @Success 200 {array} model.Trip
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 403 {object} dto.ErrorResponse "роль DRIVER без привязанного водителя"
// @Router /trips [get]
// @Security BearerAuth
func (h *TripHandler) HandleGetListTrips(w http.ResponseWriter, r *http.Request) {
	filter, err := dto.ParseTripFilter(r.URL.Query())
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

	trips, err := h.tripService.GetListTrips(r.Context(), filter)
	if err != nil {
		respondError(w, r, h.logger, err)
		return
	}

	respondSuccess(w, r, "trips list", h.logger, trips)
}

// HandleGetTripByID возвращает рейс по ID
// @Summary Получить рейс по ID
// @Tags trips
// @Produce json
// @Param id path int true "ID рейса"
// @Success 200 {object} model.Trip
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Router /trips/{id} [get]
// @Security BearerAuth
func (h *TripHandler) HandleGetTripByID(w http.ResponseWriter, r *http.Request) {
	authCtx, ok := middleware.AuthFromContext(r.Context())
	if !ok {
		respondError(w, r, h.logger, model.ErrMissingToken)
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		respondError(w, r, h.logger, model.ErrInvalidTripID)
		return
	}

	orgScope := scopeOrganizationID(authCtx)
	trip, err := h.tripService.GetTripByID(r.Context(), id, orgScope)
	if err != nil {
		respondError(w, r, h.logger, err)
		return
	}
	respondSuccess(w, r, "trip found", h.logger, trip)
}
