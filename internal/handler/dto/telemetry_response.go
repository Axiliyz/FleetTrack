package dto

import "time"

// TelemetryResponse определяет структуру JSON телеметрии
type TelemetryResponse struct {
	TelemetryID int       `json:"telemetry_id"`
	VehicleID   int       `json:"vehicle_id"`
	DeviceID    int       `json:"device_id"`
	ReceivedAt  time.Time `json:"received_at"`
	TripID      int       `json:"trip_id"`
	DistanceKm  float64   `json:"distance_km"`
	SpeedKmh    float64   `json:"speed_kmh"`
}
