package service

import (
	"context"
	"errors"
	"fleettrack/internal/model"
	"testing"
)

type mockAssignmentRepository struct {
	activeAssignment model.DeviceAssignment
	getActiveErr     error
	endErr           error

	endCalled bool
}

func (m *mockAssignmentRepository) GetActiveAssignment(ctx context.Context, deviceID int) (model.DeviceAssignment, error) {
	if m.getActiveErr != nil {
		return model.DeviceAssignment{}, m.getActiveErr
	}
	return m.activeAssignment, nil
}

func (m *mockAssignmentRepository) CreateAssignment(ctx context.Context, a *model.DeviceAssignment) error {
	return nil
}

func (m *mockAssignmentRepository) EndAssignment(ctx context.Context, deviceID int) error {
	m.endCalled = true
	return m.endErr
}

func TestGetActiveAssignment(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		repo := &mockAssignmentRepository{activeAssignment: model.DeviceAssignment{ID: 7, DeviceID: 1, VehicleID: 2}}
		svc := NewAssignmentService(repo, nil, nil, nil, nil)

		got := svc.GetActiveAssignment(context.Background(), 1)
		if got.ID != 7 {
			t.Errorf("got ID %d, want 7", got.ID)
		}
	})

	t.Run("repository error returns zero value", func(t *testing.T) {
		repo := &mockAssignmentRepository{getActiveErr: model.ErrNotFound}
		svc := NewAssignmentService(repo, nil, nil, nil, nil)

		got := svc.GetActiveAssignment(context.Background(), 1)
		if got != (model.DeviceAssignment{}) {
			t.Errorf("got %+v, want zero value", got)
		}
	})
}

func TestValidateDevice(t *testing.T) {
	tests := []struct {
		name    string
		device  model.Device
		wantErr error
	}{
		{name: "active device is valid", device: model.Device{Status: model.DeviceStatusActive}, wantErr: nil},
		{name: "inactive device is busy", device: model.Device{Status: model.DeviceStatusInactive}, wantErr: model.ErrDeviceIsBusy},
		{name: "maintenance device is busy", device: model.Device{Status: model.DeviceStatusMaintenance}, wantErr: model.ErrDeviceIsBusy},
	}

	svc := NewAssignmentService(nil, nil, nil, nil, nil)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := svc.ValidateDevice(context.Background(), tt.device)
			if !errors.Is(err, tt.wantErr) && err != tt.wantErr {
				t.Errorf("got %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateVehicle(t *testing.T) {
	tests := []struct {
		name    string
		vehicle model.Vehicle
		wantErr error
	}{
		{name: "idle vehicle is valid", vehicle: model.Vehicle{Status: model.VehicleStatusIdle}, wantErr: nil},
		{name: "on trip vehicle is busy", vehicle: model.Vehicle{Status: model.VehicleStatusOnTrip}, wantErr: model.ErrVehicleIsBusy},
		{name: "in service vehicle is busy", vehicle: model.Vehicle{Status: model.VehicleStatusInService}, wantErr: model.ErrVehicleIsBusy},
	}

	svc := NewAssignmentService(nil, nil, nil, nil, nil)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := svc.ValidateVehicle(context.Background(), tt.vehicle)
			if !errors.Is(err, tt.wantErr) && err != tt.wantErr {
				t.Errorf("got %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestEndPreviousAssignment(t *testing.T) {
	svc := NewAssignmentService(nil, nil, nil, nil, nil)

	t.Run("no active assignment is not an error", func(t *testing.T) {
		repo := &mockAssignmentRepository{getActiveErr: model.ErrNotFound}
		err := svc.EndPreviousAssignment(context.Background(), 1, repo)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if repo.endCalled {
			t.Error("EndAssignment should not be called when there is no active assignment")
		}
	})

	t.Run("active assignment gets ended", func(t *testing.T) {
		repo := &mockAssignmentRepository{activeAssignment: model.DeviceAssignment{ID: 5}}
		err := svc.EndPreviousAssignment(context.Background(), 1, repo)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if !repo.endCalled {
			t.Error("expected EndAssignment to be called")
		}
	})

	t.Run("EndAssignment error propagates", func(t *testing.T) {
		endErr := errors.New("db error")
		repo := &mockAssignmentRepository{activeAssignment: model.DeviceAssignment{ID: 5}, endErr: endErr}
		err := svc.EndPreviousAssignment(context.Background(), 1, repo)
		if !errors.Is(err, endErr) {
			t.Errorf("got %v, want %v", err, endErr)
		}
	})

	t.Run("unexpected GetActiveAssignment error propagates", func(t *testing.T) {
		getErr := errors.New("connection lost")
		repo := &mockAssignmentRepository{getActiveErr: getErr}
		err := svc.EndPreviousAssignment(context.Background(), 1, repo)
		if !errors.Is(err, getErr) {
			t.Errorf("got %v, want %v", err, getErr)
		}
		if repo.endCalled {
			t.Error("EndAssignment should not be called when GetActiveAssignment fails unexpectedly")
		}
	})
}
