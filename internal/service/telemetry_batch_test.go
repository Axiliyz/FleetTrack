package service

import (
	"context"
	"errors"
	"fleettrack/internal/logger"
	"fleettrack/internal/model"
	"testing"
)

func newIsolatedTelemetryService(vehicles *mockVehicleRepository, trips *mockTripRepository, assignments *fakeAssignmentRepo) (*TelemetryService, *mockRepository) {
	repo := &mockRepository{}
	rf := &fakeRepoFactory{telemetry: repo, trip: trips, vehicle: vehicles, assignment: assignments}
	motion := &fakeMotionService{data: &model.MotionData{DistanceKm: 1, SpeedKmh: 30}}
	svc := NewTelemetryService(repo, logger.NewStdLogger(logger.ErrorLevel), &fakeTxManager{}, rf, motion, nil, testBatchConfig)
	return svc, repo
}

func TestSaveInTx_ForeignOrganizationDoesNotTouchTrip(t *testing.T) {
	vehicles := &mockVehicleRepository{vehicle: &model.Vehicle{ID: 1, OrganizationID: 2}}
	trips := &mockTripRepository{trips: []model.Trip{{ID: 7, Status: model.TripStatusRunning}}}
	svc, repo := newIsolatedTelemetryService(vehicles, trips, &fakeAssignmentRepo{})

	results, err := svc.flushBatch(context.Background(), []model.Telemetry{
		{OrganizationID: 1, DeviceID: 1, VehicleID: 1, Lat: 55.7, Lon: 37.6},
	})
	if err != nil {
		t.Fatalf("flushBatch: %v", err)
	}
	if !errors.Is(results[0].err, model.ErrNotFound) {
		t.Fatalf("got %v, want ErrNotFound", results[0].err)
	}
	if trips.statsUpdates != 0 {
		t.Fatalf("trip stats updated %d times for a foreign vehicle", trips.statsUpdates)
	}
	if repo.saved != nil {
		t.Fatalf("telemetry saved for a foreign vehicle")
	}
}

func TestSaveInTx_DeviceNotAssignedToVehicle(t *testing.T) {
	vehicles := &mockVehicleRepository{vehicle: &model.Vehicle{ID: 1, OrganizationID: 1}}
	trips := &mockTripRepository{}
	assignments := &fakeAssignmentRepo{vehicleForDevice: map[int]int{5: 99}}
	svc, repo := newIsolatedTelemetryService(vehicles, trips, assignments)

	results, err := svc.flushBatch(context.Background(), []model.Telemetry{
		{OrganizationID: 1, DeviceID: 5, VehicleID: 1, Lat: 55.7, Lon: 37.6},
	})
	if err != nil {
		t.Fatalf("flushBatch: %v", err)
	}
	if !errors.Is(results[0].err, model.ErrDeviceNotAssigned) {
		t.Fatalf("got %v, want ErrDeviceNotAssigned", results[0].err)
	}
	if repo.saved != nil {
		t.Fatalf("telemetry saved from an unassigned device")
	}
}

func TestFlushBatch_ItemErrorDoesNotFailOthers(t *testing.T) {
	vehicles := &mockVehicleRepository{vehicle: &model.Vehicle{ID: 1, OrganizationID: 1}}
	svc, _ := newIsolatedTelemetryService(vehicles, &mockTripRepository{}, &fakeAssignmentRepo{vehicleForDevice: map[int]int{2: 99}})

	results, err := svc.flushBatch(context.Background(), []model.Telemetry{
		{OrganizationID: 1, DeviceID: 1, VehicleID: 1, Lat: 55.7, Lon: 37.6},
		{OrganizationID: 1, DeviceID: 2, VehicleID: 1, Lat: 55.7, Lon: 37.6},
		{OrganizationID: 1, DeviceID: 1, VehicleID: 1, Lat: 55.8, Lon: 37.7},
	})
	if err != nil {
		t.Fatalf("flushBatch: %v", err)
	}
	if results[0].err != nil || results[2].err != nil {
		t.Fatalf("valid items failed: %v, %v", results[0].err, results[2].err)
	}
	if !errors.Is(results[1].err, model.ErrDeviceNotAssigned) {
		t.Fatalf("got %v, want ErrDeviceNotAssigned", results[1].err)
	}
}
