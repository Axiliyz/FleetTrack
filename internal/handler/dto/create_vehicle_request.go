package dto

import "fleettrack/internal/model"

// CreateVehicleRequest описывает тело HTTP запроса на создание автомобиля
type CreateVehicleRequest struct {
	OrganizationID int    `json:"organization_id"`
	VIN            string `json:"vin"`
	NumberPlate    string `json:"number_plate"`
	Model          string `json:"model"`
}

// ToDomainModel конвертирует CreateVehicleRequest в domain-модель model.Vehicle
func (cr *CreateVehicleRequest) ToDomainModel() model.Vehicle {
	return model.Vehicle{
		OrganizationID: cr.OrganizationID,
		VIN:            cr.VIN,
		NumberPlate:    cr.NumberPlate,
		Model:          cr.Model,
		Status:         model.VehicleStatusIdle,
	}
}
