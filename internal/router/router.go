package router

import (
	"fleettrack/internal/config"
	"fleettrack/internal/handler"
	"fleettrack/internal/logger"
	"fleettrack/internal/middleware"
	"fleettrack/internal/model"
	"net/http"

	"github.com/go-chi/chi/v5"
)

func NewRouter(telemetryHandler *handler.TelemetryHandler, vehicleHandler *handler.VehicleHandler, logger logger.Logger) http.Handler {
	router := chi.NewRouter()
	router.Use(middleware.RequestID)
	router.Use(middleware.TimeoutMiddleware(config.RequestTimeout))
	router.MethodNotAllowed((func(w http.ResponseWriter, r *http.Request) {
		logger.Error(model.ErrInvalidMethod.Error())
		w.WriteHeader(http.StatusMethodNotAllowed)
	}))
	router.Post("/telemetry", telemetryHandler.HandleTelemetry)
	router.Get("/telemetry", telemetryHandler.HandleGetListTelemetry)
	router.Get("/telemetry/vehicles/{id}", telemetryHandler.HandleGetTelemetryByVehicle)
	router.Get("/telemetry/{id}", telemetryHandler.HandleGetTelemetryByID)
	router.Delete("/telemetry/{id}", telemetryHandler.HandleDeleteTelemetryByID)
	router.Delete("/telemetry/vehicles/{id}", telemetryHandler.HandleDeleteTelemetryByVehicleID)

	router.Post("/vehicles", vehicleHandler.HandlePostVehicle)
	router.Get("/vehicles", vehicleHandler.HandleGetListVehicle)
	router.Get("/vehicles/{id}", vehicleHandler.HandleGetVehicleByID)
	router.Delete("/vehicles/{id}", vehicleHandler.HandleDeleteVehicle)
	router.Patch("/vehicles/{id}", vehicleHandler.HandlePatchVehicle)

	return router
}
