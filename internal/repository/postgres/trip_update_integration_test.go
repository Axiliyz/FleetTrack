//go:build integration

package postgres

import (
	"context"
	"testing"

	"fleettrack/internal/model"
)

// TestUpdateTrip_KeepsStats проверяет, что завершённый рейс возвращается со статистикой, а не с нулями:
// этот ответ уходит и в PATCH /trips, и в событие trip на карте
func TestUpdateTrip_KeepsStats(t *testing.T) {
	ctx := context.Background()
	if _, err := TestPool.Exec(ctx, `TRUNCATE trips, drivers, organizations, vehicles CASCADE`); err != nil {
		t.Fatalf("truncate: %v", err)
	}
	var orgID, driverID, vehicleID int
	mustScan(ctx, t, &orgID, `INSERT INTO organizations (name) VALUES ('Org') RETURNING id`, nil)
	mustScan(ctx, t, &driverID, `INSERT INTO drivers (organization_id, name) VALUES ($1, 'Driver') RETURNING id`, []any{orgID})
	mustScan(ctx, t, &vehicleID, `INSERT INTO vehicles (organization_id, model) VALUES ($1, 'Model') RETURNING id`, []any{orgID})

	repo := NewPostgresTripRepository(TestPool)
	trip := model.Trip{DriverID: driverID, VehicleID: vehicleID, Status: model.TripStatusRunning}
	if err := repo.CreateTrip(ctx, &trip, &orgID); err != nil {
		t.Fatalf("CreateTrip: %v", err)
	}
	for _, speed := range []float64{80, 100} {
		if _, err := repo.UpdateTripStats(ctx, trip.ID, 10, speed); err != nil {
			t.Fatalf("UpdateTripStats: %v", err)
		}
	}

	done, err := repo.UpdateTrip(ctx, model.Trip{ID: trip.ID, Status: model.TripStatusSucceeded}, &orgID)
	if err != nil {
		t.Fatalf("UpdateTrip: %v", err)
	}
	if done.DistanceKm != 20 || done.AvgSpeedKmh != 90 || done.MaxSpeedKmh != 100 || done.TelemetryCount != 2 {
		t.Errorf("stats lost on finish: %+v", done)
	}
	if done.EndedAt == nil || done.Status != model.TripStatusSucceeded {
		t.Errorf("trip not finished: %+v", done)
	}
}
