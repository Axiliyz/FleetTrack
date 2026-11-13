package dto

import (
	"encoding/json"
	"errors"
	"fleettrack/internal/model"
	"testing"
	"time"
)

// TestTelemetryRequest_DeviceTimestamp проверяет, что время терминала доезжает до модели, а без него остаётся нулевым
func TestTelemetryRequest_DeviceTimestamp(t *testing.T) {
	var withTS TelemetryRequest
	if err := json.Unmarshal([]byte(`{"device_id":1,"vehicle_id":2,"lat":55.7,"lon":37.6,"device_timestamp":"2026-10-10T09:00:00Z"}`), &withTS); err != nil {
		t.Fatal(err)
	}
	got, err := withTS.ToDomainModel()
	if err != nil {
		t.Fatal(err)
	}
	if !got.DeviceTimestamp.Equal(time.Date(2026, 10, 10, 9, 0, 0, 0, time.UTC)) {
		t.Errorf("DeviceTimestamp = %v", got.DeviceTimestamp)
	}

	var without TelemetryRequest
	if err := json.Unmarshal([]byte(`{"device_id":1,"vehicle_id":2,"lat":55.7,"lon":37.6}`), &without); err != nil {
		t.Fatal(err)
	}
	got, err = without.ToDomainModel()
	if err != nil {
		t.Fatal(err)
	}
	if !got.DeviceTimestamp.IsZero() {
		t.Errorf("DeviceTimestamp without field = %v, want zero (service sets receive time)", got.DeviceTimestamp)
	}
}

// TestTelemetryRequest_MissingCoords проверяет, что точка без координат не превращается в (0, 0),
// а настоящий ноль (экватор, Гринвич) проходит
func TestTelemetryRequest_MissingCoords(t *testing.T) {
	cases := map[string]error{
		`{"device_id":1,"vehicle_id":2}`:                 model.ErrInvalidCoords,
		`{"device_id":1,"vehicle_id":2,"lat":55.7}`:      model.ErrInvalidCoords,
		`{"device_id":1,"vehicle_id":2,"lon":37.6}`:      model.ErrInvalidCoords,
		`{"device_id":1,"vehicle_id":2,"lat":0,"lon":0}`: nil,
	}
	for body, want := range cases {
		var req TelemetryRequest
		if err := json.Unmarshal([]byte(body), &req); err != nil {
			t.Fatal(err)
		}
		if _, err := req.ToDomainModel(); !errors.Is(err, want) {
			t.Errorf("%s: err = %v, want %v", body, err, want)
		}
	}
}
