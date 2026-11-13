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

	// уникальность VIN, номера и серийника - в пределах организации (миграция 000021),
	// поэтому конфликт возможен только со своей же записью и ничего чужого не выдаёт
	switch pgErr.ConstraintName {
	case "uq_vehicles_org_vin":
		return model.ErrDuplicateVIN
	case "uq_vehicles_org_plate":
		return model.ErrDuplicatePlate
	case "uq_devices_org_serial":
		return model.ErrDuplicateSerialNumber
	case "uq_users_email":
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

// GetByID для PostgresVehicleRepository возвращает машину по её ID; удалённые машины не возвращаются
func (r *PostgresVehicleRepository) GetByID(ctx context.Context, id int) (model.Vehicle, error) {
	const query = `
	SELECT id, organization_id, vin, number_plate, model, status, created_at, updated_at, last_telemetry_at
	FROM vehicles WHERE id = $1 AND status <> 'DELETED'`
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

// buildVehicleWhereClause берёт фильтр и возвращает кусок WHERE и срез аргументов.
// Без явного фильтра по статусу удалённые машины исключаются.
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
	} else {
		conditions = append(conditions, "status <> 'DELETED'")
	}
	if filter.CreatedFrom != nil {
		conditions = append(conditions, fmt.Sprintf("created_at >= $%d", argN))
		args = append(args, *filter.CreatedFrom)
		argN++
	}
	if filter.CreatedTo != nil {
		conditions = append(conditions, fmt.Sprintf("created_at <= $%d", argN))
		args = append(args, *filter.CreatedTo)
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

// Delete для PostgresVehicleRepository мягко удаляет машину (status = DELETED).
// Уже удалённая машина считается ненайденной, машина в рейсе не удаляется - model.ErrVehicleIsBusy.
// organizationID != nil ограничивает удаление машинами этой организации; nil снимает ограничение.
func (r *PostgresVehicleRepository) Delete(ctx context.Context, id int, organizationID *int) (model.Vehicle, error) {
	query := `
	UPDATE vehicles SET status = 'DELETED', updated_at = NOW()
	WHERE id = $1 AND status NOT IN ('DELETED', 'ON_TRIP')`
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
			return model.Vehicle{}, r.explainMiss(ctx, id, organizationID, model.ErrVehicleIsBusy)
		}
		return model.Vehicle{}, err
	}
	return v, nil
}

// explainMiss выясняет, почему UPDATE по машине не нашёл строку: машины нет (или она чужая/удалена) -
// model.ErrNotFound, машина в рейсе - onTrip. Само решение «не менять» уже принято атомарно
// в UPDATE, это чтение только выбирает ошибку для ответа
func (r *PostgresVehicleRepository) explainMiss(ctx context.Context, id int, organizationID *int, onTrip error) error {
	v, err := r.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if organizationID != nil && v.OrganizationID != *organizationID {
		return model.ErrNotFound
	}
	if v.Status == model.VehicleStatusOnTrip {
		return onTrip
	}
	return model.ErrNotFound
}

// buildVehicleSetClause берёт изменяемые поля и возвращает готовый кусок SET и срез аргументов
func buildVehicleSetClause(updater model.UpdateVehicle) (string, []any) {
	var conditions []string
	var args []any
	argN := 1
	if updater.NumberPlate != nil {
		conditions = append(conditions, fmt.Sprintf("number_plate = $%d", argN))
		args = append(args, *updater.NumberPlate)
		argN++
	}
	if updater.Status != nil {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argN))
		args = append(args, *updater.Status)
	}
	if len(conditions) == 0 {
		return "", args
	}
	return strings.Join(conditions, ", "), args
}

// Update для PostgresVehicleRepository обновляет некоторые данные авто по ID.
// Удалённые машины не изменяются и считаются ненайденными.
// Статус машины в рейсе вручную не меняется - им управляет рейс, вернётся model.ErrVehicleOnTrip.
// organizationID != nil ограничивает обновление машинами этой организации; nil снимает ограничение.
func (r *PostgresVehicleRepository) Update(ctx context.Context, id int, upd model.UpdateVehicle, organizationID *int) (model.Vehicle, error) {
	setClause, args := buildVehicleSetClause(upd)
	if setClause == "" {
		v, err := r.GetByID(ctx, id)
		if err != nil {
			return model.Vehicle{}, err
		}
		if organizationID != nil && v.OrganizationID != *organizationID {
			return model.Vehicle{}, model.ErrNotFound
		}
		return v, nil
	}

	where := fmt.Sprintf("id = $%d AND status <> 'DELETED'", len(args)+1)
	if upd.Status != nil {
		where += " AND status <> 'ON_TRIP'"
	}
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
			return model.Vehicle{}, r.explainMiss(ctx, id, organizationID, model.ErrVehicleOnTrip)
		}
		return model.Vehicle{}, mapUniqueViolation(err)
	}
	return v, nil
}

// UpdateStatus переводит машину из статуса from в статус to. Условие на from стоит в самом UPDATE,
// поэтому два рейса не займут одну машину одновременно: второй UPDATE строку уже не найдёт.
// organizationID != nil ограничивает переход машинами этой организации.
// Возвращает model.ErrNotFound, если машины нет, она чужая или сейчас не в статусе from
func (r *PostgresVehicleRepository) UpdateStatus(ctx context.Context, id int, from, to model.VehicleStatus, organizationID *int) (model.Vehicle, error) {
	const query = `
	UPDATE vehicles SET status = $3, updated_at = NOW()
	WHERE id = $1 AND status = $2 AND ($4::int IS NULL OR organization_id = $4)
	RETURNING id, organization_id, vin, number_plate, model, status, created_at, updated_at, last_telemetry_at`

	var v model.Vehicle
	err := r.db.QueryRow(ctx, query, id, from, to, organizationID).Scan(
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
