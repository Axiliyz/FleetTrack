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

// VehicleHandler обрабатывает HTTP запросы, связанные с автомобилями
type VehicleHandler struct {
	vehicleService VehicleService
	logger         logger.Logger
}

// VehicleService определяет контракт бизнес-логики, необходимой VehicleHandler
type VehicleService interface {
	// ProcessVehicle валидирует и сохраняет новый автомобиль
	ProcessVehicle(ctx context.Context, v model.Vehicle) (model.Vehicle, error)
	// GetVehicleList возвращает список автомобилей по фильтру
	GetVehicleList(ctx context.Context, filter model.VehicleFilter) ([]model.Vehicle, error)
	// GetVehicleByID возвращает автомобиль по его ID
	GetVehicleByID(ctx context.Context, id int) (model.Vehicle, error)
	// DeleteVehicleByID удаляет автомобиль по его ID
	DeleteVehicleByID(ctx context.Context, id int) (model.Vehicle, error)
	// UpdateVehicle
	UpdateVehicleByID(ctx context.Context, id int, upd model.UpdateVehicle) (model.Vehicle, error)
}

// NewVehicleHandler создаёт новый VehicleHandler с переданными сервисом и логгером
func NewVehicleHandler(s VehicleService, l logger.Logger) *VehicleHandler {
	return &VehicleHandler{
		vehicleService: s,
		logger:         l,
	}
}

// HandlePostVehicle обрабатывает POST запрос на создание нового автомобиля
func (h *VehicleHandler) HandlePostVehicle(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	var vehicleData dto.CreateVehicleRequest
	err := json.NewDecoder(r.Body).Decode(&vehicleData)
	if err != nil {
		respondError(w, r, h.logger, model.ErrInvalidJSON)
		return
	}

	vehicle := vehicleData.ToDomainModel()
	savedVehicle, err := h.vehicleService.ProcessVehicle(r.Context(), vehicle)
	if err != nil {
		respondError(w, r, h.logger, err)
		return
	}
	vehicleResponse := dto.VehicleResponse{
		ID:             savedVehicle.ID,
		OrganizationID: savedVehicle.OrganizationID,
		VIN:            savedVehicle.VIN,
		NumberPlate:    savedVehicle.NumberPlate,
		Model:          savedVehicle.Model,
		Status:         savedVehicle.Status,
	}

	respondSuccess(w, r, "vehicle created", h.logger, vehicleResponse)
}

// HandleDeleteVehicle обрабатывает DELETE запрос на удаление автомобиля по ID
func (h *VehicleHandler) HandleDeleteVehicle(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		respondError(w, r, h.logger, model.ErrInvalidVehicleID)
		return
	}

	deletedVehicle, err := h.vehicleService.DeleteVehicleByID(r.Context(), id)
	if err != nil {
		respondError(w, r, h.logger, err)
		return
	}

	vehicleResponse := dto.VehicleResponse{
		ID:             deletedVehicle.ID,
		OrganizationID: deletedVehicle.OrganizationID,
		VIN:            deletedVehicle.VIN,
		NumberPlate:    deletedVehicle.NumberPlate,
		Model:          deletedVehicle.Model,
		Status:         deletedVehicle.Status,
	}

	respondSuccess(w, r, "vehicle deleted", h.logger, vehicleResponse)
}

// HandleGetListVehicle возвращает список автомобилей с фильтрами
func (h *VehicleHandler) HandleGetListVehicle(w http.ResponseWriter, r *http.Request) {
	filter, err := dto.ParseVehicleFilter(r.URL.Query())
	if err != nil {
		respondError(w, r, h.logger, err)
		return
	}

	vehicles, err := h.vehicleService.GetVehicleList(r.Context(), filter)
	if err != nil {
		respondError(w, r, h.logger, err)
		return
	}

	responses := make([]dto.VehicleResponse, 0, len(vehicles))
	for _, t := range vehicles {
		responses = append(responses, dto.VehicleResponse{
			ID:             t.ID,
			OrganizationID: t.OrganizationID,
			VIN:            t.VIN,
			NumberPlate:    t.NumberPlate,
			Model:          t.Model,
			Status:         t.Status,
		})
	}

	respondSuccess(w, r, "Vehicles list", h.logger, responses)
}

// HandlePatchVehicle отвечает за изменение некоторых данных автомобиля
func (h *VehicleHandler) HandlePatchVehicle(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, r, h.logger, model.ErrInvalidVehicleID)
		return
	}

	var request dto.UpdateVehicleRequest
	err = json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		respondError(w, r, h.logger, model.ErrInvalidJSON)
		return
	}
	upd := request.ToDomainModel()
	changed, err := h.vehicleService.UpdateVehicleByID(r.Context(), id, upd)
	if err != nil {
		respondError(w, r, h.logger, err)
		return
	}

	vehicleResponse := dto.VehicleResponse{
		ID:             changed.ID,
		OrganizationID: changed.OrganizationID,
		VIN:            changed.VIN,
		NumberPlate:    changed.NumberPlate,
		Model:          changed.Model,
		Status:         changed.Status,
	}
	respondSuccess(w, r, "vehicle was changed", h.logger, vehicleResponse)
}

// HandleGetVehicleByID получает машину по ID
func (h *VehicleHandler) HandleGetVehicleByID(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, r, h.logger, model.ErrInvalidVehicleID)
		return
	}

	vehicle, err := h.vehicleService.GetVehicleByID(r.Context(), id)
	if err != nil {
		respondError(w, r, h.logger, err)
		return
	}

	vehicleResponse := dto.VehicleResponse{
		ID:             vehicle.ID,
		VIN:            vehicle.VIN,
		Model:          vehicle.Model,
		OrganizationID: vehicle.OrganizationID,
		NumberPlate:    vehicle.NumberPlate,
		Status:         vehicle.Status,
	}

	respondSuccess(w, r, "vehicle found", h.logger, vehicleResponse)
}
