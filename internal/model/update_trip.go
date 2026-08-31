package model

import "time"

// UpdateTrip указывает изменяемые поля рейса
type UpdateTrip struct {
	Status  *TripStatus
	EndedAt *time.Time
}
