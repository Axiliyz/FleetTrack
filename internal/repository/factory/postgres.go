package factory

import (
	"fleettrack/internal/database"
	"fleettrack/internal/repository"
	"fleettrack/internal/repository/postgres"
)

type Repositories struct {
	Device     repository.DeviceRepository
	Vehicle    repository.VehicleRepository
	Assignment repository.AssignmentRepository
}

type PostgresRepositoryFactory struct{}

func NewPostgresRepositoryFactory() *PostgresRepositoryFactory {
	return &PostgresRepositoryFactory{}
}

func (f *PostgresRepositoryFactory) New(tx database.DBTX) Repositories {
	return Repositories{
		Device:     postgres.NewPostgresDeviceRepository(tx),
		Vehicle:    postgres.NewPostgresVehicleRepository(tx),
		Assignment: postgres.NewPostgresAssignmentRepository(tx),
	}
}
