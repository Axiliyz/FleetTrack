package dto

import "fleettrack/internal/model"

// CreateVehicleRequest описывает тело HTTP запроса на создание автомобиля.
// Организации в теле нет: машина создаётся в организации из токена
type CreateVehicleRequest struct {
	VIN         string `json:"vin"`
	NumberPlate string `json:"number_plate"`
	Model       string `json:"model"`
}

// ToDomainModel конвертирует CreateVehicleRequest в domain-модель model.Vehicle организации organizationID
func (cr *CreateVehicleRequest) ToDomainModel(organizationID int) model.Vehicle {
	return model.Vehicle{
		OrganizationID: organizationID,
		VIN:            cr.VIN,
		NumberPlate:    cr.NumberPlate,
		Model:          cr.Model,
		Status:         model.VehicleStatusIdle,
	}
}
