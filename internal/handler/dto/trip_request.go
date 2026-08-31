package dto

import (
	"fleettrack/internal/model"
	"time"
)

// TripRequest - dto запроса на создание поездки
type TripRequest struct {
	DriverID  int `json:"driver_id"`
	VehicleID int `json:"vehicle_id"`
}

// ToDomain преобразует TripRequest в доменную сущность
func (r *TripRequest) ToDomain() model.Trip {
	return model.Trip{
		DriverID:  r.DriverID,
		VehicleID: r.VehicleID,
		StartedAt: time.Now(),
	}
}
