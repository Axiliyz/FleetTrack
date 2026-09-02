package postgres

import (
	"context"
	"errors"
	"fleettrack/internal/database"
	"fleettrack/internal/model"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const pgForeignKeyViolationCode = "23503"

// PostgresDriverRepository хранит водителей в PostgreSQL
type PostgresDriverRepository struct {
	db database.DBTX
}

// NewPostgresDriverRepository создаёт новый репозиторий водителей
func NewPostgresDriverRepository(db database.DBTX) *PostgresDriverRepository {
	return &PostgresDriverRepository{
		db: db,
	}
}

// Create для PostgresDriverRepository сохраняет нового водителя в БД
func (r *PostgresDriverRepository) Create(ctx context.Context, d *model.Driver) error {
	const query = `
	INSERT INTO drivers (organization_id, name)
	VALUES ($1, $2) RETURNING id, created_at`
	return r.db.QueryRow(ctx, query, d.OrganizationID, d.Name).Scan(&d.ID, &d.CreatedAt)
}

// GetByID для PostgresDriverRepository возвращает водителя по его ID
func (r *PostgresDriverRepository) GetByID(ctx context.Context, id int) (model.Driver, error) {
	const query = `
	SELECT id, organization_id, name, created_at, updated_at
	FROM drivers WHERE id = $1`
	var d model.Driver
	err := r.db.QueryRow(ctx, query, id).Scan(
		&d.ID, &d.OrganizationID, &d.Name, &d.CreatedAt, &d.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.Driver{}, model.ErrNotFound
		}
		return model.Driver{}, err
	}
	return d, nil
}

// buildDriverWhereClause берёт фильтр и возвращает готовый кусок WHERE... и срез аргументов
func buildDriverWhereClause(filter model.DriverFilter) (string, []any) {
	var conditions []string
	var args []any
	argN := 1
	if filter.OrganizationID != nil {
		conditions = append(conditions, fmt.Sprintf("organization_id = $%d", argN))
		args = append(args, *filter.OrganizationID)
		argN++ //nolint:ineffassign
	}
	if filter.Name != nil {
		conditions = append(conditions, fmt.Sprintf("name = $%d", argN))
		args = append(args, *filter.Name)
		argN++ //nolint:ineffassign
	}
	if filter.CreatedFrom != nil {
		conditions = append(conditions, fmt.Sprintf("created_at >= $%d", argN))
		args = append(args, *filter.CreatedFrom)
		argN++ //nolint:ineffassign
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

// GetList для PostgresDriverRepository возвращает список водителей
func (r *PostgresDriverRepository) GetList(ctx context.Context, filter model.DriverFilter) ([]model.Driver, error) {
	whereClause, args := buildDriverWhereClause(filter)
	query := fmt.Sprintf(`
	SELECT id, organization_id, name, created_at, updated_at
	FROM drivers %s ORDER BY id DESC LIMIT $%d OFFSET $%d`,
		whereClause, len(args)+1, len(args)+2)
	args = append(args, filter.Limit, filter.Offset)
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var drivers []model.Driver
	for rows.Next() {
		var d model.Driver
		if err := rows.Scan(&d.ID, &d.OrganizationID, &d.Name, &d.CreatedAt, &d.UpdatedAt); err != nil {
			return nil, err
		}
		drivers = append(drivers, d)
	}
	return drivers, rows.Err()
}

// buildDriverSetClause берёт изменяемые поля и возвращает готовый кусок SET... и срез аргументов
func buildDriverSetClause(upd model.UpdateDriver) (string, []any) {
	var conditions []string
	var args []any
	argN := 1
	if upd.OrganizationID != nil {
		conditions = append(conditions, fmt.Sprintf("organization_id = $%d", argN))
		args = append(args, *upd.OrganizationID)
		argN++ //nolint:ineffassign
	}
	if upd.Name != nil {
		conditions = append(conditions, fmt.Sprintf("name = $%d", argN))
		args = append(args, *upd.Name)
		argN++ //nolint:ineffassign
	}
	if len(conditions) == 0 {
		return "", args
	}
	return strings.Join(conditions, ", "), args
}

// Update для PostgresDriverRepository обновляет некоторые данные водителя по ID
func (r *PostgresDriverRepository) Update(ctx context.Context, id int, upd model.UpdateDriver) (model.Driver, error) {
	setClause, args := buildDriverSetClause(upd)
	if setClause == "" {
		return r.GetByID(ctx, id)
	}

	query := fmt.Sprintf(`
	UPDATE drivers SET %s, updated_at = NOW() WHERE id = $%d
	RETURNING id, organization_id, name, created_at, updated_at`,
		setClause, len(args)+1)

	var d model.Driver
	args = append(args, id)
	err := r.db.QueryRow(ctx, query, args...).Scan(
		&d.ID, &d.OrganizationID, &d.Name, &d.CreatedAt, &d.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.Driver{}, model.ErrNotFound
		}
		return model.Driver{}, err
	}
	return d, nil
}

// Delete для PostgresDriverRepository удаляет водителя по его ID
// Если у водителя есть рейсы, ссылающиеся на него - возвращает model.ErrDriverHasActiveTrips
func (r *PostgresDriverRepository) Delete(ctx context.Context, id int) (model.Driver, error) {
	const query = `
	DELETE FROM drivers WHERE id = $1
	RETURNING id, organization_id, name, created_at, updated_at`
	var d model.Driver
	err := r.db.QueryRow(ctx, query, id).Scan(
		&d.ID, &d.OrganizationID, &d.Name, &d.CreatedAt, &d.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.Driver{}, model.ErrNotFound
		}
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgForeignKeyViolationCode {
			return model.Driver{}, model.ErrDriverHasActiveTrips
		}
		return model.Driver{}, err
	}
	return d, nil
}
