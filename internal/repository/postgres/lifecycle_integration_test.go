//go:build integration

package postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	"fleettrack/internal/model"
)

// TestUser_SoftDeleteAndEmailCase проверяет мягкое удаление пользователя и почту без учёта регистра
func TestUser_SoftDeleteAndEmailCase(t *testing.T) {
	ctx := context.Background()
	org, _ := seedTwoOrgs(ctx, t)
	repo := NewPostgresUserRepository(TestPool)

	u := &model.User{OrganizationID: org, Name: "Саня", Email: "sanya@fleet.test", PasswordHash: "h", Role: model.UserRoleDispatcher}
	if err := repo.Create(ctx, u); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.GetByEmail(ctx, "Sanya@Fleet.TEST"); err != nil {
		t.Fatalf("lookup ignoring case: %v", err)
	}
	dup := &model.User{OrganizationID: org, Name: "Саня", Email: "SANYA@fleet.test", PasswordHash: "h", Role: model.UserRoleAnalytic}
	if err := repo.Create(ctx, dup); !errors.Is(err, model.ErrDuplicateEmail) {
		t.Fatalf("same email in other case: err = %v, want ErrDuplicateEmail", err)
	}

	// пользователь, который входил в систему, раньше не удалялся: refresh_tokens держали внешний ключ
	tokens := NewPostgresRefreshTokenRepository(TestPool)
	if err := tokens.Create(ctx, &model.RefreshToken{UserID: u.ID, TokenHash: "a1", ExpiresAt: time.Now().Add(time.Hour)}); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.DeleteByID(ctx, u.ID, &org); err != nil {
		t.Fatalf("delete user with refresh tokens: %v", err)
	}
	if _, err := repo.GetByID(ctx, u.ID); !errors.Is(err, model.ErrNotFound) {
		t.Errorf("deleted user is still visible: %v", err)
	}
	if _, err := repo.GetByEmail(ctx, "sanya@fleet.test"); !errors.Is(err, model.ErrNotFound) {
		t.Errorf("deleted user can still log in: %v", err)
	}
	// почта удалённого свободна
	again := &model.User{OrganizationID: org, Name: "Саня", Email: "sanya@fleet.test", PasswordHash: "h", Role: model.UserRoleDispatcher}
	if err := repo.Create(ctx, again); err != nil {
		t.Errorf("email of a deleted user is still taken: %v", err)
	}
}

// TestUser_DriverMustBelongToOrganization проверяет, что к аккаунту нельзя привязать водителя чужой организации
func TestUser_DriverMustBelongToOrganization(t *testing.T) {
	ctx := context.Background()
	org1, org2 := seedTwoOrgs(ctx, t)
	var foreignDriver, ownDriver int
	mustScan(ctx, t, &foreignDriver, `INSERT INTO drivers (organization_id, name) VALUES ($1, 'Чужой') RETURNING id`, []any{org2})
	mustScan(ctx, t, &ownDriver, `INSERT INTO drivers (organization_id, name) VALUES ($1, 'Свой') RETURNING id`, []any{org1})
	repo := NewPostgresUserRepository(TestPool)

	foreign := &model.User{OrganizationID: org1, DriverID: &foreignDriver, Name: "В", Email: "d1@fleet.test", PasswordHash: "h", Role: model.UserRoleDriver}
	if err := repo.Create(ctx, foreign); !errors.Is(err, model.ErrInvalidDriverID) {
		t.Fatalf("foreign driver: err = %v, want ErrInvalidDriverID", err)
	}
	own := &model.User{OrganizationID: org1, DriverID: &ownDriver, Name: "В", Email: "d2@fleet.test", PasswordHash: "h", Role: model.UserRoleDriver}
	if err := repo.Create(ctx, own); err != nil {
		t.Fatalf("own driver: %v", err)
	}
}

