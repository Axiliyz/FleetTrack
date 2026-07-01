// Package dto для транспортировки данных
package dto

import "fleettrack/internal/model"

// TelemetryRequest определяет структуру DTO запроса
type TelemetryRequest struct {
	DeviceID  int     `json:"device_id"`
	VehicleID int     `json:"vehicle_id"`
	Lat       float64 `json:"lat"`
	Lon       float64 `json:"lon"`
	Fuel      float32 `json:"fuel"`
}

// ToDomainModel преобразует TelemetryRequest в доменную модель
func (r *TelemetryRequest) ToDomainModel() model.Telemetry {
	return model.Telemetry{
		DeviceID:  r.DeviceID,
		VehicleID: r.VehicleID,
		Lat:       r.Lat,
		Lon:       r.Lon,
		Fuel:      r.Fuel,
	}
}
