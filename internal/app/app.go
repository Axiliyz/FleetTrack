// Package app собирает приложение: конфигурацию, подключение к БД,
// репозитории, сервисы, хендлеры и HTTP-сервер
package app

import (
	"context"
	"fleettrack/internal/config"
	"fleettrack/internal/database"
	"fleettrack/internal/handler"
	"fleettrack/internal/logger"
	"fleettrack/internal/notifier"
	"fleettrack/internal/repository/factory"
	"fleettrack/internal/repository/postgres"
	"fleettrack/internal/router"
	"fleettrack/internal/service"
	"fleettrack/internal/transaction"
	"fleettrack/internal/worker"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// App хранит собранные зависимости запущенного приложения
type App struct {
	Server             *http.Server
	DB                 *pgxpool.Pool
	AlertService       *service.AlertService
	NotificationWorker *worker.NotificationWorker
	cancel             context.CancelFunc
}

// New собирает приложение: подключается к БД, создаёт репозитории,
// сервисы, хендлеры и HTTP-роутер
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

	alertRepo := postgres.NewPostgresAlertRepository(pool)
	alertRuleRepo := postgres.NewPostgresAlertRuleRepository(pool)
	channelRepo := postgres.NewUserNotificationChannelRepository(pool)
	notificationRepo := postgres.NewPostgresAlertNotificationRepository(pool)
	alertService := service.NewAlertService(
		alertRepo,
		alertRuleRepo,
		channelRepo,
		notificationRepo,
		txManager,
		repoFactory,
		logger,
	)
	alertRuleHandler := handler.NewAlertRuleHandler(alertService, logger)

	workerCtx, cancel := context.WithCancel(context.Background())
	alertService.Start(workerCtx, 4)

	motionService := service.NewMotionServiceImpl()
	telemetryRepo := postgres.NewPostgresTelemetryRepository(pool)
	telemetryService := service.NewTelemetryService(telemetryRepo, logger, txManager, repoFactory, motionService, alertService)
	telemetryHandler := handler.NewTelemetryHandler(telemetryService, logger)

	vehicleRepo := postgres.NewPostgresVehicleRepository(pool)
	vehicleService := service.NewVehicleService(vehicleRepo, logger)
	vehicleHandler := handler.NewVehicleHandler(vehicleService, logger)

	orgRepo := postgres.NewPostgresOrgRepository(pool)
	orgService := service.NewOrgService(orgRepo, logger)
	orgHandler := handler.NewOrgHandler(orgService, logger)

	tripRepo := postgres.NewPostgresTripRepository(pool)
	tripService := service.NewTripService(tripRepo, logger)
	tripHandler := handler.NewTripHandler(tripService, logger)

	driverRepo := postgres.NewPostgresDriverRepository(pool)
	driverService := service.NewDriverService(driverRepo, logger)
	driverHandler := handler.NewDriverHandler(driverService, logger)

	userRepo := postgres.NewPostgresUserRepository(pool)
	userService := service.NewUserService(userRepo, logger)
	userHandler := handler.NewUserHandler(userService, logger)

	refreshTokenRepo := postgres.NewPostgresRefreshTokenRepository(pool)
	jwtService := service.NewJWTService(cfg.JWT.Secret, cfg.JWT.TTL)
	authService := service.NewAuthService(userRepo, refreshTokenRepo, jwtService, cfg.JWT.RefreshTTL, logger, txManager, repoFactory)
	authHandler := handler.NewAuthHandler(authService, logger)

	telegramSender := notifier.NewTelegramSender(cfg.Telegram.BotToken, nil)
	emailSender := notifier.NewEmailSender(cfg.SMTP)
	dispatcher := notifier.NewDispatcher(telegramSender, emailSender, logger)

	notificationWorker := worker.NewNotificationWorker(
		notificationRepo,
		dispatcher,
		logger,
		2*time.Second,
		50,
	)
	notificationWorker.Start(workerCtx)

	router := router.NewRouter(telemetryHandler, vehicleHandler, assignmentHandler, deviceHandler, orgHandler, tripHandler, driverHandler, userHandler, authHandler, jwtService, alertRuleHandler, logger)

	srv := &http.Server{
		Addr:              ":" + cfg.API.Port,
		Handler:           router,
		ReadTimeout:       10 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	return &App{
		Server:             srv,
		DB:                 pool,
		AlertService:       alertService,
		NotificationWorker: notificationWorker,
		cancel:             cancel,
	}, nil
}

// Close закрывает пул соединений с БД и останавливает фоновые воркеры
func (a *App) Close() {
	if a.NotificationWorker != nil {
		a.NotificationWorker.Stop()
	}
	if a.AlertService != nil {
		a.AlertService.Stop()
	}
	if a.cancel != nil {
		a.cancel()
	}
	a.DB.Close()
}
