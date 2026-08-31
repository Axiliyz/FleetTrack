package dto

import (
	"errors"
	"fleettrack/internal/model"
	"net/url"
	"testing"
)

func TestParseDriverFilter_Errors(t *testing.T) {
	tests := []struct {
		name    string
		query   string
		wantErr error
	}{
		{name: "empty query", query: "", wantErr: nil},
		{name: "valid organization_id", query: "organization_id=5", wantErr: nil},
		{name: "invalid organization_id", query: "organization_id=abc", wantErr: model.ErrInvalidOrganizationID},
		{name: "valid name", query: "name=Ivan", wantErr: nil},
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

			_, gotErr := ParseDriverFilter(vals)
			if !errors.Is(gotErr, tt.wantErr) {
				t.Errorf("got err %v, want %v", gotErr, tt.wantErr)
			}
		})
	}
}

func TestParseDriverFilter_Values(t *testing.T) {
	vals, err := url.ParseQuery("organization_id=5&name=Ivan+Petrov&limit=250&offset=10")
	if err != nil {
		t.Fatalf("bad test query: %v", err)
	}

	f, err := ParseDriverFilter(vals)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if f.OrganizationID == nil || *f.OrganizationID != 5 {
		t.Errorf("OrganizationID = %v, want 5", f.OrganizationID)
	}
	if f.Name == nil || *f.Name != "Ivan Petrov" {
		t.Errorf("Name = %v, want 'Ivan Petrov'", f.Name)
	}
	if f.Limit != 250 {
		t.Errorf("Limit = %v, want 250", f.Limit)
	}
	if f.Offset != 10 {
		t.Errorf("Offset = %v, want 10", f.Offset)
	}
}

func TestParseDriverFilter_Defaults(t *testing.T) {
	f, err := ParseDriverFilter(url.Values{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if f.OrganizationID != nil {
		t.Errorf("OrganizationID = %v, want nil", f.OrganizationID)
	}
	if f.Name != nil {
		t.Errorf("Name = %v, want nil", f.Name)
	}
	if f.Limit != 100 {
		t.Errorf("Limit = %v, want default 100", f.Limit)
	}
	if f.Offset != 0 {
		t.Errorf("Offset = %v, want default 0", f.Offset)
	}
}

func TestParseDriverFilter_LimitCappedAtMax(t *testing.T) {
	vals, err := url.ParseQuery("limit=999999")
	if err != nil {
		t.Fatalf("bad test query: %v", err)
	}

	f, err := ParseDriverFilter(vals)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f.Limit != 500 {
		t.Errorf("Limit = %v, want capped at 500", f.Limit)
	}
}
