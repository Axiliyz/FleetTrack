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
	GetTelemetryList(ctx context.Context, limit int) ([]model.Telemetry, error)
	GetTelemetryByID(ctx context.Context, id int) (model.Telemetry, error)
	GetTelemetryByVehicle(ctx context.Context, id int) ([]model.Telemetry, error)
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

	json.NewEncoder(w).Encode(
		dto.ErrorResponse{
			Status:    "error",
			Message:   message,
			RequestID: getRequestID(ctx),
		},
	)
}

// respondError логирует ошибку и отправляет ответ
func (h *TelemetryHandler) respondError(w http.ResponseWriter, r *http.Request, err error) {
	apiError := mapError(err)
	if apiError.Status >= 500 {
		h.logger.Error(err.Error())
	} else {
		h.logger.Warn(err.Error())
	}

	writeError(r.Context(), w, apiError.Message, apiError.Status)
}

// HandleTelemetry принимает входящий JSON
func (h *TelemetryHandler) HandleTelemetry(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	var telemetryData dto.TelemetryRequest

	err := json.NewDecoder(r.Body).Decode(&telemetryData)
	if err != nil {
		h.respondError(w, r, model.ErrInvalidJSON)
		return
	}
	telemetry := telemetryData.ToDomainModel()
	savedTelemetry, err := h.telemetryService.ProcessTelemetry(
		r.Context(),
		telemetry,
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

// HandleGetTelemetry возвращает список телеметрии
func (h *TelemetryHandler) HandleGetTelemetry(w http.ResponseWriter, r *http.Request) {
	telemetries, err := h.telemetryService.GetTelemetryList(r.Context(), 100)
	if err != nil {
		h.respondError(w, r, err)
		return
	}

	responses := make([]dto.TelemetryResponse, 0, len(telemetries))
	for _, t := range telemetries {
		responses = append(responses, dto.TelemetryResponse{
			TelemetryID: t.TelemetryID,
			VehicleID:   t.VehicleID,
			DeviceID:    t.DeviceID,
			ReceivedAt:  t.ReceivedAt,
		})
	}

	apiResponse := dto.APIResponse{
		Status:    "success",
		Message:   "Telemetry list",
		RequestID: getRequestID(r.Context()),
		Data:      responses,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	err = json.NewEncoder(w).Encode(apiResponse)
	if err != nil {
		h.logger.Error(err.Error())
	}
}

// HandleGetTelemetryByID возвращает запись телеметрии по ID
func (h *TelemetryHandler) HandleGetTelemetryByID(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		h.respondError(w, r, model.ErrInvalidTelemetryID)
		return
	}

	telemetry, err := h.telemetryService.GetTelemetryByID(r.Context(), id)
	if err != nil {
		h.respondError(w, r, err)
		return
	}

	telemetryResponse := dto.TelemetryResponse{
		TelemetryID: telemetry.TelemetryID,
		VehicleID:   telemetry.VehicleID,
		DeviceID:    telemetry.DeviceID,
		ReceivedAt:  telemetry.ReceivedAt,
	}

	apiResponse := dto.APIResponse{
		Status:    "success",
		Message:   "Telemetry found",
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

// HandlerGetTelemetryByVehicle возвращает все записи телеметрии по ID машины
func (h *TelemetryHandler) HandlerGetTelemetryByVehicle(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		h.respondError(w, r, err)
		return
	}

	telemetries, err := h.telemetryService.GetTelemetryByVehicle(r.Context(), id)
	if err != nil {
		h.respondError(w, r, err)
		return
	}

	responses := make([]dto.TelemetryResponse, 0, len(telemetries))
	for _, t := range telemetries {
		responses = append(responses, dto.TelemetryResponse{
			TelemetryID: t.TelemetryID,
			VehicleID:   t.VehicleID,
			DeviceID:    t.DeviceID,
			ReceivedAt:  t.ReceivedAt,
		})
	}

	apiResponse := dto.APIResponse{
		Status:    "success",
		Message:   "Telemetry list",
		RequestID: getRequestID(r.Context()),
		Data:      responses,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	err = json.NewEncoder(w).Encode(apiResponse)
	if err != nil {
		h.logger.Error(err.Error())
	}
}
