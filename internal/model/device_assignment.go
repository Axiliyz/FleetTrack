package model

import "time"

type DeviceAssignment struct {
	ID        int
	DeviceID  int
	VehicleID int
	StartedAt time.Time
	EndedAt   *time.Time
}
