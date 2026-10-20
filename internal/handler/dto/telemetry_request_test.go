package dto

import (
	"encoding/json"
	"testing"
	"time"
)

// TestTelemetryRequest_DeviceTimestamp проверяет, что время терминала доезжает до модели, а без него остаётся нулевым
func TestTelemetryRequest_DeviceTimestamp(t *testing.T) {
	var withTS TelemetryRequest
	if err := json.Unmarshal([]byte(`{"device_id":1,"vehicle_id":2,"lat":55.7,"lon":37.6,"device_timestamp":"2026-10-10T09:00:00Z"}`), &withTS); err != nil {
		t.Fatal(err)
	}
	if got := withTS.ToDomainModel().DeviceTimestamp; !got.Equal(time.Date(2026, 10, 10, 9, 0, 0, 0, time.UTC)) {
		t.Errorf("DeviceTimestamp = %v", got)
	}

	var without TelemetryRequest
	if err := json.Unmarshal([]byte(`{"device_id":1,"vehicle_id":2,"lat":55.7,"lon":37.6}`), &without); err != nil {
		t.Fatal(err)
	}
	if got := without.ToDomainModel().DeviceTimestamp; !got.IsZero() {
		t.Errorf("DeviceTimestamp without field = %v, want zero (service sets receive time)", got)
	}
}
