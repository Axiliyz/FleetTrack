package validator

import "fleettrack/internal/model"

// IsTripStatusValid проверяет корректность статуса рейса
func IsTripStatusValid(s model.TripStatus) bool {
	switch s {
	case model.TripStatusCancelled, model.TripStatusRunning, model.TripStatusSleeping, model.TripStatusSucceeded, model.TripStatusServing:
		return true
	default:
		return false
	}
}

// ValidateTrip валидирует статус рейса
func ValidateTrip(t model.TripStatus) error {
	if !IsTripStatusValid(t) {
		return model.ErrInvalidStatus
	}
	return nil
}
