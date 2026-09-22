// package main - главный пакет приложения
package main

import (
	"context"
	"fleettrack/internal/app"
	"fleettrack/internal/config"
	"fleettrack/internal/logger"
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
// main - точка сбора приложения
func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	logger := logger.NewStdLogger(logger.DebugLevel)

	cfg, err := config.Load()
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}

	fleetApp, err := app.New(*cfg)
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}

	defer fleetApp.Close()

	logger.Info("Starting HTTP server on port " + cfg.API.Port)
	go func() {
		if err := fleetApp.Server.ListenAndServe(); err != nil &&
			err != http.ErrServerClosed {

			logger.Error(err.Error())
		}
	}()

	<-ctx.Done()

	logger.Info("Got signal to shutdown")
	shutdownCtx, cancel := context.WithTimeout(
		context.Background(), 10*time.Second,
	)

	defer cancel()
	if err := fleetApp.Server.Shutdown(shutdownCtx); err != nil {
		logger.Error(err.Error())
	}
	logger.Info("Server successfully stopped")
}
