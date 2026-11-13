package model

// UpdateVehicle указывает изменяемые поля.
// Организацию машины поменять нельзя: такого поля здесь нет намеренно
type UpdateVehicle struct {
	NumberPlate *string
	Status      *VehicleStatus
}
