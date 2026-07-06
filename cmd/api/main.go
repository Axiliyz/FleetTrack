package main

import (
	"context"
	"fleettrack/internal/config"
	"fleettrack/internal/handler"
	"fleettrack/internal/logger"
	"fleettrack/internal/middleware"
	"fleettrack/internal/model"
	"fleettrack/internal/repository/postgres"
	"fleettrack/internal/service"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	logger := logger.NewStdLogger(logger.DebugLevel)

	cfg, err := config.Load()
	if err != nil {
		logger.Error(model.ErrConnectingDB.Error())
		return
	}

	pool, err := pgxpool.New(context.Background(), cfg.DB.DSN())
	if err != nil {
		logger.Error(err.Error())
		return
	}
	defer pool.Close()

	telemetryRepo := postgres.NewPostgresTelemetryRepository(pool)
	telemetryService := service.NewTelemetryService(telemetryRepo, logger)
	telemetryHandler := handler.NewTelemetryHandler(telemetryService, logger)

	vehicleRepo := postgres.NewPostgresVehicleRepository(pool)
	vehicleService := service.NewVehicleService(vehicleRepo, logger)
	vehicleHandler := handler.NewVehicleHandler(vehicleService, logger)

	router := chi.NewRouter()
	router.Use(middleware.RequestID)
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

	srv := &http.Server{
		Addr:    ":" + cfg.API.Port,
		Handler: router,
	}
	err = srv.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		logger.Error(err.Error())
		return
	}

}
