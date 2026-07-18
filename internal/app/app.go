// Package app собирает приложение: конфигурацию, подключение к БД,
// репозитории, сервисы, хендлеры и HTTP-сервер.
package app

import (
	"context"
	"fleettrack/internal/config"
	"fleettrack/internal/database"
	"fleettrack/internal/handler"
	"fleettrack/internal/logger"
	"fleettrack/internal/repository/factory"
	"fleettrack/internal/repository/postgres"
	"fleettrack/internal/router"
	"fleettrack/internal/service"
	"fleettrack/internal/transaction"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
)

// App хранит собранные зависимости запущенного приложения.
type App struct {
	Server *http.Server
	DB     *pgxpool.Pool
}

// New собирает приложение: подключается к БД, создаёт репозитории,
// сервисы, хендлеры и HTTP-роутер.
func New(cfg config.Config) (*App, error) {
	ctx := context.Background()
	logger := logger.NewStdLogger(logger.DebugLevel)
	pool, err := database.NewPostgresPool(ctx, cfg.DB.DSN())
	if err != nil {
		logger.Error(err.Error())
		return nil, err
	}

	repoFactory := factory.NewPostgresRepositoryFactory()
	txManager := transaction.NewPostgresTransactionManager(pool)
	assignmentService := service.NewAssignmentService(
		postgres.NewPostgresAssignmentRepository(pool),
		postgres.NewPostgresDeviceRepository(pool),
		postgres.NewPostgresVehicleRepository(pool),
		txManager,
		repoFactory,
	)
	assignmentHandler := handler.NewAssignmentHandler(assignmentService, logger)

	deviceRepo := postgres.NewPostgresDeviceRepository(pool)
	deviceService := service.NewDeviceService(deviceRepo, logger, txManager, repoFactory)
	deviceHandler := handler.NewDeviceHandler(deviceService, logger)

	telemetryRepo := postgres.NewPostgresTelemetryRepository(pool)
	telemetryService := service.NewTelemetryService(telemetryRepo, logger)
	telemetryHandler := handler.NewTelemetryHandler(telemetryService, logger)

	vehicleRepo := postgres.NewPostgresVehicleRepository(pool)
	vehicleService := service.NewVehicleService(vehicleRepo, logger)
	vehicleHandler := handler.NewVehicleHandler(vehicleService, logger)

	orgRepo := postgres.NewPostgresOrgRepository(pool)
	orgService := service.NewOrgService(orgRepo, logger)
	orgHandler := handler.NewOrgHandler(orgService, logger)

	router := router.NewRouter(telemetryHandler, vehicleHandler, assignmentHandler, deviceHandler, orgHandler, logger)

	srv := &http.Server{
		Addr:    ":" + cfg.API.Port,
		Handler: router,
	}

	return &App{
		Server: srv,
		DB:     pool,
	}, nil
}

// Close закрывает пул соединений с БД.
func (a *App) Close() {
	a.DB.Close()
}
