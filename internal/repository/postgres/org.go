package postgres

import (
	"context"
	"fleettrack/internal/database"
	"fleettrack/internal/model"
)

// PostgresOrgRepository хранит организации в PostgreSQL
type PostgresOrgRepository struct {
	db database.DBTX
}

// NewPostgresOrgRepository создаёт новый репозиторий организаций
func NewPostgresOrgRepository(db database.DBTX) *PostgresOrgRepository {
	return &PostgresOrgRepository{
		db: db,
	}
}

// CreateOrg создаёт новую организацию
func (r *PostgresOrgRepository) CreateOrg(ctx context.Context, o *model.Org) error {
	const query = `
	INSERT INTO organizations(name)
	VALUES ($1)
	RETURNING id, created_at
	`
	err := r.db.QueryRow(ctx, query, o.Name).Scan(&o.ID, &o.CreatedAt)
	if err != nil {
		return mapUniqueViolation(err)
	}
	return nil
}

// GetList возвращает список всех организаций
func (r *PostgresOrgRepository) GetList(ctx context.Context) ([]model.Org, error) {
	const query = `SELECT id, name, created_at, updated_at FROM organizations ORDER BY id DESC`
	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orgs []model.Org
	for rows.Next() {
		var o model.Org
		if err := rows.Scan(&o.ID, &o.Name, &o.CreatedAt, &o.UpdatedAt); err != nil {
			return nil, err
		}
		orgs = append(orgs, o)
	}
	return orgs, rows.Err()
}
