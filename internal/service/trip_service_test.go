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
	getErr    error
	trips     []model.Trip

	statsUpdates int
}

func (m *mockTripRepository) GetByID(ctx context.Context, id int, organizationID *int) (model.Trip, error) {
	if m.getErr != nil {
		return model.Trip{}, m.getErr
	}
	return model.Trip{ID: id}, nil
}

func (m *mockTripRepository) CreateTrip(ctx context.Context, t *model.Trip, organizationID *int) error {
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
	return m.trips, nil
}

func (m *mockTripRepository) UpdateTrip(ctx context.Context, upd model.Trip, organizationID *int) (model.Trip, error) {
	if m.updateErr != nil {
		return model.Trip{}, m.updateErr
	}
	return model.Trip{ID: upd.ID, VehicleID: 7, Status: upd.Status}, nil
}

func (m *mockTripRepository) DeleteTrip(ctx context.Context, id int, organizationID *int) (model.Trip, error) {
	if m.deleteErr != nil {
		return model.Trip{}, m.deleteErr
	}
	return model.Trip{ID: id, Status: model.TripStatusCancelled}, nil
}

func (m *mockTripRepository) UpdateTripStats(ctx context.Context, id int, distance, speed float64) (model.Trip, error) {
	m.statsUpdates++
	return model.Trip{}, nil
}

