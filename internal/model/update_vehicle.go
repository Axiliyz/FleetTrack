package model

// UpdateVehicle указывает изменяемые поля
type UpdateVehicle struct {
	OrganizationID *int
	NumberPlate    *string
	Status         *VehicleStatus
}
