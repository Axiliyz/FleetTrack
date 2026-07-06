package model

import "time"

// VehicleStatus - enum возможных состояний машины
type VehicleStatus string

const (
	// VehicleStatusIdle означает, что машина простаивает
	VehicleStatusIdle VehicleStatus = "IDLE"
	// VehicleStatusOnTrip означает, что машина находится в рейсе
	VehicleStatusOnTrip VehicleStatus = "ON_TRIP"
	// VehicleStatusInService означает, что машина находится на обслуживании
	VehicleStatusInService VehicleStatus = "IN_SERVICE"
	// VehicleStatusDeleted означает, что машина удалена (soft delete)
	VehicleStatusDeleted VehicleStatus = "DELETED"
)

// IsStatusValid проверяет, что статус входит в число допустимых значений VehicleStatus
func IsStatusValid(s VehicleStatus) bool {
	switch s {
	case VehicleStatusIdle, VehicleStatusOnTrip, VehicleStatusInService, VehicleStatusDeleted:
		return true
	default:
		return false
	}
}

// Vehicle определяет структуру машины
type Vehicle struct {
	ID             int           `json:"id"`
	OrganizationID int           `json:"organization_id"`
	VIN            string        `json:"vin"`
	NumberPlate    string        `json:"number_plate"`
	Model          string        `json:"model"`
	Status         VehicleStatus `json:"status"`
	CreatedAt      time.Time     `json:"created_at"`
	UpdatedAt      *time.Time    `json:"updated_at"`
}
