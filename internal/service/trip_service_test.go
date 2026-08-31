package service

import (
	"context"
	"fleettrack/internal/logger"
	"fleettrack/internal/model"
	"testing"
	"time"
)

type mockTripRepository struct {
	createErr error
	listErr   error
	updateErr error
	deleteErr error
}

func (m *mockTripRepository) CreateTrip(ctx context.Context, t *model.Trip) error {
	if m.createErr != nil {
		return m.createErr
	}
	t.ID = 1
	return nil
}

func (m *mockTripRepository) GetListTrips(ctx context.Context, f *model.TripFilter) ([]model.Trip, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	return []model.Trip{}, nil
}

func (m *mockTripRepository) UpdateTrip(ctx context.Context, upd model.Trip) (model.Trip, error) {
	if m.updateErr != nil {
		return model.Trip{}, m.updateErr
	}
	return model.Trip{ID: upd.ID, Status: upd.Status}, nil
}

func (m *mockTripRepository) DeleteTrip(ctx context.Context, id int) (model.Trip, error) {
	if m.deleteErr != nil {
		return model.Trip{}, m.deleteErr
	}
	return model.Trip{ID: id, Status: model.TripStatusCancelled}, nil
}

func TestAssignTrip(t *testing.T) {
	tests := []struct {
		name      string
		driverID  int
		vehicleID int
		wantErr   error
	}{
		{name: "valid", driverID: 1, vehicleID: 1, wantErr: nil},
		{name: "invalid driver id", driverID: 0, vehicleID: 1, wantErr: model.ErrInvalidDriverID},
		{name: "negative driver id", driverID: -1, vehicleID: 1, wantErr: model.ErrInvalidDriverID},
		{name: "invalid vehicle id", driverID: 1, vehicleID: 0, wantErr: model.ErrInvalidVehicleID},
	}

	repo := &mockTripRepository{}
	log := logger.NewStdLogger(logger.DebugLevel)
	svc := NewTripService(repo, log)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			trip, err := svc.AssignTrip(context.Background(), tt.driverID, tt.vehicleID)
			if err != tt.wantErr {
				t.Errorf("got %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr == nil && trip.Status != model.TripStatusRunning {
				t.Errorf("got status %v, want RUNNING", trip.Status)
			}
		})
	}
}

func TestAssignTrip_RepositoryError(t *testing.T) {
	repo := &mockTripRepository{createErr: model.ErrInvalidVehicleID}
	log := logger.NewStdLogger(logger.DebugLevel)
	svc := NewTripService(repo, log)

	_, err := svc.AssignTrip(context.Background(), 1, 1)
	if err != model.ErrInvalidVehicleID {
		t.Errorf("got %v, want %v", err, model.ErrInvalidVehicleID)
	}
}

func TestTripServiceUpdateTrip(t *testing.T) {
	tests := []struct {
		name    string
		id      int
		status  model.TripStatus
		wantErr error
	}{
		{name: "valid", id: 1, status: model.TripStatusSucceeded, wantErr: nil},
		{name: "invalid id", id: 0, status: model.TripStatusSucceeded, wantErr: model.ErrInvalidTripID},
		{name: "invalid status", id: 1, status: model.TripStatus("BOGUS"), wantErr: model.ErrInvalidStatus},
	}

	repo := &mockTripRepository{}
	log := logger.NewStdLogger(logger.DebugLevel)
	svc := NewTripService(repo, log)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.UpdateTrip(context.Background(), tt.id, model.Trip{Status: tt.status})
			if err != tt.wantErr {
				t.Errorf("got %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestTripServiceUpdateTrip_RepositoryError(t *testing.T) {
	repo := &mockTripRepository{updateErr: model.ErrTripAlreadyFinished}
	log := logger.NewStdLogger(logger.DebugLevel)
	svc := NewTripService(repo, log)

	_, err := svc.UpdateTrip(context.Background(), 1, model.Trip{Status: model.TripStatusRunning})
	if err != model.ErrTripAlreadyFinished {
		t.Errorf("got %v, want %v", err, model.ErrTripAlreadyFinished)
	}
}

func TestTripServiceDeleteTrip(t *testing.T) {
	repo := &mockTripRepository{}
	log := logger.NewStdLogger(logger.DebugLevel)
	svc := NewTripService(repo, log)

	trip, err := svc.DeleteTrip(context.Background(), 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if trip.Status != model.TripStatusCancelled {
		t.Errorf("got status %v, want CANCELLED", trip.Status)
	}

	_, err = svc.DeleteTrip(context.Background(), 0)
	if err != model.ErrInvalidTripID {
		t.Errorf("got %v, want %v", err, model.ErrInvalidTripID)
	}

	repo.deleteErr = model.ErrTripAlreadyFinished
	_, err = svc.DeleteTrip(context.Background(), 5)
	if err != model.ErrTripAlreadyFinished {
		t.Errorf("got %v, want %v", err, model.ErrTripAlreadyFinished)
	}
}

func TestGetListTrips(t *testing.T) {
	tests := []struct {
		name    string
		filter  model.TripFilter
		wantErr error
	}{
		{name: "no filters", filter: model.TripFilter{Limit: 100}, wantErr: nil},
		{name: "invalid driver id", filter: model.TripFilter{DriverID: intPtr(0), Limit: 100}, wantErr: model.ErrInvalidDriverID},
		{name: "invalid vehicle id", filter: model.TripFilter{VehicleID: intPtr(0), Limit: 100}, wantErr: model.ErrInvalidVehicleID},
		{
			name: "started_from after started_to",
			filter: model.TripFilter{
				StartedFrom: timePtr(time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)),
				StartedTo:   timePtr(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)),
				Limit:       100,
			},
			wantErr: model.ErrInvalidTimestamp,
		},
	}

	repo := &mockTripRepository{}
	log := logger.NewStdLogger(logger.DebugLevel)
	svc := NewTripService(repo, log)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.GetListTrips(context.Background(), tt.filter)
			if err != tt.wantErr {
				t.Errorf("got %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetListTrips_RepositoryError(t *testing.T) {
	repo := &mockTripRepository{listErr: model.ErrNotFound}
	log := logger.NewStdLogger(logger.DebugLevel)
	svc := NewTripService(repo, log)

	_, err := svc.GetListTrips(context.Background(), model.TripFilter{Limit: 100})
	if err != model.ErrNotFound {
		t.Errorf("got %v, want %v", err, model.ErrNotFound)
	}
}
