// Package router собирает HTTP-роуты приложения и подключает middleware.
package router

import (
	"fleettrack/internal/config"
	"fleettrack/internal/handler"
	"fleettrack/internal/logger"
	"fleettrack/internal/middleware"
	"fleettrack/internal/model"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

// NewRouter собирает HTTP-роутер приложения и подключает middleware
func NewRouter(
	telemetryHandler *handler.TelemetryHandler, vehicleHandler *handler.VehicleHandler,
	assignmentHandler *handler.AssignmentHandler, deviceHandler *handler.DeviceHandler,
	orgHandler *handler.OrgHandler, tripHandler *handler.TripHandler,
	driverHandler *handler.DriverHandler, userHandler *handler.UserHandler,
	authHandler *handler.AuthHandler, jwtParser middleware.JWTParser, alertRuleHandler *handler.AlertRuleHandler, logger logger.Logger) http.Handler {
	router := chi.NewRouter()
	router.Use(middleware.RequestID)
	router.Use(middleware.Recovery(logger))
	router.Use(middleware.LogQuery(logger))
	router.Use(middleware.TimeoutMiddleware(config.RequestTimeout))

	router.MethodNotAllowed((func(w http.ResponseWriter, r *http.Request) {
		logger.Error(model.ErrInvalidMethod.Error())
		w.WriteHeader(http.StatusMethodNotAllowed)
	}))

	authLimiter := middleware.RateLimit(5, time.Minute, logger)

	router.With(authLimiter).Post("/login", authHandler.HandleLogin)
	router.With(authLimiter).Post("/register", authHandler.HandleRegisterCompany)
	router.Post("/refresh", authHandler.HandleRefresh)
	router.Post("/logout", authHandler.HandleLogout)

	router.Get("/panic", func(w http.ResponseWriter, r *http.Request) {
		panic("panica")
	})

	router.Group(func(r chi.Router) {
		r.Use(middleware.AuthMiddleware(jwtParser, logger))

		manageFleet := middleware.RequireRole(logger, model.UserRoleAdmin, model.UserRoleDispatcher)

		r.Post("/telemetry", telemetryHandler.HandleTelemetry)
		r.Get("/telemetry", telemetryHandler.HandleGetListTelemetry)
		r.Get("/telemetry/vehicles/{id}", telemetryHandler.HandleGetTelemetryByVehicle)
		r.Get("/telemetry/{id}", telemetryHandler.HandleGetTelemetryByID)
		r.With(manageFleet).Delete("/telemetry/{id}", telemetryHandler.HandleDeleteTelemetryByID)
		r.With(manageFleet).Delete("/telemetry/vehicles/{id}", telemetryHandler.HandleDeleteTelemetryByVehicleID)

		r.Get("/vehicles", vehicleHandler.HandleGetListVehicle)
		r.Get("/vehicles/{id}", vehicleHandler.HandleGetVehicleByID)

		r.With(manageFleet).Post("/vehicles", vehicleHandler.HandlePostVehicle)
		r.With(manageFleet).Delete("/vehicles/{id}", vehicleHandler.HandleDeleteVehicle)
		r.With(manageFleet).Patch("/vehicles/{id}", vehicleHandler.HandlePatchVehicle)

		r.With(manageFleet).Post("/assignments", assignmentHandler.HandlePostAssignment)

		r.Get("/devices/{id}", deviceHandler.HandleGetDeviceByID)
		r.With(manageFleet).Post("/devices", deviceHandler.HandlePostDevice)
		r.With(manageFleet).Delete("/devices/{id}", deviceHandler.HandleDeleteDeviceByID)

		r.Get("/organizations", orgHandler.HandleGetListOrg)
		r.With(middleware.RequireRole(logger, model.UserRoleAdmin)).Post("/organizations", orgHandler.HandlePostOrg)

		r.Get("/trips", tripHandler.HandleGetListTrips)
		r.Get("/trips/{id}", tripHandler.HandleGetTripByID)
		r.With(manageFleet).Post("/trips", tripHandler.HandleAssignTrip)
		r.With(manageFleet).Patch("/trips/{id}", tripHandler.HandleUpdateTrip)
		r.With(manageFleet).Delete("/trips/{id}", tripHandler.HandleDeleteTrip)

		r.Get("/drivers", driverHandler.HandleGetListDriver)
		r.Get("/drivers/{id}", driverHandler.HandleGetDriverByID)
		r.With(manageFleet).Post("/drivers", driverHandler.HandlePostDriver)
		r.With(manageFleet).Delete("/drivers/{id}", driverHandler.HandleDeleteDriver)
		r.With(manageFleet).Patch("/drivers/{id}", driverHandler.HandlePatchDriver)

		r.Get("/users", userHandler.HandleGetListUsers)
		r.Get("/users/{id}", userHandler.HandleGetUserByID)

		r.With(middleware.RequireRole(logger, model.UserRoleAdmin)).Post("/users", userHandler.HandleCreateUser)
		r.With(middleware.RequireRole(logger, model.UserRoleAdmin)).Delete("/users/{id}", userHandler.HandleDeleteUserByID)

		// Правила алертов
		r.Get("/alert-rules", alertRuleHandler.HandleGetRulesList)
		r.Get("/alert-rules/{id}", alertRuleHandler.HandleGetRuleByID)
		r.With(manageFleet).Post("/alert-rules", alertRuleHandler.HandlePostRule)
		r.With(manageFleet).Delete("/alert-rules/{id}", alertRuleHandler.HandleDeleteRuleByID)
	})

	return router
}
