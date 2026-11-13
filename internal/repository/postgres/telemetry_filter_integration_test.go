//go:build integration

package postgres

import (
	"context"
	"testing"
	"time"

	"fleettrack/internal/model"
)

// TestTelemetryListFiltersByTripAndDriver проверяет, что фильтры trip_id и driver_id реально сужают выборку.
// На driver_id держится scopeDriverID: если фильтр не применится, DRIVER увидит телеметрию всего парка
func TestTelemetryListFiltersByTripAndDriver(t *testing.T) {
	ctx := context.Background()

	if _, err := TestPool.Exec(ctx,
		`TRUNCATE organizations, drivers, vehicles, devices, trips, telemetry CASCADE`,
	); err != nil {
		t.Fatalf("truncate: %v", err)
	}

	var orgID, vehicleID, deviceID int
	if err := TestPool.QueryRow(ctx,
		`INSERT INTO organizations (name) VALUES ('Filter Org') RETURNING id`,
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
		`INSERT INTO devices (organization_id, serial_number) VALUES ($1, 'FILTER-001') RETURNING id`,
		orgID,
	).Scan(&deviceID); err != nil {
		t.Fatalf("insert device: %v", err)
	}

	// два водителя по очереди ездят на одной машине: у каждого свой рейс и свои точки
	driverIDs := make([]int, 2)
	tripIDs := make([]int, 2)
	for i, name := range []string{"First Driver", "Second Driver"} {
		if err := TestPool.QueryRow(ctx,
			`INSERT INTO drivers (organization_id, name) VALUES ($1, $2) RETURNING id`,
			orgID, name,
		).Scan(&driverIDs[i]); err != nil {
			t.Fatalf("insert driver: %v", err)
		}
		if err := TestPool.QueryRow(ctx,
			`INSERT INTO trips (driver_id, vehicle_id) VALUES ($1, $2) RETURNING id`,
			driverIDs[i], vehicleID,
		).Scan(&tripIDs[i]); err != nil {
			t.Fatalf("insert trip: %v", err)
		}
	}

	repo := NewPostgresTelemetryRepository(TestPool)
	at := time.Date(2026, 9, 15, 12, 0, 0, 0, time.UTC)
	// первому рейсу две точки, второму одна — чтобы счёт отличался
	for i, tripID := range []int{tripIDs[0], tripIDs[0], tripIDs[1]} {
		ts := at.Add(time.Duration(i) * time.Minute)
		if err := repo.Save(ctx, &model.Telemetry{
			OrganizationID:  orgID,
			DeviceID:        deviceID,
			VehicleID:       vehicleID,
			TripID:          tripID,
			Lat:             55.7558,
			Lon:             37.6173,
			ReceivedAt:      ts,
			DeviceTimestamp: ts,
		}); err != nil {
			t.Fatalf("save telemetry: %v", err)
		}
	}

	tests := []struct {
		name   string
		filter model.TelemetryFilter
		want   int
	}{
		{name: "без фильтра — все точки", filter: model.TelemetryFilter{}, want: 3},
		{name: "по рейсу", filter: model.TelemetryFilter{TripID: intPtr(tripIDs[0])}, want: 2},
		{name: "по водителю", filter: model.TelemetryFilter{DriverID: intPtr(driverIDs[1])}, want: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.filter.OrganizationID = intPtr(orgID)
			tt.filter.Limit = 100
			got, err := repo.GetList(ctx, tt.filter)
			if err != nil {
				t.Fatalf("get list: %v", err)
			}
			if len(got) != tt.want {
				t.Errorf("got %d points, want %d", len(got), tt.want)
			}
		})
	}
}
