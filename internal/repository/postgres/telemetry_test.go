// package postgres отвечает за взаимодействие с БД PostgreSQL
package postgres

import (
	"fleettrack/internal/model"
	"testing"
	"time"
)

func intPtr(v int) *int              { return &v }
func float32Ptr(v float32) *float32  { return &v }
func float64Ptr(v float64) *float64  { return &v }
func timePtr(v time.Time) *time.Time { return &v }

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
		{
			name:      "organization_id only",
			filter:    model.TelemetryFilter{OrganizationID: intPtr(3)},
			wantWhere: "WHERE organization_id = $1",
			wantArgs:  []any{3},
		},
		{
			name:      "lat range",
			filter:    model.TelemetryFilter{LatMin: float64Ptr(10.5), LatMax: float64Ptr(20.5)},
			wantWhere: "WHERE latitude >= $1 AND latitude <= $2",
			wantArgs:  []any{10.5, 20.5},
		},
		{
			name:      "lon range",
			filter:    model.TelemetryFilter{LonMin: float64Ptr(30.5), LonMax: float64Ptr(40.5)},
			wantWhere: "WHERE longitude >= $1 AND longitude <= $2",
			wantArgs:  []any{30.5, 40.5},
		},
		{
			name: "date range",
			filter: model.TelemetryFilter{
				From: timePtr(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)),
				To:   timePtr(time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)),
			},
			wantWhere: "WHERE received_at >= $1 AND received_at <= $2",
			wantArgs: []any{
				time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
				time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
			},
		},
		{
			name: "all filters combined preserve argument order",
			filter: model.TelemetryFilter{
				OrganizationID: intPtr(3),
				VehicleID:      intPtr(5),
				DeviceID:       intPtr(7),
				FuelMin:        float32Ptr(0.1),
				FuelMax:        float32Ptr(0.9),
				LatMin:         float64Ptr(10.5),
				LatMax:         float64Ptr(20.5),
				LonMin:         float64Ptr(30.5),
				LonMax:         float64Ptr(40.5),
				From:           timePtr(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)),
				To:             timePtr(time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)),
			},
			wantWhere: "WHERE organization_id = $1 AND vehicle_id = $2 AND device_id = $3 AND fuel >= $4 AND fuel <= $5 " +
				"AND latitude >= $6 AND latitude <= $7 AND longitude >= $8 AND longitude <= $9 " +
				"AND received_at >= $10 AND received_at <= $11",
			wantArgs: []any{
				3, 5, 7, float32(0.1), float32(0.9), 10.5, 20.5, 30.5, 40.5,
				time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
				time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
			},
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
