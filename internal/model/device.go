package model

import "time"

type DeviceStatus string

const (
	// DeviceStatusActive означает, что девайс работает и может отправлять телеметрию
	DeviceStatusActive DeviceStatus = "ACTIVE"
	// DeviceStatusInactive означает, что девайс неактивен
	DeviceStatusInactive DeviceStatus = "INACTIVE"
	// DeviceStatusMaintenance означает, что девайс в ремонте
	DeviceStatusMaintenance DeviceStatus = "MAINTENANCE"
)

type Device struct {
	ID           int
	SerialNumber string
	Status       DeviceStatus
	CreatedAt    time.Time
}

// IsDeviceStatusValid проверяет, входит ли статус в допустимый список
func IsDeviceStatusValid(s DeviceStatus) bool {
	switch s {
	case DeviceStatusActive, DeviceStatusInactive, DeviceStatusMaintenance:
		return true
	default:
		return false
	}
}
