package dto

import "fleettrack/internal/model"

// UpdateTripRequest описывает тело HTTP запроса на изменение статуса рейса
type UpdateTripRequest struct {
	Status model.TripStatus `json:"status"`
}

// ToDomain преобразует UpdateTripRequest в доменную модель
func (u *UpdateTripRequest) ToDomain() model.Trip {
	return model.Trip{Status: u.Status}
}