// TestUser_LockActiveAdmins проверяет, что в список админов не попадают удалённые и чужие
func TestUser_LockActiveAdmins(t *testing.T) {
	ctx := context.Background()
	org1, org2 := seedTwoOrgs(ctx, t)
	var a1, a2, deleted, foreign int
	for _, row := range []struct {
		dst   *int
		org   int
		email string
	}{{&a1, org1, "a1@x.t"}, {&a2, org1, "a2@x.t"}, {&deleted, org1, "a3@x.t"}, {&foreign, org2, "a4@x.t"}} {
		mustScan(ctx, t, row.dst, `INSERT INTO users (organization_id, name, email, password_hash, role)
			VALUES ($1, 'Админ', $2, 'h', 'ADMIN') RETURNING id`, []any{row.org, row.email})
	}
	if _, err := TestPool.Exec(ctx, `UPDATE users SET deleted_at = NOW() WHERE id = $1`, deleted); err != nil {
		t.Fatal(err)
	}

	tx, err := TestPool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	ids, err := NewPostgresUserRepository(tx).LockActiveAdmins(ctx, org1)
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 2 || ids[0] != a1 || ids[1] != a2 {
		t.Errorf("admins = %v, want [%d %d]", ids, a1, a2)
	}
}

// TestRefreshToken_FamilyAndCleanup проверяет семьи токенов, отзыв семьи и чистку истёкших
func TestRefreshToken_FamilyAndCleanup(t *testing.T) {
	ctx := context.Background()
	org, _ := seedTwoOrgs(ctx, t)
	var userID int
	mustScan(ctx, t, &userID, `INSERT INTO users (organization_id, name, email, password_hash, role)
		VALUES ($1, 'U', 'u@x.t', 'h', 'ADMIN') RETURNING id`, []any{org})
	repo := NewPostgresRefreshTokenRepository(TestPool)

	first := &model.RefreshToken{UserID: userID, TokenHash: "h1", ExpiresAt: time.Now().Add(time.Hour)}
	if err := repo.Create(ctx, first); err != nil {
		t.Fatal(err)
	}
	if first.FamilyID == "" {
		t.Fatal("first token got no family")
	}
	rotated := &model.RefreshToken{UserID: userID, FamilyID: first.FamilyID, TokenHash: "h2", ExpiresAt: time.Now().Add(time.Hour)}
	other := &model.RefreshToken{UserID: userID, TokenHash: "h3", ExpiresAt: time.Now().Add(time.Hour)}
	expired := &model.RefreshToken{UserID: userID, TokenHash: "h4", ExpiresAt: time.Now().Add(-48 * time.Hour)}
	for _, tok := range []*model.RefreshToken{rotated, other, expired} {
		if err := repo.Create(ctx, tok); err != nil {
			t.Fatal(err)
		}
	}

	if err := repo.RevokeFamily(ctx, first.FamilyID); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.GetActiveByHash(ctx, "h2"); !errors.Is(err, model.ErrNotFound) {
		t.Errorf("token of revoked family still active: %v", err)
	}
	if _, err := repo.GetActiveByHash(ctx, "h3"); err != nil {
		t.Errorf("token of another family revoked: %v", err)
	}
	if got, err := repo.GetByHash(ctx, "h2"); err != nil || got.RevokedAt == nil {
		t.Errorf("GetByHash must return revoked token: %+v, %v", got, err)
	}

	deleted, err := repo.DeleteExpired(ctx, time.Now().Add(-24*time.Hour))
	if err != nil || deleted != 1 {
		t.Errorf("DeleteExpired = %d, %v, want 1", deleted, err)
	}
}

// vehicleFixture - организация, машина с трекером и правило DEVICE_OFFLINE с открытым алертом
type vehicleFixture struct {
	org, vehicle, device, rule, alert int
}

