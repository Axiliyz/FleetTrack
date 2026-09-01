package handler

import (
	"context"
	"encoding/json"
	"fleettrack/internal/handler/dto"
	"fleettrack/internal/logger"
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
	AssignTrip(ctx context.Context, driverID int, vehicleID int) (model.Trip, error)
	GetTripByID(ctx context.Context, id int) (model.Trip, error)
	UpdateTrip(ctx context.Context, id int, upd model.Trip) (model.Trip, error)
	DeleteTrip(ctx context.Context, id int) (model.Trip, error)
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
func (h *TripHandler) HandleAssignTrip(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	var TripData dto.TripRequest
	err := json.NewDecoder(r.Body).Decode(&TripData)
	if err != nil {
		respondError(w, r, h.logger, model.ErrInvalidJSON)
		return
	}
	trip := TripData.ToDomain()
	savedTrip, err := h.tripService.AssignTrip(r.Context(), trip.DriverID, trip.VehicleID)
	if err != nil {
		respondError(w, r, h.logger, err)
		return
	}

	respondSuccess(w, r, "trip created", h.logger, savedTrip)
}

// HandleUpdateTrip обрабатывает PATCH запрос на изменение статуса рейса
func (h *TripHandler) HandleUpdateTrip(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

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

	updatedTrip, err := h.tripService.UpdateTrip(r.Context(), id, updateData.ToDomain())
	if err != nil {
		respondError(w, r, h.logger, err)
		return
	}

	respondSuccess(w, r, "trip updated", h.logger, updatedTrip)
}

// HandleDeleteTrip обрабатывает DELETE запрос на отмену рейса
func (h *TripHandler) HandleDeleteTrip(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, r, h.logger, model.ErrInvalidTripID)
		return
	}

	deletedTrip, err := h.tripService.DeleteTrip(r.Context(), id)
	if err != nil {
		respondError(w, r, h.logger, err)
		return
	}

	respondSuccess(w, r, "trip cancelled", h.logger, deletedTrip)
}

// HandleGetListTrips возвращает список рейсов с фильтрами
func (h *TripHandler) HandleGetListTrips(w http.ResponseWriter, r *http.Request) {
	filter, err := dto.ParseTripFilter(r.URL.Query())
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
func (h *TripHandler) HandleGetTripByID(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		respondError(w, r, h.logger, model.ErrInvalidTripID)
		return
	}

	trip, err := h.tripService.GetTripByID(r.Context(), id)
	if err != nil {
		respondError(w, r, h.logger, err)
		return
	}
	respondSuccess(w, r, "trip found", h.logger, trip)
}
