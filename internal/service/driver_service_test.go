package service

import (
	"context"
	"fleettrack/internal/logger"
	"fleettrack/internal/model"
	"testing"
	"time"
)

type mockDriverRepository struct {
	createErr error
	getErr    error
	listErr   error
	deleteErr error
	updateErr error
}

func (m *mockDriverRepository) Create(ctx context.Context, d *model.Driver) error {
	if m.createErr != nil {
		return m.createErr
	}
	d.ID = 1
	return nil
}

func (m *mockDriverRepository) GetByID(ctx context.Context, id int) (model.Driver, error) {
	if m.getErr != nil {
		return model.Driver{}, m.getErr
	}
	return model.Driver{ID: id}, nil
}

func (m *mockDriverRepository) GetList(ctx context.Context, filter model.DriverFilter) ([]model.Driver, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	return []model.Driver{}, nil
}

func (m *mockDriverRepository) Delete(ctx context.Context, id int) (model.Driver, error) {
	if m.deleteErr != nil {
		return model.Driver{}, m.deleteErr
	}
	return model.Driver{ID: id}, nil
}

func (m *mockDriverRepository) Update(ctx context.Context, id int, upd model.UpdateDriver) (model.Driver, error) {
	if m.updateErr != nil {
		return model.Driver{}, m.updateErr
	}
	return model.Driver{ID: id}, nil
}

func driverFixture() model.Driver {
	return model.Driver{
		OrganizationID: 1,
		Name:           "Ivan Petrov",
	}
}

func TestCreateDriver(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(d *model.Driver)
		wantErr error
	}{
		{name: "valid", mutate: func(d *model.Driver) {}, wantErr: nil},
		{name: "invalid organization id", mutate: func(d *model.Driver) { d.OrganizationID = 0 }, wantErr: model.ErrInvalidOrganizationID},
		{name: "empty name", mutate: func(d *model.Driver) { d.Name = "" }, wantErr: model.ErrInvalidDriverName},
	}

	repo := &mockDriverRepository{}
	log := logger.NewStdLogger(logger.DebugLevel)
	svc := NewDriverService(repo, log)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := driverFixture()
			tt.mutate(&d)
			_, err := svc.CreateDriver(context.Background(), d)
			if err != tt.wantErr {
				t.Errorf("got %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestCreateDriver_RepositoryError(t *testing.T) {
	repo := &mockDriverRepository{createErr: model.ErrNotFound}
	log := logger.NewStdLogger(logger.DebugLevel)
	svc := NewDriverService(repo, log)

	_, err := svc.CreateDriver(context.Background(), driverFixture())
	if err != model.ErrNotFound {
		t.Errorf("got %v, want %v", err, model.ErrNotFound)
	}
}

func TestGetDriverByID(t *testing.T) {
	repo := &mockDriverRepository{}
	log := logger.NewStdLogger(logger.DebugLevel)
	svc := NewDriverService(repo, log)

	d, err := svc.GetDriverByID(context.Background(), 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if d.ID != 5 {
		t.Errorf("got ID %d, want 5", d.ID)
	}

	repo.getErr = model.ErrNotFound
	_, err = svc.GetDriverByID(context.Background(), 999)
	if err != model.ErrNotFound {
		t.Errorf("got %v, want %v", err, model.ErrNotFound)
	}
}

func TestGetDriverList(t *testing.T) {
	tests := []struct {
		name    string
		filter  model.DriverFilter
		wantErr error
	}{
		{name: "no filters", filter: model.DriverFilter{Limit: 100}, wantErr: nil},
		{name: "invalid organization id", filter: model.DriverFilter{OrganizationID: intPtr(0), Limit: 100}, wantErr: model.ErrInvalidOrganizationID},
		{
			name: "created_to before created_from",
			filter: model.DriverFilter{
				CreatedFrom: timePtr(time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)),
				CreatedTo:   timePtr(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)),
				Limit:       100,
			},
			wantErr: model.ErrInvalidTimestamp,
		},
	}

	repo := &mockDriverRepository{}
	log := logger.NewStdLogger(logger.DebugLevel)
	svc := NewDriverService(repo, log)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.GetDriverList(context.Background(), tt.filter)
			if err != tt.wantErr {
				t.Errorf("got %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestDeleteDriverByID(t *testing.T) {
	repo := &mockDriverRepository{}
	log := logger.NewStdLogger(logger.DebugLevel)
	svc := NewDriverService(repo, log)

	d, err := svc.DeleteDriverByID(context.Background(), 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if d.ID != 5 {
		t.Errorf("got ID %d, want 5", d.ID)
	}

	repo.deleteErr = model.ErrDriverHasActiveTrips
	_, err = svc.DeleteDriverByID(context.Background(), 5)
	if err != model.ErrDriverHasActiveTrips {
		t.Errorf("got %v, want %v", err, model.ErrDriverHasActiveTrips)
	}
}

func TestUpdateDriverByID(t *testing.T) {
	tests := []struct {
		name    string
		upd     model.UpdateDriver
		wantErr error
	}{
		{name: "valid name", upd: model.UpdateDriver{Name: strPtr("New Name")}, wantErr: nil},
		{name: "empty name", upd: model.UpdateDriver{Name: strPtr("")}, wantErr: model.ErrInvalidDriverName},
		{name: "blank name", upd: model.UpdateDriver{Name: strPtr("   ")}, wantErr: model.ErrInvalidDriverName},
		{name: "invalid organization id", upd: model.UpdateDriver{OrganizationID: intPtr(0)}, wantErr: model.ErrInvalidOrganizationID},
		{name: "no fields", upd: model.UpdateDriver{}, wantErr: nil},
	}

	repo := &mockDriverRepository{}
	log := logger.NewStdLogger(logger.DebugLevel)
	svc := NewDriverService(repo, log)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.UpdateDriverByID(context.Background(), 5, tt.upd)
			if err != tt.wantErr {
				t.Errorf("got %v, want %v", err, tt.wantErr)
			}
		})
	}

	repo.updateErr = model.ErrNotFound
	_, err := svc.UpdateDriverByID(context.Background(), 999, model.UpdateDriver{})
	if err != model.ErrNotFound {
		t.Errorf("got %v, want %v", err, model.ErrNotFound)
	}
}
