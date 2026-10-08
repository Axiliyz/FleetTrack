package service

import (
	"context"
	"errors"
	"fleettrack/internal/database"
	"fleettrack/internal/logger"
	"fleettrack/internal/model"
	"fleettrack/internal/repository"
	"fleettrack/internal/repository/factory"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type mockRepository struct {
	lastTelemetry model.Telemetry
	lastErr       error
	saved         *model.Telemetry
}

func (m *mockRepository) Save(ctx context.Context, t *model.Telemetry) error {
	m.saved = t
	return nil
}

func (m *mockRepository) GetList(ctx context.Context, filter model.TelemetryFilter) ([]model.Telemetry, error) {
	return []model.Telemetry{}, nil
}

func (m *mockRepository) GetItemByID(ctx context.Context, id int) (model.Telemetry, error) {
	return model.Telemetry{}, nil
}

func (m *mockRepository) GetListByVehicle(ctx context.Context, id int, organizationID *int) ([]model.Telemetry, error) {
	return []model.Telemetry{}, nil
}

func (m *mockRepository) DeleteItemByID(ctx context.Context, id int, organizationID *int) (model.Telemetry, error) {
	return model.Telemetry{}, nil
}

func (m *mockRepository) DeleteListByVehicle(ctx context.Context, id int, organizationID *int) ([]model.Telemetry, error) {
	return []model.Telemetry{}, nil
}

func (m *mockRepository) GetLastByVehicle(ctx context.Context, id int) (model.Telemetry, error) {
	return m.lastTelemetry, m.lastErr
}

// fakeTxManager - подмена transaction.TransactionManager: выполняет fn без реальной БД
type fakeTxManager struct {
	err error
}

func (m *fakeTxManager) WithTx(ctx context.Context, fn func(tx database.DBTX) error) error {
	if m.err != nil {
		return m.err
	}
	return fn(fakeDBTX{})
}

// fakeDBTX - подмена транзакции: принимает служебные SAVEPOINT-команды батчера
type fakeDBTX struct{}

func (fakeDBTX) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}

func (fakeDBTX) Query(context.Context, string, ...any) (pgx.Rows, error) {
	return nil, errors.New("fakeDBTX: Query not supported")
}

func (fakeDBTX) QueryRow(context.Context, string, ...any) pgx.Row {
	return nil
}

// fakeAssignmentRepo по умолчанию считает устройство назначенным той же машине (device_id == vehicle_id)
type fakeAssignmentRepo struct {
	vehicleForDevice map[int]int
}

func (f *fakeAssignmentRepo) GetActiveAssignment(ctx context.Context, deviceID int) (model.DeviceAssignment, error) {
	vehicleID, ok := f.vehicleForDevice[deviceID]
	if !ok {
		vehicleID = deviceID
	}
	return model.DeviceAssignment{DeviceID: deviceID, VehicleID: vehicleID}, nil
}

func (f *fakeAssignmentRepo) CreateAssignment(ctx context.Context, a *model.DeviceAssignment) error {
	return nil
}

func (f *fakeAssignmentRepo) EndAssignment(ctx context.Context, deviceID int) error {
	return nil
}

// fakeRepoFactory - подмена factory.RepositoryFactory: отдаёт заранее заданные моки репозиториев
type fakeRepoFactory struct {
	telemetry  repository.TelemetryRepository
	trip       repository.TripRepository
	vehicle    repository.VehicleRepository
	assignment repository.AssignmentRepository
}

func (f *fakeRepoFactory) New(tx database.DBTX) factory.Repositories {
	assignment := f.assignment
	if assignment == nil {
		assignment = &fakeAssignmentRepo{}
	}
	return factory.Repositories{
		Telemetry:  f.telemetry,
		Trip:       f.trip,
		Vehicle:    f.vehicle,
		Assignment: assignment,
	}
}

// fakeMotionService - подмена MotionService с заранее заданным результатом
type fakeMotionService struct {
	data *model.MotionData
	err  error
}

func (m *fakeMotionService) Calculate(last *model.Telemetry, cur model.Telemetry) (*model.MotionData, error) {
	return m.data, m.err
}

