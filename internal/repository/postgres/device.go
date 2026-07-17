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

// GetByID возвращает устройство по его ID
func (r *PostgresDeviceRepository) GetByID(ctx context.Context, id int) (model.Device, error) {
	query := `SELECT id, serial_number, status, created_at FROM devices WHERE id=$1`
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
