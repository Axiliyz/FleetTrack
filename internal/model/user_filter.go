package model

// UserFilter — параметры фильтрации списка пользователей
type UserFilter struct {
	OrganizationID *int
	Role           *UserRole

	Limit  int
	Offset int
}
