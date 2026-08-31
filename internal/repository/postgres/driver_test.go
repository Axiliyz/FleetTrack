package postgres

import (
	"fleettrack/internal/model"
	"testing"
	"time"
)

func strPtr(v string) *string { return &v }

func TestBuildDriverWhereClause(t *testing.T) {
	tests := []struct {
		name      string
		filter    model.DriverFilter
		wantWhere string
		wantArgs  []any
	}{
		{
			name:      "no filters",
			filter:    model.DriverFilter{},
			wantWhere: "",
			wantArgs:  nil,
		},
		{
			name:      "organization_id only",
			filter:    model.DriverFilter{OrganizationID: intPtr(5)},
			wantWhere: "WHERE organization_id = $1",
			wantArgs:  []any{5},
		},
		{
			name:      "name only",
			filter:    model.DriverFilter{Name: strPtr("Ivan")},
			wantWhere: "WHERE name = $1",
			wantArgs:  []any{"Ivan"},
		},
		{
			name: "created range",
			filter: model.DriverFilter{
				CreatedFrom: timePtr(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)),
				CreatedTo:   timePtr(time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)),
			},
			wantWhere: "WHERE created_at >= $1 AND created_at <= $2",
			wantArgs: []any{
				time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
				time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
			},
		},
		{
			name: "all filters combined preserve argument order",
			filter: model.DriverFilter{
				OrganizationID: intPtr(5),
				Name:           strPtr("Ivan"),
				CreatedFrom:    timePtr(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)),
				CreatedTo:      timePtr(time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)),
			},
			wantWhere: "WHERE organization_id = $1 AND name = $2 AND created_at >= $3 AND created_at <= $4",
			wantArgs: []any{
				5, "Ivan",
				time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
				time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotWhere, gotArgs := buildDriverWhereClause(tt.filter)
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

func TestBuildDriverSetClause(t *testing.T) {
	tests := []struct {
		name     string
		upd      model.UpdateDriver
		wantSet  string
		wantArgs []any
	}{
		{
			name:     "no fields",
			upd:      model.UpdateDriver{},
			wantSet:  "",
			wantArgs: nil,
		},
		{
			name:     "organization_id only",
			upd:      model.UpdateDriver{OrganizationID: intPtr(3)},
			wantSet:  "organization_id = $1",
			wantArgs: []any{3},
		},
		{
			name:     "name only",
			upd:      model.UpdateDriver{Name: strPtr("New Name")},
			wantSet:  "name = $1",
			wantArgs: []any{"New Name"},
		},
		{
			name:     "both fields preserve order",
			upd:      model.UpdateDriver{OrganizationID: intPtr(3), Name: strPtr("New Name")},
			wantSet:  "organization_id = $1, name = $2",
			wantArgs: []any{3, "New Name"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotSet, gotArgs := buildDriverSetClause(tt.upd)
			if gotSet != tt.wantSet {
				t.Errorf("set = %q, want %q", gotSet, tt.wantSet)
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
