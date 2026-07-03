package repository

import (
	"context"
	"fleettrack/internal/model"
	"testing"
	"time"
)

func intPtr(v int) *int             { return &v }
func float32Ptr(v float32) *float32 { return &v }
func float64Ptr(v float64) *float64 { return &v }
func timePtr(v time.Time) *time.Time { return &v }

func TestMatchesFilter(t *testing.T) {
	base := model.Telemetry{
		VehicleID:  1,
		DeviceID:   2,
		Fuel:       0.5,
		Lat:        10,
		Lon:        20,
		ReceivedAt: time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC),
	}

	tests := []struct {
		name   string
		filter model.TelemetryFilter
		want   bool
	}{
		{name: "empty filter matches anything", filter: model.TelemetryFilter{}, want: true},
		{name: "matching vehicle_id", filter: model.TelemetryFilter{VehicleID: intPtr(1)}, want: true},
		{name: "non-matching vehicle_id", filter: model.TelemetryFilter{VehicleID: intPtr(2)}, want: false},
		{name: "matching device_id", filter: model.TelemetryFilter{DeviceID: intPtr(2)}, want: true},
		{name: "non-matching device_id", filter: model.TelemetryFilter{DeviceID: intPtr(1)}, want: false},
		{name: "fuel within range", filter: model.TelemetryFilter{FuelMin: float32Ptr(0.1), FuelMax: float32Ptr(0.9)}, want: true},
		{name: "fuel below min", filter: model.TelemetryFilter{FuelMin: float32Ptr(0.6)}, want: false},
		{name: "fuel above max", filter: model.TelemetryFilter{FuelMax: float32Ptr(0.4)}, want: false},
		{name: "lat within range", filter: model.TelemetryFilter{LatMin: float64Ptr(5), LatMax: float64Ptr(15)}, want: true},
		{name: "lat below min", filter: model.TelemetryFilter{LatMin: float64Ptr(11)}, want: false},
		{name: "lat above max", filter: model.TelemetryFilter{LatMax: float64Ptr(9)}, want: false},
		{name: "lon within range", filter: model.TelemetryFilter{LonMin: float64Ptr(15), LonMax: float64Ptr(25)}, want: true},
		{name: "lon below min", filter: model.TelemetryFilter{LonMin: float64Ptr(21)}, want: false},
		{name: "lon above max", filter: model.TelemetryFilter{LonMax: float64Ptr(19)}, want: false},
		{
			name:   "received_at within [from, to]",
			filter: model.TelemetryFilter{From: timePtr(time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)), To: timePtr(time.Date(2026, 6, 30, 0, 0, 0, 0, time.UTC))},
			want:   true,
		},
		{
			name:   "received_at before from",
			filter: model.TelemetryFilter{From: timePtr(time.Date(2026, 6, 16, 0, 0, 0, 0, time.UTC))},
			want:   false,
		},
		{
			name:   "received_at after to",
			filter: model.TelemetryFilter{To: timePtr(time.Date(2026, 6, 14, 0, 0, 0, 0, time.UTC))},
			want:   false,
		},
		{
			name:   "combined filters all match",
			filter: model.TelemetryFilter{VehicleID: intPtr(1), DeviceID: intPtr(2), FuelMin: float32Ptr(0.1)},
			want:   true,
		},
		{
			name:   "combined filters one mismatches",
			filter: model.TelemetryFilter{VehicleID: intPtr(1), DeviceID: intPtr(999)},
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchesFilter(base, tt.filter)
			if got != tt.want {
				t.Errorf("matchesFilter() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMemoryTelemetryRepository_GetList_RespectsLimitAndOffset(t *testing.T) {
	repo := NewMemoryTelemetryRepository()
	for i := 0; i < 5; i++ {
		tel := model.Telemetry{VehicleID: 1, DeviceID: 1, Fuel: 0.5}
		if err := repo.Save(context.Background(), &tel); err != nil {
			t.Fatalf("unexpected error saving fixture: %v", err)
		}
	}

	res, err := repo.GetList(context.Background(), model.TelemetryFilter{VehicleID: intPtr(1), Limit: 2, Offset: 0})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res) != 2 {
		t.Errorf("got %d results, want 2 (limit)", len(res))
	}
}

func TestMemoryTelemetryRepository_GetList_FiltersOutNonMatching(t *testing.T) {
	repo := NewMemoryTelemetryRepository()
	t1 := model.Telemetry{VehicleID: 1, DeviceID: 1, Fuel: 0.5}
	t2 := model.Telemetry{VehicleID: 2, DeviceID: 1, Fuel: 0.5}
	_ = repo.Save(context.Background(), &t1)
	_ = repo.Save(context.Background(), &t2)

	res, err := repo.GetList(context.Background(), model.TelemetryFilter{VehicleID: intPtr(2), Limit: 100})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res) != 1 {
		t.Fatalf("got %d results, want 1", len(res))
	}
	if res[0].VehicleID != 2 {
		t.Errorf("got vehicle_id %d, want 2", res[0].VehicleID)
	}
}

func TestBuildWhereClause(t *testing.T) {
	tests := []struct {
		name      string
		filter    model.TelemetryFilter
		wantWhere string
		wantArgs  []any
	}{
		{
			name:      "no filters",
			filter:    model.TelemetryFilter{},
			wantWhere: "",
			wantArgs:  nil,
		},
		{
			name:      "vehicle_id only",
			filter:    model.TelemetryFilter{VehicleID: intPtr(5)},
			wantWhere: "WHERE vehicle_id = $1",
			wantArgs:  []any{5},
		},
		{
			name:      "vehicle_id and device_id",
			filter:    model.TelemetryFilter{VehicleID: intPtr(5), DeviceID: intPtr(7)},
			wantWhere: "WHERE vehicle_id = $1 AND device_id = $2",
			wantArgs:  []any{5, 7},
		},
		{
			name:      "fuel range",
			filter:    model.TelemetryFilter{FuelMin: float32Ptr(0.1), FuelMax: float32Ptr(0.9)},
			wantWhere: "WHERE fuel >= $1 AND fuel <= $2",
			wantArgs:  []any{float32(0.1), float32(0.9)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotWhere, gotArgs := buildWhereClause(tt.filter)
			if gotWhere != tt.wantWhere {
				t.Errorf("where = %q, want %q", gotWhere, tt.wantWhere)
			}
			if len(gotArgs) != len(tt.wantArgs) {
				t.Fatalf("args = %v, want %v", gotArgs, tt.wantArgs)
			}
			for i := range gotArgs {
				if gotArgs[i] != tt.wantArgs[i] {
					t.Errorf("args[%d] = %v, want %v", i, gotArgs[i], tt.wantArgs[i])
				}
			}
		})
	}
}
