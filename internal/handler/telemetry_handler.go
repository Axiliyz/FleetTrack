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
)

// TelemetryHandler передаёт данные в сервис и логирует
type TelemetryHandler struct {
	telemetryService TelemetryService
	logger           logger.Logger
}

// TelemetryService определяет контракт обработки телеметрии
type TelemetryService interface {
	ProcessTelemetry(ctx context.Context, t model.Telemetry) (model.Telemetry, error)
}

// NewTelemetryHandler создаёт новый хэндлер с заданным сервисом и логгером
func NewTelemetryHandler(service TelemetryService, logger logger.Logger) *TelemetryHandler {
	return &TelemetryHandler{
		telemetryService: service,
		logger:           logger,
	}
}

// getRequestID извлекает request ID из контекста
func getRequestID(ctx context.Context) string {
	id, ok := ctx.Value(
		middleware.RequestIDKey,
	).(string)

	if !ok {
		return "unknown"
	}

	return id
}

// writeError записывает ошибку в JSON
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
		return
	}
}

// respondError логирует ошибку и отправляет ответ
func (h *TelemetryHandler) respondError(w http.ResponseWriter, r *http.Request, err error) {
	h.logger.Error(err.Error())

	apiError := mapError(err)

	writeError(r.Context(), w, apiError.Message, apiError.Status)
}

// HandleTelemetry принимает входящий JSON
func (h *TelemetryHandler) HandleTelemetry(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	if r.Method != http.MethodPost {
		writeError(r.Context(), w, "method not supported", http.StatusMethodNotAllowed)
		return
	}

	var telemetryData model.Telemetry

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
