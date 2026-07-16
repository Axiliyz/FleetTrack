package app

import (
	"context"
	"fleettrack/internal/config"
	"fleettrack/internal/database"
	"fleettrack/internal/handler"
	"fleettrack/internal/logger"
	"fleettrack/internal/repository/postgres"
	"fleettrack/internal/router"
	"fleettrack/internal/service"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
)

type App struct {
	Server *http.Server
	DB     *pgxpool.Pool
}

func New(cfg config.Config) (*App, error) {
	ctx := context.Background()
	logger := logger.NewStdLogger(logger.DebugLevel)
	pool, err := database.NewPostgresPool(ctx, cfg.DB.DSN())
	if err != nil {
		logger.Error(err.Error())
		return nil, err
	}

	telemetryRepo := postgres.NewPostgresTelemetryRepository(pool)
	telemetryService := service.NewTelemetryService(telemetryRepo, logger)
	telemetryHandler := handler.NewTelemetryHandler(telemetryService, logger)

	vehicleRepo := postgres.NewPostgresVehicleRepository(pool)
	vehicleService := service.NewVehicleService(vehicleRepo, logger)
	vehicleHandler := handler.NewVehicleHandler(vehicleService, logger)

	router := router.NewRouter(telemetryHandler, vehicleHandler, logger)

	srv := &http.Server{
		Addr:    ":" + cfg.API.Port,
		Handler: router,
	}

	return &App{
		Server: srv,
		DB:     pool,
	}, nil
}

func (a *App) Close() {
	a.DB.Close()
}
