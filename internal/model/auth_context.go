package model

// AuthContext хранит данные аутентифицированного пользователя, извлечённые из JWT
type AuthContext struct {
	UserID         int
	OrganizationID int
	DriverID       *int
	Role           UserRole
}
