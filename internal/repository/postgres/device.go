package postgres

import (
	"context"
	"errors"
	"fleettrack/internal/model"

	"github.com/jackc/pgx/v5"
)

type PostgresDeviceRepository struct {
	db DBTX
}

func NewPostgresDeviceRepository(db DBTX) *PostgresDeviceRepository {
	return &PostgresDeviceRepository{
		db: db,
	}
}

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
