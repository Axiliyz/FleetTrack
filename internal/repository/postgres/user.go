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

// PostgresUserRepository отвечает за сохранение пользователей
type PostgresUserRepository struct {
	db database.DBTX
}

// NewPostgresUserRepository - конструктор репозитория юзеров
func NewPostgresUserRepository(db database.DBTX) *PostgresUserRepository {
	return &PostgresUserRepository{
		db: db,
	}
}

// GetByEmail возвращает юзера по email
// Возвращает найденного юзера или ошибку
func (r *PostgresUserRepository) GetByEmail(ctx context.Context, email string) (model.User, error) {
	const query = `
	SELECT id,
	organization_id,
	driver_id,
	name,
	email,
	password_hash,
	role
	FROM users WHERE email=$1`
	var u model.User
	err := r.db.QueryRow(ctx, query, email).Scan(&u.ID, &u.OrganizationID, &u.DriverID, &u.Name, &u.Email, &u.PasswordHash, &u.Role)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.User{}, model.ErrNotFound
		}
		return model.User{}, err
	}
	return u, nil
}

// GetByID получает юзера по ID
// Возвращает найденного юзера или ошибку
func (r *PostgresUserRepository) GetByID(ctx context.Context, id int) (model.User, error) {
	const query = `
	SELECT id, organization_id, driver_id, name, email, role
	FROM users WHERE id=$1`
	var u model.User
	err := r.db.QueryRow(ctx, query, id).Scan(&u.ID, &u.OrganizationID, &u.DriverID, &u.Name, &u.Email, &u.Role)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.User{}, model.ErrNotFound
		}
		return model.User{}, err
	}
	return u, nil
}

// Create создаёт нового юзера
func (r *PostgresUserRepository) Create(ctx context.Context, u *model.User) error {
	const query = `
	INSERT INTO users (
	organization_id,
	driver_id,
	name, email,
	password_hash, role
	) VALUES ($1, $2, $3, $4, $5, $6)
	RETURNING id`
	err := r.db.QueryRow(ctx, query,
		u.OrganizationID, u.DriverID, u.Name, u.Email, u.PasswordHash, u.Role,
	).Scan(&u.ID)
	if err != nil {
		return mapUniqueViolation(err)
	}
	return nil
}

// buildUserWhereClause берёт фильтр и возвращает готовый кусок WHERE... и срез аргументов
func buildUserWhereClause(filter model.UserFilter) (string, []any) {
	var conditions []string
	var args []any
	argN := 1
	if filter.OrganizationID != nil {
		conditions = append(conditions, fmt.Sprintf("organization_id = $%d", argN))
		args = append(args, *filter.OrganizationID)
		argN++ //nolint:ineffassign
	}
	if filter.Role != nil {
		conditions = append(conditions, fmt.Sprintf("role = $%d", argN))
		args = append(args, *filter.Role)
		argN++ //nolint:ineffassign
	}
	if len(conditions) == 0 {
		return "", args
	}
	return "WHERE " + strings.Join(conditions, " AND "), args
}

// GetList для PostgresUserRepository возвращает список юзеров
func (r *PostgresUserRepository) GetList(ctx context.Context, filter model.UserFilter) ([]model.User, error) {
	whereClause, args := buildUserWhereClause(filter)
	query := fmt.Sprintf(`
	SELECT id, organization_id, name, email, role
	FROM users %s ORDER BY id DESC LIMIT $%d OFFSET $%d`,
		whereClause, len(args)+1, len(args)+2)
	args = append(args, filter.Limit, filter.Offset)
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []model.User
	for rows.Next() {
		var u model.User
		if err := rows.Scan(&u.ID, &u.OrganizationID, &u.Name, &u.Email, &u.Role); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

// DeleteByID удаляет юзера по ID. organizationID != nil ограничивает удаление юзерами
// этой организации; nil - без ограничения.
func (r *PostgresUserRepository) DeleteByID(ctx context.Context, id int, organizationID *int) (model.User, error) {
	query := `DELETE FROM users WHERE id=$1`
	args := []any{id}
	if organizationID != nil {
		query += " AND organization_id = $2"
		args = append(args, *organizationID)
	}
	query += `
	RETURNING id, organization_id, name, email, role`

	var u model.User
	err := r.db.QueryRow(ctx, query, args...).Scan(
		&u.ID, &u.OrganizationID, &u.Name, &u.Email, &u.Role,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.User{}, model.ErrNotFound
		}
		return model.User{}, err
	}
	return u, nil
}