func TestProcessTelemetry(t *testing.T) {
	tests := []struct {
		name      string
		telemetry model.Telemetry
		wantErr   error
	}{
		{
			name: "valid",
			telemetry: model.Telemetry{
				DeviceID:  1,
				VehicleID: 1,
				Lat:       55.75,
				Lon:       37.61,
				Fuel:      float32Ptr(82.0),
			},
			wantErr: nil,
		},
		{
			name: "invalid device id",
			telemetry: model.Telemetry{
				DeviceID:  -1,
				VehicleID: 1,
				Lat:       55.75,
				Lon:       37.61,
				Fuel:      float32Ptr(0.8),
			},
			wantErr: model.ErrInvalidDeviceID,
		},
		{
			name: "invalid vehicle id",
			telemetry: model.Telemetry{
				DeviceID:  1,
				VehicleID: -15,
				Lat:       55.75,
				Lon:       37.61,
				Fuel:      float32Ptr(0.8),
			},
			wantErr: model.ErrInvalidVehicleID,
		},
		{
			name: "edge coords(lon=-180)",
			telemetry: model.Telemetry{
				DeviceID:  1,
				VehicleID: 1,
				Lat:       14.22,
				Lon:       -180,
				Fuel:      float32Ptr(0.8),
			},
			wantErr: nil,
		},
		{
			name: "edge coords (lat=-90)",
			telemetry: model.Telemetry{
				DeviceID:  1,
				VehicleID: 1,
				Lat:       -90,
				Lon:       37.61,
				Fuel:      float32Ptr(0.8),
			},
			wantErr: nil,
		},
		{
			name: "edge coords(lon=180)",
			telemetry: model.Telemetry{
				DeviceID:  1,
				VehicleID: 1,
				Lat:       14.22,
				Lon:       180,
				Fuel:      float32Ptr(0.8),
			},
			wantErr: nil,
		},
		{
			name: "edge coords (lat=90)",
			telemetry: model.Telemetry{
				DeviceID:  1,
				VehicleID: 1,
				Lat:       90,
				Lon:       37.61,
				Fuel:      float32Ptr(0.8),
			},
			wantErr: nil,
		},
		{
			name: "invalid coords (lat)",
			telemetry: model.Telemetry{
				DeviceID:  1,
				VehicleID: 1,
				Lat:       255.75,
				Lon:       37.61,
				Fuel:      float32Ptr(0.8),
			},
			wantErr: model.ErrInvalidCoords,
		},
		{
			name: "invalid coords(lon)",
			telemetry: model.Telemetry{
				DeviceID:  1,
				VehicleID: 1,
				Lat:       75.75,
				Lon:       317.61,
				Fuel:      float32Ptr(0.8),
			},
			wantErr: model.ErrInvalidCoords,
		},
		{
			name: "invalid fuel (> 100)",
			telemetry: model.Telemetry{
				DeviceID:  1,
				VehicleID: 1,
				Lat:       45.75,
				Lon:       17.61,
				Fuel:      float32Ptr(102.4),
			},
			wantErr: model.ErrInvalidFuel,
		},
		{
			name: "invalid fuel (< 0)",
			telemetry: model.Telemetry{
				DeviceID:  1,
				VehicleID: 1,
				Lat:       45.75,
				Lon:       17.61,
				Fuel:      float32Ptr(-0.14),
			},
			wantErr: model.ErrInvalidFuel,
		},
		{
			name: "edge fuel (= 0)",
			telemetry: model.Telemetry{
				DeviceID:  1,
				VehicleID: 1,
				Lat:       45.75,
				Lon:       17.61,
				Fuel:      float32Ptr(0),
			},
			wantErr: nil,
		},
		{
			name: "edge fuel (= 1)",
			telemetry: model.Telemetry{
				DeviceID:  1,
				VehicleID: 1,
				Lat:       45.75,
				Lon:       17.61,
				Fuel:      float32Ptr(1),
			},
			wantErr: nil,
		},
		{
			name: "edge fuel (= 100)",
			telemetry: model.Telemetry{
				DeviceID:  1,
				VehicleID: 1,
				Lat:       45.75,
				Lon:       17.61,
				Fuel:      float32Ptr(100),
			},
			wantErr: nil,
		},
	}

	repo := &mockRepository{}
	tripRepo := &mockTripRepository{trips: []model.Trip{{ID: 1, Status: model.TripStatusRunning}}}
	txManager := &fakeTxManager{}
	repoFactory := &fakeRepoFactory{telemetry: repo, trip: tripRepo, vehicle: &mockVehicleRepository{}}
	motion := &fakeMotionService{data: &model.MotionData{DistanceKm: 1.2, SpeedKmh: 40}}
	log := logger.NewStdLogger(logger.DebugLevel)
	service := NewTelemetryService(repo, log, txManager, repoFactory, motion, nil, testBatchConfig)
	batchCtx, batchCancel := context.WithCancel(context.Background())
	t.Cleanup(batchCancel)
	service.StartBatcher(batchCtx)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := service.ProcessTelemetry(context.Background(), tt.telemetry)
			if err != tt.wantErr {
				t.Errorf("got %v, want %v", err, tt.wantErr)
			}
		})
	}
}

