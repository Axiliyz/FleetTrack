package model

import "time"

// DeviceAssignment описывает связь устройства с автомобилем за период времени.
type DeviceAssignment struct {
	ID        int        `json:"id"`
	DeviceID  int        `json:"device_id"`
	VehicleID int        `json:"vehicle_id"`
	StartedAt time.Time  `json:"started_at"`
	EndedAt   *time.Time `json:"ended_at"`
}
