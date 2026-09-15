package postgres

import (
	"context"
	"fleettrack/internal/database"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// PostgresPartitionRepository отвечает за управление партициями в PostgreSQL
type PostgresPartitionRepository struct {
	db database.DBTX
}

// NewPostgresPartitionRepository создаёт репозиторий партиций
func NewPostgresPartitionRepository(db database.DBTX) *PostgresPartitionRepository {
	return &PostgresPartitionRepository{
		db: db,
	}
}

// CreatePartition создаёт партицию таблицы для диапазона дат, если она ещё не существует
func (r *PostgresPartitionRepository) CreatePartition(ctx context.Context, parentTable, partitionName string, from, to time.Time) error {
	query := fmt.Sprintf(
		`CREATE TABLE IF NOT EXISTS %s 
		PARTITION OF %s FOR VALUES FROM ('%s') TO ('%s')`,
		pgx.Identifier{partitionName}.Sanitize(),
		pgx.Identifier{parentTable}.Sanitize(),
		from.Format("2006-01-02"),
		to.Format("2006-01-02"),
	)
	_, err := r.db.Exec(ctx, query)
	return err
}
