package dto

import "fleettrack/internal/model"

// CreateDeviceRequest описывает тело HTTP запроса на создание устройства.
// Организации в теле нет: устройство создаётся в организации из токена
type CreateDeviceRequest struct {
	SerialNumber string `json:"serial_number"`
}

// ToDomainModel конвертирует CreateDeviceRequest в domain-модель model.Device организации organizationID
func (cr *CreateDeviceRequest) ToDomainModel(organizationID int) model.Device {
	return model.Device{
		SerialNumber:   cr.SerialNumber,
		Status:         model.DeviceStatusActive,
		OrganizationID: organizationID,
	}
}
