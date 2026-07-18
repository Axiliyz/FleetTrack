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
	return r.db.QueryRow(ctx, query, o.Name).Scan(&o.ID, &o.CreatedAt)
}
