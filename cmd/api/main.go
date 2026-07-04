package main

import (
	"context"
	"fleettrack/internal/config"
	"fleettrack/internal/handler"
	"fleettrack/internal/logger"
	"fleettrack/internal/middleware"
	"fleettrack/internal/model"
	"fleettrack/internal/repository"
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

	repo := repository.NewPostgresTelemetryRepository(pool)
	service := service.NewTelemetryService(repo, logger)
	handler := handler.NewTelemetryHandler(service, logger)

	router := chi.NewRouter()
	router.Use(middleware.RequestID)
	router.MethodNotAllowed((func(w http.ResponseWriter, r *http.Request) {
		logger.Error(model.ErrInvalidMethod.Error())
		w.WriteHeader(http.StatusMethodNotAllowed)
	}))
	router.Post("/telemetry", handler.HandleTelemetry)
	router.Get("/telemetry", handler.HandleGetTelemetry)
	router.Get("/telemetry/vehicle/{id}", handler.HandleGetTelemetryByVehicle)
	router.Get("/telemetry/{id}", handler.HandleGetTelemetryByID)
	router.Delete("/telemetry/{id}", handler.HandleDeleteTelemetryByID)
	router.Delete("/telemetry/vehicle/{id}", handler.HandleDeleteTelemetryByVehicleID)

	err = http.ListenAndServe(":"+cfg.API.Port, router)
	if err != nil {
		panic(err)
	}
}
