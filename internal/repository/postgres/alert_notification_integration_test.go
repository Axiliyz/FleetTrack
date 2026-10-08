//go:build integration

package postgres

import (
	"context"
	"testing"
)

// TestFetchPendingSkipLocked проверяет, что два конкурентных вызова FetchPending
// не забирают одну и ту же задачу дважды (SKIP LOCKED) и вместе покрывают
// все задачи без потерь.
func TestFetchPendingSkipLocked(t *testing.T) {
	ctx := context.Background()

	if _, err := TestPool.Exec(ctx,
		`TRUNCATE alert_rules, alerts, user_notification_channels, alert_notifications,
		 organizations, users, vehicles CASCADE`,
	); err != nil {
		t.Fatalf("truncate: %v", err)
	}

	// Подготовка зависимостей по цепочке FK: organizations -> vehicles/alert_rules/users
	// -> user_notification_channels -> alerts -> alert_notifications
	var orgID, vehicleID, ruleID, userID, channelID, alertID int

	if err := TestPool.QueryRow(ctx,
		`INSERT INTO organizations (name) VALUES ('Test Org') RETURNING id`,
	).Scan(&orgID); err != nil {
		t.Fatalf("insert organization: %v", err)
	}

	if err := TestPool.QueryRow(ctx,
		`INSERT INTO vehicles (organization_id, model) VALUES ($1, 'Test Model') RETURNING id`,
		orgID,
	).Scan(&vehicleID); err != nil {
		t.Fatalf("insert vehicle: %v", err)
	}

	if err := TestPool.QueryRow(ctx,
		`INSERT INTO alert_rules (organization_id, type, name, threshold, severity)
		 VALUES ($1, 'SPEED_EXCEEDED', 'Test Rule', 90, 'HIGH') RETURNING id`,
		orgID,
	).Scan(&ruleID); err != nil {
		t.Fatalf("insert alert_rule: %v", err)
	}

	if err := TestPool.QueryRow(ctx,
		`INSERT INTO users (organization_id, name, email, password_hash)
		 VALUES ($1, 'Test User', 'test@example.com', 'hash') RETURNING id`,
		orgID,
	).Scan(&userID); err != nil {
		t.Fatalf("insert user: %v", err)
	}

	if err := TestPool.QueryRow(ctx,
		`INSERT INTO user_notification_channels (user_id, type, config, min_severity)
		 VALUES ($1, 'EMAIL', '{}', 'LOW') RETURNING id`,
		userID,
	).Scan(&channelID); err != nil {
		t.Fatalf("insert channel: %v", err)
	}

	if err := TestPool.QueryRow(ctx,
		`INSERT INTO alerts (organization_id, vehicle_id, rule_id, type, severity, status, message)
		 VALUES ($1, $2, $3, 'SPEED_EXCEEDED', 'HIGH', 'FIRED', 'test alert') RETURNING id`,
		orgID, vehicleID, ruleID,
	).Scan(&alertID); err != nil {
		t.Fatalf("insert alert: %v", err)
	}

	const totalTasks = 10

	// Все N задач ссылаются на один и тот же alert/channel — для FetchPending
	// это не важно, ему нужны только валидные FK и status = 'PENDING'
	if _, err := TestPool.Exec(ctx,
		`INSERT INTO alert_notifications (alert_id, channel_id, status)
		 SELECT $1, $2, 'PENDING' FROM generate_series(1, $3)`,
		alertID, channelID, totalTasks,
	); err != nil {
		t.Fatalf("insert alert_notifications: %v", err)
	}

	tx1, err := TestPool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin tx1: %v", err)
	}
	repo1 := NewPostgresAlertNotificationRepository(tx1)

	tasksA, err := repo1.FetchPending(ctx, totalTasks)
	if err != nil {
		t.Fatalf("FetchPending tx1: %v", err)
	}

	tx2, err := TestPool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin tx2: %v", err)
	}
	repo2 := NewPostgresAlertNotificationRepository(tx2)

	tasksB, err := repo2.FetchPending(ctx, totalTasks)
	if err != nil {
		t.Fatalf("FetchPending tx2: %v", err)
	}

	defer tx1.Rollback(ctx)
	defer tx2.Rollback(ctx)

	if len(tasksA) != totalTasks {
		t.Errorf("expected worker A to fetch all %d tasks, got %d", totalTasks, len(tasksA))
	}
	if len(tasksB) != 0 {
		t.Errorf("expected worker B to fetch 0 tasks (all locked by tx1), got %d: %v", len(tasksB), tasksB)
	}
}
