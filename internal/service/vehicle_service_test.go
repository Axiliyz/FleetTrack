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
	statusErr error
	vehicle   *model.Vehicle
	// transitions - переходы статуса, сделанные через UpdateStatus, в формате "FROM->TO"
	transitions []string
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
	if m.vehicle != nil {
		return *m.vehicle, nil
	}
	return model.Vehicle{ID: id}, nil
}

func (m *mockVehicleRepository) GetList(ctx context.Context, filter model.VehicleFilter) ([]model.Vehicle, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	return []model.Vehicle{}, nil
}

func (m *mockVehicleRepository) Delete(ctx context.Context, id int, organizationID *int) (model.Vehicle, error) {
	if m.deleteErr != nil {
		return model.Vehicle{}, m.deleteErr
	}
	return model.Vehicle{ID: id, Status: model.VehicleStatusDeleted}, nil
}

func (m *mockVehicleRepository) Update(ctx context.Context, id int, upd model.UpdateVehicle, organizationID *int) (model.Vehicle, error) {
	if m.updateErr != nil {
		return model.Vehicle{}, m.updateErr
	}
	return model.Vehicle{ID: id}, nil
}

func (m *mockVehicleRepository) UpdateLastTelemetryAt(ctx context.Context, id int, at time.Time) error {
	return nil
}

func (m *mockVehicleRepository) UpdateStatus(ctx context.Context, id int, from, to model.VehicleStatus, organizationID *int) (model.Vehicle, error) {
	if m.statusErr != nil {
		return model.Vehicle{}, m.statusErr
	}
	m.transitions = append(m.transitions, string(from)+"->"+string(to))
	v := model.Vehicle{ID: id, Status: to}
	if organizationID != nil {
		v.OrganizationID = *organizationID
	}
	return v, nil
}

// newTestVehicleService собирает VehicleService, у которого транзакция работает с тем же моком машин
func newTestVehicleService(repo *mockVehicleRepository, log logger.Logger) *VehicleService {
	svc, _, _ := newTestVehicleServiceWith(repo, log)
	return svc
}

// newTestVehicleServiceWith - то же, но отдаёт моки привязок и алертов, чтобы проверить, что с ними сделала транзакция
func newTestVehicleServiceWith(repo *mockVehicleRepository, log logger.Logger) (*VehicleService, *mockAssignmentRepository, *mockAlertRepo) {
	assignments := &mockAssignmentRepository{}
	alerts := &mockAlertRepo{}
	rf := &fakeRepoFactory{vehicle: repo, assignment: assignments, alert: alerts}
	return NewVehicleService(repo, log, &fakeTxManager{}, rf), assignments, alerts
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
	svc := newTestVehicleService(repo, log)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := vehicleFixture()
			tt.mutate(&v)
			_, err := svc.CreateVehicle(context.Background(), v)
			if err != tt.wantErr {
				t.Errorf("got %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestProcessVehicle_RepositoryError(t *testing.T) {
	repo := &mockVehicleRepository{createErr: model.ErrDuplicateVIN}
	log := logger.NewStdLogger(logger.DebugLevel)
	svc := newTestVehicleService(repo, log)

	_, err := svc.CreateVehicle(context.Background(), vehicleFixture())
	if err != model.ErrDuplicateVIN {
		t.Errorf("got %v, want %v", err, model.ErrDuplicateVIN)
	}
}

func strPtr(v string) *string                              { return &v }
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
	svc := newTestVehicleService(repo, log)

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
	svc := newTestVehicleService(repo, log)

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
	svc := newTestVehicleService(repo, log)

	v, err := svc.DeleteVehicleByID(context.Background(), 5, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v.Status != model.VehicleStatusDeleted {
		t.Errorf("got status %v, want %v", v.Status, model.VehicleStatusDeleted)
	}

	repo.deleteErr = model.ErrNotFound
	_, err = svc.DeleteVehicleByID(context.Background(), 999, nil)
	if err != model.ErrNotFound {
		t.Errorf("got %v, want %v", err, model.ErrNotFound)
	}
}

func TestUpdateVehicleByID(t *testing.T) {
	repo := &mockVehicleRepository{}
	log := logger.NewStdLogger(logger.DebugLevel)
	svc := newTestVehicleService(repo, log)

	upd := model.UpdateVehicle{NumberPlate: strPtr("B999XY77")}
	v, err := svc.UpdateVehicleByID(context.Background(), 5, upd, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v.ID != 5 {
		t.Errorf("got ID %d, want 5", v.ID)
	}

	repo.updateErr = model.ErrNotFound
	_, err = svc.UpdateVehicleByID(context.Background(), 999, upd, nil)
	if err != model.ErrNotFound {
		t.Errorf("got %v, want %v", err, model.ErrNotFound)
	}
}

// TestDeleteVehicleByID_ReleasesTracker проверяет, что удалённая машина не оставляет открытой
// привязку трекера: иначе OfflineWorker вечно поднимал бы DEVICE_OFFLINE по несуществующей машине
func TestDeleteVehicleByID_ReleasesTracker(t *testing.T) {
	repo := &mockVehicleRepository{}
	svc, assignments, _ := newTestVehicleServiceWith(repo, logger.NewStdLogger(logger.ErrorLevel))

	if _, err := svc.DeleteVehicleByID(context.Background(), 5, nil); err != nil {
		t.Fatal(err)
	}
	if len(assignments.endedVehicles) != 1 || assignments.endedVehicles[0] != 5 {
		t.Errorf("assignment of vehicle 5 not ended: %v", assignments.endedVehicles)
	}

	repo.deleteErr = model.ErrVehicleIsBusy
	if _, err := svc.DeleteVehicleByID(context.Background(), 6, nil); err != model.ErrVehicleIsBusy {
		t.Errorf("vehicle on trip: got %v, want ErrVehicleIsBusy", err)
	}
}

// TestUpdateVehicleByID_Status проверяет, что вручную ставятся только IDLE и IN_SERVICE:
// ON_TRIP ведёт рейс, а DELETED - только DELETE
func TestUpdateVehicleByID_Status(t *testing.T) {
	svc := newTestVehicleService(&mockVehicleRepository{}, logger.NewStdLogger(logger.ErrorLevel))
	tests := []struct {
		status  model.VehicleStatus
		wantErr error
	}{
		{model.VehicleStatusIdle, nil},
		{model.VehicleStatusInService, nil},
		{model.VehicleStatusOnTrip, model.ErrInvalidStatus},
		{model.VehicleStatusDeleted, model.ErrInvalidStatus},
		{"FLYING", model.ErrInvalidStatus},
	}
	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			status := tt.status
			_, err := svc.UpdateVehicleByID(context.Background(), 5, model.UpdateVehicle{Status: &status}, nil)
			if err != tt.wantErr {
				t.Errorf("got %v, want %v", err, tt.wantErr)
			}
		})
	}

	if _, err := svc.UpdateVehicleByID(context.Background(), 5, model.UpdateVehicle{NumberPlate: strPtr("bad plate!")}, nil); err != model.ErrInvalidNumberPlate {
		t.Errorf("invalid plate: got %v", err)
	}
}
