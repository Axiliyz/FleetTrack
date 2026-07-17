// package postgres отвечает за взаимодействие с БД PostgreSQL
package postgres

import (
	"context"
	"errors"
	"fleettrack/internal/model"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
)

// PostgresTelemetryRepository позволяет сохранять данные в PostgreSQL
type PostgresTelemetryRepository struct {
	db DBTX
}

// NewPostgresTelemetryRepository создаёт репозиторий для сохранения в БД PostgreSQL
func NewPostgresTelemetryRepository(db DBTX) *PostgresTelemetryRepository {
	return &PostgresTelemetryRepository{
		db: db,
	}
}

// Save для PostgresTelemetryRepository сохраняет телеметрию в БД PostgreSQL
// Возвращает ошибку
func (r *PostgresTelemetryRepository) Save(ctx context.Context, t *model.Telemetry) error {
	err := r.db.QueryRow(ctx,
		`INSERT INTO telemetry
		(organization_id, 
		vehicle_id, 
		device_id, 
		latitude, 
		longitude, 
		fuel) 
		VALUES ($1, $2, $3, $4, $5, $6) 
		RETURNING id`,
		1, t.VehicleID, t.DeviceID, t.Lat, t.Lon, t.Fuel,
	).Scan(&t.TelemetryID)
	return err
}

// buildWhereClause берёт фильтр и возвращает готовый кусок WHERE... и срез аргументов
func buildWhereClause(filter model.TelemetryFilter) (string, []any) {
	var conditions []string
	var args []any
	argN := 1
	if filter.OrganizationID != nil {
		conditions = append(conditions, fmt.Sprintf("organization_id = $%d", argN))
		args = append(args, *filter.OrganizationID)
		argN++
	}
	if filter.VehicleID != nil {
		conditions = append(conditions, fmt.Sprintf("vehicle_id = $%d", argN))
		args = append(args, *filter.VehicleID)
		argN++
	}
	if filter.DeviceID != nil {
		conditions = append(conditions, fmt.Sprintf("device_id = $%d", argN))
		args = append(args, *filter.DeviceID)
		argN++
	}
	if filter.FuelMin != nil {
		conditions = append(conditions, fmt.Sprintf("fuel >= $%d", argN))
		args = append(args, *filter.FuelMin)
		argN++
	}
	if filter.FuelMax != nil {
		conditions = append(conditions, fmt.Sprintf("fuel <= $%d", argN))
		args = append(args, *filter.FuelMax)
		argN++
	}
	if filter.LatMin != nil {
		conditions = append(conditions, fmt.Sprintf("latitude >= $%d", argN))
		args = append(args, *filter.LatMin)
		argN++
	}
	if filter.LatMax != nil {
		conditions = append(conditions, fmt.Sprintf("latitude <= $%d", argN))
		args = append(args, *filter.LatMax)
		argN++
	}
	if filter.LonMin != nil {
		conditions = append(conditions, fmt.Sprintf("longitude >= $%d", argN))
		args = append(args, *filter.LonMin)
		argN++
	}
	if filter.LonMax != nil {
		conditions = append(conditions, fmt.Sprintf("longitude <= $%d", argN))
		args = append(args, *filter.LonMax)
		argN++
	}
	if filter.From != nil {
		conditions = append(conditions, fmt.Sprintf("received_at >= $%d", argN))
		args = append(args, *filter.From)
		argN++
	}
	if filter.To != nil {
		conditions = append(conditions, fmt.Sprintf("received_at <= $%d", argN))
		args = append(args, *filter.To)
		argN++
	}
	if len(conditions) < 1 {
		return "", args
	}
	return "WHERE " + strings.Join(conditions, " AND "), args
}

// GetList для PostgresTelemetryRepository возвращает полный список телеметрии
func (r *PostgresTelemetryRepository) GetList(ctx context.Context, filter model.TelemetryFilter) ([]model.Telemetry, error) {
	whereClause, args := buildWhereClause(filter)
	query := fmt.Sprintf(`SELECT id, device_id, vehicle_id, latitude, longitude, fuel, received_at, device_timestamp
		FROM telemetry %s ORDER BY received_at DESC LIMIT $%d OFFSET $%d`, whereClause, len(args)+1, len(args)+2)
	args = append(args, filter.Limit, filter.Offset)
	rows, err := r.db.Query(ctx, query, args...)
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

// GetItemByID для PostgresTelemetryRepository возвращает конкретную запись телеметрии по её ID
func (r *PostgresTelemetryRepository) GetItemByID(ctx context.Context, id int) (model.Telemetry, error) {
	query := `SELECT id, organization_id, device_id, vehicle_id, latitude, longitude, fuel, received_at, device_timestamp FROM telemetry WHERE id = $1`
	var t model.Telemetry
	err := r.db.QueryRow(ctx, query, id).Scan(
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

// GetListByVehicle для PostgresTelemetryRepository возвращает всю телеметрию для конкретной машины
func (r *PostgresTelemetryRepository) GetListByVehicle(ctx context.Context, id int) ([]model.Telemetry, error) {
	query := `SELECT id, organization_id, vehicle_id, device_id, latitude, longitude, fuel, received_at, device_timestamp FROM telemetry WHERE vehicle_id = $1`
	rows, err := r.db.Query(ctx, query, id)
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

// DeleteListByVehicle для PostgresTelemetryRepository удаляет всю телеметрию для конкретной машины
func (r *PostgresTelemetryRepository) DeleteListByVehicle(ctx context.Context, id int) ([]model.Telemetry, error) {
	query := `DELETE FROM telemetry WHERE vehicle_id = $1
		RETURNING id, organization_id, vehicle_id, device_id, latitude, longitude, fuel, received_at, device_timestamp`
	rows, err := r.db.Query(ctx, query, id)
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

// DeleteItemByID для PostgresTelemetryRepository удаляет телеметрию по её ID
func (r *PostgresTelemetryRepository) DeleteItemByID(ctx context.Context, id int) (model.Telemetry, error) {
	query := `DELETE FROM telemetry WHERE id = $1
		RETURNING id, organization_id, vehicle_id, device_id, latitude, longitude, fuel, received_at, device_timestamp`
	var t model.Telemetry
	err := r.db.QueryRow(ctx, query, id).Scan(
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
