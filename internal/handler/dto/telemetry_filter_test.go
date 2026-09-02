package dto

import (
	"errors"
	"fleettrack/internal/model"
	"net/url"
	"testing"
)

func TestParseTelemetryFilter_Errors(t *testing.T) {
	tests := []struct {
		name    string
		query   string
		wantErr error
	}{
		{name: "empty query", query: "", wantErr: nil},
		{name: "valid vehicle_id", query: "vehicle_id=5", wantErr: nil},
		{name: "invalid vehicle_id", query: "vehicle_id=abc", wantErr: model.ErrInvalidVehicleID},
		{name: "invalid device_id", query: "device_id=abc", wantErr: model.ErrInvalidDeviceID},
		{name: "invalid lat_min", query: "lat_min=abc", wantErr: model.ErrInvalidCoords},
		{name: "invalid lon_max", query: "lon_max=abc", wantErr: model.ErrInvalidCoords},
		{name: "invalid fuel_min", query: "fuel_min=abc", wantErr: model.ErrInvalidFuel},
		{name: "invalid fuel_max", query: "fuel_max=abc", wantErr: model.ErrInvalidFuel},
		{name: "invalid from", query: "from=not-a-date", wantErr: model.ErrInvalidTimestamp},
		{name: "invalid to", query: "to=not-a-date", wantErr: model.ErrInvalidTimestamp},
		{name: "valid from (RFC3339)", query: "from=2026-01-01T00:00:00Z", wantErr: nil},
		{name: "limit zero", query: "limit=0", wantErr: model.ErrInvalidLimit},
		{name: "limit negative", query: "limit=-5", wantErr: model.ErrInvalidLimit},
		{name: "limit not a number", query: "limit=abc", wantErr: model.ErrInvalidLimit},
		{name: "offset negative", query: "offset=-1", wantErr: model.ErrInvalidOffset},
		{name: "offset not a number", query: "offset=abc", wantErr: model.ErrInvalidOffset},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vals, err := url.ParseQuery(tt.query)
			if err != nil {
				t.Fatalf("bad test query: %v", err)
			}

			_, gotErr := ParseTelemetryFilter(vals)
			if !errors.Is(gotErr, tt.wantErr) {
				t.Errorf("got err %v, want %v", gotErr, tt.wantErr)
			}
		})
	}
}

func TestParseTelemetryFilter_Values(t *testing.T) {
	vals, err := url.ParseQuery("vehicle_id=5&device_id=7&fuel_min=0.2&fuel_max=0.9&lat_min=10.5&lat_max=20.5&limit=250&offset=10")
	if err != nil {
		t.Fatalf("bad test query: %v", err)
	}

	f, err := ParseTelemetryFilter(vals)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if f.VehicleID == nil || *f.VehicleID != 5 {
		t.Errorf("VehicleID = %v, want 5", f.VehicleID)
	}
	if f.DeviceID == nil || *f.DeviceID != 7 {
		t.Errorf("DeviceID = %v, want 7", f.DeviceID)
	}
	if f.FuelMin == nil || *f.FuelMin != 0.2 {
		t.Errorf("FuelMin = %v, want 0.2", f.FuelMin)
	}
	if f.FuelMax == nil || *f.FuelMax != 0.9 {
		t.Errorf("FuelMax = %v, want 0.9", f.FuelMax)
	}
	if f.LatMin == nil || *f.LatMin != 10.5 {
		t.Errorf("LatMin = %v, want 10.5", f.LatMin)
	}
	if f.LatMax == nil || *f.LatMax != 20.5 {
		t.Errorf("LatMax = %v, want 20.5", f.LatMax)
	}
	if f.Limit != 250 {
		t.Errorf("Limit = %v, want 250", f.Limit)
	}
	if f.Offset != 10 {
		t.Errorf("Offset = %v, want 10", f.Offset)
	}
}

func TestParseTelemetryFilter_Defaults(t *testing.T) {
	f, err := ParseTelemetryFilter(url.Values{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if f.VehicleID != nil {
		t.Errorf("VehicleID = %v, want nil", f.VehicleID)
	}
	if f.Limit != 100 {
		t.Errorf("Limit = %v, want default 100", f.Limit)
	}
	if f.Offset != 0 {
		t.Errorf("Offset = %v, want default 0", f.Offset)
	}
}

func TestParseTelemetryFilter_LimitCappedAtMax(t *testing.T) {
	vals, err := url.ParseQuery("limit=999999")
	if err != nil {
		t.Fatalf("bad test query: %v", err)
	}

	f, err := ParseTelemetryFilter(vals)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f.Limit != 500 {
		t.Errorf("Limit = %v, want capped at 500", f.Limit)
	}
}

func TestParseTelemetryFilter_FromAfterToIsNotValidatedHere(t *testing.T) {
	// ParseTelemetryFilter only parses individual fields; comparing From/To
	// against each other is the service's responsibility, not the parser's.
	vals, err := url.ParseQuery("from=2026-06-01T00:00:00Z&to=2026-01-01T00:00:00Z")
	if err != nil {
		t.Fatalf("bad test query: %v", err)
	}

	f, err := ParseTelemetryFilter(vals)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f.From == nil || f.To == nil {
		t.Fatal("expected both From and To to be set")
	}
	if !f.From.After(*f.To) {
		t.Fatalf("expected From (%v) to be after To (%v) for this test to be meaningful", f.From, f.To)
	}
}
