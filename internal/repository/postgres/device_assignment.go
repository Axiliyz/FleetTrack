package postgres

import (
	"context"
	"errors"
	"fleettrack/internal/model"

	"github.com/jackc/pgx/v5"
)

type PostgresAssignmentRepository struct {
	db DBTX
}

func NewPostgresAssignmentRepository(db DBTX) *PostgresAssignmentRepository {
	return &PostgresAssignmentRepository{
		db: db,
	}
}

func (r *PostgresAssignmentRepository) GetActiveAssignment(ctx context.Context, deviceID int) (model.DeviceAssignment, error) {
	query := `SELECT id, vehicle_id, device_id, started_at, ended_at FROM device_assignments WHERE device_id=$1 AND ended_at IS NULL`
	var as model.DeviceAssignment
	err := r.db.QueryRow(ctx, query, deviceID).Scan(&as.ID, &as.VehicleID, &as.DeviceID, &as.StartedAt, &as.EndedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.DeviceAssignment{}, model.ErrNotFound
		}
		return model.DeviceAssignment{}, err
	}
	return as, nil
}

func (r *PostgresAssignmentRepository) CreateAssignment(ctx context.Context, a *model.DeviceAssignment) error {
	err := r.db.QueryRow(ctx,
		`INSERT INTO device_assignments
	(vehicle_id, device_id, started_at)
	VALUES ($1, $2, $3)
	RETURNING id`, a.VehicleID, a.DeviceID, a.StartedAt).Scan(&a.ID)
	return err
}

func (r *PostgresAssignmentRepository) EndAssignment(ctx context.Context, deviceID int) error {
	query := `
	UPDATE device_assignments
	SET ended_at = NOW()
	WHERE device_id = $1 AND ended_at IS NULL
	RETURNING id`

	var assignmentID int
	err := r.db.QueryRow(ctx, query, deviceID).Scan(&assignmentID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.ErrNotFound
		}
		return err
	}
	return nil
}
