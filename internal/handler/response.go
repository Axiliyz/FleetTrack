package handler

import (
	"context"
	"encoding/json"
	"fleettrack/internal/handler/dto"
	"fleettrack/internal/logger"
	"fleettrack/internal/middleware"
	"net/http"
)

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
func respondError(w http.ResponseWriter, r *http.Request, logger logger.Logger, err error) {
	apiError := mapError(err)
	if apiError.Status >= 500 {
		logger.Error(err.Error())
	} else {
		logger.Warn(err.Error())
	}

	writeError(r.Context(), w, apiError.Message, apiError.Status)
}

// respondSuccess записывает JSON ответа
func respondSuccess(w http.ResponseWriter, r *http.Request, message string, logger logger.Logger, data any) {

	apiResponse := dto.APIResponse{
		Status:    "success",
		Message:   message,
		RequestID: getRequestID(r.Context()),
		Data:      data,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(apiResponse); err != nil {
		logger.Error(err.Error())
	}
}
