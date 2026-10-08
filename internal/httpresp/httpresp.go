// Package httpresp формирует JSON-ответы API в едином формате для хендлеров и middleware.
package httpresp

import (
	"encoding/json"
	"fleettrack/internal/handler/dto"
	"fleettrack/internal/logger"
	"fleettrack/internal/requestid"
	"net/http"
)

// JSON записывает статус и тело v в формате JSON.
func JSON(w http.ResponseWriter, log logger.Logger, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Error("encode response: " + err.Error())
	}
}

// Success отправляет успешный ответ в конверте dto.APIResponse.
func Success(w http.ResponseWriter, r *http.Request, log logger.Logger, status int, message string, data any) {
	JSON(w, log, status, dto.APIResponse{
		Status:    "success",
		Message:   message,
		RequestID: requestid.FromContext(r.Context()),
		Data:      data,
	})
}

// Error отправляет ответ с ошибкой в формате dto.ErrorResponse.
func Error(w http.ResponseWriter, r *http.Request, log logger.Logger, status int, message string) {
	JSON(w, log, status, dto.ErrorResponse{
		Status:    "error",
		Message:   message,
		RequestID: requestid.FromContext(r.Context()),
	})
}
