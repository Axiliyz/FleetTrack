package main

import (
	"context"
	"fleettrack/internal/handler"
	"fleettrack/internal/logger"
	"fleettrack/internal/middleware"
	"fleettrack/internal/repository"
	"fleettrack/internal/service"
	"fmt"
	"net/http"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {

	dsn := "postgres://" + os.Getenv("DB_USER") + ":" + os.Getenv("DB_PASSWORD") + "@postgres:5432/" + os.Getenv("DB_NAME")
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		fmt.Println("Ошибка", err)
	}
	defer pool.Close()

	logger := logger.NewStdLogger(logger.DebugLevel)
	repo := repository.NewPostgresTelemetryRepository(pool)
	service := service.NewTelemetryService(repo, logger)
	handler := handler.NewTelemetryHandler(service, logger)

	router := http.NewServeMux()

	router.HandleFunc("/telemetry", handler.HandleTelemetry)

	err = http.ListenAndServe(":8080", middleware.RequestID(router))
	if err != nil {
		panic(err)
	}
}