// TestProcessTelemetry_NoActiveTrip проверяет, что при отсутствии активного рейса RUNNING
// телеметрия всё равно успешно сохраняется (TripID = 0)
func TestProcessTelemetry_NoActiveTrip(t *testing.T) {
	repo := &mockRepository{}
	tripRepo := &mockTripRepository{} // trips не задан - активного рейса нет
	txManager := &fakeTxManager{}
	repoFactory := &fakeRepoFactory{telemetry: repo, trip: tripRepo, vehicle: &mockVehicleRepository{}}
	log := logger.NewStdLogger(logger.DebugLevel)
	service := NewTelemetryService(repo, log, txManager, repoFactory, &fakeMotionService{}, nil, testBatchConfig)
	batchCtx, batchCancel := context.WithCancel(context.Background())
	t.Cleanup(batchCancel)
	service.StartBatcher(batchCtx)

	valid := model.Telemetry{DeviceID: 1, VehicleID: 1, Lat: 55.75, Lon: 37.61, Fuel: float32Ptr(0.8)}
	res, err := service.ProcessTelemetry(context.Background(), valid)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.TripID != 0 {
		t.Errorf("got TripID %d, want 0", res.TripID)
	}
	if repo.saved == nil {
		t.Errorf("expected telemetry to be saved in repository")
	}
}

// TestProcessTelemetry_FirstPointForVehicle проверяет, что отсутствие предыдущей точки
// (первая телеметрия машины) - не ошибка: запись сохраняется с нулевыми DistanceKm/SpeedKmh,
// а MotionService.Calculate не вызывается
func TestProcessTelemetry_FirstPointForVehicle(t *testing.T) {
	repo := &mockRepository{lastErr: model.ErrNotFound}
	tripRepo := &mockTripRepository{trips: []model.Trip{{ID: 1, Status: model.TripStatusRunning}}}
	txManager := &fakeTxManager{}
	repoFactory := &fakeRepoFactory{telemetry: repo, trip: tripRepo, vehicle: &mockVehicleRepository{}}
	log := logger.NewStdLogger(logger.DebugLevel)
	service := NewTelemetryService(repo, log, txManager, repoFactory, &fakeMotionService{}, nil, testBatchConfig)
	batchCtx, batchCancel := context.WithCancel(context.Background())
	t.Cleanup(batchCancel)
	service.StartBatcher(batchCtx)

	valid := model.Telemetry{DeviceID: 1, VehicleID: 1, Lat: 55.75, Lon: 37.61, Fuel: float32Ptr(0.8)}
	got, err := service.ProcessTelemetry(context.Background(), valid)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.DistanceKm != 0 || got.SpeedKmh != 0 {
		t.Errorf("expected zero motion for first point, got distance=%f speed=%f", got.DistanceKm, got.SpeedKmh)
	}
}

func intPtr(v int) *int              { return &v }
func float32Ptr(v float32) *float32  { return &v }
func float64Ptr(v float64) *float64  { return &v }
func timePtr(v time.Time) *time.Time { return &v }

func TestGetTelemetryList(t *testing.T) {
	from := timePtr(time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC))
	to := timePtr(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))

	tests := []struct {
		name    string
		filter  model.TelemetryFilter
		wantErr error
	}{
		{
			name:    "no filters",
			filter:  model.TelemetryFilter{Limit: 100},
			wantErr: nil,
		},
		{
			name:    "valid vehicle filter",
			filter:  model.TelemetryFilter{VehicleID: intPtr(1), Limit: 100},
			wantErr: nil,
		},
		{
			name:    "from after to",
			filter:  model.TelemetryFilter{From: from, To: to, Limit: 100},
			wantErr: model.ErrInvalidTimestamp,
		},
		{
			name:    "fuel min greater than fuel max",
			filter:  model.TelemetryFilter{FuelMin: float32Ptr(0.9), FuelMax: float32Ptr(0.1), Limit: 100},
			wantErr: model.ErrInvalidFuel,
		},
		{
			name:    "only fuel min set is valid",
			filter:  model.TelemetryFilter{FuelMin: float32Ptr(0.1), Limit: 100},
			wantErr: nil,
		},
		{
			name:    "lat min greater than lat max",
			filter:  model.TelemetryFilter{LatMin: float64Ptr(50), LatMax: float64Ptr(10), Limit: 100},
			wantErr: model.ErrInvalidCoords,
		},
		{
			name:    "only lat min set is valid",
			filter:  model.TelemetryFilter{LatMin: float64Ptr(10), Limit: 100},
			wantErr: nil,
		},
		{
			name:    "lon min greater than lon max",
			filter:  model.TelemetryFilter{LonMin: float64Ptr(50), LonMax: float64Ptr(10), Limit: 100},
			wantErr: model.ErrInvalidCoords,
		},
	}

	repo := &mockRepository{}
	logger := logger.NewStdLogger(logger.DebugLevel)
	service := NewTelemetryService(repo, logger, nil, nil, nil, nil, testBatchConfig)
	batchCtx, batchCancel := context.WithCancel(context.Background())
	t.Cleanup(batchCancel)
	service.StartBatcher(batchCtx)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := service.GetTelemetryList(context.Background(), tt.filter)
			if err != tt.wantErr {
				t.Errorf("got %v, want %v", err, tt.wantErr)
			}
		})
	}
}

var testBatchConfig = TelemetryBatchConfig{BufferSize: 100, MaxSize: 10, MaxWait: 5 * time.Millisecond}
