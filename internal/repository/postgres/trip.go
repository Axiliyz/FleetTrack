package postgres

import (
	"context"
	"errors"
	"fleettrack/internal/database"
	"fleettrack/internal/model"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
)

// PostgresTripRepository отвечает за сохранение поездок в postgres
type PostgresTripRepository struct {
	db database.DBTX
}

// NewPostgresTripRepository создаёт репозиторий рейсов
func NewPostgresTripRepository(db database.DBTX) *PostgresTripRepository {
	return &PostgresTripRepository{
		db: db,
	}
}

// CreateTrip создаёт рейс
func (r *PostgresTripRepository) CreateTrip(ctx context.Context, t *model.Trip) error {
	const query = `
	INSERT INTO trips(driver_id, vehicle_id, status) VALUES ($1, $2, $3)
	RETURNING id, started_at, status
	`
	return r.db.QueryRow(ctx, query, t.DriverID, t.VehicleID, t.Status).Scan(&t.ID, &t.StartedAt, &t.Status)
}

// GetByID возвращает рейс по ID, или ошибку
func (r *PostgresTripRepository) GetByID(ctx context.Context, id int) (model.Trip, error) {
	const query = `SELECT id, driver_id, vehicle_id, started_at, ended_at, status FROM trips WHERE id=$1`
	var t model.Trip
	err := r.db.QueryRow(ctx, query, id).Scan(&t.ID, &t.DriverID, &t.VehicleID, &t.StartedAt, &t.EndedAt, &t.Status)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.Trip{}, model.ErrNotFound
		}
		return model.Trip{}, err
	}
	return t, nil
}

// isFinalTripStatus проверяет, завершает ли статус рейс
func isFinalTripStatus(s model.TripStatus) bool {
	switch s {
	case model.TripStatusCancelled, model.TripStatusSucceeded:
		return true
	default:
		return false
	}
}

// UpdateTrip для PostgresTripRepository обновляет статус рейса по ID
// Если статус завершающий - SUCCEEDED/CANCELLED, ended_at ставится в NOW()
// Если рейс уже находится в завершающем статусе - возвращает model.ErrTripAlreadyFinished
func (r *PostgresTripRepository) UpdateTrip(ctx context.Context, upd model.Trip) (model.Trip, error) {
	const query = `
	UPDATE trips SET
		status = $2,
		ended_at = CASE WHEN $3 THEN NOW() ELSE ended_at END
	WHERE id = $1 AND status NOT IN ($4, $5)
	RETURNING id, driver_id, vehicle_id, started_at, ended_at, status
	`

	var t model.Trip
	err := r.db.QueryRow(
		ctx, query,
		upd.ID, upd.Status, isFinalTripStatus(upd.Status),
		model.TripStatusCancelled, model.TripStatusSucceeded,
	).Scan(
		&t.ID, &t.DriverID, &t.VehicleID, &t.StartedAt, &t.EndedAt, &t.Status,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// Нет строки: либо рейса не существует, либо он уже завершён - разберёмся отдельным запросом
			if _, getErr := r.GetByID(ctx, upd.ID); getErr != nil {
				return model.Trip{}, getErr
			}
			return model.Trip{}, model.ErrTripAlreadyFinished
		}
		return model.Trip{}, err
	}
	return t, nil
}

// DeleteTrip отменяет рейс - эквивалент UpdateTrip со статусом CANCELLED
// Если рейс уже завершён (SUCCEEDED/CANCELLED) - возвращает model.ErrTripAlreadyFinished
func (r *PostgresTripRepository) DeleteTrip(ctx context.Context, id int) (model.Trip, error) {
	return r.UpdateTrip(ctx, model.Trip{ID: id, Status: model.TripStatusCancelled})
}

// buildTripWhereClause берёт фильтр и возвращает готовый кусок WHERE... и срез аргументов
func buildTripWhereClause(filter *model.TripFilter) (string, []any) {
	var conditions []string
	var args []any
	argN := 1
	if filter.DriverID != nil {
		conditions = append(conditions, fmt.Sprintf("driver_id = $%d", argN))
		args = append(args, *filter.DriverID)
		argN++
	}
	if filter.VehicleID != nil {
		conditions = append(conditions, fmt.Sprintf("vehicle_id = $%d", argN))
		args = append(args, *filter.VehicleID)
		argN++
	}
	if filter.Status != nil {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argN))
		args = append(args, *filter.Status)
		argN++
	}
	if filter.StartedFrom != nil {
		conditions = append(conditions, fmt.Sprintf("started_at >= $%d", argN))
		args = append(args, *filter.StartedFrom)
		argN++
	}
	if filter.StartedTo != nil {
		conditions = append(conditions, fmt.Sprintf("started_at <= $%d", argN))
		args = append(args, *filter.StartedTo)
		argN++
	}
	if len(conditions) == 0 {
		return "", args
	}
	return "WHERE " + strings.Join(conditions, " AND "), args
}

// GetListTrips получает список рейсов с фильтрацией
func (r *PostgresTripRepository) GetListTrips(ctx context.Context, filter *model.TripFilter) ([]model.Trip, error) {
	whereTripClause, args := buildTripWhereClause(filter)
	query := fmt.Sprintf(`
	SELECT id, driver_id, vehicle_id, started_at, ended_at, status FROM trips %s ORDER BY id DESC LIMIT $%d OFFSET $%d`,
		whereTripClause, len(args)+1, len(args)+2)
	args = append(args, filter.Limit, filter.Offset)
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}

	defer rows.Close()
	var trips []model.Trip
	for rows.Next() {
		var t model.Trip
		err = rows.Scan(
			&t.ID, &t.DriverID, &t.VehicleID, &t.StartedAt, &t.EndedAt, &t.Status,
		)
		if err != nil {
			return nil, err
		}
		trips = append(trips, t)
	}
	return trips, rows.Err()
}
