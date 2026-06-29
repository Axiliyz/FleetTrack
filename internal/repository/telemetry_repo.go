// Package repository содержит логику сохранения данных
package repository

import (
	"context"
	"fleettrack/internal/model"

	"github.com/jackc/pgx/v5/pgxpool"
)

// MemoryTelemetryRepository позволяет сохранять данные в память
type MemoryTelemetryRepository struct {
	history []model.Telemetry
	current map[int]model.Telemetry
}

// PostgresTelemetryRepository позволяет сохранять данные в PostgreSQL
type PostgresTelemetryRepository struct {
	pool *pgxpool.Pool
}

// NewMemoryTelemetryRepository создаёт новый репозиторий для сохранения в память
func NewMemoryTelemetryRepository() *MemoryTelemetryRepository {
	return &MemoryTelemetryRepository{
		history: make([]model.Telemetry, 0),
		current: make(map[int]model.Telemetry),
	}
}

// NewPostgresTelemetryRepository создаёт репозиторий для сохранения в БД PostgreSQL
func NewPostgresTelemetryRepository(pool *pgxpool.Pool) *PostgresTelemetryRepository {
	return &PostgresTelemetryRepository{
		pool: pool,
	}
}

// Save для MemoryTelemetryRepository сохраняет телеметрию в память
// Возвращает ошибку
func (r *MemoryTelemetryRepository) Save(ctx context.Context, t *model.Telemetry) error {
	r.history = append(r.history, *t)
	r.current[t.VehicleID] = *t
	return nil
}

// Save для PostgresTelemetryRepository сохраняет телеметрию в БД PostgreSQL
// Возвращает ошибку
func (r *PostgresTelemetryRepository) Save(ctx context.Context, t *model.Telemetry) error {
	err := r.pool.QueryRow(ctx,
		"insert into telemetry(organization_id, vehicle_id, device_id, latitude, longitude, fuel) values ($1, $2, $3, $4, $5, $6) RETURNING id",
		1, t.VehicleID, t.DeviceID, t.Lat, t.Lon, t.Fuel,
	).Scan(&t.TelemetryID)
	return err
}
