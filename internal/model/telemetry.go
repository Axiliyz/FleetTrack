package model

import "time"

type Telemetry struct {
	TelemetryID     int       `json:"telemetry_id"`
	VehicleID       int       `json:"vehicle_id"`
	DeviceID        int       `json:"device_id"`
	Lat             float64   `json:"lat"`
	Lon             float64   `json:"lon"`
	Fuel            float32   `json:"fuel"`
	DeviceTimestamp time.Time `json:"device_timestamp"`
	ReceivedAt      time.Time `json:"received_at"`
}
