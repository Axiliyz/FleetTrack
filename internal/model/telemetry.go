// Package model содержит основные сущности логики
package model

import "time"

// Telemetry - основной тип программы, содержит все переданные девайсом данные
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