func seedVehicleWithTracker(ctx context.Context, t *testing.T) vehicleFixture {
	t.Helper()
	var f vehicleFixture
	f.org, _ = seedTwoOrgs(ctx, t)
	mustScan(ctx, t, &f.vehicle, `INSERT INTO vehicles (organization_id, vin, number_plate, model) VALUES ($1, 'V1', 'P1', 'M') RETURNING id`, []any{f.org})
	mustScan(ctx, t, &f.device, `INSERT INTO devices (organization_id, serial_number) VALUES ($1, 'S1') RETURNING id`, []any{f.org})
	if _, err := TestPool.Exec(ctx,
		`INSERT INTO device_assignments (vehicle_id, device_id, started_at) VALUES ($1, $2, NOW() - interval '2 hours')`,
		f.vehicle, f.device); err != nil {
		t.Fatal(err)
	}
	mustScan(ctx, t, &f.rule, `INSERT INTO alert_rules (organization_id, type, name, threshold, severity)
		VALUES ($1, 'DEVICE_OFFLINE', 'Offline', 10, 'HIGH') RETURNING id`, []any{f.org})
	mustScan(ctx, t, &f.alert, `INSERT INTO alerts (organization_id, vehicle_id, rule_id, type, severity, status, message)
		VALUES ($1, $2, $3, 'DEVICE_OFFLINE', 'HIGH', 'FIRED', 'offline') RETURNING id`, []any{f.org, f.vehicle, f.rule})
	return f
}

// TestVehicle_StatusTransitions проверяет атомарный переход статуса и запреты для машины в рейсе
func TestVehicle_StatusTransitions(t *testing.T) {
	ctx := context.Background()
	f := seedVehicleWithTracker(ctx, t)
	repo := NewPostgresVehicleRepository(TestPool)

	if _, err := repo.UpdateStatus(ctx, f.vehicle, model.VehicleStatusIdle, model.VehicleStatusOnTrip, &f.org); err != nil {
		t.Fatalf("IDLE -> ON_TRIP: %v", err)
	}
	// второй рейс ту же машину уже не займёт
	if _, err := repo.UpdateStatus(ctx, f.vehicle, model.VehicleStatusIdle, model.VehicleStatusOnTrip, &f.org); !errors.Is(err, model.ErrNotFound) {
		t.Fatalf("second IDLE -> ON_TRIP: err = %v, want ErrNotFound", err)
	}

	status := model.VehicleStatusInService
	if _, err := repo.Update(ctx, f.vehicle, model.UpdateVehicle{Status: &status}, &f.org); !errors.Is(err, model.ErrVehicleOnTrip) {
		t.Errorf("manual status of vehicle on trip: err = %v, want ErrVehicleOnTrip", err)
	}
	plate := "P2"
	if _, err := repo.Update(ctx, f.vehicle, model.UpdateVehicle{NumberPlate: &plate}, &f.org); err != nil {
		t.Errorf("plate of vehicle on trip must be editable: %v", err)
	}
	if _, err := repo.Delete(ctx, f.vehicle, &f.org); !errors.Is(err, model.ErrVehicleIsBusy) {
		t.Errorf("delete vehicle on trip: err = %v, want ErrVehicleIsBusy", err)
	}
	other := f.org + 1
	if _, err := repo.Delete(ctx, f.vehicle, &other); !errors.Is(err, model.ErrNotFound) {
		t.Errorf("delete from foreign organization: err = %v, want ErrNotFound", err)
	}
}

// TestVehicle_DeletedHasNoOfflineAlerts проверяет, что удалённая машина не всплывает в поиске
// «молчащих» трекеров, а её открытые алерты закрываются
func TestVehicle_DeletedHasNoOfflineAlerts(t *testing.T) {
	ctx := context.Background()
	f := seedVehicleWithTracker(ctx, t)
	alerts := NewPostgresAlertRepository(TestPool)

	offline, err := alerts.FindOfflineVehicles(ctx, 10)
	if err != nil || len(offline) != 1 {
		t.Fatalf("before delete: %v, %v", offline, err)
	}

	if _, err := NewPostgresVehicleRepository(TestPool).Delete(ctx, f.vehicle, &f.org); err != nil {
		t.Fatal(err)
	}
	// даже если привязку забыли закрыть, удалённая машина в поиск не попадает
	if offline, err = alerts.FindOfflineVehicles(ctx, 10); err != nil || len(offline) != 0 {
		t.Errorf("deleted vehicle is still offline: %v, %v", offline, err)
	}

	if err := NewPostgresAssignmentRepository(TestPool).EndAssignmentByVehicle(ctx, f.vehicle); err != nil {
		t.Fatal(err)
	}
	resolved, err := alerts.ResolveByVehicle(ctx, f.vehicle)
	if err != nil || len(resolved) != 1 || resolved[0].ID != f.alert || resolved[0].Status != model.AlertStatusResolved {
		t.Errorf("ResolveByVehicle = %+v, %v", resolved, err)
	}
}

