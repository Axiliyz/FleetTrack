package model

import "time"

// RefreshToken — хранимая в БД запись о выданном refresh-токене
type RefreshToken struct {
	ID        int
	UserID    int
	TokenHash string
	ExpiresAt time.Time
	RevokedAt *time.Time
	CreatedAt time.Time
}
