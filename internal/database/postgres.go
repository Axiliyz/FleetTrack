package database

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// NewPostgresPool создаёт пул соединений с PostgreSQL и проверяет
// подключение через Ping.
func NewPostgresPool(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, err
	}
	cfg.MaxConns = 25
	cfg.MinConns = 5
	cfg.MaxConnLifetime = 30 * time.Minute
	cfg.MaxConnIdleTime = 5 * time.Minute

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, err
	}
	err = pool.Ping(ctx)
	if err != nil {
		return nil, err
	}

	return pool, nil
}
