package dto

import (
	"fleettrack/internal/model"
	"time"
)

// DeviceResponse описывает тело HTTP ответа с данными устройства
type DeviceResponse struct {
	ID             int                `json:"id"`
	SerialNumber   string             `json:"serial_number"`
	Status         model.DeviceStatus `json:"status"`
	CreatedAt      time.Time          `json:"created_at"`
	OrganizationID int                `json:"organization_id"`
}

// NewDeviceResponse конвертирует model.Device в DeviceResponse
func NewDeviceResponse(d model.Device) DeviceResponse {
	return DeviceResponse{
		ID:             d.ID,
		SerialNumber:   d.SerialNumber,
		Status:         d.Status,
		CreatedAt:      d.CreatedAt,
		OrganizationID: d.OrganizationID,
	}
}
