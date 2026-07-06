package dto

import (
	"errors"
	"fleettrack/internal/model"
	"net/url"
	"testing"
)

func TestParseVehicleFilter_Errors(t *testing.T) {
	tests := []struct {
		name    string
		query   string
		wantErr error
	}{
		{name: "empty query", query: "", wantErr: nil},
		{name: "valid organization_id", query: "organization_id=5", wantErr: nil},
		{name: "invalid organization_id", query: "organization_id=abc", wantErr: model.ErrInvalidOrganizationID},
		{name: "valid status", query: "status=IDLE", wantErr: nil},
		{name: "invalid status", query: "status=BOGUS", wantErr: model.ErrInvalidStatus},
		{name: "invalid created_from", query: "created_from=not-a-date", wantErr: model.ErrInvalidTimestamp},
		{name: "invalid created_to", query: "created_to=not-a-date", wantErr: model.ErrInvalidTimestamp},
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

			_, gotErr := ParseVehicleFilter(vals)
			if !errors.Is(gotErr, tt.wantErr) {
				t.Errorf("got err %v, want %v", gotErr, tt.wantErr)
			}
		})
	}
}

func TestParseVehicleFilter_Values(t *testing.T) {
	vals, err := url.ParseQuery("organization_id=5&vin=1HGCM82633A123456&number_plate=A123BC77&model=Camry&status=ON_TRIP&limit=250&offset=10")
	if err != nil {
		t.Fatalf("bad test query: %v", err)
	}

	f, err := ParseVehicleFilter(vals)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if f.OrganizationID == nil || *f.OrganizationID != 5 {
		t.Errorf("OrganizationID = %v, want 5", f.OrganizationID)
	}
	if f.VIN == nil || *f.VIN != "1HGCM82633A123456" {
		t.Errorf("VIN = %v, want 1HGCM82633A123456", f.VIN)
	}
	if f.NumberPlate == nil || *f.NumberPlate != "A123BC77" {
		t.Errorf("NumberPlate = %v, want A123BC77", f.NumberPlate)
	}
	if f.Model == nil || *f.Model != "Camry" {
		t.Errorf("Model = %v, want Camry", f.Model)
	}
	if f.Status == nil || *f.Status != model.VehicleStatusOnTrip {
		t.Errorf("Status = %v, want ON_TRIP", f.Status)
	}
	if f.Limit != 250 {
		t.Errorf("Limit = %v, want 250", f.Limit)
	}
	if f.Offset != 10 {
		t.Errorf("Offset = %v, want 10", f.Offset)
	}
}

func TestParseVehicleFilter_Defaults(t *testing.T) {
	f, err := ParseVehicleFilter(url.Values{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if f.OrganizationID != nil {
		t.Errorf("OrganizationID = %v, want nil", f.OrganizationID)
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

func TestParseVehicleFilter_LimitCappedAtMax(t *testing.T) {
	vals, err := url.ParseQuery("limit=999999")
	if err != nil {
		t.Fatalf("bad test query: %v", err)
	}

	f, err := ParseVehicleFilter(vals)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f.Limit != 500 {
		t.Errorf("Limit = %v, want capped at 500", f.Limit)
	}
}
