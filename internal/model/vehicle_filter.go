package model

import "time"

// VehicleFilter определяет поля, которые можно фильтровать для автомобилей
type VehicleFilter struct {
	OrganizationID *int
	VIN            *string
	NumberPlate    *string
	Model          *string
	Status         *VehicleStatus
	CreatedFrom    *time.Time
	CreatedTo      *time.Time

	Limit  int
	Offset int
}
