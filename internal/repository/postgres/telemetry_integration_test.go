//go:build integration

package postgres

import (
	"context"
	"testing"
	"time"

	"fleettrack/internal/model"
)

func TestTelemetryPartitioning(t *testing.T) {
	ctx := context.Background()

	if _, err := TestPool.Exec(ctx,
		`TRUNCATE organizations, drivers, vehicles, devices, trips, telemetry CASCADE`,
	); err != nil {
		t.Fatalf("truncate: %v", err)
	}

	var orgID, driverID, vehicleID, deviceID, tripID int

	if err := TestPool.QueryRow(ctx,
		`INSERT INTO organizations (name) VALUES ('Test Org') RETURNING id`,
	).Scan(&orgID); err != nil {
		t.Fatalf("insert organization: %v", err)
	}

	if err := TestPool.QueryRow(ctx,
		`INSERT INTO drivers (organization_id, name) VALUES ($1, 'Test Driver') RETURNING id`,
		orgID,
	).Scan(&driverID); err != nil {
		t.Fatalf("insert driver: %v", err)
	}

	if err := TestPool.QueryRow(ctx,
		`INSERT INTO vehicles (organization_id, model) VALUES ($1, 'Test Model') RETURNING id`,
		orgID,
	).Scan(&vehicleID); err != nil {
		t.Fatalf("insert vehicle: %v", err)
	}

	if err := TestPool.QueryRow(ctx,
		`INSERT INTO devices (organization_id, serial_number) VALUES ($1, 'TEST-SERIAL-001') RETURNING id`,
		orgID,
	).Scan(&deviceID); err != nil {
		t.Fatalf("insert device: %v", err)
	}

	if err := TestPool.QueryRow(ctx,
		`INSERT INTO trips (driver_id, vehicle_id) VALUES ($1, $2) RETURNING id`,
		driverID, vehicleID,
	).Scan(&tripID); err != nil {
		t.Fatalf("insert trip: %v", err)
	}

	repo := NewPostgresTelemetryRepository(TestPool)

	telemetry := &model.Telemetry{
		OrganizationID:  orgID,
		DeviceID:        deviceID,
		VehicleID:       vehicleID,
		TripID:          tripID,
		Lat:             55.7558,
		Lon:             37.6173,
		ReceivedAt:      time.Date(2026, 9, 15, 12, 0, 0, 0, time.UTC),
		DeviceTimestamp: time.Date(2026, 9, 15, 12, 0, 0, 0, time.UTC),
	}

	if err := repo.Save(ctx, telemetry); err != nil {
		t.Fatalf("save telemetry: %v", err)
	}

	var partition string
	if err := TestPool.QueryRow(ctx,
		`SELECT tableoid::regclass::text FROM telemetry WHERE id = $1`,
		telemetry.TelemetryID,
	).Scan(&partition); err != nil {
		t.Fatalf("select tableoid: %v", err)
	}

	const wantPartition = "telemetry_2026_09"
	if partition != wantPartition {
		t.Errorf("expected row in partition %q, got %q", wantPartition, partition)
	}
}
