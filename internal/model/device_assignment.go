package model

import "time"

// DeviceAssignment описывает связь устройства с автомобилем за период времени.
type DeviceAssignment struct {
	ID        int
	DeviceID  int
	VehicleID int
	StartedAt time.Time
	EndedAt   *time.Time
}
