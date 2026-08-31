package dto

import (
	"fleettrack/internal/model"
	"time"
)

// DriverResponse описывает тело HTTP ответа с данными водителя
type DriverResponse struct {
	ID             int        `json:"id"`
	OrganizationID int        `json:"organization_id"`
	Name           string     `json:"name"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      *time.Time `json:"updated_at"`
}

// NewDriverResponse - конструктор для HTTP ответа по водителю
func NewDriverResponse(d model.Driver) DriverResponse {
	return DriverResponse{
		ID:             d.ID,
		OrganizationID: d.OrganizationID,
		Name:           d.Name,
		CreatedAt:      d.CreatedAt,
		UpdatedAt:      d.UpdatedAt,
	}
}
