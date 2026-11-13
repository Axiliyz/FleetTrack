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

// NewVehicleResponse конвертирует model.Vehicle в VehicleResponse
func NewVehicleResponse(v model.Vehicle) VehicleResponse {
	return VehicleResponse{
		ID:             v.ID,
		OrganizationID: v.OrganizationID,
		VIN:            v.VIN,
		NumberPlate:    v.NumberPlate,
		Model:          v.Model,
		Status:         v.Status,
	}
}
