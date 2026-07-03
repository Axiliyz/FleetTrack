// Package model содержит основные сущности логики
package model

import "time"

type TelemetryFilter struct {
	VehicleID      *int
	DeviceID       *int
	TripID         *int
	DriverID       *int
	OrganizationID *int

	FuelMin, FuelMax *float32
	LatMin, LatMax   *float64
	LonMin, LonMax   *float64

	From, To *time.Time

	Limit  int
	Offset int
}
