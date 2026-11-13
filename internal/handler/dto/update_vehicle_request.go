package dto

import "fleettrack/internal/model"

// UpdateVehicleRequest описывает тело HTTP запроса на частичное обновление автомобиля.
// Поля organization_id нет: хендлер разбирает тело с DisallowUnknownFields,
// и попытка перенести машину в другую организацию получает 400
type UpdateVehicleRequest struct {
	NumberPlate *string              `json:"number_plate"`
	Status      *model.VehicleStatus `json:"status"`
}

// ToDomainModel преобразует UpdateVehicleRequest в доменную модель
func (u *UpdateVehicleRequest) ToDomainModel() model.UpdateVehicle {
	return model.UpdateVehicle{
		NumberPlate: u.NumberPlate,
		Status:      u.Status,
	}
}
