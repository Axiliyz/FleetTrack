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

// GetByEmail возвращает юзера по email без учёта регистра; удалённые юзеры не находятся
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
	FROM users WHERE LOWER(email) = LOWER($1) AND deleted_at IS NULL`
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

// GetByID получает юзера по ID; удалённый юзер считается ненайденным
// Возвращает найденного юзера или ошибку
func (r *PostgresUserRepository) GetByID(ctx context.Context, id int) (model.User, error) {
	const query = `
	SELECT id, organization_id, driver_id, name, email, role
	FROM users WHERE id = $1 AND deleted_at IS NULL`
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

// Create создаёт нового юзера. Привязанный водитель должен быть из той же организации:
// проверка стоит в самом INSERT, иначе водителя чужой организации можно было бы привязать по ID.
// Возвращает model.ErrInvalidDriverID, если водителя нет в организации юзера
func (r *PostgresUserRepository) Create(ctx context.Context, u *model.User) error {
	const query = `
	INSERT INTO users (
	organization_id,
	driver_id,
	name, email,
	password_hash, role
	)
	SELECT $1, $2, $3, $4, $5, $6
	WHERE $2::bigint IS NULL
	   OR EXISTS (SELECT 1 FROM drivers WHERE id = $2 AND organization_id = $1)
	RETURNING id`
	err := r.db.QueryRow(ctx, query,
		u.OrganizationID, u.DriverID, u.Name, u.Email, u.PasswordHash, u.Role,
	).Scan(&u.ID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.ErrInvalidDriverID
		}
		return mapUniqueViolation(err)
	}
	return nil
}

// buildUserWhereClause берёт фильтр и возвращает готовый кусок WHERE... и срез аргументов
func buildUserWhereClause(filter model.UserFilter) (string, []any) {
	conditions := []string{"deleted_at IS NULL"}
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
	}
	return "WHERE " + strings.Join(conditions, " AND "), args
}

// GetList для PostgresUserRepository возвращает список юзеров
func (r *PostgresUserRepository) GetList(ctx context.Context, filter model.UserFilter) ([]model.User, error) {
	whereClause, args := buildUserWhereClause(filter)
	query := fmt.Sprintf(`
	SELECT id, organization_id, driver_id, name, email, role
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
		if err := rows.Scan(&u.ID, &u.OrganizationID, &u.DriverID, &u.Name, &u.Email, &u.Role); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

// DeleteByID мягко удаляет юзера по ID: строка остаётся, потому что на неё ссылаются
// алерты (кто принял, кто закрыл). organizationID != nil ограничивает удаление юзерами
// этой организации; nil - без ограничения.
func (r *PostgresUserRepository) DeleteByID(ctx context.Context, id int, organizationID *int) (model.User, error) {
	query := `UPDATE users SET deleted_at = NOW() WHERE id = $1 AND deleted_at IS NULL`
	args := []any{id}
	if organizationID != nil {
		query += " AND organization_id = $2"
		args = append(args, *organizationID)
	}
	query += `
	RETURNING id, organization_id, driver_id, name, email, role`

	var u model.User
	err := r.db.QueryRow(ctx, query, args...).Scan(
		&u.ID, &u.OrganizationID, &u.DriverID, &u.Name, &u.Email, &u.Role,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.User{}, model.ErrNotFound
		}
		return model.User{}, err
	}
	return u, nil
}

// LockActiveAdmins блокирует строки действующих админов организации (FOR UPDATE)
// до конца транзакции и возвращает их ID
func (r *PostgresUserRepository) LockActiveAdmins(ctx context.Context, organizationID int) ([]int, error) {
	const query = `
	SELECT id FROM users
	WHERE organization_id = $1 AND role = 'ADMIN' AND deleted_at IS NULL
	ORDER BY id
	FOR UPDATE`
	rows, err := r.db.Query(ctx, query, organizationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}
