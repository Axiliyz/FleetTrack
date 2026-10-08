// Package app собирает приложение: конфигурацию, подключение к БД,
// репозитории, сервисы, хендлеры и HTTP-серверы
package app

import (
	"context"
	"fleettrack/internal/config"
	"fleettrack/internal/database"
	"fleettrack/internal/handler"
	"fleettrack/internal/logger"
	"fleettrack/internal/metrics"
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
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// stopper - фоновый воркер, который останавливается по отмене контекста и ждёт завершения в Stop
type stopper interface {
	Stop()
}

// App хранит собранные зависимости запущенного приложения
type App struct {
	// Server обслуживает публичное API
	Server *http.Server
	// MetricsServer отдаёт метрики Prometheus на отдельном внутреннем порту
	MetricsServer *http.Server

	db               *pgxpool.Pool
	logger           logger.Logger
	telemetryService *service.TelemetryService
	alertService     *service.AlertService
	periodicWorkers  []stopper

	cancelWorkers context.CancelFunc
	cancelBatcher context.CancelFunc
	cancelAlerts  context.CancelFunc
}

// New собирает приложение: подключается к БД, создаёт репозитории,
// сервисы, хендлеры, HTTP-серверы и запускает фоновые воркеры
//
// Фоновые компоненты получают собственные контексты, а не ctx запуска: они должны
// жить до вызова Close.
//
//nolint:contextcheck
func New(ctx context.Context, cfg config.Config, log logger.Logger) (*App, error) {
	pool, err := database.NewPostgresPool(ctx, cfg.DB.DSN(), cfg.DB.MaxConns)
	if err != nil {
		return nil, err
	}

	repoFactory := factory.NewPostgresRepositoryFactory()
	txManager := transaction.NewPostgresTransactionManager(pool, transaction.WithObserver(metrics.RecordDBTxPhase))

	deviceRepo := postgres.NewPostgresDeviceRepository(pool)
	vehicleRepo := postgres.NewPostgresVehicleRepository(pool)
	assignmentRepo := postgres.NewPostgresAssignmentRepository(pool)
	telemetryRepo := postgres.NewPostgresTelemetryRepository(pool)
	orgRepo := postgres.NewPostgresOrgRepository(pool)
	tripRepo := postgres.NewPostgresTripRepository(pool)
	driverRepo := postgres.NewPostgresDriverRepository(pool)
	userRepo := postgres.NewPostgresUserRepository(pool)
	refreshTokenRepo := postgres.NewPostgresRefreshTokenRepository(pool)
	alertRepo := postgres.NewPostgresAlertRepository(pool)
	alertRuleRepo := postgres.NewPostgresAlertRuleRepository(pool)
	channelRepo := postgres.NewUserNotificationChannelRepository(pool)
	notificationRepo := postgres.NewPostgresAlertNotificationRepository(pool)
	partitionRepo := postgres.NewPostgresPartitionRepository(pool)

	alertService := service.NewAlertService(
		alertRepo, alertRuleRepo, channelRepo, notificationRepo,
		txManager, repoFactory, log, cfg.Workers.AlertQueueSize,
	)
	metrics.RegisterAlertQueueLength(func() float64 { return float64(alertService.QueueLength()) })

	telemetryService := service.NewTelemetryService(
		telemetryRepo, log, txManager, repoFactory, service.NewMotionServiceImpl(), alertService,
		service.TelemetryBatchConfig{
			BufferSize: cfg.Workers.TelemetryBufferSize,
			MaxSize:    cfg.Workers.TelemetryBatchSize,
			MaxWait:    cfg.Workers.TelemetryBatchWait,
		},
	)

	assignmentService := service.NewAssignmentService(assignmentRepo, deviceRepo, vehicleRepo, txManager, repoFactory)
	deviceService := service.NewDeviceService(deviceRepo, log, txManager, repoFactory)
	vehicleService := service.NewVehicleService(vehicleRepo, log)
	orgService := service.NewOrgService(orgRepo, log)
	tripService := service.NewTripService(tripRepo, log)
	driverService := service.NewDriverService(driverRepo, log)
	userService := service.NewUserService(userRepo, log)
	jwtService := service.NewJWTService(cfg.JWT.Secret, cfg.JWT.TTL)
	authService := service.NewAuthService(userRepo, refreshTokenRepo, jwtService, cfg.JWT.RefreshTTL, log, txManager, repoFactory)
	healthService := service.NewHealthService(pool)

	dispatcher := notifier.NewDispatcher(
		notifier.NewTelegramSender(cfg.Telegram.BotToken, nil),
		notifier.NewEmailSender(cfg.SMTP),
		log,
	)

	a := &App{
		db:               pool,
		logger:           log,
		telemetryService: telemetryService,
		alertService:     alertService,
	}

	// Контексты разделены, чтобы при остановке гасить компоненты по очереди:
	// сначала периодические воркеры, затем батчер телеметрии, затем очередь алертов.
	alertsCtx, cancelAlerts := context.WithCancel(context.Background())
	batcherCtx, cancelBatcher := context.WithCancel(context.Background())
	workerCtx, cancelWorkers := context.WithCancel(context.Background())
	a.cancelAlerts, a.cancelBatcher, a.cancelWorkers = cancelAlerts, cancelBatcher, cancelWorkers

	alertService.Start(alertsCtx, cfg.Workers.AlertWorkers)
	telemetryService.StartBatcher(batcherCtx)

	notificationWorker := worker.NewNotificationWorker(notificationRepo, dispatcher, log, 2*time.Second, 50)
	telegramWorker := worker.NewTelegramWorker(cfg.Telegram.BotToken, alertService, channelRepo, nil, log)
	offlineWorker := worker.NewOfflineWorker(alertRepo, alertRuleRepo, alertService, log, 30*time.Second)
	partitionWorker := worker.NewPartitionWorker(partitionRepo, log, 24*time.Hour, 3)

	notificationWorker.Start(workerCtx)
	telegramWorker.Start(workerCtx)
	offlineWorker.Start(workerCtx)
	partitionWorker.Start(workerCtx)
	a.periodicWorkers = []stopper{notificationWorker, telegramWorker, offlineWorker, partitionWorker}

	apiRouter := router.NewRouter(
		handler.NewTelemetryHandler(telemetryService, log),
		handler.NewVehicleHandler(vehicleService, log),
		handler.NewAssignmentHandler(assignmentService, log),
		handler.NewDeviceHandler(deviceService, log),
		handler.NewOrgHandler(orgService, log),
		handler.NewTripHandler(tripService, log),
		handler.NewDriverHandler(driverService, log),
		handler.NewUserHandler(userService, log),
		handler.NewAuthHandler(authService, log),
		jwtService,
		handler.NewAlertRuleHandler(alertService, log),
		handler.NewHealthHandler(healthService, log),
		cfg.API.RequestTimeout,
		log,
	)

	a.Server = &http.Server{
		Addr:              ":" + cfg.API.Port,
		Handler:           apiRouter,
		ReadTimeout:       10 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	metricsMux := http.NewServeMux()
	metricsMux.Handle("/metrics", promhttp.Handler())
	a.MetricsServer = &http.Server{
		Addr:              ":" + cfg.API.MetricsPort,
		Handler:           metricsMux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Info("background workers started: alerts, telemetry batcher, notifications, telegram, offline, partitions")
	return a, nil
}

// Close останавливает фоновую обработку и закрывает пул соединений.
// Вызывать после остановки HTTP-серверов, чтобы новые запросы уже не приходили.
//
// Порядок важен: периодические воркеры -> батчер телеметрии (дописывает буфер) ->
// очередь алертов (дочитывается до конца) -> пул соединений.
func (a *App) Close() {
	a.cancelWorkers()
	for _, w := range a.periodicWorkers {
		w.Stop()
	}

	a.cancelBatcher()
	a.telemetryService.WaitBatcher()

	a.alertService.Stop()
	a.cancelAlerts()

	a.db.Close()
	a.logger.Info("application stopped")
}
