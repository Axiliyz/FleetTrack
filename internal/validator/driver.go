package validator

import "fleettrack/internal/model"

// ValidateDriver валидирует водителя
func ValidateDriver(d model.Driver) error {
	if d.OrganizationID < 1 {
		return model.ErrInvalidOrganizationID
	}

	if !IsNameValid(d.Name) {
		return model.ErrInvalidDriverName
	}
	return nil
}
