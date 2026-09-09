package postgres

import (
	"context"
	"errors"
	"fleettrack/internal/database"
	"fleettrack/internal/model"

	"github.com/jackc/pgx/v5"
)

// PostgresDeviceRepository хранит устройства в PostgreSQL
type PostgresDeviceRepository struct {
	db database.DBTX
}

// NewPostgresDeviceRepository создаёт новый репозиторий устройств
func NewPostgresDeviceRepository(db database.DBTX) *PostgresDeviceRepository {
	return &PostgresDeviceRepository{
		db: db,
	}
}

// Create создаёт новый девайс
func (r *PostgresDeviceRepository) Create(ctx context.Context, d *model.Device) error {
	err := r.db.QueryRow(ctx, `INSERT INTO devices 
	(
		serial_number, status, organization_id
	) VALUES ($1, $2, $3)
		RETURNING id, created_at`,
		d.SerialNumber, d.Status, d.OrganizationID).Scan(&d.ID, &d.CreatedAt)
	if err != nil {
		return mapUniqueViolation(err)
	}
	return nil
}

// GetByID возвращает устройство по его ID
func (r *PostgresDeviceRepository) GetByID(ctx context.Context, id int) (model.Device, error) {
	const query = `SELECT 
	id, 
	serial_number, 
	status, 
	created_at,
	organization_id
	FROM devices WHERE id=$1`
	var d model.Device
	err := r.db.QueryRow(ctx, query, id).Scan(&d.ID, &d.SerialNumber, &d.Status, &d.CreatedAt, &d.OrganizationID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.Device{}, model.ErrNotFound
		}
		return model.Device{}, err
	}
	return d, nil
}

// Delete удаляет девайс по ID. organizationID != nil ограничивает удаление устройствами
// этой организации; nil означает отсутствие ограничения (ADMIN может удалить устройство
// любой организации).
func (r *PostgresDeviceRepository) Delete(ctx context.Context, id int, organizationID *int) (model.Device, error) {
	query := `
	UPDATE devices
	SET status = 'INACTIVE'
	WHERE id = $1`
	args := []any{id}
	if organizationID != nil {
		query += " AND organization_id = $2"
		args = append(args, *organizationID)
	}
	query += `
	RETURNING id, serial_number, status, created_at, organization_id`

	var d model.Device
	err := r.db.QueryRow(ctx, query, args...).Scan(
		&d.ID, &d.SerialNumber, &d.Status, &d.CreatedAt, &d.OrganizationID,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.Device{}, model.ErrNotFound
		}
		return model.Device{}, err
	}
	return d, nil
}
