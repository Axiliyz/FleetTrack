package dto

// CreateAssignmentRequest описывает тело HTTP запроса на привязку устройства к автомобилю
type CreateAssignmentRequest struct {
	DeviceID  int `json:"device_id"`
	VehicleID int `json:"vehicle_id"`
}
