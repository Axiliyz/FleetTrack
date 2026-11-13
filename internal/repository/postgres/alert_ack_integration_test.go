//go:build integration

package postgres

import (
	"context"
	"errors"
	"testing"

	"fleettrack/internal/model"
)

// alertAckFixture — две организации, в каждой машина, правило, диспетчер и открытый алерт
type alertAckFixture struct {
	org1, org2     int
	user1, user2   int
	alert1, alert2 int
}

func seedAlertAck(ctx context.Context, t *testing.T) alertAckFixture {
	t.Helper()
	if _, err := TestPool.Exec(ctx,
		`TRUNCATE alert_rules, alerts, user_notification_channels, alert_notifications,
		 organizations, users, vehicles CASCADE`,
	); err != nil {
		t.Fatalf("truncate: %v", err)
	}

	var f alertAckFixture
	for i, dst := range []struct{ org, user, alert *int }{
		{&f.org1, &f.user1, &f.alert1},
		{&f.org2, &f.user2, &f.alert2},
	} {
		var vehicleID, ruleID int
		mustScan(ctx, t, dst.org, `INSERT INTO organizations (name) VALUES ($1) RETURNING id`, []any{"Org " + string(rune('A'+i))})
		mustScan(ctx, t, &vehicleID, `INSERT INTO vehicles (organization_id, model) VALUES ($1, 'Test Model') RETURNING id`, []any{*dst.org})
		mustScan(ctx, t, &ruleID,
			`INSERT INTO alert_rules (organization_id, type, name, threshold, severity)
			 VALUES ($1, 'SPEED_EXCEEDED', 'Speed', 90, 'HIGH') RETURNING id`, []any{*dst.org})
		mustScan(ctx, t, dst.user,
			`INSERT INTO users (organization_id, name, email, password_hash, role)
			 VALUES ($1, 'Dispatcher', $2, 'hash', 'DISPATCHER') RETURNING id`,
			[]any{*dst.org, string(rune('a'+i)) + "@example.com"})
		mustScan(ctx, t, dst.alert,
			`INSERT INTO alerts (organization_id, vehicle_id, rule_id, type, severity, status, message, value)
			 VALUES ($1, $2, $3, 'SPEED_EXCEEDED', 'HIGH', 'FIRED', '104 км/ч', 104) RETURNING id`,
			[]any{*dst.org, vehicleID, ruleID})
	}
	return f
}

func mustScan(ctx context.Context, t *testing.T, dst *int, query string, args []any) {
	t.Helper()
	if err := TestPool.QueryRow(ctx, query, args...).Scan(dst); err != nil {
		t.Fatalf("%s: %v", query, err)
	}
}

// TestAcknowledgeAlert_ForeignOrganization проверяет, что чужой алерт нельзя принять и он остаётся нетронутым в базе
func TestAcknowledgeAlert_ForeignOrganization(t *testing.T) {
	ctx := context.Background()
	f := seedAlertAck(ctx, t)
	repo := NewPostgresAlertRepository(TestPool)

	_, err := repo.AcknowledgeAlert(ctx, f.alert2, f.user1, f.org1)
	if !errors.Is(err, model.ErrNotFound) {
		t.Fatalf("ack foreign alert: err = %v, want ErrNotFound", err)
	}

	// главное — не ответ, а состояние базы: строка не должна была измениться ни на мгновение
	var status string
	var ackBy *int
	if err := TestPool.QueryRow(ctx, `SELECT status, acknowledged_by FROM alerts WHERE id = $1`, f.alert2).Scan(&status, &ackBy); err != nil {
		t.Fatalf("select: %v", err)
	}
	if status != string(model.AlertStatusFired) || ackBy != nil {
		t.Fatalf("foreign alert changed: status=%s acknowledged_by=%v", status, ackBy)
	}
}

// TestAcknowledgeAlert_OwnAndRepeat проверяет успешное принятие и повторную попытку
func TestAcknowledgeAlert_OwnAndRepeat(t *testing.T) {
	ctx := context.Background()
	f := seedAlertAck(ctx, t)
	repo := NewPostgresAlertRepository(TestPool)

	a, err := repo.AcknowledgeAlert(ctx, f.alert1, f.user1, f.org1)
	if err != nil {
		t.Fatalf("ack own alert: %v", err)
	}
	if a.Status != model.AlertStatusAcknowledged || a.AcknowledgedBy == nil || *a.AcknowledgedBy != f.user1 || a.AcknowledgedAt == nil {
		t.Fatalf("unexpected alert after ack: %+v", a)
	}

	if _, err := repo.AcknowledgeAlert(ctx, f.alert1, f.user1, f.org1); !errors.Is(err, model.ErrNotFound) {
		t.Fatalf("repeat ack: err = %v, want ErrNotFound", err)
	}
}

// TestAlertGetByID_ScopedByOrganization проверяет, что чужой алерт неотличим от несуществующего
func TestAlertGetByID_ScopedByOrganization(t *testing.T) {
	ctx := context.Background()
	f := seedAlertAck(ctx, t)
	repo := NewPostgresAlertRepository(TestPool)

	if a, err := repo.GetByID(ctx, f.alert1, f.org1); err != nil || a.ID != f.alert1 {
		t.Fatalf("own alert: a=%+v err=%v", a, err)
	}
	if _, err := repo.GetByID(ctx, f.alert1, f.org2); !errors.Is(err, model.ErrNotFound) {
		t.Fatalf("foreign alert: err = %v, want ErrNotFound", err)
	}
	if _, err := repo.GetByID(ctx, 999_999, f.org1); !errors.Is(err, model.ErrNotFound) {
		t.Fatalf("missing alert: err = %v, want ErrNotFound", err)
	}
}

// TestAlertGetList_OrganizationAndLimit проверяет изоляцию и пагинацию списка
func TestAlertGetList_OrganizationAndLimit(t *testing.T) {
	ctx := context.Background()
	f := seedAlertAck(ctx, t)
	repo := NewPostgresAlertRepository(TestPool)

	alerts, err := repo.GetList(ctx, model.AlertFilter{OrganizationID: &f.org1, Limit: 10})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(alerts) != 1 || alerts[0].ID != f.alert1 {
		t.Fatalf("got %+v, want only alert %d", alerts, f.alert1)
	}

	all, err := repo.GetList(ctx, model.AlertFilter{Limit: 1})
	if err != nil {
		t.Fatalf("list with limit: %v", err)
	}
	if len(all) != 1 {
		t.Fatalf("limit 1 returned %d rows", len(all))
	}
}

// TestGetAuthContextByTelegramChatID проверяет JOIN канала с пользователем
func TestGetAuthContextByTelegramChatID(t *testing.T) {
	ctx := context.Background()
	f := seedAlertAck(ctx, t)
	repo := NewUserNotificationChannelRepository(TestPool)

	if _, err := TestPool.Exec(ctx,
		`INSERT INTO user_notification_channels (user_id, type, config, min_severity)
		 VALUES ($1, 'TELEGRAM', '{"chat_id": 555}', 'LOW')`, f.user1,
	); err != nil {
		t.Fatalf("insert channel: %v", err)
	}

	got, err := repo.GetAuthContextByTelegramChatID(ctx, 555)
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	want := model.AuthContext{UserID: f.user1, OrganizationID: f.org1, Role: model.UserRoleDispatcher}
	if got != want {
		t.Fatalf("got %+v, want %+v", got, want)
	}

	if _, err := repo.GetAuthContextByTelegramChatID(ctx, 777); !errors.Is(err, model.ErrNotFound) {
		t.Fatalf("unknown chat: err = %v, want ErrNotFound", err)
	}
}
