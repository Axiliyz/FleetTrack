//go:build integration

package postgres

import (
	"context"
	"testing"
	"time"
)

// snapshotFixture — организация 1 с тремя машинами (в рейсе, без телеметрии, удалённая) и чужая организация 2
type snapshotFixture struct {
	org1, org2                     int
	onTrip, silent, deleted, other int
	newestPointID                  int
	oldRunningTrip, recentDoneTrip int
	oldDoneTrip, otherTrip         int
	oldOpenAlert, recentClosed     int
	oldClosedAlert, otherAlert     int
}

func seedSnapshot(ctx context.Context, t *testing.T) snapshotFixture {
	t.Helper()
	if _, err := TestPool.Exec(ctx,
		`TRUNCATE telemetry, trips, device_assignments, devices, alert_rules, alerts,
		 user_notification_channels, alert_notifications, drivers, organizations, users, vehicles CASCADE`,
	); err != nil {
		t.Fatalf("truncate: %v", err)
	}

	var f snapshotFixture
	var driver1, driver2, device1, device2, rule1, rule2 int
	mustScan(ctx, t, &f.org1, `INSERT INTO organizations (name) VALUES ('Org 1') RETURNING id`, nil)
	mustScan(ctx, t, &f.org2, `INSERT INTO organizations (name) VALUES ('Org 2') RETURNING id`, nil)
	mustScan(ctx, t, &driver1, `INSERT INTO drivers (organization_id, name) VALUES ($1, 'Driver 1') RETURNING id`, []any{f.org1})
	mustScan(ctx, t, &driver2, `INSERT INTO drivers (organization_id, name) VALUES ($1, 'Driver 2') RETURNING id`, []any{f.org2})
	mustScan(ctx, t, &device1, `INSERT INTO devices (organization_id, serial_number) VALUES ($1, 'SNAP-1') RETURNING id`, []any{f.org1})
	mustScan(ctx, t, &device2, `INSERT INTO devices (organization_id, serial_number) VALUES ($1, 'SNAP-2') RETURNING id`, []any{f.org2})

	vehicle := `INSERT INTO vehicles (organization_id, model, status) VALUES ($1, 'Test Model', $2) RETURNING id`
	mustScan(ctx, t, &f.onTrip, vehicle, []any{f.org1, "ON_TRIP"})
	mustScan(ctx, t, &f.silent, vehicle, []any{f.org1, "IDLE"})
	mustScan(ctx, t, &f.deleted, vehicle, []any{f.org1, "DELETED"})
	mustScan(ctx, t, &f.other, vehicle, []any{f.org2, "ON_TRIP"})

	trip := `INSERT INTO trips (driver_id, vehicle_id, started_at, ended_at, status)
		VALUES ($1, $2, NOW() - $3::interval, $4::timestamptz, $5) RETURNING id`
	mustScan(ctx, t, &f.oldRunningTrip, trip, []any{driver1, f.onTrip, "3 days", nil, "RUNNING"})
	mustScan(ctx, t, &f.recentDoneTrip, trip, []any{driver1, f.silent, "2 hours", time.Now().Add(-time.Hour), "SUCCEEDED"})
	mustScan(ctx, t, &f.oldDoneTrip, trip, []any{driver1, f.silent, "3 days", time.Now().Add(-70 * time.Hour), "SUCCEEDED"})
	mustScan(ctx, t, &f.otherTrip, trip, []any{driver2, f.other, "1 hour", nil, "RUNNING"})

	point := `INSERT INTO telemetry (organization_id, device_id, vehicle_id, latitude, longitude, fuel,
		received_at, device_timestamp, trip_id, distance_km, speed_kmh)
		VALUES ($1, $2, $3, $4, $5, $6, NOW() - $7::interval, NOW() - $7::interval, $8, 1.5, 80) RETURNING id`
	var ignored int
	mustScan(ctx, t, &ignored, point, []any{f.org1, device1, f.onTrip, 54.0, 37.0, 70, "3 days", f.oldRunningTrip})
	mustScan(ctx, t, &ignored, point, []any{f.org1, device1, f.onTrip, 55.7, 37.6, 61, "10 minutes", f.oldRunningTrip})
	// самая свежая точка — без рейса: trip_id в telemetry допускает NULL
	mustScan(ctx, t, &f.newestPointID, point, []any{f.org1, device1, f.onTrip, 55.8, 37.7, 60, "1 minute", nil})
	mustScan(ctx, t, &ignored, point, []any{f.org1, device1, f.deleted, 56.0, 38.0, 50, "1 minute", nil})
	mustScan(ctx, t, &ignored, point, []any{f.org2, device2, f.other, 59.9, 30.3, 40, "1 minute", f.otherTrip})

	// last_telemetry_at намеренно чуть позади самой свежей точки — запрос должен её всё равно найти
	if _, err := TestPool.Exec(ctx,
		`UPDATE vehicles SET last_telemetry_at = NOW() - INTERVAL '5 minutes' WHERE id = ANY($1)`,
		[]int{f.onTrip, f.deleted, f.other},
	); err != nil {
		t.Fatalf("update last_telemetry_at: %v", err)
	}

	rule := `INSERT INTO alert_rules (organization_id, type, name, threshold, severity)
		VALUES ($1, 'SPEED_EXCEEDED', 'Speed', 90, 'HIGH') RETURNING id`
	mustScan(ctx, t, &rule1, rule, []any{f.org1})
	mustScan(ctx, t, &rule2, rule, []any{f.org2})

	alert := `INSERT INTO alerts (organization_id, vehicle_id, rule_id, type, severity, status, message, created_at)
		VALUES ($1, $2, $3, 'SPEED_EXCEEDED', 'HIGH', $4, 'test', NOW() - $5::interval) RETURNING id`
	mustScan(ctx, t, &f.oldOpenAlert, alert, []any{f.org1, f.onTrip, rule1, "ACKNOWLEDGED", "3 days"})
	mustScan(ctx, t, &f.recentClosed, alert, []any{f.org1, f.onTrip, rule1, "RESOLVED", "1 hour"})
	mustScan(ctx, t, &f.oldClosedAlert, alert, []any{f.org1, f.onTrip, rule1, "RESOLVED", "3 days 1 hour"})
	mustScan(ctx, t, &f.otherAlert, alert, []any{f.org2, f.other, rule2, "FIRED", "1 hour"})
	return f
}

