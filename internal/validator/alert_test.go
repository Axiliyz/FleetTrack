package validator

import (
	"fleettrack/internal/model"
	"testing"
)

// TestIsAlertStatusValid проверяет допустимые статусы жизненного цикла алерта
func TestIsAlertStatusValid(t *testing.T) {
	for _, s := range []model.AlertStatus{model.AlertStatusFired, model.AlertStatusAcknowledged, model.AlertStatusResolved} {
		if !IsAlertStatusValid(s) {
			t.Errorf("%s must be valid", s)
		}
	}
	for _, s := range []model.AlertStatus{"", "OPEN", "fired"} {
		if IsAlertStatusValid(s) {
			t.Errorf("%q must be invalid", s)
		}
	}
}

// TestIsSeverityValid проверяет допустимые уровни критичности
func TestIsSeverityValid(t *testing.T) {
	for _, s := range []model.SeverityLevel{model.SeverityLevelLow, model.SeverityLevelMedium, model.SeverityLevelHigh, model.SeverityLevelCritical} {
		if !IsSeverityValid(s) {
			t.Errorf("%s must be valid", s)
		}
	}
	for _, s := range []model.SeverityLevel{"", "URGENT", "high"} {
		if IsSeverityValid(s) {
			t.Errorf("%q must be invalid", s)
		}
	}
}
