// Package postgres отвечает за взаимодействие с БД PostgreSQL
package postgres

import (
	"context"
	"errors"
	"fleettrack/internal/database"
	"fleettrack/internal/model"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const pgUniqueViolationCode = "23505"

// mapUniqueViolation проверяет ошибки уникальности и подключения к БД
func mapUniqueViolation(err error) error {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != pgUniqueViolationCode {
		return err
	}

	switch pgErr.ConstraintName {
	case "vehicles_vin_key":
		return model.ErrDuplicateVIN
	case "vehicles_number_plate_key":
		return model.ErrDuplicatePlate
	case "devices_serial_number_key":
		return model.ErrDuplicateSerialNumber
	case "organizations_name_key":
		return model.ErrDuplicateOrgName
	case "users_email_key":
		return model.ErrDuplicateEmail
	case "idx_users_driver_id":
		return model.ErrDriverAlreadyLinked
	case "idx_alerts_active_unique":
		return model.ErrDuplicateAlert
	default:
		return err
	}
}

// PostgresVehicleRepository хранит автомобили в PostgreSQL
type PostgresVehicleRepository struct {
	db database.DBTX
}

// NewPostgresVehicleRepository создаёт новый репозиторий автомобилей
func NewPostgresVehicleRepository(db database.DBTX) *PostgresVehicleRepository {
	return &PostgresVehicleRepository{
		db: db,
	}
}

// Create для PostgresVehicleRepository сохраняет новую машину в БД
func (r *PostgresVehicleRepository) Create(ctx context.Context, v *model.Vehicle) error {
	const query = `
	INSERT INTO vehicles (
		organization_id,
		vin,
		number_plate,
		model,
		status
	)
	VALUES ($1, $2, $3, $4, $5) RETURNING id, created_at`
	err := r.db.QueryRow(ctx,
		query,
		v.OrganizationID,
		v.VIN,
		v.NumberPlate,
		v.Model,
		v.Status,
	).Scan(&v.ID, &v.CreatedAt)
	if err != nil {
		return mapUniqueViolation(err)
	}
	return nil
}

// GetByID для PostgresVehicleRepository возвращает машину по её ID
func (r *PostgresVehicleRepository) GetByID(ctx context.Context, id int) (model.Vehicle, error) {
	const query = `
	SELECT id, organization_id, vin, number_plate, model, status, created_at, updated_at, last_telemetry_at
	FROM vehicles WHERE id = $1`
	var v model.Vehicle
	err := r.db.QueryRow(ctx, query, id).Scan(
		&v.ID, &v.OrganizationID, &v.VIN, &v.NumberPlate,
		&v.Model, &v.Status, &v.CreatedAt, &v.UpdatedAt, &v.LastTelemetryAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.Vehicle{}, model.ErrNotFound
		}
		return model.Vehicle{}, err
	}
	return v, nil
}

// buildWhereClause берёт фильтр и возвращает готовый кусок WHERE... и срез аргументов
func buildVehicleWhereClause(filter model.VehicleFilter) (string, []any) {
	var conditions []string
	var args []any
	argN := 1
	if filter.OrganizationID != nil {
		conditions = append(conditions, fmt.Sprintf("organization_id = $%d", argN))
		args = append(args, *filter.OrganizationID)
		argN++
	}
	if filter.VIN != nil {
		conditions = append(conditions, fmt.Sprintf("vin = $%d", argN))
		args = append(args, *filter.VIN)
		argN++
	}
	if filter.NumberPlate != nil {
		conditions = append(conditions, fmt.Sprintf("number_plate = $%d", argN))
		args = append(args, *filter.NumberPlate)
		argN++
	}
	if filter.Model != nil {
		conditions = append(conditions, fmt.Sprintf("model = $%d", argN))
		args = append(args, *filter.Model)
		argN++
	}
	if filter.Status != nil {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argN))
		args = append(args, *filter.Status)
		argN++
	}
	if filter.CreatedFrom != nil {
		conditions = append(conditions, fmt.Sprintf("created_at >= $%d", argN))
		args = append(args, *filter.CreatedFrom)
		argN++
	}
	if filter.CreatedTo != nil {
		conditions = append(conditions, fmt.Sprintf("created_at <= $%d", argN))
		args = append(args, *filter.CreatedTo)
		argN++ //nolint:ineffassign
	}
	if len(conditions) == 0 {
		return "", args
	}
	return "WHERE " + strings.Join(conditions, " AND "), args
}

