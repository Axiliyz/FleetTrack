package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fleettrack/internal/handler/dto"
	"fleettrack/internal/logger"
	"fleettrack/internal/middleware"
	"fleettrack/internal/model"
	"fleettrack/internal/service"
	"net/http"
)

type TelemetryHandler struct {
	telemetryService *service.TelemetryService
	logger           logger.Logger
}

func NewTelemetryHandler(service *service.TelemetryService, logger logger.Logger) *TelemetryHandler {
	return &TelemetryHandler{
		telemetryService: service,
		logger:           logger,
	}
}

func writeError(ctx context.Context, w http.ResponseWriter, message string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)

	json.NewEncoder(w).Encode(
		model.ErrorResponse{
			Status:    "error",
			Message:   message,
			RequestID: ctx.Value(middleware.RequestIDKey).(string),
		},
	)
}

func (h *TelemetryHandler) HandleTelemetry(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	if r.Method != http.MethodPost {
		writeError(r.Context(), w, "Method not supported", http.StatusMethodNotAllowed)
		h.logger.Error("Method not supported")
		return
	}

	var telemetryData model.Telemetry

	// // TODO: ТУТ ЭТОГО БЫТЬ НЕ ДОЛЖНО, НАДО ЧТОБЫ ПРИХОДИЛО С ESP
	// telemetryData.DeviceTimestamp = time.Now()
	if err := json.NewDecoder(r.Body).Decode(&telemetryData); err != nil {
		writeError(r.Context(), w, "Invalid JSON", http.StatusBadRequest)
		h.logger.Error("Invalid JSON")
		return
	}

	savedTelemetry, err := h.telemetryService.ProcessTelemetry(
		r.Context(),
		telemetryData,
	)

	if err != nil {
		if errors.Is(err, model.ErrInvalidID) {
			writeError(r.Context(), w, err.Error(), http.StatusBadRequest)
			h.logger.Error("Invalid ID or vehicle number")
			return
		} else if errors.Is(err, model.ErrInvalidCoords) {
			writeError(r.Context(), w, err.Error(), http.StatusBadRequest)
			h.logger.Error("Invalid coords")
			return
		} else {
			writeError(r.Context(), w, err.Error(), http.StatusInternalServerError)
			h.logger.Error("Internal server error")
			return
		}
	}

	response := dto.TelemetryResponse{
		TelemetryID: savedTelemetry.TelemetryID,
		VehicleID:   savedTelemetry.VehicleID,
		DeviceID:    savedTelemetry.DeviceID,
		ReceivedAt:  savedTelemetry.ReceivedAt,
	}

	apiResponse := dto.APIResponse{
		Status:    "success",
		Message:   "Telemetry saved",
		RequestID: r.Context().Value(middleware.RequestIDKey).(string),
		Data:      response,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(apiResponse)
}
