package handler

import (
	"context"
	"encoding/json"
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
		dto.ErrorResponse{
			Status:    "error",
			Message:   message,
			RequestID: getRequestID(ctx),
		},
	)

	if err != nil {
		writeError(ctx, w, model.ErrEncoding.Error(), http.StatusInternalServerError)
	}
}

func (h *TelemetryHandler) respondError(w http.ResponseWriter, r *http.Request, err error) {
	h.logger.Error(err.Error())

	apiError := mapError(err)

	writeError(r.Context(), w, apiError.Message, apiError.Status)
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
		h.respondError(w, r, model.ErrInvalidJSON)
		return
	}

	savedTelemetry, err := h.telemetryService.ProcessTelemetry(
		r.Context(),
		telemetryData,
	)

	if err != nil {
		h.respondError(w, r, err)
		return
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
		RequestID: getRequestID(r.Context()),
		Data:      telemetryResponse,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	err = json.NewEncoder(w).Encode(apiResponse)
	if err != nil {
		h.logger.Error(err.Error())
	}
}
