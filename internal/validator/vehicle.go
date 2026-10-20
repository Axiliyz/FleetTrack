// Package validator содержит валидаторы для основных сущностей
package validator

import (
	"fleettrack/internal/model"
	"strings"
)

// ValidateVehicle валидирует автомобиль
func ValidateVehicle(v model.Vehicle) error {
	if !IsVINValid(v.VIN) {
		return model.ErrInvalidVIN
	}

	if !IsNumberPlateValid(v.NumberPlate) {
		return model.ErrInvalidNumberPlate
	}

	if v.OrganizationID < 1 {
		return model.ErrInvalidOrganizationID
	}

	if strings.TrimSpace(v.Model) == "" {
		return model.ErrInvalidModel
	}

	if !model.IsStatusValid(v.Status) {
		return model.ErrInvalidStatus
	}
	return nil
}
