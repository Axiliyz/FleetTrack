package repository

import (
	"context"
	"fleettrack/internal/model"

	"github.com/jackc/pgx/v5/pgxpool"
)

type MemoryTelemetryRepository struct {
	history []model.Telemetry
	current map[int]model.Telemetry
}

type PostgresTelemetryRepository struct {
	pool *pgxpool.Pool
}

func NewMemoryTelemetryRepository() *MemoryTelemetryRepository {
	return &MemoryTelemetryRepository{
		history: make([]model.Telemetry, 0),
		current: make(map[int]model.Telemetry),
	}
}

func NewPostgresTelemetryRepository(pool *pgxpool.Pool) *PostgresTelemetryRepository {
	return &PostgresTelemetryRepository{
		pool: pool,
	}
}

func (r *MemoryTelemetryRepository) Save(ctx context.Context, t model.Telemetry) error {
	r.history = append(r.history, t)
	r.current[t.VehicleID] = t
	return nil
}

func (r *PostgresTelemetryRepository) Save(ctx context.Context, t model.Telemetry) error {
	_, err := r.pool.Exec(ctx, "insert into telemetry(organization_id, vehicle_id, device_id, latitude, longitude, fuel) values ($1, $2, $3, $4, $5, $6)", 1, t.VehicleID, t.DeviceID, t.Lat, t.Lon, t.Fuel)
	if err != nil {
		return err
	}

	return nil
}
