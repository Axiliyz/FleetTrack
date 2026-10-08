// package main - главный пакет приложения
package main

import (
	"context"
	"errors"
	"fleettrack/internal/app"
	"fleettrack/internal/config"
	"fleettrack/internal/logger"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// @title FleetTrack API
// @version 1.0
// @description Бэкенд системы мониторинга и телематики автопарка
// @host localhost:8080
// @BasePath /
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Введите: Bearer <access_token>
// main - точка входа: запускает приложение и завершает процесс с кодом из run
func main() {
	os.Exit(run())
}

func run() int {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load()
	if err != nil {
		logger.NewStdLogger(logger.ErrorLevel).Error("load config: " + err.Error())
		return 1
	}
	log := logger.NewStdLogger(cfg.Log.Level)

	fleetApp, err := app.New(ctx, *cfg, log)
	if err != nil {
		log.Error("init app: " + err.Error())
		return 1
	}
	defer fleetApp.Close()

	serverErr := make(chan error, 2)
	serve := func(name string, srv *http.Server) {
		log.Info(fmt.Sprintf("starting %s server on %s", name, srv.Addr))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- fmt.Errorf("%s server: %w", name, err)
		}
	}
	go serve("api", fleetApp.Server)
	go serve("metrics", fleetApp.MetricsServer)

	exitCode := 0
	select {
	case <-ctx.Done():
		log.Info("got signal to shutdown")
	case err := <-serverErr:
		log.Error(err.Error())
		exitCode = 1
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	for _, srv := range []*http.Server{fleetApp.Server, fleetApp.MetricsServer} {
		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.Error("shutdown " + srv.Addr + ": " + err.Error())
		}
	}
	log.Info("servers stopped")
	return exitCode
}
