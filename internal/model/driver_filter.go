package model

import "time"

// DriverFilter определяет поля, которые можно фильтровать для водителей
type DriverFilter struct {
	OrganizationID *int
	Name           *string
	CreatedFrom    *time.Time
	CreatedTo      *time.Time

	Limit  int
	Offset int
}