// newTestTripService собирает TripService, у которого транзакция работает с теми же моками рейсов и машин
func newTestTripService(repo *mockTripRepository, vehicles *mockVehicleRepository, log logger.Logger) *TripService {
	rf := &fakeRepoFactory{trip: repo, vehicle: vehicles}
	return NewTripService(repo, log, &fakeTxManager{}, rf)
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
	svc := newTestTripService(repo, &mockVehicleRepository{}, log)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			trip, err := svc.AssignTrip(context.Background(), tt.driverID, tt.vehicleID, nil)
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
	svc := newTestTripService(repo, &mockVehicleRepository{}, log)

	_, err := svc.AssignTrip(context.Background(), 1, 1, nil)
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
	svc := newTestTripService(repo, &mockVehicleRepository{}, log)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.UpdateTrip(context.Background(), tt.id, model.Trip{Status: tt.status}, nil)
			if err != tt.wantErr {
				t.Errorf("got %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestTripServiceUpdateTrip_RepositoryError(t *testing.T) {
	repo := &mockTripRepository{updateErr: model.ErrTripAlreadyFinished}
	log := logger.NewStdLogger(logger.DebugLevel)
	svc := newTestTripService(repo, &mockVehicleRepository{}, log)

	_, err := svc.UpdateTrip(context.Background(), 1, model.Trip{Status: model.TripStatusRunning}, nil)
	if err != model.ErrTripAlreadyFinished {
		t.Errorf("got %v, want %v", err, model.ErrTripAlreadyFinished)
	}
}

func TestTripServiceDeleteTrip(t *testing.T) {
	repo := &mockTripRepository{}
	log := logger.NewStdLogger(logger.DebugLevel)
	svc := newTestTripService(repo, &mockVehicleRepository{}, log)

	trip, err := svc.DeleteTrip(context.Background(), 5, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if trip.Status != model.TripStatusCancelled {
		t.Errorf("got status %v, want CANCELLED", trip.Status)
	}

	_, err = svc.DeleteTrip(context.Background(), 0, nil)
	if err != model.ErrInvalidTripID {
		t.Errorf("got %v, want %v", err, model.ErrInvalidTripID)
	}

	repo.updateErr = model.ErrTripAlreadyFinished
	_, err = svc.DeleteTrip(context.Background(), 5, nil)
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
		{name: "min_avg_speed zero", filter: model.TripFilter{MinAvgSpeed: float64Ptr(0), Limit: 100}, wantErr: model.ErrInvalidSpeed},
		{name: "max_avg_speed zero", filter: model.TripFilter{MaxAvgSpeed: float64Ptr(0), Limit: 100}, wantErr: model.ErrInvalidSpeed},
		{
			name:    "min_avg_speed greater than max_avg_speed",
			filter:  model.TripFilter{MinAvgSpeed: float64Ptr(20), MaxAvgSpeed: float64Ptr(10), Limit: 100},
			wantErr: model.ErrInvalidSpeed,
		},
		{name: "min_max_speed zero", filter: model.TripFilter{MinMaxSpeed: float64Ptr(0), Limit: 100}, wantErr: model.ErrInvalidSpeed},
		{name: "max_max_speed zero", filter: model.TripFilter{MaxMaxSpeed: float64Ptr(0), Limit: 100}, wantErr: model.ErrInvalidSpeed},
		{
			name:    "min_max_speed greater than max_max_speed",
			filter:  model.TripFilter{MinMaxSpeed: float64Ptr(120), MaxMaxSpeed: float64Ptr(30), Limit: 100},
			wantErr: model.ErrInvalidSpeed,
		},
	}

	repo := &mockTripRepository{}
	log := logger.NewStdLogger(logger.DebugLevel)
	svc := newTestTripService(repo, &mockVehicleRepository{}, log)

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
	svc := newTestTripService(repo, &mockVehicleRepository{}, log)

	_, err := svc.GetListTrips(context.Background(), model.TripFilter{Limit: 100})
	if err != model.ErrNotFound {
		t.Errorf("got %v, want %v", err, model.ErrNotFound)
	}
}

func TestGetTripByID(t *testing.T) {
	repo := &mockTripRepository{}
	log := logger.NewStdLogger(logger.DebugLevel)
	svc := newTestTripService(repo, &mockVehicleRepository{}, log)

	trip, err := svc.GetTripByID(context.Background(), 1, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if trip.ID != 1 {
		t.Errorf("got trip id %d, want 1", trip.ID)
	}

	_, err = svc.GetTripByID(context.Background(), 0, nil)
	if err != model.ErrInvalidTripID {
		t.Errorf("got %v, want %v", err, model.ErrInvalidTripID)
	}

	repo.getErr = model.ErrNotFound
	_, err = svc.GetTripByID(context.Background(), 1, nil)
	if err != model.ErrNotFound {
		t.Errorf("got %v, want %v", err, model.ErrNotFound)
	}
}

// TestTripService_VehicleStatus проверяет, что статус машины ведёт рейс:
// старт рейса занимает машину, завершение и отмена освобождают, промежуточные статусы не трогают
func TestTripService_VehicleStatus(t *testing.T) {
	org := 1
	log := logger.NewStdLogger(logger.ErrorLevel)

	t.Run("старт рейса: IDLE -> ON_TRIP", func(t *testing.T) {
		vehicles := &mockVehicleRepository{}
		svc := newTestTripService(&mockTripRepository{}, vehicles, log)
		if _, err := svc.AssignTrip(context.Background(), 1, 7, &org); err != nil {
			t.Fatal(err)
		}
		if len(vehicles.transitions) != 1 || vehicles.transitions[0] != "IDLE->ON_TRIP" {
			t.Errorf("transitions = %v", vehicles.transitions)
		}
	})

	t.Run("машина занята: второй рейс не стартует", func(t *testing.T) {
		vehicles := &mockVehicleRepository{
			statusErr: model.ErrNotFound,
			vehicle:   &model.Vehicle{ID: 7, OrganizationID: org, Status: model.VehicleStatusOnTrip},
		}
		trips := &mockTripRepository{}
		svc := newTestTripService(trips, vehicles, log)
		if _, err := svc.AssignTrip(context.Background(), 1, 7, &org); err != model.ErrVehicleIsBusy {
			t.Errorf("err = %v, want ErrVehicleIsBusy", err)
		}
	})

	t.Run("машина чужой организации неотличима от несуществующей", func(t *testing.T) {
		vehicles := &mockVehicleRepository{
			statusErr: model.ErrNotFound,
			vehicle:   &model.Vehicle{ID: 7, OrganizationID: 2, Status: model.VehicleStatusIdle},
		}
		svc := newTestTripService(&mockTripRepository{}, vehicles, log)
		if _, err := svc.AssignTrip(context.Background(), 1, 7, &org); err != model.ErrNotFound {
			t.Errorf("err = %v, want ErrNotFound", err)
		}
	})

	for _, status := range []model.TripStatus{model.TripStatusSucceeded, model.TripStatusCancelled} {
		t.Run(string(status)+": ON_TRIP -> IDLE", func(t *testing.T) {
			vehicles := &mockVehicleRepository{}
			svc := newTestTripService(&mockTripRepository{}, vehicles, log)
			if _, err := svc.UpdateTrip(context.Background(), 3, model.Trip{Status: status}, &org); err != nil {
				t.Fatal(err)
			}
			if len(vehicles.transitions) != 1 || vehicles.transitions[0] != "ON_TRIP->IDLE" {
				t.Errorf("transitions = %v", vehicles.transitions)
			}
		})
	}

	t.Run("промежуточный статус машину не освобождает", func(t *testing.T) {
		vehicles := &mockVehicleRepository{}
		svc := newTestTripService(&mockTripRepository{}, vehicles, log)
		if _, err := svc.UpdateTrip(context.Background(), 3, model.Trip{Status: model.TripStatusSleeping}, &org); err != nil {
			t.Fatal(err)
		}
		if len(vehicles.transitions) != 0 {
			t.Errorf("transitions = %v, want none", vehicles.transitions)
		}
	})
}
