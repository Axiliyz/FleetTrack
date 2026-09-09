package dto

import "fleettrack/internal/model"

// CreateDeviceRequest описывает тело HTTP запроса на создание устройства
type CreateDeviceRequest struct {
	SerialNumber   string `json:"serial_number"`
	OrganizationID int    `json:"organization_id"`
}

// ToDomainModel конвертирует CreateDeviceRequest в domain-модель model.Device
func (cr *CreateDeviceRequest) ToDomainModel() model.Device {
	return model.Device{
		SerialNumber:   cr.SerialNumber,
		Status:         model.DeviceStatusActive,
		OrganizationID: cr.OrganizationID,
	}
}
