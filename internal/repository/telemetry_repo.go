// Package repository содержит логику сохранения данных
package repository

import (
	"context"
	"errors"
	"fleettrack/internal/model"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// MemoryTelemetryRepository позволяет сохранять данные в память
type MemoryTelemetryRepository struct {
	telemetry map[int]model.Telemetry
	byVehicle map[int][]model.Telemetry
	current   map[int]model.Telemetry
	nextID    int
}

// PostgresTelemetryRepository позволяет сохранять данные в PostgreSQL
type PostgresTelemetryRepository struct {
	pool *pgxpool.Pool
}

// NewMemoryTelemetryRepository создаёт новый репозиторий для сохранения в память
func NewMemoryTelemetryRepository() *MemoryTelemetryRepository {
	return &MemoryTelemetryRepository{
		telemetry: make(map[int]model.Telemetry),
		byVehicle: make(map[int][]model.Telemetry),
		current:   make(map[int]model.Telemetry),
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
	r.nextID++
	t.TelemetryID = r.nextID
	r.telemetry[t.TelemetryID] = *t
	r.byVehicle[t.VehicleID] = append(r.byVehicle[t.VehicleID], *t)
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

// GetList для PostgresTelemetryRepository возвращает полный список телеметрии
func (r *PostgresTelemetryRepository) GetList(ctx context.Context, limit int) ([]model.Telemetry, error) {
	query := `SELECT id, device_id, vehicle_id, latitude, longitude, fuel, received_at, device_timestamp
		FROM telemetry ORDER BY received_at DESC LIMIT $1`
	rows, err := r.pool.Query(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var telemetries []model.Telemetry
	for rows.Next() {
		var t model.Telemetry
		err = rows.Scan(&t.TelemetryID, &t.DeviceID, &t.VehicleID, &t.Lat, &t.Lon, &t.Fuel, &t.ReceivedAt, &t.DeviceTimestamp)
		if err != nil {
			return nil, err
		}
		telemetries = append(telemetries, t)
	}
	return telemetries, rows.Err()
}

// GetList для MemoryTelemetryRepository возвращает полный список телеметрии
func (r *MemoryTelemetryRepository) GetList(ctx context.Context, limit int) ([]model.Telemetry, error) {
	res := make([]model.Telemetry, 0, limit)
	count := 0
	for _, t := range r.telemetry {
		if count >= limit {
			break
		}
		res = append(res, t)
		count++
	}
	return res, nil
}

// GetItemByID для MemoryTelemetryRepository возвращает конкретную запись телеметрии по её ID
func (r *MemoryTelemetryRepository) GetItemByID(ctx context.Context, id int) (model.Telemetry, error) {
	res, ok := r.telemetry[id]
	if !ok {
		return model.Telemetry{}, model.ErrNotFound
	}
	return res, nil
}

// GetItemByID для PostgresTelemetryRepository возвращает конкретную запись телеметрии по её ID
func (r *PostgresTelemetryRepository) GetItemByID(ctx context.Context, id int) (model.Telemetry, error) {
	query := `SELECT id, organization_id, device_id, vehicle_id, latitude, longitude, fuel, received_at, device_timestamp FROM telemetry WHERE id = $1`
	var t model.Telemetry
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&t.TelemetryID, &t.OrganizationID, &t.DeviceID,
		&t.VehicleID, &t.Lat, &t.Lon, &t.Fuel,
		&t.ReceivedAt, &t.DeviceTimestamp,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.Telemetry{}, model.ErrNotFound
		}
		return model.Telemetry{}, err
	}
	return t, nil
}

// GetListByVehicle для MemoryTelemetryRepository возвращает всю телеметрию для конкретной машины
func (r *MemoryTelemetryRepository) GetListByVehicle(ctx context.Context, id int) ([]model.Telemetry, error) {
	res, ok := r.byVehicle[id]
	if !ok {
		return []model.Telemetry{}, model.ErrNotFound
	}
	return res, nil
}

// GetListByVehicle для PostgresTelemetryRepository возвращает всю телеметрию для конкретной машины
func (r *PostgresTelemetryRepository) GetListByVehicle(ctx context.Context, id int) ([]model.Telemetry, error) {
	query := `SELECT id, organization_id, vehicle_id, device_id, latitude, longitude, fuel, received_at, device_timestamp FROM telemetry WHERE vehicle_id = $1`
	rows, err := r.pool.Query(ctx, query, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var telemetries []model.Telemetry
	for rows.Next() {
		var t model.Telemetry
		err = rows.Scan(
			&t.TelemetryID, &t.OrganizationID, &t.VehicleID, &t.DeviceID,
			&t.Lat, &t.Lon, &t.Fuel,
			&t.ReceivedAt, &t.DeviceTimestamp,
		)
		if err != nil {
			return nil, err
		}
		telemetries = append(telemetries, t)
	}
	if len(telemetries) == 0 {
		return nil, model.ErrNotFound
	}
	return telemetries, rows.Err()
}

// DeleteListByVehicle для MemoryTelemetryRepository удаляет всю телеметрию для конкретной машины
func (r *MemoryTelemetryRepository) DeleteListByVehicle(ctx context.Context, id int) ([]model.Telemetry, error) {
	deleted, ok := r.byVehicle[id]
	if !ok {
		return nil, model.ErrNotFound
	}
	delete(r.byVehicle, id)
	delete(r.current, id)
	for _, t := range deleted {
		delete(r.telemetry, t.TelemetryID)
	}
	return deleted, nil
}

// DeleteListByVehicle для PostgresTelemetryRepository удаляет всю телеметрию для конкретной машины
func (r *PostgresTelemetryRepository) DeleteListByVehicle(ctx context.Context, id int) ([]model.Telemetry, error) {
	query := `DELETE FROM telemetry WHERE vehicle_id = $1
		RETURNING id, organization_id, vehicle_id, device_id, latitude, longitude, fuel, received_at, device_timestamp`
	rows, err := r.pool.Query(ctx, query, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var telemetries []model.Telemetry
	for rows.Next() {
		var t model.Telemetry
		err = rows.Scan(
			&t.TelemetryID, &t.OrganizationID, &t.VehicleID, &t.DeviceID,
			&t.Lat, &t.Lon, &t.Fuel,
			&t.ReceivedAt, &t.DeviceTimestamp,
		)
		if err != nil {
			return nil, err
		}
		telemetries = append(telemetries, t)
	}
	if len(telemetries) == 0 {
		return nil, model.ErrNotFound
	}
	return telemetries, rows.Err()
}

// DeleteItemByID для MemoryTelemetryRepository удаляет телеметрию по её ID
func (r *MemoryTelemetryRepository) DeleteItemByID(ctx context.Context, id int) (model.Telemetry, error) {
	t, ok := r.telemetry[id]
	if !ok {
		return model.Telemetry{}, model.ErrNotFound
	}
	delete(r.telemetry, id)

	list := r.byVehicle[t.VehicleID]
	for i, item := range list {
		if item.TelemetryID == id {
			r.byVehicle[t.VehicleID] = append(list[:i], list[i+1:]...)
			break
		}
	}

	if r.current[t.VehicleID].TelemetryID == id {
		delete(r.current, t.VehicleID)
	}
	return t, nil
}

// DeleteItemByID для PostgresTelemetryRepository удаляет телеметрию по её ID
func (r *PostgresTelemetryRepository) DeleteItemByID(ctx context.Context, id int) (model.Telemetry, error) {
	query := `DELETE FROM telemetry WHERE id = $1
		RETURNING id, organization_id, vehicle_id, device_id, latitude, longitude, fuel, received_at, device_timestamp`
	var t model.Telemetry
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&t.TelemetryID, &t.OrganizationID, &t.VehicleID, &t.DeviceID,
		&t.Lat, &t.Lon, &t.Fuel,
		&t.ReceivedAt, &t.DeviceTimestamp,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.Telemetry{}, model.ErrNotFound
		}
		return model.Telemetry{}, err
	}
	return t, nil
}
