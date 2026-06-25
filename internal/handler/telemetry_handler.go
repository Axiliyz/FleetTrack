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

func getRequestID(ctx context.Context) string {
	id, ok := ctx.Value(
		middleware.RequestIDKey,
	).(string)

	if !ok {
		return "unknown"
	}

	return id
}

func writeError(ctx context.Context, w http.ResponseWriter, message string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)

	err := json.NewEncoder(w).Encode(
		model.ErrorResponse{
			Status:    "error",
			Message:   message,
			RequestID: getRequestID(ctx),
		},
	)

	if err != nil {
		writeError(ctx, w, model.ErrEncoding.Error(), http.StatusBadRequest)
	}
}

func (h *TelemetryHandler) HandleTelemetry(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	if r.Method != http.MethodPost {
		writeError(r.Context(), w, "method not supported", http.StatusMethodNotAllowed)
		h.logger.Error("method not supported")
		return
	}

	var telemetryData model.Telemetry

	// // TODO: ТУТ ЭТОГО БЫТЬ НЕ ДОЛЖНО, НАДО ЧТОБЫ ПРИХОДИЛО С ESP
	// telemetryData.DeviceTimestamp = time.Now()
	err := json.NewDecoder(r.Body).Decode(&telemetryData)
	if err != nil {
		writeError(r.Context(), w, "invalid JSON", http.StatusBadRequest)
		h.logger.Error("invalid JSON")
		return
	}

	savedTelemetry, err := h.telemetryService.ProcessTelemetry(
		r.Context(),
		telemetryData,
	)

	if err != nil {
		if errors.Is(err, model.ErrInvalidID) {
			writeError(r.Context(), w, err.Error(), http.StatusBadRequest)
			h.logger.Error("invalid ID or vehicle number")
			return
		} else if errors.Is(err, model.ErrInvalidCoords) {
			writeError(r.Context(), w, err.Error(), http.StatusBadRequest)
			h.logger.Error("invalid coords")
			return
		} else {
			writeError(r.Context(), w, err.Error(), http.StatusInternalServerError)
			h.logger.Error("internal server error")
			return
		}
	}

	telemetryResponse := dto.TelemetryResponse{
		TelemetryID: savedTelemetry.TelemetryID,
		VehicleID:   savedTelemetry.VehicleID,
		DeviceID:    savedTelemetry.DeviceID,
		ReceivedAt:  savedTelemetry.ReceivedAt,
	}

	apiResponse := dto.APIResponse{
		Status:    "success",
		Message:   "Telemetry saved",
		RequestID: r.Context().Value(middleware.RequestIDKey).(string),
		Data:      telemetryResponse,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	err = json.NewEncoder(w).Encode(apiResponse)
	if err != nil {
		writeError(r.Context(), w, model.ErrEncoding.Error(), http.StatusBadRequest)
		h.logger.Error("failed encoding")
	}
}
