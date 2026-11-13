//go:build integration

package postgres

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

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
	ensureCurrentPartitions(ctx, TestPool)
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

// ensureCurrentPartitions создаёт партиции telemetry и alerts на текущий и следующий месяц - то же,
// что делает PartitionWorker при старте приложения. Миграции заводят партиции на фиксированные
// месяцы, и без этого тесты падали бы с «no partition found», как только календарь уйдёт дальше
func ensureCurrentPartitions(ctx context.Context, pool *pgxpool.Pool) {
	repo := NewPostgresPartitionRepository(pool)
	now := time.Now().UTC()
	for i := 0; i <= 1; i++ {
		from := time.Date(now.Year(), now.Month()+time.Month(i), 1, 0, 0, 0, 0, time.UTC)
		to := from.AddDate(0, 1, 0)
		for _, table := range []string{"telemetry", "alerts"} {
			name := fmt.Sprintf("%s_%d_%02d", table, from.Year(), from.Month())
			if err := repo.CreatePartition(ctx, table, name, from, to); err != nil {
				panic(err)
			}
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
