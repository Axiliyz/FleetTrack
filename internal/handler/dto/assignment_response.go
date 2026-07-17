package dto

import (
	"fleettrack/internal/model"
	"time"
)

// AssignmentResponse описывает тело HTTP ответа со связью устройства и автомобиля
type AssignmentResponse struct {
	ID        int        `json:"id"`
	DeviceID  int        `json:"device_id"`
	VehicleID int        `json:"vehicle_id"`
	StartedAt time.Time  `json:"started_at"`
	EndedAt   *time.Time `json:"ended_at,omitempty"`
}

// NewAssignmentResponse конвертирует model.DeviceAssignment в AssignmentResponse
func NewAssignmentResponse(a model.DeviceAssignment) AssignmentResponse {
	return AssignmentResponse{
		ID:        a.ID,
		DeviceID:  a.DeviceID,
		VehicleID: a.VehicleID,
		StartedAt: a.StartedAt,
		EndedAt:   a.EndedAt,
	}
}
