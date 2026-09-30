//go:build integration

// Файл объявлен как package postgres_test (внешний тестовый пакет), а не
// package postgres, потому что service.NewAlertService требует
// factory.RepositoryFactory, а пакет factory сам импортирует postgres —
// импорт factory из package postgres создал бы цикл postgres -> factory -> postgres.
// postgres_test лежит в той же папке, но считается отдельным пакетом и может
// свободно импортировать и postgres, и factory.
package postgres_test

import (
	"context"
	"sync"
	"testing"

	"fleettrack/internal/logger"
	"fleettrack/internal/model"
	"fleettrack/internal/repository/factory"
	"fleettrack/internal/repository/postgres"
	"fleettrack/internal/service"
	"fleettrack/internal/transaction"
)

// TestEvaluateTelemetryNoDuplicateAlerts проверяет, что два конкурентных вызова
// EvaluateTelemetry для одного и того же нарушения (автомобиль + правило) создают
// ровно один алерт, а не два — благодаря pg_advisory_xact_lock в AlertService.fireAlert.
//
// В отличие от теста на SKIP LOCKED, здесь не нужен трюк с удержанием открытой
// транзакции: pg_advisory_xact_lock БЛОКИРУЕТ вторую транзакцию, а не пропускает
// её. Поэтому корректность гарантирована при любом соотношении времени выполнения
// двух вызовов — даже если они не пересекутся физически, ничего не сломается.
// close(start) здесь только увеличивает шанс реального пересечения, а не является
// обязательным условием прохождения теста.
func TestEvaluateTelemetryNoDuplicateAlerts(t *testing.T) {
	ctx := context.Background()

	if _, err := postgres.TestPool.Exec(ctx,
		`TRUNCATE alert_rules, alerts, user_notification_channels, alert_notifications,
		 organizations, vehicles CASCADE`,
	); err != nil {
		t.Fatalf("truncate: %v", err)
	}

	var orgID, vehicleID, ruleID int

	if err := postgres.TestPool.QueryRow(ctx,
		`INSERT INTO organizations (name) VALUES ('Test Org') RETURNING id`,
	).Scan(&orgID); err != nil {
		t.Fatalf("insert organization: %v", err)
	}

	if err := postgres.TestPool.QueryRow(ctx,
		`INSERT INTO vehicles (organization_id, model) VALUES ($1, 'Test Model') RETURNING id`,
		orgID,
	).Scan(&vehicleID); err != nil {
		t.Fatalf("insert vehicle: %v", err)
	}

	if err := postgres.TestPool.QueryRow(ctx,
		`INSERT INTO alert_rules (organization_id, type, name, threshold, severity)
		 VALUES ($1, 'SPEED_EXCEEDED', 'Test Rule', 90, 'HIGH') RETURNING id`,
		orgID,
	).Scan(&ruleID); err != nil {
		t.Fatalf("insert alert_rule: %v", err)
	}

	alertService := service.NewAlertService(
		postgres.NewPostgresAlertRepository(postgres.TestPool),
		postgres.NewPostgresAlertRuleRepository(postgres.TestPool),
		postgres.NewUserNotificationChannelRepository(postgres.TestPool),
		postgres.NewPostgresAlertNotificationRepository(postgres.TestPool),
		transaction.NewPostgresTransactionManager(postgres.TestPool),
		factory.NewPostgresRepositoryFactory(),
		logger.NewStdLogger(logger.ErrorLevel),
	)

	// Одна и та же телеметрия передаётся в обе горутины: превышение скорости
	// 120 км/ч при пороге правила 90 км/ч, для одного и того же автомобиля
	telemetry := model.Telemetry{
		OrganizationID: orgID,
		VehicleID:      vehicleID,
		SpeedKmh:       120,
	}

	start := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(2)

	for i := 0; i < 2; i++ {
		go func() {
			defer wg.Done()
			<-start
			if err := alertService.EvaluateTelemetry(ctx, telemetry); err != nil {
				t.Errorf("EvaluateTelemetry: %v", err)
			}
		}()
	}

	close(start)
	wg.Wait()

	var alertCount int
	if err := postgres.TestPool.QueryRow(ctx,
		`SELECT COUNT(*) FROM alerts WHERE vehicle_id = $1 AND rule_id = $2`,
		vehicleID, ruleID,
	).Scan(&alertCount); err != nil {
		t.Fatalf("count alerts: %v", err)
	}

	if alertCount != 1 {
		t.Errorf("expected exactly 1 alert after two concurrent violations, got %d", alertCount)
	}
}
