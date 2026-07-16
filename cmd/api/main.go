// package main - главный пакет приложения
package main

import (
	"context"
	"fleettrack/internal/app"
	"fleettrack/internal/config"
	"fleettrack/internal/logger"
	"fleettrack/internal/model"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// main - точка сбора приложения
func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	logger := logger.NewStdLogger(logger.DebugLevel)

	cfg, err := config.Load()
	if err != nil {
		logger.Error(model.ErrConnectingDB.Error())
		return
	}

	fleetApp, err := app.New(*cfg)
	if err != nil {
		logger.Error(err.Error())
		return
	}

	defer fleetApp.Close()

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
	logger.Info("Served successfully stopped")
}
