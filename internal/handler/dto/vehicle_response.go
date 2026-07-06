package dto

import "fleettrack/internal/model"

// VehicleResponse описывает тело HTTP ответа с данными автомобиля
type VehicleResponse struct {
	ID             int                 `json:"id"`
	OrganizationID int                 `json:"organization_id"`
	VIN            string              `json:"vin"`
	NumberPlate    string              `json:"number_plate"`
	Model          string              `json:"model"`
	Status         model.VehicleStatus `json:"status"`
}
