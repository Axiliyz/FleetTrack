// Package model содержит основные сущности логики
package model

// UpdateVehicle указывает изменяемые поля
type UpdateVehicle struct {
	OrganizationID *int
	NumberPlate    *string
	Status         *VehicleStatus
}
