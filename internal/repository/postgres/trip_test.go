package postgres

import (
	"fleettrack/internal/model"
	"testing"
	"time"
)

func tripStatusPtr(s model.TripStatus) *model.TripStatus { return &s }

func TestBuildTripWhereClause(t *testing.T) {
	tests := []struct {
		name      string
		filter    *model.TripFilter
		wantWhere string
		wantArgs  []any
	}{
		{
			name:      "no filters",
			filter:    &model.TripFilter{},
			wantWhere: "",
			wantArgs:  nil,
		},
		{
			name:      "driver_id only",
			filter:    &model.TripFilter{DriverID: intPtr(5)},
			wantWhere: "WHERE t.driver_id = $1",
			wantArgs:  []any{5},
		},
		{
			name:      "vehicle_id only",
			filter:    &model.TripFilter{VehicleID: intPtr(7)},
			wantWhere: "WHERE t.vehicle_id = $1",
			wantArgs:  []any{7},
		},
		{
			name:      "organization_id only",
			filter:    &model.TripFilter{OrganizationID: intPtr(3)},
			wantWhere: "WHERE d.organization_id = $1",
			wantArgs:  []any{3},
		},
		{
			name:      "status only",
			filter:    &model.TripFilter{Status: tripStatusPtr(model.TripStatusRunning)},
			wantWhere: "WHERE t.status = $1",
			wantArgs:  []any{model.TripStatusRunning},
		},
		{
			name: "started range",
			filter: &model.TripFilter{
				StartedFrom: timePtr(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)),
				StartedTo:   timePtr(time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)),
			},
			wantWhere: "WHERE t.started_at >= $1 AND t.started_at <= $2",
			wantArgs: []any{
				time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
				time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
			},
		},
		{
			name:      "min_distance only",
			filter:    &model.TripFilter{MinDistance: float64Ptr(10)},
			wantWhere: "WHERE t.distance_km >= $1",
			wantArgs:  []any{10.0},
		},
		{
			name:      "max_distance only",
			filter:    &model.TripFilter{MaxDistance: float64Ptr(100)},
			wantWhere: "WHERE t.distance_km <= $1",
			wantArgs:  []any{100.0},
		},
		{
			name:      "min_avg_speed only",
			filter:    &model.TripFilter{MinAvgSpeed: float64Ptr(20)},
			wantWhere: "WHERE t.avg_speed_kmh >= $1",
			wantArgs:  []any{20.0},
		},
		{
			name:      "max_avg_speed only",
			filter:    &model.TripFilter{MaxAvgSpeed: float64Ptr(80)},
			wantWhere: "WHERE t.avg_speed_kmh <= $1",
			wantArgs:  []any{80.0},
		},
		{
			name:      "min_max_speed only",
			filter:    &model.TripFilter{MinMaxSpeed: float64Ptr(30)},
			wantWhere: "WHERE t.max_speed_kmh >= $1",
			wantArgs:  []any{30.0},
		},
		{
			name:      "max_max_speed only",
			filter:    &model.TripFilter{MaxMaxSpeed: float64Ptr(120)},
			wantWhere: "WHERE t.max_speed_kmh <= $1",
			wantArgs:  []any{120.0},
		},
		{
			name: "all filters combined preserve argument order",
			filter: &model.TripFilter{
				DriverID:       intPtr(5),
				VehicleID:      intPtr(7),
				OrganizationID: intPtr(3),
				Status:         tripStatusPtr(model.TripStatusRunning),
				StartedFrom:    timePtr(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)),
				StartedTo:      timePtr(time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)),
			},
			wantWhere: "WHERE t.driver_id = $1 AND t.vehicle_id = $2 AND d.organization_id = $3 AND t.status = $4 AND t.started_at >= $5 AND t.started_at <= $6",
			wantArgs: []any{
				5, 7, 3, model.TripStatusRunning,
				time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
				time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotWhere, gotArgs := buildTripWhereClause(tt.filter)
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

func TestIsFinalTripStatus(t *testing.T) {
	tests := []struct {
		name   string
		status model.TripStatus
		want   bool
	}{
		{name: "running", status: model.TripStatusRunning, want: false},
		{name: "sleeping", status: model.TripStatusSleeping, want: false},
		{name: "serving", status: model.TripStatusServing, want: false},
		{name: "succeeded", status: model.TripStatusSucceeded, want: true},
		{name: "cancelled", status: model.TripStatusCancelled, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isFinalTripStatus(tt.status)
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}
