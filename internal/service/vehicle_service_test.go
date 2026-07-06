package service

import (
	"context"
	"fleettrack/internal/logger"
	"fleettrack/internal/model"
	"testing"
	"time"
)

type mockVehicleRepository struct {
	createErr error
	getErr    error
	listErr   error
	deleteErr error
	updateErr error
}

func (m *mockVehicleRepository) Create(ctx context.Context, v *model.Vehicle) error {
	if m.createErr != nil {
		return m.createErr
	}
	v.ID = 1
	return nil
}

func (m *mockVehicleRepository) GetByID(ctx context.Context, id int) (model.Vehicle, error) {
	if m.getErr != nil {
		return model.Vehicle{}, m.getErr
	}
	return model.Vehicle{ID: id}, nil
}

func (m *mockVehicleRepository) GetList(ctx context.Context, filter model.VehicleFilter) ([]model.Vehicle, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	return []model.Vehicle{}, nil
}

func (m *mockVehicleRepository) Delete(ctx context.Context, id int) (model.Vehicle, error) {
	if m.deleteErr != nil {
		return model.Vehicle{}, m.deleteErr
	}
	return model.Vehicle{ID: id, Status: model.VehicleStatusDeleted}, nil
}

func (m *mockVehicleRepository) Update(ctx context.Context, id int, upd model.UpdateVehicle) (model.Vehicle, error) {
	if m.updateErr != nil {
		return model.Vehicle{}, m.updateErr
	}
	return model.Vehicle{ID: id}, nil
}

func vehicleFixture() model.Vehicle {
	return model.Vehicle{
		OrganizationID: 1,
		VIN:            "1HGCM82633A123456",
		NumberPlate:    "A123BC77",
		Model:          "Toyota Camry",
		Status:         model.VehicleStatusIdle,
	}
}

func TestProcessVehicle(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(v *model.Vehicle)
		wantErr error
	}{
		{name: "valid", mutate: func(v *model.Vehicle) {}, wantErr: nil},
		{name: "invalid vin", mutate: func(v *model.Vehicle) { v.VIN = "SHORT" }, wantErr: model.ErrInvalidVIN},
		{name: "invalid organization id", mutate: func(v *model.Vehicle) { v.OrganizationID = 0 }, wantErr: model.ErrInvalidOrganizationID},
		{name: "invalid number plate", mutate: func(v *model.Vehicle) { v.NumberPlate = "X" }, wantErr: model.ErrInvalidNumberPlate},
	}

	repo := &mockVehicleRepository{}
	log := logger.NewStdLogger(logger.DebugLevel)
	svc := NewVehicleService(repo, log)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := vehicleFixture()
			tt.mutate(&v)
			_, err := svc.ProcessVehicle(context.Background(), v)
			if err != tt.wantErr {
				t.Errorf("got %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestProcessVehicle_RepositoryError(t *testing.T) {
	repo := &mockVehicleRepository{createErr: model.ErrDuplicateVIN}
	log := logger.NewStdLogger(logger.DebugLevel)
	svc := NewVehicleService(repo, log)

	_, err := svc.ProcessVehicle(context.Background(), vehicleFixture())
	if err != model.ErrDuplicateVIN {
		t.Errorf("got %v, want %v", err, model.ErrDuplicateVIN)
	}
}

func strPtr(v string) *string                               { return &v }
func statusPtr(v model.VehicleStatus) *model.VehicleStatus { return &v }

func TestGetVehicleList(t *testing.T) {
	tests := []struct {
		name    string
		filter  model.VehicleFilter
		wantErr error
	}{
		{name: "no filters", filter: model.VehicleFilter{Limit: 100}, wantErr: nil},
		{name: "invalid organization id", filter: model.VehicleFilter{OrganizationID: intPtr(0), Limit: 100}, wantErr: model.ErrInvalidOrganizationID},
		{name: "invalid vin length", filter: model.VehicleFilter{VIN: strPtr("SHORT"), Limit: 100}, wantErr: model.ErrInvalidVIN},
		{name: "number plate too short", filter: model.VehicleFilter{NumberPlate: strPtr("A1"), Limit: 100}, wantErr: model.ErrInvalidNumberPlate},
		{name: "number plate too long", filter: model.VehicleFilter{NumberPlate: strPtr("A123456789"), Limit: 100}, wantErr: model.ErrInvalidNumberPlate},
		{name: "valid number plate", filter: model.VehicleFilter{NumberPlate: strPtr("A123BC77"), Limit: 100}, wantErr: nil},
		{name: "invalid status", filter: model.VehicleFilter{Status: statusPtr("BOGUS"), Limit: 100}, wantErr: model.ErrInvalidStatus},
		{
			name: "created_to before created_from",
			filter: model.VehicleFilter{
				CreatedFrom: timePtr(time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)),
				CreatedTo:   timePtr(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)),
				Limit:       100,
			},
			wantErr: model.ErrInvalidTimestamp,
		},
	}

	repo := &mockVehicleRepository{}
	log := logger.NewStdLogger(logger.DebugLevel)
	svc := NewVehicleService(repo, log)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.GetVehicleList(context.Background(), tt.filter)
			if err != tt.wantErr {
				t.Errorf("got %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetVehicleByID(t *testing.T) {
	repo := &mockVehicleRepository{}
	log := logger.NewStdLogger(logger.DebugLevel)
	svc := NewVehicleService(repo, log)

	v, err := svc.GetVehicleByID(context.Background(), 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v.ID != 5 {
		t.Errorf("got ID %d, want 5", v.ID)
	}

	repo.getErr = model.ErrNotFound
	_, err = svc.GetVehicleByID(context.Background(), 999)
	if err != model.ErrNotFound {
		t.Errorf("got %v, want %v", err, model.ErrNotFound)
	}
}

func TestDeleteVehicleByID(t *testing.T) {
	repo := &mockVehicleRepository{}
	log := logger.NewStdLogger(logger.DebugLevel)
	svc := NewVehicleService(repo, log)

	v, err := svc.DeleteVehicleByID(context.Background(), 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v.Status != model.VehicleStatusDeleted {
		t.Errorf("got status %v, want %v", v.Status, model.VehicleStatusDeleted)
	}

	repo.deleteErr = model.ErrNotFound
	_, err = svc.DeleteVehicleByID(context.Background(), 999)
	if err != model.ErrNotFound {
		t.Errorf("got %v, want %v", err, model.ErrNotFound)
	}
}

func TestUpdateVehicleByID(t *testing.T) {
	repo := &mockVehicleRepository{}
	log := logger.NewStdLogger(logger.DebugLevel)
	svc := NewVehicleService(repo, log)

	upd := model.UpdateVehicle{NumberPlate: strPtr("B999XY77")}
	v, err := svc.UpdateVehicleByID(context.Background(), 5, upd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v.ID != 5 {
		t.Errorf("got ID %d, want 5", v.ID)
	}

	repo.updateErr = model.ErrNotFound
	_, err = svc.UpdateVehicleByID(context.Background(), 999, upd)
	if err != model.ErrNotFound {
		t.Errorf("got %v, want %v", err, model.ErrNotFound)
	}
}
