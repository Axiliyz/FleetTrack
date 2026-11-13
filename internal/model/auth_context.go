package model

import "time"

// AuthContext хранит данные аутентифицированного пользователя, извлечённые из JWT
type AuthContext struct {
	UserID         int
	OrganizationID int
	DriverID       *int
	Role           UserRole
	// ExpiresAt - когда истекает access-токен. Обычный запрос живёт секунды и до этого
	// не доживает, а SSE-поток живёт часами - по ExpiresAt он закрывается сам
	ExpiresAt time.Time
}
