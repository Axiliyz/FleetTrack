package validator

import (
	"fleettrack/internal/model"
	"testing"
)

func TestIsTripStatusValid(t *testing.T) {
	tests := []struct {
		name   string
		status model.TripStatus
		want   bool
	}{
		{name: "running", status: model.TripStatusRunning, want: true},
		{name: "cancelled", status: model.TripStatusCancelled, want: true},
		{name: "succeeded", status: model.TripStatusSucceeded, want: true},
		{name: "sleeping", status: model.TripStatusSleeping, want: true},
		{name: "serving", status: model.TripStatusServing, want: true},
		{name: "unknown", status: model.TripStatus("BOGUS"), want: false},
		{name: "empty", status: model.TripStatus(""), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsTripStatusValid(tt.status)
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidateTrip(t *testing.T) {
	if err := ValidateTrip(model.TripStatusRunning); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if err := ValidateTrip(model.TripStatus("BOGUS")); err != model.ErrInvalidStatus {
		t.Errorf("got %v, want %v", err, model.ErrInvalidStatus)
	}
}