// GetList для PostgresVehicleRepository возвращает список машин
func (r *PostgresVehicleRepository) GetList(ctx context.Context, filter model.VehicleFilter) ([]model.Vehicle, error) {
	whereVehicleClause, args := buildVehicleWhereClause(filter)
	query := fmt.Sprintf(`
		SELECT 
		id, 
		organization_id, 
		vin, 
		number_plate, 
		model, 
		status, 
		created_at, 
		updated_at,
		last_telemetry_at
		FROM vehicles %s ORDER BY id DESC LIMIT $%d OFFSET $%d`,
		whereVehicleClause, len(args)+1, len(args)+2,
	)
	args = append(args, filter.Limit, filter.Offset)
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var vehicles []model.Vehicle
	for rows.Next() {
		var v model.Vehicle
		err = rows.Scan(
			&v.ID, &v.OrganizationID, &v.VIN, &v.NumberPlate,
			&v.Model, &v.Status, &v.CreatedAt, &v.UpdatedAt, &v.LastTelemetryAt,
		)
		if err != nil {
			return nil, err
		}
		vehicles = append(vehicles, v)
	}
	return vehicles, rows.Err()
}

// Delete для PostgresVehicleRepository удаляет машину по её ID.
// organizationID != nil ограничивает удаление машинами этой организации (для не-ADMIN);
// nil означает отсутствие ограничения (ADMIN может удалить машину любой организации).
func (r *PostgresVehicleRepository) Delete(ctx context.Context, id int, organizationID *int) (model.Vehicle, error) {
	query := `
	UPDATE vehicles SET status = 'DELETED'
	WHERE id = $1`
	args := []any{id}
	if organizationID != nil {
		query += " AND organization_id = $2"
		args = append(args, *organizationID)
	}
	query += `
	RETURNING id, organization_id, vin, number_plate, model, status, created_at, updated_at, last_telemetry_at`

	var v model.Vehicle
	err := r.db.QueryRow(ctx, query, args...).Scan(
		&v.ID, &v.OrganizationID, &v.VIN, &v.NumberPlate,
		&v.Model, &v.Status, &v.CreatedAt, &v.UpdatedAt, &v.LastTelemetryAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.Vehicle{}, model.ErrNotFound
		}
		return model.Vehicle{}, err
	}
	return v, nil
}

// buildVehicleSetClause берёт изменяемые поля и возвращает готовый кусок WHERE... и срез аргументов
func buildVehicleSetClause(updater model.UpdateVehicle) (string, []any) {
	var conditions []string
	var args []any
	argN := 1
	if updater.OrganizationID != nil {
		conditions = append(conditions, fmt.Sprintf("organization_id = $%d", argN))
		args = append(args, *updater.OrganizationID)
		argN++
	}
	if updater.NumberPlate != nil {
		conditions = append(conditions, fmt.Sprintf("number_plate = $%d", argN))
		args = append(args, *updater.NumberPlate)
		argN++
	}
	if updater.Status != nil {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argN))
		args = append(args, *updater.Status)
		argN++ //nolint:ineffassign
	}
	if len(conditions) == 0 {
		return "", args
	}
	return strings.Join(conditions, ", "), args
}

// Update для PostgresVehicleRepository обновляет некоторые данные авто по ID.
// organizationID != nil ограничивает обновление машинами этой организации (для не-ADMIN);
// nil означает отсутствие ограничения (ADMIN может изменить машину любой организации).
func (r *PostgresVehicleRepository) Update(ctx context.Context, id int, upd model.UpdateVehicle, organizationID *int) (model.Vehicle, error) {
	setClause, args := buildVehicleSetClause(upd)
	if setClause == "" {
		return r.GetByID(ctx, id)
	}

	where := fmt.Sprintf("id = $%d", len(args)+1)
	args = append(args, id)
	if organizationID != nil {
		where += fmt.Sprintf(" AND organization_id = $%d", len(args)+1)
		args = append(args, *organizationID)
	}

	query := fmt.Sprintf(`
		UPDATE vehicles SET %s, updated_at = NOW() WHERE %s
		RETURNING id, organization_id, vin, number_plate, model, status, created_at, updated_at, last_telemetry_at
		`, setClause, where,
	)

	var v model.Vehicle
	err := r.db.QueryRow(ctx, query, args...).Scan(
		&v.ID, &v.OrganizationID, &v.VIN, &v.NumberPlate,
		&v.Model, &v.Status, &v.CreatedAt, &v.UpdatedAt, &v.LastTelemetryAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.Vehicle{}, model.ErrNotFound
		}
		return model.Vehicle{}, err
	}
	return v, nil
}

// UpdateLastTelemetryAt устанавливает время последней телеметрии
func (r *PostgresVehicleRepository) UpdateLastTelemetryAt(ctx context.Context, vehicleID int, at time.Time) error {
	const query = `UPDATE vehicles SET last_telemetry_at = $1 WHERE id = $2`
	_, err := r.db.Exec(ctx, query, at, vehicleID)
	return err
}
