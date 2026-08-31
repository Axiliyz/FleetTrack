package model

import "time"

// Driver определяет структуру водителя
type Driver struct {
	ID             int        `json:"id"`
	OrganizationID int        `json:"organization_id"`
	Name           string     `json:"name"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      *time.Time `json:"updated_at"`
}
