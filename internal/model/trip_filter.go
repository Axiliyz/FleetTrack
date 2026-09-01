package model

import "time"

// TripFilter определяет поля, которые можно фильтровать для рейсов
type TripFilter struct {
	DriverID  *int
	VehicleID *int
	Status    *TripStatus

	StartedFrom *time.Time
	StartedTo   *time.Time

	MinDistance *float64
	MaxDistance *float64
	MinAvgSpeed *float64
	MaxAvgSpeed *float64
	MinMaxSpeed *float64
	MaxMaxSpeed *float64

	Limit  int
	Offset int
}
