// Package router собирает HTTP-роуты приложения и подключает middleware.
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

// NewRouter собирает HTTP-роутер приложения и подключает middleware
func NewRouter(telemetryHandler *handler.TelemetryHandler, vehicleHandler *handler.VehicleHandler, assignmentHandler *handler.AssignmentHandler, deviceHandler *handler.DeviceHandler, orgHandler *handler.OrgHandler, logger logger.Logger) http.Handler {
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

	router.Post("/assignments", assignmentHandler.HandlePostAssignment)

	router.Post("/devices", deviceHandler.HandlePostDevice)
	router.Get("/devices/{id}", deviceHandler.HandleGetDeviceByID)
	router.Delete("/devices/{id}", deviceHandler.HandleDeleteDeviceByID)

	router.Post("/organizations", orgHandler.HandlePostOrg)
	return router
}
