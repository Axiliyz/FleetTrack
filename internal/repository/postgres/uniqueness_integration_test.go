//go:build integration

package postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	"fleettrack/internal/model"
)

// seedTwoOrgs очищает справочники и заводит две организации
func seedTwoOrgs(ctx context.Context, t *testing.T) (org1, org2 int) {
	t.Helper()
	if _, err := TestPool.Exec(ctx,
		`TRUNCATE organizations, users, vehicles, devices, device_assignments, drivers, trips,
		 telemetry, alert_rules, alerts, refresh_tokens CASCADE`,
	); err != nil {
		t.Fatalf("truncate: %v", err)
	}
	mustScan(ctx, t, &org1, `INSERT INTO organizations (name) VALUES ('Ромашка') RETURNING id`, nil)
	// то же название - другая организация: название больше не уникально (миграция 000021)
	mustScan(ctx, t, &org2, `INSERT INTO organizations (name) VALUES ('Ромашка') RETURNING id`, nil)
	return org1, org2
}

// TestVehicleUniqueness_PerOrganization проверяет, что VIN и номер уникальны только внутри организации:
// чужой клиент с той же машиной не узнаёт о ней из ответа «уже есть» и не блокирует её добавление
func TestVehicleUniqueness_PerOrganization(t *testing.T) {
	ctx := context.Background()
	org1, org2 := seedTwoOrgs(ctx, t)
	repo := NewPostgresVehicleRepository(TestPool)

	newVehicle := func(org int) *model.Vehicle {
		return &model.Vehicle{OrganizationID: org, VIN: "XTA21099012345678", NumberPlate: "A123BC77", Model: "Lada", Status: model.VehicleStatusIdle}
	}

	first := newVehicle(org1)
	if err := repo.Create(ctx, first); err != nil {
		t.Fatalf("first: %v", err)
	}
	if err := repo.Create(ctx, newVehicle(org2)); err != nil {
		t.Fatalf("same VIN in another organization: %v", err)
	}
	if err := repo.Create(ctx, newVehicle(org1)); !errors.Is(err, model.ErrDuplicateVIN) {
		t.Fatalf("same VIN in the same organization: err = %v, want ErrDuplicateVIN", err)
	}

	// удалённая машина VIN и номер не держит: ту же машину можно завести заново
	if _, err := repo.Delete(ctx, first.ID, &org1); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if err := repo.Create(ctx, newVehicle(org1)); err != nil {
		t.Fatalf("re-create after soft delete: %v", err)
	}
}

// TestDeviceUniqueness_PerOrganization проверяет, что серийник своего трекера нельзя «занять» из чужой организации
func TestDeviceUniqueness_PerOrganization(t *testing.T) {
	ctx := context.Background()
	org1, org2 := seedTwoOrgs(ctx, t)
	repo := NewPostgresDeviceRepository(TestPool)

	if err := repo.Create(ctx, &model.Device{OrganizationID: org2, SerialNumber: "TRK-001", Status: model.DeviceStatusActive}); err != nil {
		t.Fatalf("competitor: %v", err)
	}
	if err := repo.Create(ctx, &model.Device{OrganizationID: org1, SerialNumber: "TRK-001", Status: model.DeviceStatusActive}); err != nil {
		t.Fatalf("owner can't add own tracker: %v", err)
	}
	err := repo.Create(ctx, &model.Device{OrganizationID: org1, SerialNumber: "TRK-001", Status: model.DeviceStatusActive})
	if !errors.Is(err, model.ErrDuplicateSerialNumber) {
		t.Fatalf("duplicate in own organization: err = %v, want ErrDuplicateSerialNumber", err)
	}
}

// TestTelemetryCoordinatesPrecision проверяет, что координаты хранятся с 6 знаками (~11 см), а не с 4 (~11 м):
// округлённая предыдущая точка давала стоящей машине фантомную скорость
func TestTelemetryCoordinatesPrecision(t *testing.T) {
	ctx := context.Background()
	org, _ := seedTwoOrgs(ctx, t)
	var vehicleID, deviceID int
	mustScan(ctx, t, &vehicleID, `INSERT INTO vehicles (organization_id, model) VALUES ($1, 'M') RETURNING id`, []any{org})
	mustScan(ctx, t, &deviceID, `INSERT INTO devices (organization_id, serial_number) VALUES ($1, 'S') RETURNING id`, []any{org})

	repo := NewPostgresTelemetryRepository(TestPool)
	p := model.Telemetry{OrganizationID: org, VehicleID: vehicleID, DeviceID: deviceID, Lat: 55.751234, Lon: 37.618765}
	p.ReceivedAt = time.Now()
	p.DeviceTimestamp = p.ReceivedAt
	if err := repo.Save(ctx, &p); err != nil {
		t.Fatal(err)
	}
	last, err := repo.GetLastByVehicle(ctx, vehicleID)
	if err != nil {
		t.Fatal(err)
	}
	if last.Lat != 55.751234 || last.Lon != 37.618765 {
		t.Errorf("stored %.6f, %.6f - precision lost", last.Lat, last.Lon)
	}
}