// TestSnapshotLatestPositions проверяет выбор последней точки по каждой машине
func TestSnapshotLatestPositions(t *testing.T) {
	ctx := context.Background()
	f := seedSnapshot(ctx, t)
	repo := NewPostgresSnapshotRepository(TestPool)

	points, err := repo.LatestPositions(ctx, f.org1, 100)
	if err != nil {
		t.Fatalf("LatestPositions: %v", err)
	}
	// машина без телеметрии, удалённая машина и чужая организация в ответ не попадают
	if len(points) != 1 {
		t.Fatalf("got %d points: %+v, want 1", len(points), points)
	}
	p := points[0]
	if p.VehicleID != f.onTrip || p.TelemetryID != f.newestPointID {
		t.Fatalf("point = %+v, want newest point %d of vehicle %d", p, f.newestPointID, f.onTrip)
	}
	if p.Lat != 55.8 || p.Lon != 37.7 || p.Fuel == nil || *p.Fuel != 60 || p.TripID != 0 || p.OrganizationID != f.org1 {
		t.Errorf("point fields = %+v", p)
	}
}

// TestSnapshotShiftTrips проверяет окно смены для рейсов
func TestSnapshotShiftTrips(t *testing.T) {
	ctx := context.Background()
	f := seedSnapshot(ctx, t)
	repo := NewPostgresSnapshotRepository(TestPool)

	trips, err := repo.ShiftTrips(ctx, f.org1, time.Now().Add(-24*time.Hour), 100)
	if err != nil {
		t.Fatalf("ShiftTrips: %v", err)
	}
	got := map[int]bool{}
	for _, tr := range trips {
		got[tr.ID] = true
	}
	if !got[f.oldRunningTrip] || !got[f.recentDoneTrip] {
		t.Errorf("missing trips: running=%v recent=%v", got[f.oldRunningTrip], got[f.recentDoneTrip])
	}
	if got[f.oldDoneTrip] || got[f.otherTrip] || len(trips) != 2 {
		t.Errorf("unexpected trips: %+v", trips)
	}
}

// TestSnapshotShiftAlerts проверяет окно смены для алертов
func TestSnapshotShiftAlerts(t *testing.T) {
	ctx := context.Background()
	f := seedSnapshot(ctx, t)
	repo := NewPostgresSnapshotRepository(TestPool)

	alerts, err := repo.ShiftAlerts(ctx, f.org1, time.Now().Add(-24*time.Hour), 100)
	if err != nil {
		t.Fatalf("ShiftAlerts: %v", err)
	}
	got := map[int]bool{}
	for _, a := range alerts {
		got[a.ID] = true
	}
	if !got[f.oldOpenAlert] || !got[f.recentClosed] {
		t.Errorf("missing alerts: old open=%v recent closed=%v", got[f.oldOpenAlert], got[f.recentClosed])
	}
	if got[f.oldClosedAlert] || got[f.otherAlert] || len(alerts) != 2 {
		t.Errorf("unexpected alerts: %+v", alerts)
	}
}
