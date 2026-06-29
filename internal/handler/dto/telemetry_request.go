package dto

import "fleettrack/internal/model"

type TelemetryRequest struct {
	DeviceID  int     `json:"device_id"`
	VehicleID int     `json:"vehicle_id"`
	Lat       float64 `json:"lat"`
	Lon       float64 `json:"lon"`
	Fuel      float32 `json:"fuel"`
}

func (r *TelemetryRequest) ToDomainModel() model.Telemetry {
	return model.Telemetry{
		DeviceID:  r.DeviceID,
		VehicleID: r.VehicleID,
		Lat:       r.Lat,
		Lon:       r.Lon,
		Fuel:      r.Fuel,
	}
}
