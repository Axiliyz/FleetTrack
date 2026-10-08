//go:build integration

package postgres

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"testing"

	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TestPool — пул соединений с тестовым Postgres, общий для всех тестов пакета
var TestPool *pgxpool.Pool

// TestMain поднимает Postgres в контейнере на весь пакет, прогоняет миграции
// и создаёт общий пул соединений для интеграционных тестов
func TestMain(m *testing.M) {
	ctx := context.Background()
	container, err := startPostgres(ctx)
	if err != nil {
		panic(err)
	}
	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		panic(err)
	}
	applyMigrations(ctx, dsn)
	TestPool = connectPool(ctx, dsn)
	code := m.Run()
	TestPool.Close()
	container.Terminate(ctx)

	os.Exit(code)
}

// startPostgres поднимает контейнер с PostgreSQL для интеграционных тестов
// Возвращает контейнер и ошибку, если он не поднялся
func startPostgres(ctx context.Context) (*tcpostgres.PostgresContainer, error) {
	container, err := tcpostgres.Run(ctx, "postgres:16-alpine",
		tcpostgres.WithDatabase("testdb"),
		tcpostgres.WithUsername("testuser"),
		tcpostgres.WithPassword("testpassword"),
		tcpostgres.BasicWaitStrategies())
	if err != nil {
		return nil, err
	}
	return container, err
}

// applyMigrations последовательно выполняет все файлы *.up.sql из migrations/
// Паникует при любой ошибке: без применённых миграций тестам делать нечего
func applyMigrations(ctx context.Context, dsn string) {
	migrations, err := filepath.Glob("../../../migrations/*.up.sql")
	if err != nil {
		panic(err)
	}
	slices.Sort(migrations)
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		panic(err)
	}
	defer conn.Close(ctx)

	for _, path := range migrations {
		sql, err := os.ReadFile(path)
		if err != nil {
			panic(err)
		}
		_, err = conn.Exec(ctx, string(sql))
		if err != nil {
			panic(err)
		}
	}

}

// connectPool создаёт пул соединений с тестовой БД
// Паникует, если не удалось подключиться
func connectPool(ctx context.Context, dsn string) *pgxpool.Pool {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		panic(err)
	}

	return pool
}
