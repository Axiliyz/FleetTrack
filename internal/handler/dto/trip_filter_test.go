package dto

import (
	"errors"
	"fleettrack/internal/model"
	"net/url"
	"testing"
)

func TestParseTripFilter_Errors(t *testing.T) {
	tests := []struct {
		name    string
		query   string
		wantErr error
	}{
		{name: "empty query", query: "", wantErr: nil},
		{name: "valid driver_id", query: "driver_id=5", wantErr: nil},
		{name: "invalid driver_id", query: "driver_id=abc", wantErr: model.ErrInvalidDriverID},
		{name: "valid vehicle_id", query: "vehicle_id=5", wantErr: nil},
		{name: "invalid vehicle_id", query: "vehicle_id=abc", wantErr: model.ErrInvalidVehicleID},
		{name: "valid status", query: "status=RUNNING", wantErr: nil},
		{name: "invalid status", query: "status=BOGUS", wantErr: model.ErrInvalidStatus},
		{name: "invalid started_from", query: "started_from=not-a-date", wantErr: model.ErrInvalidTimestamp},
		{name: "invalid started_to", query: "started_to=not-a-date", wantErr: model.ErrInvalidTimestamp},
		{name: "limit zero", query: "limit=0", wantErr: model.ErrInvalidLimit},
		{name: "limit negative", query: "limit=-5", wantErr: model.ErrInvalidLimit},
		{name: "offset negative", query: "offset=-1", wantErr: model.ErrInvalidOffset},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vals, err := url.ParseQuery(tt.query)
			if err != nil {
				t.Fatalf("bad test query: %v", err)
			}

			_, gotErr := ParseTripFilter(vals)
			if !errors.Is(gotErr, tt.wantErr) {
				t.Errorf("got err %v, want %v", gotErr, tt.wantErr)
			}
		})
	}
}

func TestParseTripFilter_Values(t *testing.T) {
	vals, err := url.ParseQuery("driver_id=5&vehicle_id=7&status=SUCCEEDED&limit=250&offset=10")
	if err != nil {
		t.Fatalf("bad test query: %v", err)
	}

	f, err := ParseTripFilter(vals)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if f.DriverID == nil || *f.DriverID != 5 {
		t.Errorf("DriverID = %v, want 5", f.DriverID)
	}
	if f.VehicleID == nil || *f.VehicleID != 7 {
		t.Errorf("VehicleID = %v, want 7", f.VehicleID)
	}
	if f.Status == nil || *f.Status != model.TripStatusSucceeded {
		t.Errorf("Status = %v, want SUCCEEDED", f.Status)
	}
	if f.Limit != 250 {
		t.Errorf("Limit = %v, want 250", f.Limit)
	}
	if f.Offset != 10 {
		t.Errorf("Offset = %v, want 10", f.Offset)
	}
}

func TestParseTripFilter_Defaults(t *testing.T) {
	f, err := ParseTripFilter(url.Values{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if f.DriverID != nil {
		t.Errorf("DriverID = %v, want nil", f.DriverID)
	}
	if f.Status != nil {
		t.Errorf("Status = %v, want nil", f.Status)
	}
	if f.Limit != 100 {
		t.Errorf("Limit = %v, want default 100", f.Limit)
	}
	if f.Offset != 0 {
		t.Errorf("Offset = %v, want default 0", f.Offset)
	}
}

func TestParseTripFilter_LimitCappedAtMax(t *testing.T) {
	vals, err := url.ParseQuery("limit=999999")
	if err != nil {
		t.Fatalf("bad test query: %v", err)
	}

	f, err := ParseTripFilter(vals)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f.Limit != 500 {
		t.Errorf("Limit = %v, want capped at 500", f.Limit)
	}
}
