package model

import "time"

// RefreshToken — хранимая в БД запись о выданном refresh-токене
type RefreshToken struct {
	ID     int
	UserID int
	// FamilyID объединяет все токены одной цепочки ротаций, начиная с логина
	FamilyID  string
	TokenHash string
	ExpiresAt time.Time
	RevokedAt *time.Time
	CreatedAt time.Time
}
