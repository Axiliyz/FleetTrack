package validator

import (
	"fleettrack/internal/model"
	"strings"
)

// ValidateDriver валидирует водителя
func ValidateDriver(d model.Driver) error {
	if d.OrganizationID < 1 {
		return model.ErrInvalidOrganizationID
	}

	if strings.TrimSpace(d.Name) == "" || len(d.Name) > 35 {
		return model.ErrInvalidDriverName
	}
	return nil
}