// TestAlertRule_SoftDelete проверяет, что удалённое правило не проверяется и не видно, но его алерты сохраняют ссылку
func TestAlertRule_SoftDelete(t *testing.T) {
	ctx := context.Background()
	f := seedVehicleWithTracker(ctx, t)
	rules := NewPostgresAlertRuleRepository(TestPool)

	if _, err := rules.DeleteByID(ctx, f.rule); err != nil {
		t.Fatalf("delete rule that already fired: %v", err)
	}
	if _, err := rules.GetByID(ctx, f.rule); !errors.Is(err, model.ErrNotFound) {
		t.Errorf("deleted rule visible: %v", err)
	}
	active, err := rules.GetActiveRulesByOrg(ctx, f.org)
	if err != nil || len(active) != 0 {
		t.Errorf("deleted rule still active: %v, %v", active, err)
	}
	list, err := rules.GetList(ctx, model.AlertRuleFilter{OrganizationID: &f.org})
	if err != nil || len(list) != 0 {
		t.Errorf("deleted rule in list: %v, %v", list, err)
	}
	var ruleID int
	mustScan(ctx, t, &ruleID, `SELECT rule_id FROM alerts WHERE id = $1`, []any{f.alert})
	if ruleID != f.rule {
		t.Errorf("alert lost its rule: %d", ruleID)
	}
}

// TestTelemetry_DriverScopeByID проверяет, что точка чужого рейса для водителя неотличима от несуществующей
func TestTelemetry_DriverScopeByID(t *testing.T) {
	ctx := context.Background()
	f := seedVehicleWithTracker(ctx, t)
	var driver, otherDriver, trip int
	mustScan(ctx, t, &driver, `INSERT INTO drivers (organization_id, name) VALUES ($1, 'D') RETURNING id`, []any{f.org})
	mustScan(ctx, t, &otherDriver, `INSERT INTO drivers (organization_id, name) VALUES ($1, 'O') RETURNING id`, []any{f.org})
	mustScan(ctx, t, &trip, `INSERT INTO trips (driver_id, vehicle_id) VALUES ($1, $2) RETURNING id`, []any{driver, f.vehicle})

	repo := NewPostgresTelemetryRepository(TestPool)
	p := model.Telemetry{OrganizationID: f.org, VehicleID: f.vehicle, DeviceID: f.device, TripID: trip, Lat: 55.7, Lon: 37.6, ReceivedAt: time.Now(), DeviceTimestamp: time.Now()}
	if err := repo.Save(ctx, &p); err != nil {
		t.Fatal(err)
	}

	if _, err := repo.GetItemByID(ctx, p.TelemetryID, &driver); err != nil {
		t.Errorf("own driver: %v", err)
	}
	if _, err := repo.GetItemByID(ctx, p.TelemetryID, &otherDriver); !errors.Is(err, model.ErrNotFound) {
		t.Errorf("other driver: err = %v, want ErrNotFound", err)
	}
	if list, err := repo.GetListByVehicle(ctx, f.vehicle, &f.org, &otherDriver); !errors.Is(err, model.ErrNotFound) {
		t.Errorf("other driver by vehicle: %v, %v", list, err)
	}

	deleted, err := repo.DeleteListByVehicle(ctx, f.vehicle, &f.org)
	if err != nil || deleted != 1 {
		t.Errorf("DeleteListByVehicle = %d, %v, want 1", deleted, err)
	}
}
