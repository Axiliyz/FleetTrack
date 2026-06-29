// Package DTO для транспортировки данных
package dto

import "time"

// TelemetryResponse определяет структуру JSON телеметрии
type TelemetryResponse struct {
	TelemetryID int       `json:"telemetry_id"`
	VehicleID   int       `json:"vehicle_id"`
	DeviceID    int       `json:"device_id"`
	ReceivedAt  time.Time `json:"received_at"`
}
