// Package database содержит общие абстракции доступа к БД, используемые
// как отдельным соединением, так и транзакциями.
package database

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// DBTX — общий интерфейс для *pgxpool.Pool и pgx.Tx, позволяющий
// репозиториям работать как с обычным соединением, так и с транзакцией.
type DBTX interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}
