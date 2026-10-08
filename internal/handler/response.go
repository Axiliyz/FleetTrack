package handler

import (
	"fleettrack/internal/httpresp"
	"fleettrack/internal/logger"
	"net/http"
)

// respondError переводит ошибку в HTTP-статус, логирует её и отправляет ответ
func respondError(w http.ResponseWriter, r *http.Request, log logger.Logger, err error) {
	apiError := mapError(err)
	if apiError.Status >= http.StatusInternalServerError {
		log.Error(err.Error())
	} else {
		log.Warn(err.Error())
	}
	httpresp.Error(w, r, log, apiError.Status, apiError.Message)
}

// respondSuccess отправляет успешный ответ со статусом 200
func respondSuccess(w http.ResponseWriter, r *http.Request, message string, log logger.Logger, data any) {
	httpresp.Success(w, r, log, http.StatusOK, message, data)
}

// respondCreated отправляет успешный ответ со статусом 201
func respondCreated(w http.ResponseWriter, r *http.Request, message string, log logger.Logger, data any) {
	httpresp.Success(w, r, log, http.StatusCreated, message, data)
}
