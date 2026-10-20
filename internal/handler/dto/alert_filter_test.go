package dto

import (
	"errors"
	"fleettrack/internal/model"
	"net/url"
	"testing"
	"time"
)

// TestParseAlertFilter проверяет разбор query-параметров GET /alerts
func TestParseAlertFilter(t *testing.T) {
	f, err := ParseAlertFilter(url.Values{
		"status":          {"FIRED"},
		"severity":        {"CRITICAL"},
		"vehicle_id":      {"3"},
		"created_from":    {"2026-10-10T09:00:00Z"},
		"limit":           {"20"},
		"offset":          {"40"},
		"organization_id": {"999"},
	})
	if err != nil {
		t.Fatalf("ParseAlertFilter: %v", err)
	}
	if f.Status == nil || *f.Status != model.AlertStatusFired {
		t.Errorf("status = %v", f.Status)
	}
	if f.Severity == nil || *f.Severity != model.SeverityLevelCritical {
		t.Errorf("severity = %v", f.Severity)
	}
	if f.VehicleID == nil || *f.VehicleID != 3 {
		t.Errorf("vehicle_id = %v", f.VehicleID)
	}
	if f.CreatedAt == nil || !f.CreatedAt.Equal(time.Date(2026, 10, 10, 9, 0, 0, 0, time.UTC)) {
		t.Errorf("created_from = %v", f.CreatedAt)
	}
	if f.Limit != 20 || f.Offset != 40 {
		t.Errorf("limit/offset = %d/%d", f.Limit, f.Offset)
	}
	// организацию из query не читаем никогда — её выставляет хендлер по токену
	if f.OrganizationID != nil {
		t.Errorf("organization_id must be ignored, got %d", *f.OrganizationID)
	}
}

// TestParseAlertFilter_Defaults проверяет значения по умолчанию и потолок limit
func TestParseAlertFilter_Defaults(t *testing.T) {
	f, err := ParseAlertFilter(url.Values{})
	if err != nil {
		t.Fatalf("empty query: %v", err)
	}
	if f.Status != nil || f.Severity != nil || f.VehicleID != nil || f.CreatedAt != nil {
		t.Errorf("empty query must not set filters: %+v", f)
	}
	if f.Limit != 100 || f.Offset != 0 {
		t.Errorf("default limit/offset = %d/%d, want 100/0", f.Limit, f.Offset)
	}

	f, err = ParseAlertFilter(url.Values{"limit": {"100000"}})
	if err != nil || f.Limit != 500 {
		t.Errorf("limit cap: limit=%d err=%v, want 500", f.Limit, err)
	}
}

// TestParseAlertFilter_Errors проверяет, что каждое кривое значение даёт свою ошибку
func TestParseAlertFilter_Errors(t *testing.T) {
	tests := []struct {
		name  string
		query url.Values
		want  error
	}{
		{"неизвестный статус", url.Values{"status": {"OPEN"}}, model.ErrInvalidStatus},
		{"статус в нижнем регистре", url.Values{"status": {"fired"}}, model.ErrInvalidStatus},
		{"неизвестная критичность", url.Values{"severity": {"URGENT"}}, model.ErrInvalidSeverity},
		{"vehicle_id не число", url.Values{"vehicle_id": {"abc"}}, model.ErrInvalidVehicleID},
		{"дата не RFC3339", url.Values{"created_from": {"10.10.2026"}}, model.ErrInvalidTimestamp},
		{"нулевой limit", url.Values{"limit": {"0"}}, model.ErrInvalidLimit},
		{"отрицательный offset", url.Values{"offset": {"-1"}}, model.ErrInvalidOffset},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := ParseAlertFilter(tt.query); !errors.Is(err, tt.want) {
				t.Errorf("err = %v, want %v", err, tt.want)
			}
		})
	}
}

// TestUpdateAlertRequest_Validate проверяет, что через API можно только принять алерт
func TestUpdateAlertRequest_Validate(t *testing.T) {
	tests := []struct {
		status model.AlertStatus
		want   error
	}{
		{model.AlertStatusAcknowledged, nil},
		{model.AlertStatusResolved, model.ErrInvalidStatus},
		{model.AlertStatusFired, model.ErrInvalidStatus},
		{"", model.ErrInvalidStatus},
	}
	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			if err := (UpdateAlertRequest{Status: tt.status}).Validate(); !errors.Is(err, tt.want) {
				t.Errorf("err = %v, want %v", err, tt.want)
			}
		})
	}
}

// TestNewFleetSnapshotResponse_EmptyListsAreArrays проверяет, что nil-срезы снимка превращаются в пустые массивы
func TestNewFleetSnapshotResponse_EmptyListsAreArrays(t *testing.T) {
	res := NewFleetSnapshotResponse(model.FleetSnapshot{Org: model.Org{ID: 1, Name: "Org"}})
	if res.Vehicles == nil || res.Drivers == nil || res.Trips == nil || res.Positions == nil || res.Alerts == nil || res.Rules == nil {
		t.Errorf("nil list in response: %+v", res)
	}
	if res.Org.ID != 1 || res.Org.Name != "Org" {
		t.Errorf("org = %+v", res.Org)
	}
}
