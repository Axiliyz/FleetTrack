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
		serial_number, status
	) VALUES ($1, $2)
		RETURNING id, created_at`,
		d.SerialNumber, d.Status).Scan(&d.ID, &d.CreatedAt)
	if err != nil {
		return mapUniqueViolation(err)
	}
	return nil
}

// GetByID возвращает устройство по его ID
func (r *PostgresDeviceRepository) GetByID(ctx context.Context, id int) (model.Device, error) {
	const query = `SELECT id, serial_number, status, created_at FROM devices WHERE id=$1`
	var d model.Device
	err := r.db.QueryRow(ctx, query, id).Scan(&d.ID, &d.SerialNumber, &d.Status, &d.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.Device{}, model.ErrNotFound
		}
		return model.Device{}, err
	}
	return d, nil
}

// Delete удаляет девайс по ID
func (r *PostgresDeviceRepository) Delete(ctx context.Context, id int) (model.Device, error) {
	const query = `
	UPDATE devices 
	SET status = 'INACTIVE'
	WHERE id = $1
	RETURNING id, serial_number, status, created_at
	`
	var d model.Device
	err := r.db.QueryRow(ctx, query, id).Scan(
		&d.ID, &d.SerialNumber, &d.Status, &d.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.Device{}, model.ErrNotFound
		}
		return model.Device{}, err
	}
	return d, nil
}
