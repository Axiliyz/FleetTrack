package service

import (
	"context"
	"errors"
	"fleettrack/internal/model"
	"testing"
	"time"
)

type fakeSnapshotSources struct {
	orgs      []model.Org
	vehicles  []model.Vehicle
	vehErr    error
	positions []model.Telemetry

	vehicleFilter model.VehicleFilter
	driverFilter  model.DriverFilter
	ruleFilter    model.AlertRuleFilter
	tripsSince    time.Time
	alertsSince   time.Time
	orgAsked      []int
}

// у четырёх источников одинаковое имя метода GetList, поэтому каждый — своя обёртка над общим состоянием,
// а методы SnapshotRepository реализует сам fakeSnapshotSources

type vehiclesFake struct{ f *fakeSnapshotSources }

func (v vehiclesFake) GetList(_ context.Context, filter model.VehicleFilter) ([]model.Vehicle, error) {
	v.f.vehicleFilter = filter
	return v.f.vehicles, v.f.vehErr
}

type driversFake struct{ f *fakeSnapshotSources }

func (d driversFake) GetList(_ context.Context, filter model.DriverFilter) ([]model.Driver, error) {
	d.f.driverFilter = filter
	return []model.Driver{{ID: 1, OrganizationID: *filter.OrganizationID, Name: "Ильдар Сафин"}}, nil
}

type rulesFake struct{ f *fakeSnapshotSources }

func (r rulesFake) GetList(_ context.Context, filter model.AlertRuleFilter) ([]model.AlertRule, error) {
	r.f.ruleFilter = filter
	return nil, nil
}

type orgsFake struct{ f *fakeSnapshotSources }

func (o orgsFake) GetList(_ context.Context, organizationID int) ([]model.Org, error) {
	o.f.orgAsked = append(o.f.orgAsked, organizationID)
	return o.f.orgs, nil
}

func (f *fakeSnapshotSources) LatestPositions(_ context.Context, organizationID, limit int) ([]model.Telemetry, error) {
	f.orgAsked = append(f.orgAsked, organizationID)
	return f.positions, nil
}

func (f *fakeSnapshotSources) ShiftTrips(_ context.Context, organizationID int, since time.Time, limit int) ([]model.Trip, error) {
	f.orgAsked = append(f.orgAsked, organizationID)
	f.tripsSince = since
	return nil, nil
}

func (f *fakeSnapshotSources) ShiftAlerts(_ context.Context, organizationID int, since time.Time, limit int) ([]model.Alert, error) {
	f.orgAsked = append(f.orgAsked, organizationID)
	f.alertsSince = since
	return nil, nil
}

func newFleetServiceWithFakes(f *fakeSnapshotSources, now time.Time) *FleetService {
	s := NewFleetService(vehiclesFake{f}, driversFake{f}, rulesFake{f}, orgsFake{f}, f)
	s.now = func() time.Time { return now }
	return s
}

// TestFleetService_Snapshot проверяет, что снимок собирается строго по организации actor и с ограничениями
func TestFleetService_Snapshot(t *testing.T) {
	now := time.Date(2026, 10, 10, 12, 0, 0, 0, time.UTC)
	f := &fakeSnapshotSources{
		orgs:      []model.Org{{ID: 5, Name: "Северный тракт"}},
		vehicles:  []model.Vehicle{{ID: 3, OrganizationID: 5}},
		positions: []model.Telemetry{{VehicleID: 3, Lat: 55.75, Lon: 37.61}},
	}
	svc := newFleetServiceWithFakes(f, now)

	snap, err := svc.Snapshot(context.Background(), model.AuthContext{UserID: 1, OrganizationID: 5, Role: model.UserRoleDispatcher})
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}

	if snap.Org.ID != 5 || len(snap.Vehicles) != 1 || len(snap.Positions) != 1 || len(snap.Drivers) != 1 {
		t.Fatalf("unexpected snapshot: %+v", snap)
	}
	for _, org := range f.orgAsked {
		if org != 5 {
			t.Fatalf("source asked for organization %d, want 5", org)
		}
	}
	if *f.vehicleFilter.OrganizationID != 5 || f.vehicleFilter.Limit != snapshotMaxRows {
		t.Errorf("vehicle filter = %+v", f.vehicleFilter)
	}
	if *f.driverFilter.OrganizationID != 5 || f.driverFilter.Limit != snapshotMaxRows {
		t.Errorf("driver filter = %+v", f.driverFilter)
	}
	if f.ruleFilter.Limit == nil || *f.ruleFilter.Limit != snapshotMaxRows {
		t.Errorf("rule filter limit = %v", f.ruleFilter.Limit)
	}
	wantSince := now.Add(-snapshotShiftWindow)
	if !f.tripsSince.Equal(wantSince) || !f.alertsSince.Equal(wantSince) {
		t.Errorf("since: trips=%v alerts=%v, want %v", f.tripsSince, f.alertsSince, wantSince)
	}
}

// TestFleetService_SnapshotErrors проверяет права и обработку ошибок источников
func TestFleetService_SnapshotErrors(t *testing.T) {
	dispatcher := model.AuthContext{UserID: 1, OrganizationID: 5, Role: model.UserRoleDispatcher}
	errDB := errors.New("connection reset")

	tests := []struct {
		name    string
		actor   model.AuthContext
		sources *fakeSnapshotSources
		wantErr error
	}{
		{
			name:    "водителю обзор парка не положен",
			actor:   model.AuthContext{UserID: 2, OrganizationID: 5, Role: model.UserRoleDriver},
			sources: &fakeSnapshotSources{orgs: []model.Org{{ID: 5}}},
			wantErr: model.ErrForbidden,
		},
		{
			name:    "организация не найдена",
			actor:   dispatcher,
			sources: &fakeSnapshotSources{},
			wantErr: model.ErrNotFound,
		},
		{
			name:    "ошибка источника пробрасывается с контекстом",
			actor:   dispatcher,
			sources: &fakeSnapshotSources{orgs: []model.Org{{ID: 5}}, vehErr: errDB},
			wantErr: errDB,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := newFleetServiceWithFakes(tt.sources, time.Now())
			_, err := svc.Snapshot(context.Background(), tt.actor)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("err = %v, want %v", err, tt.wantErr)
			}
		})
	}
}
