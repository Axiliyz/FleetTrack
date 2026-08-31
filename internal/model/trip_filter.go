package model

import "time"

// TripFilter определяет поля, которые можно фильтровать для рейсов
type TripFilter struct {
	DriverID  *int
	VehicleID *int
	Status    *TripStatus

	StartedFrom *time.Time
	StartedTo   *time.Time

	Limit  int
	Offset int
}
