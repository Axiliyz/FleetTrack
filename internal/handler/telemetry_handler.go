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
	GetTelemetryByID(ctx context.Context, id int) (model.Telemetry, error)
	GetTelemetryByVehicle(ctx context.Context, id int) ([]model.Telemetry, error)
	DeleteTelemetryByID(ctx context.Context, id int, organizationID *int) (model.Telemetry, error)
	DeleteTelemetryByVehicle(ctx context.Context, id int, organizationID *int) ([]model.Telemetry, error)
}

// NewTelemetryHandler создаёт новый хэндлер с заданным сервисом и логгером
func NewTelemetryHandler(service TelemetryService, logger logger.Logger) *TelemetryHandler {
	return &TelemetryHandler{
		telemetryService: service,
		logger:           logger,
	}
}

// HandleTelemetry принимает входящий JSON
func (h *TelemetryHandler) HandleTelemetry(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

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

	telemetry := telemetryData.ToDomainModel()
	telemetry.OrganizationID = authCtx.OrganizationID
	savedTelemetry, err := h.telemetryService.ProcessTelemetry(
		r.Context(),
		telemetry,
	)

	if err != nil {
		respondError(w, r, h.logger, err)
		return
	}

	telemetryResponse := dto.TelemetryResponse{
		TelemetryID: savedTelemetry.TelemetryID,
		VehicleID:   savedTelemetry.VehicleID,
		DeviceID:    savedTelemetry.DeviceID,
		ReceivedAt:  savedTelemetry.ReceivedAt,
		TripID:      savedTelemetry.TripID,
		DistanceKm:  savedTelemetry.DistanceKm,
		SpeedKmh:    savedTelemetry.SpeedKmh,
	}

	respondSuccess(w, r, "Telemetry got to post", h.logger, telemetryResponse)
}

// HandleGetListTelemetry возвращает список телеметрии
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

	responses := make([]dto.TelemetryResponse, 0, len(telemetries))
	for _, t := range telemetries {
		responses = append(responses, dto.TelemetryResponse{
			TelemetryID: t.TelemetryID,
			VehicleID:   t.VehicleID,
			DeviceID:    t.DeviceID,
			ReceivedAt:  t.ReceivedAt,
			TripID:      t.TripID,
			DistanceKm:  t.DistanceKm,
			SpeedKmh:    t.SpeedKmh,
		})
	}

	respondSuccess(w, r, "Telemetry list", h.logger, responses)
}

// HandleGetTelemetryByID возвращает запись телеметрии по ID
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

	telemetry, err := h.telemetryService.GetTelemetryByID(r.Context(), id)
	if err != nil {
		respondError(w, r, h.logger, err)
		return
	}
	if err := requireOwnOrg(authCtx, telemetry.OrganizationID); err != nil {
		respondError(w, r, h.logger, err)
		return
	}

	telemetryResponse := dto.TelemetryResponse{
		TelemetryID: telemetry.TelemetryID,
		VehicleID:   telemetry.VehicleID,
		DeviceID:    telemetry.DeviceID,
		ReceivedAt:  telemetry.ReceivedAt,
		TripID:      telemetry.TripID,
		DistanceKm:  telemetry.DistanceKm,
		SpeedKmh:    telemetry.SpeedKmh,
	}

	respondSuccess(w, r, "Telemetry found", h.logger, telemetryResponse)
}

// HandleGetTelemetryByVehicle возвращает все записи телеметрии по ID машины
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

	telemetries, err := h.telemetryService.GetTelemetryByVehicle(r.Context(), id)
	if err != nil {
		respondError(w, r, h.logger, err)
		return
	}
	owned := telemetries[:0]
	for _, t := range telemetries {
		if t.OrganizationID == authCtx.OrganizationID {
			owned = append(owned, t)
		}
	}
	telemetries = owned

	responses := make([]dto.TelemetryResponse, 0, len(telemetries))
	for _, t := range telemetries {
		responses = append(responses, dto.TelemetryResponse{
			TelemetryID: t.TelemetryID,
			VehicleID:   t.VehicleID,
			DeviceID:    t.DeviceID,
			ReceivedAt:  t.ReceivedAt,
			TripID:      t.TripID,
			DistanceKm:  t.DistanceKm,
			SpeedKmh:    t.SpeedKmh,
		})
	}

	respondSuccess(w, r, "Telemetry list", h.logger, responses)
}

// HandleDeleteTelemetryByID удаляет телеметрию по её ID
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

	telemetryResponse := dto.TelemetryResponse{
		TelemetryID: deleted.TelemetryID,
		VehicleID:   deleted.VehicleID,
		DeviceID:    deleted.DeviceID,
		ReceivedAt:  deleted.ReceivedAt,
		TripID:      deleted.TripID,
		DistanceKm:  deleted.DistanceKm,
		SpeedKmh:    deleted.SpeedKmh,
	}

	respondSuccess(w, r, "Telemetry deleted", h.logger, telemetryResponse)
}

// HandleDeleteTelemetryByVehicleID удаляет телеметрию по машине по её ID
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

	telemetries, err := h.telemetryService.DeleteTelemetryByVehicle(r.Context(), id, orgScope)
	if err != nil {
		respondError(w, r, h.logger, err)
		return
	}
	responses := make([]dto.TelemetryResponse, 0, len(telemetries))
	for _, t := range telemetries {
		responses = append(responses, dto.TelemetryResponse{
			TelemetryID: t.TelemetryID,
			VehicleID:   t.VehicleID,
			DeviceID:    t.DeviceID,
			ReceivedAt:  t.ReceivedAt,
			TripID:      t.TripID,
			DistanceKm:  t.DistanceKm,
			SpeedKmh:    t.SpeedKmh,
		})
	}

	respondSuccess(w, r, "Telemetries of vehicle deleted", h.logger, responses)
}
