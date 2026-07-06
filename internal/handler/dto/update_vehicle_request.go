package dto

import "fleettrack/internal/model"

// UpdateVehicleRequest описывает тело HTTP запроса на частичное обновление автомобиля
type UpdateVehicleRequest struct {
	OrganizationID *int                 `json:"organization_id"`
	NumberPlate    *string              `json:"number_plate"`
	Status         *model.VehicleStatus `json:"status"`
}

func (u *UpdateVehicleRequest) ToDomainModel() model.UpdateVehicle {
	return model.UpdateVehicle{
		OrganizationID: u.OrganizationID,
		NumberPlate:    u.NumberPlate,
		Status:         u.Status,
	}
}
