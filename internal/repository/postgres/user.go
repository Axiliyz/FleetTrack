package postgres

import (
	"context"
	"errors"
	"fleettrack/internal/database"
	"fleettrack/internal/model"

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

// GetByEmail возвращает пользователя по email
func (r *PostgresUserRepository) GetByEmail(ctx context.Context, email string) (model.User, error) {
	const query = `
	SELECT id, organization_id, name, role 
	FROM users WHERE email=$1`
	var u model.User
	err := r.db.QueryRow(ctx, query, email).Scan(&u.ID, &u.OrganizationID, &u.Name, &u.Role)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.User{}, model.ErrNotFound
		}
		return model.User{}, err
	}
	return u, nil
}
