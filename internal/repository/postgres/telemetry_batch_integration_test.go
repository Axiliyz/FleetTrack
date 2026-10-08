//go:build integration

package postgres_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"fleettrack/internal/logger"
	"fleettrack/internal/model"
	"fleettrack/internal/repository/factory"
	"fleettrack/internal/repository/postgres"
	"fleettrack/internal/service"
	"fleettrack/internal/transaction"
)

// TestTelemetryBatchIsolatesFailedItem проверяет, что запись, упавшая внутри пачки,
// откатывается до своего SAVEPOINT и не мешает остальным записям этой же пачки.
func TestTelemetryBatchIsolatesFailedItem(t *testing.T) {
	ctx := context.Background()
	if _, err := postgres.TestPool.Exec(ctx,
		`TRUNCATE telemetry, device_assignments, devices, trips, vehicles, organizations CASCADE`,
	); err != nil {
		t.Fatalf("truncate: %v", err)
	}

	var orgID, foreignOrgID, vehicleID, foreignVehicleID, deviceID int
	mustScan := func(dst *int, query string, args ...any) {
		t.Helper()
		if err := postgres.TestPool.QueryRow(ctx, query, args...).Scan(dst); err != nil {
			t.Fatalf("%s: %v", query, err)
		}
	}
	mustScan(&orgID, `INSERT INTO organizations (name) VALUES ('Own Org') RETURNING id`)
	mustScan(&foreignOrgID, `INSERT INTO organizations (name) VALUES ('Foreign Org') RETURNING id`)
	mustScan(&vehicleID, `INSERT INTO vehicles (organization_id, model, vin, number_plate) VALUES ($1, 'Own', 'BATCHVIN000000001', 'AA0001AA') RETURNING id`, orgID)
	mustScan(&foreignVehicleID, `INSERT INTO vehicles (organization_id, model, vin, number_plate) VALUES ($1, 'Foreign', 'BATCHVIN000000002', 'AA0002AA') RETURNING id`, foreignOrgID)
	mustScan(&deviceID, `INSERT INTO devices (organization_id, serial_number) VALUES ($1, 'BATCH-1') RETURNING id`, orgID)
	if _, err := postgres.TestPool.Exec(ctx,
		`INSERT INTO device_assignments (vehicle_id, device_id) VALUES ($1, $2)`, vehicleID, deviceID,
	); err != nil {
		t.Fatalf("insert assignment: %v", err)
	}

	svc := service.NewTelemetryService(
		postgres.NewPostgresTelemetryRepository(postgres.TestPool),
		logger.NewStdLogger(logger.ErrorLevel),
		transaction.NewPostgresTransactionManager(postgres.TestPool),
		factory.NewPostgresRepositoryFactory(),
		service.NewMotionServiceImpl(),
		nil,
		service.TelemetryBatchConfig{BufferSize: 10, MaxSize: 2, MaxWait: time.Second},
	)
	batchCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	svc.StartBatcher(batchCtx)

	// MaxSize=2 и долгий MaxWait гарантируют, что обе записи уйдут одной пачкой
	inputs := []model.Telemetry{
		{OrganizationID: orgID, DeviceID: deviceID, VehicleID: vehicleID, Lat: 55.7, Lon: 37.6},
		{OrganizationID: orgID, DeviceID: deviceID, VehicleID: foreignVehicleID, Lat: 55.7, Lon: 37.6},
	}
	errs := make([]error, len(inputs))
	var wg sync.WaitGroup
	for i, in := range inputs {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, errs[i] = svc.ProcessTelemetry(ctx, in)
		}()
	}
	wg.Wait()

	if errs[0] != nil {
		t.Fatalf("valid item failed: %v", errs[0])
	}
	if !errors.Is(errs[1], model.ErrNotFound) {
		t.Fatalf("foreign vehicle item: got %v, want ErrNotFound", errs[1])
	}

	var own, foreign int
	mustScan(&own, `SELECT count(*) FROM telemetry WHERE vehicle_id = $1`, vehicleID)
	mustScan(&foreign, `SELECT count(*) FROM telemetry WHERE vehicle_id = $1`, foreignVehicleID)
	if own != 1 || foreign != 0 {
		t.Fatalf("stored rows: own=%d foreign=%d, want 1 and 0", own, foreign)
	}
}
