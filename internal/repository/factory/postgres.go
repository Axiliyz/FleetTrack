package factory

import (
	"fleettrack/internal/database"
	"fleettrack/internal/repository"
	"fleettrack/internal/repository/postgres"
)

// Repositories собирает репозитории, работающие поверх одного соединения/транзакции
type Repositories struct {
	Device     repository.DeviceRepository
	Vehicle    repository.VehicleRepository
	Assignment repository.AssignmentRepository
	Telemetry  repository.TelemetryRepository
	Trip       repository.TripRepository
}

// PostgresRepositoryFactory - реализация RepositoryFactory поверх PostgreSQL
type PostgresRepositoryFactory struct{}

// NewPostgresRepositoryFactory создаёт новую фабрику репозиториев PostgreSQL
func NewPostgresRepositoryFactory() *PostgresRepositoryFactory {
	return &PostgresRepositoryFactory{}
}

// New создаёт набор PostgreSQL-репозиториев поверх переданного соединения/транзакции
func (f *PostgresRepositoryFactory) New(tx database.DBTX) Repositories {
	return Repositories{
		Device:     postgres.NewPostgresDeviceRepository(tx),
		Vehicle:    postgres.NewPostgresVehicleRepository(tx),
		Assignment: postgres.NewPostgresAssignmentRepository(tx),
		Telemetry:  postgres.NewPostgresTelemetryRepository(tx),
		Trip:       postgres.NewPostgresTripRepository(tx),
	}
}
