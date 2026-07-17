package dto

import "fleettrack/internal/model"

// CreateDeviceRequest описывает тело HTTP запроса на создание устройства
type CreateDeviceRequest struct {
	SerialNumber string `json:"serial_number"`
}

// ToDomainModel конвертирует CreateDeviceRequest в domain-модель model.Device
func (cr *CreateDeviceRequest) ToDomainModel() model.Device {
	return model.Device{
		SerialNumber: cr.SerialNumber,
		Status:       model.DeviceStatusActive,
	}
}
