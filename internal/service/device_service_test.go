package service

import (
	"context"
	"fleettrack/internal/database"
	"fleettrack/internal/logger"
	"fleettrack/internal/model"
	"fleettrack/internal/repository"
	"fleettrack/internal/repository/factory"
	"testing"
)

type mockDeviceRepo struct {
	createErr error
	getErr    error
	deleteErr error
}

func (m *mockDeviceRepo) Create(ctx context.Context, d *model.Device) error {
	if m.createErr != nil {
		return m.createErr
	}
	d.ID = 1
	return nil
}

func (m *mockDeviceRepo) GetByID(ctx context.Context, id int) (model.Device, error) {
	if m.getErr != nil {
		return model.Device{}, m.getErr
	}
	return model.Device{ID: id}, nil
}

func (m *mockDeviceRepo) Delete(ctx context.Context, id int) (model.Device, error) {
	if m.deleteErr != nil {
		return model.Device{}, m.deleteErr
	}
	return model.Device{ID: id, Status: model.DeviceStatusInactive}, nil
}

// mockTxManager выполняет fn напрямую, без реальной транзакции
type mockTxManager struct{}

func (m *mockTxManager) WithTx(ctx context.Context, fn func(tx database.DBTX) error) error {
	return fn(nil)
}

// mockRepoFactory возвращает заранее заданные репозитории вместо реальных, привязанных к tx
type mockRepoFactory struct {
	deviceRepo     repository.DeviceRepository
	assignmentRepo repository.AssignmentRepository
}

func (m *mockRepoFactory) New(tx database.DBTX) factory.Repositories {
	return factory.Repositories{
		Device:     m.deviceRepo,
		Assignment: m.assignmentRepo,
	}
}

func deviceFixture() model.Device {
	return model.Device{
		SerialNumber: "DEV-001",
		Status:       model.DeviceStatusActive,
	}
}

func TestProcessDevice(t *testing.T) {
	tests := []struct {
		name    string
		device  model.Device
		wantErr error
	}{
		{name: "valid", device: deviceFixture(), wantErr: nil},
		{name: "empty serial number", device: model.Device{SerialNumber: ""}, wantErr: model.ErrInvalidSerialNumber},
		{name: "blank serial number", device: model.Device{SerialNumber: "   "}, wantErr: model.ErrInvalidSerialNumber},
	}

	repo := &mockDeviceRepo{}
	log := logger.NewStdLogger(logger.DebugLevel)
	svc := NewDeviceService(repo, log, nil, nil)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.ProcessDevice(context.Background(), tt.device)
			if err != tt.wantErr {
				t.Errorf("got %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestProcessDevice_RepositoryError(t *testing.T) {
	repo := &mockDeviceRepo{createErr: model.ErrDuplicateSerialNumber}
	log := logger.NewStdLogger(logger.DebugLevel)
	svc := NewDeviceService(repo, log, nil, nil)

	_, err := svc.ProcessDevice(context.Background(), deviceFixture())
	if err != model.ErrDuplicateSerialNumber {
		t.Errorf("got %v, want %v", err, model.ErrDuplicateSerialNumber)
	}
}

func TestGetDeviceByID(t *testing.T) {
	repo := &mockDeviceRepo{}
	log := logger.NewStdLogger(logger.DebugLevel)
	svc := NewDeviceService(repo, log, nil, nil)

	d, err := svc.GetDeviceByID(context.Background(), 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if d.ID != 5 {
		t.Errorf("got ID %d, want 5", d.ID)
	}

	repo.getErr = model.ErrNotFound
	_, err = svc.GetDeviceByID(context.Background(), 999)
	if err != model.ErrNotFound {
		t.Errorf("got %v, want %v", err, model.ErrNotFound)
	}
}

func TestDeleteDevice(t *testing.T) {
	t.Run("no active assignment is not an error", func(t *testing.T) {
		deviceRepo := &mockDeviceRepo{}
		assignmentRepo := &mockAssignmentRepository{getActiveErr: model.ErrNotFound}
		svc := NewDeviceService(deviceRepo, logger.NewStdLogger(logger.DebugLevel), &mockTxManager{}, &mockRepoFactory{deviceRepo: deviceRepo, assignmentRepo: assignmentRepo})

		d, err := svc.DeleteDevice(context.Background(), 5)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if d.Status != model.DeviceStatusInactive {
			t.Errorf("got status %v, want %v", d.Status, model.DeviceStatusInactive)
		}
	})

	t.Run("active assignment gets ended before delete", func(t *testing.T) {
		deviceRepo := &mockDeviceRepo{}
		assignmentRepo := &mockAssignmentRepository{activeAssignment: model.DeviceAssignment{ID: 1}}
		svc := NewDeviceService(deviceRepo, logger.NewStdLogger(logger.DebugLevel), &mockTxManager{}, &mockRepoFactory{deviceRepo: deviceRepo, assignmentRepo: assignmentRepo})

		_, err := svc.DeleteDevice(context.Background(), 5)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !assignmentRepo.endCalled {
			t.Error("expected EndAssignment to be called")
		}
	})

	t.Run("unexpected EndAssignment error propagates", func(t *testing.T) {
		deviceRepo := &mockDeviceRepo{}
		endErr := model.ErrConnectingDB
		assignmentRepo := &mockAssignmentRepository{endErr: endErr}
		svc := NewDeviceService(deviceRepo, logger.NewStdLogger(logger.DebugLevel), &mockTxManager{}, &mockRepoFactory{deviceRepo: deviceRepo, assignmentRepo: assignmentRepo})

		_, err := svc.DeleteDevice(context.Background(), 5)
		if err != endErr {
			t.Errorf("got %v, want %v", err, endErr)
		}
	})

	t.Run("Delete error propagates", func(t *testing.T) {
		deviceRepo := &mockDeviceRepo{deleteErr: model.ErrNotFound}
		assignmentRepo := &mockAssignmentRepository{getActiveErr: model.ErrNotFound}
		svc := NewDeviceService(deviceRepo, logger.NewStdLogger(logger.DebugLevel), &mockTxManager{}, &mockRepoFactory{deviceRepo: deviceRepo, assignmentRepo: assignmentRepo})

		_, err := svc.DeleteDevice(context.Background(), 999)
		if err != model.ErrNotFound {
			t.Errorf("got %v, want %v", err, model.ErrNotFound)
		}
	})
}
