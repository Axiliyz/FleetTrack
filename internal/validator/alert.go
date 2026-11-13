package validator

import "fleettrack/internal/model"

// IsAlertStatusValid проверяет, что статус входит в жизненный цикл алерта
func IsAlertStatusValid(s model.AlertStatus) bool {
	switch s {
	case model.AlertStatusFired, model.AlertStatusAcknowledged, model.AlertStatusResolved:
		return true
	default:
		return false
	}
}

// IsSeverityValid проверяет корректность уровня критичности
func IsSeverityValid(s model.SeverityLevel) bool {
	switch s {
	case model.SeverityLevelLow, model.SeverityLevelMedium, model.SeverityLevelHigh, model.SeverityLevelCritical:
		return true
	default:
		return false
	}
}
