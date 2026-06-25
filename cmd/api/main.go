package main

import (
	"fleettrack/internal/handler"
	"fleettrack/internal/logger"
	"fleettrack/internal/middleware"
	"fleettrack/internal/repository"
	"fleettrack/internal/service"
	"net/http"
)

func main() {
	logger := logger.NewStdLogger(logger.DebugLevel)
	repo := repository.NewMemoryTelemetryRepository(logger)
	service := service.NewTelemetryService(repo)
	handler := handler.NewTelemetryHandler(service, logger)

	router := http.NewServeMux()

	router.HandleFunc("/telemetry", handler.HandleTelemetry)

	err := http.ListenAndServe(":8080", middleware.RequestID(router))
	if err != nil {
		panic(err)
	}
}
